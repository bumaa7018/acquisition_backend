package jwt

import (
	"context"
	"errors"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      domain.ID `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Position    string    `json:"position"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Manager struct {
	accessSecret      string
	refreshSecret     string
	accessTTL         time.Duration
	refreshTTL        time.Duration
	revocationChecker RevocationChecker
	sessionValidator  SessionValidator
}

type RevocationChecker interface {
	IsTokenRevoked(ctx context.Context, jti string) (bool, error)
}

// SessionValidator — тухайн хэрэглэгчийн токен эх цэгийг шалгаж, нууц үг
// сэргээх / албадан гаргах үед олгогдсон бүх хуучин токеныг хүчингүй болгоно.
type SessionValidator interface {
	IsSessionValid(ctx context.Context, userID domain.ID, issuedAt time.Time) (bool, error)
}

func NewManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (m *Manager) SetRevocationChecker(checker RevocationChecker) {
	m.revocationChecker = checker
}

func (m *Manager) SetSessionValidator(v SessionValidator) {
	m.sessionValidator = v
}

type UserInfo struct {
	ID          domain.ID
	Username    string
	Email       string
	FullName    string
	FirstName   string
	LastName    string
	Position    string
	Roles       []string
	Permissions []string
}

func (m *Manager) GenerateTokenPair(u UserInfo) (*TokenPair, error) {
	access, err := m.generateToken(u, m.accessSecret, m.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := m.generateToken(UserInfo{ID: u.ID}, m.refreshSecret, m.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (m *Manager) generateToken(u UserInfo, secret string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:      u.ID,
		Username:    u.Username,
		Email:       u.Email,
		FullName:    u.FullName,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Position:    u.Position,
		Roles:       u.Roles,
		Permissions: u.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (m *Manager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	return m.ValidateAccessTokenContext(context.Background(), tokenStr)
}

func (m *Manager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return m.ValidateRefreshTokenContext(context.Background(), tokenStr)
}

func (m *Manager) ValidateAccessTokenContext(ctx context.Context, tokenStr string) (*Claims, error) {
	return m.validateToken(ctx, tokenStr, m.accessSecret)
}

func (m *Manager) ValidateRefreshTokenContext(ctx context.Context, tokenStr string) (*Claims, error) {
	return m.validateToken(ctx, tokenStr, m.refreshSecret)
}

func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr, m.accessSecret)
}

func (m *Manager) ParseRefreshToken(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr, m.refreshSecret)
}

func (m *Manager) parseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrExpiredToken
		}
		return nil, domain.ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}
	return claims, nil
}

func (m *Manager) validateToken(ctx context.Context, tokenStr, secret string) (*Claims, error) {
	claims, err := m.parseToken(tokenStr, secret)
	if err != nil {
		return nil, err
	}
	if m.revocationChecker == nil || claims.ID == "" {
		return claims, nil
	}
	revoked, err := m.revocationChecker.IsTokenRevoked(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, domain.ErrInvalidToken
	}
	// Хэрэглэгчийн токен эх цэгийг шалгана (нууц үг сэргээх зэргээр хүчингүй болсон эсэх).
	if m.sessionValidator != nil && claims.IssuedAt != nil {
		valid, err := m.sessionValidator.IsSessionValid(ctx, claims.UserID, claims.IssuedAt.Time)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, domain.ErrInvalidToken
		}
	}
	return claims, nil
}
