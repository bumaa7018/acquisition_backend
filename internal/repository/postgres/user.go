package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/domain"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (id, email, password, first_name, last_name, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.Password, u.FirstName, u.LastName,
		u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, email, password, first_name, last_name, is_active, created_at, updated_at, deleted_at
	          FROM users WHERE id = $1 AND deleted_at IS NULL`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password, first_name, last_name, is_active, created_at, updated_at, deleted_at
	          FROM users WHERE email = $1 AND deleted_at IS NULL`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	query := `UPDATE users SET email=$1, first_name=$2, last_name=$3, is_active=$4, updated_at=$5
	          WHERE id=$6 AND deleted_at IS NULL`
	res, err := r.db.Exec(ctx, query,
		u.Email, u.FirstName, u.LastName, u.IsActive, u.UpdatedAt, u.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	res, err := r.db.Exec(ctx,
		`UPDATE users SET deleted_at=$1 WHERE id=$2 AND deleted_at IS NULL`, now, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, f domain.UserFilter) ([]*domain.User, int64, error) {
	args := []any{}
	argIdx := 1
	where := "WHERE deleted_at IS NULL"

	if f.Search != "" {
		where += fmt.Sprintf(" AND (email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	if f.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *f.IsActive)
		argIdx++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	query := fmt.Sprintf(`SELECT id, email, password, first_name, last_name, is_active, created_at, updated_at, deleted_at
	                       FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
			&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *UserRepo) GetPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `SELECT DISTINCT p.name
	          FROM permissions p
	          JOIN role_permissions rp ON rp.permission_id = p.id
	          JOIN user_roles ur ON ur.role_id = rp.role_id
	          WHERE ur.user_id = $1`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		perms = append(perms, name)
	}
	return perms, rows.Err()
}
