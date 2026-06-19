package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
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
// @Description  Create a new user account
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body registerBody true "Registration payload"
// @Success      201  {object}  dataResponse{data=userResponse}
// @Failure      400  {object}  errorResponse
// @Failure      409  {object}  errorResponse "Email already exists"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	user, err := h.authSvc.Register(c.Request.Context(), service.RegisterRequest{
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
// @Description  Authenticate with email and password, receive JWT token pair
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body loginBody true "Login credentials"
// @Success      200  {object}  dataResponse{data=tokenPairResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse "Invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	tokens, err := h.authSvc.Login(c.Request.Context(), service.LoginRequest{
		Email:    req.Email,
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
// @Description  Exchange a valid refresh token for a new token pair
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body refreshBody true "Refresh token"
// @Success      200  {object}  dataResponse{data=tokenPairResponse}
// @Failure      401  {object}  errorResponse "Invalid or expired refresh token"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
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
// @Description  Invalidate the current session (client must discard the token)
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dataResponse{data=messageResponse}
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	response.OK(c, gin.H{"message": "logged out successfully"})
}

// ForgotPassword godoc
// @Summary      Forgot password
// @Description  Request a password reset token (always returns 200 to prevent email enumeration)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body forgotPasswordBody true "Email address"
// @Success      200  {object}  dataResponse{data=messageResponse}
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	// Алдааг илчлэхгүй — email байхгүй ч 200 буцаана (enumeration хамгаалалт)
	_ = h.authSvc.ForgotPassword(c.Request.Context(), req.Email)
	response.OK(c, gin.H{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Reset user password using a valid reset token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body resetPasswordBody true "Token and new password"
// @Success      200  {object}  dataResponse{data=messageResponse}
// @Failure      400  {object}  errorResponse
// @Failure      401  {object}  errorResponse "Invalid or expired token"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{Error: err.Error()})
		return
	}
	if err := h.authSvc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "password reset successfully"})
}
