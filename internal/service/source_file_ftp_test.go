package service_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Фейк FTP сервер ──────────────────────────────────────────
//
// Бодит FTP протоколоор (USER/PASS/TYPE/PASV/SIZE/RETR/QUIT) ярина — client-ыг
// mock-гүйгээр, төгсгөлөөс төгсгөл хүртэл шалгана.

type fakeFTPServer struct {
	t     *testing.T
	ln    net.Listener
	files map[string][]byte
	// wantUser/wantPass — нэвтрэх мэдээлэл. Хоосон бол ямар ч утга зөвшөөрнө.
	wantUser, wantPass string

	mu       sync.Mutex
	gotUser  string
	gotPass  string
	commands []string
	// noSize — SIZE командыг дэмжихгүй сервер (500 буцаана).
	noSize bool
}

func newFakeFTPServer(t *testing.T, files map[string][]byte) *fakeFTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &fakeFTPServer{t: t, ln: ln, files: files}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeFTPServer) addr() string { return s.ln.Addr().String() }

func (s *fakeFTPServer) record(cmd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, cmd)
}

func (s *fakeFTPServer) credentials() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gotUser, s.gotPass
}

func (s *fakeFTPServer) issued() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

func (s *fakeFTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener хаагдсан
		}
		go s.handle(conn)
	}
}

func (s *fakeFTPServer) handle(conn net.Conn) {
	defer conn.Close()
	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	send := func(format string, args ...any) {
		fmt.Fprintf(w, format+"\r\n", args...)
		_ = w.Flush()
	}

	send("220 fake ftp ready")

	// PASV-аар нээгдсэн өгөгдлийн listener
	var dataLn net.Listener
	defer func() {
		if dataLn != nil {
			_ = dataLn.Close()
		}
	}()

	authed := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		cmd, arg, _ := strings.Cut(line, " ")
		cmd = strings.ToUpper(cmd)
		s.record(cmd + " " + arg)

		switch cmd {
		case "USER":
			s.mu.Lock()
			s.gotUser = arg
			s.mu.Unlock()
			send("331 need password")
		case "PASS":
			s.mu.Lock()
			s.gotPass = arg
			user := s.gotUser
			s.mu.Unlock()
			if s.wantUser != "" && (user != s.wantUser || arg != s.wantPass) {
				send("530 login incorrect")
				continue
			}
			authed = true
			send("230 logged in")
		case "FEAT":
			send("211-Features:")
			send(" SIZE")
			send("211 End")
		case "TYPE":
			send("200 type set")
		case "OPTS":
			send("200 ok")
		case "SIZE":
			if s.noSize {
				send("500 not supported")
				continue
			}
			if data, ok := s.files[strings.TrimPrefix(arg, "/")]; ok {
				send("213 %d", len(data))
				continue
			}
			send("550 file not found")
		case "EPSV":
			// Зөвхөн PASV-г дэмжинэ — client PASV руу унана.
			send("500 not supported")
		case "PASV":
			if !authed {
				send("530 not logged in")
				continue
			}
			if dataLn != nil {
				_ = dataLn.Close()
			}
			dataLn, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				send("425 cannot open data connection")
				continue
			}
			port := dataLn.Addr().(*net.TCPAddr).Port
			send("227 Entering Passive Mode (127,0,0,1,%d,%d)", port>>8, port&0xFF)
		case "RETR":
			data, ok := s.files[strings.TrimPrefix(arg, "/")]
			if !ok {
				send("550 file not found")
				continue
			}
			if dataLn == nil {
				send("425 use PASV first")
				continue
			}
			send("150 opening data connection")
			dataConn, aerr := dataLn.Accept()
			if aerr != nil {
				send("426 data connection failed")
				continue
			}
			_, _ = dataConn.Write(data)
			_ = dataConn.Close()
			send("226 transfer complete")
		case "QUIT":
			send("221 bye")
			return
		default:
			send("500 unknown command")
		}
	}
}

// ── Тестүүд ──────────────────────────────────────────────────

const pdfContent = "%PDF-1.4 e2e source file"

func ftpFiles() map[string][]byte {
	return map[string][]byte{
		"mgis/011/01107/abc-6a7e697d329b7.pdf": []byte(pdfContent),
	}
}

// ip + нэвтрэх нэр/код-оор файл татна.
func TestFTPSourceFileClient_OpenWithCredentials(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	srv.wantUser, srv.wantPass = "gis_reader", "s3cret"

	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "gis_reader", Password: "s3cret",
	}, 5*time.Second)

	f, err := client.Open(context.Background(), "", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
	assert.Equal(t, int64(len(pdfContent)), f.SizeBytes, "SIZE-аар хэмжээ тодорхойлогдоно")
	assert.Contains(t, f.ContentType, "pdf", "өргөтгөлөөс MIME төрөл")

	user, pass := srv.credentials()
	assert.Equal(t, "gis_reader", user)
	assert.Equal(t, "s3cret", pass)
}

