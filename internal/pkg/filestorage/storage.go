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
	Save(ctx context.Context, namespace, ownerID, filename, contentType string, data []byte) (*StoredFile, error)
	Read(ctx context.Context, fileURL string) ([]byte, error)
	Delete(ctx context.Context, fileURL string) error
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

func (s *MockStorage) Save(_ context.Context, namespace, ownerID, filename, contentType string, data []byte) (*StoredFile, error) {
	if s.BaseURL == "" || s.LocalDir == "" {
		return nil, errors.New("file storage is not configured")
	}

	cleanName := filepath.Base(filename)
	relativePath := filepath.Join(namespace, ownerID, uuid.New().String()+"_"+cleanName)
	localPath := filepath.Join(s.LocalDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return nil, err
	}

	return &StoredFile{
		URL:       s.BaseURL + "/" + filepath.ToSlash(relativePath),
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
