//go:build e2e

// ҮНЭЛГЭЭНИЙ БҮХЭЛ ФЛОУГ ЭХНЭЭС ДУУСТАЛ шалгана (БОДИТ PostgreSQL дээр).
//
//	Excel импорт → засвар → баримт хавсаргах → илгээх → хянах → баталгаажуулах
//	→ нэгж талбар "Чөлөөлсөн" → хүчингүй болгох
//
// Яагаад нэг тест: алхмууд бие биенээсээ хамаардаг (тайлангүй бол илгээхгүй,
// илгээгээгүй бол баталгаажихгүй, баталгаажаагүй бол чөлөөлөхгүй). Тус тусад
// нь шалгавал гинжин холбоос тасарсныг олохгүй — яг тэр л алдаанууд гарч байв.
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

func docTypeID(t *testing.T, f *fixture, code string) int {
	t.Helper()
	var id int
	require.NoError(t, f.pool.QueryRow(context.Background(),
		`SELECT id FROM document_type WHERE type=$1`, code).Scan(&id),
		"document_type %q байхгүй (seeder ажиллуулаагүй байна уу?)", code)
	return id
}

func attachDoc(t *testing.T, f *fixture, parcelID uuid.UUID, code, name, url string) *domain.Document {
	t.Helper()
	doc := &domain.Document{
		ID: uuid.New(), Name: name, FileType: "application/pdf", FileURL: url,
		DocumentTypeID: docTypeID(t, f, code), UploadedAt: time.Now(),
	}
	require.NoError(t, f.repo.AddParcelDocument(context.Background(), parcelID, doc))
	return doc
}

