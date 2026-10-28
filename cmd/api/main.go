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
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"acquisition_backend/internal/config"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/legacyimport"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/email"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/pkg/geoserver"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/pkg/logstack"
	"acquisition_backend/internal/pkg/notify"
	"acquisition_backend/internal/repository/postgres"
	"acquisition_backend/internal/seed"
	"acquisition_backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "acquisition_backend/docs"
)

func main() {
	// -seed-only — миграц + startup seeder-ийг ажиллуулаад HTTP сервер
	// АСААХГҮЙГЭЭР гарна (`make seed`). Сан бэлдэх ажлыг серверийн
	// амьдралын мөчлөгөөс салгаж, скриптээс дуудахад тохиромжтой.
	seedOnly := flag.Bool("seed-only", false, "migration + seed ажиллуулаад гарах (HTTP сервер асаахгүй)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// LOG_LEVEL-ийг .env-ээс уншсаны дараа лог түвшинг бодит утгаар нь дахин тохируулна.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.ParseLogLevel(cfg.App.LogLevel),
	})))

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

	if err := runMigrations(cfg.DB.MigrateURL(), cfg.AuthDB.DSN()); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	// Auth repo-ууд → authDB
	// sdplatform DB-д (authDB) бид ЮУ Ч нэмэхгүй — аппын өөрийн төлөв mainDB-д.
	userRepo := postgres.NewUserRepoFor("auth", authDB, mainDB)
	roleRepo := postgres.NewRoleRepo(authDB)
	menuRepo := postgres.NewMenuRepo(authDB)
	permRepo := postgres.NewPermissionRepo(authDB)
	pwdResetRepo := postgres.NewPasswordResetRepo(mainDB)
	revokedTokenRepo := postgres.NewRevokedTokenRepo(mainDB)
	hrRepo := postgres.NewHRRepo(authDB)

	// Төлөвлөгөө: ЗӨВХӨН appdb-ийн plan хүснэгтэд, зөвхөн дундын сервисээс
	// татсан мэдээллээр. Уншилт ч эндээс явна — authdb-д plan хайхгүй.
	planStoreRepo := postgres.NewPlanStoreRepo(mainDB)
	landAcqRepo := postgres.NewLandAcquisitionRepo(mainDB)
	parcelStatusRepo := postgres.NewParcelStatusRepo(mainDB)
	documentTypeRepo := postgres.NewDocumentTypeRepo(mainDB)
	decisionDraftRepo := postgres.NewDecisionDraftRepo(mainDB)
	decisionWorkTypeRepo := postgres.NewDecisionWorkTypeRepo(mainDB)
	decisionBudgetRepo := postgres.NewDecisionBudgetRepo(mainDB)
	decisionReviewUnitRepo := postgres.NewDecisionReviewUnitRepo(mainDB)
	// Системийн тохиргоо (app_setting) — одоогоор газрын зургийн суурь зураг.
	settingRepo := postgres.NewSettingRepo(mainDB)

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

	// Үнэлгээний байгууллагын identity/profile нь mainDB-ийн app-owned
	// valuation_org_profile хүснэгтэд байна. sdplatform-ийн ердийн HR лавлах
	// энэ бүртгэлийн эх сурвалж биш.
	// Байгууллага, ажилтан БОЛОН тэдний нэвтрэх эрх БҮГД mainDB-д — эдгээр
	// хэрэглэгч sdplatform-д бүртгэгддэггүй (000024 миграцыг үз).
	valuationOrgRepo := postgres.NewValuationOrgRepo(mainDB)
	// Хоёр дахь нэвтрэлтийн эх сурвалж: appdb дээрх үнэлгээний байгууллагын
	// ажилтан. Токенд `src` тэмдэг бичигдэж, ID мөргөлдөхөөс хамгаална.
	valuationOrgLoginRepo := postgres.NewValuationOrgLoginRepo(mainDB)
	authSvc := service.NewAuthService(userRepo, pwdResetRepo, jwtManager, emailSender, cfg.Email.ResetURL, revokedTokenRepo, valuationOrgLoginRepo)
	// Нууц үг сэргээх зэргээр хүчингүй болсон токеныг хүсэлт бүрт шалгана.
	jwtManager.SetSessionValidator(userRepo)
	// valuationOrgLoginRepo — `/users/me` нь appdb дээрх хэрэглэгчийг ч
	// шийдэх ёстой (эс бөгөөс 404 эсвэл өөр хүний бүртгэл буцна).
	userSvc := service.NewUserService(userRepo, hrRepo, valuationOrgLoginRepo)
	roleSvc := service.NewRoleService(roleRepo)
	menuSvc := service.NewMenuService(menuRepo, roleRepo)
	permSvc := service.NewPermissionService(permRepo)
	organizationSvc := service.NewOrganizationService(hrRepo)
	departmentSvc := service.NewDepartmentService(hrRepo)
	positionSvc := service.NewPositionService(hrRepo)
	personSvc := service.NewPersonService(hrRepo)
	employeeSvc := service.NewEmployeeService(hrRepo)
	valuationOrgSvc := service.NewValuationOrgService(valuationOrgRepo)
	// Файлын систем — системд оруулсан БҮХ файл (баримт, зураг, тайлан, дроны
	// ортофото) MinIO/S3 дээр хадгалагдана. API нь өөр дээрээ файл хадгалахгүй,
	// зөвхөн object-ийн замыг DB-д бичнэ.
	fileStorage, s3Storage, err := buildFileStorage(ctx, cfg.FileServer)
	if err != nil {
		slog.Error("file storage init failed", "error", err)
		os.Exit(1)
	}

	// Мэдэгдэл: main DB-д хадгалж, in-memory hub-аар SSE стрийм рүү түгээнэ.
	notificationRepo := postgres.NewNotificationRepo(mainDB)
	notificationHub := notify.NewHub()
	notificationSvc := service.NewNotificationService(notificationRepo, notificationHub)
	auditRepo := postgres.NewAuditLogRepo(mainDB)
	auditSvc := service.NewAuditService(auditRepo)
	clientRequestRepo := postgres.NewClientRequestRepo(mainDB)
	clientRequestSvc := service.NewClientRequestService(clientRequestRepo)

	// Дроны ортофото — зураг тус бүр GeoServer дээр өөрийн COG давхаргатай.
	// GeoServer растрыг ФАЙЛЫН СИСТЕМЭЭС ШУУД (HTTP Range) уншина: backend
	// растрыг өөр дээрээ ХАДГАЛАХГҮЙ, хуваалцсан хавтас ч шаардахгүй.
	droneStorage := service.DroneStorage{
		Publisher: geoserver.NewClient(
			cfg.GeoServer.URL,
			cfg.GeoServer.User,
			cfg.GeoServer.Password,
			cfg.GeoServer.Workspace,
		),
	}

	middlewareBaseURL := strings.TrimRight(cfg.Middleware.BaseURL, "/")
	parcelInfoClient := service.NewHTTPParcelInfoClient(middlewareBaseURL, service.MiddlewareCredentials{
		Username: cfg.Middleware.Username,
		Password: cfg.Middleware.Password,
	})

	// УБЕГ-ийн өмчлөлийн бүртгэл — ХУР (XYP) гарц. ГУС-ийн дундын сервисээс
	// ТУСДАА хаягтай тул өөрийн клиенттэй. Тохируулаагүй бол клиент үүсгэхгүй:
	// өмчлөлийн мэдээлэл татах үйлдэл "холбогдоогүй" гэж буцна.
	var ownershipInfoClient service.OwnershipInfoClient
	if xypBaseURL := strings.TrimRight(cfg.XYP.BaseURL, "/"); xypBaseURL != "" {
		ownershipInfoClient = service.NewHTTPOwnershipInfoClient(xypBaseURL)
		slog.Info("XYP ownership gateway ready", "base_url", xypBaseURL)
	} else {
		slog.Warn("XYP тохируулаагүй — УБЕГ-ийн өмчлөлийн мэдээлэл татагдахгүй (XYP_BASE_URL)")
	}

	// ГАДНЫ эх системийн файл сервер — татсан хавсралтын ФАЙЛ тэнд байдаг.
	// ЗӨВХӨН УНШИНА (системийн үндсэн файл сервер нь MinIO — fileStorage).
	//
	// FTP (ip + нэвтрэх нэр/код) нь тэргүүн эрхтэй; ftp_id-д FTP хост
	// тохируулаагүй бол HTTP руу унана. Тохируулаагүй бол хавсралт
	// татах/бүртгэх нь ажиллах ч файл нээх үед ErrSourceFileNotConfigured.
	var sourceFileClient service.SourceFileClient
	if target, ok := cfg.SourceFile.FTPTarget(); ok {
		sourceTarget := service.SourceFTPTarget{
			FtpID:       target.FtpID,
			Addr:        target.Addr,
			Username:    target.Username,
			Password:    target.Password,
			RootDir:     target.RootDir,
			TLS:         target.Protocol == config.ProtocolFTPS,
			ImplicitTLS: target.ImplicitTLS,
		}
		// Протокол нь тогтмол порт/ssl-ээс тооцогдоно (config.FTPTarget).
		if target.Protocol == config.ProtocolSFTP {
			sourceFileClient = service.NewSFTPSourceFileClient(sourceTarget, target.Timeout)
		} else {
			sourceFileClient = service.NewFTPSourceFileClient(sourceTarget, target.Timeout)
		}
		slog.Info("source file server ready",
			"protocol", target.Protocol, "addr", target.Addr,
			"ftp_id", target.FtpID, "user", target.Username, "root_dir", target.RootDir)
	} else {
		slog.Warn("source file server тохируулаагүй — татсан хавсралтын файл нээгдэхгүй " +
			"(SOURCE_FTP_HOST)")
	}

	// E-Mongolia-ийн мэдэгдлийн гарц (2.8.5). Бодит холболтын заавар гараагүй
	// тул одоогоор МОК: илгээлт логлогдож, түүхэд status="mocked" гэж бичигдэнэ.
	emongoliaClient := service.NewMockEMongoliaClient()
	slog.Warn("E-Mongolia гарц МОК режимд — мэдэгдэл гадаад систем рүү илгээгдэхгүй")

	landSvc := service.NewLandService(landAcqRepo, fileStorage, userRepo, notificationSvc, auditSvc, parcelInfoClient, droneStorage, planStoreRepo, sourceFileClient, ownershipInfoClient, emailSender, emongoliaClient)
	planSvc := service.NewPlanService(planStoreRepo, parcelInfoClient)
	parcelStatusSvc := service.NewParcelStatusService(parcelStatusRepo)
	settingSvc := service.NewSettingService(settingRepo)
	parcelSurveySvc := service.NewParcelSurveyService(postgres.NewParcelSurveyRepo(mainDB), auditSvc)

	// ГАДААД системийн API-ийн түлхүүрүүд (.env: EXTERNAL_API_KEYS).
	// Хоосон бол /api/v1/external/* гарцууд ХААЛТТАЙ.
	externalAPIKeys := []middleware.ExternalAPIKey{}
	for _, k := range cfg.ExternalAPI.ParsedKeys() {
		externalAPIKeys = append(externalAPIKeys, middleware.ExternalAPIKey{Label: k.Label, Key: k.Key})
	}
	if len(externalAPIKeys) == 0 {
		slog.Warn("EXTERNAL_API_KEYS тохируулаагүй — гадаад системийн API хаалттай")
	} else {
		labels := make([]string, 0, len(externalAPIKeys))
		for _, k := range externalAPIKeys {
			labels = append(labels, k.Label)
		}
		slog.Info("external API ready", "clients", strings.Join(labels, ","))
	}
	documentTypeSvc := service.NewDocumentTypeService(documentTypeRepo)
	// Ажлын төрөл / Төсөв нь ижил интерфейстэй тул router-ийн type switch
	// тэдгээрийг ялгаж чадахгүй — нэг багцаар дамжуулна.
	decisionSvcs := service.DecisionServices{
		Draft:      service.NewDecisionDraftService(decisionDraftRepo),
		WorkType:   service.NewDecisionOptionService(decisionWorkTypeRepo),
		Budget:     service.NewDecisionOptionService(decisionBudgetRepo),
		ReviewUnit: service.NewDecisionOptionService(decisionReviewUnitRepo),
	}

	if err := seed.RunStartup(ctx, mainDB, authDB, seed.StartupOptions{
		Admin:   cfg.Admin,
		Finance: cfg.Finance,
		// Хуучин локал файлуудыг файлын систем руу зөөх алхамд
		LegacyFiles:  filestorage.NewMockStorage(legacyFileBaseURL(cfg), cfg.FileServer.LocalDir),
		Files:        s3Storage,
		FilesBaseURL: cfg.FileServer.BaseURL,
		// Зөвхөн ХУУЧИН (мозайк хавтсанд байсан) зургийг зөөх шаардлагатай үед
		LegacyDroneDir: os.Getenv("LEGACY_DRONE_DIR"),
	}); err != nil {
		slog.Error("startup seed failed", "error", err)
		os.Exit(1)
	}

	if *seedOnly {
		slog.Info("seed-only горим: migration + seed дууслаа, сервер асаахгүй")
		return
	}

	// Хуучин мэдээллийн импорт — ГУС руу цуварсан, 300 мс завсарлагатай (legacyimport.DefaultGusDelay).
	legacyImportSvc := legacyimport.NewService(mainDB, legacyimport.GusConfig{
		BaseURL:  middlewareBaseURL,
		Username: cfg.Middleware.Username,
		Password: cfg.Middleware.Password,
	}).WithCleanup(fileStorage, droneStorage) // «Хасах» үед файл ба дроны давхарга

	router := handler.NewRouter(mainDB, jwtManager, authSvc, userSvc, roleSvc, permSvc, landSvc, planSvc, parcelStatusSvc, documentTypeSvc, notificationSvc, auditSvc, clientRequestSvc, organizationSvc, departmentSvc, positionSvc, personSvc, employeeSvc, valuationOrgSvc, decisionSvcs, menuSvc, settingSvc, parcelSurveySvc, legacyImportSvc, externalAPIKeys)

	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: router,
		// SSE стрийм (мэдэгдэл) урт наслах холболт тул WriteTimeout тавихгүй —
		// тэгэхгүй бол 15 секунд тутам стрийм тасарна. Удаан клиентээс
		// хамгаалалтыг ReadHeaderTimeout + IdleTimeout хангана.
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Лог хянах stack (Loki+Promtail+Grafana) ажиллаж байгаа эсэхийг
	// шалгаж, шаардлагатай бол дэвсгэрт нь автоматаар асаана. API-ийн
	// асаалтыг хэзээ ч зогсоохгүй (docker байхгүй орчинд чимээгүй алгасна).
	go logstack.EnsureRunning(ctx, ".")

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

