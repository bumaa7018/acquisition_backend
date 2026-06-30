package service

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

type RoleService interface {
	Create(ctx context.Context, req CreateRoleRequest) (*domain.Role, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*domain.Role, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*domain.Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveFromUser(ctx context.Context, userID, roleID uuid.UUID) error
}

type CreateRoleRequest struct {
	Name        string `validate:"required"`
	Description string
}

type UpdateRoleRequest struct {
	Name        string
	Description string
}

type roleService struct {
	roleRepo repository.RoleRepository
}

func NewRoleService(roleRepo repository.RoleRepository) RoleService {
	return &roleService{roleRepo: roleRepo}
}

func (s *roleService) Create(ctx context.Context, req CreateRoleRequest) (*domain.Role, error) {
	now := time.Now()
	role := &domain.Role{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *roleService) Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.roleRepo.Delete(ctx, id)
}

func (s *roleService) List(ctx context.Context) ([]*domain.Role, error) {
	return s.roleRepo.List(ctx)
}

func (s *roleService) AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.roleRepo.AssignPermission(ctx, roleID, permissionID)
}

func (s *roleService) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.roleRepo.RemovePermission(ctx, roleID, permissionID)
}

func (s *roleService) AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.roleRepo.AssignToUser(ctx, userID, roleID)
}

func (s *roleService) RemoveFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.roleRepo.RemoveFromUser(ctx, userID, roleID)
}
