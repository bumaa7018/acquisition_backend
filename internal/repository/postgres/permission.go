package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PermissionRepo — sdplatform.sd_permission дээр ШУУД ажиллана.
//
// Энэ схемд эрх нь ЗӨВХӨН үйлдэл (read/create/update/delete); нөөц нь ролийн
// role_type_code-оос ирдэг тул энд resource багана байхгүй.
type PermissionRepo struct {
	db *pgxpool.Pool
}

func NewPermissionRepo(db *pgxpool.Pool) *PermissionRepo {
	return &PermissionRepo{db: db}
}

const permissionSelectSQL = `
	SELECT p.permission_id, COALESCE(p.code, ''), COALESCE(p.name, ''),
	       COALESCE(p.created_at::timestamptz, NOW())
	FROM sdplatform.sd_permission p`

func scanPermission(row rowScanner, p *domain.Permission) error {
	if err := row.Scan(&p.ID, &p.Code, &p.Action, &p.CreatedAt); err != nil {
		return err
	}
	p.Name = p.Action
	return nil
}

func (r *PermissionRepo) Create(ctx context.Context, p *domain.Permission) error {
	action := p.Action
	if action == "" {
		// "land:read" хэлбэрээр ирвэл үйлдлийг нь салгаж авна.
		_, action = domain.SplitPermissionKey(p.Name)
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_permission (code, name, created_at)
		VALUES (LEFT(COALESCE(NULLIF($1, ''), $2), 45), LEFT($2, 250), $3)
		RETURNING permission_id`,
		p.Code, action, p.CreatedAt).Scan(&p.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	p.Action = action
	p.Name = action
	return nil
}

func (r *PermissionRepo) GetByID(ctx context.Context, id domain.ID) (*domain.Permission, error) {
	p := &domain.Permission{}
	if err := scanPermission(r.db.QueryRow(ctx, permissionSelectSQL+` WHERE p.permission_id = $1`, id), p); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

// GetByName — "land:read" гэж ирвэл үйлдлийг нь ("read") хайна.
func (r *PermissionRepo) GetByName(ctx context.Context, name string) (*domain.Permission, error) {
	_, action := domain.SplitPermissionKey(name)
	p := &domain.Permission{}
	if err := scanPermission(r.db.QueryRow(ctx, permissionSelectSQL+` WHERE p.name = $1`, action), p); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PermissionRepo) Delete(ctx context.Context, id domain.ID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM sdplatform.sd_permission WHERE permission_id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PermissionRepo) List(ctx context.Context) ([]*domain.Permission, error) {
	rows, err := r.db.Query(ctx, permissionSelectSQL+` ORDER BY COALESCE(p.name, '')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*domain.Permission
	for rows.Next() {
		p := &domain.Permission{}
		if err := scanPermission(rows, p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}
