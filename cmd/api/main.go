// @title           Go API Template
// @version         1.0
// @description     Production-ready Go REST API — Clean Architecture, JWT Auth, RBAC.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  admin@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/go-api-template/internal/config"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/pkg/email"
	"github.com/your-org/go-api-template/internal/pkg/filestorage"
	jwtpkg "github.com/your-org/go-api-template/internal/pkg/jwt"
	"github.com/your-org/go-api-template/internal/repository/postgres"
	"github.com/your-org/go-api-template/internal/service"

	_ "github.com/your-org/go-api-template/docs"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Үндсэн DB — бизнесийн мэдээлэл
	mainDB, err := connectDB(ctx, cfg.DB.DSN())
	if err != nil {
		slog.Error("failed to connect to main database", "error", err)
		os.Exit(1)
	}
	defer mainDB.Close()

	// Auth DB — хэрэглэгч, эрх, нэвтрэх мэдээлэл
	authDB, err := connectDB(ctx, cfg.AuthDB.DSN())
	if err != nil {
		slog.Error("failed to connect to auth database", "error", err)
		os.Exit(1)
	}
	defer authDB.Close()

	if err := runMigrations(cfg.DB.MigrateURL(), cfg.AuthDB.MigrateURL()); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	// Auth repo-ууд → authDB
	userRepo := postgres.NewUserRepo(authDB)
	roleRepo := postgres.NewRoleRepo(authDB)
	permRepo := postgres.NewPermissionRepo(authDB)
	pwdResetRepo := postgres.NewPasswordResetRepo(authDB)

	// Auth DB-аас план унших, main DB-д чөлөөлөлт бичих
	planRepo := postgres.NewPlanRepo(authDB)
	landAcqRepo := postgres.NewLandAcquisitionRepo(mainDB)

	jwtManager := jwtpkg.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	var emailSender email.Sender
	if cfg.Email.Enabled() {
		emailSender = email.NewSMTPSender(
			cfg.Email.Host, cfg.Email.Port,
			cfg.Email.Username, cfg.Email.Password,
			cfg.Email.From,
		)
	} else {
		emailSender = email.NoopSender{}
	}

	authSvc := service.NewAuthService(userRepo, pwdResetRepo, jwtManager, emailSender, cfg.Email.ResetURL)
	userSvc := service.NewUserService(userRepo)
	roleSvc := service.NewRoleService(roleRepo)
	permSvc := service.NewPermissionService(permRepo)
	fileStorage := filestorage.NewMockStorage(cfg.FileServer.BaseURL, cfg.FileServer.LocalDir)
	landSvc := service.NewLandService(planRepo, landAcqRepo, fileStorage)
	planSvc := service.NewPlanService(planRepo)

	if err := ensureSeedHistory(ctx, mainDB); err != nil {
		slog.Error("failed to create seed_history", "error", err)
		os.Exit(1)
	}

	// Auth seeder — хэрэглэгч, эрх, роль
	if applied, _ := isSeedApplied(ctx, mainDB, "auth_seeder"); !applied {
		seeder := NewSeeder(authDB, userRepo, roleRepo, permRepo)
		if err := seeder.Seed(ctx, cfg.Admin); err != nil {
			slog.Warn("auth seed failed", "error", err)
		} else if err := markSeedApplied(ctx, mainDB, "auth_seeder"); err != nil {
			slog.Warn("failed to mark auth_seeder", "error", err)
		}
	} else {
		slog.Info("seed skipped (already applied)", "name", "auth_seeder")
	}

	// Main seeder — аймаг, сум, баг, газар чөлөөлөлт, нэгж талбар, барилга
	if applied, _ := isSeedApplied(ctx, mainDB, "main_seeder"); !applied {
		mainSeeder := NewMainSeeder(mainDB)
		if err := mainSeeder.Seed(ctx); err != nil {
			slog.Warn("main seed failed", "error", err)
		} else if err := markSeedApplied(ctx, mainDB, "main_seeder"); err != nil {
			slog.Warn("failed to mark main_seeder", "error", err)
		}
	} else {
		slog.Info("seed skipped (already applied)", "name", "main_seeder")
	}

	// Buildings seeder — өмнө main_seeder ажилласан орчинд барилгын жишээ өгөгдөл нөхөж оруулна
	if applied, _ := isSeedApplied(ctx, mainDB, "buildings_seeder"); !applied {
		mainSeeder := NewMainSeeder(mainDB)
		if err := mainSeeder.seedDefaultBuildings(ctx); err != nil {
			slog.Warn("buildings seed failed", "error", err)
		} else if err := markSeedApplied(ctx, mainDB, "buildings_seeder"); err != nil {
			slog.Warn("failed to mark buildings_seeder", "error", err)
		}
	} else {
		slog.Info("seed skipped (already applied)", "name", "buildings_seeder")
	}

	// Plans seeder — auth DB-д төлөвлөгөө, эх нэгж талбар
	if applied, _ := isSeedApplied(ctx, mainDB, "plans_seeder"); !applied {
		plansSeeder := NewPlansSeeder(authDB)
		if err := plansSeeder.Seed(ctx); err != nil {
			slog.Warn("plans seed failed", "error", err)
		} else if err := markSeedApplied(ctx, mainDB, "plans_seeder"); err != nil {
			slog.Warn("failed to mark plans_seeder", "error", err)
		}
	} else {
		slog.Info("seed skipped (already applied)", "name", "plans_seeder")
	}

	router := handler.NewRouter(mainDB, jwtManager, authSvc, userSvc, roleSvc, permSvc, landSvc, planSvc)

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting",
			"port", cfg.App.Port,
			"swagger", "http://localhost:"+cfg.App.Port+"/swagger/index.html",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}

func connectDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	var db *pgxpool.Pool
	var err error
	for i := 0; i < 10; i++ {
		db, err = pgxpool.New(ctx, dsn)
		if err == nil {
			if pingErr := db.Ping(ctx); pingErr == nil {
				slog.Info("database connected")
				return db, nil
			}
		}
		slog.Info("waiting for database...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}
