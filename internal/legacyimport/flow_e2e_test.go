//go:build e2e

package legacyimport

import (
	"acquisition_backend/internal/seed"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// E2E: жишээ файлуудаар (government-legacy-import/input) бүтэн урсгал —
// хил → нэгж талбар → үнэлгээ → нэгтгэл → «Оруулах». ГУС-ыг mock-оор орлуулна.
//
//	E2E_DATABASE_URL=postgres://… go test -tags e2e ./internal/legacyimport/ -run E2E
func TestE2ELegacyImportFlow(t *testing.T) {
	dsn := os.Getenv("E2E_DATABASE_URL")
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	boundaryFiles := loadDir(t, sampleDir(t, "чөлөөлөлт"))
	parcelFiles := loadDir(t, sampleDir(t, "нэгж-талбар"))
	reportFiles := loadDir(t, sampleDir(t, "үнэлгээ"))

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close) // t.Cleanup LIFO — цэвэрлэгээний дараа хаагдана

	// ── ГУС mock ────────────────────────────────────────────────────────────
	sets := groupShapefiles(parcelFiles)
	features, _, err := readShapefileSet(sets[0])
	require.NoError(t, err)
	geomByID := map[string]string{}
	var ids []string
	for _, f := range features {
		id := parcelKey(attr(f.props, parcelIDFields...))
		if _, dup := geomByID[id]; !dup {
			ids = append(ids, id)
		}
		geomByID[id] = f.wkt
	}
	missing, inUb := map[string]bool{}, map[string]bool{}
	for i, id := range ids {
		if i%10 == 9 {
			missing[id] = true
			if len(missing)%2 == 1 {
				inUb[id] = true
			}
		}
	}
	var gusCalls atomic.Int64
	planFound := atomic.Bool{}
	var pathsMu sync.Mutex
	paths := map[string]int{} // хүсэлт бүрийн тоо — давхар дуудлага илрүүлнэ
	gus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gusCalls.Add(1)
		pathsMu.Lock()
		paths[r.URL.RequestURI()]++
		pathsMu.Unlock()
		require.Equal(t, "gus-user", r.Header.Get("X-Middleware-Username"))
		send := func(data any) {
			if data == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		}
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		switch {
		case r.URL.Path == "/plan/project":
			if !planFound.Load() {
				send(nil)
				return
			}
			send(map[string]any{"code": "2026-04-011-9878", "parcel_id": "4932803", "name": "Тест төлөвлөгөө",
				"gazner": "Ханын материал орчим", "approved_date": "2025-12-05", "geometry_wkt": "POLYGON((106.7 47.85,106.95 47.85,106.95 48,106.7 48,106.7 47.85))"})
		case strings.HasPrefix(r.URL.Path, "/parcel/info/") && geomByID[id] != "" && !missing[id]:
			send(map[string]any{"parcel_id": id, "geometry_wkt": geomByID[id], "area_m2": 650, "right_type": 3, "app_no": "APP-" + id})
		case strings.HasPrefix(r.URL.Path, "/parcel/ub/") && inUb[id]:
			send(map[string]any{"parcel_id": id, "geometry_wkt": geomByID[id], "area_m2": 650})
		default:
			send(nil)
		}
	}))
	defer gus.Close()

	// ── Лавлах өгөгдөл ──────────────────────────────────────────────────────
	var generalID, subID int
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO acquisition_category(name,parent_id,sort_order)
		VALUES('E2E импорт', NULL, 99) RETURNING id`).Scan(&generalID))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO acquisition_category(name,parent_id,sort_order)
		VALUES('E2E дэд', $1, 1) RETURNING id`, generalID).Scan(&subID))
	// Системийн төлөв (101–109) — импорт кодыг эдгээртэй тулгана.
	require.NoError(t, seed.NewMainSeeder(pool).SyncParcelStatusWorkflow(ctx))
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, q := range []string{
			`DELETE FROM decision_draft WHERE created_by = 'legacy-import' OR proposal_no = 'E2E-CTRL'`,
			`DELETE FROM land_acquisition WHERE created_by IN ($1, 'e2e-other')
			   OR legacy_import_id IN (SELECT id FROM legacy_import_stage.session WHERE created_by = 'e2e')`,
			`DELETE FROM legacy_import_stage.rollback_log WHERE session_id IN (SELECT id FROM legacy_import_stage.session WHERE created_by = 'e2e')`,
			`DELETE FROM legacy_import_stage.owned WHERE session_id IN (SELECT id FROM legacy_import_stage.session WHERE created_by = 'e2e')`,
			`DELETE FROM legacy_import_stage.session WHERE created_by = 'e2e'`,
			`DELETE FROM plan WHERE plan_code IN ('E2E-OTHER-PLAN', '2026-04-011-9878')`,
			`DELETE FROM right_holder h WHERE NOT EXISTS (SELECT 1 FROM parcel_holder x WHERE x.holder_id = h.id)
			   AND NOT EXISTS (SELECT 1 FROM parcel_detail x WHERE x.holder_id = h.id)`,
			`DELETE FROM document WHERE name LIKE 'E2E%'`,
			`DELETE FROM notification WHERE title LIKE 'E2E%'`,
		} {
			if strings.Contains(q, "$1") {
				_, _ = pool.Exec(c, q, Actor)
			} else {
				_, _ = pool.Exec(c, q)
			}
		}
		_, _ = pool.Exec(c, `DELETE FROM acquisition_category WHERE id IN ($1, $2)`, subID, generalID)
	})
	systemRows := func() int {
		var n int
		require.NoError(t, pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM land_acquisition WHERE created_by=$1)
			+ (SELECT count(*) FROM parcel WHERE created_by=$1) + (SELECT count(*) FROM compensation WHERE created_by=$1)`, Actor).Scan(&n))
		return n
	}
	require.Zero(t, systemRows(), "өмнөх гүйлтийн импорт үлдсэн байна")

	// ── Хяналтын (ӨӨР) мэдээлэл — «Хасах» үед огт хөндөгдөх ёсгүй ─────────────
	// ids[0] — хуучин файлд байгаа ч санд ӨӨР чөлөөлөлтөд бүртгэлтэй нэгж талбар.
	preexisting := ids[0]
	var ctrlPlan, ctrlAcq, ctrlParcel, ctrlParcel2, ctrlHolder, ctrlDraft string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO plan(id, plan_code) VALUES(gen_random_uuid(), 'E2E-OTHER-PLAN') RETURNING id`).Scan(&ctrlPlan))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition(plan_code, plan_id, acquisition_name, created_by)
		VALUES('E2E-OTHER-PLAN', $1, 'E2E бусад чөлөөлөлт', 'e2e-other') RETURNING id`, ctrlPlan).Scan(&ctrlAcq))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel(acquisition_id, parcel_id, created_by) VALUES($1, $2, 'e2e-other') RETURNING id`,
		ctrlAcq, preexisting).Scan(&ctrlParcel))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel(acquisition_id, parcel_id, created_by) VALUES($1, 'E2E-OTHER-2', 'e2e-other') RETURNING id`,
		ctrlAcq).Scan(&ctrlParcel2))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO right_holder(name, register_no) VALUES('E2E бусад эзэмшигч', 'E2E-RH-OTHER') RETURNING id`).Scan(&ctrlHolder))
	_, err = pool.Exec(ctx, `INSERT INTO parcel_holder(parcel_id, holder_id, acquisition_id) VALUES($1, $2, $3)`, ctrlParcel2, ctrlHolder, ctrlAcq)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO compensation(acquisition_id, parcel_id, amount, created_by) VALUES($1, $2, 1000, 'e2e-other')`, ctrlAcq, preexisting)
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO decision_draft(proposal_no, acquisition_id) VALUES('E2E-CTRL', $1) RETURNING id`, ctrlAcq).Scan(&ctrlDraft))
	_, err = pool.Exec(ctx, `INSERT INTO decision_draft_parcel(decision_draft_id, parcel_id) VALUES($1, $2)`, ctrlDraft, ctrlParcel2)
	require.NoError(t, err)

	svc := NewService(pool, GusConfig{BaseURL: gus.URL, Username: "gus-user", Password: "gus-pass", Delay: time.Millisecond})
	id, err := svc.Create(ctx, "e2e", ModeFull)
	require.NoError(t, err)
	choice := CategoryChoice{General: strconv.Itoa(generalID), Sub: strconv.Itoa(subID)}

	// 1a. Төлөвлөгөө олдоогүй — цааш явуулахгүй.
	st, err := svc.UploadBoundary(ctx, id, boundaryFiles, choice)
	require.NoError(t, err)
	require.False(t, st.Boundary.CanProceed)
	require.True(t, hasCode(st.Boundary.Rows[0].Issues, "plan_not_found"))
	_, err = svc.UploadParcels(ctx, id, parcelFiles)
	var step *StepError
	require.ErrorAs(t, err, &step, "хил алдаатай бол нэгж талбар руу шилжихгүй")

	// 1b. Буруу ангилал.
	planFound.Store(true)
	st, err = svc.UploadBoundary(ctx, id, boundaryFiles, CategoryChoice{General: "999999"})
	require.NoError(t, err)
	require.False(t, st.Boundary.CanProceed)

	// 1c. Зөв — UTM 48N хил EPSG:4326 руу хөрвөж, төлөвлөгөөний дотор.
	st, err = svc.UploadBoundary(ctx, id, boundaryFiles, choice)
	require.NoError(t, err)
	require.True(t, st.Boundary.CanProceed, "%+v", st.Boundary.Rows)
	require.Equal(t, 100.0, *st.Boundary.Rows[0].InsidePlanPct)
	require.True(t, strings.HasPrefix(st.Boundary.Rows[0].Name, "4932803 "))

	// Хилийн гадна нэгж талбар гаргахын тулд түр хилийг хоёр дахин жижигрүүлнэ.
	_, err = pool.Exec(ctx, `UPDATE legacy_import_stage.boundary SET geom = ST_Intersection(geom,
		ST_MakeEnvelope(ST_XMin(geom), ST_YMin(geom), (ST_XMin(geom)+ST_XMax(geom))/2, ST_YMax(geom), 4326)) WHERE session_id=$1`, id)
	require.NoError(t, err)

	// 2. Нэгж талбар.
	_, err = svc.UploadParcels(ctx, id, parcelFiles)
	require.NoError(t, err)
	svc.waitIdle()
	st, err = svc.Get(ctx, id)
	require.NoError(t, err)
	p := st.Parcels
	require.True(t, p.GusDone)
	require.Equal(t, 177, p.Stats.Total)
	require.Equal(t, 1, p.Stats.Duplicates)
	require.Positive(t, p.Stats.Outside, "хилийн гаднах нэгж талбар илэрнэ")
	require.Equal(t, len(inUb), p.Stats.Gus["ub"])
	require.Equal(t, len(missing)-len(inUb), p.Stats.Gus["missing"])
	// Нэгж талбарын шалгалт: ижил хүсэлт НЭГ л удаа; ГУС-д олдсон нэгж талбар
	// НЭГ хүсэлт (info), олдоогүй нь info → ub (хуучин дугааргүй бол 2).
	pathsMu.Lock()
	perParcel := map[string]int{}
	for path, n := range paths {
		if strings.HasPrefix(path, "/parcel/") {
			require.Equal(t, 1, n, "давхар дуудлага: %s", path)
			id := strings.SplitN(path[strings.LastIndex(path, "/")+1:], "?", 2)[0]
			perParcel[id] += n
		}
	}
	pathsMu.Unlock()
	for _, pid := range ids {
		if pid == preexisting {
			require.Zero(t, perParcel[pid], "санд байгаа нэгж талбарыг ГУС-аас хайхгүй")
		} else if !missing[pid] {
			require.Equal(t, 1, perParcel[pid], "ГУС-д олдсон %s", pid)
		}
	}

	// 3. Үнэлгээ.
	_, err = svc.UploadReports(ctx, id, reportFiles)
	require.NoError(t, err)
	svc.waitIdle()
	st, err = svc.Get(ctx, id)
	require.NoError(t, err)
	require.True(t, st.Reports.CanProceed)
	require.Equal(t, 176, st.Reports.Stats.Rows)
	require.Equal(t, 176, st.Reports.Stats.Staged)
	require.InDelta(t, 35979437529, st.Reports.Stats.TotalAmount, 1)

	// 4. Нэгтгэл — «Оруулах»-аас өмнө системийн хүснэгтэд юу ч бичигдээгүй.
	require.True(t, st.Summary.CanCommit, "%v", st.Summary.Blockers)
	require.Len(t, st.Summary.Acquisitions, 1)
	require.Zero(t, systemRows())

	before0 := tableCounts(t, ctx, pool)

	// 5. Оруулах — ГУС-ыг дахин дуудахгүй.
	before := gusCalls.Load()
	require.NoError(t, svc.Commit(ctx, id))
	svc.waitIdle()
	require.Equal(t, before, gusCalls.Load(), "шалгалтын үед татсан хариуг хэрэглэнэ")
	st, err = svc.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "committed", st.Session.Status, "%+v", st.Result)
	require.Zero(t, st.Result.Log.Failed, "%+v", st.Result.Log.Entries)
	require.Equal(t, 175, st.Result.Parcels.Imported, "санд байсан нэгж талбарыг алгасна")
	require.Equal(t, 1, st.Result.Parcels.AlreadyImported)
	t.Logf("олговор: applied=%d total=%.0f", st.Result.Compensations.Applied, st.Result.Compensations.TotalAmount)

	var acq, parcels, noBase, staged int
	var amount float64
	require.NoError(t, pool.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM land_acquisition WHERE created_by=$1),
			(SELECT count(*) FROM parcel WHERE created_by=$1),
			(SELECT count(*) FROM parcel WHERE created_by=$1 AND geometry IS NULL),
			(SELECT COALESCE(sum(amount),0)::float8 FROM compensation WHERE created_by=$1),
			(SELECT count(*) FROM legacy_import_stage.parcel WHERE session_id=$2)`, Actor, id).
		Scan(&acq, &parcels, &noBase, &amount, &staged))
	require.Equal(t, 1, acq)
	require.Equal(t, 175, parcels)
	require.Equal(t, len(missing)-len(inUb), noBase, "ГУС-д олдоогүй нэгж талбар үндсэн хилгүй")
	for _, e := range st.Result.Log.Entries {
		if e.Key == preexisting {
			t.Logf("санд байсан нэгж талбар: %s/%s %s", e.Level, e.Stage, e.Message)
		}
	}
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE parcel_id = $1 AND created_by = $2`, preexisting, Actor),
		"санд байсан нэгж талбарт олговор оруулахгүй")
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE parcel_id = $1`, preexisting), "өөрийнх нь олговор хэвээр")
	require.Zero(t, staged, "түр мөрүүд цэвэрлэгдэнэ")
	var ctrlParcelAcq string
	require.NoError(t, pool.QueryRow(ctx, `SELECT acquisition_id::text FROM parcel WHERE parcel_id = $1`, preexisting).Scan(&ctrlParcelAcq))
	require.Equal(t, ctrlAcq, ctrlParcelAcq, "санд байсан нэгж талбар өөр чөлөөлөлтөд хэвээр")
	// Захирамжийн төсөл: дугаар «ОН/дугаар» (legacy/ угтваргүй), төсөв тодорхойгүй бол «Нийслэл».
	drafts := count(t, ctx, pool, `SELECT count(*) FROM decision_draft WHERE created_by = $1`, Actor)
	require.Positive(t, drafts, "импорт захирамж үүсгэнэ")
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM decision_draft WHERE created_by = $1 AND proposal_no !~ '^[0-9]{4}/'`, Actor),
		"захирамжийн дугаар legacy/ угтваргүй")
	require.Equal(t, drafts, count(t, ctx, pool, `SELECT count(*) FROM decision_draft d JOIN decision_budget b ON b.id = d.budget_id
		WHERE d.created_by = $1 AND b.name = 'Нийслэл'`, Actor), "санхүүжилтийн төрөл тодорхойгүй бол Нийслэл")
	// Олгосон хувь ба эхний явц — төлөвлөгөө (ГУС) 2025-12-05-нд батлагдсан.
	assertPaidAndInitialHistory(t, ctx, pool, ownedIDs(t, ctx, pool, id, "land_acquisition"), "2025-12-05")

	_, err = svc.UploadParcels(ctx, id, parcelFiles)
	require.ErrorIs(t, err, ErrCommitted)
	// Оруулсан импорт — зөвхөн харна: дахин оруулах, устгах боломжгүй.
	require.ErrorIs(t, svc.Commit(ctx, id), ErrCommitted)
	require.ErrorIs(t, svc.Delete(ctx, id), ErrCommitted)
	_, err = svc.Get(ctx, id)
	require.NoError(t, err, "түүх хэвээр үлдэнэ")

	testRollback(t, ctx, pool, svc, id, rollbackControl{acq: ctrlAcq, plan: ctrlPlan, parcel: ctrlParcel,
		parcel2: ctrlParcel2, holder: ctrlHolder, draft: ctrlDraft}, before0)
}
