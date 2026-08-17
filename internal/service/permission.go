package service

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type PermissionService interface {
	Create(ctx context.Context, req CreatePermissionRequest) (*domain.Permission, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Permission, error)
	Delete(ctx context.Context, id domain.ID) error
	List(ctx context.Context) ([]*domain.Permission, error)
}

type CreatePermissionRequest struct {
	Name        string `validate:"required"`
	Description string
	Resource    string `validate:"required"`
	Action      string `validate:"required"`
}

type permissionService struct {
	permRepo repository.PermissionRepository
}

func NewPermissionService(permRepo repository.PermissionRepository) PermissionService {
	return &permissionService{permRepo: permRepo}
}

func (s *permissionService) Create(ctx context.Context, req CreatePermissionRequest) (*domain.Permission, error) {
	p := &domain.Permission{
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
		CreatedAt:   time.Now(),
	}
	if err := s.permRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *permissionService) GetByID(ctx context.Context, id domain.ID) (*domain.Permission, error) {
	return s.permRepo.GetByID(ctx, id)
}

func (s *permissionService) Delete(ctx context.Context, id domain.ID) error {
	return s.permRepo.Delete(ctx, id)
}

func (s *permissionService) List(ctx context.Context) ([]*domain.Permission, error) {
	return s.permRepo.List(ctx)
}
