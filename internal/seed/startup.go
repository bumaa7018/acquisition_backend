package seed

import (
	"context"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StartupOptions struct {
	Admin           config.AdminConfig
	Employee        config.EmployeeConfig
	Finance         config.FinanceConfig
	ProfessionalOrg config.ProfessionalOrgConfig

	UserRepo repository.UserRepository
	RoleRepo repository.RoleRepository
	PermRepo repository.PermissionRepository

	// Файлын системд шилжүүлэх алхамд хэрэглэгдэнэ. Legacy нь хуучин локал
	// хадгалалт, Files нь шинэ (MinIO). Хоёулаа nil байж болно — тэр үед
	// шилжүүлэлт алгасагдана.
	LegacyFiles  filestorage.Storage
	Files        filestorage.Storage
	FilesBaseURL string
	// LegacyDroneDir — ХУУЧИН мозайк хавтас (LEGACY_DRONE_DIR). Файлын систем нэвтрэхээс өмнө
	// орсон дроны зургууд ЗӨВХӨН тэнд байгаа тул тэндээс зөөнө.
	LegacyDroneDir string
}

func RunStartup(ctx context.Context, mainDB, authDB *pgxpool.Pool, opts StartupOptions) error {
	if err := ensureSeedHistory(ctx, mainDB); err != nil {
		return err
	}

	runSeedSteps(ctx, mainDB, []seedStep{
		{
			name:    "000001_auth",
			aliases: []string{"auth_seeder"},
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).Seed(ctx, opts.Admin, opts.Employee)
			},
		},
		{
			name:    "000002_main",
			aliases: []string{"main_seeder"},
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).Seed(ctx)
			},
		},
		{
			name:    "000003_assets_backfill",
			aliases: []string{"assets_seeder", "buildings_seeder"},
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedDefaultAssets(ctx)
			},
		},
		{
			name:    "000004_plans",
			aliases: []string{"plans_seeder"},
			run: func(ctx context.Context) error {
				return NewPlansSeeder(authDB).Seed(ctx)
			},
		},
		{
			name:    "000005_compensations",
			aliases: []string{"compensation_seeder"},
			run: func(ctx context.Context) error {
				acqID, err := findSeedLandAcquisitionID(ctx, mainDB)
				if err != nil {
					return seedError("land_acquisition LA-2024-001 not found", err)
				}
				return NewMainSeeder(mainDB).seedCompensations(ctx, acqID)
			},
		},
		{
			name: "000006_document_types",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedDocumentTypes(ctx)
			},
		},
		{
			name: "000007_auth_access_users",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).Seed(ctx, opts.Admin, opts.Employee)
			},
		},
		{
			name: "000008_compensation_access",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedCompensationAccess(ctx)
			},
		},
		{
			// Хавсралтын төрлүүдийг дахин синк хийнэ (valuation_report нэмэгдсэн).
			// seedDocumentTypes нь ON CONFLICT DO NOTHING тул давхардахгүй.
			name: "000009_document_types_resync",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedDocumentTypes(ctx)
			},
		},
		{
			// Migration 000048-ийн schema-д харгалзах data backfill (зөвхөн seeder-ээр).
			name: "000010_selected_valuation_backfill",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedSelectedValuationBackfill(ctx)
			},
		},
		{
			// Migration 000050-ийн өмнөх lad.file_type утгыг document.document_type_id руу.
			name: "000011_document_type_backfill",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedDocumentTypeBackfill(ctx)
			},
		},
		{
			// Эрхийн нэршлийн зөрчлүүд: admin:*, users:read→senior, орфон цэвэрлэгээ.
			name: "000012_permission_fixes",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedPermissionFixes(ctx)
			},
		},
		{
			// Санхүүгийн мэргэжилтэн, мэргэжлийн байгууллагын анхны хэрэглэгчид
			// (роль нь 000008_compensation_access-д үүссэн байх ёстой).
			name: "000013_external_users",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedExternalUsers(ctx, opts.Finance, opts.ProfessionalOrg)
			},
		},
		{
			name: "000014_audit_permission",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedAuditPermission(ctx)
			},
		},
		{
			// admin роль-д БҮХ эрхийг олгоно. compensation:* нь admin роль
			// үүссэний дараа нэмэгдсэн тул түүнд олгогдоогүй үлдэж, эрх
			// нэмэгдүүлэлтийн шалгалт админд гадаад роль олгохыг хориглож байсан.
			name: "000015_admin_all_permissions",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedAdminAllPermissions(ctx)
			},
		},
		{
			// Хуучин ЛОКАЛ файлуудыг файлын систем (MinIO) руу зөөж, DB дахь
			// URL-уудыг шинэчилнэ. API нь файлыг өөр дээрээ хадгалахаа больсон.
			name: "000016_file_storage_migration",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedFileStorageMigration(
					ctx, opts.LegacyFiles, opts.Files, opts.FilesBaseURL)
			},
		},
		{
			// Файлын систем нэвтрэхээс өмнө орсон дроны ортофотуудыг зөөнө
			// (тэдгээр нь зөвхөн GeoServer-ийн мозайк хавтсанд байсан).
			name: "000017_drone_images_to_file_storage",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedDroneImagesToFileStorage(
					ctx, opts.Files, opts.LegacyDroneDir)
			},
		},
	})

	return nil
}
