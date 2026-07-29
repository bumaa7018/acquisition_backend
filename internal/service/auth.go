package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/email"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
	Logout(ctx context.Context, accessToken, refreshToken string) error
	ForgotPassword(ctx context.Context, emailAddr string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

type RegisterRequest struct {
	Username  string `validate:"required"`
	Email     string `validate:"omitempty"`
	Password  string `validate:"required,min=8"`
	FirstName string `validate:"required"`
	LastName  string `validate:"required"`
	Position  string
}

type LoginRequest struct {
	Username string `validate:"required"`
	Password string `validate:"required"`
}

type authService struct {
	userRepo            repository.UserRepository
	pwdResetRepo        repository.PasswordResetRepository
	tokenRevocationRepo repository.TokenRevocationRepository
	jwtManager          *jwt.Manager
	emailSender         email.Sender
	resetBaseURL        string
}

func NewAuthService(
	userRepo repository.UserRepository,
	pwdResetRepo repository.PasswordResetRepository,
	jwtManager *jwt.Manager,
	emailSender email.Sender,
	resetBaseURL string,
	tokenRevocationRepo ...repository.TokenRevocationRepository,
) AuthService {
	var revocationRepo repository.TokenRevocationRepository
	if len(tokenRevocationRepo) > 0 {
		revocationRepo = tokenRevocationRepo[0]
		jwtManager.SetRevocationChecker(revocationRepo)
	}
	return &authService{
		userRepo:            userRepo,
		pwdResetRepo:        pwdResetRepo,
		tokenRevocationRepo: revocationRepo,
		jwtManager:          jwtManager,
		emailSender:         emailSender,
		resetBaseURL:        resetBaseURL,
	}
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*domain.User, error) {
	existing, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	hashed, err := hash.Password(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		ID:        uuid.New(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashed,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Position:  req.Position,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *authService) Login(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == domain.ErrNotFound {
		user, err = s.userRepo.GetByEmail(ctx, req.Username)
	}
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if !hash.ComparePassword(user.Password, req.Password) {
		return nil, domain.ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, domain.ErrForbidden
	}

	perms, err := s.userRepo.GetPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles, err := s.userRepo.GetRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	tokens, err := s.jwtManager.GenerateTokenPair(jwt.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    strings.TrimSpace(user.FirstName + " " + user.LastName),
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Position:    user.Position,
		Roles:       roles,
		Permissions: perms,
	})
	if err != nil {
		return nil, err
	}
	if err := s.cleanupExpiredRevokedTokens(ctx); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := s.jwtManager.ValidateRefreshTokenContext(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	// Идэвхгүй болгосон хэрэглэгч refresh-ээр хандах эрхээ сунгаж болохгүй.
	if !user.IsActive {
		return nil, domain.ErrForbidden
	}
	perms, err := s.userRepo.GetPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roles, err := s.userRepo.GetRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	tokens, err := s.jwtManager.GenerateTokenPair(jwt.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    strings.TrimSpace(user.FirstName + " " + user.LastName),
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Position:    user.Position,
		Roles:       roles,
		Permissions: perms,
	})
	if err != nil {
		return nil, err
	}
	if err := s.cleanupExpiredRevokedTokens(ctx); err != nil {
		return nil, err
	}
	if err := s.revokeClaims(ctx, claims, "refresh"); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *authService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken != "" {
		if claims, err := s.jwtManager.ParseAccessToken(accessToken); err == nil {
			if err := s.revokeClaims(ctx, claims, "access"); err != nil {
				return err
			}
		}
	}
	if refreshToken != "" {
		if claims, err := s.jwtManager.ParseRefreshToken(refreshToken); err == nil {
			if err := s.revokeClaims(ctx, claims, "refresh"); err != nil {
				return err
			}
		}
	}
	return nil
}

// ForgotPassword нь reset token үүсгэж, email-р илгээнэ.
// Email байхгүй ч алдаа буцаахгүй (enumeration хамгаалалт).
func (s *authService) ForgotPassword(ctx context.Context, emailAddr string) error {
	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil {
		// email олдохгүй ч мэдэгдэхгүй — чимээгүй буцаана
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)

	now := time.Now()
	resetToken := &domain.PasswordResetToken{
		ID:     uuid.New(),
		UserID: user.ID,
		// Токеныг DB-д hash хэлбэрээр хадгална — DB задарсан ч токен шууд
		// ашиглагдахгүй. Хэрэглэгч рүү зөвхөн түүхий токен (email-ээр) очно.
		Token:     hashResetToken(token),
		ExpiresAt: now.Add(1 * time.Hour),
		Used:      false,
		CreatedAt: now,
	}
	if err := s.pwdResetRepo.Create(ctx, resetToken); err != nil {
		return err
	}

	resetLink := s.resetBaseURL + token
	return s.emailSender.SendPasswordReset(emailAddr, resetLink)
}

// hashResetToken — reset токеныг DB-д хадгалах/хайхад ашиглах SHA-256 hash.
func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	resetToken, err := s.pwdResetRepo.GetByToken(ctx, hashResetToken(token))
	if err != nil {
		return domain.ErrInvalidToken
	}
	if resetToken.Used || time.Now().After(resetToken.ExpiresAt) {
		return domain.ErrExpiredToken
	}

	hashed, err := hash.Password(newPassword)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return err
	}
	// UpdatePassword-ээр бичнэ: Update() нь password баганад хүрдэггүй тул
	// түүгээр дамжуулбал нууц үг СОЛИГДОХГҮЙ (хэрэглэгч солигдсон гэж бодох).
	if err := s.userRepo.UpdatePassword(ctx, user.ID, hashed); err != nil {
		return err
	}
	if err := s.pwdResetRepo.MarkUsed(ctx, resetToken.ID); err != nil {
		return err
	}
	// Нууц үг солигдсон тул өмнө олгогдсон бүх токеныг хүчингүй болгоно.
	return s.userRepo.InvalidateSessions(ctx, user.ID)
}

func (s *authService) cleanupExpiredRevokedTokens(ctx context.Context) error {
	if s.tokenRevocationRepo == nil {
		return nil
	}
	return s.tokenRevocationRepo.DeleteExpired(ctx)
}

func (s *authService) revokeClaims(ctx context.Context, claims *jwt.Claims, tokenType string) error {
	if s.tokenRevocationRepo == nil || claims == nil || claims.ID == "" || claims.ExpiresAt == nil {
		return nil
	}
	return s.tokenRevocationRepo.RevokeToken(ctx, &domain.RevokedToken{
		JTI:       claims.ID,
		UserID:    claims.UserID,
		TokenType: tokenType,
		ExpiresAt: claims.ExpiresAt.Time,
		RevokedAt: time.Now(),
	})
}
