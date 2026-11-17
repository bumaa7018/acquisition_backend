//go:build e2e

// «Хронос» — нэгж талбарын геометр (3857), чөлөөлсөн огноо (is_released төлөвт
// АНХ шилжсэн), төлөвийн түүх (эхэнд «Хүлээгдэж буй»), төлөвийн бүртгэл, сар
// бүрийн явцын өөрчлөлт, нөхөх олговор/захирамж/олголтын огноо.
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run Chronos -v
package postgres_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"acquisition_backend/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestE2EChronos(t *testing.T) {
	dsn := os.Getenv("E2E_DATABASE_URL")
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var dbName string
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName))
	if dbName == "appdb" {
		t.Skip("ажлын appdb дээр ажиллуулахгүй — хуулбар DB (appdb_e2e) ашиглана уу")
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM document WHERE name = 'e2e-chronos-receipt'`)
		_, _ = pool.Exec(c, `DELETE FROM decision_draft WHERE created_by = 'e2e-chronos'`)
		_, _ = pool.Exec(c, `DELETE FROM compensation WHERE parcel_id LIKE 'E2ECHR%'`)
		_, _ = pool.Exec(c, `DELETE FROM parcel_status_history WHERE parcel_id LIKE 'E2ECHR%'`)
		_, _ = pool.Exec(c, `DELETE FROM parcel WHERE parcel_id LIKE 'E2ECHR%'`)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE created_by = 'e2e-chronos'`)
	})
	var acqID, otherAcq uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
		VALUES ('E2E-CHR', 'E2E Хронос', 2, 'e2e-chronos') RETURNING id`).Scan(&acqID))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
		VALUES ('E2E-CHR2', 'E2E бусад', 2, 'e2e-chronos') RETURNING id`).Scan(&otherAcq))
	poly := `ST_GeomFromText('POLYGON((106.9 47.9, 106.9001 47.9, 106.9001 47.9001, 106.9 47.9001, 106.9 47.9))', 4326)`
	// Геометргүй нэгж талбар — статистикт орно (геометр нь null).
	_, err = pool.Exec(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by) VALUES ($1, 'E2ECHR0009', 105, 'e2e')`, otherAcq)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by, geometry) VALUES
		($1, 'E2ECHR0001', 109, 'e2e', `+poly+`),
		($1, 'E2ECHR0002', 104, 'e2e', `+poly+`),
		($2, 'E2ECHR0003', 109, 'e2e', `+poly+`)`, acqID, otherAcq)
	require.NoError(t, err)
	// 0001: 2021-03 үнэлгээ → 2021-07 чөлөөлсөн → 2022-01 татгалзсан → 2022-05 дахин чөлөөлсөн.
	_, err = pool.Exec(ctx, `INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, status_date) VALUES
		('E2ECHR0001', $1, 104, '2021-03-10'), ('E2ECHR0001', $1, 109, '2021-07-15'),
		('E2ECHR0001', $1, 106, '2022-01-05'), ('E2ECHR0001', $1, 109, '2022-05-01'),
		('E2ECHR0002', $1, 104, '2021-04-01'),
		('E2ECHR0003', $2, 109, '2023-02-02')`, acqID, otherAcq)
	require.NoError(t, err)

	// Нөхөх олговор: 0001 — бүгд батлагдсан (сүүлийнх 2021-06-15), 0002 — илгээсэн.
	// 0001 — эзэмшигч, зориулалт, эрх; олговрын 3 ангилал (газар / үл хөдлөх / эд хөрөнгө).
	_, err = pool.Exec(ctx, `UPDATE parcel SET landuse_name = 'Гэр хороолол', right_type = 2 WHERE parcel_id = 'E2ECHR0001'`)
	require.NoError(t, err)
	var holderID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO right_holder (last_name, name) VALUES ('Бат', 'Дорж') RETURNING id`).Scan(&holderID))
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM right_holder WHERE id = $1`, holderID) })
	_, err = pool.Exec(ctx, `INSERT INTO parcel_holder (parcel_id, holder_id, holder_role, main_applicant)
		SELECT id, $1, 'holder', true FROM parcel WHERE parcel_id = 'E2ECHR0001'`, holderID)
	require.NoError(t, err)
	var realID, propID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO assets (acquisition_id, parcel_id, asset_type) VALUES ($1, 'E2ECHR0001', 'real_state') RETURNING id`, acqID).Scan(&realID))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO assets (acquisition_id, parcel_id, asset_type) VALUES ($1, 'E2ECHR0001', 'property') RETURNING id`, acqID).Scan(&propID))
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM assets WHERE parcel_id LIKE 'E2ECHR%'`) })
	_, err = pool.Exec(ctx, `INSERT INTO compensation (acquisition_id, parcel_id, asset_id, amount, status, reviewed_at, created_at) VALUES
		($1, 'E2ECHR0001', $2, 200, 'approved', '2021-06-10', '2021-05-10'),
		($1, 'E2ECHR0001', $3, 50, 'approved', '2021-06-10', '2021-05-10')`, acqID, realID, propID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO compensation (acquisition_id, parcel_id, amount, status, reviewed_at, created_at) VALUES
		($1, 'E2ECHR0001', 1000, 'approved', '2021-06-01', '2021-05-01'),
		($1, 'E2ECHR0001', 500, 'approved', '2021-06-15', '2021-05-20'),
		($1, 'E2ECHR0002', 300, 'submitted', NULL, '2021-08-01')`, acqID)
	require.NoError(t, err)
	// Захирамж: 0001 нь 2021-12-01-нд холбогдож, 2022-02-01-нд батлагдсан.
	var draftID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO decision_draft (proposal_no, created_by, status, confirmed_at)
		VALUES ('E2E-CHR', 'e2e-chronos', 2, '2022-02-01') RETURNING id`).Scan(&draftID))
	_, err = pool.Exec(ctx, `INSERT INTO decision_draft_parcel (decision_draft_id, parcel_id, linked_at)
		SELECT $1, id, '2021-12-01' FROM parcel WHERE parcel_id = 'E2ECHR0001'`, draftID)
	require.NoError(t, err)
	// Олголт: 60% — 2022-03-01. Нэгж талбарын олголтын шатаас (хавсралтаас биш —
	// шат нь баримт хавсаргах эсвэл хуучин мэдээллийн импортоор бөглөгдөнө).
	_, err = pool.Exec(ctx, `UPDATE parcel SET compensation_paid_60 = true, compensation_paid_60_at = '2022-03-01'
		WHERE parcel_id = 'E2ECHR0001'`)
	require.NoError(t, err)

	repo := postgres.NewLandAcquisitionRepo(pool)
	data, err := repo.GetChronos(ctx, []uuid.UUID{acqID}, nil)
	require.NoError(t, err)
	require.False(t, data.Truncated)
	require.Len(t, data.Parcels, 2, "зөвхөн сонгосон чөлөөлөлт")
	p1, p2 := data.Parcels[0], data.Parcels[1]
	require.Equal(t, "E2ECHR0001", p1.ParcelID)
	require.NotNil(t, p1.ReleasedAt)
	require.Equal(t, "2021-07", p1.ReleasedAt.Format("2006-01"), "АНХ чөлөөлсөн огноо")
	require.Len(t, p1.Events, 5, "эхэнд «Хүлээгдэж буй» нэмэгдэнэ")
	require.Equal(t, 101, p1.Events[0].StatusID, "Хүлээгдэж буй")
	require.Equal(t, "2021-03-10", p1.Events[0].Date.Format("2006-01-02"), "чөлөөлөлт үүсгэснээс өмнө түүх эхэлсэн тул эхний түүхийн өдөр")
	require.Equal(t, 104, p1.Events[1].StatusID, "огноогоор эрэмбэлсэн")
	require.Equal(t, 1750.0, p1.CompAmount)
	require.Equal(t, [3]float64{1500, 200, 50}, [3]float64{p1.CompLand, p1.CompRealState, p1.CompProperty}, "газар / үл хөдлөх / эд хөрөнгө")
	require.Equal(t, "Бат Дорж", p1.HolderName)
	require.Equal(t, "Гэр хороолол", p1.LanduseName)
	require.Equal(t, 2, p1.RightType)
	require.Equal(t, "2021-05-01", p1.CompAt.Format("2006-01-02"))
	require.Equal(t, "2021-06-15", p1.CompApprovedAt.Format("2006-01-02"))
	require.Equal(t, draftID, *p1.DecisionID)
	require.Equal(t, "2022-02-01", p1.DecisionConfirmedAt.Format("2006-01-02"))
	require.Len(t, p1.Payments, 1)
	require.Equal(t, 60, p1.Payments[0].Stage)
	require.Equal(t, "2022-03-01", p1.Payments[0].Date)
	require.Nil(t, p2.CompApprovedAt, "бүгд батлагдаагүй")
	require.Nil(t, p2.DecisionID)
	// Захирамжийн төсөл — чөлөөлөлтийн хүрээгээр, баталсан огноотой.
	require.Len(t, data.Decisions, 1)
	require.Equal(t, draftID, data.Decisions[0].ID)
	require.NotNil(t, data.Decisions[0].ConfirmedAt)
	require.Nil(t, p2.ReleasedAt)
	require.Equal(t, "E2E Хронос", p1.AcquisitionName)

	var geom struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}
	require.NoError(t, json.Unmarshal(p1.Geometry, &geom))
	require.Equal(t, "Polygon", geom.Type)
	require.Greater(t, geom.Coordinates[0][0][0], 1e7, "EPSG:3857 метр (106° ≈ 11.9 сая м)")

	// Сар бүрийн явцын өөрчлөлт (сонгосон чөлөөлөлтийн бүх түүх).
	months := map[string]int{}
	for _, m := range data.Months {
		months[m.Month] = m.Count
	}
	require.Equal(t, map[string]int{"2021-03": 2, "2021-04": 2, "2021-07": 1, "2022-01": 1, "2022-05": 1}, months)
	require.Equal(t, "2021-03", data.Months[0].Month, "эрэмбэлсэн")
	// Төлөвийн бүртгэл — өнгө, эрэмбэ, чөлөөлсөн дүрэм.
	require.NotEmpty(t, data.Statuses)
	var released bool
	for _, s := range data.Statuses {
		if s.ID == 109 {
			released = s.IsReleased
		}
	}
	require.True(t, released)

	// Хоосон жагсаалт = бүх чөлөөлөлт (бусад тест өгөгдөл байж болох тул зөвхөн агуулахыг шалгана).
	all, err := repo.GetChronos(ctx, nil, nil)
	require.NoError(t, err)
	codes := map[string]bool{}
	for _, p := range all.Parcels {
		codes[p.ParcelID] = true
	}
	require.True(t, codes["E2ECHR0003"])
	require.True(t, codes["E2ECHR0009"], "геометргүй нэгж талбар ч орно")
	for _, p := range all.Parcels {
		if p.ParcelID == "E2ECHR0009" {
			require.JSONEq(t, "null", string(p.Geometry))
		}
	}
}

// Хуучин мэдээллийн импорт: олговор/захирамжийг ИМПОРТЫН өдрөөр бүртгэж,
// баталдаг ч архивын огноо нь compensation_date / decision_date-д байна.
// Хронос архивын (эрт) огноог авна — эс бөгөөс сар сонгоход бүх дүн 0 байж
// байгаад импортын сард л гарч ирдэг байв.
func TestE2EChronosArchiveDates(t *testing.T) {
	dsn := os.Getenv("E2E_DATABASE_URL")
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var dbName string
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName))
	if dbName == "appdb" {
		t.Skip("ажлын appdb дээр ажиллуулахгүй — хуулбар DB (appdb_e2e) ашиглана уу")
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM decision_draft WHERE created_by = 'e2e-chronos-arch'`)
		_, _ = pool.Exec(c, `DELETE FROM compensation WHERE parcel_id LIKE 'E2ECHA%'`)
		_, _ = pool.Exec(c, `DELETE FROM parcel WHERE parcel_id LIKE 'E2ECHA%'`)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE created_by = 'e2e-chronos-arch'`)
	})
	var acqID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
		VALUES ('E2E-CHA', 'E2E архив', 2, 'e2e-chronos-arch') RETURNING id`).Scan(&acqID))
	_, err = pool.Exec(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by) VALUES
		($1, 'E2ECHA0001', 109, 'e2e'), ($1, 'E2ECHA0002', 105, 'e2e')`, acqID)
	require.NoError(t, err)
	// 0001 — архивын олговор (2025-01-24), импортын өдөр (2026-10-05) бүртгэж баталсан.
	// 0002 — энгийн урсгал: огноогүй олговор, бүртгэсэн цаг нь өөрөө үнэн.
	_, err = pool.Exec(ctx, `INSERT INTO compensation (acquisition_id, parcel_id, amount, status, compensation_date, reviewed_at, created_at) VALUES
		($1, 'E2ECHA0001', 1000, 'approved', '2025-01-24', '2026-10-05', '2026-10-05'),
		($1, 'E2ECHA0002', 500, 'approved', NULL, '2026-03-02', '2026-03-01')`, acqID)
	require.NoError(t, err)
	// Захирамж 2025-01-24-нд гарсан, импортын өдөр бүртгэж баталсан; батлагдаагүй төсөл огноотой ч батлагдаагүй хэвээр.
	var archDraft, openDraft uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO decision_draft (proposal_no, created_by, status, decision_date, confirmed_at, created_at)
		VALUES ('E2E-CHA-1', 'e2e-chronos-arch', 2, '2025-01-24', '2026-10-05', '2026-10-05') RETURNING id`).Scan(&archDraft))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO decision_draft (proposal_no, created_by, status, decision_date, created_at)
		VALUES ('E2E-CHA-2', 'e2e-chronos-arch', 1, '2025-01-24', '2026-03-05') RETURNING id`).Scan(&openDraft))
	_, err = pool.Exec(ctx, `INSERT INTO decision_draft_parcel (decision_draft_id, parcel_id, linked_at)
		SELECT $1::uuid, id, '2026-10-05'::timestamptz FROM parcel WHERE parcel_id = 'E2ECHA0001'
		UNION ALL SELECT $2::uuid, id, '2026-03-06'::timestamptz FROM parcel WHERE parcel_id = 'E2ECHA0002'`, archDraft, openDraft)
	require.NoError(t, err)

	data, err := postgres.NewLandAcquisitionRepo(pool).GetChronos(ctx, []uuid.UUID{acqID}, nil)
	require.NoError(t, err)
	require.Len(t, data.Parcels, 2)
	day := func(tm *time.Time) string {
		require.NotNil(t, tm)
		return tm.Format("2006-01-02")
	}
	arch, plain := data.Parcels[0], data.Parcels[1]
	require.Equal(t, "2025-01-24", day(arch.CompAt), "олговор — архивын огноо")
	require.Equal(t, "2025-01-24", day(arch.CompApprovedAt), "баталсан — архивын огноо")
	require.Equal(t, "2025-01-24", day(arch.DecisionLinkedAt), "захирамжид холбогдсон — захирамжийн огноо")
	require.Equal(t, "2025-01-24", day(arch.DecisionConfirmedAt), "захирамж батлагдсан — захирамжийн огноо")
	require.Equal(t, "2026-03-01", day(plain.CompAt), "огноогүй олговор — бүртгэсэн цаг")
	require.Equal(t, "2026-03-02", day(plain.CompApprovedAt))
	require.Nil(t, plain.DecisionConfirmedAt, "батлагдаагүй захирамж — огноотой ч батлагдаагүй")
	decisions := map[uuid.UUID]string{}
	for _, d := range data.Decisions {
		confirmed := "-"
		if d.ConfirmedAt != nil {
			confirmed = d.ConfirmedAt.Format("2006-01-02")
		}
		decisions[d.ID] = d.CreatedAt.Format("2006-01-02") + "/" + confirmed
	}
	require.Equal(t, "2025-01-24/2025-01-24", decisions[archDraft], "архивын захирамж — захирамжийн огноогоор")
	require.Equal(t, "2025-01-24/-", decisions[openDraft], "батлагдаагүй төсөл батлагдсанд тооцогдохгүй")
}
