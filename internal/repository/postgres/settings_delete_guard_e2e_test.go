//go:build e2e

// Тохиргооны устгалын хамгаалалт (хавсралтын төрөл, захирамжийн сонголт) ба
// «Хянагдаж буй газар» (Төсөл хянах нэгж) — БОДИТ PostgreSQL дээр.
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run SettingsDeleteGuard -v
package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/postgres"
	"acquisition_backend/internal/seed"
	"acquisition_backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestE2ESettingsDeleteGuard(t *testing.T) {
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
	cleanup := func(sql string, args ...any) {
		t.Cleanup(func() {
			c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, _ = pool.Exec(c, sql, args...)
		})
	}
	var inUse *domain.InUseError

	// ── Төсөл хянах нэгж ─────────────────────────────────────────────────
	unitSvc := service.NewDecisionOptionService(postgres.NewDecisionReviewUnitRepo(pool))
	unit, err := unitSvc.Create(ctx, service.CreateDecisionOptionRequest{Code: "e2e_guard_unit", Name: "E2E Хуулийн хэлтэс"})
	require.NoError(t, err)
	cleanup(`DELETE FROM decision_review_unit WHERE id = $1`, unit.ID)
	spare, err := unitSvc.Create(ctx, service.CreateDecisionOptionRequest{Code: "e2e_guard_spare", Name: "E2E ашиглаагүй"})
	require.NoError(t, err)

	var draftID string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO decision_draft (proposal_no, created_by)
		VALUES ('E2E-GUARD-1', 'e2e-guard') RETURNING id`).Scan(&draftID))
	cleanup(`DELETE FROM decision_draft WHERE id = $1`, draftID) // явцын бичлэг CASCADE

	draftRepo := postgres.NewDecisionDraftRepo(pool)
	h := &domain.DecisionDraftProgressHistory{
		DecisionDraftID: draftID, ProgressType: domain.DecisionDraftProgressReviewing,
		ReviewUnitID: &unit.ID, ProgressDate: time.Now(), CreatedBy: "e2e-guard",
	}
	require.NoError(t, draftRepo.AddProgressHistory(ctx, h))
	require.Equal(t, "E2E Хуулийн хэлтэс", h.Recipient, "recipient = нэгжийн нэр")
	hist, err := draftRepo.ListProgressHistory(ctx, draftID)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, unit.ID, *hist[0].ReviewUnitID)
	// Олдохгүй нэгж — оруулахгүй.
	missing := 999999
	require.ErrorIs(t, draftRepo.AddProgressHistory(ctx, &domain.DecisionDraftProgressHistory{
		DecisionDraftID: draftID, ProgressType: domain.DecisionDraftProgressReviewing,
		ReviewUnitID: &missing, ProgressDate: time.Now(), CreatedBy: "e2e-guard",
	}), domain.ErrInvalidInput)

	// Хуучин «Хэнд» чөлөөт текст — seed нь нэгж болгон үүсгэж холбоно.
	var legacyID string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO decision_draft_progress_history
		(decision_draft_id, progress_type, recipient, progress_date, created_by)
		VALUES ($1, 'reviewing', '  E2E Хуучин хянагч ', CURRENT_DATE, 'e2e-guard') RETURNING id`, draftID).Scan(&legacyID))
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM decision_draft_progress_history WHERE id = $1`, legacyID)
		_, _ = pool.Exec(c, `DELETE FROM decision_review_unit WHERE name = 'E2E Хуучин хянагч'`)
	})
	require.NoError(t, seed.NewMainSeeder(pool).SeedDecisionReviewUnitsFromRecipients(ctx))
	require.NoError(t, seed.NewMainSeeder(pool).SeedDecisionReviewUnitsFromRecipients(ctx), "давтан ажиллахад алдаагүй")
	var legacyUnit *int
	var units int
	require.NoError(t, pool.QueryRow(ctx, `SELECT review_unit_id FROM decision_draft_progress_history WHERE id = $1`, legacyID).Scan(&legacyUnit))
	require.NotNil(t, legacyUnit, "хуучин бичлэг нэгжтэй холбогдсон")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM decision_review_unit WHERE name = 'E2E Хуучин хянагч'`).Scan(&units))
	require.Equal(t, 1, units, "давхардахгүй")

	// Ашиглагдаж буй нэгжийг устгахгүй; ашиглаагүйг устгана.
	require.ErrorAs(t, unitSvc.Delete(ctx, unit.ID), &inUse)
	require.Contains(t, inUse.Message, "1 захирамжийн явцын бичлэг")
	require.NoError(t, unitSvc.Delete(ctx, spare.ID))

	// ── Ажлын төрөл / Төсөв — захирамжид холбогдсон бол устгахгүй ─────────
	wtSvc := service.NewDecisionOptionService(postgres.NewDecisionWorkTypeRepo(pool))
	wt, err := wtSvc.Create(ctx, service.CreateDecisionOptionRequest{Code: "e2e_guard_wt", Name: "E2E ажлын төрөл"})
	require.NoError(t, err)
	cleanup(`DELETE FROM decision_work_type WHERE id = $1`, wt.ID)
	_, err = pool.Exec(ctx, `UPDATE decision_draft SET work_type_id = $1 WHERE id = $2`, wt.ID, draftID)
	require.NoError(t, err)
	require.ErrorAs(t, wtSvc.Delete(ctx, wt.ID), &inUse)
	require.Contains(t, inUse.Message, "1 захирамжийн төсөл")
	_, err = pool.Exec(ctx, `UPDATE decision_draft SET work_type_id = NULL WHERE id = $1`, draftID)
	require.NoError(t, err)
	require.NoError(t, wtSvc.Delete(ctx, wt.ID))

	// ── Хавсралтын төрөл ─────────────────────────────────────────────────
	dtRepo := postgres.NewDocumentTypeRepo(pool)
	dtSvc := service.NewDocumentTypeService(dtRepo)
	// Системийн код — устгахгүй, кодыг солихгүй.
	var reportID int
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM document_type WHERE type = 'valuation_report'`).Scan(&reportID))
	require.ErrorAs(t, dtSvc.Delete(ctx, reportID), &inUse)
	_, err = dtSvc.Update(ctx, reportID, service.UpdateDocumentTypeRequest{Type: "renamed_code"})
	require.ErrorAs(t, err, &inUse)

	dt, err := dtSvc.Create(ctx, service.CreateDocumentTypeRequest{Type: "e2e_guard_doc", Name: "E2E хавсралт", Target: "both"})
	require.NoError(t, err)
	cleanup(`DELETE FROM document_type WHERE id = $1`, dt.ID)
	// Нэгж талбарын ажлын урсгалын шаардлагад ашиглагдсан.
	var wfID int
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM parcel_workflow ORDER BY id LIMIT 1`).Scan(&wfID))
	_, err = pool.Exec(ctx, `INSERT INTO parcel_workflow_required_document (workflow_id, document_type_id) VALUES ($1, $2)`, wfID, dt.ID)
	require.NoError(t, err)
	cleanup(`DELETE FROM parcel_workflow_required_document WHERE document_type_id = $1`, dt.ID)
	require.ErrorAs(t, dtSvc.Delete(ctx, dt.ID), &inUse)
	require.Contains(t, inUse.Message, "ажлын урсгалын 1 шаардлага")
	_, err = pool.Exec(ctx, `DELETE FROM parcel_workflow_required_document WHERE document_type_id = $1`, dt.ID)
	require.NoError(t, err)
	// Хавсралт оруулсан.
	var docID string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO document (name, file_type, file_url, document_type_id)
		VALUES ('e2e.pdf', 'application/pdf', '/api/files/e2e.pdf', $1) RETURNING id`, dt.ID).Scan(&docID))
	cleanup(`DELETE FROM document WHERE id = $1`, docID)
	require.ErrorAs(t, dtSvc.Delete(ctx, dt.ID), &inUse)
	require.Contains(t, inUse.Message, "1 хавсралт")
	_, err = pool.Exec(ctx, `DELETE FROM document WHERE id = $1`, docID)
	require.NoError(t, err)
	// Ашиглагдаагүй болсон — устгана.
	require.NoError(t, dtSvc.Delete(ctx, dt.ID))
}
