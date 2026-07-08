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

	"acquisition_backend/internal/config"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/pkg/email"
	"acquisition_backend/internal/pkg/filestorage"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/repository/postgres"
	"acquisition_backend/internal/seed"
	"acquisition_backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "acquisition_backend/docs"
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
	revokedTokenRepo := postgres.NewRevokedTokenRepo(authDB)

	// Auth DB-аас план унших, main DB-д чөлөөлөлт бичих
	planRepo := postgres.NewPlanRepo(authDB)
	landAcqRepo := postgres.NewLandAcquisitionRepo(mainDB)
	acquisitionProgressStatusRepo := postgres.NewAcquisitionProgressStatusRepo(mainDB)
	parcelStatusRepo := postgres.NewParcelStatusRepo(mainDB)
	documentTypeRepo := postgres.NewDocumentTypeRepo(mainDB)

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

	authSvc := service.NewAuthService(userRepo, pwdResetRepo, jwtManager, emailSender, cfg.Email.ResetURL, revokedTokenRepo)
	// Нууц үг сэргээх зэргээр хүчингүй болсон токеныг хүсэлт бүрт шалгана.
	jwtManager.SetSessionValidator(userRepo)
	userSvc := service.NewUserService(userRepo)
	roleSvc := service.NewRoleService(roleRepo)
	permSvc := service.NewPermissionService(permRepo)
	fileStorage := filestorage.NewMockStorage(cfg.FileServer.BaseURL, cfg.FileServer.LocalDir)
	landSvc := service.NewLandService(planRepo, landAcqRepo, fileStorage, userRepo)
	planSvc := service.NewPlanService(planRepo)
	acquisitionProgressStatusSvc := service.NewAcquisitionProgressStatusService(acquisitionProgressStatusRepo)
	parcelStatusSvc := service.NewParcelStatusService(parcelStatusRepo)
	documentTypeSvc := service.NewDocumentTypeService(documentTypeRepo)

	if err := seed.RunStartup(ctx, mainDB, authDB, seed.StartupOptions{
		Admin:           cfg.Admin,
		Employee:        cfg.Employee,
		Finance:         cfg.Finance,
		ProfessionalOrg: cfg.ProfessionalOrg,
		UserRepo:        userRepo,
		RoleRepo:        roleRepo,
		PermRepo:        permRepo,
	}); err != nil {
		slog.Error("failed to create seed_history", "error", err)
		os.Exit(1)
	}

	router := handler.NewRouter(mainDB, jwtManager, authSvc, userSvc, roleSvc, permSvc, landSvc, planSvc, acquisitionProgressStatusSvc, parcelStatusSvc, documentTypeSvc)

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
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Dashboard зэрэг handler-ууд нэг хүсэлтэд хэд хэдэн асуулгыг зэрэг ажиллуулдаг
	// тул pgx-ийн анхдагч MaxConns (≈CPU тоо) хурдан дүүрч, хүсэлтүүд холболт
	// хүлээж түгжирдэг. Pool-ийг өргөн байлгаж түгжрэлээс сэргийлнэ.
	cfg.MaxConns = 30
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = time.Hour

	var db *pgxpool.Pool
	for i := 0; i < 10; i++ {
		db, err = pgxpool.NewWithConfig(ctx, cfg)
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
