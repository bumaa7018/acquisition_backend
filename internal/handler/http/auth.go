package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/your-org/go-api-template/internal/pkg/i18n"
	"github.com/your-org/go-api-template/internal/pkg/response"
	"github.com/your-org/go-api-template/internal/service"
)

type AuthHandler struct {
	authSvc  service.AuthService
	validate *validator.Validate
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, validate: validator.New()}
}

type registerRequest struct {
	Username  string `json:"username" validate:"required"`
	Email     string `json:"email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// Register godoc
// @Summary      Register a new user
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body registerBody true "Registration payload"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	user, err := h.authSvc.Register(c.Request.Context(), service.RegisterRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, user)
}

// Login godoc
// @Summary      Login
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body loginBody true "Login credentials"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	tokens, err := h.authSvc.Login(c.Request.Context(), service.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tokens)
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body refreshBody true "Refresh token"
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	tokens, err := h.authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tokens)
}

// Logout godoc
// @Summary      Logout
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: i18n.T(i18n.Detect(c.GetHeader("Accept-Language")), i18n.MsgLogout),
	})
}

// ForgotPassword godoc
// @Summary      Forgot password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body forgotPasswordBody true "Email address"
// @Success      200  {object}  response.Response
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	_ = h.authSvc.ForgotPassword(c.Request.Context(), req.Email)
	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: i18n.T(i18n.Detect(c.GetHeader("Accept-Language")), i18n.MsgPassResetSent),
	})
}

// ResetPassword godoc
// @Summary      Reset password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body resetPasswordBody true "Token and new password"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.authSvc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: i18n.T(i18n.Detect(c.GetHeader("Accept-Language")), i18n.MsgPassReset),
	})
}
