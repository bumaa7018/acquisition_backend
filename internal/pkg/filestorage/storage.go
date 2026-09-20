package filestorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type StoredFile struct {
	URL       string
	Name      string
	Type      string
	SizeBytes int64
}

// PresignedUpload — browser файлыг ШУУД файлын систем руу тавих зөвшөөрөл.
//
// API нь байтыг дамжуулахгүй тул файлын хэмжээ API-ийн санах ой/хугацаанаас
// хамаарахгүй — хэдэн GB ортофотог ч байршуулж болно.
type PresignedUpload struct {
	// URL — browser ЭНЭ URL руу PUT хийнэ (зөвшөөрөл нь URL дотор шифрлэгдсэн)
	URL string `json:"url"`
	// Method — үргэлж PUT
	Method string `json:"method"`
	// FileURL — байршуулалт бүтсэний дараа DB-д бичигдэх харьцангуй зам
	FileURL string `json:"file_url"`
	// StoredName — хадгалалтад үүсэх файлын нэр (мозайкийн `location`)
	StoredName string `json:"stored_name"`
	// ExpiresInSeconds — зөвшөөрлийн хүчинтэй хугацаа
	ExpiresInSeconds int `json:"expires_in_seconds"`
}

// ObjectStat — хадгалалтад БОДИТООР байгаа объектын мэдээлэл. Байршуулсан
// хэмжээг клиентээс АВАХГҮЙ, эндээс уншина.
type ObjectStat struct {
	SizeBytes   int64
	ContentType string
}

// DirectUploader — browser-аас шууд байршуулалтыг дэмждэг хадгалалт (S3/MinIO).
// Байтыг API-аар дамжуулахгүй тул бүх шалгалт байршуулсны ДАРАА хадгалалт дээрх
// объект дээр хийгдэнэ (Stat + ReadRange).
type DirectUploader interface {
	PresignUpload(ctx context.Context, namespace, ownerID, filename, contentType string, expiry time.Duration) (*PresignedUpload, error)
	// URLFor — нэрнээс file_url-ийг бүтээнэ. Бүртгэх үед замыг СЕРВЕР өөрөө
	// гаргаж авахад хэрэглэнэ — клиентээс зам хүлээж авбал өөр эзэмшигчийн
	// объектыг бүртгэх боломж үүснэ.
	URLFor(namespace, ownerID, storedName string) string
	Stat(ctx context.Context, fileURL string) (*ObjectStat, error)
	// ReadRange — объектын зөвхөн эхний хэсгийг унших (magic bytes шалгах).
	ReadRange(ctx context.Context, fileURL string, offset, length int64) ([]byte, error)
	// ObjectURL — хадгалалт дээрх объектын ШУУД хаяг (жишээ:
	// http://minio:9000/gov-files/drone/<id>/<нэр>.tif).
	//
	// GeoServer растрыг ЭНЭ хаягаар HTTP Range хүсэлтээр уншина — иймд
	// сүлжээний дотоод хаяг байх ёстой (frontend-ийн /api/files proxy биш).
	ObjectURL(fileURL string) (string, error)
}

type Storage interface {
	// unique=true үед файлын нэрэнд uuid угтвар нэмж давхардлаас сэргийлнэ;
	// false үед өгсөн нэрээр нь яг хадгална (дуудагч давхардлыг өөрөө хариуцна).
	Save(ctx context.Context, namespace, ownerID, filename, contentType string, data []byte, unique bool) (*StoredFile, error)
	// SaveStream — файлыг бүхэлд нь санах ойд уншихгүйгээр урсгалаар хадгална.
	// Дроны ортофото зэрэг зуу-мянган MB файл дээр Save() нь файлын хэмжээгээр
	// RAM зарцуулж процессыг унагадаг тул том файлд ҮРГЭЛЖ үүнийг хэрэглэнэ.
	SaveStream(ctx context.Context, namespace, ownerID, filename, contentType string, r io.Reader, size int64, unique bool) (*StoredFile, error)
	Read(ctx context.Context, fileURL string) ([]byte, error)
	// OpenStream — файлыг санах ойд бүтнээр уншихгүйгээр урсгалаар нээнэ.
	// Дуудагч хаах хариуцлагатай.
	OpenStream(ctx context.Context, fileURL string) (io.ReadCloser, error)
	Delete(ctx context.Context, fileURL string) error
}

// sanitizeObjectName — ОБЪЕКТЫН нэрэнд ашиглах аюулгүй хэлбэрт оруулна.
//
// MinIO/S3 сервер зарим тэмдэгтийг зөвшөөрдөггүй ("Objectname contains
// unsupported characters" 500 алдаа) — хэрэглэгчийн файлын нэрэнд `:`, `?`,
// `*`, `"`, `<`, `>`, `|`, `\` эсвэл удирдлагын тэмдэгт орвол БАЙРШУУЛАЛТ
// БҮТЭЛГҮЙТДЭГ. Нэрийг харуулахдаа ЭХ нэрийг (cleanName) ашигладаг тул энд
// зөвхөн ХАДГАЛАХ нэрийг цэвэрлэнэ — кирилл үсэг (UTF-8) хэвээр үлдэнэ.
func sanitizeObjectName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 0x20 || r == 0x7f: // удирдлагын тэмдэгт
			b.WriteByte('_')
		case strings.ContainsRune(`\/:*?"<>|`, r):
			b.WriteByte('_')
		case r == ' ':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		out = "file"
	}
	if len(out) > 180 {
		out = out[:180]
	}
	return out
}

