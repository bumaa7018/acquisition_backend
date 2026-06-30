package http

import (
	"context"
	"net/http"

	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db HealthChecker
}

func NewHealthHandler(db HealthChecker) *HealthHandler {
	return &HealthHandler{db: db}
}

// Live godoc
// @Summary      Liveness probe
// @Description  Returns 200 if the server is running
// @Tags         Health
// @Produce      json
// @Success      200  {object}  dataResponse{data=messageResponse}
// @Router       /health [get]
func (h *HealthHandler) Live(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}

// Ready godoc
// @Summary      Readiness probe
// @Description  Returns 200 if the server can reach the database, 503 otherwise
// @Tags         Health
// @Produce      json
// @Success      200  {object}  dataResponse{data=messageResponse}
// @Failure      503  {object}  errorResponse "Database unavailable"
// @Router       /health/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.db.Ping(c.Request.Context()); err != nil {
		response.Abort(c, http.StatusServiceUnavailable, i18n.ErrDBUnavailable)
		return
	}
	response.OK(c, gin.H{"status": "ready"})
}
