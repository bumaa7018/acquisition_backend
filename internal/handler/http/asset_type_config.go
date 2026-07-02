package http

import (
	"net/http"
	"strconv"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AssetTypeConfigHandler struct {
	svc service.LandService
}

func NewAssetTypeConfigHandler(svc service.LandService) *AssetTypeConfigHandler {
	return &AssetTypeConfigHandler{svc: svc}
}

// --- AssetSpecType ---

func (h *AssetTypeConfigHandler) ListSpecTypes(c *gin.Context) {
	list, err := h.svc.ListAssetSpecTypes(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

func (h *AssetTypeConfigHandler) CreateSpecType(c *gin.Context) {
	var body struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t := &domain.AssetSpecType{Code: body.Code, Name: body.Name, SortOrder: body.SortOrder}
	if err := h.svc.CreateAssetSpecType(c.Request.Context(), t); err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *AssetTypeConfigHandler) UpdateSpecType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t := &domain.AssetSpecType{ID: id, Code: body.Code, Name: body.Name, SortOrder: body.SortOrder}
	if err := h.svc.UpdateAssetSpecType(c.Request.Context(), t); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, t)
}

func (h *AssetTypeConfigHandler) DeleteSpecType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteAssetSpecType(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// --- AssetCalcType ---

func (h *AssetTypeConfigHandler) ListCalcTypes(c *gin.Context) {
	list, err := h.svc.ListAssetCalcTypes(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

func (h *AssetTypeConfigHandler) CreateCalcType(c *gin.Context) {
	var body struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		DefaultUnit string `json:"default_unit"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t := &domain.AssetCalcType{Code: body.Code, Name: body.Name, DefaultUnit: body.DefaultUnit, SortOrder: body.SortOrder}
	if err := h.svc.CreateAssetCalcType(c.Request.Context(), t); err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *AssetTypeConfigHandler) UpdateCalcType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		DefaultUnit string `json:"default_unit"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t := &domain.AssetCalcType{ID: id, Code: body.Code, Name: body.Name, DefaultUnit: body.DefaultUnit, SortOrder: body.SortOrder}
	if err := h.svc.UpdateAssetCalcType(c.Request.Context(), t); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, t)
}

func (h *AssetTypeConfigHandler) DeleteCalcType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteAssetCalcType(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
