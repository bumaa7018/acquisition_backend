//go:build e2e

// PurgeParcels — шинэ чөлөөлөх хилд ОРООГҮЙ болсон нэгж талбарыг БҮРМӨСӨН
// устгах урсгалын E2E шалгалт (БОДИТ PostgreSQL/PostGIS дээр).
//
// Ажиллуулах:
//
//	make test-e2e
//	# эсвэл
//	E2E_DATABASE_URL='postgres://postgres:pass@localhost:5432/appdb?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run E2E_Purge -v
//
// Яагаад E2E шаардлагатай вэ: устгалт нь 17 хүснэгтийг ДАРААЛУУЛЖ цэвэрлэдэг.
// Дараалал буруу бол FK зөрчил гарна, харин FK-гүй хүснэгт (compensation,
// parcel_valuation_submission_history) дээр мөр ГАНЦААРДАН үлдэнэ — хоёуланг
// mock-оор шалгах боломжгүй.
package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// purgeFixture — устгагдах нэгж талбар + түүний ХҮҮ мөрүүд бүрэн (хүснэгт
// тус бүрд дор хаяж нэг мөр), мөн УСТАХГҮЙ хөрш нэгж талбар.
type purgeFixture struct {
	f *fixture

	staleID   uuid.UUID // хилээс гарсан — устана
	staleCode string
	keepID    uuid.UUID // хилд хэвээр — үлдэнэ
	keepCode  string

	staleDocID  uuid.UUID // зөвхөн stale-д холбогдсон → устана
	sharedDocID uuid.UUID // stale + keep хоёуланд → ҮЛДЭНЭ
	compID      uuid.UUID
	assetID     uuid.UUID
}

