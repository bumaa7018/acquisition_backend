package http

import (
	"errors"
	"net/http"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type droneImageResponse struct {
	*domain.DroneImage
	// LayerName — WMS-д явуулах давхаргын нэр. Зураг тус бүр өөрийн
	// давхаргатай (GeoServer растрыг файлын системээс шууд уншина) тул
	// нэр нь зураг бүрт ӨӨР.
	LayerName string `json:"layer_name"`
	// PublishError — GeoServer-т давхарга үүсгэхэд гарсан алдаа (байвал).
	// Зураг хадгалалтад бүтэн орсон тул хүсэлт нь бүтсэн гэж тооцогдоно —
	// гэхдээ шалтгааныг ХАРУУЛАХГҮЙ бол хэрэглэгч "Шинэчлэх дарна уу" гэсэн
	// ерөнхий мессежээс өөр юу ч харахгүй, шалтгаан серверийн лог дээр үлдэнэ.
	PublishError string `json:"publish_error,omitempty"`
}

func (h *LandHandler) toDroneImageResponse(img *domain.DroneImage) droneImageResponse {
	return droneImageResponse{
		DroneImage: img,
		LayerName:  h.svc.DroneLayerName(img.FileName),
	}
}

func (h *LandHandler) toDroneImageResponses(list []*domain.DroneImage) []droneImageResponse {
	out := make([]droneImageResponse, 0, len(list))
	for _, img := range list {
		out = append(out, h.toDroneImageResponse(img))
	}
	return out
}

// droneError — service давхаргын дроны алдааг HTTP статус болгоно.
func droneError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDroneFileType):
		response.Abort(c, http.StatusBadRequest, i18n.ErrDroneFileType)
	case errors.Is(err, service.ErrDroneObjectMissing):
		response.Abort(c, http.StatusBadRequest, i18n.ErrDroneUploadIncomplete)
	case errors.Is(err, service.ErrDroneAlreadyRegistered):
		response.Abort(c, http.StatusConflict, i18n.ErrDroneAlreadyRegistered)
	case errors.Is(err, service.ErrDroneTooLarge):
		response.Abort(c, http.StatusRequestEntityTooLarge, i18n.ErrDroneTooLarge)
	case errors.Is(err, service.ErrDroneNotConfigured),
		errors.Is(err, service.ErrDroneDirectUploadUnsupported):
		response.Abort(c, http.StatusServiceUnavailable, i18n.ErrInternalServer)
	default:
		response.Error(c, err)
	}
}

// ListDroneImages godoc
// @Summary      List drone orthophotos of an acquisition
// @Tags         LandAcquisitions
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Acquisition UUID"
// @Success      200 {object} response.Response
// @Router       /land-acquisitions/{id}/drone-images [get]
func (h *LandHandler) ListDroneImages(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	list, err := h.svc.ListDroneImages(c.Request.Context(), id)
	if err != nil {
		droneError(c, err)
		return
	}
	response.OK(c, h.toDroneImageResponses(list))
}

type droneUploadURLRequest struct {
	// FileName — хэрэглэгчийн сонгосон файлын нэр (өргөтгөл шалгахад хэрэглэнэ)
	FileName string `json:"file_name" binding:"required,max=255"`
}

type droneRegisterRequest struct {
	// StoredName — upload-url-ээс буцсан `stored_name`. Объектын БҮТЭН замыг
	// сервер өөрөө бүтээх тул зөвхөн нэр хүлээж авна.
	StoredName string `json:"stored_name" binding:"required,max=128"`
	// OriginalName — жагсаалтад харуулах нэр (зөвхөн танилцуулга)
	OriginalName string `json:"original_name" binding:"max=255"`
}

// CreateDroneUploadURL godoc
// @Summary      Get a direct-upload URL for a drone orthophoto
// @Description  Browser файлыг ШУУД файлын систем руу PUT хийнэ — байт API-аар
// @Description  дамжихгүй тул файлын хэмжээ хязгаарлагдахгүй. Байршуулсны дараа
// @Description  POST /drone-images-ээр бүртгүүлнэ.
// @Tags         LandAcquisitions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string                true "Acquisition UUID"
// @Param        body body droneUploadURLRequest true "File name"
// @Success      200 {object} response.Response
// @Router       /land-acquisitions/{id}/drone-images/upload-url [post]
func (h *LandHandler) CreateDroneUploadURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req droneUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}
	presigned, err := h.svc.CreateDroneUploadURL(c.Request.Context(), id, req.FileName)
	if err != nil {
		droneError(c, err)
		return
	}
	response.OK(c, presigned)
}

// RegisterDroneImage godoc
// @Summary      Register a directly-uploaded drone orthophoto
// @Description  upload-url-ээр байршуулсан объектыг шалгаж (байгаа эсэх, хэмжээ,
// @Description  GeoTIFF эсэх) чөлөөлөлтөд холбоно.
// @Tags         LandAcquisitions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string               true "Acquisition UUID"
// @Param        body body droneRegisterRequest true "Uploaded object"
// @Success      201 {object} response.Response
// @Router       /land-acquisitions/{id}/drone-images [post]
func (h *LandHandler) RegisterDroneImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req droneRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}

	var uploadedBy *domain.ID
	if raw, ok := c.Get(middleware.CtxUserID); ok {
		if uid, ok := raw.(domain.ID); ok && uid != 0 {
			uploadedBy = &uid
		}
	}

	img, publishErr, err := h.svc.RegisterDroneImage(c.Request.Context(), id, req.StoredName, req.OriginalName, uploadedBy)
	if err != nil {
		droneError(c, err)
		return
	}
	res := h.toDroneImageResponse(img)
	if publishErr != nil {
		res.PublishError = publishErr.Error()
	}
	response.Created(c, res)
}

// RefreshDroneImages godoc
// @Summary      Publish drone orthophotos as GeoServer layers
// @Description  Бүртгэлтэй зургуудыг GeoServer-т давхарга болгон нийтэлж,
// @Description  GeoServer-ийн кэшийг цэвэрлэнэ. Frontend зураг байршуулсны
// @Description  дараа автоматаар дуудна.
// @Tags         LandAcquisitions
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Acquisition UUID"
// @Success      200 {object} response.Response
// @Router       /land-acquisitions/{id}/drone-images/refresh [post]
func (h *LandHandler) RefreshDroneImages(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	list, err := h.svc.RefreshDroneImages(c.Request.Context(), id)
	if err != nil {
		droneError(c, err)
		return
	}
	response.OK(c, h.toDroneImageResponses(list))
}

// DeleteDroneImage godoc
// @Summary      Delete a drone orthophoto
// @Tags         LandAcquisitions
// @Security     BearerAuth
// @Produce      json
// @Param        id       path string true "Acquisition UUID"
// @Param        image_id path string true "Drone image UUID"
// @Success      204
// @Router       /land-acquisitions/{id}/drone-images/{image_id} [delete]
func (h *LandHandler) DeleteDroneImage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	imageID, err := uuid.Parse(c.Param("image_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.DeleteDroneImage(c.Request.Context(), id, imageID); err != nil {
		droneError(c, err)
		return
	}
	response.NoContent(c)
}
