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

type PermissionRepo struct {
	db *pgxpool.Pool
}

func NewPermissionRepo(db *pgxpool.Pool) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) Create(ctx context.Context, p *domain.Permission) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO permissions (id, name, description, resource, action, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, p.Description, p.Resource, p.Action, p.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *PermissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	p := &domain.Permission{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, resource, action, created_at FROM permissions WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PermissionRepo) GetByName(ctx context.Context, name string) (*domain.Permission, error) {
	p := &domain.Permission{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, resource, action, created_at FROM permissions WHERE name = $1`, name).
		Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PermissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM permissions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PermissionRepo) List(ctx context.Context) ([]*domain.Permission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, resource, action, created_at FROM permissions ORDER BY resource, action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		p := &domain.Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
