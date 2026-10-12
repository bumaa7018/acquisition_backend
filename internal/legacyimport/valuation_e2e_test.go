//go:build e2e

package legacyimport

import (
	"acquisition_backend/internal/seed"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// valuationWorkbook — жишээ эксэл + «2015» sheet (архивын адил оноор):
//
//	мөр 3: Хана-2-ын 1-р нэгж талбар ДАХИН (өөр он), нэр нь ТОМ үсгээр → D2…, нэг төсөл
//	мөр 4: дугааргүй → ERR…
//	мөр 5: «Өөр авто зам 7» төсөл, ГУС-д олдохгүй дугаар → эксэлийн мэдээллээр
//	мөр 6: «Өөр авто зам 7», зөвхөн data_ub-д байгаа дугаар
func valuationWorkbook(t *testing.T, sample []byte) []byte {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(sample))
	require.NoError(t, err)
	defer f.Close()
	src := f.GetSheetList()[0]
	srcIdx, err := f.GetSheetIndex(src)
	require.NoError(t, err)
	idx, err := f.NewSheet("2015")
	require.NoError(t, err)
	require.NoError(t, f.CopySheet(srcIdx, idx))
	rows, err := f.GetRows("2015")
	require.NoError(t, err)
	for r := len(rows); r >= 7; r-- {
		require.NoError(t, f.RemoveRow("2015", r))
	}
	set := func(cell, v string) { require.NoError(t, f.SetCellValue("2015", cell, v)) }
	project, err := f.GetCellValue("2015", "C3")
	require.NoError(t, err)
	set("C3", strings.ToUpper(project))
	set("K4", "")
	for _, r := range []string{"5", "6"} {
		set("B"+r, "Авто зам")
		set("C"+r, "Өөр авто зам 7")
	}
	set("K5", "9999000001")
	set("K6", "9999000002")
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	return buf.Bytes()
}

