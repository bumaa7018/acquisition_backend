package http

import (
	"net/http"
	"strconv"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AcquisitionCategoryHandler struct {
	svc service.LandService
}

func NewAcquisitionCategoryHandler(svc service.LandService) *AcquisitionCategoryHandler {
	return &AcquisitionCategoryHandler{svc: svc}
}

func (h *AcquisitionCategoryHandler) List(c *gin.Context) {
	var parentID *int
	if s := c.Query("parent_id"); s != "" {
		id, err := strconv.Atoi(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		parentID = &id
	}
	list, err := h.svc.ListAcquisitionCategories(c.Request.Context(), parentID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []*domain.AcquisitionCategory{}
	}
	response.OK(c, list)
}

func (h *AcquisitionCategoryHandler) Create(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		ParentID  *int   `json:"parent_id"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	cat, err := h.svc.CreateAcquisitionCategory(c.Request.Context(), req.Name, req.ParentID, req.SortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 201, "data": cat})
}

func (h *AcquisitionCategoryHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	cat, err := h.svc.UpdateAcquisitionCategory(c.Request.Context(), id, req.Name, req.SortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, cat)
}

func (h *AcquisitionCategoryHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteAcquisitionCategory(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
