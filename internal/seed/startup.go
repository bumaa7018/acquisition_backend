package seed

import (
	"context"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StartupOptions struct {
	Admin    config.AdminConfig
	Employee config.EmployeeConfig

	UserRepo repository.UserRepository
	RoleRepo repository.RoleRepository
	PermRepo repository.PermissionRepository
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
	})

	return nil
}
