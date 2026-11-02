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
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// rollbackControl — импортоос ГАДУУРХ мэдээлэл (TestE2ELegacyImportFlow үүсгэнэ).
type rollbackControl struct {
	acq, plan, parcel, parcel2, holder, draft string
}

// tableCounts — public схемийн хүснэгт бүрийн мөрийн тоо.
func tableCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool) map[string]int64 {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY 1`)
	require.NoError(t, err)
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	out := map[string]int64{}
	for _, n := range names {
		var c int64
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM public.`+pgx.Identifier{n}.Sanitize()).Scan(&c))
		out[n] = c
	}
	return out
}

func count(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx, sql, args...).Scan(&n))
	return n
}

func execSQL(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	_, err := pool.Exec(ctx, sql, args...)
	require.NoError(t, err, sql)
}

func ownedIDs(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, kind string) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT ref_id::text FROM legacy_import_stage.owned WHERE session_id = $1 AND kind = $2 ORDER BY label`, id, kind)
	require.NoError(t, err)
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return out
}

// findLog — тухайн алхам/төрөл/үйлдэл/шошготой бичлэг.
func findLog(entries []RollbackLogEntry, step, kind, action, label string) *RollbackLogEntry {
	for i := range entries {
		e := entries[i]
		if e.Step == step && e.Kind == kind && e.Action == action && (label == "" || e.Label == label) {
			return &entries[i]
		}
	}
	return nil
}

func countLog(entries []RollbackLogEntry, step, kind, action string) int {
	n := 0
	for _, e := range entries {
		if e.Step == step && e.Kind == kind && e.Action == action {
			n++
		}
	}
	return n
}

// testRollback — БҮРЭН устгал: импортын бүртгэлийн мөр бүр устана, өөр газраас
// устсан нь логт «алгассан» гэж бичигдэнэ; бусад мэдээлэл НЭГ Ч мөр хөндөгдөхгүй.
func testRollback(t *testing.T, ctx context.Context, pool *pgxpool.Pool, svc *Service, id string, ctrl rollbackControl, before map[string]int64) {
	acqs := ownedIDs(t, ctx, pool, id, "land_acquisition")
	require.Len(t, acqs, 1, "импортын бүртгэлд чөлөөлөлт")
	ourAcq := acqs[0]
	parcels := ownedIDs(t, ctx, pool, id, "parcel")
	require.Len(t, parcels, 175, "импортын бүртгэлд нэгж талбар бүр")
	require.NotEmpty(t, ownedIDs(t, ctx, pool, id, "valuation"))
	code := func(parcelID string) string {
		var c string
		require.NoError(t, pool.QueryRow(ctx, `SELECT label FROM legacy_import_stage.owned WHERE session_id = $1 AND kind = 'parcel' AND ref_id = $2`, id, parcelID).Scan(&c))
		return c
	}
	var valued []string
	require.NoError(t, collectStrings(ctx, pool, &valued, `SELECT o.ref_id::text FROM legacy_import_stage.owned o
		WHERE o.session_id = $1 AND o.kind = 'valuation' ORDER BY o.label`, id))
	require.GreaterOrEqual(t, len(valued), 3)

	// ── Өөр газраас аль хэдийн устгасан мэдээлэл ────────────────────────────
	goneParcel, goneValuation, humanOn := valued[0], valued[1], valued[2]
	execSQL(t, ctx, pool, `DELETE FROM compensation WHERE acquisition_id = $1 AND parcel_id = $2`, ourAcq, code(goneParcel))
	execSQL(t, ctx, pool, `DELETE FROM parcel WHERE id = $1`, goneParcel)
	execSQL(t, ctx, pool, `DELETE FROM compensation WHERE acquisition_id = $1 AND parcel_id = $2`, ourAcq, code(goneValuation))
	execSQL(t, ctx, pool, `DELETE FROM assets WHERE acquisition_id = $1 AND parcel_id = $2`, ourAcq, code(goneValuation))
	execSQL(t, ctx, pool, `DELETE FROM land_valuation WHERE acquisition_id = $1 AND parcel_id = $2`, ourAcq, code(goneValuation))
	execSQL(t, ctx, pool, `DELETE FROM parcel_valuation_submission WHERE acquisition_id = $1 AND parcel_id = $2`, ourAcq, code(goneValuation))
	execSQL(t, ctx, pool, `DELETE FROM parcel_valuation_submission_history WHERE acquisition_id = $1 AND parcel_id = $2`, ourAcq, code(goneValuation))

	// ── Импортын нэгж талбар/чөлөөлөлтөд ДАРАА нь хүн нэмсэн (мөн устана) ──
	execSQL(t, ctx, pool, `UPDATE land_acquisition SET acquisition_name = acquisition_name || ' (засав)' WHERE id = $1`, ourAcq)
	execSQL(t, ctx, pool, `INSERT INTO compensation(acquisition_id, parcel_id, amount, created_by) VALUES($1, $2, 5, 'human')`, ourAcq, code(humanOn))
	execSQL(t, ctx, pool, `INSERT INTO notification(id, recipient_id, title, type, parcel_id) VALUES(gen_random_uuid(), 1, 'E2E манай нэгж', 'info', $1)`, humanOn)
	execSQL(t, ctx, pool, `INSERT INTO notification(id, recipient_id, title, type, acquisition_id) VALUES(gen_random_uuid(), 1, 'E2E манай', 'info', $1)`, ourAcq)
	execSQL(t, ctx, pool, `INSERT INTO notification(id, recipient_id, title, type, acquisition_id) VALUES(gen_random_uuid(), 1, 'E2E бусад', 'info', $1)`, ctrl.acq)
	var docParcel, docAcq, docShared string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO document(name, file_type, file_url) VALUES('E2E нэгжийн баримт', 'pdf', 'e2e://p.pdf') RETURNING id`).Scan(&docParcel))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO document(name, file_type, file_url) VALUES('E2E чөлөөлөлтийн баримт', 'pdf', 'e2e://a.pdf') RETURNING id`).Scan(&docAcq))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO document(name, file_type, file_url) VALUES('E2E дундын баримт', 'pdf', 'e2e://s.pdf') RETURNING id`).Scan(&docShared))
	execSQL(t, ctx, pool, `INSERT INTO parcel_document(parcel_id, document_id) VALUES($1, $2)`, humanOn, docParcel)
	execSQL(t, ctx, pool, `INSERT INTO land_acquisition_document(land_acquisition_id, document_id) VALUES($1, $2), ($1, $3)`, ourAcq, docAcq, docShared)
	execSQL(t, ctx, pool, `INSERT INTO land_acquisition_document(land_acquisition_id, document_id) VALUES($1, $2)`, ctrl.acq, docShared)

	// Импорт үүсгэсэн эзэмшигч/захирамжийг ӨӨР нэгж талбарт холбовол үлдэнэ.
	var keptHolder, keptDraft string
	if h := ownedIDs(t, ctx, pool, id, "right_holder"); len(h) > 0 {
		keptHolder = h[0]
		execSQL(t, ctx, pool, `INSERT INTO parcel_holder(parcel_id, holder_id, acquisition_id) VALUES($1, $2, $3)`, ctrl.parcel, keptHolder, ctrl.acq)
	}
	if d := ownedIDs(t, ctx, pool, id, "decision_draft"); len(d) > 0 {
		keptDraft = d[0]
		execSQL(t, ctx, pool, `INSERT INTO decision_draft_parcel(decision_draft_id, parcel_id, linked_by) VALUES($1, $2, 'human')`, keptDraft, ctrl.parcel)
	}
	t.Logf("бүртгэл: эзэмшигч=%v захирамж=%v", keptHolder != "", keptDraft != "")

	listItem := func() SessionListItem {
		t.Helper()
		items, _, err := svc.List(ctx, ListFilter{PageSize: maxListPageSize})
		require.NoError(t, err)
		for _, it := range items {
			if it.ID == id {
				return it
			}
		}
		t.Fatalf("жагсаалтад %s алга", id)
		return SessionListItem{}
	}
	require.True(t, listItem().Deletable, "оруулсан импорт устгах боломжтой")

	// ── Урьдчилсан харагдац ────────────────────────────────────────────────
	pv, err := svc.RollbackPreview(ctx, id)
	require.NoError(t, err)
	require.True(t, pv.CanRollback, "%v", pv.Blockers)
	require.Equal(t, RollbackCount{Total: 175, Exists: 174, Missing: 1}, pv.Parcels)
	require.Equal(t, 2, pv.Valuations.Missing, "өөр газраас устсан 2 үнэлгээ")
	require.Len(t, pv.Acquisitions, 1)
	require.False(t, pv.Acquisitions[0].Keep, "%s", pv.Acquisitions[0].KeepReason)
	require.Equal(t, 1, pv.AddedLater["compensations"])
	t.Logf("preview: %+v", pv)

	// ── Устгах ──────────────────────────────────────────────────────────────
	res, err := svc.Rollback(ctx, id, "e2e admin")
	require.NoError(t, err)
	require.Equal(t, 174, res.Counts["deleted:parcel"])
	require.Equal(t, 1, res.Counts["skipped:parcel"])
	require.Equal(t, 1, res.Counts["deleted:land_acquisition"])
	require.Len(t, res.Acquisitions, 1)
	require.Empty(t, res.Kept)

	st, err := svc.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", st.Session.Status)
	require.NotNil(t, st.Session.Rollback)
	require.Equal(t, "e2e admin", st.Session.Rollback.By)
	log := st.RollbackLog
	require.NotEmpty(t, log, "устгалын лог заавал бичигдэнэ")
	require.NotNil(t, findLog(log, "parcel", "parcel", "skipped", code(goneParcel)), "өөр газраас устсан нэгж талбар алгасагдана")
	require.NotNil(t, findLog(log, "valuation", "valuation", "skipped", code(goneParcel)))
	require.NotNil(t, findLog(log, "valuation", "valuation", "skipped", code(goneValuation)), "өөр газраас устсан үнэлгээ алгасагдана")
	require.NotNil(t, findLog(log, "valuation", "valuation", "deleted", code(humanOn)))
	require.Contains(t, findLog(log, "valuation", "valuation", "deleted", code(humanOn)).Message, "импортын дараа нэмсэн олговор 1")
	require.Equal(t, 174, countLog(log, "parcel", "parcel", "deleted"))
	require.NotNil(t, findLog(log, "acquisition", "land_acquisition", "deleted", ""))
	require.NotNil(t, findLog(log, "parcel", "document", "deleted", "E2E нэгжийн баримт"))
	require.NotNil(t, findLog(log, "acquisition", "document", "deleted", "E2E чөлөөлөлтийн баримт"))
	require.Nil(t, findLog(log, "acquisition", "document", "deleted", "E2E дундын баримт"))
	if countLog(log, "valuation", "decision_link", "detached") == 0 {
		t.Log("захирамжийн холбоос үүсээгүй")
	}
	// Лог ба алхмын дараалал: үнэлгээ → нэгж талбар → чөлөөлөлт.
	order := map[string]int{"valuation": 1, "parcel": 2, "acquisition": 3, "file": 4}
	for i := 1; i < len(log); i++ {
		require.LessOrEqual(t, order[log[i-1].Step], order[log[i].Step], "алхмын дараалал: %s → %s", log[i-1].Step, log[i].Step)
	}
	item := listItem()
	require.Equal(t, "rolled_back", item.Status)
	require.False(t, item.Deletable)
	require.NotNil(t, item.Rollback)
	require.ErrorIs(t, svc.Delete(ctx, id), ErrCommitted, "устгасан импортын түүх үлдэнэ")
	require.Error(t, svc.Commit(ctx, id), "устгасан импортыг дахин оруулахгүй")
	_, err = svc.Rollback(ctx, id, "e2e admin")
	require.Error(t, err, "дахин устгахгүй")

	// Импортын бүх мэдээлэл устсан.
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = $1`, ourAcq))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id = ANY($1::uuid[]) OR created_by = $2`, parcels, Actor))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE acquisition_id = $1 OR created_by = $2`, ourAcq, Actor))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM notification WHERE acquisition_id = $1 OR parcel_id = $2`, ourAcq, humanOn))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM document WHERE id IN ($1, $2)`, docParcel, docAcq))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM plan WHERE id = ANY($1::uuid[])`, ownedIDs(t, ctx, pool, id, "plan")))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM funding_sources WHERE id = ANY($1::uuid[])`, ownedIDs(t, ctx, pool, id, "funding_source")))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM right_holder WHERE id = ANY($1::uuid[]) AND id::text IS DISTINCT FROM $2`,
		ownedIDs(t, ctx, pool, id, "right_holder"), keptHolder))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM decision_draft WHERE id = ANY($1::uuid[]) AND id::text IS DISTINCT FROM $2`,
		ownedIDs(t, ctx, pool, id, "decision_draft"), keptDraft))

	// Бусад мэдээлэл хэвээр.
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = $1 AND acquisition_name = 'E2E бусад чөлөөлөлт'`, ctrl.acq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM plan WHERE id = $1`, ctrl.plan))
	require.Equal(t, 2, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id IN ($1, $2) AND acquisition_id = $3`, ctrl.parcel, ctrl.parcel2, ctrl.acq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE acquisition_id = $1`, ctrl.acq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM right_holder WHERE id = $1`, ctrl.holder))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM decision_draft_parcel WHERE decision_draft_id = $1 AND parcel_id = $2`, ctrl.draft, ctrl.parcel2))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM notification WHERE acquisition_id = $1`, ctrl.acq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM document WHERE id = $1`, docShared), "өөр чөлөөлөлттэй дундын баримт үлдэнэ")
	if keptHolder != "" {
		require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM right_holder WHERE id = $1`, keptHolder))
		require.NotNil(t, findLog(log, "parcel", "right_holder", "kept", ""))
	}
	if keptDraft != "" {
		require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM decision_draft WHERE id = $1 AND acquisition_id IS NULL`, keptDraft))
		require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM decision_draft_parcel WHERE decision_draft_id = $1 AND parcel_id = $2`, keptDraft, ctrl.parcel))
		require.NotNil(t, findLog(log, "valuation", "decision_draft", "kept", ""))
	}

	// Хүснэгт бүр «Оруулах»-аас өмнөхөөс ЦӨӨРӨХГҮЙ (бусад мэдээлэл устаагүй).
	expected := map[string]int64{"document": 1, "land_acquisition_document": 1, "notification": 1}
	if keptHolder != "" {
		expected["right_holder"], expected["parcel_holder"] = 1, 1
	}
	if keptDraft != "" {
		expected["decision_draft"], expected["decision_draft_parcel"] = 1, 1
	}
	after := tableCounts(t, ctx, pool)
	for table, n := range before {
		require.GreaterOrEqual(t, after[table], n, "%s: бусад мэдээлэл устсан байна", table)
		if diff := after[table] - n; diff != expected[table] {
			switch table {
			case "audit_log", "decision_work_type", "decision_budget":
				t.Logf("%s: +%d (лавлах/түүх)", table, diff)
			default:
				t.Errorf("%s: %d → %d (хүлээсэн өсөлт %d)", table, n, after[table], expected[table])
			}
		}
	}
}

