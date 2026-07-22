package http

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"acquisition_backend/internal/domain"
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

const (
	docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	// docxAllowedDocumentType — Гэрээг хэвлэхэд Хурлын тэмдэглэлийг Word
	// баримт байдлаар нэгтгэдэг тул зөвхөн энэ төрөлд PDF-ээс гадна DOCX зөвшөөрнө.
	docxAllowedDocumentType = "meeting_minutes"
)

// isDocxFile — .docx (Office Open XML) файл эсэхийг өргөтгөл + zip гарын үсгээр шалгана.
// http.DetectContentType нь docx-ийн далд zip бүтцийг тусгайлан танихгүй
// ("application/zip" гэж буцаадаг тул) энд өргөтгөлтэй хослуулж шалгана.
func isDocxFile(filename string, data []byte) bool {
	if !strings.HasSuffix(strings.ToLower(filename), ".docx") {
		return false
	}
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

// allowedDocumentContentType — хавсралт байршуулах үед агуулгын төрлийг шалгана.
// Ерөнхийдөө зөвхөн PDF зөвшөөрнө; "Хурлын тэмдэглэл" (meeting_minutes) төрөлд
// Гэрээтэй хэвлэхдээ нэгтгэдэг тул Word (docx) файл нэмж зөвшөөрнө.
func allowedDocumentContentType(filename string, data []byte, dt *domain.DocumentType) (string, bool) {
	if dt != nil && dt.Type == docxAllowedDocumentType && isDocxFile(filename, data) {
		return docxContentType, true
	}
	return allowedPDFContentType(data)
}
