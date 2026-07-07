package filestorage

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type StoredFile struct {
	URL       string
	Name      string
	Type      string
	SizeBytes int64
}

type Storage interface {
	// unique=true үед файлын нэрэнд uuid угтвар нэмж давхардлаас сэргийлнэ;
	// false үед өгсөн нэрээр нь яг хадгална (дуудагч давхардлыг өөрөө хариуцна).
	Save(ctx context.Context, namespace, ownerID, filename, contentType string, data []byte, unique bool) (*StoredFile, error)
	Read(ctx context.Context, fileURL string) ([]byte, error)
	Delete(ctx context.Context, fileURL string) error
}

type DisabledStorage struct{}

func (DisabledStorage) Save(context.Context, string, string, string, string, []byte, bool) (*StoredFile, error) {
	return nil, errors.New("file storage is not configured")
}

func (DisabledStorage) Read(context.Context, string) ([]byte, error) {
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

func (s *MockStorage) Save(_ context.Context, namespace, ownerID, filename, contentType string, data []byte, unique bool) (*StoredFile, error) {
	if s.BaseURL == "" || s.LocalDir == "" {
		return nil, errors.New("file storage is not configured")
	}

	cleanName := filepath.Base(filename)
	storedName := cleanName
	if unique {
		storedName = uuid.New().String() + "_" + cleanName
	}
	relativePath := filepath.Join(namespace, ownerID, storedName)
	localPath := filepath.Join(s.LocalDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return nil, err
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
		SizeBytes: int64(len(data)),
	}, nil
}

func (s *MockStorage) Read(_ context.Context, fileURL string) ([]byte, error) {
	localPath, err := s.localPath(fileURL)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(localPath)
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