func newPurgeFixture(t *testing.T) *purgeFixture {
	t.Helper()
	ctx := context.Background()

	// Хоёр нэгж талбар: аль нь ч файл устгахгүй тул wantFiles=0.
	f := newFixture(t, 0, domain.ParcelStatusPending, domain.ParcelStatusProcessing)
	p := &purgeFixture{f: f}

	// newFixture нь төлөв → id map хийдэг; кодыг DB-ээс уншина.
	p.staleID = f.parcels[domain.ParcelStatusPending]
	p.keepID = f.parcels[domain.ParcelStatusProcessing]
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT parcel_id FROM parcel WHERE id = $1`, p.staleID).Scan(&p.staleCode))
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT parcel_id FROM parcel WHERE id = $1`, p.keepID).Scan(&p.keepCode))

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := f.pool.Exec(ctx, q, args...)
		require.NoError(t, err, q)
	}

	// ── Баримт: нэг нь ЗӨВХӨН stale-д, нэг нь ХУВААЛЦСАН ────────────────────
	p.staleDocID, p.sharedDocID = uuid.New(), uuid.New()
	f.docIDs = append(f.docIDs, p.staleDocID, p.sharedDocID)
	for _, d := range []struct {
		id  uuid.UUID
		url string
	}{{p.staleDocID, "/api/files/e2e-stale.pdf"}, {p.sharedDocID, "/api/files/e2e-shared.pdf"}} {
		exec(`INSERT INTO document (id, name, file_type, file_url, uploaded_at)
		      VALUES ($1, $2, 'application/pdf', $3, NOW())`,
			d.id, "E2E purge "+d.url, d.url)
	}
	exec(`INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1, $2)`, p.staleID, p.staleDocID)
	exec(`INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1, $2)`, p.staleID, p.sharedDocID)
	exec(`INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1, $2)`, p.keepID, p.sharedDocID)

	// ── FK-ТЭЙ хүү мөрүүд (parcel.id-аар) ───────────────────────────────────
	exec(`INSERT INTO parcel_detail (id, parcel_id, acquisition_id, right_type, app_no)
	      VALUES (uuid_generate_v4(), $1, $2, 2, 'E2E-APP')`, p.staleID, f.acqID)
	exec(`INSERT INTO parcel_payment (id, parcel_id, amount, currency, paid_at, created_at)
	      VALUES (uuid_generate_v4(), $1, 1000, 'MNT', NOW(), NOW())`, p.staleID)

	// ── FK-ГҮЙ / кодоор холбогдсон мөрүүд ───────────────────────────────────
	p.compID = uuid.New()
	exec(`INSERT INTO compensation (id, acquisition_id, parcel_id, target_type, amount, valuation_report_url, created_at)
	      VALUES ($1, $2, $3, 'parcel', 5000, '/api/files/e2e-report.pdf', NOW())`,
		p.compID, f.acqID, p.staleCode)
	exec(`INSERT INTO compensation_grant (id, compensation_id, acquisition_id, created_at)
	      VALUES (uuid_generate_v4(), $1, $2, NOW())`, p.compID, f.acqID)

	p.assetID = uuid.New()
	exec(`INSERT INTO assets (id, acquisition_id, parcel_id, photo_pdf_url, created_at)
	      VALUES ($1, $2, $3, '/api/files/e2e-photo.pdf', NOW())`, p.assetID, f.acqID, p.staleCode)
	exec(`INSERT INTO asset_spec (id, asset_id, spec_type_id, value)
	      VALUES (uuid_generate_v4(), $1, (SELECT id FROM asset_spec_type ORDER BY id LIMIT 1), 'e2e')`, p.assetID)

	exec(`INSERT INTO land_valuation (id, acquisition_id, parcel_id, created_at)
	      VALUES (uuid_generate_v4(), $1, $2, NOW())`, f.acqID, p.staleCode)
	// parcel_status_history.id нь serial (integer) — uuid БИШ.
	exec(`INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, status_date, created_by)
	      VALUES ($1, $2, 0, NOW(), 'e2e')`, p.staleCode, f.acqID)

	// Хөршийн (keep) мөр — устгалтад хөндөгдөх ЁСГҮЙ.
	exec(`INSERT INTO compensation (id, acquisition_id, parcel_id, target_type, amount, created_at)
	      VALUES (uuid_generate_v4(), $1, $2, 'parcel', 7777, NOW())`, f.acqID, p.keepCode)

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// FK-гүй хүснэгтүүд land_acquisition-ийн CASCADE-д ХАМААРАХГҮЙ тул
		// тестийн дараа өөрсдөө цэвэрлэнэ (устгалт бүтэлгүйтсэн ч).
		for _, q := range []string{
			`DELETE FROM parcel_status_history WHERE parcel_id = ANY($1)`,
			`DELETE FROM land_valuation WHERE parcel_id = ANY($1)`,
		} {
			_, _ = f.pool.Exec(cctx, q, []string{p.staleCode, p.keepCode})
		}
	})
	return p
}

func countRows(t *testing.T, p *purgeFixture, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, p.f.pool.QueryRow(context.Background(), query, args...).Scan(&n))
	return n
}

// Устгахаас ӨМНӨ мөр бүр БАЙГАА эсэхийг батална — эс бөгөөс "устсан" гэсэн
// шалгалт хуурамчаар дамжина (мөр эхнээсээ байгаагүй бол).
func assertChildRowsExist(t *testing.T, p *purgeFixture) {
	t.Helper()
	for name, q := range map[string]string{
		"parcel_detail":   `SELECT count(*) FROM parcel_detail WHERE parcel_id = $1`,
		"parcel_payment":  `SELECT count(*) FROM parcel_payment WHERE parcel_id = $1`,
		"parcel_document": `SELECT count(*) FROM parcel_document WHERE parcel_id = $1`,
	} {
		assert.NotZero(t, countRows(t, p, q, p.staleID), "%s мөр устгахын ӨМНӨ байх ёстой", name)
	}
	for name, q := range map[string]string{
		"compensation":          `SELECT count(*) FROM compensation WHERE parcel_id = $1`,
		"assets":                `SELECT count(*) FROM assets WHERE parcel_id = $1`,
		"land_valuation":        `SELECT count(*) FROM land_valuation WHERE parcel_id = $1`,
		"parcel_status_history": `SELECT count(*) FROM parcel_status_history WHERE parcel_id = $1`,
	} {
		assert.NotZero(t, countRows(t, p, q, p.staleCode), "%s мөр устгахын ӨМНӨ байх ёстой", name)
	}
	assert.NotZero(t, countRows(t, p, `SELECT count(*) FROM compensation_grant WHERE compensation_id = $1`, p.compID))
	assert.NotZero(t, countRows(t, p, `SELECT count(*) FROM asset_spec WHERE asset_id = $1`, p.assetID))
}

