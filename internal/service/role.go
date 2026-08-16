package service

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

)

type RoleService interface {
	Create(ctx context.Context, req CreateRoleRequest) (*domain.Role, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Role, error)
	Update(ctx context.Context, id domain.ID, req UpdateRoleRequest) (*domain.Role, error)
	Delete(ctx context.Context, id domain.ID) error
	List(ctx context.Context) ([]*domain.Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID domain.ID) error
	RemovePermission(ctx context.Context, roleID, permissionID domain.ID) error
	AssignToUser(ctx context.Context, userID, roleID domain.ID) error
	RemoveFromUser(ctx context.Context, userID, roleID domain.ID) error
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

func (s *roleService) GetByID(ctx context.Context, id domain.ID) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *roleService) Update(ctx context.Context, id domain.ID, req UpdateRoleRequest) (*domain.Role, error) {
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

func (s *roleService) Delete(ctx context.Context, id domain.ID) error {
	return s.roleRepo.Delete(ctx, id)
}

func (s *roleService) List(ctx context.Context) ([]*domain.Role, error) {
	return s.roleRepo.List(ctx)
}

func (s *roleService) AssignPermission(ctx context.Context, roleID, permissionID domain.ID) error {
	return s.roleRepo.AssignPermission(ctx, roleID, permissionID)
}

func (s *roleService) RemovePermission(ctx context.Context, roleID, permissionID domain.ID) error {
	return s.roleRepo.RemovePermission(ctx, roleID, permissionID)
}

func (s *roleService) AssignToUser(ctx context.Context, userID, roleID domain.ID) error {
	return s.roleRepo.AssignToUser(ctx, userID, roleID)
}

func (s *roleService) RemoveFromUser(ctx context.Context, userID, roleID domain.ID) error {
	return s.roleRepo.RemoveFromUser(ctx, userID, roleID)
}