func TestE2E_ValuationFullFlow(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 0, domain.ParcelStatusEvaluating)
	parcelID := f.parcels[domain.ParcelStatusEvaluating]
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&code))
	_, err := f.pool.Exec(ctx, `UPDATE land_acquisition SET status=$2 WHERE id=$1`,
		f.acqID, domain.LandAcquisitionStatusField)
	require.NoError(t, err)

	// ── 1. EXCEL ИМПОРТ ─────────────────────────────────────────────────────
	// Барилга (итгэлцүүртэй тооцоолол) + эд хөрөнгө + газрын үнэлгээ + тайлбар.
	var unitCostType, coeffType, netType int
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order, grp)
		VALUES ($1,'Барилгын нэгжийн өртөг (full-flow)','төгрөг/м²',1,'')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		"ff_unit_"+uuid.NewString()[:8]).Scan(&unitCostType))
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order, grp)
		VALUES ($1,'Үнийн өсөлт (full-flow)','%',7,'')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		"ff_coeff_"+uuid.NewString()[:8]).Scan(&coeffType))
	require.NoError(t, f.pool.QueryRow(ctx, `
		INSERT INTO asset_calc_type (code, name, default_unit, sort_order, grp)
		VALUES ($1,'Барилгын нөхөн орлуулах өртөг (full-flow)','₮',10,'')
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		"ff_net_"+uuid.NewString()[:8]).Scan(&netType))

	imp := &domain.ValuationImport{
		ParcelID:      code,
		ValuationType: domain.ValuationTypeAsset,
		Land:          domain.LandValuation{LandAreaM2: 500, BasePricePerM2: 3000, OwnershipCertNo: "ГЭ-123"},
		Assets: []domain.ValuationImportAsset{
			{
				AssetNumber: "1", AssetType: "real_state", AssetName: "Байшин-1",
				AreaM2: 100, FloorCount: 2, Unit: "м2", CompensationAmount: 79_800_000,
				Calculations: []domain.AssetCalculation{
					{CalcTypeID: unitCostType, Unit: "төгрөг/м²", Value: 1_000_000},
					{CalcTypeID: coeffType, Unit: "%", Value: 1.05, CalcGroup: "Итгэлцүүр"},
					{CalcTypeID: netType, Unit: "₮", Value: 79_800_000},
				},
			},
			{
				AssetNumber: "2", AssetType: "property", AssetName: "Хашаа",
				AreaM2: 20, Unit: "м", UnitPrice: 50_000, CompensationAmount: 1_000_000,
			},
		},
		Sections: []domain.ValuationSectionInput{
			{Key: "building_cost", Note: "Өртгийн хандлагаар тооцов", SectionNo: "3.5",
				TableLabel: "Хүснэгт-5", Title: "Барилга байгууламжийн өртгийн хандлагаарх тооцоолол", SortOrder: 5},
		},
		CreatedBy: "e2e",
	}
	res, err := f.repo.ImportValuation(ctx, f.acqID, imp)
	require.NoError(t, err, "1. Excel импорт")
	assert.Equal(t, 2, res.CreatedAssets)
	// 2 хөрөнгийн олговор + газрын үнэлгээний нэгж талбар түвшний нэг мөр.
	assert.Equal(t, 3, res.CreatedComps)

	assets, _, err := f.repo.ListAssets(ctx, f.acqID, domain.AssetFilter{ParcelID: code, Page: 1, PageSize: 50})
	require.NoError(t, err)
	require.Len(t, assets, 2)
	var building *domain.Asset
	for _, a := range assets {
		if a.AssetType == "real_state" {
			building = a
		}
	}
	require.NotNil(t, building)

	// Excel дэх БҮЛЭГ ба ДАРААЛАЛ хадгалагдсан эсэх.
	calcs, err := f.repo.ListAssetCalculations(ctx, f.acqID, building.ID)
	require.NoError(t, err)
	require.Len(t, calcs, 3)
	assert.Equal(t, "Итгэлцүүр", calcs[1].CalcGroup, "2. бүлэг Excel-ээс хадгалагдана")
	assert.Equal(t, "Барилгын нэгжийн өртөг (full-flow)", calcs[0].CalcName, "мөрийн дараалал Excel-ийнх")

	// ── 2. ЗАСВАР (илгээхээс өмнө) ──────────────────────────────────────────
	area := 120.0
	updated, err := f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
		ID: building.ID, AcquisitionID: f.acqID, AreaM2: &area,
	})
	require.NoError(t, err, "3. хэсэгчилсэн засвар (parcel_id илгээхгүй)")
	assert.Equal(t, 120.0, updated.AreaM2)
	assert.Equal(t, "Байшин-1", updated.AssetName, "илгээгээгүй талбар хэвээр")

	require.NoError(t, f.repo.UpsertAssetCalculations(ctx, f.acqID, building.ID,
		[]domain.AssetCalculation{{AssetID: building.ID, CalcTypeID: coeffType, Unit: "%", Value: 1.10}}),
		"4. тооцоолол засах")
	calcs, err = f.repo.ListAssetCalculations(ctx, f.acqID, building.ID)
	require.NoError(t, err)
	assert.Equal(t, "Итгэлцүүр", calcs[1].CalcGroup, "зөвхөн тоо засахад бүлэг арилахгүй")
	assert.Equal(t, 1.10, calcs[1].Value)

	// ── 3. ИЛГЭЭХ — баримтгүйгээр БОЛОХГҮЙ ──────────────────────────────────
	submit := domain.ValuationTransition{
		AcquisitionID: f.acqID, ParcelID: parcelID, ValuationType: domain.ValuationTypeAsset,
		Action: "submit", Note: "Илгээв", UserID: "Үнэлгээчин",
	}
	_, err = f.repo.TransitionValuationSubmission(ctx, submit)
	assert.ErrorIs(t, err, domain.ErrSubmitReportRequired, "5. тайлангүй илгээхгүй")

	report := attachDoc(t, f, parcelID, "valuation_report", "Үнэлгээний тайлан",
		"/api/files/parcel/e2e/ff-report.pdf")
	_, err = f.repo.TransitionValuationSubmission(ctx, submit)
	assert.ErrorIs(t, err, domain.ErrSubmitWorkPhotoRequired, "6. ажлын зурагггүй илгээхгүй")

	attachDoc(t, f, parcelID, "work_photo", "Ажлын зураг", "/api/files/parcel/e2e/ff-photo.jpg")

	sub, err := f.repo.TransitionValuationSubmission(ctx, submit)
	require.NoError(t, err, "7. хоёр баримттай илгээх")
	assert.Equal(t, domain.ValuationStatusSubmitted, sub.Status)

	// ── 4. ИЛГЭЭСНИЙ ДАРАА ЗАСАХГҮЙ ─────────────────────────────────────────
	name := "Өөрчилсөн нэр"
	_, err = f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
		ID: building.ID, AcquisitionID: f.acqID, AssetName: &name,
	})
	assert.ErrorIs(t, err, domain.ErrValuationSubmittedLocked, "8. илгээсний дараа засвар хаалттай")

	// ── 5. БУЦААХ → дахин засах → дахин илгээх ──────────────────────────────
	ret := submit
	ret.Action, ret.Note, ret.UserID = "return", "Дутуу", "Санхүү"
	_, err = f.repo.TransitionValuationSubmission(ctx, ret)
	require.NoError(t, err, "9. буцаах")
	_, err = f.repo.UpdateAsset(ctx, &domain.AssetUpdate{
		ID: building.ID, AcquisitionID: f.acqID, AssetName: &name,
	})
	require.NoError(t, err, "10. буцаагдсаны дараа засагдана")
	_, err = f.repo.TransitionValuationSubmission(ctx, submit)
	require.NoError(t, err, "11. дахин илгээх (баримтууд хэвээр)")

	// ── 6. БАТАЛГААЖУУЛАХ ───────────────────────────────────────────────────
	approve := submit
	approve.Action, approve.Note, approve.UserID = "approve", "Зөвшөөрөв", "Санхүү"
	sub, err = f.repo.TransitionValuationSubmission(ctx, approve)
	require.NoError(t, err, "12. баталгаажуулах")
	assert.Equal(t, domain.ValuationStatusApproved, sub.Status)

	var selected *string
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT selected_valuation_type FROM parcel WHERE id=$1`, parcelID).Scan(&selected))
	require.NotNil(t, selected)
	assert.Equal(t, domain.ValuationTypeAsset, *selected, "13. үндсэн урсгал тавигдана")

	var approvedComps int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM compensation WHERE acquisition_id=$1 AND status='approved'`,
		f.acqID).Scan(&approvedComps))
	assert.Greater(t, approvedComps, 0, "14. олговрууд баталгаажина")

	// ── 7. НЭГЖ ТАЛБАР "ЧӨЛӨӨЛСӨН" ─────────────────────────────────────────
	require.NoError(t, f.repo.UpdateParcelStatus(ctx, domain.ParcelStatusChange{
		ParcelID: parcelID, AcquisitionID: f.acqID,
		StatusID: domain.ParcelStatusReleased, CreatedBy: "Ажилтан", Reason: "Чөлөөлөв",
	}), "15. чөлөөлөх (илгээхэд хавсаргасан тайлан шаардлагыг хангана)")

	var parcelStatus int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT status FROM parcel WHERE id=$1`, parcelID).Scan(&parcelStatus))
	assert.Equal(t, domain.ParcelStatusReleased, parcelStatus)

	// ── 8. ХҮЧИНГҮЙ БОЛГОХ ──────────────────────────────────────────────────
	cancel := submit
	cancel.Action, cancel.Note, cancel.UserID = "cancel", "Дахин үнэлнэ", "Ахлах"
	cancel.AttachmentURL, cancel.AttachmentName = "/api/files/valuation-cancel/e2e/c.pdf", "c.pdf"
	sub, err = f.repo.TransitionValuationSubmission(ctx, cancel)
	require.NoError(t, err, "16. хүчингүй болгох")
	assert.Equal(t, domain.ValuationStatusDraft, sub.Status)

	// Өгөгдөл цэвэрлэгдсэн эсэх.
	for _, table := range []string{"assets", "compensation", "land_valuation"} {
		var count int
		require.NoError(t, f.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM `+table+` WHERE acquisition_id=$1`, f.acqID).Scan(&count))
		assert.Zero(t, count, "17. %s цэвэрлэгдэнэ", table)
	}

	// Файлууд хавсралтаас гарч, ЗӨВХӨН түүхэнд үлдэнэ.
	docs, err := f.repo.ListParcelDocuments(ctx, parcelID)
	require.NoError(t, err)
	assert.Empty(t, docs, "18. тайлан/ажлын зураг хавсралтаас хасагдана")

	snapshots, err := f.repo.ListValuationSnapshots(ctx, f.acqID, parcelID, domain.ValuationTypeAsset)
	require.NoError(t, err)
	require.Len(t, snapshots, 1, "19. түүхэн snapshot үүснэ")
	labels := map[string]string{}
	for _, file := range snapshots[0].Files {
		labels[file.Label] = file.URL
	}
	assert.Equal(t, report.FileURL, labels["Үнэлгээний тайлан"], "20. тайлан түүхэнд")
	assert.Equal(t, "/api/files/parcel/e2e/ff-photo.jpg", labels["Ажлын зураг"], "21. ажлын зураг түүхэнд")
	assert.Equal(t, domain.ParcelStatusEvaluating, func() int {
		var s int
		require.NoError(t, f.pool.QueryRow(ctx, `SELECT status FROM parcel WHERE id=$1`, parcelID).Scan(&s))
		return s
	}(), "22. нэгж талбар «Үнэлгээ хийх» рүү буцна")

	// ── 9. ШИНЭ ҮНЭЛГЭЭ ЦЭВЭР ХУУДСААС ──────────────────────────────────────
	_, err = f.repo.TransitionValuationSubmission(ctx, submit)
	assert.ErrorIs(t, err, domain.ErrSubmitReportRequired,
		"23. шинэ үнэлгээ өөрийн тайлангаа дахин хавсаргана")
}
