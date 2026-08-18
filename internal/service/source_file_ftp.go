package service

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"mime"
	"net"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jlaffaye/ftp"
)

// SourceFTPTarget — эх системийн файл серверийн холболтын мэдээлэл: .env-ийн
// SOURCE_FTP_ID / IP / USER / PASSWORD дөрвөн параметр.
type SourceFTPTarget struct {
	// FtpID — ЭНЭ сервер аль ftp_id-г хариуцахыг заана (ГУС хавсралт тус
	// бүрт ftp_id-г хамт буцаадаг). Хоосон бол бүх ftp_id-г хариуцна.
	FtpID string
	// Addr — "host:порт"
	Addr     string
	Username string
	Password string
	// RootDir — серверийн үндсэн хавтас (ж: /DATA). Хавсралтын харьцангуй
	// зам түүн дээр залгагдана.
	RootDir string
	// TLS — FTPS (шифрлэсэн холболт).
	TLS bool
	// ImplicitTLS — холболт шууд TLS-ээр эхлэх эсэх (порт 990). false үед
	// AUTH TLS-ээр (explicit) шифрлэгдэнэ.
	ImplicitTLS bool
}

// ftpSourceFileClient — эх системийн файл серверээс FTP-ээр файл ТАТНА
// (унших үйлдэл л байна).
//
// Хавсралт нь дундын сервисийн DB-д БАЙХГҮЙ: зөвхөн зам (file_url) ба аль
// сервер дээр байгаа нь (ftp_id) бүртгэлтэй.
//
// Холболт нь файл ТУС БҮРТ шинээр гарна (pool байхгүй): FTP-ийн нэг холболт
// нэг өгөгдлийн урсгал л дэмждэг тул зэрэгцээ татах үед хуваалцах боломжгүй.
type ftpSourceFileClient struct {
	target  SourceFTPTarget
	timeout time.Duration
}

// NewFTPSourceFileClient — FTP-ээр файл татах client.
//
// target.Addr хоосон бол Open нь ErrSourceFileNotConfigured буцаана (сервис
// асахад саад болохгүй).
func NewFTPSourceFileClient(target SourceFTPTarget, timeout time.Duration) SourceFileClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &ftpSourceFileClient{target: target, timeout: timeout}
}

// ftpStream — файлын урсгалыг ба түүний ДОР байгаа FTP холболтыг хамт хаана.
// Урсгалыг хаалгүй холболтыг хаавал татаж дуусаагүй байт тасарна; холболтыг
// хаалгүй орхивол сервер дээрх session хуримтлагдана.
type ftpStream struct {
	resp *ftp.Response
	conn *ftp.ServerConn
}

func (s *ftpStream) Read(p []byte) (int, error) { return s.resp.Read(p) }

func (s *ftpStream) Close() error {
	err := s.resp.Close()
	// Quit нь 221-ийг хүлээдэг — алдаа гарлаа ч холболт хаагдана.
	if qerr := s.conn.Quit(); err == nil && qerr != nil {
		err = qerr
	}
	return err
}

