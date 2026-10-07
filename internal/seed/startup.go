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

	if err := runSeedSteps(ctx, mainDB, []seedStep{
		{
			// Лавлахууд эхэлж — bs_person.type нь cl_person_type-аас хамаардаг.
			name: "000000_auth_lookups",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedAuthLookups(ctx)
			},
		},
		{
			// appdb-ийн лавлахууд (parcel_status г.м.) — бусад seed эдгээрийн
			// FK-д хамаардаг тул эхэнд.
			name: "000000_main_reference",
			run: func(ctx context.Context) error {
				return NewReferenceSeeder(mainDB).Seed(ctx)
			},
		},
		{
			name:    "000001_auth",
			aliases: []string{"auth_seeder"},
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).Seed(ctx, opts.Admin, opts.Employee)
			},
		},
		// 000002_main, 000003_assets_backfill, 000005_compensations УСТСАН:
		// эдгээр нь demo чөлөөлөлт, parcel/assets/compensation өгөгдөл үүсгэдэг
		// байсан. Production/startup seeder бодит чөлөөлөлтийн дата үүсгэх ёсгүй.
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
			// Үнэлгээний хүснэгт (Excel эх файл) төрөл нэмэгдсэн ба баталгаажсан
			// тайлангийн нэр өөрчлөгдсөн тул дахин синк хийнэ.
			name: "000011_document_types_valuation_source",
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
			name:     "000016_file_storage_migration",
			optional: true,
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedFileStorageMigration(
					ctx, opts.LegacyFiles, opts.Files, opts.FilesBaseURL)
			},
		},
		{
			// Файлын систем нэвтрэхээс өмнө орсон дроны ортофотуудыг зөөнө
			// (тэдгээр нь зөвхөн GeoServer-ийн мозайк хавтсанд байсан).
			name:     "000017_drone_images_to_file_storage",
			optional: true,
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedDroneImagesToFileStorage(
					ctx, opts.Files, opts.LegacyDroneDir)
			},
		},
		{
			name: "000018_hr_permissions",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedHRPermissions(ctx)
			},
		},
		// 000019_hr_backfill УСТСАН: sdplatform.sd_organization рүү default
		// байгууллага insert хийдэг байсан. Startup seeder sd_organization-д
		// дата нэмэх ёсгүй.
		{
			// 000019 нь ажилтныг department_id=NULL-тай үүсгэсэн тул "Алба,
			// хэлтэс" лавлах хоосон, "Ажилтан" дээр хэлтэс харагдахгүй байв.
			name: "000020_hr_departments",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedHRDepartments(ctx)
			},
		},
		{
			// Захирамжийн төслийн "Ажлын төрөл" / "Төсөв" сонголтын анхны
			// утгууд. Цаашид Тохиргоо цэснээс удирдагдана.
			name: "000021_decision_draft_options",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedDecisionOptions(ctx)
			},
		},
		{
			// Захирамжийн төслийн маршрутууд land:*-аас decision:* руу шилжсэн.
			// Админ роль-д decision:* бүрийг олгоно, мөн land:<action> эрхтэй
			// ролиудад харгалзах decision:<action>-ийг тусгана.
			name: "000022_decision_permissions",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedDecisionPermissions(ctx)
			},
		},
		{
			// Migration 000068-ийн plan хүснэгтэд харгалзах data backfill.
			// Төлөвлөгөөгөөр хайх/сонгох үйлдэл цаашид plan хүснэгтээс уншина
			// тул хуучин чөлөөлөлтүүдийн төлөвлөгөөг тийш зөөнө.
			name: "000023_plan_backfill",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedPlanBackfill(ctx)
			},
		},
		{
			name: "000024_system10_access_policy",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedSystem10AccessPolicy(ctx)
			},
		},
		{
			// 000025_external_valuation_users УСТСАН: үнэлгээний байгууллагын
			// ажилтанд sdplatform.sd_user мөр үүсгэдэг байсан. Тэдний нэвтрэх
			// эрх appdb-д шилжсэн (000024 миграц) — 000031-ийг үз.
			//
			// mika / finance_specialist ролиудын ЭРХ нь энэ алхмаар синк
			// болдог байсан тул түүнийг шууд дуудна (хэрэглэгч үүсгэхгүй).
			name: "000025_compensation_access_resync",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedCompensationAccess(ctx)
			},
		},
		{
			name: "000026_legacy_sdplatform_system10_seed",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedLegacySDPlatformSystem10(ctx)
			},
		},
		{
			name: "000027_system10_role_matrix",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedSystemRoles(ctx)
			},
		},
		// 000028_valuation_org_registry УСТСАН: professional_org хэрэглэгчдэд
		// sdplatform.sd_organization дээр байгууллага автоматаар үүсгэдэг байсан.
		{
			// ensureAction нь "structure:create" мэт цэгтэй үйлдлийг олж чаддаггүй
			// байсан тул seed алхам бүр шинэ sd_permission мөр үүсгэж, шинэ сан
			// дээр 5 хувь давхардаж байв (мөн sd_form_permission /
			// sd_role_permission олширсон). Шалтгаан нь ensureAction дотор
			// зассан — энэ алхам үүссэн давхардлыг нэгтгэнэ.
			name: "000029_permission_dedupe",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).
					SeedPermissionDedupe(ctx)
			},
		},
		{
			name: "000030_acquisition_categories",
			run: func(ctx context.Context) error {
				return NewReferenceSeeder(mainDB).SeedAcquisitionCategories(ctx)
			},
		},
		{
			// Үнэлгээний байгууллага + нэвтрэх эрхтэй ажилтан, БҮГД appdb-д.
			// Зөвхөн valuation_org_employee хоосон үед ажиллана.
			name: "000031_valuation_org_employees",
			run: func(ctx context.Context) error {
				return SeedValuationOrgEmployees(ctx, mainDB)
			},
		},
		{
			// codelists.cl_person_type-ийн код 10..100-г дахин синк хийнэ.
			// 000000_auth_lookups нь эрт орчинд ажиллаж дуусаад seed_history-д
			// бүртгэгдсэн тул шинэ кодыг аваагүй authdb-нүүд үлдсэн: тэдгээрт
			// bs_person."type"=10 (иргэн) бичихэд FK зөрчлөөр (ErrInvalidInput)
			// хүн үүсгэх боломжгүй байв. seedPersonTypes нь код/тайлбар давхцвал
			// огт хөндөхгүй тул дахин ажиллуулах аюулгүй.
			name: "000032_person_types_resync",
			run: func(ctx context.Context) error {
				return NewSeeder(authDB, opts.UserRepo, opts.RoleRepo, opts.PermRepo).SeedAuthLookups(ctx)
			},
		},
		{
			// "Ажлын зураг" (work_photo) төрөл нэмэгдсэн: үнэлгээ илгээхэд
			// тайлантай хамт заавал хавсаргана. Мөн valuation_report-ийн нэр
			// "Үнэлгээний тайлан" болж өөрчлөгдсөн (илгээх шатанд хавсаргадаг
			// тул "баталгаажсан" нэр урсгалтай зөрчилдөж байв).
			name: "000036_document_types_work_photo_and_report_rename",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedDocumentTypes(ctx)
			},
		},
		{
			// Migration 000039-ийн parcel_valuation_snapshot.files-д харгалзах
			// өгөгдлийн засвар: ӨМНӨ цуцлагдсан үнэлгээний файлуудыг тухайн
			// snapshot-д холбож, илгээлтийн мөрөнд үлдсэн хуучин тайланг
			// цэвэрлэнэ (эс бөгөөс шинэ үнэлгээн дээр харагдсаар байна).
			// (000033-ыг устсан valuation_login алхам эзэлсэн тул 000034.)
			name: "000034_cancelled_valuation_files",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedCancelledValuationFiles(ctx)
			},
		},
		{
			// Migration 000047: шилжилтэд ЗААВАЛ байх хавсралт. Өмнө кодонд
			// хатуу шаарддаг байсныг (… → Чөлөөлсөн: Үнэлгээний тайлан,
			// … → Татгалзсан: баримт) өгөгдмөл тохиргоо болгоно.
			name: "000047_parcel_workflow_required_documents",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedParcelWorkflowRequiredDocuments(ctx)
			},
		},
		{
			// Migration 000048: «Нөхөх олговор олгосон баримт» төрөл (олголтын
			// шат 60 / 40 / бүрэн — нэгж талбарын олголтын гүйцэтгэлд тусна).
			name: "000048_document_types_compensation_paid_receipt",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).seedDocumentTypes(ctx)
			},
		},
	}); err != nil {
		return err
	}

	return nil
}
