package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ensureSeedHistory(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS seed_history (
			name       TEXT PRIMARY KEY,
			seeded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func isSeedApplied(ctx context.Context, db *pgxpool.Pool, name string) (bool, error) {
	var count int
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM seed_history WHERE name = $1`, name).Scan(&count)
	return count > 0, err
}

func markSeedApplied(ctx context.Context, db *pgxpool.Pool, name string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO seed_history (name, seeded_at) VALUES ($1, $2) ON CONFLICT (name) DO NOTHING
	`, name, time.Now())
	return err
}
