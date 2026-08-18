//go:build e2e

// Чөлөөлөлт устгах урсгалын E2E шалгалт — БОДИТ PostgreSQL/PostGIS дээр.
//
// Ажиллуулах:
//
//	make test-e2e
//	# эсвэл
//	E2E_DATABASE_URL='postgres://postgres:pass@localhost:5432/appdb?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -v
//
// Шалгах зүйлс:
//   - Чөлөөлсөн нэгж талбартай / баталгаажсан захирамжтай бол устгахгүй
//   - Устгалт нь SOFT: мөр хэвээр, deleted_at тэмдэглэгдэнэ
//   - CASCADE ажиллахгүй: нэгж талбар, хавсралт, захирамж хэвээр
//   - Уншилтууд (GetByID/List/Suggest/FilterOptions/GlobalParcels/Report)
//     устгасныг ХАРУУЛАХГҮЙ
//   - Файлууд фонд, хоорондоо зайтай устгагдана
package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/repository"
	"acquisition_backend/internal/repository/postgres"
	"acquisition_backend/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── бэлтгэл ──────────────────────────────────────────────────

func dsn() string {
	if v := os.Getenv("E2E_DATABASE_URL"); v != "" {
		return v
	}
	get := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		get("DB_USER", "postgres"), get("DB_PASSWORD", "postgres"),
		get("DB_HOST", "localhost"), get("DB_PORT", "5432"),
		get("DB_NAME", "appdb"), get("DB_SSLMODE", "disable"))
}

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn())
	if err != nil {
		t.Skipf("DB холбогдсонгүй (%v) — E2E алгаслаа", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("DB ping бүтсэнгүй (%v) — E2E алгаслаа", err)
	}
	// Миграц 000003 хэрэгжсэн эсэхийг шалгана — soft delete-ийн багана.
	var exists bool
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name='land_acquisition' AND column_name='deleted_at'
		)`).Scan(&exists))
	if !exists {
		pool.Close()
		t.Skip("land_acquisition.deleted_at байхгүй — `make migrate-main-up` ажиллуулна уу")
	}
	return pool
}

// recorder — файл устгах дуудалт ба хоорондын зайг хөтөлдөг хадгалалт.
type recorder struct {
	filestorage.DisabledStorage
	mu      sync.Mutex
	deleted []string
	times   []time.Time
	want    int
	done    chan struct{}
}

func newRecorder(want int) *recorder {
	return &recorder{want: want, done: make(chan struct{})}
}

func (r *recorder) Delete(_ context.Context, fileURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = append(r.deleted, fileURL)
	r.times = append(r.times, time.Now())
	if r.want > 0 && len(r.deleted) == r.want {
		close(r.done)
	}
	return nil
}

func (r *recorder) wait(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(timeout):
		r.mu.Lock()
		got := len(r.deleted)
		r.mu.Unlock()
		t.Fatalf("файлын цэвэрлэгээ дуусаагүй: %d/%d", got, r.want)
	}
}

func (r *recorder) snapshot() ([]string, []time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	gaps := make([]time.Duration, 0, len(r.times))
	for i := 1; i < len(r.times); i++ {
		gaps = append(gaps, r.times[i].Sub(r.times[i-1]))
	}
	return append([]string(nil), r.deleted...), gaps
}

// fixture — нэг тестийн бүрэн өгөгдөл.
type fixture struct {
	pool  *pgxpool.Pool
	repo  repository.LandAcquisitionRepository
	files *recorder
	svc   service.LandService

	acqID    uuid.UUID
	planCode string
	// нэгж талбарууд: төлөв → id
	parcels map[int]uuid.UUID
	docIDs  []uuid.UUID
	draftID uuid.UUID
}

const e2eBoundary = "POLYGON((106.76 47.66,106.82 47.66,106.82 47.72,106.76 47.72,106.76 47.66))"

// newFixture — чөлөөлөлт + нэгж талбарууд + хавсралт + дроны зураг үүсгэнэ.
// Тест дуусахад БҮГДИЙГ хатуу устгана (soft delete-ээс үл хамааран).
func newFixture(t *testing.T, wantFiles int, parcelStatuses ...int) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := openPool(t)
	files := newRecorder(wantFiles)

	f := &fixture{
		pool:     pool,
		repo:     postgres.NewLandAcquisitionRepo(pool),
		files:    files,
		acqID:    uuid.New(),
		planCode: "E2E-DEL-" + uuid.NewString()[:8],
		parcels:  map[int]uuid.UUID{},
	}
	f.svc = service.NewLandService(f.repo, files, nil)

	_, err := pool.Exec(ctx, `
		INSERT INTO land_acquisition (id, plan_code, acquisition_name, status, geometry, area_m2, created_at, updated_at)
		VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326), 41250000, NOW(), NOW())
	`, f.acqID, f.planCode, "E2E устгах тест", domain.LandAcquisitionStatusNew, e2eBoundary)
	require.NoError(t, err)

	for i, status := range parcelStatuses {
		pid := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO parcel (id, parcel_id, acquisition_id, status, area_m2, created_at)
			VALUES ($1, $2, $3, $4, 500, NOW())
		`, pid, fmt.Sprintf("E2E%s%02d", uuid.NewString()[:6], i), f.acqID, status)
		require.NoError(t, err)
		f.parcels[status] = pid
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if f.draftID != uuid.Nil {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM decision_draft_parcel WHERE decision_draft_id = $1`, f.draftID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM decision_draft WHERE id = $1`, f.draftID)
		}
		// land_acquisition-ийг ХАТУУ устгахад CASCADE-ээр туслах мөрүүд цэвэрлэгдэнэ.
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM land_acquisition WHERE id = $1`, f.acqID)
		for _, id := range f.docIDs {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM document WHERE id = $1`, id)
		}
		pool.Close()
	})
	return f
}