func (c *ftpSourceFileClient) Open(ctx context.Context, ftpID, filePath string) (*SourceFile, error) {
	cleanPath, err := sanitizeSourcePath(filePath)
	if err != nil {
		return nil, err
	}

	target, err := resolveSourceTarget(c.target, ftpID)
	if err != nil {
		return nil, err
	}

	// ГУС-аас ирэх file_url нь ХАРЬЦАНГУЙ зам байх нь хэвийн
	// (mgis/011/01107/…-6a7e697d329b7.pdf) ч БҮТЭН URL байж ч мэднэ. Тэр
	// тохиолдолд URL нь серверийг ӨӨРӨӨ заана.
	if urlTarget, urlPath, uerr := parseSourceURL(cleanPath, target); uerr != nil {
		return nil, uerr
	} else if urlPath != "" {
		target, cleanPath = urlTarget, urlPath
	}
	// Серверийн үндсэн хавтас (root_dir) дээр залгана.
	cleanPath = joinRootDir(target.RootDir, cleanPath)

	opts := []ftp.DialOption{
		ftp.DialWithContext(ctx),
		ftp.DialWithTimeout(c.timeout),
	}
	if target.TLS {
		// Сервер дотоод сүлжээнд IP-гаар холбогддог тул сертификатын нэрийг
		// шалгах боломжгүй — TLS нь дамжуулалтыг шифрлэх зорилготой.
		tlsCfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec // дотоод сүлжээ, IP-гаар холбогдоно
		if target.ImplicitTLS {
			opts = append(opts, ftp.DialWithTLS(tlsCfg))
		} else {
			opts = append(opts, ftp.DialWithExplicitTLS(tlsCfg))
		}
	}

	conn, err := ftp.Dial(target.Addr, opts...)
	if err != nil {
		slog.Error("эх системийн FTP серверт холбогдож чадсангүй",
			"addr", target.Addr, "ftp_id", ftpID, "error", err)
		return nil, domain.ErrInternalServer
	}

	// Нэвтрэх нэр хоосон бол анонимоор нэвтэрнэ (нээлттэй сервер).
	user, pass := target.Username, target.Password
	if user == "" {
		user, pass = "anonymous", "anonymous"
	}
	if err := conn.Login(user, pass); err != nil {
		_ = conn.Quit()
		slog.Error("эх системийн FTP серверт нэвтэрч чадсангүй",
			"addr", target.Addr, "ftp_id", ftpID, "user", user, "error", err)
		return nil, domain.ErrInternalServer
	}

	// Хэмжээг татахаас ӨМНӨ авна (Content-Length-д хэрэглэнэ). SIZE командыг
	// дэмждэггүй сервер ч байдаг тул алдааг үл хэрэгсэнэ (-1 → chunked).
	sizeBytes := int64(-1)
	if size, serr := conn.FileSize(cleanPath); serr == nil {
		sizeBytes = size
	}

	resp, err := conn.Retr(cleanPath)
	if err != nil {
		_ = conn.Quit()
		// Файл байхгүй нь 550-аар ирдэг — 404 болгож ялгана.
		if isFTPNotFound(err) {
			return nil, domain.ErrNotFound
		}
		slog.Error("эх системийн FTP серверээс файл татаж чадсангүй",
			"addr", target.Addr, "ftp_id", ftpID, "path", cleanPath, "error", err)
		return nil, domain.ErrInternalServer
	}

	return &SourceFile{
		// FTP нь MIME төрөл буцаадаггүй — өргөтгөлөөс тодорхойлно.
		ContentType: contentTypeByExt(cleanPath),
		SizeBytes:   sizeBytes,
		Content:     &ftpStream{resp: resp, conn: conn},
	}, nil
}

// parseSourceURL — file_url нь БҮТЭН URL эсэхийг шалгаж, FTP-д хэрэглэх
// сервер + замыг гаргана.
//
// ГУС-аас ирэх file_url нь ихэвчлэн харьцангуй зам (mgis/011/…) — тэр үед
// path="" буцаж, дуудагч тохиргооны серверийг хэрэглэнэ. Бүтэн URL үед:
//
//   - ftp://[нэр[:код]@]хост[:порт]/зам — URL нь СЕРВЕРИЙГ өөрөө заана.
//     Нэвтрэх мэдээлэл URL-д байвал түүнийг, байхгүй бол тохиргооныхыг авна
//     (нэвтрэх код нь URL-д ил байх нь эрсдэлтэй тул тохиргоог урьтуулах
//     шаардлагагүй — URL-д зөвхөн нэр байвал кодыг тохиргооноос нөхнө).
//   - бусад схем (http, sftp, file, …) — дэмжигдэхгүй: татах нь ЗӨВХӨН
//     FTP-ээр хийгдэнэ.
func parseSourceURL(filePath string, cfgTarget SourceFTPTarget) (SourceFTPTarget, string, error) {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasPrefix(lower, "ftp://"):
		// доор
	case strings.Contains(lower, "://"):
		slog.Warn("эх системийн file_url дэмжигдэхгүй схемтэй", "file_url", filePath)
		return SourceFTPTarget{}, "", domain.ErrInvalidInput
	default:
		// Харьцангуй зам — тохиргооны сервер хэрэглэгдэнэ.
		return SourceFTPTarget{}, "", nil
	}

	u, err := url.Parse(filePath)
	if err != nil || u.Host == "" {
		slog.Warn("эх системийн ftp:// хаяг уншигдсангүй", "file_url", filePath, "error", err)
		return SourceFTPTarget{}, "", domain.ErrInvalidInput
	}

	target := SourceFTPTarget{
		Addr:     withFTPPort(u.Host),
		Username: cfgTarget.Username,
		Password: cfgTarget.Password,
	}
	if u.User != nil {
		if name := u.User.Username(); name != "" {
			target.Username = name
			if pass, ok := u.User.Password(); ok {
				target.Password = pass
			}
		}
	}

	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return SourceFTPTarget{}, "", domain.ErrInvalidInput
	}
	return target, path, nil
}

