package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
)

// SourceFile — эх системийн файл серверээс нээсэн файлын урсгал.
// Дуудагч Content-ийг ХААХ хариуцлагатай.
type SourceFile struct {
	ContentType string
	SizeBytes   int64
	Content     io.ReadCloser
}

// SourceFileClient — эх системийн (ГУС) файл сервер.
//
// Хавсралтын файл нь дундын сервисийн DB-д БАЙХГҮЙ — зөвхөн зам (file_url) ба
// аль сервер дээр байгаа нь (ftp_id) бүртгэлтэй. Файлыг browser руу шууд
// өгөхгүй: сервер дотоод сүлжээнд байж болох ба хандалтын эрх шалгагдах ёстой
// тул API нь эндээс уншиж дамжуулна.
type SourceFileClient interface {
	// Open — ftpID дээрх filePath файлыг нээнэ. ftpID хоосон бол анхдагч
	// хаяг хэрэглэгдэнэ.
	Open(ctx context.Context, ftpID, filePath string) (*SourceFile, error)
}

type httpSourceFileClient struct {
	// baseURL — анхдагч сервер (ftp_id-д тохирох тусгай хаяг байхгүй үед)
	baseURL string
	// byFtpID — ftp_id → хаяг
	byFtpID map[string]string
	http    *http.Client
}

// NewHTTPSourceFileClient — эх системийн файл серверийн client.
// baseURL болон byFtpID хоёулаа хоосон бол Open нь ErrSourceFileNotConfigured
// буцаана (сервис асахад саад болохгүй).
func NewHTTPSourceFileClient(baseURL string, byFtpID map[string]string, timeout time.Duration) SourceFileClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &httpSourceFileClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		byFtpID: byFtpID,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *httpSourceFileClient) Open(ctx context.Context, ftpID, filePath string) (*SourceFile, error) {
	reqURL, err := c.resolve(ftpID, filePath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		resp.Body.Close()
		return nil, domain.ErrInternalServer
	}

	// ContentLength нь мэдэгдэхгүй үед -1 ирнэ — дамжуулагч тэрийг зохицуулна.
	return &SourceFile{
		ContentType: resp.Header.Get("Content-Type"),
		SizeBytes:   resp.ContentLength,
		Content:     resp.Body,
	}, nil
}

// resolve — ftp_id-гаар хостыг сонгож, бүтэн URL угсарна.
func (c *httpSourceFileClient) resolve(ftpID, filePath string) (string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", domain.ErrInvalidInput
	}
	// Эх системийн өгөгдөл тул зам нь итгэмжлэгдээгүй: ".." нь тохируулсан
	// хостын хүрээнээс гаргах эрсдэлтэй.
	for _, part := range strings.Split(strings.ReplaceAll(filePath, "\\", "/"), "/") {
		if part == ".." {
			return "", domain.ErrInvalidInput
		}
	}

	base := c.baseURL
	if ftpID = strings.TrimSpace(ftpID); ftpID != "" {
		if v, ok := c.byFtpID[ftpID]; ok {
			base = v
		}
	}
	if base == "" {
		return "", domain.ErrSourceFileNotConfigured
	}

	// Зам нь аль хэдийн бүтэн URL байвал (эх систем өөрчлөгдвөл) шууд авна.
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		return filePath, nil
	}

	reqURL, err := url.JoinPath(base, filePath)
	if err != nil {
		return "", fmt.Errorf("source file url: %w", err)
	}
	return reqURL, nil
}