// addAcquisitionDocument — чөлөөлөлтийн хавсралт (файлтай).
func (f *fixture) addAcquisitionDocument(t *testing.T, fileURL string) {
	t.Helper()
	docID := uuid.New()
	_, err := f.pool.Exec(context.Background(), `
		INSERT INTO document (id, name, file_type, file_url, uploaded_at)
		VALUES ($1, $2, 'application/pdf', $3, NOW())
	`, docID, "e2e-acq.pdf", fileURL)
	require.NoError(t, err)
	_, err = f.pool.Exec(context.Background(), `
		INSERT INTO acquisition_document (acquisition_id, document_id) VALUES ($1, $2)
	`, f.acqID, docID)
	require.NoError(t, err)
	f.docIDs = append(f.docIDs, docID)
}

// addParcelDocument — нэгж талбарын хавсралт (файлтай).
func (f *fixture) addParcelDocument(t *testing.T, parcelID uuid.UUID, fileURL string) {
	t.Helper()
	docID := uuid.New()
	_, err := f.pool.Exec(context.Background(), `
		INSERT INTO document (id, name, file_type, file_url, uploaded_at)
		VALUES ($1, $2, 'application/pdf', $3, NOW())
	`, docID, "e2e-parcel.pdf", fileURL)
	require.NoError(t, err)
	_, err = f.pool.Exec(context.Background(), `
		INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1, $2)
	`, parcelID, docID)
	require.NoError(t, err)
	f.docIDs = append(f.docIDs, docID)
}

// addDroneImage — дроны ортофото (файлтай).
func (f *fixture) addDroneImage(t *testing.T, fileURL string) {
	t.Helper()
	_, err := f.pool.Exec(context.Background(), `
		INSERT INTO drone_image (id, acquisition_id, file_name, original_name, file_url, uploaded_at)
		VALUES ($1, $2, 'e2e.tif', 'e2e.tif', $3, NOW())
	`, uuid.New(), f.acqID, fileURL)
	require.NoError(t, err)
}

