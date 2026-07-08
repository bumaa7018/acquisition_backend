package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (id, username, email, password, first_name, last_name, position, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.Exec(ctx, query,
		u.ID, u.Username, u.Email, u.Password, u.FirstName, u.LastName,
		u.Position, u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, username, email, password, first_name, last_name, COALESCE(position, ''), is_active, created_at, updated_at, deleted_at
	          FROM users WHERE id = $1 AND deleted_at IS NULL`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.Position, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, username, email, password, first_name, last_name, COALESCE(position, ''), is_active, created_at, updated_at, deleted_at
	          FROM users WHERE email = $1 AND deleted_at IS NULL`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.Position, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, email, password, first_name, last_name, COALESCE(position, ''), is_active, created_at, updated_at, deleted_at
	          FROM users WHERE username = $1 AND deleted_at IS NULL`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.Position, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	query := `UPDATE users SET username=$1, email=$2, first_name=$3, last_name=$4, position=$5, is_active=$6, updated_at=$7
	          WHERE id=$8 AND deleted_at IS NULL`
	res, err := r.db.Exec(ctx, query,
		u.Username, u.Email, u.FirstName, u.LastName, u.Position, u.IsActive, u.UpdatedAt, u.ID)
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
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d OR position ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	if f.Role != "" {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = users.id AND r.name = $%d
		)`, argIdx)
		args = append(args, f.Role)
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
	query := fmt.Sprintf(`SELECT id, username, email, password, first_name, last_name, COALESCE(position, ''), is_active, created_at, updated_at, deleted_at
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
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
			&u.Position, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// InvalidateSessions — хэрэглэгчийн токен эх цэгийг NOW() болгож, өмнө олгогдсон
// бүх токеныг хүчингүй болгоно (секундэд тайрч, шинэ нэвтрэлттэй мөргөлдөхөөс сэргийлнэ).
func (r *UserRepo) InvalidateSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET tokens_valid_from = date_trunc('second', NOW()) WHERE id = $1`, userID)
	return err
}

// IsSessionValid — эх цэг тавиагүй бол (NULL) эсвэл токены issuedAt эх цэгээс
// хойш байвал true. Эх цэгээс өмнө олгогдсон токенд false.
func (r *UserRepo) IsSessionValid(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	var invalid bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE id = $1 AND tokens_valid_from IS NOT NULL AND tokens_valid_from > $2
		)
	`, userID, issuedAt).Scan(&invalid)
	if err != nil {
		return false, err
	}
	return !invalid, nil
}

func (r *UserRepo) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `SELECT r.name FROM roles r
	          JOIN user_roles ur ON ur.role_id = r.id
	          WHERE ur.user_id = $1`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
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
