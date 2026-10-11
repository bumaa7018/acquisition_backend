//go:build e2e

// Нэгж талбарын төлөвийн ДҮРЭМ (migration 000049) — seeder-ийн тохиргоо
// (appdb2-ийн 101–109 төлөв) уншигдана; тохиргоог өөрчлөхөд дүрэм дагана.
// Төлөвийг id-аар танина (код нь орчин бүрт өөр байж болно).
// БОДИТ PostgreSQL дээр. Зөвхөн ӨӨРИЙН үүсгэсэн мөрийг цэвэрлэж, өөрчилсөн
// тохиргоог буцаана.
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run ParcelStatusRules -v
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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestE2EParcelStatusRules(t *testing.T) {
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
	repo := postgres.NewLandAcquisitionRepo(pool)
	statusRepo := postgres.NewParcelStatusRepo(pool)
	svc := service.NewLandService(repo, &psrdFiles{}, nil)

	// Тохиргоог тестийн дараа seeder-ийн утга руу БУЦААНА (хуулбар авбал id
	// шилжүүлгээс өмнөх 0–8 id-тай үлдэж сэргээгдэхгүй).
	t.Cleanup(func() {
		_ = seed.NewMainSeeder(pool).SyncParcelStatusWorkflow(context.Background())
	})

	// ── Seed — appdb2-ийн төлөв, ажлын урсгал ────────────────────────────
	require.NoError(t, seed.NewMainSeeder(pool).SyncParcelStatusWorkflow(ctx))
	require.NoError(t, seed.NewMainSeeder(pool).SyncParcelStatusWorkflow(ctx), "давтан ажиллахад алдаагүй")
	rules := func(id int) domain.ParcelStatusRules {
		r, err := repo.GetParcelStatusRules(ctx, id)
		require.NoError(t, err)
		return *r
	}
	const draft, payment = domain.ParcelStatusDecisionDraft, domain.ParcelStatusPayment
	require.Equal(t, domain.ParcelStatusRules{IsFinal: true, LocksProgress: true, RequiresReason: true}, rules(domain.ParcelStatusAffectedOut))
	require.Equal(t, domain.ParcelStatusRules{RequiresReason: true}, rules(domain.ParcelStatusRejected))
	require.Equal(t, domain.ParcelStatusRules{IsFinal: true, LocksProgress: true, IsReleased: true}, rules(domain.ParcelStatusReleased))
	require.Equal(t, domain.ParcelStatusRules{IsValuationStage: true}, rules(domain.ParcelStatusEvaluating))
	require.Equal(t, domain.ParcelStatusRules{IsReleased: true}, rules(draft))
	require.Equal(t, domain.ParcelStatusRules{IsReleased: true}, rules(payment))
	require.Equal(t, domain.ParcelStatusRules{}, rules(domain.ParcelStatusPending))
	require.Equal(t, domain.ParcelStatusRules{}, rules(domain.ParcelStatusProcessing))
	// Ажлын урсгал — 17 шилжилт, 131 заавал хавсралт.
	var transitions, required int
	require.NoError(t, pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM parcel_workflow),
		(SELECT COUNT(*) FROM parcel_workflow_required_document)`).Scan(&transitions, &required))
	require.Equal(t, 17, transitions)
	require.Equal(t, 131, required)
	// Төлөвийн CRUD-ийн жагсаалтад дүрэм ирнэ.
	list, err := statusRepo.List(ctx)
	require.NoError(t, err)
	for _, s := range list {
		if s.ID == domain.ParcelStatusReleased {
			require.Equal(t, "109", s.Code)
			require.True(t, s.IsReleased && s.IsFinal)
		}
	}

	// ── Өгөгдөл ────────────────────────────────────────────────────────────
	var acqID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
		VALUES ('E2E-RULES', 'E2E төлөвийн дүрэм', 2, 'e2e-rules') RETURNING id`).Scan(&acqID))
	newParcel := func(code string, status int) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by)
			VALUES ($1, $2, $3, 'e2e-rules') RETURNING id`, acqID, code, status).Scan(&id))
		return id
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM parcel_status_history WHERE acquisition_id = $1`, acqID)
		_, _ = pool.Exec(c, `DELETE FROM parcel WHERE acquisition_id = $1`, acqID)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE id = $1`, acqID)
	})
	evaluating := newParcel("E2ERULES0001", domain.ParcelStatusEvaluating)
	released := newParcel("E2ERULES0002", domain.ParcelStatusReleased)
	rejected := newParcel("E2ERULES0003", domain.ParcelStatusRejected)
	draftRepo := postgres.NewDecisionDraftRepo(pool)
	// Захирамжид холбох: хайлт (ReleasedOnly) ба холбох шалгалт (IsReleased).
	linkable := func() []string {
		list, _, err := repo.ListGlobalParcels(ctx, domain.GlobalParcelFilter{
			AcquisitionID: acqID.String(), ReleasedOnly: true, Page: 1, PageSize: 50,
		})
		require.NoError(t, err)
		var codes []string
		for _, gp := range list {
			codes = append(codes, gp.ParcelID)
		}
		return codes
	}
	canLink := func(id uuid.UUID) bool {
		info, err := draftRepo.GetParcelInfo(ctx, id.String())
		require.NoError(t, err)
		return info.IsReleased
	}

	// Өгөгдмөл: өмнөх үр дүн.
	nonFinal, err := repo.HasNonFinalParcels(ctx, acqID)
	require.NoError(t, err)
	require.True(t, nonFinal, "Үнэлгээ хийх нь эцсийн биш")
	refs, err := repo.ListParcelRefs(ctx, acqID)
	require.NoError(t, err)
	releasedRefs := map[string]bool{}
	for _, r := range refs {
		releasedRefs[r.ParcelID] = r.Released
	}
	require.Equal(t, map[string]bool{"E2ERULES0001": false, "E2ERULES0002": true, "E2ERULES0003": false}, releasedRefs)
	check, err := repo.CheckAcquisitionDeletable(ctx, acqID)
	require.NoError(t, err)
	require.Equal(t, 1, check.ReleasedParcels)
	require.Equal(t, []string{"E2ERULES0002"}, linkable())
	require.True(t, canLink(released))
	require.False(t, canLink(evaluating))
	require.False(t, canLink(rejected))
	evalList, _, err := repo.GetParcels(ctx, acqID, domain.ParcelFilter{ValuationStageOnly: true})
	require.NoError(t, err)
	require.Len(t, evalList, 1)
	require.Equal(t, "E2ERULES0001", evalList[0].ParcelID)
	p, err := repo.GetParcelByID(ctx, evaluating)
	require.NoError(t, err)
	require.True(t, p.StatusIsValuationStage)
	// Чөлөөлөгдсөн төлөвт батлагдсан олговоргүй шилжихгүй (өмнөх дүрэм).
	require.ErrorIs(t, svc.UpdateParcelStatus(ctx, domain.ParcelStatusChange{
		ParcelID: evaluating, AcquisitionID: acqID, StatusID: domain.ParcelStatusReleased, CreatedBy: "e2e-rules",
	}, nil), domain.ErrValuationReportRequired)

	// ── Тохиргоог өөрчилбөл дүрэм ДАГАНА ──────────────────────────────────
	setRule := func(id int, col string, v bool) {
		_, err := pool.Exec(ctx, `UPDATE parcel_status SET `+col+` = $2 WHERE id = $1`, id, v)
		require.NoError(t, err)
	}
	// «Үнэлгээ хийх», «Татгалзсан»-ыг эцсийн болговол чөлөөлөлтийн бүх нэгж талбар эцсийн.
	setRule(domain.ParcelStatusEvaluating, "is_final", true)
	setRule(domain.ParcelStatusRejected, "is_final", true)
	nonFinal, err = repo.HasNonFinalParcels(ctx, acqID)
	require.NoError(t, err)
	require.False(t, nonFinal)
	// «Чөлөөлсөн»-ийг чөлөөлөгдсөн биш болговол устгах хаалт, хилийн хамгаалалт алга.
	setRule(domain.ParcelStatusReleased, "is_released", false)
	check, err = repo.CheckAcquisitionDeletable(ctx, acqID)
	require.NoError(t, err)
	require.Zero(t, check.ReleasedParcels)
	// «Захирамжид холбогдоно» (is_released) унтарсан — захирамжид холбогдохгүй.
	require.Empty(t, linkable())
	require.False(t, canLink(released))
	// «Татгалзсан»-д асаавал захирамжид холбогдоно.
	setRule(domain.ParcelStatusRejected, "is_released", true)
	require.Equal(t, []string{"E2ERULES0003"}, linkable())
	require.True(t, canLink(rejected))
	setRule(domain.ParcelStatusRejected, "is_released", false)
	// Тэгвэл чөлөөлөхөд олговор шаардахгүй (нэрээр биш тохиргоогоор).
	require.NoError(t, svc.UpdateParcelStatus(ctx, domain.ParcelStatusChange{
		ParcelID: evaluating, AcquisitionID: acqID, StatusID: domain.ParcelStatusReleased, CreatedBy: "e2e-rules",
	}, nil))
	// «Зөвшилцөх»-ийг үнэлгээний шат болговол гадаад жагсаалтад орно.
	setRule(domain.ParcelStatusProcessing, "is_valuation_stage", true)
	setRule(domain.ParcelStatusReleased, "is_valuation_stage", true)
	evalList, _, err = repo.GetParcels(ctx, acqID, domain.ParcelFilter{ValuationStageOnly: true})
	require.NoError(t, err)
	require.Len(t, evalList, 2, "Чөлөөлсөн (одоо үнэлгээний шат) 2 нэгж талбар")
	p, err = repo.GetParcelByID(ctx, released)
	require.NoError(t, err)
	require.True(t, p.StatusIsValuationStage)
	// «Явц түгжинэ»: «Чөлөөлсөн» (өгөгдмөл түгжээтэй) төлөвөөс явц солихгүй.
	toProcessing := domain.ParcelStatusChange{ParcelID: released, AcquisitionID: acqID, StatusID: domain.ParcelStatusProcessing, CreatedBy: "e2e-rules"}
	require.ErrorIs(t, svc.UpdateParcelStatus(ctx, toProcessing, nil), domain.ErrParcelProgressLocked)
	p, err = repo.GetParcelByID(ctx, released)
	require.NoError(t, err)
	require.Equal(t, domain.ParcelStatusReleased, p.StatusID, "төлөв солигдоогүй")
	// Түгжээг унтраавал солигдоно.
	setRule(domain.ParcelStatusReleased, "locks_progress", false)
	require.NoError(t, svc.UpdateParcelStatus(ctx, toProcessing, nil))
	// Шалтгаан заавлын тохиргоо уншигдана.
	setRule(domain.ParcelStatusPending, "requires_reason", true)
	require.True(t, rules(domain.ParcelStatusPending).RequiresReason)
}
