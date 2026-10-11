//go:build e2e

package legacyimport

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Хуучин хавсралт: PDF-ийн нэрийг нэгж талбарын дугаартай ЯГ тулгана —
// олдсон (өөр чөлөөлөлтүүдээс), олдоогүй, өмнө оруулсан баримт.
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/legacyimport/ -run MatchAttachments -v
func TestE2EMatchAttachments(t *testing.T) {
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
	// Цэвэрлэгээг мөр оруулахаас ӨМНӨ бүртгэнэ — дунд нь унасан ч үлдэгдэлгүй.
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM document WHERE name = 'e2e-att-receipt'`)
		_, _ = pool.Exec(c, `DELETE FROM parcel WHERE created_by = 'e2e-att'`)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE created_by = 'e2e-att'`)
	})
	newAcq := func(name string) string {
		var id string
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
			VALUES ('E2E-ATT', $1, 3, 'e2e-att') RETURNING id::text`, name).Scan(&id))
		return id
	}
	acqA, acqB := newAcq("E2E хавсралт А"), newAcq("E2E хавсралт Б")
	newParcel := func(acq, code string) string {
		var id string
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by)
			VALUES ($1, $2, 109, 'e2e-att') RETURNING id::text`, acq, code).Scan(&id))
		return id
	}
	single := newParcel(acqA, "E2EATT00001")
	other := newParcel(acqB, "E2EATT00002")
	// «single» нэгж талбарт өмнө оруулсан нэг баримт.
	var docID string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO document (name, file_type, file_url, document_type_id)
		SELECT 'e2e-att-receipt', 'application/pdf', '/api/files/e2e.pdf', id FROM document_type
		 WHERE type = 'compensation_paid_receipt' RETURNING id::text`).Scan(&docID))
	_, err = pool.Exec(ctx, `INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1, $2)`, single, docID)
	require.NoError(t, err)

	svc := &Service{db: pool}
	got, err := svc.MatchAttachments(ctx, []string{" E2EATT00001", "E2EATT00002", "E2EATT99999", "E2EATT00001", ""})
	require.NoError(t, err)
	require.Len(t, got, 3, "давхардал, хоосон нэр хасагдана; оролтын дараалал")
	require.Equal(t, "E2EATT00001", got[0].ParcelNumber)
	require.NotNil(t, got[0].Parcel)
	require.Equal(t, single, got[0].Parcel.ID)
	require.Equal(t, acqA, got[0].Parcel.AcquisitionID)
	require.Equal(t, 1, got[0].Parcel.Receipts, "өмнө оруулсан баримт тоологдоно")
	require.Equal(t, "E2EATT00002", got[1].ParcelNumber)
	require.Equal(t, other, got[1].Parcel.ID)
	require.Equal(t, "E2E хавсралт Б", got[1].Parcel.AcquisitionName)
	require.Zero(t, got[1].Parcel.Receipts)
	require.Equal(t, "E2EATT99999", got[2].ParcelNumber)
	require.Nil(t, got[2].Parcel, "олдоогүй")
}