// Гол шалгалт: хилээс гарсан нэгж талбар нь ХҮҮ мөрүүдтэйгээ (FK-гүй
// хүснэгтүүд ч) БҮРМӨСӨН устаж, ҮЛДЭХ талбарын өгөгдөл хөндөгдөхгүй.
func TestE2E_PurgeParcels_DeletesParcelAndAllChildRows(t *testing.T) {
	ctx := context.Background()
	p := newPurgeFixture(t)
	assertChildRowsExist(t, p)

	removed, fileURLs, err := p.f.repo.PurgeParcels(ctx, p.f.acqID,
		[]uuid.UUID{p.staleID}, []string{p.staleCode})

	require.NoError(t, err)
	assert.Equal(t, 1, removed, "нэг нэгж талбар устах ёстой")

	// ── Нэгж талбар өөрөө: SOFT биш, БҮРМӨСӨН ───────────────────────────────
	assert.Zero(t, countRows(t, p, `SELECT count(*) FROM parcel WHERE id = $1`, p.staleID),
		"нэгж талбарын мөр бүрмөсөн устах ёстой (deleted_at биш)")

	// ── FK-тэй хүү мөрүүд ───────────────────────────────────────────────────
	for name, q := range map[string]string{
		"parcel_detail":   `SELECT count(*) FROM parcel_detail WHERE parcel_id = $1`,
		"parcel_payment":  `SELECT count(*) FROM parcel_payment WHERE parcel_id = $1`,
		"parcel_document": `SELECT count(*) FROM parcel_document WHERE parcel_id = $1`,
	} {
		assert.Zero(t, countRows(t, p, q, p.staleID), "%s мөр устах ёстой", name)
	}

	// ── FK-ГҮЙ / кодоор холбогдсон мөрүүд (CASCADE тэднийг ЦЭВЭРЛЭХГҮЙ) ────
	for name, q := range map[string]string{
		"compensation":          `SELECT count(*) FROM compensation WHERE parcel_id = $1`,
		"assets":                `SELECT count(*) FROM assets WHERE parcel_id = $1`,
		"land_valuation":        `SELECT count(*) FROM land_valuation WHERE parcel_id = $1`,
		"parcel_status_history": `SELECT count(*) FROM parcel_status_history WHERE parcel_id = $1`,
	} {
		assert.Zero(t, countRows(t, p, q, p.staleCode), "%s мөр ганцаардан үлдэх ЁСГҮЙ", name)
	}
	assert.Zero(t, countRows(t, p, `SELECT count(*) FROM compensation_grant WHERE compensation_id = $1`, p.compID),
		"олговрын хуваарь устах ёстой")
	assert.Zero(t, countRows(t, p, `SELECT count(*) FROM asset_spec WHERE asset_id = $1`, p.assetID),
		"хөрөнгийн үзүүлэлт устах ёстой")

	// ── Баримт: зөвхөн stale-д холбогдсон нь устаж, ХУВААЛЦСАН нь үлдэнэ ───
	assert.Zero(t, countRows(t, p, `SELECT count(*) FROM document WHERE id = $1`, p.staleDocID),
		"зөвхөн устсан талбарт холбогдсон баримт устана")
	assert.Equal(t, 1, countRows(t, p, `SELECT count(*) FROM document WHERE id = $1`, p.sharedDocID),
		"өөр талбартай ХУВААЛЦСАН баримт үлдэх ёстой")
	assert.Equal(t, 1, countRows(t, p,
		`SELECT count(*) FROM parcel_document WHERE parcel_id = $1 AND document_id = $2`,
		p.keepID, p.sharedDocID), "үлдэх талбарын баримтын холбоос хөндөгдөхгүй")

	// ── Устгах файлын URL-ууд дуудагчид буцна ───────────────────────────────
	assert.Contains(t, fileURLs, "/api/files/e2e-stale.pdf")
	assert.Contains(t, fileURLs, "/api/files/e2e-report.pdf", "үнэлгээний тайлан")
	assert.Contains(t, fileURLs, "/api/files/e2e-photo.pdf", "хөрөнгийн зураг")
	assert.NotContains(t, fileURLs, "/api/files/e2e-shared.pdf",
		"хуваалцсан баримтын файлыг устгаж БОЛОХГҮЙ")

	// ── ҮЛДЭХ нэгж талбар бүрэн бүтэн ───────────────────────────────────────
	assert.Equal(t, 1, countRows(t, p, `SELECT count(*) FROM parcel WHERE id = $1`, p.keepID))
	assert.Equal(t, 1, countRows(t, p,
		`SELECT count(*) FROM compensation WHERE parcel_id = $1 AND amount = 7777`, p.keepCode))
}

