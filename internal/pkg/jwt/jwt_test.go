package jwt_test

import (
	"context"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	jwtpkg "acquisition_backend/internal/pkg/jwt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type revocationChecker struct {
	revokedJTI string
}

func (c revocationChecker) IsTokenRevoked(_ context.Context, jti string) (bool, error) {
	return jti == c.revokedJTI, nil
}

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
	userID := domain.ID(42)
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
	userID := domain.ID(42)

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
	userID := domain.ID(42)
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
	userID := domain.ID(42)
	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{ID: userID})
	require.NoError(t, err)

	_, err = m.ValidateAccessToken(pair.AccessToken)
	assert.ErrorIs(t, err, domain.ErrExpiredToken)
}

func TestValidateAccessTokenRejectsRevokedToken(t *testing.T) {
	m := newManager()
	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{ID: domain.ID(201)})
	require.NoError(t, err)

	claims, err := m.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)
	m.SetRevocationChecker(revocationChecker{revokedJTI: claims.ID})

	_, err = m.ValidateAccessToken(pair.AccessToken)
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
}
