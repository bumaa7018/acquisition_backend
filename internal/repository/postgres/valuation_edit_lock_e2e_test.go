//go:build e2e

package postgres_test

import (
	"context"
	"testing"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Үнэлгээг ИЛГЭЭСЭН/БАТАЛГААЖСАН үед мэдээлэл засагдахгүй байх ёстой.
//
// Яагаад: мэргэжлийн байгууллага илгээсний дараа ч хөрөнгө/тооцоолол/олговрыг
// өөрчилж чаддаг байсан тул санхүүгийн хянасан тоо ба хавсаргасан тайлан хоёр
// чимээгүйхэн зөрөх боломжтой байв. Засвар нь ЗӨВХӨН draft/returned төлөвт.
func TestE2E_ValuationEditLockedAfterSubmit(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusEvaluating)
	parcelID := f.parcels[domain.ParcelStatusEvaluating]
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&code))
	_, err := f.pool.Exec(ctx, `UPDATE land_acquisition SET status=$2 WHERE id=$1`, f.acqID, domain.LandAcquisitionStatusField)
	require.NoError(t, err)

	var assetID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO assets (acquisition_id, parcel_id, asset_name, asset_type, photo_pdf_url, valuation_type)
		VALUES ($1,$2,'Байшин-1','real_state','/test/photo.pdf','asset') RETURNING id`,
		f.acqID, code).Scan(&assetID))

	name := "Байшин-1 (засварласан)"
	edit := func() error {
		_, err := f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
			ID: assetID, AcquisitionID: f.acqID, AssetName: &name,
		})
		return err
	}

	// 1) Илгээгээгүй (draft) — засагдана.
	require.NoError(t, edit())

	// Илгээхэд шаардагдах баримтууд (тайлан + ажлын зураг).
	attachSubmitDocuments(t, f, parcelID)
	base := domain.ValuationTransition{
		AcquisitionID: f.acqID, ParcelID: parcelID, ValuationType: domain.ValuationTypeAsset,
		Action: "submit", Note: "Submit", UserID: "Appraiser",
	}
	_, err = f.repo.TransitionValuationSubmission(ctx, base)
	require.NoError(t, err)

	// 2) Илгээсэн — хаалттай.
	assert.ErrorIs(t, edit(), domain.ErrValuationSubmittedLocked)
	assert.ErrorIs(t, f.repo.UpsertAssetCalculations(ctx, f.acqID, assetID, nil),
		domain.ErrValuationSubmittedLocked)
	assert.ErrorIs(t, f.repo.CreateCompensation(ctx, &domain.Compensation{
		AcquisitionID: f.acqID, TargetType: "asset", ParcelID: code, AssetID: &assetID,
		CompensationType: "cash", CoveragePercent: 100, Amount: 1000, ValuationType: "asset",
	}), domain.ErrValuationSubmittedLocked)
	assert.ErrorIs(t, f.repo.UpsertLandValuation(ctx, &domain.LandValuation{
		AcquisitionID: f.acqID, ParcelID: code, ValuationType: "asset",
		LandAreaM2: 100, BasePricePerM2: 2500,
	}), domain.ErrValuationSubmittedLocked)

	// 3) Буцаагдсан — дахин засагдана.
	base.Action, base.Note, base.UserID = "return", "Дутуу", "Finance"
	_, err = f.repo.TransitionValuationSubmission(ctx, base)
	require.NoError(t, err)
	require.NoError(t, edit())

	// 4) Баталгаажсан — дахин хаалттай.
	base.Action, base.Note, base.UserID = "submit", "Submit", "Appraiser"
	_, err = f.repo.TransitionValuationSubmission(ctx, base)
	require.NoError(t, err)
	base.Action, base.Note, base.UserID = "approve", "Approve", "Finance"
	_, err = f.repo.TransitionValuationSubmission(ctx, base)
	require.NoError(t, err)
	assert.ErrorIs(t, edit(), domain.ErrValuationSubmittedLocked)
}

// Өртгийн мөр Excel дэх БҮЛГЭЭ (Итгэлцүүр) хадгална — лавлахын grp хоосон ч.
func TestE2E_AssetCalculationKeepsExcelGroup(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusEvaluating)
	parcelID := f.parcels[domain.ParcelStatusEvaluating]
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&code))

	var assetID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO assets (acquisition_id, parcel_id, asset_name, asset_type, valuation_type)
		VALUES ($1,$2,'Байшин-1','real_state','asset') RETURNING id`, f.acqID, code).Scan(&assetID))

	// Лавлахын төрөл нь БҮЛЭГГҮЙ (ихэнх суулгасан төрөл ийм).
	var calcTypeID int
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order, grp)
		VALUES ($1,'Үнийн өсөлт (тест)','%',7,'')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		"e2e_price_growth_"+uuid.NewString()[:8]).Scan(&calcTypeID))

	require.NoError(t, f.repo.UpsertAssetCalculations(ctx, f.acqID, assetID, []domain.AssetCalculation{
		{AssetID: assetID, CalcTypeID: calcTypeID, Unit: "%", Value: 1.05, CalcGroup: "Итгэлцүүр"},
	}))
	list, err := f.repo.ListAssetCalculations(ctx, f.acqID, assetID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Итгэлцүүр", list[0].CalcGroup)

	// Зөвхөн ТООГ засахад (group хоосон ирэхэд) бүлэг хэвээр үлдэнэ.
	require.NoError(t, f.repo.UpsertAssetCalculations(ctx, f.acqID, assetID, []domain.AssetCalculation{
		{AssetID: assetID, CalcTypeID: calcTypeID, Unit: "%", Value: 1.10},
	}))
	list, err = f.repo.ListAssetCalculations(ctx, f.acqID, assetID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Итгэлцүүр", list[0].CalcGroup)
	assert.Equal(t, 1.10, list[0].Value)
}

// Дэлгэцээс ЗАРИМ талбарыг л засахад ажиллах ёстой.
//
// Регресс: хөрөнгийн цонх нь parcel_id / asset_type / notes зэргийг илгээдэггүй
// тул UPDATE нь хоосон parcel_id-гаар мөр хайж олохгүй, хэрэглэгчид 404
// харагддаг байв. Мөн илгээгээгүй талбарууд хоосон утгаар дарагдаж байсан.
func TestE2E_UpdateAssetPartialKeepsOtherFields(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusEvaluating)
	parcelID := f.parcels[domain.ParcelStatusEvaluating]
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&code))

	var assetID uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO assets (acquisition_id, parcel_id, asset_number, asset_type, asset_name,
		                    unit, capacity, notes, area_m2, unit_price, valuation_type)
		VALUES ($1,$2,'A-1','real_state','Байшин-1','м2','2 давхар','тэмдэглэл',100,1000,'asset')
		RETURNING id`, f.acqID, code).Scan(&assetID))

	// Дэлгэц шиг: ЗӨВХӨН талбай ба нэгж үнэ (Хүснэгт-5-ын засвар).
	area, price := 120.0, 2500.0
	updated, err := f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
		ID: assetID, AcquisitionID: f.acqID, AreaM2: &area, UnitPrice: &price,
	})
	require.NoError(t, err, "parcel_id илгээгээгүй ч засвар ажиллана")
	assert.Equal(t, 120.0, updated.AreaM2)
	assert.Equal(t, 2500.0, updated.UnitPrice)
	// Илгээгээгүй талбарууд ХЭВЭЭР.
	assert.Equal(t, "Байшин-1", updated.AssetName)
	assert.Equal(t, "A-1", updated.AssetNumber)
	assert.Equal(t, "real_state", updated.AssetType)
	assert.Equal(t, "м2", updated.Unit)
	assert.Equal(t, "2 давхар", updated.Capacity)
	assert.Equal(t, "тэмдэглэл", updated.Notes)
	assert.Equal(t, code, updated.ParcelID)

	// Хоосон МӨР илгээвэл (санаатай цэвэрлэх) тэр нь хүчинтэй.
	empty := ""
	updated, err = f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
		ID: assetID, AcquisitionID: f.acqID, Description: &empty, Notes: &empty,
	})
	require.NoError(t, err)
	assert.Empty(t, updated.Notes)

	// Өөр чөлөөлөлтийн id-гаар засах боломжгүй (404).
	_, err = f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
		ID: assetID, AcquisitionID: uuid.New(), AssetName: &empty,
	})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// "Үнэлгээний тайлан"-г ИЛГЭЭХИЙН ӨМНӨ хавсаргаж чадах ёстой.
//
// Регресс: AddParcelDocument нь valuation_report төрлийг ЗӨВХӨН үнэлгээ
// баталгаажсан (parcel.selected_valuation_type тавигдсан) нэгж талбарт
// оруулдаг байв. Тайланг илгээхэд заавал болгосон тул энэ хаалт нь өөрийгөө
// хаадаг байсан: хавсаргах гэхээр "баталгаажсаны дараа хавсаргана" гэж
// татгалзаж, улмаар илгээх ч боломжгүй болдог байв.
func TestE2E_ValuationReportAttachableBeforeApproval(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusEvaluating)
	parcelID := f.parcels[domain.ParcelStatusEvaluating]

	// Нэгж талбар дээр ямар ч үнэлгээ баталгаажаагүй.
	var selected *string
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT selected_valuation_type FROM parcel WHERE id=$1`, parcelID).Scan(&selected))
	require.Nil(t, selected)

	var typeID int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM document_type WHERE type='valuation_report'`).Scan(&typeID))

	doc := &domain.Document{
		ID: uuid.New(), Name: "Үнэлгээний тайлан", FileType: "application/pdf",
		FileURL:        "/api/files/parcel/e2e/report-before-approval.pdf",
		DocumentTypeID: typeID, UploadedAt: time.Now(),
	}
	require.NoError(t, f.repo.AddParcelDocument(ctx, parcelID, doc),
		"тайланг баталгаажихаас өмнө хавсаргаж чадах ёстой")

	docs, err := f.repo.ListParcelDocuments(ctx, parcelID)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, doc.FileURL, docs[0].FileURL)
}
