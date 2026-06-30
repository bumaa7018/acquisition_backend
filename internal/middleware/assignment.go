package middleware

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AssignmentChecker — IsAssigned шалгах интерфейс (LandService-ийг тойруулан дамжуулахад ашиглана)
type AssignmentChecker interface {
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error)
}

// RequireAssignedOrSenior — ахлах мэргэжилтэн эсвэл тухайн чөлөөлөлтэд хуваарилагдсан ажилтан
// байвал зөвшөөрнө, эс бөгөөс 403 буцаана.
func RequireAssignedOrSenior(svc AssignmentChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ахлах мэргэжилтэн бол шалгуур шаардахгүй
		rolesAny, _ := c.Get(CtxRoles)
		if roles, ok := rolesAny.([]string); ok {
			for _, r := range roles {
				if r == "senior_specialist" || r == "Ахлах мэргэжилтэн" {
					c.Next()
					return
				}
			}
		}

		acqID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		userIDRaw, _ := c.Get(CtxUserID)
		userID, ok := userIDRaw.(uuid.UUID)
		if !ok {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		assigned, err := svc.IsAssigned(c.Request.Context(), acqID, userID)
		if err != nil || !assigned {
			response.Abort(c, 403, i18n.ErrNotAssigned)
			return
		}

		c.Next()
	}
}

// isSenior — контекстоос ахлах мэргэжилтэн эсэхийг шалгана
func IsSenior(c *gin.Context) bool {
	rolesAny, _ := c.Get(CtxRoles)
	roles, ok := rolesAny.([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == "senior_specialist" || r == "Ахлах мэргэжилтэн" {
			return true
		}
	}
	return false
}

// GetCallerID — контекстоос userID авна
func GetCallerID(c *gin.Context) (uuid.UUID, bool) {
	raw, _ := c.Get(CtxUserID)
	id, ok := raw.(uuid.UUID)
	return id, ok
}

// GetCallerName — контекстоос хэрэглэгчийн нэр авна
func GetCallerName(c *gin.Context) string {
	raw, _ := c.Get(CtxUserName)
	name, _ := raw.(string)
	return name
}

var _ = domain.ErrForbidden // ensure domain import used
