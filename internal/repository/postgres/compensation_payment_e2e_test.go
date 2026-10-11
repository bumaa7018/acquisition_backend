//go:build e2e

// «Нөхөх олговор олгосон баримт» — олголтын шат (60 / 40 / бүрэн) ба огноо
// нэгж талбарт тусах, баримт устгахад дагаж шинэчлэгдэх. БОДИТ PostgreSQL
// дээр (seed + repo + service). Зөвхөн ӨӨРИЙН үүсгэсэн мөрийг цэвэрлэнэ.
//
//	E2E_DATABASE_URL='postgres://postgres:postgres@localhost:5432/appdb_e2e?sslmode=disable' \
//	  go test -tags e2e ./internal/repository/postgres/ -run CompensationPayment -v
package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/postgres"
	"acquisition_backend/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestE2ECompensationPaymentStages(t *testing.T) {
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
	svc := service.NewLandService(repo, &psrdFiles{}, nil)
	// Хавсралтын төрөл — seed (давтан дуудахад аюулгүй).
	_, err = pool.Exec(ctx, `INSERT INTO document_type (type, name, target)
		VALUES ('compensation_paid_receipt', 'Нөхөх олговор олгосон баримт', 'parcel') ON CONFLICT (type) DO NOTHING`)
	require.NoError(t, err)
	var receiptType, otherType int
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM document_type WHERE type = 'compensation_paid_receipt'`).Scan(&receiptType))
	require.NoError(t, pool.QueryRow(ctx, `SELECT id FROM document_type WHERE type = 'contract'`).Scan(&otherType))

	var acqID, parcelID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO land_acquisition (plan_code, acquisition_name, status, created_by)
		VALUES ('E2E-PAY', 'E2E олговрын гүйцэтгэл', 2, 'e2e-pay') RETURNING id`).Scan(&acqID))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO parcel (acquisition_id, parcel_id, status, created_by)
		VALUES ($1, 'E2EPAY000001', 104, 'e2e-pay') RETURNING id`, acqID).Scan(&parcelID))
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM document WHERE id IN (SELECT document_id FROM parcel_document WHERE parcel_id = $1)`, parcelID)
		_, _ = pool.Exec(c, `DELETE FROM parcel WHERE id = $1`, parcelID)
		_, _ = pool.Exec(c, `DELETE FROM land_acquisition WHERE id = $1`, acqID)
	})

	day := func(v string) *time.Time {
		d, err := time.Parse("2006-01-02", v)
		require.NoError(t, err)
		return &d
	}
	upload := func(stage int, date *time.Time, name string) (*domain.Document, error) {
		var p *service.DocumentPayment
		if stage != 0 {
			p = &service.DocumentPayment{Stage: stage, Date: date}
		}
		return svc.UploadParcelDocumentWithPayment(ctx, parcelID, "barimt.pdf", name, "application/pdf", []byte("%PDF-1.7"), "e2e-pay", receiptType, p)
	}
	type state struct {
		p60, p40, pfull bool
		d60, d40, dfull *time.Time
	}
	read := func() state {
		var s state
		require.NoError(t, pool.QueryRow(ctx, `SELECT compensation_paid_60, compensation_paid_40, compensation_paid_full,
			compensation_paid_60_at, compensation_paid_40_at, compensation_paid_full_at FROM parcel WHERE id = $1`, parcelID).
			Scan(&s.p60, &s.p40, &s.pfull, &s.d60, &s.d40, &s.dfull))
		return s
	}
	ymd := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02")
	}

	// Шатгүй эсвэл буруу шат / ирээдүйн огноо → хүлээж авахгүй.
	_, err = upload(0, nil, "")
	require.ErrorIs(t, err, domain.ErrInvalidInput, "төлбөрийн баримтад шат заавал")
	_, err = upload(50, nil, "")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	_, err = upload(60, func() *time.Time { v := time.Now().AddDate(0, 0, 3); return &v }(), "")
	require.ErrorIs(t, err, domain.ErrInvalidInput, "ирээдүйн огноо")
	_, err = svc.UploadParcelDocumentWithPayment(ctx, parcelID, "geree.pdf", "", "application/pdf", []byte("%PDF-1.7"), "e2e-pay", otherType,
		&service.DocumentPayment{Stage: 60})
	require.ErrorIs(t, err, domain.ErrInvalidInput, "өөр төрөлд шат хүлээж авахгүй")
	require.Equal(t, state{}, read(), "алдаатай оруулалт нэгж талбарт нөлөөлөхгүй")

	// 60% → нэр «… /60%/», нэгж талбарт 60 ба огноо.
	d60, err := upload(60, day("2026-09-01"), "")
	require.NoError(t, err)
	require.Equal(t, "Нөхөх олговор олгосон баримт /60%/", d60.Name)
	s := read()
	require.True(t, s.p60)
	require.False(t, s.p40 || s.pfull)
	require.Equal(t, "2026-09-01", ymd(s.d60))
	require.Equal(t, 60, domain.CompensationPaidPercent(s.p60, s.p40, s.pfull))

	// 40% (хэрэглэгчийн нэр + гараас бичсэн хуучин тэмдэглэгээг солино) → 100%.
	d40, err := upload(40, day("2026-10-01"), "Төлбөрийн баримт /60%/")
	require.NoError(t, err)
	require.Equal(t, "Төлбөрийн баримт /40%/", d40.Name)
	s = read()
	require.True(t, s.p60 && s.p40)
	require.Equal(t, "2026-10-01", ymd(s.d40))
	require.Equal(t, 100, domain.CompensationPaidPercent(s.p60, s.p40, s.pfull))

	// Баримтын жагсаалтад шат, огноо.
	docs, err := repo.ListParcelDocuments(ctx, parcelID)
	require.NoError(t, err)
	stages := map[int]string{}
	for _, d := range docs {
		stages[d.PaymentStage] = ymd(d.PaymentDate)
	}
	require.Equal(t, map[int]string{60: "2026-09-01", 40: "2026-10-01"}, stages)

	// 60%-ийн баримт устгахад нэгж талбарын 60 дагаж арилна.
	require.NoError(t, svc.DeleteParcelDocument(ctx, parcelID, d60.ID))
	s = read()
	require.False(t, s.p60)
	require.Nil(t, s.d60)
	require.True(t, s.p40)
	require.Equal(t, 40, domain.CompensationPaidPercent(s.p60, s.p40, s.pfull))

	// Бүрэн — огноо өгөөгүй бол өнөөдөр; нэр «/бүрэн/».
	full, err := upload(100, nil, "")
	require.NoError(t, err)
	require.Equal(t, "Нөхөх олговор олгосон баримт /бүрэн/", full.Name)
	s = read()
	require.True(t, s.pfull)
	require.NotNil(t, s.dfull)
	require.WithinDuration(t, time.Now(), *s.dfull, 48*time.Hour)

	// Олон баримттай шат — нэгийг устгахад нөгөө нь хэвээр (огноо нь үлдсэнийх).
	second40, err := upload(40, day("2026-10-03"), "")
	require.NoError(t, err)
	require.Equal(t, "2026-10-03", ymd(read().d40), "сүүлийн огноо")
	require.NoError(t, svc.DeleteParcelDocument(ctx, parcelID, second40.ID))
	s = read()
	require.True(t, s.p40)
	require.Equal(t, "2026-10-01", ymd(s.d40))

	// Төлбөрийн бус баримт устгахад гүйцэтгэл хөндөгдөхгүй.
	other, err := svc.UploadParcelDocument(ctx, parcelID, "geree.pdf", "", "application/pdf", []byte("%PDF-1.7"), "e2e-pay", otherType)
	require.NoError(t, err)
	require.Zero(t, other.PaymentStage)
	before := read()
	require.NoError(t, svc.DeleteParcelDocument(ctx, parcelID, other.ID))
	require.Equal(t, before, read())

	// Жагсаалтын «Хаяг» — au1/au2/au3-ийн НЭР (кодоор нь хайна).
	var au1, au1Name, au2, au2Name, au3, au3Name string
	if err := pool.QueryRow(ctx, `SELECT a3.au1_code, a1.name, a3.au2_code, a2.name, a3.code, a3.name
		FROM au3 a3 JOIN au2 a2 ON a2.code = a3.au2_code JOIN au1 a1 ON a1.code = a3.au1_code LIMIT 1`).
		Scan(&au1, &au1Name, &au2, &au2Name, &au3, &au3Name); err == nil {
		_, err = pool.Exec(ctx, `UPDATE parcel SET au1_code = $2, au2_code = $3, au3_code = $4 WHERE id = $1`, parcelID, au1, au2, au3)
		require.NoError(t, err)
	}

	// Жагсаалт ба дэлгэрэнгүйд гүйцэтгэл ирнэ.
	list, _, err := repo.GetParcels(ctx, acqID, domain.ParcelFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].CompensationPaid40 && list[0].CompensationPaidFull)
	require.Equal(t, []string{au1Name, au2Name, au3Name}, []string{list[0].AU1Name, list[0].AU2Name, list[0].AU3Name})

	// «Хаяг»-т гудамж, хашаа; «Зориулалт»-д нэр.
	_, err = pool.Exec(ctx, `UPDATE parcel SET address_streetname = 'Энхтайваны өргөн чөлөө', address_khashaa = '12',
		landuse = '0101', landuse_name = 'Гэр бүлийн хэрэгцээ' WHERE id = $1`, parcelID)
	require.NoError(t, err)
	list, _, err = repo.GetParcels(ctx, acqID, domain.ParcelFilter{})
	require.NoError(t, err)
	require.Equal(t, "Энхтайваны өргөн чөлөө", list[0].AddressStreet)
	require.Equal(t, "12", list[0].AddressKhashaa)
	require.Equal(t, "Гэр бүлийн хэрэгцээ", list[0].LanduseName)
	full2, err := repo.GetParcelDetailByID(ctx, parcelID)
	require.NoError(t, err)
	require.Equal(t, "2026-10-01", ymd(full2.CompensationPaid40At))
}
