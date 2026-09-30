package http

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"acquisition_backend/internal/legacyimport"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LegacyImportHandler — хуучин мэдээллийн импорт (зөвхөн админ).
//
//	GET    /legacy-import/categories
//	GET    /legacy-import/sessions            POST /legacy-import/sessions
//	GET    /legacy-import/sessions/:id        DELETE /legacy-import/sessions/:id
//	POST   /legacy-import/sessions/:id/boundary|parcels|reports  (multipart: files[], category, subCategory)
//	POST   /legacy-import/sessions/:id/commit
type LegacyImportHandler struct {
	svc *legacyimport.Service
}

func NewLegacyImportHandler(svc *legacyimport.Service) *LegacyImportHandler {
	return &LegacyImportHandler{svc: svc}
}

const (
	// maxLegacyImportUpload — нэг хүсэлтийн нийт хэмжээ (shapefile-ийн бүх хэсэг
	// эсвэл хэд хэдэн эксэл).
	maxLegacyImportUpload = 300 << 20
	maxLegacyImportFile   = 200 << 20
)

// legacyImportError — импортын алдааг мессежтэй нь HTTP хариу болгоно.
func legacyImportError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	var step *legacyimport.StepError
	switch {
	case errors.Is(err, legacyimport.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, legacyimport.ErrCommitted), errors.Is(err, legacyimport.ErrBusy), errors.As(err, &step):
		status = http.StatusConflict
	}
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "Импортын явцад алдаа гарлаа: " + msg
	}
	c.AbortWithStatusJSON(status, response.Response{Code: status, Error: msg, Message: msg})
}

func (h *LegacyImportHandler) sessionID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Code: http.StatusBadRequest,
			Error: "Импортын дугаар буруу", Message: "Импортын дугаар буруу"})
		return "", false
	}
	return id, true
}

func (h *LegacyImportHandler) Categories(c *gin.Context) {
	list, err := h.svc.Categories(c.Request.Context())
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *LegacyImportHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *LegacyImportHandler) Create(c *gin.Context) {
	id, err := h.svc.Create(c.Request.Context(), contextUserDisplayName(c))
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

func (h *LegacyImportHandler) Get(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	state, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, state)
}

func (h *LegacyImportHandler) Delete(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

// Progress — ГУС-ын шалгалт / оруулалтын явц (бүтэн төлөвгүй, хөнгөн).
func (h *LegacyImportHandler) Progress(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	progress, err := h.svc.Progress(c.Request.Context(), id)
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, progress)
}

// RollbackPreview — «Хасах»-ын өмнө юу устахыг харуулна (юу ч өөрчлөхгүй).
func (h *LegacyImportHandler) RollbackPreview(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	preview, err := h.svc.RollbackPreview(c.Request.Context(), id)
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, preview)
}

// Rollback — оруулсан импортыг холбоотой бүх мэдээлэлтэй нь бүрэн хасна.
// Баталгаажуулалт: body {"confirm": "<импортын дугаар>"}.
func (h *LegacyImportHandler) Rollback(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	var body struct {
		Confirm string `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Confirm != id {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Code: http.StatusBadRequest,
			Error: "Хасахыг баталгаажуулаагүй байна", Message: "Хасахыг баталгаажуулаагүй байна"})
		return
	}
	result, err := h.svc.Rollback(c.Request.Context(), id, contextUserDisplayName(c))
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, result)
}

// readUploads — multipart "files" талбарын бүх файл (зөвхөн basename).
func readUploads(c *gin.Context) ([]legacyimport.UploadedFile, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLegacyImportUpload)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		msg := "multipart/form-data хэлбэрээр файл илгээнэ үү"
		if isBodyTooLarge(err) {
			msg = "Файл хэт том (нийт 300 MB-аас их)"
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Code: http.StatusBadRequest, Error: msg, Message: msg})
		return nil, false
	}
	var files []legacyimport.UploadedFile
	for _, fh := range c.Request.MultipartForm.File["files"] {
		data, err := readMultipartFile(fh, maxLegacyImportFile)
		if err != nil {
			msg := fh.Filename + ": файлыг уншиж чадсангүй"
			if errors.Is(err, errUploadTooLarge) {
				msg = fh.Filename + ": файл хэт том (200 MB-аас их)"
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Code: http.StatusBadRequest, Error: msg, Message: msg})
			return nil, false
		}
		name := strings.NewReplacer("/", "_", "\\", "_").Replace(filepath.Base(fh.Filename))
		files = append(files, legacyimport.UploadedFile{Name: name, Data: data})
	}
	return files, true
}

func (h *LegacyImportHandler) Upload(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	files, ok := readUploads(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	var (
		state *legacyimport.State
		err   error
	)
	switch c.Param("step") {
	case "boundary":
		state, err = h.svc.UploadBoundary(ctx, id, files, legacyimport.CategoryChoice{
			General: c.PostForm("category"), Sub: c.PostForm("subCategory")})
	case "parcels":
		state, err = h.svc.UploadParcels(ctx, id, files)
	case "reports":
		state, err = h.svc.UploadReports(ctx, id, files)
	default:
		c.AbortWithStatusJSON(http.StatusNotFound, response.Response{Code: http.StatusNotFound, Error: "Алхам олдсонгүй", Message: "Алхам олдсонгүй"})
		return
	}
	if err != nil {
		legacyImportError(c, err)
		return
	}
	response.OK(c, state)
}

func (h *LegacyImportHandler) Commit(c *gin.Context) {
	id, ok := h.sessionID(c)
	if !ok {
		return
	}
	if err := h.svc.Commit(c.Request.Context(), id); err != nil {
		legacyImportError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, response.Response{Code: http.StatusAccepted, Data: gin.H{"id": id, "status": "committing"}, Message: "OK"})
}
