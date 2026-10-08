//go:build e2e

// Нэгж талбарын төлөвийн ДҮРЭМ (migration 000049) — өгөгдмөл тохиргоо нь
// өмнөх хатуу кодын дүрмийг ЯГ давтана; тохиргоог өөрчлөхөд дүрэм дагана.
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

	// Тохиргоог тестийн дараа БУЦААНА.
	type snap struct {
		id                                        int
		final, locks, valuation, released, reason bool
	}
	var before []snap
	rows, err := pool.Query(ctx, `SELECT id, is_final, locks_progress, is_valuation_stage, is_released, requires_reason FROM parcel_status`)
	require.NoError(t, err)
	for rows.Next() {
		var s snap
		require.NoError(t, rows.Scan(&s.id, &s.final, &s.locks, &s.valuation, &s.released, &s.reason))
		before = append(before, s)
	}
	rows.Close()
	t.Cleanup(func() {
		for _, s := range before {
			_, _ = pool.Exec(context.Background(), `UPDATE parcel_status SET is_final=$2, locks_progress=$3, is_valuation_stage=$4,
				is_released=$5, requires_reason=$6 WHERE id=$1`, s.id, s.final, s.locks, s.valuation, s.released, s.reason)
		}
	})

	// ── Seed — өмнөх хатуу кодын дүрэм ────────────────────────────────────
	require.NoError(t, seed.NewMainSeeder(pool).SeedParcelStatusFlags(ctx))
	rules := func(code string) domain.ParcelStatusRules {
		var id int
		require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM parcel_status WHERE code = $1`, code).Scan(&id))
		r, err := repo.GetParcelStatusRules(ctx, id)
		require.NoError(t, err)
		return *r
	}
	require.Equal(t, domain.ParcelStatusRules{IsFinal: true, LocksProgress: true, RequiresReason: true}, rules("affected_out"))
	require.Equal(t, domain.ParcelStatusRules{IsFinal: true, RequiresReason: true}, rules("rejected"))
	require.Equal(t, domain.ParcelStatusRules{IsFinal: true, LocksProgress: true, IsReleased: true}, rules("released"))
	require.Equal(t, domain.ParcelStatusRules{IsValuationStage: true}, rules("evaluating"))
	require.Equal(t, domain.ParcelStatusRules{}, rules("pending"))
	require.Equal(t, domain.ParcelStatusRules{}, rules("processing"))
	// Төлөвийн CRUD-ийн жагсаалтад дүрэм ирнэ.
	list, err := statusRepo.List(ctx)
	require.NoError(t, err)
	for _, s := range list {
		if s.Code == "released" {
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
	setRule := func(code, col string, v bool) {
		_, err := pool.Exec(ctx, `UPDATE parcel_status SET `+col+` = $2 WHERE code = $1`, code, v)
		require.NoError(t, err)
	}
	// «Үнэлгээ хийх»-ийг эцсийн болговол чөлөөлөлтийн бүх нэгж талбар эцсийн.
	setRule("evaluating", "is_final", true)
	nonFinal, err = repo.HasNonFinalParcels(ctx, acqID)
	require.NoError(t, err)
	require.False(t, nonFinal)
	// «Чөлөөлсөн»-ийг чөлөөлөгдсөн биш болговол устгах хаалт, хилийн хамгаалалт алга.
	setRule("released", "is_released", false)
	check, err = repo.CheckAcquisitionDeletable(ctx, acqID)
	require.NoError(t, err)
	require.Zero(t, check.ReleasedParcels)
	// «Захирамжид холбогдоно» (is_released) унтарсан — захирамжид холбогдохгүй.
	require.Empty(t, linkable())
	require.False(t, canLink(released))
	// «Татгалзсан»-д асаавал захирамжид холбогдоно.
	setRule("rejected", "is_released", true)
	require.Equal(t, []string{"E2ERULES0003"}, linkable())
	require.True(t, canLink(rejected))
	setRule("rejected", "is_released", false)
	// Тэгвэл чөлөөлөхөд олговор шаардахгүй (нэрээр биш тохиргоогоор).
	require.NoError(t, svc.UpdateParcelStatus(ctx, domain.ParcelStatusChange{
		ParcelID: evaluating, AcquisitionID: acqID, StatusID: domain.ParcelStatusReleased, CreatedBy: "e2e-rules",
	}, nil))
	// «Зөвшилцөх»-ийг үнэлгээний шат болговол гадаад жагсаалтад орно.
	setRule("processing", "is_valuation_stage", true)
	setRule("released", "is_valuation_stage", true)
	evalList, _, err = repo.GetParcels(ctx, acqID, domain.ParcelFilter{ValuationStageOnly: true})
	require.NoError(t, err)
	require.Len(t, evalList, 2, "Чөлөөлсөн (одоо үнэлгээний шат) 2 нэгж талбар")
	p, err = repo.GetParcelByID(ctx, released)
	require.NoError(t, err)
	require.True(t, p.StatusIsValuationStage)
	// «Явц түгжинэ»: «Чөлөөлсөн» (өгөгдмөл түгжээтэй) төлөвөөс явц солихгүй.
	toProcessing := domain.ParcelStatusChange{ParcelID: released, AcquisitionID: acqID, StatusID: 1, CreatedBy: "e2e-rules"}
	require.ErrorIs(t, svc.UpdateParcelStatus(ctx, toProcessing, nil), domain.ErrParcelProgressLocked)
	p, err = repo.GetParcelByID(ctx, released)
	require.NoError(t, err)
	require.Equal(t, domain.ParcelStatusReleased, p.StatusID, "төлөв солигдоогүй")
	// Түгжээг унтраавал солигдоно.
	setRule("released", "locks_progress", false)
	require.NoError(t, svc.UpdateParcelStatus(ctx, toProcessing, nil))
	// Шалтгаан заавлын тохиргоо уншигдана.
	setRule("pending", "requires_reason", true)
	require.True(t, rules("pending").RequiresReason)
}
