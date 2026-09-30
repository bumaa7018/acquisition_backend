//go:build e2e

// Dashboard: захирамжийн төслийн тоо (явцаар) ба нөхөх олговрын задаргаа.
//
//	E2E_DATABASE_URL='postgres://…' go test -tags e2e ./internal/repository/postgres/ -run DashboardDecisions -v
package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestDashboardDecisionsAndCompensation(t *testing.T) {
	url := os.Getenv("E2E_DATABASE_URL")
	if url == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	exec := func(sql string, args ...any) {
		t.Helper()
		_, err := pool.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	one := func(sql string, args ...any) string {
		t.Helper()
		var id string
		require.NoError(t, pool.QueryRow(ctx, sql, args...).Scan(&id), sql)
		return id
	}
	exec(`INSERT INTO parcel_status(id, code, name) VALUES (0,'e2e0','Хүлээгдэж буй') ON CONFLICT DO NOTHING`)

	acqA := one(`INSERT INTO land_acquisition(plan_code, acquisition_name, created_by) VALUES ('E2E-DASH-A', 'E2E dash A', 'e2e-dash') RETURNING id`)
	acqB := one(`INSERT INTO land_acquisition(plan_code, acquisition_name, created_by) VALUES ('E2E-DASH-B', 'E2E dash B', 'e2e-dash') RETURNING id`)
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM decision_draft WHERE proposal_no LIKE 'E2E-DASH-%'`)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE created_by = 'e2e-dash'`)
	})

	// Нэгж талбар бүр НЭГ олговортой: p1..p5 = 100..500₮ (A), pb = 9000₮ (B).
	parcels := map[string]string{}
	for i, code := range []string{"E2E-DASH-P1", "E2E-DASH-P2", "E2E-DASH-P3", "E2E-DASH-P4", "E2E-DASH-P5"} {
		parcels[code] = one(`INSERT INTO parcel(acquisition_id, parcel_id, status, created_by) VALUES ($1, $2, 0, 'e2e-dash') RETURNING id`, acqA, code)
		exec(`INSERT INTO compensation(acquisition_id, parcel_id, amount, valuation_type, created_by) VALUES ($1, $2, $3, 'asset', 'e2e-dash')`,
			acqA, code, float64((i+1)*100))
		exec(`INSERT INTO parcel_status_history(parcel_id, acquisition_id, status_id, status_date, created_by) VALUES ($1, $2, 0, '2025-06-01', 'e2e-dash')`, code, acqA)
	}
	// Өөр үнэлгээний төрлийн олговор — сонгосон төрөл (asset) биш тул тоологдохгүй.
	exec(`INSERT INTO compensation(acquisition_id, parcel_id, amount, valuation_type, created_by) VALUES ($1, 'E2E-DASH-P5', 77777, 'land', 'e2e-dash')`, acqA)
	pb := one(`INSERT INTO parcel(acquisition_id, parcel_id, status, created_by) VALUES ($1, 'E2E-DASH-PB', 0, 'e2e-dash') RETURNING id`, acqB)
	exec(`INSERT INTO compensation(acquisition_id, parcel_id, amount, valuation_type, created_by) VALUES ($1, 'E2E-DASH-PB', 9000, 'asset', 'e2e-dash')`, acqB)

	draft := func(no string, status int, acq *string) string {
		return one(`INSERT INTO decision_draft(proposal_no, status, acquisition_id) VALUES ($1, $2, $3) RETURNING id`, no, status, acq)
	}
	progress := func(draftID, kind string) {
		ph := one(`INSERT INTO decision_draft_progress_history(decision_draft_id, progress_type, progress_date) VALUES ($1, $2, CURRENT_DATE) RETURNING id`, draftID, kind)
		exec(`UPDATE decision_draft SET current_progress_history_id = $1 WHERE id = $2`, ph, draftID)
	}
	link := func(draftID, parcelID string, removed bool) {
		if removed {
			exec(`INSERT INTO decision_draft_parcel(decision_draft_id, parcel_id, removed_at) VALUES ($1, $2, now())`, draftID, parcelID)
			return
		}
		exec(`INSERT INTO decision_draft_parcel(decision_draft_id, parcel_id) VALUES ($1, $2)`, draftID, parcelID)
	}
	d1 := draft("E2E-DASH-1", domain.DecisionDraftStatusConfirmed, nil) // Баталгаажсан, p1
	link(d1, parcels["E2E-DASH-P1"], false)
	d2 := draft("E2E-DASH-2", domain.DecisionDraftStatusDraft, nil) // Баталгаажуулах, p2
	progress(d2, domain.DecisionDraftProgressConfirming)
	link(d2, parcels["E2E-DASH-P2"], false)
	d3 := draft("E2E-DASH-3", domain.DecisionDraftStatusDraft, nil) // Хянагдаж буй, p3
	progress(d3, domain.DecisionDraftProgressReviewing)
	link(d3, parcels["E2E-DASH-P3"], false)
	draft("E2E-DASH-4", domain.DecisionDraftStatusDraft, &acqA)         // Төсөл — чөлөөлөлтийн дугаараар, нэгж талбаргүй
	d5 := draft("E2E-DASH-5", domain.DecisionDraftStatusConfirmed, nil) // p4-ийн ХАССАН холбоос — тоологдохгүй
	link(d5, parcels["E2E-DASH-P4"], true)
	d6 := draft("E2E-DASH-6", domain.DecisionDraftStatusConfirmed, &acqB) // Өөр чөлөөлөлт — тоологдохгүй
	link(d6, pb, false)

	// Захирамжгүй нэгж талбарын үнэлгээ: p3, p4 баталгаажсан; p5 баталгаажаагүй.
	for _, code := range []string{"E2E-DASH-P3", "E2E-DASH-P4"} {
		exec(`INSERT INTO parcel_valuation_submission(acquisition_id, parcel_id, valuation_type, status) VALUES ($1, $2, 'asset', $3)`,
			acqA, code, domain.ValuationStatusApproved)
	}
	exec(`INSERT INTO parcel_valuation_submission(acquisition_id, parcel_id, valuation_type, status) VALUES ($1, 'E2E-DASH-P5', 'asset', 'submitted')`, acqA)

	repo := postgres.NewLandAcquisitionRepo(pool)
	for _, tc := range []struct {
		name   string
		filter domain.DashboardFilter
	}{
		{"он шүүлтгүй", domain.DashboardFilter{AcquisitionID: acqA}},
		{"он шүүлттэй", domain.DashboardFilter{AcquisitionID: acqA, Years: []int{2025}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sum, err := repo.GetDashboard(ctx, tc.filter)
			require.NoError(t, err)
			require.Equal(t, domain.DashboardDecisions{Total: 4, Confirmed: 1, Confirming: 1, Reviewing: 1, Draft: 1}, sum.Decisions)
			require.EqualValues(t, 1, sum.TotalOrders, "НИЙТ ЗАХИРАМЖ = баталгаажсан")
			require.Equal(t, domain.DashboardCompensation{Total: 1500, Issued: 300, Approved: 700, Unapproved: 500}, sum.Compensation,
				"захирамж гарсан = p1 (баталгаажсан) + p2 (баталгаажуулах); баталгаажсан = p3 + p4; баталгаажаагүй = p5")
			require.InDelta(t, 1500, sum.TotalCompensation, 0.001)
		})
	}

	// Хоёр чөлөөлөлт хамт (шүүлтгүйтэй адил) — B-ийнх нэмэгдэнэ.
	sum, err := repo.GetDashboard(ctx, domain.DashboardFilter{PlanCode: "E2E-DASH-"})
	require.NoError(t, err)
	require.EqualValues(t, 5, sum.Decisions.Total)
	require.EqualValues(t, 2, sum.Decisions.Confirmed)
	require.InDelta(t, 10500, sum.Compensation.Total, 0.001)
	require.InDelta(t, 9300, sum.Compensation.Issued, 0.001)
}
