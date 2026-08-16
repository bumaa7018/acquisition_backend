package postgres

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RevokedTokenRepo — аппын ӨӨРИЙН өгөгдлийн сан дээр ажиллана.
type RevokedTokenRepo struct {
	db     *pgxpool.Pool
	source string
}

func NewRevokedTokenRepo(db *pgxpool.Pool) *RevokedTokenRepo {
	return &RevokedTokenRepo{db: db, source: userSourceAuth}
}

func (r *RevokedTokenRepo) RevokeToken(ctx context.Context, token *domain.RevokedToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO revoked_tokens (jti, user_source, user_id, token_type, expires_at, revoked_at)
		VALUES ($1, $6, $2, $3, $4, $5)
		ON CONFLICT (jti) DO NOTHING
	`, token.JTI, token.UserID, token.TokenType, token.ExpiresAt, token.RevokedAt, r.source)
	return err
}

func (r *RevokedTokenRepo) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	var revoked bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM revoked_tokens
			WHERE jti = $1
			  AND expires_at > NOW()
		)
	`, jti).Scan(&revoked)
	return revoked, err
}

func (r *RevokedTokenRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM revoked_tokens WHERE expires_at < $1`, time.Now())
	return err
}
