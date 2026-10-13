package seed

import (
	"context"
	"fmt"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/pkg/filestorage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StartupOptions struct {
	// Admin, Finance — ЗӨВХӨН хоосон authdb (sd_user мөргүй) дээр үүсэх анхны
	// админ ба санхүүгийн мэргэжилтэн.
	Admin   config.AdminConfig
	Finance config.FinanceConfig

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
	// authdb (бусад системтэй хуваалцсан sdplatform сан) — startup БҮРТ
	// шаардлагатай мөр байгаа эсэхийг шалгаж, дутууг л нэмнэ. EnsureAuthDB-г үз.
	//
	// Өмнө нь authdb-ийн 17 нэг удаагийн алхам (000000_auth_lookups,
	// 000001_auth, 000007, 000008, 000012–000015, 000018, 000020, 000022,
	// 000024–000027, 000029, 000032) appdb-ийн seed_history-д бүртгэгддэг байв.
	// Тэдгээр нь давхардсан, өөр системийн мөрийг өөрчилдөг, жишээ хэрэглэгч
	// нэмдэг байсан тул бүгдийг энэ нэг шалгалтаар сольсон.
	if err := NewSeeder(authDB).EnsureAuthDB(ctx, opts.Admin, opts.Finance); err != nil {
		return fmt.Errorf("ensure authdb: %w", err)
	}

	if err := ensureSeedHistory(ctx, mainDB); err != nil {
		return err
	}

	// Доорх алхмууд ЗӨВХӨН аппын өөрийн сан (appdb)-д ажиллана.
	if err := runSeedSteps(ctx, mainDB, []seedStep{
		{
			// appdb-ийн лавлахууд (parcel_status г.м.) — бусад seed эдгээрийн
			// FK-д хамаардаг тул эхэнд.
			name: "000000_main_reference",
			run: func(ctx context.Context) error {
				return NewReferenceSeeder(mainDB).Seed(ctx)
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
			// Захирамжийн төслийн "Ажлын төрөл" / "Төсөв" сонголтын анхны
			// утгууд. Цаашид Тохиргоо цэснээс удирдагдана.
			name: "000021_decision_draft_options",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedDecisionOptions(ctx)
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
		{
			// Migration 000049: нэгж талбарын төлөвийн дүрэм (эцсийн, явц
			// түгжих, үнэлгээний шат, чөлөөлөгдсөн, шалтгаан заавал) — өмнөх
			// хатуу кодын дүрмийг өгөгдмөл тохиргоо болгоно.
			name: "000049_parcel_status_flags",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedParcelStatusFlags(ctx)
			},
		},
		{
			// Migration 000050: «Хэнд» чөлөөт текстийг «Төсөл хянах нэгж» болгоно.
			name: "000050_decision_review_units",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedDecisionReviewUnitsFromRecipients(ctx)
			},
		},
		{
			// Migration 000053: нэгж талбарын төлөвийн id = code (0–8 → 101–109,
			// нэгж талбар/түүх/ажлын урсгалтай нь). Төлөв, ажлын урсгал,
			// шилжилтийн заавал хавсралтыг 103.87.255.222/appdb2-тэй тэнцүүлнэ.
			// 000051-ээс ӨМНӨ: тэр нь анхны төлөвийг шинэ id-аар (101) хайдаг —
			// хараахан ажиллаагүй сан дээр 101 үүсэхээс өмнө ажиллавал юу ч
			// нэмэлгүй «дууссан» гэж тэмдэглэгдэнэ.
			name: "000053_parcel_status_code_ids",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SyncParcelStatusWorkflow(ctx)
			},
		},
		{
			// Бүх нэгж талбарт анхны «Хүлээгдэж буй» түүх (хуучин импорт зөвхөн
			// эцсийн төлөвийг бичдэг байсан).
			name: "000051_parcel_pending_history",
			run: func(ctx context.Context) error {
				return NewMainSeeder(mainDB).SeedParcelPendingHistory(ctx)
			},
		},
	}); err != nil {
		return err
	}

	return nil
}
