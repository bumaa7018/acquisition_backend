//go:build e2e

package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func approvedValuationFixture(t *testing.T, parcelStatus int) (*fixture, domain.ValuationTransition) {
	t.Helper()
	ctx := context.Background()
	f := newFixture(t, 0, parcelStatus)
	parcelID := f.parcels[parcelStatus]
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT parcel_id FROM parcel WHERE id=$1`, parcelID).Scan(&code))
	_, err := f.pool.Exec(ctx, `UPDATE land_acquisition SET status=$2 WHERE id=$1`, f.acqID, domain.LandAcquisitionStatusField)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `INSERT INTO land_valuation (acquisition_id, parcel_id, land_area_m2, base_price_per_m2) VALUES ($1,$2,100,2500)`, f.acqID, code)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `INSERT INTO assets (acquisition_id, parcel_id, asset_name, photo_pdf_url) VALUES ($1,$2,'Original asset','/test/photo.pdf')`, f.acqID, code)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `INSERT INTO compensation (acquisition_id, parcel_id, amount, valuation_report_url) VALUES ($1,$2,250000,'/test/report.pdf')`, f.acqID, code)
	require.NoError(t, err)
	base := domain.ValuationTransition{
		AcquisitionID: f.acqID, ParcelID: parcelID, ValuationType: domain.ValuationTypeAsset,
		Action: "submit", Note: "Submit", UserID: "Submitting Employee",
	}
	sub, err := f.repo.TransitionValuationSubmission(ctx, base)
	require.NoError(t, err)
	assert.Equal(t, "Submitting Employee", sub.SubmittedBy)
	base.Action, base.Note, base.UserID = "approve", "Approve", "Approving Employee"
	sub, err = f.repo.TransitionValuationSubmission(ctx, base)
	require.NoError(t, err)
	assert.Equal(t, "Approving Employee", sub.ReviewedBy)
	base.Action, base.Note, base.UserID = "cancel", "Revalue", "Cancelling Employee"
	base.AttachmentURL, base.AttachmentName = "/test/cancel.pdf", "cancel.pdf"
	return f, base
}

func TestE2E_ValuationCancelAllParcelStatuses(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t)
	var hasAuthTable bool
	require.NoError(t, pool.QueryRow(ctx, `SELECT to_regclass('sdplatform.sd_user') IS NOT NULL`).Scan(&hasAuthTable))
	require.False(t, hasAuthTable, "this regression must run on appdb without the authdb user table")
	rows, err := pool.Query(ctx, `SELECT id FROM parcel_status ORDER BY id`)
	require.NoError(t, err)
	var statuses []int
	for rows.Next() {
		var status int
		require.NoError(t, rows.Scan(&status))
		statuses = append(statuses, status)
	}
	require.NoError(t, rows.Err())
	rows.Close()
	pool.Close()
	require.NotEmpty(t, statuses)
	for _, status := range statuses {
		t.Run(fmt.Sprintf("parcel_status_%d", status), func(t *testing.T) {
			f, cancel := approvedValuationFixture(t, status)
			sub, err := f.repo.GetValuationSubmission(ctx, f.acqID, cancel.ParcelID, "asset")
			require.NoError(t, err)
			assert.Equal(t, domain.ValuationStatusApproved, sub.Status)
			listed, total, err := f.repo.List(ctx, domain.LandAcquisitionFilter{
				PlanCode: f.planCode, HasReviewableValuation: true, Page: 1, PageSize: 10,
			})
			require.NoError(t, err)
			assert.Equal(t, int64(1), total, "finance must see acquisitions with approved valuations")
			require.Len(t, listed, 1)
			assert.Equal(t, f.acqID, listed[0].ID)
			sub, err = f.repo.TransitionValuationSubmission(ctx, cancel)
			require.NoError(t, err)
			assert.Equal(t, domain.ValuationStatusDraft, sub.Status)
			assert.Empty(t, sub.SubmittedBy)
			assert.Equal(t, cancel.AttachmentURL, sub.AttachmentURL)
			var parcelStatus int
			var selected *string
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT status, selected_valuation_type FROM parcel WHERE id=$1`, cancel.ParcelID).Scan(&parcelStatus, &selected))
			assert.Equal(t, domain.ParcelStatusEvaluating, parcelStatus)
			assert.Nil(t, selected)
			for _, table := range []string{"land_valuation", "assets", "compensation"} {
				var count int
				require.NoError(t, f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM `+table+` WHERE acquisition_id=$1`, f.acqID).Scan(&count))
				assert.Zero(t, count, table)
			}
			snapshots, err := f.repo.ListValuationSnapshots(ctx, f.acqID, cancel.ParcelID, "asset")
			require.NoError(t, err)
			require.Len(t, snapshots, 1)
			snapshot := snapshots[0]
			assert.Equal(t, "Cancelling Employee", snapshot.CancelledBy)
			assert.Equal(t, "Submitting Employee", snapshot.SubmittedBy)
			assert.Equal(t, "Approving Employee", snapshot.ReviewedBy)
			assert.Equal(t, float64(250000), snapshot.TotalAmount)
			var assets, compensations []map[string]any
			require.NoError(t, json.Unmarshal(snapshot.Assets, &assets))
			require.NoError(t, json.Unmarshal(snapshot.Compensations, &compensations))
			require.Len(t, assets, 1)
			require.Len(t, compensations, 1)
			assert.Equal(t, "Original asset", assets[0]["asset_name"])
			assert.Equal(t, "/test/report.pdf", compensations[0]["valuation_report_url"])
			// Үнэлгээний тайлан нь ТҮҮХЭНД үлдэнэ: олговрын мөр устсан ч
			// snapshot дээрх холбоосоор татагдана.
			assert.Equal(t, "/test/report.pdf", snapshot.ReportURL)
			assert.NotEmpty(t, snapshot.ReportName)
			history, err := f.repo.ListValuationSubmissionHistory(ctx, f.acqID, cancel.ParcelID, "asset")
			require.NoError(t, err)
			require.Len(t, history, 3)
			assert.Equal(t, "cancel", history[0].Action)
			assert.Equal(t, "Cancelling Employee", history[0].CreatedBy)
			assert.Equal(t, cancel.AttachmentURL, history[0].AttachmentURL)
			statusHistory, err := f.repo.ListParcelStatusHistory(ctx, cancel.ParcelID)
			require.NoError(t, err)
			require.Len(t, statusHistory, 1)
			assert.Equal(t, "Cancelling Employee", statusHistory[0].CreatedBy)
			_, err = f.repo.TransitionValuationSubmission(ctx, cancel)
			assert.ErrorIs(t, err, domain.ErrInvalidTransition)
			// A new round must not overwrite the cancelled snapshot.
			cancel.Action, cancel.UserID = "submit", "New Appraiser"
			sub, err = f.repo.TransitionValuationSubmission(ctx, cancel)
			require.NoError(t, err)
			assert.Equal(t, domain.ValuationStatusSubmitted, sub.Status)
			assert.Equal(t, "New Appraiser", sub.SubmittedBy)
			snapshots, err = f.repo.ListValuationSnapshots(ctx, f.acqID, cancel.ParcelID, "asset")
			require.NoError(t, err)
			require.Len(t, snapshots, 1)
			assert.Equal(t, "Submitting Employee", snapshots[0].SubmittedBy)
		})
	}
}

func TestE2E_ValuationCancelGuardsPreserveApprovedData(t *testing.T) {
	for _, scenario := range []string{"confirmed acquisition", "missing attachment", "report not required", "missing selected type"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			f, cancel := approvedValuationFixture(t, domain.ParcelStatusEvaluating)
			var expected error
			switch scenario {
			case "confirmed acquisition":
				_, err := f.pool.Exec(ctx, `UPDATE land_acquisition SET status=$2 WHERE id=$1`, f.acqID, domain.LandAcquisitionStatusVerified)
				require.NoError(t, err)
				expected = domain.ErrAcquisitionLocked
			case "missing attachment":
				cancel.AttachmentURL = ""
				expected = domain.ErrValuationCancelAttachmentRequired
			case "report not required":
				// Баталгаажсан тайлан нь нэгж талбарыг "Чөлөөлсөн" болгоход
				// шаардагдана; үнэлгээг ХҮЧИНГҮЙ болгоход зөвхөн цуцлалтын
				// хавсралт л шаардлагатай тул тайлангүй ч амжилттай явна.
				_, err := f.pool.Exec(ctx, `UPDATE compensation SET valuation_report_url='' WHERE acquisition_id=$1`, f.acqID)
				require.NoError(t, err)
			case "missing selected type":
				_, err := f.pool.Exec(ctx, `UPDATE parcel SET selected_valuation_type=NULL, status=1 WHERE id=$1`, cancel.ParcelID)
				require.NoError(t, err)
			}
			_, err := f.repo.TransitionValuationSubmission(ctx, cancel)
			if expected == nil {
				require.NoError(t, err)
				var status int
				require.NoError(t, f.pool.QueryRow(ctx, `SELECT status FROM parcel WHERE id=$1`, cancel.ParcelID).Scan(&status))
				assert.Equal(t, domain.ParcelStatusEvaluating, status)
				return
			}
			require.ErrorIs(t, err, expected)
			sub, err := f.repo.GetValuationSubmission(ctx, f.acqID, cancel.ParcelID, "asset")
			require.NoError(t, err)
			assert.Equal(t, domain.ValuationStatusApproved, sub.Status)
			snapshots, err := f.repo.ListValuationSnapshots(ctx, f.acqID, cancel.ParcelID, "asset")
			require.NoError(t, err)
			assert.Empty(t, snapshots)
			var count int
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM compensation WHERE acquisition_id=$1`, f.acqID).Scan(&count))
			assert.Equal(t, 1, count)
		})
	}
}

