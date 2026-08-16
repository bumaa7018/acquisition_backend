package service

import (
	"context"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

// MenuService — цэсний харагдац (эрхийн 1-р давхарга).
type MenuService interface {
	List(ctx context.Context) ([]*domain.Menu, error)
	ListByRole(ctx context.Context, roleID domain.ID) ([]*domain.Menu, error)
	ListForUser(ctx context.Context, userID domain.ID) ([]*domain.Menu, error)
	Attach(ctx context.Context, roleID, menuID domain.ID) error
	Detach(ctx context.Context, roleID, menuID domain.ID) error
}

type menuService struct {
	repo     repository.MenuRepository
	roleRepo repository.RoleRepository
}

func NewMenuService(repo repository.MenuRepository, roleRepo repository.RoleRepository) MenuService {
	return &menuService{repo: repo, roleRepo: roleRepo}
}

func (s *menuService) List(ctx context.Context) ([]*domain.Menu, error) {
	return s.repo.List(ctx)
}

func (s *menuService) ListByRole(ctx context.Context, roleID domain.ID) ([]*domain.Menu, error) {
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return nil, err
	}
	return s.repo.ListByRole(ctx, roleID)
}

func (s *menuService) ListForUser(ctx context.Context, userID domain.ID) ([]*domain.Menu, error) {
	if userID == 0 {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListByUser(ctx, userID)
}

func (s *menuService) Attach(ctx context.Context, roleID, menuID domain.ID) error {
	if roleID == 0 || menuID == 0 {
		return domain.ErrInvalidInput
	}
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return err
	}
	return s.repo.AttachToRole(ctx, roleID, menuID)
}

func (s *menuService) Detach(ctx context.Context, roleID, menuID domain.ID) error {
	if roleID == 0 || menuID == 0 {
		return domain.ErrInvalidInput
	}
	return s.repo.DetachFromRole(ctx, roleID, menuID)
}
