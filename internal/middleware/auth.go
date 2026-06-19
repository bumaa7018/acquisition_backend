package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/your-org/go-api-template/internal/pkg/jwt"
	"github.com/your-org/go-api-template/internal/pkg/response"
)

const (
	CtxUserID      = "userID"
	CtxPermissions = "permissions"
)

func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, ErrMissingToken)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Error(c, ErrMissingToken)
			return
		}

		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			response.Error(c, err)
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxPermissions, claims.Permissions)
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permsAny, exists := c.Get(CtxPermissions)
		if !exists {
			response.Error(c, ErrForbidden)
			return
		}
		perms, ok := permsAny.([]string)
		if !ok {
			response.Error(c, ErrForbidden)
			return
		}
		for _, p := range perms {
			if p == permission {
				c.Next()
				return
			}
		}
		response.Error(c, ErrForbidden)
	}
}
