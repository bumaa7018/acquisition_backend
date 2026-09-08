package service_test

import (
	"context"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEmailSender — email.Sender interface-ийн mock
type mockEmailSender struct{ mock.Mock }

func (m *mockEmailSender) SendPasswordReset(to, link string) error {
	return m.Called(to, link).Error(0)
}

func (m *mockEmailSender) SendWithAttachment(to, subject, bodyText, attachmentName string, attachment []byte) error {
	return m.Called(to, subject, bodyText, attachmentName, attachment).Error(0)
}

func newJWTManager() *jwtpkg.Manager {
	return jwtpkg.NewManager(
		"access-secret-32-chars-padding-1234",
		"refresh-secret-32-chars-padding-1234",
		15*time.Minute, 168*time.Hour,
	)
}

func newAuthSvc(userRepo *mocks.MockUserRepository, pwdRepo *mocks.MockPasswordResetRepository, es *mockEmailSender) service.AuthService {
	return service.NewAuthService(userRepo, pwdRepo, newJWTManager(), es, "http://localhost:3000/reset?token=")
}

// ── Login ────────────────────────────────────────────────────

func TestAuthService_Login_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	pwdRepo := &mocks.MockPasswordResetRepository{}
	es := &mockEmailSender{}

	userID := domain.ID(42)
	hashedPwd := "$2a$10$P/29K73vxNfgjtfK6/aASuSCtdYJzUSYKH5wCI9IUyQukG3aXJ782" // "secret"
	user := &domain.User{ID: userID, Username: "testuser", Password: hashedPwd, IsActive: true}

	userRepo.On("GetActiveByLogin", context.Background(), "testuser").Return(user, nil)
	userRepo.On("GetPermissions", context.Background(), userID).Return([]string{"users:read"}, nil)
	userRepo.On("GetRoles", context.Background(), userID).Return([]string{"admin"}, nil)

	svc := newAuthSvc(userRepo, pwdRepo, es)
	tokens, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "testuser", Password: "secret",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	pwdRepo := &mocks.MockPasswordResetRepository{}
	es := &mockEmailSender{}

	hashedPwd := "$2a$10$P/29K73vxNfgjtfK6/aASuSCtdYJzUSYKH5wCI9IUyQukG3aXJ782"
	user := &domain.User{ID: domain.ID(7), Username: "testuser", Password: hashedPwd, IsActive: true}
	userRepo.On("GetActiveByLogin", context.Background(), "testuser").Return(user, nil)

	svc := newAuthSvc(userRepo, pwdRepo, es)
	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "testuser", Password: "wrongpassword",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	pwdRepo := &mocks.MockPasswordResetRepository{}
	es := &mockEmailSender{}

	userRepo.On("GetActiveByLogin", context.Background(), "noone").Return(nil, domain.ErrNotFound)

	svc := newAuthSvc(userRepo, pwdRepo, es)
	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "noone", Password: "anything",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

// ── ForgotPassword ───────────────────────────────────────────

func TestAuthService_ForgotPassword_SendsEmail(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	pwdRepo := &mocks.MockPasswordResetRepository{}
	es := &mockEmailSender{}

	user := &domain.User{ID: domain.ID(7), Email: "test@example.com"}
	userRepo.On("GetByEmail", context.Background(), "test@example.com").Return(user, nil)
	pwdRepo.On("Create", context.Background(), mock.MatchedBy(func(t *domain.PasswordResetToken) bool { return t != nil })).Return(nil)
	es.On("SendPasswordReset", "test@example.com", mock.AnythingOfType("string")).Return(nil)

	svc := newAuthSvc(userRepo, pwdRepo, es)
	err := svc.ForgotPassword(context.Background(), "test@example.com")

	require.NoError(t, err)
	es.AssertExpectations(t)
	pwdRepo.AssertExpectations(t)
}

func TestAuthService_ForgotPassword_UnknownEmail_NoError(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	pwdRepo := &mocks.MockPasswordResetRepository{}
	es := &mockEmailSender{}

	userRepo.On("GetByEmail", context.Background(), "unknown@example.com").Return(nil, domain.ErrNotFound)
	es.AssertNotCalled(t, "SendPasswordReset")

	svc := newAuthSvc(userRepo, pwdRepo, es)
	err := svc.ForgotPassword(context.Background(), "unknown@example.com")

	require.NoError(t, err)
}

// ── ResetPassword ────────────────────────────────────────────

func TestAuthService_ResetPassword_ExpiredToken(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	pwdRepo := &mocks.MockPasswordResetRepository{}
	es := &mockEmailSender{}

	resetToken := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    domain.ID(7),
		Token:     "sometoken",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Used:      false,
	}
	// Токен DB-д hash хэлбэрээр хайгддаг тул аргументыг чөлөөтэй тааруулна.
	pwdRepo.On("GetByToken", context.Background(), mock.Anything).Return(resetToken, nil)

	svc := newAuthSvc(userRepo, pwdRepo, es)
	err := svc.ResetPassword(context.Background(), "sometoken", "NewPassword123!")

	assert.ErrorIs(t, err, domain.ErrExpiredToken)
}

func mock_any_user() interface{} {
	return mock.MatchedBy(func(u *domain.User) bool { return u != nil })
}
