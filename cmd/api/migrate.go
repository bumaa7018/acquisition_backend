package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func runMigrations(mainDSN, authDSN string) error {
	if err := runMigration("file://migrations/main", mainDSN, "main"); err != nil {
		return err
	}
	return ensureAuthSchema(authDSN)
}

func runMigration(path, dsn, label string) error {
	m, err := migrate.New(path, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	slog.Info("migrations applied", "db", label)
	return nil
}

func ensureAuthSchema(dsn string) error {
	sqlBytes, err := os.ReadFile("migrations/auth/000001_authdb_schema.up.sql")
	if err != nil {
		return err
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return err
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(ctx, string(sqlBytes)); err != nil {
		return err
	}
	slog.Info("auth schema ensured")
	return nil
}
