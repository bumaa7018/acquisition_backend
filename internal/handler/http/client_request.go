package http

import (
	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ClientRequestHandler struct {
	svc service.ClientRequestService
}

func NewClientRequestHandler(svc service.ClientRequestService) *ClientRequestHandler {
	return &ClientRequestHandler{svc: svc}
}

func (h *ClientRequestHandler) List(c *gin.Context) {
	filter := domain.ClientRequestFilter{
		Method:   c.Query("method"),
		Host:     c.Query("host"),
		Path:     c.Query("path"),
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
	}
	if t, ok := parseAuditTime(c.Query("created_from")); ok {
		filter.CreatedFrom = &t
	}
	if t, ok := parseAuditTime(c.Query("created_to")); ok {
		filter.CreatedTo = &t
	}
	list, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, filter.Page, filter.PageSize)
}
