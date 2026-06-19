package main

import (
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(mainDSN, authDSN string) error {
	if err := runMigration("file://migrations/main", mainDSN, "main"); err != nil {
		return err
	}
	return runMigration("file://migrations/auth", authDSN, "auth")
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