// storedFileName — хадгалах нэрийг бэлдэнэ. Adapter-ууд (локал ба S3) ижил
// дүрмээр нэрлэснээр шилжүүлэлт болон URL-ийн бүтэц таарна.
//
// cleanName — хэрэглэгчид ХАРУУЛАХ эх нэр, storedName — объектын түлхүүрт
// орох цэвэршүүлсэн нэр.
func storedFileName(filename string, unique bool) (cleanName, storedName string) {
	cleanName = filepath.Base(filename)
	storedName = sanitizeObjectName(cleanName)
	if unique {
		storedName = uuid.New().String() + "_" + storedName
	}
	return cleanName, storedName
}

type DisabledStorage struct{}

func (DisabledStorage) Save(context.Context, string, string, string, string, []byte, bool) (*StoredFile, error) {
	return nil, errors.New("file storage is not configured")
}

func (DisabledStorage) SaveStream(context.Context, string, string, string, string, io.Reader, int64, bool) (*StoredFile, error) {
	return nil, errors.New("file storage is not configured")
}

func (DisabledStorage) Read(context.Context, string) ([]byte, error) {
	return nil, errors.New("file storage is not configured")
}

func (DisabledStorage) OpenStream(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("file storage is not configured")
}

func (DisabledStorage) Delete(context.Context, string) error {
	return nil
}

type MockStorage struct {
	BaseURL  string
	LocalDir string
}

func NewMockStorage(baseURL, localDir string) *MockStorage {
	return &MockStorage{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		LocalDir: localDir,
	}
}

func (s *MockStorage) Save(ctx context.Context, namespace, ownerID, filename, contentType string, data []byte, unique bool) (*StoredFile, error) {
	return s.SaveStream(ctx, namespace, ownerID, filename, contentType, bytes.NewReader(data), int64(len(data)), unique)
}

func (s *MockStorage) SaveStream(_ context.Context, namespace, ownerID, filename, contentType string, r io.Reader, size int64, unique bool) (*StoredFile, error) {
	if s.BaseURL == "" || s.LocalDir == "" {
		return nil, errors.New("file storage is not configured")
	}

	cleanName, storedName := storedFileName(filename, unique)
	relativePath := filepath.Join(namespace, ownerID, storedName)
	localPath := filepath.Join(s.LocalDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, err
	}
	written, err := writeFileStream(localPath, r)
	if err != nil {
		return nil, err
	}
	if size < 0 {
		size = written
	}

	// URL-ийн сегмент бүрийг escape хийнэ — эс бөгөөс нэрэнд '%', зай зэрэг
	// тэмдэгт орсон файлын URL хүчингүй болж, Delete/Read болон browser-ийн
	// линк ажиллахгүй (файл орфон үлддэг байсан баг).
	segments := strings.Split(filepath.ToSlash(relativePath), "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}

	return &StoredFile{
		URL:       s.BaseURL + "/" + strings.Join(segments, "/"),
		Name:      cleanName,
		Type:      contentType,
		SizeBytes: size,
	}, nil
}

// writeFileStream — уншигчийг файл руу урсгалаар бичнэ (санах ойд бүтнээр
// хуулахгүй). Алдаа гарвал хагас бичигдсэн файлыг үлдээхгүй.
func writeFileStream(path string, r io.Reader) (int64, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return 0, err
	}
	return n, nil
}

func (s *MockStorage) Read(_ context.Context, fileURL string) ([]byte, error) {
	localPath, err := s.localPath(fileURL)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(localPath)
}

func (s *MockStorage) OpenStream(_ context.Context, fileURL string) (io.ReadCloser, error) {
	localPath, err := s.localPath(fileURL)
	if err != nil {
		return nil, err
	}
	return os.Open(localPath)
}

func (s *MockStorage) Delete(_ context.Context, fileURL string) error {
	localPath, err := s.localPath(fileURL)
	if err != nil {
		return err
	}
	if err := os.Remove(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *MockStorage) localPath(fileURL string) (string, error) {
	// Хуучин бичлэгүүдийн URL escape хийгдээгүй (түүхий '%', зай агуулсан) байж
	// болзошгүй тул url.Parse-д найдалгүй эхлээд шууд prefix-ээр тасалж авна.
	if rel, ok := strings.CutPrefix(fileURL, s.BaseURL+"/"); ok {
		// Шинэ (escape хийгдсэн) URL бол буцааж тайлна; түүхий хуучин URL дээр
		// PathUnescape алдаа өгвөл байгаагаар нь ашиглана.
		if decoded, err := url.PathUnescape(rel); err == nil {
			rel = decoded
		}
		if rel == "" || strings.Contains(rel, "..") {
			return "", errors.New("invalid file URL")
		}
		return filepath.Join(s.LocalDir, filepath.FromSlash(rel)), nil
	}

	base, err := url.Parse(s.BaseURL)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(fileURL)
	if err != nil {
		return "", err
	}

	relativePath := strings.TrimPrefix(u.Path, strings.TrimRight(base.Path, "/")+"/")
	if relativePath == u.Path {
		return "", errors.New("file URL is outside configured mock file server")
	}
	return filepath.Join(s.LocalDir, filepath.FromSlash(relativePath)), nil
}
