package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/email"
	"github.com/your-org/go-api-template/internal/pkg/hash"
	"github.com/your-org/go-api-template/internal/pkg/jwt"
	"github.com/your-org/go-api-template/internal/repository"
)

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req LoginRequest) (*jwt.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
	ForgotPassword(ctx context.Context, emailAddr string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

type RegisterRequest struct {
	Email     string `validate:"required,email"`
	Password  string `validate:"required,min=8"`
	FirstName string `validate:"required"`
	LastName  string `validate:"required"`
}

type LoginRequest struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

type authService struct {
	userRepo     repository.UserRepository
	pwdResetRepo repository.PasswordResetRepository
	jwtManager   *jwt.Manager
	emailSender  email.Sender
	resetBaseURL string
}

func NewAuthService(
	userRepo repository.UserRepository,
	pwdResetRepo repository.PasswordResetRepository,
	jwtManager *jwt.Manager,
	emailSender email.Sender,
	resetBaseURL string,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		pwdResetRepo: pwdResetRepo,
		jwtManager:   jwtManager,
		emailSender:  emailSender,
		resetBaseURL: resetBaseURL,
	}
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*domain.User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
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
		Email:     req.Email,
		Password:  hashed,
		FirstName: req.FirstName,
		LastName:  req.LastName,
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
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
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
	return s.jwtManager.GenerateTokenPair(user.ID, perms)
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	perms, err := s.userRepo.GetPermissions(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	return s.jwtManager.GenerateTokenPair(claims.UserID, perms)
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
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     token,
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

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	resetToken, err := s.pwdResetRepo.GetByToken(ctx, token)
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
	user.Password = hashed
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}
	return s.pwdResetRepo.MarkUsed(ctx, resetToken.ID)
}
