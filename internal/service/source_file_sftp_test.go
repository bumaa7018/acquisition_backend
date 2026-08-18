package service_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// ── Фейк SFTP (SSH) сервер ───────────────────────────────────
//
// ГУС-ийн холболтын хүснэгтэд порт 22 бичигдсэн тул бодит урсгал нь SFTP.
// Тест нь ЖИНХЭНЭ SSH handshake + sftp subsystem-ээр ярина (mock биш).

type fakeSFTPServer struct {
	ln       net.Listener
	rootDir  string
	wantUser string
	wantPass string

	mu    sync.Mutex
	users []string
}

// newFakeSFTPServer — rootDir хавтсыг SFTP-ээр гаргана.
func newFakeSFTPServer(t *testing.T, rootDir, user, pass string) *fakeSFTPServer {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)

	s := &fakeSFTPServer{rootDir: rootDir, wantUser: user, wantPass: pass}

	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			s.mu.Lock()
			s.users = append(s.users, c.User())
			s.mu.Unlock()
			if c.User() != s.wantUser || string(password) != s.wantPass {
				return nil, errBadCredentials
			}
			return nil, nil
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s.ln = ln
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			go s.handle(conn, cfg)
		}
	}()
	return s
}

var errBadCredentials = &sshAuthError{}

type sshAuthError struct{}

func (*sshAuthError) Error() string { return "invalid credentials" }

func (s *fakeSFTPServer) addr() string { return s.ln.Addr().String() }

func (s *fakeSFTPServer) loginUsers() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.users...)
}

func (s *fakeSFTPServer) handle(conn net.Conn, cfg *ssh.ServerConfig) {
	defer conn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return // нэвтрэлт бүтсэнгүй
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "only session")
			continue
		}
		ch, chReqs, aerr := newCh.Accept()
		if aerr != nil {
			return
		}
		go func() {
			for req := range chReqs {
				ok := req.Type == "subsystem" && len(req.Payload) >= 4 &&
					string(req.Payload[4:]) == "sftp"
				if req.WantReply {
					_ = req.Reply(ok, nil)
				}
			}
		}()
		go func() {
			defer ch.Close()
			srv, serr := sftp.NewServer(ch, sftp.ReadOnly())
			if serr != nil {
				return
			}
			_ = srv.Serve()
		}()
	}
}

// sftpRoot — /DATA/mgis/011/01107/abc.pdf бүтэцтэй хавтас бэлдэнэ.
func sftpRoot(t *testing.T) (rootDir, relPath string) {
	t.Helper()
	base := t.TempDir()
	relPath = "mgis/011/01107/abc-6a7e697d329b7.pdf"
	full := filepath.Join(base, "DATA", filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(pdfContent), 0o600))
	// root_dir нь СЕРВЕР дээрх абсолют зам (тестэд TempDir дотор).
	return filepath.ToSlash(filepath.Join(base, "DATA")), relPath
}

// ── Тестүүд ──────────────────────────────────────────────────

// root_dir + харьцангуй зам → бүтэн зам болж татагдана.
func TestSFTPSourceFileClient_OpenWithRootDir(t *testing.T) {
	rootDir, relPath := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "gis_reader", "s3cret")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: srv.addr(), Username: "gis_reader", Password: "s3cret",
		RootDir: rootDir,
	}, 10*time.Second)

	f, err := client.Open(context.Background(), "3", relPath)
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
	assert.Equal(t, int64(len(pdfContent)), f.SizeBytes)
	assert.Contains(t, f.ContentType, "pdf")
	assert.Equal(t, []string{"gis_reader"}, srv.loginUsers())
}

// Замын эхэнд "/" байсан ч root_dir дээр зөв залгагдана.
func TestSFTPSourceFileClient_LeadingSlashPath(t *testing.T) {
	rootDir, relPath := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "u", "p")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "u", Password: "p", RootDir: rootDir,
	}, 10*time.Second)

	f, err := client.Open(context.Background(), "", "/"+relPath)
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
}

// Файл байхгүй бол 404.
func TestSFTPSourceFileClient_NotFound(t *testing.T) {
	rootDir, _ := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "u", "p")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "u", Password: "p", RootDir: rootDir,
	}, 10*time.Second)

	_, err := client.Open(context.Background(), "", "mgis/байхгүй.pdf")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Буруу нэвтрэх мэдээлэл.
func TestSFTPSourceFileClient_LoginFails(t *testing.T) {
	rootDir, relPath := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "gis_reader", "s3cret")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "gis_reader", Password: "буруу", RootDir: rootDir,
	}, 10*time.Second)

	_, err := client.Open(context.Background(), "", relPath)
	assert.ErrorIs(t, err, domain.ErrInternalServer)
}

