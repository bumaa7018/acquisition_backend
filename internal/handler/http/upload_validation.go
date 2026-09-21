package http

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"acquisition_backend/internal/domain"
)

// Файл оруулах дээд хэмжээ — frontend-ийн шалгалттай ижил (20MB).
const (
	maxDocumentUploadSize = 20 * 1024 * 1024
	maxAssetPhotoSize     = 20 * 1024 * 1024
	maxShapefileSize      = 20 * 1024 * 1024
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

// allowedEstimatedValueContentType — ТӨСӨӨЛЛИЙН үнэлгээний хавсралт.
//
// PDF-ээс гадна ЗУРАГ (JPEG/PNG) зөвшөөрнө: жишиг үнийн лавлагаа, зах зээлийн
// судалгааг ихэвчлэн сканнердсан/зурагдсан хэлбэрээр хавсаргадаг ба тэр
// файлыг PDF болгохыг шаардах нь ажлыг л хүндрүүлнэ. Гүйцэтгэх файл
// (exe/zip г.м.) хэзээ ч зөвшөөрөгдөхгүй — DetectContentType-ийн шалгалт нь
// өргөтгөл биш АГУУЛГААР шийднэ.
func allowedEstimatedValueContentType(data []byte) (string, bool) {
	contentType := detectContentType(data)
	switch contentType {
	case "application/pdf", "image/jpeg", "image/png":
		return contentType, true
	}
	return contentType, false
}

const (
	docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	// xlsxAllowedDocumentType — "Үнэлгээний хүснэгт" нь Excel файл ӨӨРӨӨ
	// (үнэлгээг оруулсан эх хүснэгт) тул зөвхөн энэ төрөлд .xlsx зөвшөөрнө.
	xlsxAllowedDocumentType = "valuation_source"
	// docxAllowedDocumentType — Гэрээг хэвлэхэд Хурлын тэмдэглэлийг Word
	// баримт байдлаар нэгтгэдэг тул зөвхөн энэ төрөлд PDF-ээс гадна DOCX зөвшөөрнө.
	docxAllowedDocumentType = "meeting_minutes"
	// valuationReportDocumentType — "Үнэлгээний тайлан". Үнэлгээний
	// байгууллагууд тайлангаа Word-ээр бэлддэг тул PDF-ээс гадна DOCX зөвшөөрнө.
	valuationReportDocumentType = "valuation_report"
	// workPhotoDocumentType — "Ажлын зураг". Хээрийн судалгааны зургууд нь
	// гар утаснаас шууд ордог тул JPEG/PNG зөвшөөрнө (PDF-д хөрвүүлэхийг
	// шаардах нь ажлыг л хүндрүүлнэ).
	workPhotoDocumentType = "work_photo"
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

// isXlsxFile — .xlsx (Office Open XML) эсэхийг өргөтгөл + zip гарын үсгээр шалгана
// (docx-тэй ижил шалтгаан: DetectContentType нь "application/zip" гэж буцаана).
func isXlsxFile(filename string, data []byte) bool {
	if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return false
	}
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

// allowedDocumentContentType — хавсралт байршуулах үед агуулгын төрлийг шалгана.
// Ерөнхийдөө зөвхөн PDF зөвшөөрнө; "Хурлын тэмдэглэл" (meeting_minutes) төрөлд
// Гэрээтэй хэвлэхдээ нэгтгэдэг тул Word (docx), "Үнэлгээний хүснэгт"
// (valuation_source) төрөлд үнэлгээг оруулсан эх Excel (xlsx) зөвшөөрнө.
func allowedDocumentContentType(filename string, data []byte, dt *domain.DocumentType) (string, bool) {
	if dt != nil && dt.Type == docxAllowedDocumentType && isDocxFile(filename, data) {
		return docxContentType, true
	}
	if dt != nil && dt.Type == valuationReportDocumentType && isDocxFile(filename, data) {
		return docxContentType, true
	}
	if dt != nil && dt.Type == workPhotoDocumentType {
		if contentType := detectContentType(data); contentType == "image/jpeg" || contentType == "image/png" {
			return contentType, true
		}
	}
	if dt != nil && dt.Type == xlsxAllowedDocumentType && isXlsxFile(filename, data) {
		return xlsxContentType, true
	}
	return allowedPDFContentType(data)
}