// SOURCE_FTP_ID тохируулсан үед хавсралтын ftp_id ТААРВАЛ татна.
func TestFTPSourceFileClient_MatchingFtpID(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: srv.addr(),
	}, 5*time.Second)

	f, err := client.Open(context.Background(), "3", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
}

// ftp_id ТААРАХГҮЙ бол өөр сервер дээрх файл — хаяг бидэнд байхгүй.
func TestFTPSourceFileClient_MismatchedFtpID(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: srv.addr(),
	}, 5*time.Second)

	_, err := client.Open(context.Background(), "1", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	assert.ErrorIs(t, err, domain.ErrSourceFileNotConfigured)
	assert.Empty(t, srv.issued(), "таарахгүй үед серверт холбогдох ёсгүй")
}

// SOURCE_FTP_ID хоосон бол БҮХ ftp_id-г энэ сервер хариуцна.
func TestFTPSourceFileClient_EmptyFtpIDServesAll(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	for _, ftpID := range []string{"", "1", "3", "99"} {
		f, err := client.Open(context.Background(), ftpID, "mgis/011/01107/abc-6a7e697d329b7.pdf")
		require.NoError(t, err, "ftp_id=%q", ftpID)
		f.Content.Close()
	}
}

// Файл байхгүй бол 550 → ErrNotFound (500 биш).
func TestFTPSourceFileClient_NotFound(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	_, err := client.Open(context.Background(), "", "mgis/байхгүй.pdf")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Буруу нэвтрэх мэдээлэл — 530.
func TestFTPSourceFileClient_LoginFails(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	srv.wantUser, srv.wantPass = "gis_reader", "s3cret"

	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), Username: "gis_reader", Password: "буруу",
	}, 5*time.Second)

	_, err := client.Open(context.Background(), "", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	assert.ErrorIs(t, err, domain.ErrInternalServer)
}

// SIZE-г дэмждэггүй сервер дээр ч файл татагдана (хэмжээ -1).
func TestFTPSourceFileClient_WithoutSIZE(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	srv.noSize = true

	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)
	f, err := client.Open(context.Background(), "", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
	assert.Equal(t, int64(-1), f.SizeBytes)
}

// Нэвтрэх нэр хоосон бол анонимоор нэвтэрнэ.
func TestFTPSourceFileClient_AnonymousLogin(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	f, err := client.Open(context.Background(), "", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	f.Content.Close()

	user, _ := srv.credentials()
	assert.Equal(t, "anonymous", user)
}

// Зам нь итгэмжлэгдээгүй: ".." нь серверийн хүрээнээс гаргах эрсдэлтэй.
func TestFTPSourceFileClient_RejectsPathTraversal(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	for _, badPath := range []string{"../etc/passwd", "mgis/../../etc/passwd", `mgis\..\..\secret.pdf`, "  "} {
		_, err := client.Open(context.Background(), "", badPath)
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "зам: %q", badPath)
	}
	assert.Empty(t, srv.issued(), "буруу зам дээр серверт холбогдох ёсгүй")
}

// Файлыг хаахад FTP холболт мөн хаагдана (session хуримтлагдахгүй).
func TestFTPSourceFileClient_ClosesConnection(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	f, err := client.Open(context.Background(), "", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	_, err = io.ReadAll(f.Content)
	require.NoError(t, err)
	require.NoError(t, f.Content.Close())

	// QUIT илгээгдсэн байх ёстой. Сан нь QUIT-ийн 221 хариуг ХҮЛЭЭДЭГГҮЙ тул
	// сервер бичлэг хийхийг хэсэг хүлээнэ.
	assert.Eventually(t, func() bool {
		for _, cmd := range srv.issued() {
			if strings.HasPrefix(cmd, "QUIT") {
				return true
			}
		}
		return false
	}, 3*time.Second, 20*time.Millisecond,
		"холболт QUIT-ээр хаагдах ёстой: %v", srv.issued())
}

// Тохируулаагүй бол ErrSourceFileNotConfigured.
func TestFTPSourceFileClient_NotConfigured(t *testing.T) {
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{}, 5*time.Second)
	_, err := client.Open(context.Background(), "1", "a.pdf")
	assert.ErrorIs(t, err, domain.ErrSourceFileNotConfigured)
}

// ── ГУС-ийн хавсралтыг FTP-ээр ХАРАХ (service давхарга) ──────
//
// Урсгал: DB-д зөвхөн зам (file_url) + ftp_id хадгалагдана → API нь FTP
// сервертэй холбогдож урсгалаар дамжуулна. Файл нь ХААНА Ч (MinIO, диск)
// ХУУЛАГДАХГҮЙ — татахгүйгээр харах зарчим.
func TestLandService_OpenParcelDocumentFile_ViaFTP(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	srv.wantUser, srv.wantPass = "gis_reader", "s3cret"

	landRepo := &mocks.MockLandAcquisitionRepository{}
	// MinIO нь DisabledStorage — хуулах ямар ч боломжгүй байхад ажиллах ёстой.
	sourceFiles := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		FtpID: "3", Addr: srv.addr(), Username: "gis_reader", Password: "s3cret",
	}, 5*time.Second)
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, sourceFiles)

	parcelID, docID := uuid.New(), uuid.New()
	landRepo.On("GetParcelDocument", ctx, parcelID, docID).Return(&domain.Document{
		ID:          docID,
		Name:        "Гэрчилгээ",
		FileType:    "pdf",
		FileURL:     "mgis/011/01107/abc-6a7e697d329b7.pdf",
		SourceDocID: "77",
		SourceFtpID: "3",
	}, nil)

	f, err := svc.OpenParcelDocumentFile(ctx, parcelID, docID)
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
	assert.Equal(t, "Гэрчилгээ.pdf", f.Name, "татахад харагдах нэр")
	assert.Contains(t, f.ContentType, "pdf")

	user, _ := srv.credentials()
	assert.Equal(t, "gis_reader", user, "FTP-д тохируулсан хэрэглэгчээр нэвтрэх ёстой")
}

