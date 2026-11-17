package http

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

// roleSessionInvalidator — ролийн эрх өөрчлөгдөхөд түүнийг эзэмшигч бүх
// хэрэглэгчийн токеныг хүчингүй болгоно (эрх нь JWT-д агуулагддаг).
type roleSessionInvalidator interface {
	InvalidateRoleSessions(ctx context.Context, roleID domain.ID) error
}

// SetRoleSessionInvalidator — router нь userSvc-ийг өгнө. Өгөөгүй (nil) үед
// (тест) сессийг хөндөхгүй.
func (h *RoleHandler) SetRoleSessionInvalidator(s roleSessionInvalidator) { h.sessions = s }
func (h *MenuHandler) SetRoleSessionInvalidator(s roleSessionInvalidator) { h.sessions = s }

// invalidateRoleHolders — эрх өөрчилсний ДАРАА дуудна. Алдаа гарвал хариуг
// бичээд false: эрх өөрчлөгдсөн ч хуучин токенууд хүчинтэй үлдсэнийг
// админ мэдэх ёстой.
func invalidateRoleHolders(c *gin.Context, s roleSessionInvalidator, roleID domain.ID) bool {
	if s == nil {
		return true
	}
	if err := s.InvalidateRoleSessions(c.Request.Context(), roleID); err != nil {
		response.Error(c, err)
		return false
	}
	return true
}

// callerCanRevokeMenuPermissions — цэсээс ролийн эрхийг ХАСАХ (цэс салгах,
// цэсийн эрхийг устгах/багасгах) үед хасагдах эрх БҮР дуудагчид байх ёстой —
// олгохтой ижил дүрэм (callerCanConfigureRolePermission). Өмнө нь хасах
// замууд дуудагчийн эрхийг шалгадаггүй тул roles:update-тэй хэрэглэгч
// өөрөөсөө өндөр эрхтэй ролийн (жишээ нь админы) эрхийг хурааж чаддаг байв.
//
// keep — хадгалагдах эрхийн ID-ууд (бүгдийг хасах үед nil).
func callerCanRevokeMenuPermissions(
	c *gin.Context, roleSvc service.RoleService, menuSvc service.MenuService,
	roleID, menuID domain.ID, keep map[domain.ID]bool,
) bool {
	if callerHasRole(c, "admin") {
		return true
	}
	if roleSvc == nil || menuSvc == nil {
		return false
	}
	role, err := roleSvc.GetByID(c.Request.Context(), roleID)
	if err != nil {
		return false
	}
	// Админы ролийн эрхийг зөвхөн админ өөрчилнө.
	if role.IsSystemAdmin() || domain.IsAdminRoleName(role.Name) {
		return false
	}
	available, err := menuSvc.ListPermissions(c.Request.Context(), menuID)
	if err != nil {
		return false
	}
	onMenu := make(map[string]domain.ID, len(available))
	for _, p := range available {
		onMenu[p.Name] = p.ID
	}
	for _, p := range role.Permissions {
		id, ok := onMenu[p.Name]
		if !ok || keep[id] {
			continue
		}
		if !callerHasPermission(c, p.Name) {
			return false
		}
	}
	return true
}
