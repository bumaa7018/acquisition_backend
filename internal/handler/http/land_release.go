package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

// Буулгалтын нэг оруулалтын нийт хэмжээ (олон shapefile/zip + захирамж + зураг).
const maxLandReleaseFormSize = 150 * 1024 * 1024

// Зургуудаас үүсгэсэн PDF-ийн дээд хэмжээ (frontend зургийг жижигрүүлж шахна).
const maxLandReleasePhotoSize = 50 * 1024 * 1024

// LandReleaseHandler — «Буулгалтын мэдээлэл».
type LandReleaseHandler struct {
	svc service.LandReleaseService
}

func NewLandReleaseHandler(svc service.LandReleaseService) *LandReleaseHandler {
	return &LandReleaseHandler{svc: svc}
}

func landReleaseFilter(c *gin.Context) domain.LandReleaseFilter {
	return domain.LandReleaseFilter{
		Search: c.Query("search"),
		Status: queryInt(c, "status", 0),
		Year:   queryInt(c, "year", 0),
		// Буулгасан байгууламжийн төрөл
		StructureTypeID: queryInt(c, "structure_type_id", 0),
		Page:            queryInt(c, "page", 1),
		PageSize:        queryInt(c, "page_size", 20),
	}
}

// landReleaseError — файлын уншилтын алдааг (координатын систем г.м.) файлын
// нэртэй нь шууд, бусдыг ердийн замаар.
func landReleaseError(c *gin.Context, err error) {
	var fe *service.LandReleaseFileError
	if errors.As(err, &fe) {
		response.AbortValidation(c, fe)
		return
	}
	response.Error(c, err)
}

// readShapeFiles — multipart-ын "files" талбар (олон).
func readShapeFiles(c *gin.Context) ([]service.UploadedFile, bool) {
	form, err := c.MultipartForm()
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		} else {
			response.Error(c, domain.ErrLandReleaseNoFiles)
		}
		return nil, false
	}
	var out []service.UploadedFile
	for _, fh := range form.File["files"] {
		data, err := readMultipartFile(fh, maxShapefileSize*2)
		if err == errUploadTooLarge {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return nil, false
		}
		if err != nil {
			response.Error(c, domain.ErrLandReleaseNoFiles)
			return nil, false
		}
		out = append(out, service.UploadedFile{Filename: fh.Filename, Data: data})
	}
	if len(out) == 0 {
		response.Error(c, domain.ErrLandReleaseNoFiles)
		return nil, false
	}
	return out, true
}

// readDecree — "decree_number", "decree_date", "decree_file" (PDF). Файл
// оруулаагүй бол nil (шаардлагатай эсэхийг service шийднэ).
func readDecree(c *gin.Context) (*service.LandReleaseDecreeInput, bool) {
	in := &service.LandReleaseDecreeInput{
		Number: strings.TrimSpace(c.PostForm("decree_number")),
		Date:   strings.TrimSpace(c.PostForm("decree_date")),
	}
	fh, err := c.FormFile("decree_file")
	if err != nil {
		if in.Number == "" {
			return nil, true
		}
		return in, true
	}
	data, err := readMultipartFile(fh, maxDocumentUploadSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return nil, false
	}
	if err != nil || len(data) == 0 {
		response.Error(c, domain.ErrLandReleaseDecreeRequired)
		return nil, false
	}
	contentType, ok := allowedPDFContentType(data)
	if !ok {
		response.Error(c, domain.ErrLandReleaseDecreeRequired)
		return nil, false
	}
	in.File = &service.UploadedFile{Filename: fh.Filename, ContentType: contentType, Data: data}
	return in, true
}

// readPhoto — "photo_file" (зургуудаас үүсгэсэн PDF), "photo_pages". Файл
// ирээгүй бол nil; ирсэн бол PDF байх ёстой.
func readPhoto(c *gin.Context, required bool) (*service.LandReleasePhotoInput, bool) {
	fh, err := c.FormFile("photo_file")
	if err != nil {
		if isBodyTooLarge(err) {
			response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
			return nil, false
		}
		if required {
			response.Error(c, domain.ErrLandReleasePhoto)
			return nil, false
		}
		return nil, true
	}
	data, err := readMultipartFile(fh, maxLandReleasePhotoSize)
	if err == errUploadTooLarge {
		response.Abort(c, http.StatusBadRequest, i18n.ErrFileTooLarge)
		return nil, false
	}
	if err != nil || len(data) == 0 {
		response.Error(c, domain.ErrLandReleasePhoto)
		return nil, false
	}
	contentType, ok := allowedPDFContentType(data)
	if !ok {
		response.Error(c, domain.ErrLandReleasePhoto)
		return nil, false
	}
	pages, _ := strconv.Atoi(c.PostForm("photo_pages"))
	return &service.LandReleasePhotoInput{
		File:      service.UploadedFile{Filename: fh.Filename, ContentType: contentType, Data: data},
		PageCount: pages,
	}, true
}