// ── file_url нь БҮТЭН URL байх тохиолдол ─────────────────────

// ГУС нь ftp:// хаяг буцаавал URL нь СЕРВЕРИЙГ өөрөө заана (тохиргооны
// анхдагч хостыг дарна), нэвтрэх мэдээлэл нь тохиргооноос нөхөгдөнө.
func TestFTPSourceFileClient_AbsoluteFTPURL(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	srv.wantUser, srv.wantPass = "gis_reader", "s3cret"

	// Анхдагч хост нь ӨӨР (хүрэх боломжгүй) — URL дээрх хост хэрэглэгдэх ёстой.
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		Addr: "127.0.0.1:1", Username: "gis_reader", Password: "s3cret",
	}, 5*time.Second)

	f, err := client.Open(context.Background(), "",
		"ftp://"+srv.addr()+"/mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))

	user, pass := srv.credentials()
	assert.Equal(t, "gis_reader", user, "нэвтрэх мэдээлэл тохиргооноос нөхөгдөх ёстой")
	assert.Equal(t, "s3cret", pass)
}

// ftp:// хаяг дотор нэвтрэх мэдээлэл байвал түүнийг хэрэглэнэ.
func TestFTPSourceFileClient_AbsoluteFTPURLWithCredentials(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	srv.wantUser, srv.wantPass = "urluser", "urlpass"

	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		Addr: "127.0.0.1:1", Username: "cfguser", Password: "cfgpass",
	}, 5*time.Second)

	f, err := client.Open(context.Background(), "",
		"ftp://urluser:urlpass@"+srv.addr()+"/mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	f.Content.Close()

	user, pass := srv.credentials()
	assert.Equal(t, "urluser", user)
	assert.Equal(t, "urlpass", pass)
}

// Дэмжигдэхгүй схем — татах нь ЗӨВХӨН FTP-ээр хийгдэнэ (http ч дэмжигдэхгүй).
func TestFTPSourceFileClient_RejectsUnsupportedScheme(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	for _, ref := range []string{
		"http://10.0.0.5:8080/a.pdf", "https://host/a.pdf",
		"sftp://host/a.pdf", "file:///etc/passwd", "ftp://" + srv.addr() + "/",
	} {
		_, err := client.Open(context.Background(), "", ref)
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "ref: %s", ref)
	}
}

// "/"-ээр эхэлсэн (серверийн үндэснээс) зам ажиллана.
func TestFTPSourceFileClient_LeadingSlashPath(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	f, err := client.Open(context.Background(), "", "/mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))
}

// root_dir тохируулсан үед харьцангуй зам түүн дээр залгагдана
// (ГУС: root_dir=/DATA → RETR /DATA/mgis/011/…).
func TestFTPSourceFileClient_JoinsRootDir(t *testing.T) {
	srv := newFakeFTPServer(t, map[string][]byte{
		"DATA/mgis/011/abc.pdf": []byte(pdfContent),
	})
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{
		Addr: srv.addr(), RootDir: "/DATA",
	}, 5*time.Second)

	f, err := client.Open(context.Background(), "", "mgis/011/abc.pdf")
	require.NoError(t, err)
	defer f.Content.Close()

	body, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, pdfContent, string(body))

	var retr string
	for _, cmd := range srv.issued() {
		if strings.HasPrefix(cmd, "RETR ") {
			retr = strings.TrimPrefix(cmd, "RETR ")
		}
	}
	assert.Equal(t, "/DATA/mgis/011/abc.pdf", retr, "root_dir залгагдах ёстой")
}

// root_dir хоосон бол зам хэвээр.
func TestFTPSourceFileClient_NoRootDir(t *testing.T) {
	srv := newFakeFTPServer(t, ftpFiles())
	client := service.NewFTPSourceFileClient(service.SourceFTPTarget{Addr: srv.addr()}, 5*time.Second)

	f, err := client.Open(context.Background(), "", "mgis/011/01107/abc-6a7e697d329b7.pdf")
	require.NoError(t, err)
	f.Content.Close()

	for _, cmd := range srv.issued() {
		if strings.HasPrefix(cmd, "RETR ") {
			assert.Equal(t, "RETR mgis/011/01107/abc-6a7e697d329b7.pdf", cmd)
		}
	}
}
