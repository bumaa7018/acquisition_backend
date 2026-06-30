package http

import (
	"net/http"
	"strconv"

	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DocumentTypeHandler struct {
	svc service.DocumentTypeService
}

func NewDocumentTypeHandler(svc service.DocumentTypeService) *DocumentTypeHandler {
	return &DocumentTypeHandler{svc: svc}
}

func currentUserEmail(c *gin.Context) *string {
	if uid, ok := c.Get("userID"); ok {
		s := uid.(uuid.UUID).String()
		return &s
	}
	return nil
}

func (h *DocumentTypeHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("target"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, list)
}

func (h *DocumentTypeHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	dt, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dt)
}

func (h *DocumentTypeHandler) Create(c *gin.Context) {
	var req service.CreateDocumentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.CreatedBy = currentUserEmail(c)
	dt, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, dt)
}

func (h *DocumentTypeHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req service.UpdateDocumentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.UpdatedBy = currentUserEmail(c)
	dt, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dt)
}

func (h *DocumentTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