// buildFileStorage — тохиргооноос файлын системийн adapter бэлдэнэ.
//
// s3 (анхдагч): MinIO/S3 — API файлыг өөр дээрээ ХАДГАЛАХГҮЙ.
// mock: хуучин локал хадгалалт — зөвхөн шилжилт/тестэд.
//
// Хоёр дахь буцаах утга нь S3 adapter (nil байж болно) — шилжүүлэлтийн seeder
// алхамд bucket бэлдэх зэрэгт хэрэглэгдэнэ.
func buildFileStorage(ctx context.Context, cfg config.FileServerConfig) (filestorage.Storage, *filestorage.S3Storage, error) {
	if strings.EqualFold(cfg.Mode, "mock") {
		slog.Warn("file storage: локал mock режим — production-д S3 хэрэглэнэ")
		return filestorage.NewMockStorage(cfg.BaseURL, cfg.LocalDir), nil, nil
	}

	s3, err := filestorage.NewS3Storage(filestorage.S3Options{
		Endpoint:          cfg.S3Endpoint,
		GeoServerEndpoint: cfg.S3GeoServerEndpoint,
		AccessKey:         cfg.S3AccessKey,
		SecretKey:         cfg.S3SecretKey,
		Bucket:            cfg.S3Bucket,
		BaseURL:           cfg.BaseURL,
		UseSSL:            cfg.S3UseSSL,
	})
	if err != nil {
		return nil, nil, err
	}
	// Bucket-ийг эхлэхэд нь бэлдэнэ (идемпотент) — эс тэгвээс эхний upload унана.
	if err := s3.EnsureBucket(ctx); err != nil {
		return nil, nil, err
	}
	slog.Info("file storage ready", "mode", "s3",
		"endpoint", cfg.S3Endpoint, "bucket", cfg.S3Bucket, "base_url", cfg.BaseURL,
		"geoserver_endpoint", cfg.S3GeoServerEndpoint)

	return s3, s3, nil
}

// legacyFileBaseURL — хуучин локал хадгалалтын URL угтвар.
//
// Шилжүүлэлт нь DB дахь ХУУЧИН URL-уудыг (жишээ:
// http://localhost:8080/mock-files/...) локал зам болгож уншина. Тохиргоо
// аль хэдийн шинэ (/api/files) болсон тул угтварыг тусад нь тодорхойлно.
func legacyFileBaseURL(cfg *config.Config) string {
	if v := os.Getenv("LEGACY_FILE_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:" + cfg.App.Port + "/mock-files"
}
