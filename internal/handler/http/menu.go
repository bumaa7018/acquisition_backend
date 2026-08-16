package http

import (
	"net/http"

	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

// MenuHandler — цэсний харагдац болон роль ↔ цэсний холбоос.
type MenuHandler struct {
	svc service.MenuService
}

func NewMenuHandler(svc service.MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
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
