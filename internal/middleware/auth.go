package middleware

import (
	"strings"

	"acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	CtxUserID       = "userID"
	CtxUserName     = "userName"
	CtxUserPosition = "userPosition"
	CtxRoles        = "roles"
	CtxPermissions  = "permissions"
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
		userName := claims.FullName
		if userName == "" {
			userName = strings.TrimSpace(claims.FirstName + " " + claims.LastName)
		}
		c.Set(CtxUserName, userName)
		c.Set(CtxUserPosition, claims.Position)
		c.Set(CtxRoles, claims.Roles)
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

// RequireAnyPermission — өгсөн permission-уудын аль нэг нь байвал нэвтэрнэ.
// Дотоод ажилтнууд land:*, гадаад ролиуд (мэрг.байгууллага/МИКА/санхүү)
// compensation:* эрхээр ижил маршрутыг ашиглахад хэрэглэнэ.
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range permissions {
			if hasPermission(c, p) {
				c.Next()
				return
			}
		}
		response.Error(c, ErrForbidden)
	}
}

func RequirePermissionOrRole(permission string, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasPermission(c, permission) || hasAnyRole(c, allowedRoles...) {
			c.Next()
			return
		}
		response.Error(c, ErrForbidden)
	}
}

func RequireAnyRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasAnyRole(c, allowedRoles...) {
			c.Next()
			return
		}
		response.Error(c, ErrForbidden)
	}
}

func DenyAnyRole(blockedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasAnyRole(c, blockedRoles...) {
			response.Error(c, ErrForbidden)
			return
		}
		c.Next()
	}
}

func RequireOnlyRole(allowedRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesAny, exists := c.Get(CtxRoles)
		if !exists {
			response.Error(c, ErrForbidden)
			return
		}
		roles, ok := rolesAny.([]string)
		if !ok || len(roles) == 0 {
			response.Error(c, ErrForbidden)
			return
		}
		for _, r := range roles {
			if r != allowedRole {
				response.Error(c, ErrForbidden)
				return
			}
		}
		c.Next()
	}
}

func hasPermission(c *gin.Context, permission string) bool {
	permsAny, exists := c.Get(CtxPermissions)
	if !exists {
		return false
	}
	perms, ok := permsAny.([]string)
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

func hasAnyRole(c *gin.Context, allowedRoles ...string) bool {
	rolesAny, exists := c.Get(CtxRoles)
	if !exists {
		return false
	}
	roles, ok := rolesAny.([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		for _, allowed := range allowedRoles {
			if r == allowed {
				return true
			}
		}
	}
	return false
}