// withFTPPort — "10.0.0.5" → "10.0.0.5:21" (ftp:// URL-д порт заагаагүй үед).
// config.withFTPPort-той ижил дүрэм; давхардлыг арилгахын тулд service
// давхарга нь config-оос хамаарах ёсгүй тул тусад нь байна.
func withFTPPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, "21")
}

// resolveSourceTarget — тохируулсан серверийг буцаана (FTP/SFTP хоёуланд).
//
// SOURCE_FTP_ID тохируулсан бол хавсралтын ftp_id ТҮҮНТЭЙ таарах ёстой:
// таарахгүй бол өөр сервер дээрх файл гэсэн үг тул хаяг нь бидэнд байхгүй.
// SOURCE_FTP_ID хоосон бол бүх ftp_id-г ЭНЭ сервер хариуцна.
func resolveSourceTarget(target SourceFTPTarget, ftpID string) (SourceFTPTarget, error) {
	if strings.TrimSpace(target.Addr) == "" {
		return SourceFTPTarget{}, domain.ErrSourceFileNotConfigured
	}
	want := strings.TrimSpace(target.FtpID)
	got := strings.TrimSpace(ftpID)
	if want != "" && got != "" && want != got {
		slog.Warn("хавсралтын ftp_id тохируулсан сервертэй таарахгүй",
			"doc_ftp_id", got, "configured_ftp_id", want)
		return SourceFTPTarget{}, domain.ErrSourceFileNotConfigured
	}
	return target, nil
}

// joinRootDir — серверийн үндсэн хавтас дээр хавсралтын замыг залгана.
// root_dir="/DATA", file_url="mgis/011/a.pdf" → "/DATA/mgis/011/a.pdf".
func joinRootDir(rootDir, filePath string) string {
	rootDir = strings.TrimRight(strings.TrimSpace(rootDir), "/")
	if rootDir == "" {
		return filePath
	}
	return rootDir + "/" + strings.TrimPrefix(filePath, "/")
}

// isFTPNotFound — FTP-ийн 550 (File unavailable) хариу. Сан нь протоколын
// алдааг *textproto.Error хэлбэрээр буцаана.
func isFTPNotFound(err error) bool {
	var protoErr *textproto.Error
	if errors.As(err, &protoErr) {
		return protoErr.Code == ftp.StatusFileUnavailable ||
			protoErr.Code == ftp.StatusFileActionIgnored
	}
	return false
}

// contentTypeByExt — өргөтгөлөөс MIME төрөл. Танихгүй бол octet-stream.
func contentTypeByExt(filePath string) string {
	if ct := mime.TypeByExtension(strings.ToLower(path.Ext(filePath))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// sanitizeSourcePath — эх системээс ирсэн замыг шалгана.
//
// Зам нь ИТГЭМЖЛЭГДЭЭГҮЙ: ".." нь тохируулсан серверийн хүрээнээс гарах
// эрсдэлтэй. Windows-ийн "\" зураасыг "/" болгож нэгтгэнэ.
func sanitizeSourcePath(filePath string) (string, error) {
	filePath = strings.TrimSpace(strings.ReplaceAll(filePath, "\\", "/"))
	if filePath == "" {
		return "", domain.ErrInvalidInput
	}
	for _, part := range strings.Split(filePath, "/") {
		if part == ".." {
			return "", domain.ErrInvalidInput
		}
	}
	return filePath, nil
}