// Хоосон жагсаалт — юу ч устгахгүй, алдаа гаргахгүй (хил солиход зөрүү
// байхгүй тохиолдол).
func TestE2E_PurgeParcels_EmptyListIsNoop(t *testing.T) {
	p := newPurgeFixture(t)

	removed, urls, err := p.f.repo.PurgeParcels(context.Background(), p.f.acqID, nil, nil)

	require.NoError(t, err)
	assert.Zero(t, removed)
	assert.Empty(t, urls)
	assert.Equal(t, 1, countRows(t, p, `SELECT count(*) FROM parcel WHERE id = $1`, p.staleID),
		"хоосон жагсаалт дээр нэгж талбар хөндөгдөх ЁСГҮЙ")
}

// InsertParcelCodes нь ГЕОМЕТР болон бүртгэлийн баганыг хадгалж, MULTIPOLYGON
// ирсэн ч нэгж талбарыг геометртэйгээ бүртгэх ёстой (нэг хэсэгтэй бол).
func TestE2E_InsertParcelCodes_StoresGeometryAndAttributes(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	code := fmt.Sprintf("E2EG%s", uuid.NewString()[:8])
	multi := "MULTIPOLYGON(((106.77 47.67,106.79 47.67,106.79 47.69,106.77 47.69,106.77 47.67)))"

	res, err := f.repo.InsertParcelCodes(ctx, f.acqID, []*domain.SourceParcel{{
		ParcelID:       code,
		GeometryWKT:    multi,
		AU2Code:        "0101",
		RightType:      2,
		Landuse:        "2201",
		LanduseName:    "Гэр бүлийн хэрэгцээ",
		AreaM2:         700,
		AddressKhashaa: "12",
		AddressStreet:  "Их тойруу",
		PropertyNo:     "Э-1",
	}}, e2eBoundary)

	require.NoError(t, err)
	assert.Equal(t, 1, res.Created)

	var (
		hasGeom     bool
		geomType    string
		overlapM2   *float64
		au2, landus string
		areaM2      float64
		rightType   int
	)
	require.NoError(t, f.pool.QueryRow(ctx, `
		SELECT geometry IS NOT NULL, COALESCE(GeometryType(geometry), ''),
		       acquisition_area_m2, COALESCE(au2_code,''), COALESCE(landuse,''),
		       COALESCE(area_m2,0), COALESCE(right_type,0)
		FROM parcel WHERE parcel_id = $1
	`, code).Scan(&hasGeom, &geomType, &overlapM2, &au2, &landus, &areaM2, &rightType))

	assert.True(t, hasGeom, "MULTIPOLYGON ирсэн ч геометр хадгалагдах ёстой")
	assert.Equal(t, "POLYGON", geomType, "geometry(Polygon,4326) багана болж хөрвөнө")
	require.NotNil(t, overlapM2, "чөлөөлөгдөх талбай тооцоологдох ёстой")
	assert.Greater(t, *overlapM2, 0.0)
	assert.Equal(t, "0101", au2)
	assert.Equal(t, "2201", landus)
	assert.Equal(t, 700.0, areaM2)
	assert.Equal(t, 2, rightType)
}

