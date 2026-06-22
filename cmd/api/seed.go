package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/config"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/pkg/hash"
	"github.com/your-org/go-api-template/internal/repository"
)

var defaultPermissions = []domain.Permission{
	{Name: "users:read", Description: "Read users", Resource: "users", Action: "read"},
	{Name: "users:create", Description: "Create users", Resource: "users", Action: "create"},
	{Name: "users:update", Description: "Update users", Resource: "users", Action: "update"},
	{Name: "users:delete", Description: "Delete users", Resource: "users", Action: "delete"},
	{Name: "roles:read", Description: "Read roles", Resource: "roles", Action: "read"},
	{Name: "roles:create", Description: "Create roles", Resource: "roles", Action: "create"},
	{Name: "roles:update", Description: "Update roles", Resource: "roles", Action: "update"},
	{Name: "roles:delete", Description: "Delete roles", Resource: "roles", Action: "delete"},
	{Name: "permissions:read", Description: "Read permissions", Resource: "permissions", Action: "read"},
	{Name: "permissions:create", Description: "Create permissions", Resource: "permissions", Action: "create"},
	{Name: "permissions:delete", Description: "Delete permissions", Resource: "permissions", Action: "delete"},
	{Name: "land:read", Description: "Read land acquisitions", Resource: "land", Action: "read"},
	{Name: "land:create", Description: "Create land acquisitions", Resource: "land", Action: "create"},
	{Name: "land:update", Description: "Update land acquisitions", Resource: "land", Action: "update"},
	{Name: "land:delete", Description: "Delete land acquisitions", Resource: "land", Action: "delete"},
}

type Seeder struct {
	db       *pgxpool.Pool
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

func NewSeeder(db *pgxpool.Pool, userRepo repository.UserRepository, roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *Seeder {
	return &Seeder{db: db, userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo}
}

func (s *Seeder) Seed(ctx context.Context, adminCfg config.AdminConfig) error {
	permIDs, err := s.seedPermissions(ctx)
	if err != nil {
		return err
	}
	adminRole, err := s.seedAdminRole(ctx, permIDs)
	if err != nil {
		return err
	}
	return s.seedAdminUser(ctx, adminCfg, adminRole.ID)
}

func (s *Seeder) seedPermissions(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, p := range defaultPermissions {
		existing, err := s.permRepo.GetByName(ctx, p.Name)
		if err == nil {
			ids = append(ids, existing.ID)
			continue
		}
		p.ID = uuid.New()
		p.CreatedAt = time.Now()
		if err := s.permRepo.Create(ctx, &p); err != nil {
			return nil, err
		}
		ids = append(ids, p.ID)
		slog.Info("permission created", "name", p.Name)
	}
	return ids, nil
}

func (s *Seeder) seedAdminRole(ctx context.Context, permIDs []uuid.UUID) (*domain.Role, error) {
	existing, err := s.roleRepo.GetByName(ctx, "admin")
	if err == nil {
		for _, pid := range permIDs {
			_ = s.roleRepo.AssignPermission(ctx, existing.ID, pid)
		}
		return existing, nil
	}

	now := time.Now()
	role := &domain.Role{
		ID:          uuid.New(),
		Name:        "admin",
		Description: "System administrator with full access",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	for _, pid := range permIDs {
		if err := s.roleRepo.AssignPermission(ctx, role.ID, pid); err != nil {
			return nil, err
		}
	}
	slog.Info("admin role created")
	return role, nil
}

func (s *Seeder) seedAdminUser(ctx context.Context, cfg config.AdminConfig, roleID uuid.UUID) error {
	existing, err := s.userRepo.GetByEmail(ctx, cfg.Email)
	if err == nil {
		_ = s.roleRepo.AssignToUser(ctx, existing.ID, roleID)
		return nil
	}

	hashed, err := hash.Password(cfg.Password)
	if err != nil {
		return err
	}

	now := time.Now()
	user := &domain.User{
		ID:        uuid.New(),
		Username:  "admin",
		Email:     cfg.Email,
		Password:  hashed,
		FirstName: "Admin",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	if err := s.roleRepo.AssignToUser(ctx, user.ID, roleID); err != nil {
		return err
	}
	slog.Info("admin user created", "email", cfg.Email)
	return nil
}
