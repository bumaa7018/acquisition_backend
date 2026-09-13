package http

import (
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func contextUserIDString(c *gin.Context) string {
	raw, ok := c.Get(middleware.CtxUserID)
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case domain.ID:
		return fmt.Sprint(v)
	case int:
		return fmt.Sprint(v)
	case int64:
		return fmt.Sprint(v)
	case string:
		return v
	case uuid.UUID:
		if v == uuid.Nil {
			return ""
		}
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func contextUserDisplayName(c *gin.Context) string {
	if name := strings.TrimSpace(middleware.GetCallerName(c)); name != "" {
		return name
	}
	if id := contextUserIDString(c); id != "" {
		return "Нэр бүртгэгдээгүй"
	}
	return "system"
}
