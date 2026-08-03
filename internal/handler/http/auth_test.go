package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	tokens := &jwtpkg.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
	mockSvc.On("Login", mock.Anything, service.LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	}).Return(tokens, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"username":"testuser","password":"Password123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "access", data["access_token"])
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	mockSvc.On("Login", mock.Anything, service.LoginRequest{
		Username: "testuser",
		Password: "wrong",
	}).Return(nil, domain.ErrInvalidCredentials)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"username":"testuser","password":"wrong"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	tokens := &jwtpkg.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}
	mockSvc.On("RefreshToken", mock.Anything, "valid-refresh-token").Return(tokens, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"refresh_token":"valid-refresh-token"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RefreshToken(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "new-access", data["access_token"])
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_InvalidToken(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	mockSvc.On("RefreshToken", mock.Anything, "bad-token").Return(nil, domain.ErrInvalidToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"refresh_token":"bad-token"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Logout(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)
	mockSvc.On("Logout", mock.Anything, "", "").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

	h.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	mockSvc.On("ResetPassword", mock.Anything, "valid-token", "NewPassword123!").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"token":"valid-token","new_password":"NewPassword123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ResetPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ResetPassword_ExpiredToken(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	mockSvc.On("ResetPassword", mock.Anything, "expired-token", "NewPassword123!").Return(domain.ErrExpiredToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"token":"expired-token","new_password":"NewPassword123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ResetPassword(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_ForgotPassword_AlwaysOK(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	mockSvc.On("ForgotPassword", mock.Anything, "unknown@example.com").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"email":"unknown@example.com"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ForgotPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
