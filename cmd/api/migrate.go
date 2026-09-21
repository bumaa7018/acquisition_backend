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

// baselineVersion — migrations/main/000001_appdb_baseline нь хуучин 96
// миграцын ЭЦСИЙН үр дүн. Тэр бүтэцтэй сан үүгээр "тамгалагдана".
const baselineVersion = 1

func runMigrations(mainDSN, authDSN string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := reconcileMainBaseline(ctx, mainDSN); err != nil {
		return err
	}
	if err := runMigration("file://migrations/main", mainDSN, "main"); err != nil {
		return err
	}
	return ensureAuthSchema(authDSN)
}

// reconcileMainBaseline — appdb-ийн миграцын төлөвийг baseline-тай эвлэрүүлнэ.
//
// Baseline нь ЗӨВХӨН хоосон санд ажиллах ёстой (доторх 146 ADD CONSTRAINT нь
// давталтыг тэвчихгүй). Аль хэдийн бүтэцтэй санд — хуучин version 96, эсвэл
// дампаас сэргээсэн — түүнийг ажиллуулбал ойлгомжгүй алдаанд унаад
// schema_migrations нь dirty=1 болж үлддэг: тэр үеэс апп restart бүрд
// "Dirty database version 1" гэж уначихдаг. Журам ёсоор тийм санд зөвхөн
// тамга дарна (force 1) — үүнийг энд автоматжуулав.
//
// Baseline нэг гүйлгээгээр ажилладаг тул dirty=1 нь "юу ч хэрэгжээгүй, бүгд
// буцаагдсан" гэсэн үг — иймд төлөвийг зөв рүү нь засах нь аюулгүй.
// Version 2-оос хойшхийг ХӨНДӨХГҮЙ.
func reconcileMainBaseline(ctx context.Context, dsn string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(ctx) }()

	// parcel — baseline үүсгэдэг аппын цөм хүснэгт: бүтэц байгаа эсэхийн шалгуур.
	// valuation_org_profile — хамгийн сүүлийн (000096) хуучин миграцын хүснэгт:
	// сан 96 хүртэлээ бүрэн явсан эсэхийн шалгуур.
	var hasStructure, hasLastLegacy, hasState bool
	if err := conn.QueryRow(ctx, `
		SELECT to_regclass('public.parcel') IS NOT NULL,
		       to_regclass('public.valuation_org_profile') IS NOT NULL,
		       to_regclass('public.schema_migrations') IS NOT NULL`,
	).Scan(&hasStructure, &hasLastLegacy, &hasState); err != nil {
		return err
	}

	var version int
	var dirty, recorded bool
	if hasState {
		err := conn.QueryRow(ctx, `SELECT version, dirty FROM public.schema_migrations LIMIT 1`).Scan(&version, &dirty)
		switch {
		case err == nil:
			recorded = true
		case errors.Is(err, pgx.ErrNoRows):
		default:
			return err
		}
	}

	// Хоосон санд гацсан dirty baseline — тэмдэглэгээг арилгаад дахин оролдоно.
	if recorded && dirty && version == baselineVersion && !hasStructure {
		if _, err := conn.Exec(ctx, `DELETE FROM public.schema_migrations`); err != nil {
			return err
		}
		slog.Warn("dirty baseline cleared, retrying from scratch", "db", "main")
		return nil
	}

	// Бүтэц байхгүй, эсвэл төлөв нь ЦЭВЭР бүртгэгдсэн — хэвийн зам, хөндөхгүй.
	//
	// АНХААР: өмнө нь энд `version == baselineVersion` гэж шаарддаг байсан тул
	// 1-ээс ДЭЭШ явсан (жишээ нь 40) ЦЭВЭР сан бүрийг "тамга байхгүй" гэж үзэж,
	// restart бүрд schema_migrations-ыг 1 болгож ЭГҮҮЛЭЭД 2..N-г дахин
	// ажиллуулдаг байв. Тэр давталтын дундуур (түгжээ/1 минутын хугацаа дуусах)
	// тасрахад сан `dirty` болж, апп "Dirty database version N" гэж уначихдаг.
	// Цэвэр төлөвтэй санг ямар ч хувилбарт нь ХӨНДӨХГҮЙ.
	if !hasStructure || (recorded && !dirty) {
		return nil
	}
	if recorded && dirty && version != baselineVersion {
		return nil
	}

	// Эндээс цааш: бүтэц нь байгаа мөртлөө тамга нь байхгүй/dirty.
	if !hasLastLegacy {
		return errors.New("appdb-д аппын бүтэц байгаа ч 000096-гийн valuation_org_profile алга: " +
			"сан хуучин миграцаа бүрэн дуусгаагүй байна — гараар шалгана уу")
	}

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.schema_migrations (version bigint not null primary key, dirty boolean not null);
		DELETE FROM public.schema_migrations;
		INSERT INTO public.schema_migrations (version, dirty) VALUES (1, false);`,
	); err != nil {
		return err
	}
	slog.Warn("existing appdb structure detected, baseline stamped instead of run",
		"db", "main", "version", baselineVersion)
	return nil
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
