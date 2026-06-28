package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/config"
)

func main() {
	dbName := flag.String("db", "main", "database to seed: main or auth")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/seed-sql -db main|auth scripts/file.sql")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	dsn := cfg.DB.DSN()
	switch *dbName {
	case "main":
		dsn = cfg.DB.DSN()
	case "auth":
		dsn = cfg.AuthDB.DSN()
	default:
		fmt.Fprintf(os.Stderr, "unknown db %q; use main or auth\n", *dbName)
		os.Exit(2)
	}

	sqlBytes, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "read sql file: %v\n", err)
		os.Exit(1)
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse dsn: %v\n", err)
		os.Exit(1)
	}
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	poolCfg.ConnConfig.OnNotice = func(_ *pgconn.PgConn, notice *pgconn.Notice) {
		fmt.Fprintf(os.Stderr, "NOTICE: %s\n", notice.Message)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping db: %v\n", err)
		os.Exit(1)
	}

	if _, err := db.Exec(ctx, string(sqlBytes)); err != nil {
		fmt.Fprintf(os.Stderr, "execute sql: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("seed applied: db=%s file=%s\n", *dbName, flag.Arg(0))
}
