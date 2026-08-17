package http

import (
	"net/http"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

// MenuHandler — цэсний харагдац болон роль ↔ цэсний холбоос.
type MenuHandler struct {
	svc     service.MenuService
	roleSvc service.RoleService
}

func NewMenuHandler(svc service.MenuService, optionalRoleSvc ...service.RoleService) *MenuHandler {
	var roleSvc service.RoleService
	if len(optionalRoleSvc) > 0 {
		roleSvc = optionalRoleSvc[0]
	}
	return &MenuHandler{svc: svc, roleSvc: roleSvc}
}

// List godoc
// @Summary Системийн бүх цэс
// @Tags menus
// @Success 200 {object} response.SuccessResponse
// @Router /menus [get]
func (h *MenuHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, items)
}

// Mine godoc
// @Summary Нэвтэрсэн хэрэглэгчид харагдах цэс
// @Tags menus
// @Success 200 {object} response.SuccessResponse
// @Router /menus/me [get]
func (h *MenuHandler) Mine(c *gin.Context) {
	items, err := h.svc.ListForUser(c.Request.Context(), callerID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, items)
}

// ListByRole godoc
// @Summary Тухайн роль дээр холбогдсон цэсүүд
// @Tags menus
// @Param id path int true "Role ID"
// @Success 200 {object} response.SuccessResponse
// @Router /roles/{id}/menus [get]
func (h *MenuHandler) ListByRole(c *gin.Context) {
	roleID, ok := parseIDParam(c)
	if !ok {
		return
	}
	items, err := h.svc.ListByRole(c.Request.Context(), roleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, items)
}

type attachMenuBody struct {
	MenuID int32 `json:"menu_id"`
}

// Attach godoc
// @Summary Роль дээр цэс холбох
// @Tags menus
// @Param id path int true "Role ID"
// @Success 201 {object} response.SuccessResponse
// @Router /roles/{id}/menus [post]
func (h *MenuHandler) Attach(c *gin.Context) {
	roleID, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b attachMenuBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.svc.Attach(c.Request.Context(), roleID, b.MenuID); err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"role_id": roleID, "menu_id": b.MenuID})
}

type setMenuPermissionsBody struct {
	PermissionIDs []string `json:"permission_ids"`
}

// SetRoleMenuPermissions godoc
// @Summary Роль + цэс дээр permission-үүд хадгалах
// @Tags menus
// @Param id path int true "Role ID"
// @Param menu_id path int true "Menu ID"
// @Success 200 {object} response.SuccessResponse
// @Router /roles/{id}/menus/{menu_id}/permissions [put]
func (h *MenuHandler) SetRoleMenuPermissions(c *gin.Context) {
	if h.roleSvc == nil {
		response.Error(c, domain.ErrInvalidInput)
		return
	}
	roleID, ok := parseIDParam(c)
	if !ok {
		return
	}
	menuID, ok := parseID(c.Param("menu_id"))
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var b setMenuPermissionsBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	available, err := h.svc.ListPermissions(c.Request.Context(), menuID)
	if err != nil {
		response.Error(c, err)
		return
	}
	availableByID := make(map[domain.ID]domain.Permission, len(available))
	for _, p := range available {
		availableByID[p.ID] = p
	}
	permissionIDs := make([]domain.ID, 0, len(b.PermissionIDs))
	for _, raw := range b.PermissionIDs {
		permissionID, ok := parseID(raw)
		if !ok {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return
		}
		perm, ok := availableByID[permissionID]
		if !ok {
			response.Error(c, domain.ErrInvalidInput)
			return
		}
		if !callerHasRole(c, "admin") && !callerHasPermission(c, perm.Name) {
			response.Error(c, domain.ErrForbidden)
			return
		}
		permissionIDs = append(permissionIDs, permissionID)
	}
	if err := h.roleSvc.SetMenuPermissions(c.Request.Context(), roleID, menuID, permissionIDs); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"role_id": roleID, "menu_id": menuID, "permission_ids": b.PermissionIDs})
}

// RemoveRoleMenuPermissions godoc
// @Summary Ролиос цэс болон тухайн цэсний permission-үүдийг салгах
// @Tags menus
// @Param id path int true "Role ID"
// @Param menu_id path int true "Menu ID"
// @Success 204 "No Content"
// @Router /roles/{id}/menus/{menu_id}/permissions [delete]
func (h *MenuHandler) RemoveRoleMenuPermissions(c *gin.Context) {
	if h.roleSvc == nil {
		response.Error(c, domain.ErrInvalidInput)
		return
	}
	roleID, ok := parseIDParam(c)
	if !ok {
		return
	}
	menuID, ok := parseID(c.Param("menu_id"))
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.roleSvc.RemoveMenuPermissions(c.Request.Context(), roleID, menuID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// Detach godoc
// @Summary Ролиос цэсийг салгах
// @Tags menus
// @Param id path int true "Role ID"
// @Param menu_id path int true "Menu ID"
// @Success 204 "No Content"
// @Router /roles/{id}/menus/{menu_id} [delete]
func (h *MenuHandler) Detach(c *gin.Context) {
	roleID, ok := parseIDParam(c)
	if !ok {
		return
	}
	menuID, ok := parseID(c.Param("menu_id"))
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.svc.Detach(c.Request.Context(), roleID, menuID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