// parcelSourceRow — refresh шалгалтад уншигдах бүртгэлийн баганууд.
type parcelSourceRow struct {
	geomType                     string
	au1, au2, landuse, landuseNm string
	khashaa, street, propertyNo  string
	oldParcelID                  string
	areaM2                       float64
	rightType, status            int
	acquisitionM2, remainingM2   *float64
	hasAcquisitionGeom           bool
	appNo                        string
}

func readParcelSourceRow(t *testing.T, f *fixture, code string) parcelSourceRow {
	t.Helper()
	var r parcelSourceRow
	require.NoError(t, f.pool.QueryRow(context.Background(), `
		SELECT COALESCE(GeometryType(p.geometry), ''),
		       COALESCE(p.au1_code,''), COALESCE(p.au2_code,''),
		       COALESCE(p.landuse,''), COALESCE(p.landuse_name,''),
		       COALESCE(p.address_khashaa,''), COALESCE(p.address_streetname,''),
		       COALESCE(p.property_no,''), COALESCE(p.old_parcel_id,''),
		       COALESCE(p.area_m2,0), COALESCE(p.right_type,0), p.status,
		       p.acquisition_area_m2, p.remaining_area_m2, p.acquisition_geom IS NOT NULL,
		       COALESCE((SELECT d.app_no FROM parcel_detail d WHERE d.parcel_id = p.id LIMIT 1), '')
		FROM parcel p WHERE p.parcel_id = $1
	`, code).Scan(
		&r.geomType, &r.au1, &r.au2, &r.landuse, &r.landuseNm,
		&r.khashaa, &r.street, &r.propertyNo, &r.oldParcelID,
		&r.areaM2, &r.rightType, &r.status,
		&r.acquisitionM2, &r.remainingM2, &r.hasAcquisitionGeom, &r.appNo,
	))
	return r
}

