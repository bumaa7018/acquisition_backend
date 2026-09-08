package http

// Системийн тохиргооны маршрутууд. Уншилт нэвтэрсэн бүх хэрэглэгчид (газрын
// зураг тэднийг ижил суурь зурагтай харуулах ёстой), БИЧИХ нь admin:update
// эрхтэйд (router-т хаалгална).

import (
	"net/http"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	svc service.SettingService
}

func NewSettingHandler(svc service.SettingService) *SettingHandler {
	return &SettingHandler{svc: svc}
}

type basemapRequest struct {
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Enabled     bool      `json:"enabled"`
	Layer       string    `json:"layer"`
	MaxZoom     int       `json:"max_zoom"`
	Extent      []float64 `json:"extent"`
	Attribution string    `json:"attribution"`
	Name        string    `json:"name"`
}

// GetBasemap godoc
//
//	@Summary		Газрын зургийн суурь зургийн тохиргоо
//	@Description	Гараас тохируулсан суурь зураг (tile/wms/image). Тохируулаагүй бол data=null — дэлгэц үндсэн суурь зургаа хэрэглэнэ.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/settings/basemap [get]
func (h *SettingHandler) GetBasemap(c *gin.Context) {
	basemap, err := h.svc.GetBasemap(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	// Тохируулаагүй нь алдаа биш.
	if basemap == nil {
		response.OK(c, nil)
		return
	}
	response.OK(c, basemap)
}

// SaveBasemap godoc
//
//	@Summary		Суурь зургийн хаягийг тохируулах
//	@Description	type: "xyz" (tile — {x}/{y}/{z} орлуулагчтай), "wms" (растер зураг, layer нэртэй), "image" (тогтмол зураг, extent-тэй). url хоосон бол үндсэн суурь зураг хэрэглэгдэнэ.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		basemapRequest	true	"Суурь зургийн тохиргоо"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Security		BearerAuth
//	@Router			/settings/basemap [put]
func (h *SettingHandler) SaveBasemap(c *gin.Context) {
	var req basemapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidInput)
		return
	}
	basemap, err := h.svc.SaveBasemap(c.Request.Context(), domain.Basemap{
		Type:        req.Type,
		URL:         req.URL,
		Enabled:     req.Enabled,
		Layer:       req.Layer,
		MaxZoom:     req.MaxZoom,
		Extent:      req.Extent,
		Attribution: req.Attribution,
		Name:        req.Name,
	}, contextUserIDString(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, basemap, i18n.MsgUpdated)
}

// ResetBasemap godoc
//
//	@Summary		Суурь зургийн тохиргоог үндсэн зураг рүү буцаах
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	response.Response
//	@Security		BearerAuth
//	@Router			/settings/basemap [delete]
func (h *SettingHandler) ResetBasemap(c *gin.Context) {
	if err := h.svc.ResetBasemap(c.Request.Context()); err != nil {
		response.Error(c, err)
		return
	}
	response.OKMsg(c, nil, i18n.MsgDeleted)
}
