package middleware

import (
	"strings"

	"acquisition_backend/internal/pkg/actorctx"
	"acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	CtxUserID       = "userID"
	CtxOrgID        = "orgID"
	CtxUserName     = "userName"
	CtxUserPosition = "userPosition"
	CtxRoles        = "roles"
	CtxPermissions  = "permissions"
	// CtxValuationOrg — хэрэглэгч үнэлгээний байгууллагын ажилтан эсэх
	// (jwt.Claims.ValuationOrg). Эдгээр хэрэглэгчийн хандалт РОЛЬ/ЭРХЭЭР БИШ,
	// зөвхөн үүгээр шийдэгдэнэ.
	CtxValuationOrg = "isValuationOrg"
	// CtxUserSource — хэрэглэгчийн бүртгэл аль санд байгаа (domain.UserSource*).
	//
	// CtxUserID нь хоёр санд МӨРГӨЛДӨХ int32 тул хэрэглэгчид ХАРЬЯАЛАГДАХ
	// өгөгдлийг (мэдэгдэл г.м.) хайхад ЗӨВХӨН энэ хоёрыг ХОСООР нь хэрэглэнэ.
	CtxUserSource = "userSource"
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

		claims, err := jwtManager.ValidateAccessTokenContext(c.Request.Context(), parts[1])
		if err != nil {
			response.Error(c, err)
			return
		}

		c.Set(CtxUserID, claims.UserID)
		if claims.OrgID != nil {
			c.Set(CtxOrgID, *claims.OrgID)
		}
		userName := claims.FullName
		if userName == "" {
			userName = strings.TrimSpace(claims.FirstName + " " + claims.LastName)
		}
		c.Set(CtxUserName, userName)
		c.Set(CtxUserPosition, claims.Position)
		c.Set(CtxRoles, claims.Roles)
		c.Set(CtxPermissions, claims.Permissions)
		c.Set(CtxValuationOrg, claims.ValuationOrg)
		c.Set(CtxUserSource, claims.UserSource())

		// Actor-ийг request context-д хийж service давхаргад дамжуулна
		// (мэдэгдэл илгээхэд хэн үйлдэл хийснийг тэндээс уншина).
		c.Request = c.Request.WithContext(actorctx.With(c.Request.Context(), actorctx.Actor{
			ID:        claims.UserID,
			Source:    claims.UserSource(),
			Name:      userName,
			Position:  claims.Position,
			Roles:     claims.Roles,
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		}))
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hasAnyRole(c, "admin", "Админ") {
			c.Next()
			return
		}
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
		if hasAnyRole(c, "admin", "Админ") {
			c.Next()
			return
		}
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
		if hasAnyRole(c, "admin", "Админ") || hasAnyRole(c, allowedRoles...) {
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

// ЖИЧ: RequireOnlyRole (ЗӨВХӨН тухайн роль-той хэрэглэгчийг нэвтрүүлэх) нь
// зөвхөн /prof группэд хэрэглэгддэг байсан бөгөөс хасагдав — үнэлгээний
// байгууллага роль эзэмшихээ больж, RequireValuationOrgCaller-ээр шалгагдана
// (internal/middleware/valuation_org.go).

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