func TestE2E_ValuationStatusRejectsWrongAcquisition(t *testing.T) {
	f, cancel := approvedValuationFixture(t, domain.ParcelStatusEvaluating)
	_, err := f.repo.GetValuationSubmission(context.Background(), uuid.New(), cancel.ParcelID, "asset")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Үнэлгээний тайлан нь нэгж талбарын ХАВСРАЛТААР (parcel_document)
// бүртгэгдсэн бол цуцлахад хавсралтаас хасагдаж, зөвхөн snapshot дээрх
// холбоосоор татагдана — файл нь хадгалалтад үлдэнэ.
func TestE2E_ValuationCancelMovesReportDocumentToSnapshot(t *testing.T) {
	ctx := context.Background()
	f, cancel := approvedValuationFixture(t, domain.ParcelStatusEvaluating)

	var docTypeID int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT id FROM document_type WHERE type='valuation_report'`).Scan(&docTypeID))

	// Олговрын хуучин холбоосыг арилгаж ЗӨВХӨН хавсралтын флоуг шалгана.
	_, err := f.pool.Exec(ctx, `UPDATE compensation SET valuation_report_url='' WHERE acquisition_id=$1`, f.acqID)
	require.NoError(t, err)

	docID := uuid.New()
	const reportURL = "/api/files/parcel/e2e/report.pdf"
	_, err = f.pool.Exec(ctx, `
		INSERT INTO document (id, name, file_type, file_url, document_type_id)
		VALUES ($1, 'Үнэлгээний тайлан', 'application/pdf', $2, $3)`, docID, reportURL, docTypeID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx,
		`INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1,$2)`, cancel.ParcelID, docID)
	require.NoError(t, err)

	_, err = f.repo.TransitionValuationSubmission(ctx, cancel)
	require.NoError(t, err)

	snapshots, err := f.repo.ListValuationSnapshots(ctx, f.acqID, cancel.ParcelID, "asset")
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	assert.Equal(t, reportURL, snapshots[0].ReportURL)
	assert.Equal(t, "Үнэлгээний тайлан", snapshots[0].ReportName)

	var linked, docs int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM parcel_document WHERE parcel_id=$1`, cancel.ParcelID).Scan(&linked))
	assert.Zero(t, linked, "тайлан нэгж талбарын хавсралтаас хасагдана")
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM document WHERE id=$1`, docID).Scan(&docs))
	assert.Zero(t, docs)
}

// Цуцлахад ЭНЭ үнэлгээний БҮХ файл (Баталгаажсан тайлан, Тайлан, Үнэлгээний
// хүснэгт) нэгж талбарын хавсралт/илгээлтээс хасагдаж, ЗӨВХӨН тухайн
// snapshot-д холбогдоно. Дараагийн үнэлгээ цэвэр хуудаснаас эхэлнэ.
func TestE2E_ValuationCancelBindsAllFilesToSnapshot(t *testing.T) {
	ctx := context.Background()
	f, cancel := approvedValuationFixture(t, domain.ParcelStatusEvaluating)

	// Хуучин олговрын холбоосыг арилгаж ЗӨВХӨН хавсралтын флоуг шалгана.
	_, err := f.pool.Exec(ctx, `UPDATE compensation SET valuation_report_url='' WHERE acquisition_id=$1`, f.acqID)
	require.NoError(t, err)

	attach := func(docType, name, url string) uuid.UUID {
		t.Helper()
		var docTypeID int
		require.NoError(t, f.pool.QueryRow(ctx,
			`SELECT id FROM document_type WHERE type=$1`, docType).Scan(&docTypeID))
		docID := uuid.New()
		_, err := f.pool.Exec(ctx, `
			INSERT INTO document (id, name, file_type, file_url, document_type_id)
			VALUES ($1,$2,'application/pdf',$3,$4)`, docID, name, url, docTypeID)
		require.NoError(t, err)
		_, err = f.pool.Exec(ctx,
			`INSERT INTO parcel_document (parcel_id, document_id) VALUES ($1,$2)`, cancel.ParcelID, docID)
		require.NoError(t, err)
		return docID
	}
	const (
		reportURL = "/api/files/parcel/e2e/approved.pdf"
		sourceURL = "/api/files/parcel/e2e/valuation.xlsx"
		draftURL  = "/api/files/valuation-draft/e2e/draft.pdf"
	)
	reportDocID := attach("valuation_report", "Баталгаажсан.pdf", reportURL)
	sourceDocID := attach("valuation_source", "Үнэлгээ.xlsx", sourceURL)
	// Илгээхэд хавсаргасан тайлан нь parcel_document БИШ, илгээлтийн мөрөн дээр.
	_, err = f.pool.Exec(ctx, `
		UPDATE parcel_valuation_submission SET draft_report_url=$2, draft_report_name='Тайлан.pdf'
		WHERE acquisition_id=$1`, f.acqID, draftURL)
	require.NoError(t, err)

	_, err = f.repo.TransitionValuationSubmission(ctx, cancel)
	require.NoError(t, err)

	snapshots, err := f.repo.ListValuationSnapshots(ctx, f.acqID, cancel.ParcelID, "asset")
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	byLabel := map[string]domain.ValuationSnapshotFile{}
	for _, file := range snapshots[0].Files {
		byLabel[file.Label] = file
	}
	require.Len(t, byLabel, 3)
	assert.Equal(t, reportURL, byLabel["Баталгаажсан тайлан"].URL)
	assert.Equal(t, "Баталгаажсан.pdf", byLabel["Баталгаажсан тайлан"].Name)
	assert.Equal(t, draftURL, byLabel["Тайлан"].URL)
	assert.Equal(t, sourceURL, byLabel["Үнэлгээний хүснэгт"].URL)

	// Нэгж талбар дээр нэг ч хавсралт үлдэхгүй, илгээлтийн тайлан цэвэрлэгдэнэ.
	var linked, docs int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM parcel_document WHERE parcel_id=$1`, cancel.ParcelID).Scan(&linked))
	assert.Zero(t, linked)
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM document WHERE id=ANY($1)`,
		[]uuid.UUID{reportDocID, sourceDocID}).Scan(&docs))
	assert.Zero(t, docs)
	var leftoverDraft string
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT draft_report_url FROM parcel_valuation_submission WHERE acquisition_id=$1`,
		f.acqID).Scan(&leftoverDraft))
	assert.Empty(t, leftoverDraft, "хуучин тайлан шинэ үнэлгээн дээр үлдэхгүй")
}
