package http

import (
	"net/http"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditHandler struct {
	svc service.AuditService
}

func NewAuditHandler(svc service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) List(c *gin.Context) {
	filter := domain.AuditLogFilter{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ResourceID:   c.Query("resource_id"),
		Page:         queryInt(c, "page", 1),
		PageSize:     queryInt(c, "page_size", 20),
	}
	if v := c.Query("actor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return
		}
		filter.ActorID = &id
	}
	if v := c.Query("acquisition_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return
		}
		filter.AcquisitionID = &id
	}
	if v := c.Query("parcel_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return
		}
		filter.ParcelID = &id
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

func parseAuditTime(v string) (time.Time, bool) {
	if v == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t, true
	}
	return time.Time{}, false
}
