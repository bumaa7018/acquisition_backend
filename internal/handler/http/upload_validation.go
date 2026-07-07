package http

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Файл оруулах дээд хэмжээ — frontend-ийн 50MB шалгалттай ижил.
const (
	maxDocumentUploadSize = 50 * 1024 * 1024
	maxAssetPhotoSize     = 50 * 1024 * 1024
	maxShapefileSize      = 50 * 1024 * 1024
	maxShapefileFormSize  = maxShapefileSize + 1024*1024
)

var errUploadTooLarge = errors.New("upload too large")

func readMultipartFile(fh *multipart.FileHeader, maxSize int64) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, errUploadTooLarge
	}
	return data, nil
}

func isBodyTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "request body too large")
}

func detectContentType(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) > 512 {
		data = data[:512]
	}
	return http.DetectContentType(data)
}

func allowedPDFContentType(data []byte) (string, bool) {
	contentType := detectContentType(data)
	return contentType, contentType == "application/pdf"
}
