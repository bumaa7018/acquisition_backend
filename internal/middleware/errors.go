package middleware

import (
	"net/http"

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
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					response.Response{Error: "internal server error"})
			}
		}()
		c.Next()
	}
}
