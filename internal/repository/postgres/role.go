package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/domain"
)

type RoleRepo struct {
	db *pgxpool.Pool
}

func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO roles (id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		role.ID, role.Name, role.Description, role.CreatedAt, role.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *RoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role := &domain.Role{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles WHERE id = $1`, id).
		Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	perms, err := r.getPermissions(ctx, id)
	if err != nil {
		return nil, err
	}
	role.Permissions = perms
	return role, nil
}

func (r *RoleRepo) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	role := &domain.Role{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles WHERE name = $1`, name).
		Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *RoleRepo) Update(ctx context.Context, role *domain.Role) error {
	res, err := r.db.Exec(ctx,
		`UPDATE roles SET name=$1, description=$2, updated_at=$3 WHERE id=$4`,
		role.Name, role.Description, role.UpdatedAt, role.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *RoleRepo) List(ctx context.Context) ([]*domain.Role, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role := &domain.Role{}
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *RoleRepo) AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		roleID, permissionID)
	return err
}

func (r *RoleRepo) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`, roleID, permissionID)
	return err
}

func (r *RoleRepo) AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, roleID)
	return err
}

func (r *RoleRepo) RemoveFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	return err
}

func (r *RoleRepo) getPermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.name, p.description, p.resource, p.action, p.created_at
		 FROM permissions p JOIN role_permissions rp ON rp.permission_id = p.id WHERE rp.role_id = $1`,
		roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		p := domain.Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