// Гол шалгалт: ХУУЧИН хувилбараар зөвхөн ДУГААРААР бүртгэгдсэн нэгж талбар
// дээр дахин "нэгж талбар татах" хийхэд ГУС-аас ирсэн бүртгэлийн багана бүр
// нөхөгдөж, чөлөөлөгдөх хил/талбай тооцоологдох ёстой. Хэрэглэгчийн өгөгдөл
// (төлөв) ба эрх зүйн дэлгэрэнгүй (parcel_detail.app_no) хөндөгдөхгүй.
func TestE2E_InsertParcelCodes_RefreshesExistingRow(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0)

	code := fmt.Sprintf("E2ER%s", uuid.NewString()[:8])
	// Хилээс (e2eBoundary: 106.76..106.82) ХАГАС гарсан нэгж талбар — давхцал
	// ба үлдэх талбай хоёулаа 0-ээс их гарна.
	parcelWKT := "POLYGON((106.80 47.70,106.84 47.70,106.84 47.71,106.80 47.71,106.80 47.70))"

	// ── 1. ХУУЧИН хувилбарын бүртгэл: зөвхөн дугаар ─────────────────────────
	res, err := f.repo.InsertParcelCodes(ctx, f.acqID,
		[]*domain.SourceParcel{{ParcelID: code}}, e2eBoundary)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)

	before := readParcelSourceRow(t, f, code)
	require.Empty(t, before.geomType, "хуучин бүртгэл геометргүй байх ёстой")
	require.Empty(t, before.landuse)

	var parcelUUID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM parcel WHERE parcel_id = $1`, code).Scan(&parcelUUID))

	// Хэрэглэгчийн төлөв, эрх зүйн дэлгэрэнгүй, геометрээс нөхсөн АУ код —
	// дахин татахад хөндөгдөх ЁСГҮЙ.
	_, err = f.pool.Exec(ctx,
		`UPDATE parcel SET status = $2, au1_code = '01' WHERE id = $1`,
		parcelUUID, domain.ParcelStatusProcessing)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `
		INSERT INTO parcel_detail (id, parcel_id, acquisition_id, right_type, app_no)
		VALUES (uuid_generate_v4(), $1, $2, 2, 'E2E-KEEP-APP')
	`, parcelUUID, f.acqID)
	require.NoError(t, err)

	// ── 2. ДАХИН татах: бүх багана шинэчлэгдэнэ ─────────────────────────────
	res, err = f.repo.InsertParcelCodes(ctx, f.acqID, []*domain.SourceParcel{{
		ParcelID:       code,
		OldParcelID:    "OLD-1",
		GeometryWKT:    parcelWKT,
		AU2Code:        "0101",
		RightType:      2,
		Landuse:        "2201",
		LanduseName:    "Гэр бүлийн хэрэгцээ",
		AreaM2:         3_000_000,
		AddressKhashaa: "12",
		AddressStreet:  "Их тойруу",
		PropertyNo:     "Э-1",
	}}, e2eBoundary)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created, "байгаа мөр дахин үүсэх ЁСГҮЙ")
	assert.Equal(t, 1, res.Existing)
	assert.Equal(t, 0, res.Skipped)

	after := readParcelSourceRow(t, f, code)
	assert.Equal(t, "POLYGON", after.geomType, "геометр нөхөгдөх ёстой")
	assert.Equal(t, "OLD-1", after.oldParcelID)
	assert.Equal(t, "0101", after.au2)
	assert.Equal(t, "2201", after.landuse)
	assert.Equal(t, "Гэр бүлийн хэрэгцээ", after.landuseNm)
	assert.Equal(t, "12", after.khashaa)
	assert.Equal(t, "Их тойруу", after.street)
	assert.Equal(t, "Э-1", after.propertyNo)
	assert.Equal(t, 3_000_000.0, after.areaM2)
	assert.Equal(t, 2, after.rightType)

	// Геометр орж ирсэн тул чөлөөлөгдөх хил/талбай, үлдэх талбай бодогдоно.
	assert.True(t, after.hasAcquisitionGeom, "чөлөөлөгдөх хил тооцоологдох ёстой")
	require.NotNil(t, after.acquisitionM2)
	assert.Greater(t, *after.acquisitionM2, 0.0)
	require.NotNil(t, after.remainingM2, "хилээс гарсан хэсэг бий тул үлдэх талбай бодогдоно")
	assert.Greater(t, *after.remainingM2, 0.0)

	// ХӨНДӨГДӨХГҮЙ: хэрэглэгчийн төлөв, дэлгэрэнгүй, геометрээс нөхсөн au1.
	assert.Equal(t, domain.ParcelStatusProcessing, after.status, "төлөв хөндөгдөх ЁСГҮЙ")
	assert.Equal(t, "E2E-KEEP-APP", after.appNo, "parcel_detail хөндөгдөх ЁСГҮЙ")
	assert.Equal(t, "01", after.au1, "эх систем au1 буцаадаггүй — хадгалагдсан утга үлдэнэ")

	// ── 3. ДУТУУ мэдээлэл ирвэл хадгалагдсаныг ХООСЛОХГҮЙ ───────────────────
	res, err = f.repo.InsertParcelCodes(ctx, f.acqID,
		[]*domain.SourceParcel{{ParcelID: code}}, e2eBoundary)
	require.NoError(t, err)
	require.Equal(t, 1, res.Existing)

	kept := readParcelSourceRow(t, f, code)
	assert.Equal(t, "POLYGON", kept.geomType, "геометр хоосон ирэхэд УСТАХ ёсгүй")
	assert.Equal(t, "2201", kept.landuse)
	assert.Equal(t, "Их тойруу", kept.street)
	assert.Equal(t, 3_000_000.0, kept.areaM2)
	assert.True(t, kept.hasAcquisitionGeom)
}
