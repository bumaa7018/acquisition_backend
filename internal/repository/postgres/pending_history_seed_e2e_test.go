//go:build e2e

// Анхны «Хүлээгдэж буй» түүх — seed 000051 бүх нэгж талбарт нэмнэ (давхардалгүй).
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run PendingHistory -v
package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"acquisition_backend/internal/seed"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestE2EParcelPendingHistorySeed(t *testing.T) {
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
		_, _ = pool.Exec(c, `DELETE FROM parcel_status_history WHERE parcel_id LIKE 'E2EPND%'`)
		_, _ = pool.Exec(c, `DELETE FROM parcel WHERE parcel_id LIKE 'E2EPND%'`)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE created_by = 'e2e-pending'`)
	})
	var acqID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by, created_at)
		VALUES ('E2E-PND', 'E2E pending', 3, 'e2e-pending', '2024-01-10') RETURNING id`).Scan(&acqID))
	_, err = pool.Exec(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by) VALUES
		($1, 'E2EPND0001', 5, 'e2e'), ($1, 'E2EPND0002', 2, 'e2e'), ($1, 'E2EPND0003', 0, 'e2e')`, acqID)
	require.NoError(t, err)
	// 0001 — хуучин (чөлөөлөлт үүсгэснээс өмнө) эцсийн төлөв; 0002 — дараа нь; 0003 — аль хэдийн pending-тэй.
	_, err = pool.Exec(ctx, `INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, status_date, created_by) VALUES
		('E2EPND0001', $1, 5, '2023-05-01', 'legacy-import'),
		('E2EPND0002', $1, 2, '2024-03-01', 'x'),
		('E2EPND0003', $1, 0, '2024-01-10', 'system')`, acqID)
	require.NoError(t, err)

	s := seed.NewMainSeeder(pool)
	require.NoError(t, s.SeedParcelPendingHistory(ctx))
	require.NoError(t, s.SeedParcelPendingHistory(ctx), "давтан ажиллахад давхардахгүй")

	first := func(code string) (int, time.Time, int) {
		var status, n int
		var at time.Time
		require.NoError(t, pool.QueryRow(ctx, `SELECT status_id, status_date FROM parcel_status_history
			WHERE parcel_id = $1 ORDER BY status_date, id LIMIT 1`, code).Scan(&status, &at))
		require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM parcel_status_history WHERE parcel_id = $1 AND status_id = 0`, code).Scan(&n))
		return status, at, n
	}
	st, at, n := first("E2EPND0001")
	require.Equal(t, 0, st, "хамгийн эхэнд «Хүлээгдэж буй»")
	require.Equal(t, 1, n)
	require.True(t, at.Before(time.Date(2023, 5, 1, 0, 0, 0, 0, time.Local)), "эцсийн төлөвөөс өмнө: %v", at)
	st, at, n = first("E2EPND0002")
	require.Equal(t, 0, st)
	require.Equal(t, 1, n)
	require.Equal(t, "2024-01-10", at.Format("2006-01-02"), "чөлөөлөлт үүсгэсэн өдөр")
	_, _, n = first("E2EPND0003")
	require.Equal(t, 1, n, "аль хэдийн байгаа бол нэмэхгүй")
}