// linkDecisionDraft — захирамжийн төсөл үүсгэж, нэгж талбарыг холбоно.
func (f *fixture) linkDecisionDraft(t *testing.T, parcelID uuid.UUID, status int) {
	t.Helper()
	f.draftID = uuid.New()
	_, err := f.pool.Exec(context.Background(), `
		INSERT INTO decision_draft (id, proposal_no, decree_number, status, created_at, updated_at)
		VALUES ($1, $2, 'А/E2E-123', $3, NOW(), NOW())
	`, f.draftID, "E2E-"+uuid.NewString()[:8], status)
	require.NoError(t, err)
	_, err = f.pool.Exec(context.Background(), `
		INSERT INTO decision_draft_parcel (id, decision_draft_id, parcel_id, linked_at, linked_by)
		VALUES ($1, $2, $3, NOW(), 'e2e')
	`, uuid.New(), f.draftID, parcelID)
	require.NoError(t, err)
}

func (f *fixture) rawDeletedAt(t *testing.T) *time.Time {
	t.Helper()
	var deletedAt *time.Time
	require.NoError(t, f.pool.QueryRow(context.Background(),
		`SELECT deleted_at FROM land_acquisition WHERE id = $1`, f.acqID).Scan(&deletedAt))
	return deletedAt
}

func (f *fixture) countParcels(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM parcel WHERE acquisition_id = $1`, f.acqID).Scan(&n))
	return n
}

// assertVisibleBeforeDelete — устгахаас өмнө БҮХ уншилт чөлөөлөлтийг харуулж
// байгаа эсэх. Ингэснээр устгасны дараах "хоосон" шалгалтууд нь бодитоор
// soft delete-ийн шүүлтийг хэмжинэ (хоосон өгөгдлөөс болж өнгөрөхгүй).
func assertVisibleBeforeDelete(t *testing.T, f *fixture) {
	t.Helper()
	ctx := context.Background()

	acq, err := f.repo.GetByID(ctx, f.acqID)
	require.NoError(t, err)
	require.Equal(t, f.acqID, acq.ID)

	list, total, err := f.repo.List(ctx, domain.LandAcquisitionFilter{
		PlanCode: f.planCode, Page: 1, PageSize: 50,
	})
	require.NoError(t, err)
	require.Len(t, list, 1, "List устгахаас өмнө харуулах ёстой")
	require.Equal(t, int64(1), total)

	sugg, err := f.repo.Suggest(ctx, f.planCode, 0)
	require.NoError(t, err)
	require.Len(t, sugg, 1, "Suggest устгахаас өмнө харуулах ёстой")

	opts, err := f.repo.ListFilterOptions(ctx)
	require.NoError(t, err)
	found := false
	for _, o := range opts {
		if o.ID == f.acqID {
			found = true
			break
		}
	}
	require.True(t, found, "ListFilterOptions устгахаас өмнө харуулах ёстой")

	globals, _, err := f.repo.ListGlobalParcels(ctx, domain.GlobalParcelFilter{
		AcquisitionID: f.acqID.String(), Page: 1, PageSize: 50,
	})
	require.NoError(t, err)
	require.NotEmpty(t, globals, "ListGlobalParcels устгахаас өмнө харуулах ёстой")

	rows, _, err := f.repo.ListReportParcels(ctx, domain.ReportParcelFilter{
		AcquisitionID: f.acqID.String(), Page: 1, PageSize: 50,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rows, "тайлан устгахаас өмнө харуулах ёстой")

	summary, err := f.repo.GetReportSummary(ctx, domain.ReportParcelFilter{
		AcquisitionID: f.acqID.String(),
	})
	require.NoError(t, err)
	require.NotZero(t, summary.ParcelCount, "тайлангийн карт устгахаас өмнө тоолох ёстой")
}

// ── 1. Хязгаарлалт: Чөлөөлсөн нэгж талбар ────────────────────

func TestE2E_Delete_RejectsReleasedParcel(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusPending, domain.ParcelStatusReleased)

	// Шалгалт нь төлөвийг зөв уншиж байгаа эсэх
	check, err := f.repo.CheckAcquisitionDeletable(ctx, f.acqID)
	require.NoError(t, err)
	assert.Equal(t, 2, check.ParcelCount)
	assert.Equal(t, 1, check.ReleasedParcels)
	assert.Equal(t, 1, check.ParcelStatusCounts[domain.ParcelStatusPending])
	assert.Equal(t, 1, check.ParcelStatusCounts[domain.ParcelStatusReleased])

	err = f.svc.Delete(ctx, f.acqID, "e2e-user")
	assert.Equal(t, domain.ErrHasReleasedParcels, err)

	// Устгагдаагүй байх ёстой
	assert.Nil(t, f.rawDeletedAt(t), "deleted_at тэмдэглэгдэх ёсгүй")
	acq, err := f.repo.GetByID(ctx, f.acqID)
	require.NoError(t, err)
	assert.Equal(t, f.acqID, acq.ID)
}

// ── 2. Хязгаарлалт: баталгаажсан захирамж ────────────────────

func TestE2E_Delete_RejectsConfirmedDecisionDraft(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusEvaluating)
	parcelID := f.parcels[domain.ParcelStatusEvaluating]
	f.linkDecisionDraft(t, parcelID, domain.DecisionDraftStatusConfirmed)

	check, err := f.repo.CheckAcquisitionDeletable(ctx, f.acqID)
	require.NoError(t, err)
	assert.Equal(t, 1, check.ConfirmedDecisions)
	assert.Contains(t, check.DecisionNumbers, "А/E2E-123")

	err = f.svc.Delete(ctx, f.acqID, "e2e-user")
	assert.Equal(t, domain.ErrConfirmedDecisionLinked, err)
	assert.Nil(t, f.rawDeletedAt(t))

	// Захирамжийн холбоос хэвээр (CASCADE ажиллаагүй)
	var links int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT count(*) FROM decision_draft_parcel WHERE decision_draft_id = $1`, f.draftID).Scan(&links))
	assert.Equal(t, 1, links)
}