// AddPhoto — газарт зураг (зургуудаас үүсгэсэн PDF) нэмнэ.
func (h *LandReleaseHandler) AddPhoto(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLandReleasePhotoSize+1<<20)
	photo, ok := readPhoto(c, true)
	if !ok {
		return
	}
	lr, err := h.svc.AddPhoto(c.Request.Context(), id, photo, contextUserIDString(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, lr)
}

func (h *LandReleaseHandler) List(c *gin.Context) {
	f := landReleaseFilter(c)
	items, total, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	if items == nil {
		items = []*domain.LandRelease{}
	}
	response.Paginated(c, items, total, f.Page, f.PageSize)
}

func (h *LandReleaseHandler) Map(c *gin.Context) {
	items, err := h.svc.Map(c.Request.Context(), landReleaseFilter(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKCompressed(c, items)
}

// StructureTypes — буулгасан байгууламжийн төрлүүд.
func (h *LandReleaseHandler) StructureTypes(c *gin.Context) {
	items, err := h.svc.ListStructureTypes(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	if items == nil {
		items = []*domain.LandReleaseStructureType{}
	}
	response.OK(c, items)
}

func (h *LandReleaseHandler) Stats(c *gin.Context) {
	st, err := h.svc.Stats(c.Request.Context(), landReleaseFilter(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, st)
}

func (h *LandReleaseHandler) Get(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	lr, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, lr)
}

// Preview — файлуудыг уншиж (координатын систем шалгаж) газруудыг буцаана.
func (h *LandReleaseHandler) Preview(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLandReleaseFormSize)
	files, ok := readShapeFiles(c)
	if !ok {
		return
	}
	features, err := h.svc.Preview(c.Request.Context(), files)
	if err != nil {
		landReleaseError(c, err)
		return
	}
	response.OK(c, features)
}

// Create — нэр, тайлбар, төлөв, files[] (+ «Захирамж гарч чөлөөлсөн» бол захирамж).
func (h *LandReleaseHandler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLandReleaseFormSize)
	files, ok := readShapeFiles(c)
	if !ok {
		return
	}
	decree, ok := readDecree(c)
	if !ok {
		return
	}
	photo, ok := readPhoto(c, false)
	if !ok {
		return
	}
	status, _ := strconv.Atoi(c.PostForm("status"))
	// structure_type_id — байгаа төрөл; structure_type_name — шинэ төрөл.
	var typeID *int
	if v, err := strconv.Atoi(c.PostForm("structure_type_id")); err == nil && v > 0 {
		typeID = &v
	}
	n, err := h.svc.Create(c.Request.Context(), service.CreateLandReleaseRequest{
		Name:              c.PostForm("name"),
		Description:       c.PostForm("description"),
		Status:            status,
		StructureTypeID:   typeID,
		StructureTypeName: c.PostForm("structure_type_name"),
		Files:             files,
		Decree:            decree,
		Photo:             photo,
	}, contextUserIDString(c))
	if err != nil {
		landReleaseError(c, err)
		return
	}
	response.Created(c, gin.H{"created": n})
}

// ChangeStatus — status, note (+ «Захирамж гарч чөлөөлсөн» бол захирамж).
func (h *LandReleaseHandler) ChangeStatus(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDocumentUploadSize+1<<20)
	decree, ok := readDecree(c)
	if !ok {
		return
	}
	status, _ := strconv.Atoi(c.PostForm("status"))
	lr, err := h.svc.ChangeStatus(c.Request.Context(), id, service.ChangeLandReleaseStatusRequest{
		Status: status,
		Note:   c.PostForm("note"),
		Decree: decree,
	}, contextUserIDString(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, lr)
}