func projectByName(t *testing.T, v *ReportView, name string) ValuationProject {
	t.Helper()
	for _, p := range v.Projects {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("«%s» төсөл олдсонгүй", name)
	return ValuationProject{}
}

// «ЗӨВХӨН ҮНЭЛГЭЭ» горимын бүтэн урсгал — ГУС mock.
func TestE2ELegacyImportValuationOnly(t *testing.T) {
	dsn := os.Getenv("E2E_DATABASE_URL")
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	samples := loadDir(t, sampleDir(t, "үнэлгээ"))
	var sample UploadedFile
	for _, f := range samples {
		if strings.HasSuffix(f.Name, ".xlsx") {
			sample = f
		}
	}
	require.NotEmpty(t, sample.Data)
	workbook := valuationWorkbook(t, sample.Data)
	files := []UploadedFile{{Name: "archive-test.xlsx", Data: workbook}}
	parsed, _, err := readReport("archive-test.xlsx", workbook)
	require.NoError(t, err)
	require.Len(t, parsed, 180)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// ── ГУС mock: SHP-ийн хилээр info, 9999000002 зөвхөн ub ─────────────────
	features, _, err := readShapefileSet(groupShapefiles(loadDir(t, sampleDir(t, "нэгж-талбар")))[0])
	require.NoError(t, err)
	geomByID := map[string]string{}
	for _, f := range features {
		geomByID[parcelKey(attr(f.props, parcelIDFields...))] = f.wkt
	}
	var mu sync.Mutex
	paths := map[string]int{}
	gus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths[r.URL.RequestURI()]++
		mu.Unlock()
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		send := func(d map[string]any) { _ = json.NewEncoder(w).Encode(map[string]any{"data": d}) }
		switch {
		case strings.HasPrefix(r.URL.Path, "/parcel/info/") && geomByID[id] != "":
			send(map[string]any{"parcel_id": id, "geometry_wkt": geomByID[id], "area_m2": 650, "right_type": 3})
		case r.URL.Path == "/parcel/ub/9999000002":
			send(map[string]any{"parcel_id": id, "geometry_wkt": "POLYGON((106.8 47.9,106.801 47.9,106.801 47.901,106.8 47.901,106.8 47.9))"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer gus.Close()

	// ── Лавлах ба хяналтын (ӨӨР) мэдээлэл ─────────────────────────────────
	var catIDs []int
	for _, name := range []string{categoryGer, categoryRoad} {
		var id int
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO acquisition_category(name, parent_id, sort_order) VALUES ($1, NULL, 90) RETURNING id`, name).Scan(&id))
		catIDs = append(catIDs, id)
	}
	// Системийн төлөв (101–109) — импорт кодыг эдгээртэй тулгана.
	require.NoError(t, seed.NewMainSeeder(pool).SyncParcelStatusWorkflow(ctx))
	const dbParcel = "1802300711" // Хана-2-ын 2-р мөр — санд ӨӨР чөлөөлөлтөд
	var ctrlAcq, ctrlParcel string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition(plan_code, acquisition_name, created_by)
		VALUES ('E2E-VAL-CTRL', 'E2E бусад', 'e2e-val') RETURNING id`).Scan(&ctrlAcq))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel(acquisition_id, parcel_id, created_by) VALUES ($1, $2, 'e2e-val') RETURNING id`,
		ctrlAcq, dbParcel).Scan(&ctrlParcel))
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sessions := `SELECT id FROM legacy_import_stage.session WHERE created_by = 'e2e-val'`
		for _, q := range []string{
			`DELETE FROM decision_draft WHERE created_by = 'legacy-import'`,
			`DELETE FROM land_acquisition WHERE created_by IN ('legacy-import', 'e2e-val') OR legacy_import_id IN (` + sessions + `)`,
			`DELETE FROM legacy_import_stage.rollback_log WHERE session_id IN (` + sessions + `)`,
			`DELETE FROM legacy_import_stage.owned WHERE session_id IN (` + sessions + `)`,
			`DELETE FROM legacy_import_stage.session WHERE created_by = 'e2e-val'`,
			`DELETE FROM right_holder h WHERE NOT EXISTS (SELECT 1 FROM parcel_holder x WHERE x.holder_id = h.id)
			   AND NOT EXISTS (SELECT 1 FROM parcel_detail x WHERE x.holder_id = h.id)`,
		} {
			_, _ = pool.Exec(c, q)
		}
		_, _ = pool.Exec(c, `DELETE FROM acquisition_category WHERE id = ANY($1)`, catIDs)
	})
	before := tableCounts(t, ctx, pool)

	svc := NewService(pool, GusConfig{BaseURL: gus.URL, Username: "u", Password: "p", Delay: time.Millisecond})
	_, err = svc.Create(ctx, "e2e-val", "bogus")
	require.Error(t, err, "горим буруу бол үүсгэхгүй")
	id, err := svc.Create(ctx, "e2e-val", ModeValuation)
	require.NoError(t, err)

	// Хил, нэгж талбарын файл ХҮЛЭЭН АВАХГҮЙ.
	_, err = svc.UploadBoundary(ctx, id, files, CategoryChoice{})
	require.ErrorIs(t, err, errValuationMode)
	_, err = svc.UploadParcels(ctx, id, files)
	require.ErrorIs(t, err, errValuationMode)

	// ── Үнэлгээ ─────────────────────────────────────────────────────────────
	_, err = svc.UploadReports(ctx, id, files)
	require.NoError(t, err)
	svc.waitIdle()
	st, err := svc.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, ModeValuation, st.Session.Mode)
	v := st.Reports
	require.True(t, v.GusDone)
	require.True(t, v.CanProceed, "%+v", v.Projects)
	require.Equal(t, 180, v.Stats.Rows)
	require.Equal(t, 1, v.Stats.DB, "санд байгаа нэгж талбар алгасагдана")
	require.Equal(t, 1, v.Stats.Synthetic)
	require.Equal(t, 1, v.Stats.Duplicates)
	require.Equal(t, 1, v.Stats.Excel)
	require.Equal(t, 1, v.Stats.GusFromUb)
	require.Equal(t, 2, v.Stats.Projects, "том үсгээр бичсэн нэр нэг төсөлд нэгтгэгдэнэ")
	byName := map[string]ValuationProject{}
	for _, p := range v.Projects {
		byName[p.Name] = p
	}
	road := byName["Өөр авто зам 7"]
	require.Equal(t, categoryName{General: categoryRoad}, road.Category)
	require.Equal(t, 2, road.Parcels)
	require.Len(t, v.Projects, 2)
	for name, p := range byName {
		if name != road.Name {
			require.True(t, p.CategoryMatched, "«орон сууцжуулах» төрөл ангилалд таарна")
			require.Equal(t, categoryName{General: categoryGer}, p.Category)
			require.Equal(t, 177, p.Parcels, "176 − 1 (санд) + 2 («2015»-ийн D2, ERR)")
			require.Len(t, p.Variants, 2)
		}
	}
	require.True(t, st.Summary.CanCommit, "%v", st.Summary.Blockers)
	require.Len(t, st.Summary.Acquisitions, 2)

	// ГУС: ижил хүсэлт НЭГ удаа; санд байгаа ба дугааргүй мөрийг хайхгүй.
	mu.Lock()
	for p, n := range paths {
		require.Equal(t, 1, n, "давхар дуудлага: %s", p)
		require.NotContains(t, p, dbParcel, "санд байгаа нэгж талбарыг ГУС-аас хайхгүй")
	}
	gusCalls := len(paths)
	mu.Unlock()

	// ── Төслийн тохиргоо: ангилал (гараар), мэдээлэл, хил — бүгд заавал биш ──
	var main ValuationProject
	for name, p := range byName {
		if name != road.Name {
			main = p
		}
	}
	// Санд ижил нэртэй ангилал байж болно — нэрээр хайхад эхнийх нь.
	var gerID int
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM acquisition_category WHERE parent_id IS NULL AND name = $1 ORDER BY id LIMIT 1`,
		categoryGer).Scan(&gerID))
	ger := strconv.Itoa(gerID)
	require.Equal(t, &CategoryChoice{General: ger}, main.CategoryIDs, "автомат ангиллын ID сонгох талбарт")
	require.False(t, main.CategoryChosen)

	// Авто замын төслийг гараар «Гэр хороолол» руу.
	st, err = svc.UpdateProject(ctx, id, ProjectUpdate{Name: road.Name, Category: &CategoryChoice{General: ger}})
	require.NoError(t, err)
	road = projectByName(t, st.Reports, road.Name)
	require.True(t, road.CategoryChosen)
	require.Equal(t, categoryName{General: categoryGer}, road.Category)

	_, err = svc.UpdateProject(ctx, id, ProjectUpdate{Name: road.Name, Category: &CategoryChoice{General: "-1"}})
	require.Error(t, err, "санд байхгүй ангилал")
	_, err = svc.UpdateProject(ctx, id, ProjectUpdate{Name: main.Name, Info: &AcquisitionInfo{StartDate: "2015-13-01"}})
	require.Error(t, err, "буруу огноо")
	_, err = svc.UpdateProject(ctx, id, ProjectUpdate{Name: main.Name, Info: &AcquisitionInfo{StartDate: "2016-01-01", EndDate: "2015-01-01"}})
	require.Error(t, err, "дуусах < эхлэх")
	_, err = svc.UpdateProject(ctx, id, ProjectUpdate{Name: "байхгүй төсөл"})
	require.Error(t, err)

	info := AcquisitionInfo{ImplementingOrg: " НЗДТГ ", ResponsibleOrg: "ГЗБГЗЗГ", Reason: "Орон сууцжуулах",
		StartDate: "2015-01-01", EndDate: "2016-12-31", NithDecreeNumber: "15/12", GroupListNumber: "3"}
	st, err = svc.UpdateProject(ctx, id, ProjectUpdate{Name: main.Name, Info: &info})
	require.NoError(t, err)
	require.Equal(t, "НЗДТГ", st.ProjectSettings[main.Name].Info.ImplementingOrg, "зай арилна")
	require.NotNil(t, st.ProjectSettings[road.Name].Category, "өөр төслийн тохиргоо дарагдахгүй")
	require.True(t, projectByName(t, st.Reports, main.Name).HasInfo)

	// Хил: хоёр shapefile → алдаа; жишээ хил (UTM 48N, нэг полигон) → 4326.
	hil := loadDir(t, sampleDir(t, "чөлөөлөлт"))
	var twoSets []UploadedFile
	for _, f := range hil {
		twoSets = append(twoSets, f, UploadedFile{Name: "b" + f.Name, Data: f.Data})
	}
	_, err = svc.UploadProjectBoundary(ctx, id, main.Name, twoSets)
	require.Error(t, err)
	st, err = svc.UploadProjectBoundary(ctx, id, main.Name, hil)
	require.NoError(t, err)
	b := st.ProjectSettings[main.Name].Boundary
	require.NotNil(t, b)
	require.True(t, strings.HasPrefix(b.WKT, "POLYGON"), b.WKT[:min(40, len(b.WKT))])
	require.Greater(t, b.AreaM2, 0.0)
	require.Equal(t, b.AreaM2, *projectByName(t, st.Reports, main.Name).AreaM2)
	st, err = svc.RemoveProjectBoundary(ctx, id, main.Name)
	require.NoError(t, err)
	require.Nil(t, st.ProjectSettings[main.Name].Boundary)
	require.Nil(t, projectByName(t, st.Reports, main.Name).AreaM2)
	st, err = svc.UploadProjectBoundary(ctx, id, main.Name, hil)
	require.NoError(t, err)
	require.True(t, st.Summary.CanCommit, "%v", st.Summary.Blockers)
	mu.Lock()
	require.Equal(t, gusCalls, len(paths), "төслийн тохиргоо ГУС дуудахгүй")
	mu.Unlock()

	// ── Оруулах ─────────────────────────────────────────────────────────────
	require.NoError(t, svc.Commit(ctx, id))
	svc.waitIdle()
	st, err = svc.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "committed", st.Session.Status, "%+v", st.Result)
	require.Zero(t, st.Result.Log.Failed, "%+v", st.Result.Log.Entries)
	require.Equal(t, 2, st.Result.Acquisitions.Created)
	require.Equal(t, 179, st.Result.Parcels.Imported)
	require.Equal(t, 1, st.Result.Parcels.AlreadyImported)
	var want float64
	for i := range parsed {
		if parsed[i].ParcelID != dbParcel {
			want += parsed[i].amount()
		}
	}
	require.InDelta(t, want, st.Result.Compensations.TotalAmount, 1)

	acqs := ownedIDs(t, ctx, pool, id, "land_acquisition")
	require.Len(t, acqs, 2)
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = ANY($1::uuid[]) AND (plan_geom IS NOT NULL OR plan_id IS NOT NULL)`, acqs),
		"төлөвлөгөөгүй")
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = ANY($1::uuid[]) AND geometry IS NULL AND area_m2 IS NULL
		AND acquisition_name = 'Өөр авто зам 7' AND implementing_org = '' AND start_date IS NULL AND nith_decree_number <> ''`, acqs), "тохиргоогүй төсөл ХИЛГҮЙ, мэдээлэлгүй (НИТХ тогтоол эксэлээс)")
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = ANY($1::uuid[]) AND acquisition_name = $2
		AND GeometryType(geometry) = 'POLYGON' AND abs(area_m2 - $3) < 1
		AND implementing_org = 'НЗДТГ' AND responsible_org = 'ГЗБГЗЗГ' AND reason = 'Орон сууцжуулах'
		AND start_date = '2015-01-01' AND end_date = '2016-12-31' AND nith_decree_number = '15/12' AND group_list_number = '3'`,
		acqs, main.Name, b.AreaM2), "хил ба мэдээлэл чөлөөлөлтөд")
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition la JOIN acquisition_category c ON c.id = la.general_category_id
		WHERE la.id = ANY($1::uuid[]) AND la.acquisition_name = 'Өөр авто зам 7' AND c.name = $2`, acqs, categoryGer), "гараар сонгосон ангилал")
	require.Equal(t, count(t, ctx, pool, `SELECT count(*) FROM au3 WHERE geometry IS NOT NULL AND ST_Intersects(geometry, ST_GeomFromText($1, 4326))`, b.WKT),
		count(t, ctx, pool, `SELECT count(*) FROM land_acquisition_au WHERE land_acquisition_id = ANY($1::uuid[])`, acqs), "хилтэй огтлолцох баг/хороо")
	require.Equal(t, 179, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE acquisition_id = ANY($1::uuid[])`, acqs))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE parcel_id = 'D21802301540' AND acquisition_id = ANY($1::uuid[])`, acqs))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE parcel_id LIKE 'ERR%' AND acquisition_id = ANY($1::uuid[])`, acqs))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE parcel_id = '9999000001' AND geometry IS NULL`), "ГУС-д олдоогүй — хилгүй")
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id NOT IN ('9999000001')
		AND parcel_id NOT LIKE 'ERR%' AND acquisition_geom IS NULL`, acqs), "ГУС-ын хил газрын зурагт (acquisition_geom) харагдана")
	require.Equal(t, 4, count(t, ctx, pool, `SELECT count(*) FROM parcel_status_history WHERE acquisition_id = ANY($1::uuid[])
		AND status_date::date = '2015-12-31'`, acqs), "«2015» sheet-ийн мөрүүд тухайн оны огноотой")
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM parcel p WHERE p.acquisition_id = ANY($1::uuid[])
		AND p.status <> 109 AND EXISTS (SELECT 1 FROM compensation c WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id AND c.amount > 0)`, acqs),
		"олговортой нэгж талбар «Чөлөөлсөн»")
	// Олгосон хувь ба эхний явц — «зөвхөн үнэлгээ»-нд төлөвлөгөөгүй тул чөлөөлөлт үүсгэсэн огноо.
	assertPaidAndInitialHistory(t, ctx, pool, acqs, "")
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id = $1 AND acquisition_id = $2`, ctrlParcel, ctrlAcq),
		"санд байсан нэгж талбар хөндөгдөхгүй")

	// Жагсаалт: горим, төслийн тоо.
	items, err := svc.List(ctx)
	require.NoError(t, err)
	for _, it := range items {
		if it.ID == id {
			require.Equal(t, ModeValuation, it.Mode)
			require.Equal(t, 2, it.Acquisitions)
			require.True(t, it.Deletable)
		}
	}

	// ── Устгах — бүтэн горимтой ижил, бусад мэдээлэл хөндөгдөхгүй ───────────
	res, err := svc.Rollback(ctx, id, "e2e admin")
	require.NoError(t, err)
	require.Equal(t, 179, res.Counts["deleted:parcel"])
	require.Equal(t, 2, res.Counts["deleted:land_acquisition"])
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = ANY($1::uuid[])`, acqs))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE created_by = $1`, Actor))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE created_by = $1`, Actor))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id = $1`, ctrlParcel))
	after := tableCounts(t, ctx, pool)
	for table, n := range before {
		require.GreaterOrEqual(t, after[table], n, "%s: бусад мэдээлэл устсан байна", table)
		switch table {
		case "audit_log", "decision_work_type", "decision_budget":
		default:
			require.Equal(t, n, after[table], "%s: импортын мөр үлдсэн", table)
		}
	}
}
