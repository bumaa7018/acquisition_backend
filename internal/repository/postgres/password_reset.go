package postgres

import (
	"context"
	"errors"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PasswordResetRepo — аппын ӨӨРИЙН өгөгдлийн сан дээр ажиллана
// (sdplatform схемд ийм ойлголт байхгүй тул тэнд хүснэгт нэмэхгүй).
type PasswordResetRepo struct {
	db     *pgxpool.Pool
	source string
}

func NewPasswordResetRepo(db *pgxpool.Pool) *PasswordResetRepo {
	return &PasswordResetRepo{db: db, source: userSourceAuth}
}

func (r *PasswordResetRepo) Create(ctx context.Context, t *domain.PasswordResetToken) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_source, user_id, token, expires_at, used, created_at)
		 VALUES ($1, $7, $2, $3, $4, $5, $6)`,
		t.ID, t.UserID, t.Token, t.ExpiresAt, t.Used, t.CreatedAt, r.source)
	return err
}

func (r *PasswordResetRepo) GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	t := &domain.PasswordResetToken{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, used, created_at FROM password_reset_tokens
		 WHERE token = $1 AND user_source = $2`, token, r.source).
		Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.Used, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *PasswordResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE password_reset_tokens SET used = true WHERE id = $1`, id)
	return err
}

func (r *PasswordResetRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM password_reset_tokens WHERE expires_at < $1`, time.Now())
	return err
}
