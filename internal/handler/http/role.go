package http

import (
	"net/http"

	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type RoleHandler struct {
	roleSvc  service.RoleService
	validate *validator.Validate
}

func NewRoleHandler(roleSvc service.RoleService) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc, validate: validator.New()}
}

// List godoc
// @Summary      List roles
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.roleSvc.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, roles)
}

// GetByID godoc
// @Summary      Get role by ID
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Role UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	role, err := h.roleSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, role)
}

// Create godoc
// @Summary      Create role
// @Tags         Roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body createRoleBody true "Role payload"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	role, err := h.roleSvc.Create(c.Request.Context(), service.CreateRoleRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, role)
}

// Update godoc
// @Summary      Update role
// @Tags         Roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Role UUID"
// @Param        body body      updateRoleBody  true  "Fields to update"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	role, err := h.roleSvc.Update(c.Request.Context(), id, service.UpdateRoleRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, role)
}

// Delete godoc
// @Summary      Delete role
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Role UUID"
// @Success      204
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.roleSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// AssignPermission godoc
// @Summary      Assign permission to role
// @Tags         Roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Role UUID"
// @Param        body body      assignPermissionBody true  "Permission to assign"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	var req struct {
		PermissionID string `json:"permission_id" validate:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	permID, err := uuid.Parse(req.PermissionID)
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.roleSvc.AssignPermission(c.Request.Context(), roleID, permID); err != nil {
		response.Error(c, err)
		return
	}
	l := i18n.Detect(c.GetHeader("Accept-Language"))
	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: i18n.T(l, i18n.MsgAssigned),
	})
}

// RemovePermission godoc
// @Summary      Remove permission from role
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Param        id            path  string  true  "Role UUID"
// @Param        permission_id path  string  true  "Permission UUID"
// @Success      204
// @Failure      400  {object}  response.Response
// @Router       /roles/{id}/permissions/{permission_id} [delete]
func (h *RoleHandler) RemovePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	permID, err := uuid.Parse(c.Param("permission_id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.roleSvc.RemovePermission(c.Request.Context(), roleID, permID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