// Баталгаажаагүй (draft) захирамж нь устгалыг зогсоохгүй.
func TestE2E_Delete_AllowsUnconfirmedDecisionDraft(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusProcessing)
	f.linkDecisionDraft(t, f.parcels[domain.ParcelStatusProcessing], domain.DecisionDraftStatusDraft)

	check, err := f.repo.CheckAcquisitionDeletable(ctx, f.acqID)
	require.NoError(t, err)
	assert.Equal(t, 0, check.ConfirmedDecisions)

	require.NoError(t, f.svc.Delete(ctx, f.acqID, "e2e-user"))
	assert.NotNil(t, f.rawDeletedAt(t))

	// Захирамжийн төсөл БОЛОН холбоос нь хэвээр үлдэнэ (soft delete)
	var links int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT count(*) FROM decision_draft_parcel WHERE decision_draft_id = $1`, f.draftID).Scan(&links))
	assert.Equal(t, 1, links, "soft delete нь захирамжийн холбоосыг устгах ёсгүй")
}

// ── 3. Хязгаарлалт: төлөв ────────────────────────────────────

func TestE2E_Delete_RejectsNonNewStatus(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusPending)
	_, err := f.pool.Exec(ctx, `UPDATE land_acquisition SET status = $2 WHERE id = $1`,
		f.acqID, domain.LandAcquisitionStatusField)
	require.NoError(t, err)

	assert.Equal(t, domain.ErrForbidden, f.svc.Delete(ctx, f.acqID, "e2e-user"))
	assert.Nil(t, f.rawDeletedAt(t))
}

// ── 4. SOFT delete + файл цэвэрлэгээ ────────────────────────

func TestE2E_Delete_SoftDeletesAndCleansFiles(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 3, domain.ParcelStatusPending, domain.ParcelStatusEvaluating)
	f.addAcquisitionDocument(t, "acq/e2e-acq.pdf")
	f.addParcelDocument(t, f.parcels[domain.ParcelStatusPending], "parcel/e2e-parcel.pdf")
	f.addDroneImage(t, "drone/e2e.tif")

	// Файлын жагсаалт бүх эх сурвалжийг хамрах ёстой
	urls, err := f.repo.ListAcquisitionFileURLs(ctx, f.acqID)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{"acq/e2e-acq.pdf", "parcel/e2e-parcel.pdf", "drone/e2e.tif"}, urls)

	// Устгахаас ӨМНӨ уншилтууд БҮГД харуулж байгааг батална — үгүй бол
	// доорх "хоосон" assert-ууд хуурамчаар өнгөрнө.
	assertVisibleBeforeDelete(t, f)

	start := time.Now()
	require.NoError(t, f.svc.Delete(ctx, f.acqID, "e2e-user"))
	// Устгалт нь файлын цэвэрлэгээг ХҮЛЭЭХГҮЙ (фонд ажиллана)
	assert.Less(t, time.Since(start), 200*time.Millisecond,
		"устгалт нь файлын цэвэрлэгээг хүлээх ёсгүй")

	// 4.1 Мөр хэвээр, deleted_at/deleted_by тэмдэглэгдсэн
	deletedAt := f.rawDeletedAt(t)
	require.NotNil(t, deletedAt, "мөр устсан байна — soft delete ажиллаагүй")
	var deletedBy *string
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT deleted_by FROM land_acquisition WHERE id = $1`, f.acqID).Scan(&deletedBy))
	require.NotNil(t, deletedBy)
	assert.Equal(t, "e2e-user", *deletedBy)

	// 4.2 CASCADE ажиллаагүй — нэгж талбар, хавсралт хэвээр
	assert.Equal(t, 2, f.countParcels(t), "нэгж талбар устсан байна (CASCADE)")
	var docs int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT count(*) FROM acquisition_document WHERE acquisition_id = $1`, f.acqID).Scan(&docs))
	assert.Equal(t, 1, docs)

	// 4.3 Уншилтууд устгасныг ХАРУУЛАХГҮЙ
	_, err = f.repo.GetByID(ctx, f.acqID)
	assert.ErrorIs(t, err, domain.ErrNotFound, "GetByID устгасныг буцаах ёсгүй")

	list, _, err := f.repo.List(ctx, domain.LandAcquisitionFilter{PlanCode: f.planCode, Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Empty(t, list, "List устгасныг харуулах ёсгүй")

	sugg, err := f.repo.Suggest(ctx, f.planCode, 0)
	require.NoError(t, err)
	assert.Empty(t, sugg, "Suggest устгасныг харуулах ёсгүй")

	opts, err := f.repo.ListFilterOptions(ctx)
	require.NoError(t, err)
	for _, o := range opts {
		assert.NotEqual(t, f.acqID, o.ID, "ListFilterOptions устгасныг харуулах ёсгүй")
	}

	globals, _, err := f.repo.ListGlobalParcels(ctx, domain.GlobalParcelFilter{
		AcquisitionID: f.acqID.String(), Page: 1, PageSize: 50,
	})
	require.NoError(t, err)
	assert.Empty(t, globals, "ListGlobalParcels устгасны нэгж талбарыг харуулах ёсгүй")

	reportRows, _, err := f.repo.ListReportParcels(ctx, domain.ReportParcelFilter{
		AcquisitionID: f.acqID.String(), Page: 1, PageSize: 50,
	})
	require.NoError(t, err)
	assert.Empty(t, reportRows, "тайланд устгасны нэгж талбар орох ёсгүй")

	summary, err := f.repo.GetReportSummary(ctx, domain.ReportParcelFilter{
		AcquisitionID: f.acqID.String(),
	})
	require.NoError(t, err)
	assert.Zero(t, summary.ParcelCount, "тайлангийн картад устгасан орох ёсгүй")
	assert.Zero(t, summary.AcquisitionCount, "тайлангийн картад устгасан орох ёсгүй")

	// 4.4 Файлууд фонд, хоорондоо зайтай устсан
	f.files.wait(t, 15*time.Second)
	deleted, gaps := f.files.snapshot()
	assert.ElementsMatch(t,
		[]string{"acq/e2e-acq.pdf", "parcel/e2e-parcel.pdf", "drone/e2e.tif"}, deleted)
	require.Len(t, gaps, 2)
	for i, gap := range gaps {
		assert.Greater(t, gap, 100*time.Millisecond,
			"%d-р файл хэт хурдан устлаа: %s", i+2, gap)
	}

	// 4.5 Дахин устгах гэвэл олдохгүй
	assert.Equal(t, domain.ErrNotFound, f.svc.Delete(ctx, f.acqID, "e2e-user"))
}

// ── 5. Нэгж талбарын дугаар дахин хэрэглэгдэх ────────────────

// Устгасан чөлөөлөлтийн нэгж талбарын дугаарыг ШИНЭ чөлөөлөлт эргүүлэн
// эзэмшинэ. parcel.parcel_id глобал unique тул үүнгүй бол буруу бүртгэсэн
// чөлөөлөлтийг устгаад ДАХИН бүртгэхэд нэгж талбар бүгд алгасагдана.
func TestE2E_Delete_FreesParcelCodeForNewAcquisition(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusPending)
	oldParcelID := f.parcels[domain.ParcelStatusPending]

	var parcelCode string
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT parcel_id FROM parcel WHERE id = $1`, oldParcelID).Scan(&parcelCode))

	require.NoError(t, f.svc.Delete(ctx, f.acqID, "e2e-user"))

	// Нэгж талбар мөн "устгасан" гэж тэмдэглэгдсэн байх (мөр хэвээр).
	var parcelDeletedAt *time.Time
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT deleted_at FROM parcel WHERE id = $1`, oldParcelID).Scan(&parcelDeletedAt))
	require.NotNil(t, parcelDeletedAt, "нэгж талбар тэмдэглэгдээгүй")

	// Дугаараар ГЛОБАЛ хайхад устгасан мөр буцахгүй.
	_, err := f.repo.GetParcelByCode(ctx, parcelCode)
	assert.ErrorIs(t, err, domain.ErrNotFound, "GetParcelByCode устгасныг буцаах ёсгүй")

	// ШИНЭ чөлөөлөлт ижил дугаарыг бүртгэнэ.
	newAcqID := uuid.New()
	_, err = f.pool.Exec(ctx, `
		INSERT INTO land_acquisition (id, plan_code, acquisition_name, status, geometry, created_at, updated_at)
		VALUES ($1, $2, 'E2E дахин бүртгэл', $3, ST_GeomFromText($4, 4326), NOW(), NOW())
	`, newAcqID, f.planCode+"-2", domain.LandAcquisitionStatusNew, e2eBoundary)
	require.NoError(t, err)
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = f.pool.Exec(cctx, `DELETE FROM land_acquisition WHERE id = $1`, newAcqID)
	})

	res, err := f.repo.InsertParcelCodes(ctx, newAcqID, []string{parcelCode}, e2eBoundary)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Total)
	assert.Equal(t, 0, res.Skipped, "устгасан чөлөөлөлтийн дугаар алгасагдах ёсгүй")
	assert.Equal(t, 1, res.Created, "нэгж талбар шинэ чөлөөлөлтөд бүртгэгдэх ёстой")

	// Мөр нь ШИНЭ чөлөөлөлтөд шилжиж, дахин "ажиллаж байгаа" болсон.
	var (
		owner  uuid.UUID
		delAt  *time.Time
		status int
	)
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT acquisition_id, deleted_at, COALESCE(status,0) FROM parcel WHERE parcel_id = $1`,
		parcelCode).Scan(&owner, &delAt, &status))
	assert.Equal(t, newAcqID, owner)
	assert.Nil(t, delAt)
	assert.Equal(t, domain.ParcelStatusPending, status)

	// Шинэ чөлөөлөлтийн жагсаалтад харагдана.
	parcels, total, err := f.repo.GetParcels(ctx, newAcqID, domain.ParcelFilter{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, parcels, 1)
	assert.Equal(t, parcelCode, parcels[0].ParcelID)
}

// Файлгүй чөлөөлөлт устгахад цэвэрлэгээ огт хийгдэхгүй.
func TestE2E_Delete_NoFilesNoCleanup(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	urls, err := f.repo.ListAcquisitionFileURLs(ctx, f.acqID)
	require.NoError(t, err)
	assert.Empty(t, urls)

	require.NoError(t, f.svc.Delete(ctx, f.acqID, "e2e-user"))
	assert.NotNil(t, f.rawDeletedAt(t))

	time.Sleep(300 * time.Millisecond)
	deleted, _ := f.files.snapshot()
	assert.Empty(t, deleted)
}
