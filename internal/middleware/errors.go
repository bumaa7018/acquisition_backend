package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/response"
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