// ftp_id таарахгүй бол өөр сервер дээрх файл — холбогдохгүй.
func TestSFTPSourceFileClient_MismatchedFtpID(t *testing.T) {
	rootDir, relPath := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "u", "p")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: srv.addr(), Username: "u", Password: "p", RootDir: rootDir,
	}, 10*time.Second)

	_, err := client.Open(context.Background(), "1", relPath)
	assert.ErrorIs(t, err, domain.ErrSourceFileNotConfigured)
	assert.Empty(t, srv.loginUsers(), "таарахгүй үед серверт холбогдох ёсгүй")
}

// Зам нь итгэмжлэгдээгүй: ".." хаагдана (root_dir-ээс гарах эрсдэл).
func TestSFTPSourceFileClient_RejectsPathTraversal(t *testing.T) {
	rootDir, _ := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "u", "p")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "u", Password: "p", RootDir: rootDir,
	}, 10*time.Second)

	for _, bad := range []string{"../etc/passwd", "mgis/../../etc/passwd", "  "} {
		_, err := client.Open(context.Background(), "", bad)
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "зам: %q", bad)
	}
	assert.Empty(t, srv.loginUsers(), "буруу зам дээр серверт холбогдох ёсгүй")
}

// SFTP-д бүтэн URL дэмжигдэхгүй (зөвхөн зам).
func TestSFTPSourceFileClient_RejectsURL(t *testing.T) {
	rootDir, _ := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "u", "p")

	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "u", Password: "p", RootDir: rootDir,
	}, 10*time.Second)

	for _, ref := range []string{"ftp://host/a.pdf", "http://host/a.pdf"} {
		_, err := client.Open(context.Background(), "", ref)
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "ref: %s", ref)
	}
}

// Тохируулаагүй бол ErrSourceFileNotConfigured.
func TestSFTPSourceFileClient_NotConfigured(t *testing.T) {
	client := service.NewSFTPSourceFileClient(service.SourceFTPTarget{}, 5*time.Second)
	_, err := client.Open(context.Background(), "3", "a.pdf")
	assert.ErrorIs(t, err, domain.ErrSourceFileNotConfigured)
}

// ── ГУС-ийн хавсралтыг SFTP-ээр ХАРАХ (service давхарга) ─────
//
// DB-д зөвхөн зам + ftp_id хадгалагдана → API нь SFTP сервертэй холбогдож
// урсгалаар дамжуулна. Файл MinIO-д ХУУЛАГДАХГҮЙ (DisabledStorage-тай ч
// ажиллана).
func TestLandService_OpenParcelDocumentFile_ViaSFTP(t *testing.T) {
	rootDir, relPath := sftpRoot(t)
	srv := newFakeSFTPServer(t, rootDir, "gis_reader", "s3cret")

	landRepo := &mocks.MockLandAcquisitionRepository{}
	sourceFiles := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: srv.addr(), Username: "gis_reader", Password: "s3cret",
		RootDir: rootDir,
	}, 10*time.Second)
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, sourceFiles)

	parcelID, docID := uuid.New(), uuid.New()
	landRepo.On("GetParcelDocument", ctx, parcelID, docID).Return(&domain.Document{
		ID: docID, Name: "Гэрчилгээ", FileType: "pdf",
		FileURL: relPath, SourceDocID: "77", SourceFtpID: "3",
	}, nil)

	f, err := svc.OpenParcelDocumentFile(ctx, parcelID, docID)
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
	assert.Equal(t, "Гэрчилгээ.pdf", f.Name)
}

// ── FTP сервер рүү ЯМАР ЗАМААР дуудагдаж байгааг шалгах ──────
//
// pathRecorder — SFTP хүсэлтийн замыг ЯГ байгаагаар нь тэмдэглэдэг серверийн
// handler (файлын систем рүү буулгахгүй тул "/DATA/…" замыг шууд харна).
type pathRecorder struct {
	content []byte

	mu    sync.Mutex
	calls []string
}

func (h *pathRecorder) record(method, p string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, method+" "+p)
}

func (h *pathRecorder) recorded() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.calls...)
}

func (h *pathRecorder) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	h.record("READ", r.Filepath)
	return bytes.NewReader(h.content), nil
}

func (h *pathRecorder) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	h.record(r.Method, r.Filepath)
	if r.Method == "Stat" {
		return fileInfoList{&stubFileInfo{name: path.Base(r.Filepath), size: int64(len(h.content))}}, nil
	}
	return nil, sftp.ErrSSHFxOpUnsupported
}

