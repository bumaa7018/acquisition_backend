package http

import (
	"net/http"
	"strconv"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AcquisitionWorkflowHandler struct {
	svc service.LandService
}

func NewAcquisitionWorkflowHandler(svc service.LandService) *AcquisitionWorkflowHandler {
	return &AcquisitionWorkflowHandler{svc: svc}
}

func (h *AcquisitionWorkflowHandler) List(c *gin.Context) {
	list, err := h.svc.ListAcquisitionWorkflows(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	if list == nil {
		list = []*domain.AcquisitionWorkflow{}
	}
	response.OK(c, list)
}

func (h *AcquisitionWorkflowHandler) Create(c *gin.Context) {
	var req struct {
		FromStatusID *int `json:"from_status_id"`
		ToStatusID   int  `json:"to_status_id"`
		SortOrder    int  `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ToStatusID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_status_id required"})
		return
	}
	if _, ok := domain.AcquisitionStatusLabels[req.ToStatusID]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_status_id"})
		return
	}
	if req.FromStatusID != nil {
		if _, ok := domain.AcquisitionStatusLabels[*req.FromStatusID]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_status_id"})
			return
		}
	}
	w, err := h.svc.CreateAcquisitionWorkflow(c.Request.Context(), req.FromStatusID, req.ToStatusID, req.SortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 201, "data": w})
}

func (h *AcquisitionWorkflowHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteAcquisitionWorkflow(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// ListAcquisitionStatuses returns the hard-coded acquisition status definitions.
// Used by the frontend to populate from/to dropdowns.
func (h *AcquisitionWorkflowHandler) ListStatuses(c *gin.Context) {
	type statusItem struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var items []statusItem
	for _, id := range domain.AcquisitionStatusOrder {
		items = append(items, statusItem{ID: id, Name: domain.AcquisitionStatusLabels[id]})
	}
	response.OK(c, items)
}
