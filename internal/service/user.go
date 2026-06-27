package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/hash"
	"github.com/your-org/go-api-template/internal/repository"
)

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error)
	Create(ctx context.Context, req CreateUserRequest) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CreateUserRequest struct {
	Username  string `validate:"required"`
	Email     string `validate:"omitempty"`
	Password  string `validate:"required,min=8"`
	FirstName string `validate:"required"`
	LastName  string `validate:"required"`
	Position  string
	IsActive  bool
}

type UpdateUserRequest struct {
	Username  string `validate:"omitempty"`
	Email     string `validate:"omitempty"`
	FirstName string
	LastName  string
	Position  string
	IsActive  *bool
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *userService) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, total, nil
}

func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
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
		IsActive:  req.IsActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Position != "" {
		user.Position = req.Position
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}
