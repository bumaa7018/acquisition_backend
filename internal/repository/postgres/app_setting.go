package postgres

// Системийн тохиргоо (app_setting) — key → jsonb.

import (
	"context"
	"errors"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type settingRepo struct{ db *pgxpool.Pool }

func NewSettingRepo(db *pgxpool.Pool) repository.SettingRepository { return &settingRepo{db: db} }

// Get — тохиргооны JSON утга. Мөр байхгүй бол ErrNotFound (тохируулаагүй).
func (r *settingRepo) Get(ctx context.Context, key string) (*domain.SettingRecord, error) {
	rec := &domain.SettingRecord{Key: key}
	err := r.db.QueryRow(ctx, `
		SELECT value::text, updated_at, COALESCE(updated_by,'')
		FROM app_setting WHERE key = $1
	`, key).Scan(&rec.Value, &rec.UpdatedAt, &rec.UpdatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// Upsert — тохиргоог орлуулна (value нь бүрэн бүтэн JSON байх ёстой).
func (r *settingRepo) Upsert(ctx context.Context, key, valueJSON, updatedBy string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO app_setting (key, value, updated_by)
		VALUES ($1, $2::jsonb, NULLIF($3,''))
		ON CONFLICT (key) DO UPDATE SET
			value      = EXCLUDED.value,
			updated_at = NOW(),
			updated_by = EXCLUDED.updated_by
	`, key, valueJSON, updatedBy)
	return err
}

// Delete — тохиргоог устгана (дэлгэц өгөгдмөл рүү буцна).
func (r *settingRepo) Delete(ctx context.Context, key string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM app_setting WHERE key = $1`, key)
	return err
}
