package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"acquisition_backend/internal/pkg/i18n"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authSvc  service.AuthService
	validate *validator.Validate
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, validate: validator.New()}
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// Login godoc
// @Summary      Login (дотоод хэрэглэгч)
// @Description  ЗӨВХӨН authdb (sdplatform.sd_user)-ээс шалгана. Мэргэжлийн
// @Description  байгууллага /auth/prof/login-ээр нэвтэрнэ.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body loginBody true "Login credentials"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	h.login(c, h.authSvc.Login)
}

// LoginProfessionalOrg godoc
// @Summary      Login (мэргэжлийн байгууллага)
// @Description  ЗӨВХӨН appdb-ийн valuation_org_employee-ээс шалгана. Дотоод
// @Description  хэрэглэгч энд нэвтрэхгүй.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body loginBody true "Login credentials"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/prof/login [post]
func (h *AuthHandler) LoginProfessionalOrg(c *gin.Context) {
	h.login(c, h.authSvc.LoginProfessionalOrg)
}

// login — хоёр нэвтрэх урсгалын нийтлэг хэсэг. Ялгаа нь ЗӨВХӨН аль сангаас
// хэрэглэгч хайхад л байгаа тул үлдсэнийг давхардуулахгүй.
func (h *AuthHandler) login(c *gin.Context, do func(context.Context, service.LoginRequest) (*jwtpkg.TokenPair, error)) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AbortValidation(c, err)
		return
	}
	tokens, err := do(c.Request.Context(), service.LoginRequest{
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
	var req logoutRequest
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		_ = json.NewDecoder(c.Request.Body).Decode(&req)
	}
	_ = h.authSvc.Logout(c.Request.Context(), bearerToken(c.GetHeader("Authorization")), req.RefreshToken)
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

func bearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
