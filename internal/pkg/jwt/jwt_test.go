package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	jwtpkg "github.com/your-org/go-api-template/internal/pkg/jwt"
)

func newManager() *jwtpkg.Manager {
	return jwtpkg.NewManager(
		"access-secret-32-chars-padding-1234",
		"refresh-secret-32-chars-padding-1234",
		15*time.Minute,
		168*time.Hour,
	)
}

func TestGenerateAndValidateTokenPair(t *testing.T) {
	m := newManager()
	userID := uuid.New()
	perms := []string{"users:read", "users:write"}

	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{ID: userID, Username: "testuser", Permissions: perms})
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)

	claims, err := m.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, perms, claims.Permissions)
}

func TestValidateRefreshToken(t *testing.T) {
	m := newManager()
	userID := uuid.New()

	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{ID: userID})
	require.NoError(t, err)

	claims, err := m.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestInvalidAccessTokenReturnsError(t *testing.T) {
	m := newManager()
	_, err := m.ValidateAccessToken("invalid.token.here")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestWrongSecretRefreshToken(t *testing.T) {
	m := newManager()
	userID := uuid.New()
	pair, _ := m.GenerateTokenPair(jwtpkg.UserInfo{ID: userID})

	// Access token should fail refresh validation
	_, err := m.ValidateRefreshToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestExpiredToken(t *testing.T) {
	m := jwtpkg.NewManager(
		"access-secret-32-chars-padding-1234",
		"refresh-secret-32-chars-padding-1234",
		-1*time.Second, // already expired
		168*time.Hour,
	)
	userID := uuid.New()
	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{ID: userID})
	require.NoError(t, err)

	_, err = m.ValidateAccessToken(pair.AccessToken)
	assert.ErrorIs(t, err, domain.ErrExpiredToken)
}
