package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/pkg/i18n"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
)

type PermissionHandler struct {
	permSvc  service.PermissionService
	validate *validator.Validate
}

func NewPermissionHandler(permSvc service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permSvc: permSvc, validate: validator.New()}
}

// List godoc
// @Summary      List permissions
// @Description  Returns all permissions. Requires permissions:read permission.
// @Tags         Permissions
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dataResponse{data=[]permissionResponse}
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Router       /permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	perms, err := h.permSvc.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, perms)
}

// Create godoc
// @Summary      Create permission
// @Description  Create a new permission. Requires permissions:create permission.
// @Tags         Permissions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body createPermissionBody true "Permission payload"
// @Success      201  {object}  dataResponse{data=permissionResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      409  {object}  errorResponse "Permission already exists"
// @Router       /permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
		Resource    string `json:"resource" validate:"required"`
		Action      string `json:"action" validate:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	perm, err := h.permSvc.Create(c.Request.Context(), service.CreatePermissionRequest{
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, perm)
}

// Delete godoc
// @Summary      Delete permission
// @Description  Delete a permission. Requires permissions:delete permission.
// @Tags         Permissions
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Permission UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return
	}
	if err := h.permSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
