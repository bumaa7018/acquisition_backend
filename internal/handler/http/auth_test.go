package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/handler/http/mocks"
	jwtpkg "github.com/your-org/go-api-template/internal/pkg/jwt"
	"github.com/your-org/go-api-template/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	user := &domain.User{ID: uuid.New(), Email: "test@example.com"}
	mockSvc.On("Register", mock.Anything, service.RegisterRequest{
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"email":"test@example.com","password":"Password123!","first_name":"Test","last_name":"User"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"email":"not-an-email","password":"short"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockSvc := &mocks.MockAuthService{}
	h := handler.NewAuthHandler(mockSvc)

	tokens := &jwtpkg.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
	mockSvc.On("Login", mock.Anything, service.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}).Return(tokens, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"email":"test@example.com","password":"Password123!"}`
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
		Email:    "test@example.com",
		Password: "wrong",
	}).Return(nil, domain.ErrInvalidCredentials)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"email":"test@example.com","password":"wrong"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

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