// Бичих/командын handler — зөвхөн уншина (эх систем рүү бичихгүй).
func (h *pathRecorder) Filewrite(*sftp.Request) (io.WriterAt, error) {
	return nil, sftp.ErrSSHFxPermissionDenied
}

func (h *pathRecorder) Filecmd(*sftp.Request) error {
	return sftp.ErrSSHFxPermissionDenied
}

type fileInfoList []os.FileInfo

func (l fileInfoList) ListAt(dst []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(dst, l[offset:])
	if n < len(dst) {
		return n, io.EOF
	}
	return n, nil
}

type stubFileInfo struct {
	name string
	size int64
}

func (f *stubFileInfo) Name() string       { return f.name }
func (f *stubFileInfo) Size() int64        { return f.size }
func (f *stubFileInfo) Mode() os.FileMode  { return 0o644 }
func (f *stubFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (f *stubFileInfo) IsDir() bool        { return false }
func (f *stubFileInfo) Sys() any           { return nil }

// newRecordingSFTPServer — замыг тэмдэглэдэг SFTP сервер.
func newRecordingSFTPServer(t *testing.T, rec *pathRecorder, user, pass string) string {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)

	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if c.User() != user || string(password) != pass {
				return nil, errBadCredentials
			}
			return nil, nil
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, aerr := ln.Accept()
			if aerr != nil {
				return
			}
			go func() {
				defer conn.Close()
				sshConn, chans, reqs, serr := ssh.NewServerConn(conn, cfg)
				if serr != nil {
					return
				}
				defer sshConn.Close()
				go ssh.DiscardRequests(reqs)
				for newCh := range chans {
					if newCh.ChannelType() != "session" {
						_ = newCh.Reject(ssh.UnknownChannelType, "only session")
						continue
					}
					ch, chReqs, aerr := newCh.Accept()
					if aerr != nil {
						return
					}
					go func() {
						for req := range chReqs {
							ok := req.Type == "subsystem" && len(req.Payload) >= 4 &&
								string(req.Payload[4:]) == "sftp"
							if req.WantReply {
								_ = req.Reply(ok, nil)
							}
						}
					}()
					go func() {
						defer ch.Close()
						_ = sftp.NewRequestServer(ch, sftp.Handlers{
							FileGet:  rec,
							FilePut:  rec,
							FileCmd:  rec,
							FileList: rec,
						}).Serve()
					}()
				}
			}()
		}
	}()
	return ln.Addr().String()
}

// ГУС-ийн ЯГ тэр хавсралтыг нээхэд сервер рүү
// "/DATA/mgis/011/01119/…/…-6261020030b0d.pdf" гэж дуудагдаж байгааг батална.
func TestLandService_SourceDocument_RequestsExactFTPPath(t *testing.T) {
	const fileURL = "mgis/011/01119/1462302475/01119-15-00627-22/" +
		"01119-15-00627-22-02-6261020030b0d.pdf"
	const wantPath = "/DATA/" + fileURL

	rec := &pathRecorder{content: []byte(pdfContent)}
	addr := newRecordingSFTPServer(t, rec, "gis_reader", "s3cret")

	landRepo := &mocks.MockLandAcquisitionRepository{}
	// Тохиргоо: SOURCE_FTP_ID=3, root_dir=/DATA (статик), порт 22 → SFTP
	sourceFiles := service.NewSFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: addr, Username: "gis_reader", Password: "s3cret",
		RootDir: "/DATA",
	}, 10*time.Second)
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, sourceFiles)

	parcelID, docID := uuid.New(), uuid.New()
	landRepo.On("GetParcelDocument", ctx, parcelID, docID).Return(&domain.Document{
		ID: docID, Name: "Гэрчилгээ", FileType: "pdf",
		FileURL: fileURL, SourceDocID: "77", SourceFtpID: "3",
	}, nil)

	f, err := svc.OpenParcelDocumentFile(ctx, parcelID, docID)
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))

	// Сервер рүү хүрсэн ЗАМ — root_dir + file_url
	calls := rec.recorded()
	for _, call := range calls {
		t.Logf("FTP сервер рүү: %s", call)
	}
	assert.Contains(t, calls, "Stat "+wantPath, "хэмжээ авах зам: %v", calls)
	assert.Contains(t, calls, "READ "+wantPath, "файл унших зам: %v", calls)
	for _, call := range calls {
		assert.Contains(t, call, wantPath,
			"өөр зам рүү хандсан байна: %s", call)
	}
}
