package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
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
// @Description  Returns all roles with their permissions. Requires roles:read permission.
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dataResponse{data=[]roleResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
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
// @Description  Returns a single role with its permissions. Requires roles:read permission.
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Role UUID"
// @Success      200  {object}  dataResponse{data=roleResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid id"})
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
// @Description  Create a new role. Requires roles:create permission.
// @Tags         Roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body createRoleBody true "Role payload"
// @Success      201  {object}  dataResponse{data=roleResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      409  {object}  errorResponse "Role name already exists"
// @Router       /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
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
// @Description  Update role name or description. Requires roles:update permission.
// @Tags         Roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string          true  "Role UUID"
// @Param        body body      updateRoleBody  true  "Fields to update"
// @Success      200  {object}  dataResponse{data=roleResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid id"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
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
// @Description  Delete a role. Requires roles:delete permission.
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Role UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid id"})
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
// @Description  Add a permission to a role. Requires roles:update permission.
// @Tags         Roles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string               true  "Role UUID"
// @Param        body body      assignPermissionBody true  "Permission to assign"
// @Success      200  {object}  dataResponse{data=messageResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Router       /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid role id"})
		return
	}
	var req struct {
		PermissionID string `json:"permission_id" validate:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	permID, err := uuid.Parse(req.PermissionID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid permission_id"})
		return
	}
	if err := h.roleSvc.AssignPermission(c.Request.Context(), roleID, permID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "permission assigned"})
}

// RemovePermission godoc
// @Summary      Remove permission from role
// @Description  Remove a permission from a role. Requires roles:update permission.
// @Tags         Roles
// @Security     BearerAuth
// @Produce      json
// @Param        id            path  string  true  "Role UUID"
// @Param        permission_id path  string  true  "Permission UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Router       /roles/{id}/permissions/{permission_id} [delete]
func (h *RoleHandler) RemovePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid role id"})
		return
	}
	permID, err := uuid.Parse(c.Param("permission_id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid permission_id"})
		return
	}
	if err := h.roleSvc.RemovePermission(c.Request.Context(), roleID, permID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
