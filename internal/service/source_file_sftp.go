package service

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// sftpSourceFileClient — эх системийн файл серверээс SFTP (SSH, порт 22)-ээр
// файл ТАТНА. Унших үйлдэл л байна.
//
// ГУС-ийн холболтын хүснэгтэд порт 22 бичигдсэн бол сервер нь FTP БИШ, SSH
// дээрх SFTP байна (тэр үед passive/ssl тохиргоо хамааралгүй — SSH нь өөрөө
// шифрлэгдсэн).
//
// Холболт нь файл ТУС БҮРТ шинээр гарна: урсгал хаагдахад SFTP session ба SSH
// холболт хамт хаагдана.
type sftpSourceFileClient struct {
	target  SourceFTPTarget
	timeout time.Duration
}

// NewSFTPSourceFileClient — SFTP-ээр файл татах client.
func NewSFTPSourceFileClient(target SourceFTPTarget, timeout time.Duration) SourceFileClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &sftpSourceFileClient{target: target, timeout: timeout}
}

// sftpStream — файлын урсгал ба доорх session/холболтыг хамт хаана.
type sftpStream struct {
	file   *sftp.File
	client *sftp.Client
	conn   *ssh.Client
}

func (s *sftpStream) Read(p []byte) (int, error) { return s.file.Read(p) }

func (s *sftpStream) Close() error {
	err := s.file.Close()
	if cerr := s.client.Close(); err == nil {
		err = cerr
	}
	if cerr := s.conn.Close(); err == nil {
		err = cerr
	}
	return err
}

func (c *sftpSourceFileClient) Open(ctx context.Context, ftpID, filePath string) (*SourceFile, error) {
	cleanPath, err := sanitizeSourcePath(filePath)
	if err != nil {
		return nil, err
	}
	target, err := resolveSourceTarget(c.target, ftpID)
	if err != nil {
		return nil, err
	}
	// SFTP-д ftp:// хэлбэрийн URL хамаарахгүй — зөвхөн зам.
	if strings.Contains(strings.ToLower(cleanPath), "://") {
		slog.Warn("SFTP урсгалд бүтэн URL дэмжигдэхгүй", "file_url", cleanPath)
		return nil, domain.ErrInvalidInput
	}
	fullPath := joinRootDir(target.RootDir, cleanPath)

	user, pass := target.Username, target.Password
	if user == "" {
		return nil, domain.ErrSourceFileNotConfigured
	}

	// HostKeyCallback — эх систем нь дотоод сүлжээнд IP-гаар холбогддог тул
	// хостын түлхүүр урьдчилан бүртгэгдээгүй. Дамжуулалт шифрлэгдэнэ, харин
	// сервер нь дотоод сүлжээнд байна гэдэгт л найдна.
	sshCfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // дотоод сүлжээ, IP-гаар холбогдоно
		Timeout:         c.timeout,
	}

	// Dial нь context-ыг дэмждэггүй тул сүлжээний холболтыг өөрсдөө нээж,
	// дараа нь SSH handshake хийнэ (ингэснээр хүсэлт цуцлагдахад тасарна).
	dialer := &net.Dialer{Timeout: c.timeout}
	netConn, err := dialer.DialContext(ctx, "tcp", target.Addr)
	if err != nil {
		slog.Error("эх системийн SFTP серверт холбогдож чадсангүй",
			"addr", target.Addr, "ftp_id", ftpID, "error", err)
		return nil, domain.ErrInternalServer
	}
	_ = netConn.SetDeadline(time.Now().Add(c.timeout))

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, target.Addr, sshCfg)
	if err != nil {
		_ = netConn.Close()
		slog.Error("эх системийн SFTP серверт нэвтэрч чадсангүй",
			"addr", target.Addr, "ftp_id", ftpID, "user", user, "error", err)
		return nil, domain.ErrInternalServer
	}
	// Handshake дууссаны дараа deadline-ыг цуцална — файл татах нь урт байж
	// болно (дуудагч ctx-ээр хянана).
	_ = netConn.SetDeadline(time.Time{})
	conn := ssh.NewClient(sshConn, chans, reqs)

	client, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		slog.Error("SFTP session нээгдсэнгүй", "addr", target.Addr, "error", err)
		return nil, domain.ErrInternalServer
	}

	// Хэмжээг татахаас өмнө авна (Content-Length-д).
	sizeBytes := int64(-1)
	if st, serr := client.Stat(fullPath); serr == nil {
		sizeBytes = st.Size()
	} else if errors.Is(serr, fs.ErrNotExist) {
		_ = client.Close()
		_ = conn.Close()
		return nil, domain.ErrNotFound
	}

	file, err := client.Open(fullPath)
	if err != nil {
		_ = client.Close()
		_ = conn.Close()
		if errors.Is(err, fs.ErrNotExist) {
			return nil, domain.ErrNotFound
		}
		slog.Error("эх системийн SFTP серверээс файл татаж чадсангүй",
			"addr", target.Addr, "path", fullPath, "error", err)
		return nil, domain.ErrInternalServer
	}

	return &SourceFile{
		ContentType: contentTypeByExt(fullPath),
		SizeBytes:   sizeBytes,
		Content:     &sftpStream{file: file, client: client, conn: conn},
	}, nil
}