// ── Чөлөөлөлтөд импортоос ӨӨР нэгж талбар орсон бол чөлөөлөлт ҮЛДЭНЭ ─────

func TestE2ELegacyImportRollbackKeepsForeignParcels(t *testing.T) {
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
	t.Cleanup(pool.Close)

	sets := groupShapefiles(parcelFiles)
	features, _, err := readShapefileSet(sets[0])
	require.NoError(t, err)
	geomByID := map[string]string{}
	for _, f := range features {
		geomByID[parcelKey(attr(f.props, parcelIDFields...))] = f.wkt
	}
	gus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		switch {
		case r.URL.Path == "/plan/project":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"code": "2026-04-011-9878", "parcel_id": "4932803",
				"name": "Тест төлөвлөгөө", "gazner": "Ханын материал орчим",
				"geometry_wkt": "POLYGON((106.7 47.85,106.95 47.85,106.95 48,106.7 48,106.7 47.85))"}})
		case strings.HasPrefix(r.URL.Path, "/parcel/info/") && geomByID[id] != "":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"parcel_id": id, "geometry_wkt": geomByID[id], "area_m2": 650}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer gus.Close()

	var generalID int
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO acquisition_category(name,parent_id,sort_order)
		VALUES('E2E үлдээх', NULL, 97) RETURNING id`).Scan(&generalID))
	// Системийн төлөв (101–109) — импорт кодыг эдгээртэй тулгана.
	require.NoError(t, seed.NewMainSeeder(pool).SyncParcelStatusWorkflow(ctx))
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sessions := `SELECT id FROM legacy_import_stage.session WHERE created_by = 'e2e-keep'`
		for _, q := range []string{
			`DELETE FROM decision_draft WHERE created_by = 'legacy-import'`,
			`DELETE FROM land_acquisition WHERE created_by IN ('legacy-import', 'e2e-other') OR legacy_import_id IN (` + sessions + `)`,
			`DELETE FROM legacy_import_stage.rollback_log WHERE session_id IN (` + sessions + `)`,
			`DELETE FROM legacy_import_stage.owned WHERE session_id IN (` + sessions + `)`,
			`DELETE FROM legacy_import_stage.session WHERE created_by = 'e2e-keep'`,
			`DELETE FROM plan WHERE plan_code IN ('E2E-OTHER-KEEP', '2026-04-011-9878')`,
			`DELETE FROM right_holder h WHERE NOT EXISTS (SELECT 1 FROM parcel_holder x WHERE x.holder_id = h.id)
			   AND NOT EXISTS (SELECT 1 FROM parcel_detail x WHERE x.holder_id = h.id)`,
		} {
			_, _ = pool.Exec(c, q)
		}
		_, _ = pool.Exec(c, `DELETE FROM acquisition_category WHERE id = $1`, generalID)
	})

	svc := NewService(pool, GusConfig{BaseURL: gus.URL, Username: "u", Password: "p", Delay: time.Millisecond})
	id, err := svc.Create(ctx, "e2e-keep", "")
	require.NoError(t, err)
	_, err = svc.UploadBoundary(ctx, id, boundaryFiles, CategoryChoice{General: strconv.Itoa(generalID)})
	require.NoError(t, err)
	_, err = svc.UploadParcels(ctx, id, parcelFiles)
	require.NoError(t, err)
	svc.waitIdle()
	_, err = svc.UploadReports(ctx, id, reportFiles)
	require.NoError(t, err)
	svc.waitIdle()
	require.NoError(t, svc.Commit(ctx, id))
	svc.waitIdle()
	st, err := svc.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "committed", st.Session.Status, "%+v", st.Result)
	ourAcq := ownedIDs(t, ctx, pool, id, "land_acquisition")[0]
	ourParcels := ownedIDs(t, ctx, pool, id, "parcel")
	require.Len(t, ourParcels, 176)

	// Хүн импортын чөлөөлөлтөд нэгж талбар нэмсэн.
	var humanParcel string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel(acquisition_id, parcel_id, created_by) VALUES($1, 'E2E-HUMAN-K', 'human') RETURNING id`,
		ourAcq).Scan(&humanParcel))
	execSQL(t, ctx, pool, `INSERT INTO compensation(acquisition_id, parcel_id, amount, created_by) VALUES($1, 'E2E-HUMAN-K', 77, 'human')`, ourAcq)
	// Өөр чөлөөлөлтийн нэгж талбар импортын санхүүжилтийн холбоосыг хэрэглэж байна.
	var otherAcq, otherParcel, ourLink string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition(plan_code, acquisition_name, created_by)
		VALUES('E2E-OTHER-KEEP', 'E2E бусад', 'e2e-other') RETURNING id`).Scan(&otherAcq))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel(acquisition_id, parcel_id, created_by) VALUES($1, 'E2E-OTHER-K', 'e2e-other') RETURNING id`,
		otherAcq).Scan(&otherParcel))
	if links := ownedIDs(t, ctx, pool, id, "funding_link"); len(links) > 0 {
		ourLink = links[0]
		var draft string
		require.NoError(t, pool.QueryRow(ctx, `SELECT decision_draft_id::text FROM decision_draft_funding_source WHERE id = $1`, ourLink).Scan(&draft))
		execSQL(t, ctx, pool, `INSERT INTO decision_draft_parcel(decision_draft_id, parcel_id, funding_link_id, linked_by)
			VALUES($1, $2, $3, 'human')`, draft, otherParcel, ourLink)
	}

	pv, err := svc.RollbackPreview(ctx, id)
	require.NoError(t, err)
	require.True(t, pv.CanRollback, "%v", pv.Blockers)
	require.Len(t, pv.Acquisitions, 1)
	require.True(t, pv.Acquisitions[0].Keep)
	require.Equal(t, []string{"E2E-HUMAN-K"}, pv.Acquisitions[0].Others, "үлдэх нэгж талбарыг хэрэглэгчид харуулна")

	res, err := svc.Rollback(ctx, id, "e2e admin")
	require.NoError(t, err)
	require.Equal(t, 176, res.Counts["deleted:parcel"])
	require.Empty(t, res.Acquisitions)
	require.Len(t, res.Kept, 1)

	// Импортын нэгж талбар, үнэлгээ устсан; хүний нэмсэн нэгж талбар ба чөлөөлөлт үлдсэн.
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id = ANY($1::uuid[])`, ourParcels))
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE acquisition_id = $1 AND created_by = $2`, ourAcq, Actor))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM land_acquisition WHERE id = $1`, ourAcq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id = $1 AND acquisition_id = $2`, humanParcel, ourAcq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM compensation WHERE acquisition_id = $1 AND parcel_id = 'E2E-HUMAN-K'`, ourAcq))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM parcel WHERE id = $1`, otherParcel))
	require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM plan WHERE plan_code = '2026-04-011-9878'`), "үлдсэн чөлөөлөлтийн төлөвлөгөө үлдэнэ")

	st, err = svc.Get(ctx, id)
	require.NoError(t, err)
	kept := findLog(st.RollbackLog, "acquisition", "land_acquisition", "kept", "")
	require.NotNil(t, kept)
	require.Contains(t, kept.Message, "E2E-HUMAN-K")
	require.NotNil(t, findLog(st.RollbackLog, "acquisition", "plan", "kept", ""))
	if ourLink != "" {
		require.Equal(t, 1, count(t, ctx, pool, `SELECT count(*) FROM decision_draft_parcel WHERE parcel_id = $1 AND funding_link_id = $2`, otherParcel, ourLink),
			"өөр нэгж талбарын захирамжийн холбоос хөндөгдөхгүй")
		require.NotNil(t, findLog(st.RollbackLog, "valuation", "funding_link", "kept", ""))
	}
}
