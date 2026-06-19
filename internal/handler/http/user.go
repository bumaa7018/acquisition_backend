package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/middleware"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
)

type UserHandler struct {
	userSvc  service.UserService
	validate *validator.Validate
}

func NewUserHandler(userSvc service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc, validate: validator.New()}
}

// GetMe godoc
// @Summary      Get own profile
// @Description  Returns the authenticated user's profile
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dataResponse{data=userResponse}
// @Failure      401  {object}  errorResponse
// @Router       /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get(middleware.CtxUserID)
	id, _ := userID.(uuid.UUID)

	user, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// List godoc
// @Summary      List users
// @Description  Returns a paginated list of users. Requires users:read permission.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        page      query  int     false  "Page number"       default(1)
// @Param        page_size query  int     false  "Items per page"    default(20)
// @Param        search    query  string  false  "Search by email/name"
// @Param        is_active query  bool    false  "Filter by active status"
// @Success      200  {object}  paginatedUsersResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Router       /users [get]
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	filter := domain.UserFilter{
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}
	if activeStr := c.Query("is_active"); activeStr != "" {
		active := activeStr == "true"
		filter.IsActive = &active
	}

	users, total, err := h.userSvc.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, users, total, page, pageSize)
}

// GetByID godoc
// @Summary      Get user by ID
// @Description  Returns a single user. Requires users:read permission.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse "Invalid UUID"
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid id"})
		return
	}
	user, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// Create godoc
// @Summary      Create user
// @Description  Create a new user. Requires users:create permission.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body createUserBody true "User payload"
// @Success      201  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      409  {object}  errorResponse "Email already exists"
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Email     string `json:"email" validate:"required,email"`
		Password  string `json:"password" validate:"required,min=8"`
		FirstName string `json:"first_name" validate:"required"`
		LastName  string `json:"last_name" validate:"required"`
		IsActive  bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	user, err := h.userSvc.Create(c.Request.Context(), service.CreateUserRequest{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsActive:  req.IsActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, user)
}

// Update godoc
// @Summary      Update user
// @Description  Update user fields. Requires users:update permission.
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path      string          true  "User UUID"
// @Param        body body      updateUserBody  true  "Fields to update"
// @Success      200  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid id"})
		return
	}
	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	user, err := h.userSvc.Update(c.Request.Context(), id, service.UpdateUserRequest{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsActive:  req.IsActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// Delete godoc
// @Summary      Delete user (soft)
// @Description  Soft-deletes a user by setting deleted_at. Requires users:delete permission.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse
// @Failure      403  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: "invalid id"})
		return
	}
	if err := h.userSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
