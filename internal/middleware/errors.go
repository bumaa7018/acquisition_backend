package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

var (
	ErrMissingToken = domain.ErrUnauthorized
	ErrForbidden    = domain.ErrForbidden
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					"error", r,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					response.Response{Error: "internal server error"})
			}
		}()
		c.Next()
	}
}
