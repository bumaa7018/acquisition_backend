package postgres

import (
	"context"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// ReplaceParcelMortgages — нэгж талбарын барьцааны бүртгэлийг ГУС-аас ирсэн
// жагсаалтаар БҮРЭН шинэчилнэ. Барьцаа нь эх системд цуцлагдаж/өөрчлөгдөж
// болох тул нэмэх биш ОРЛУУЛАХ семантик (эзэмшигчтэй ижил).
func (r *landAcqRepo) ReplaceParcelMortgages(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	appNo string,
	items []*domain.ParcelMortgage,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ДАВХАРДУУЛАХГҮЙ: байгаа мөрийг ШИНЭЧИЛНЭ (id хэвээр), эх системд
	// байхгүй болсныг л устгана. Бүгдийг устгаад дахин үүсгэвэл мөр бүр
	// шинэ id авч, гадны лавлагаа тасарна.
	keep := make([]string, 0, len(items))
	for _, m := range items {
		if m != nil && m.SourceID != "" {
			keep = append(keep, m.SourceID)
		}
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM parcel_mortgage WHERE parcel_id = $1 AND NOT (source_id = ANY($2))",
		parcelID, keep,
	); err != nil {
		return 0, err
	}

	saved := 0
	for _, m := range items {
		if m == nil || m.SourceID == "" {
			continue
		}
		itemAppNo := appNo
		if itemAppNo == "" {
			itemAppNo = m.AppNo
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_mortgage (
				parcel_id, acquisition_id, source_id, source_parent_id, app_no,
				start_mortgage_period, end_mortgage_period,
				loan_contract_no, mortgage_contract_no,
				loan_contract_amount, monetary_unit_value,
				unit_type, mortgage_type, status_id, status_name
			) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''),
			          NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''),
			          $10, $11, NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''))
			ON CONFLICT (parcel_id, source_id) DO UPDATE SET
				acquisition_id        = EXCLUDED.acquisition_id,
				source_parent_id      = EXCLUDED.source_parent_id,
				app_no                = EXCLUDED.app_no,
				start_mortgage_period = EXCLUDED.start_mortgage_period,
				end_mortgage_period   = EXCLUDED.end_mortgage_period,
				loan_contract_no      = EXCLUDED.loan_contract_no,
				mortgage_contract_no  = EXCLUDED.mortgage_contract_no,
				loan_contract_amount  = EXCLUDED.loan_contract_amount,
				monetary_unit_value   = EXCLUDED.monetary_unit_value,
				unit_type             = EXCLUDED.unit_type,
				mortgage_type         = EXCLUDED.mortgage_type,
				status_id             = EXCLUDED.status_id,
				status_name           = EXCLUDED.status_name,
				updated_at            = NOW()
		`, parcelID, acquisitionID, m.SourceID, m.SourceParentID, itemAppNo,
			m.StartMortgagePeriod, m.EndMortgagePeriod,
			m.LoanContractNo, m.MortgageContractNo,
			m.LoanContractAmount, m.MonetaryUnitValue,
			m.UnitType, m.MortgageType, m.StatusID, m.StatusName,
		); err != nil {
			return saved, err
		}
		saved++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
}

// ListParcelMortgages — эцэг барьцаа ЭХЭНД, давхар барьцаа нь эцгийнхээ ард.
func (r *landAcqRepo) ListParcelMortgages(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelMortgage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, source_id, COALESCE(source_parent_id,''), COALESCE(app_no,''),
		       COALESCE(start_mortgage_period,''), COALESCE(end_mortgage_period,''),
		       COALESCE(loan_contract_no,''), COALESCE(mortgage_contract_no,''),
		       COALESCE(loan_contract_amount,0), COALESCE(monetary_unit_value,0),
		       COALESCE(unit_type,''), COALESCE(mortgage_type,''),
		       COALESCE(status_id,''), COALESCE(status_name,'')
		FROM parcel_mortgage
		WHERE parcel_id = $1
		ORDER BY COALESCE(source_parent_id, source_id), source_parent_id NULLS FIRST, source_id
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelMortgage
	for rows.Next() {
		m := domain.ParcelMortgage{}
		if err := rows.Scan(&m.ID, &m.SourceID, &m.SourceParentID, &m.AppNo,
			&m.StartMortgagePeriod, &m.EndMortgagePeriod,
			&m.LoanContractNo, &m.MortgageContractNo,
			&m.LoanContractAmount, &m.MonetaryUnitValue,
			&m.UnitType, &m.MortgageType, &m.StatusID, &m.StatusName); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// ReplaceParcelCourtDecisions — шүүхийн шийдвэрийг бүрэн шинэчилнэ.
func (r *landAcqRepo) ReplaceParcelCourtDecisions(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	appNo string,
	items []*domain.ParcelCourtDecision,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	keep := make([]string, 0, len(items))
	for _, d := range items {
		if d != nil && d.SourceID != "" {
			keep = append(keep, d.SourceID)
		}
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM parcel_court_decision WHERE parcel_id = $1 AND NOT (source_id = ANY($2))",
		parcelID, keep,
	); err != nil {
		return 0, err
	}

	saved := 0
	for _, d := range items {
		if d == nil || d.SourceID == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_court_decision (
				parcel_id, acquisition_id, source_id, app_no,
				start_period, end_period, court_status, status_name, court_decision_no
			) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
			          NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''))
			ON CONFLICT (parcel_id, source_id) DO UPDATE SET
				acquisition_id    = EXCLUDED.acquisition_id,
				app_no            = EXCLUDED.app_no,
				start_period      = EXCLUDED.start_period,
				end_period        = EXCLUDED.end_period,
				court_status      = EXCLUDED.court_status,
				status_name       = EXCLUDED.status_name,
				court_decision_no = EXCLUDED.court_decision_no,
				updated_at        = NOW()
		`, parcelID, acquisitionID, d.SourceID, appNo,
			d.StartPeriod, d.EndPeriod, d.CourtStatus, d.StatusName, d.CourtDecisionNo,
		); err != nil {
			return saved, err
		}
		saved++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
}

func (r *landAcqRepo) ListParcelCourtDecisions(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelCourtDecision, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, source_id, COALESCE(app_no,''), COALESCE(start_period,''),
		       COALESCE(end_period,''), COALESCE(court_status,''),
		       COALESCE(status_name,''), COALESCE(court_decision_no,'')
		FROM parcel_court_decision
		WHERE parcel_id = $1
		ORDER BY start_period DESC NULLS LAST, source_id
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelCourtDecision
	for rows.Next() {
		d := domain.ParcelCourtDecision{}
		if err := rows.Scan(&d.ID, &d.SourceID, &d.AppNo, &d.StartPeriod,
			&d.EndPeriod, &d.CourtStatus, &d.StatusName, &d.CourtDecisionNo); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// ReplaceParcelInvoices — төлбөрийн нэхэмжлэлийг ДАВХАРДУУЛАХГҮЙ шинэчилнэ:
// invoice_no-гоор upsert, эх системд байхгүй болсныг устгана.
func (r *landAcqRepo) ReplaceParcelInvoices(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	items []*domain.ParcelInvoice,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	keep := make([]string, 0, len(items))
	for _, v := range items {
		if v != nil && v.InvoiceNo != "" {
			keep = append(keep, v.InvoiceNo)
		}
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM parcel_invoice WHERE parcel_id = $1 AND NOT (invoice_no = ANY($2))",
		parcelID, keep,
	); err != nil {
		return 0, err
	}

	saved := 0
	for _, v := range items {
		if v == nil || v.InvoiceNo == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_invoice (
				parcel_id, acquisition_id, invoice_no, description,
				amount, paid_amount, status_id, status_name
			) VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, NULLIF($8, ''))
			ON CONFLICT (parcel_id, invoice_no) DO UPDATE SET
				acquisition_id = EXCLUDED.acquisition_id,
				description    = EXCLUDED.description,
				amount         = EXCLUDED.amount,
				paid_amount    = EXCLUDED.paid_amount,
				status_id      = EXCLUDED.status_id,
				status_name    = EXCLUDED.status_name,
				updated_at     = NOW()
		`, parcelID, acquisitionID, v.InvoiceNo, v.Description,
			v.Amount, v.PaidAmount, v.StatusID, v.StatusName,
		); err != nil {
			return saved, err
		}
		saved++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
}

func (r *landAcqRepo) ListParcelInvoices(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelInvoice, error) {
	rows, err := r.db.Query(ctx, `
		SELECT invoice_no, COALESCE(description,''), COALESCE(amount,0),
		       COALESCE(paid_amount,0), COALESCE(status_id,0), COALESCE(status_name,'')
		FROM parcel_invoice WHERE parcel_id = $1 ORDER BY invoice_no
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelInvoice
	for rows.Next() {
		v := domain.ParcelInvoice{}
		if err := rows.Scan(&v.InvoiceNo, &v.Description, &v.Amount,
			&v.PaidAmount, &v.StatusID, &v.StatusName); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// UpsertParcelMonitorings — monitoring_id-гаар давхардуулахгүй шинэчилнэ.
func (r *landAcqRepo) UpsertParcelMonitorings(
	ctx context.Context,
	parcelID, acquisitionID uuid.UUID,
	items []*domain.ParcelMonitoring,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	keep := make([]string, 0, len(items))
	for _, m := range items {
		if m != nil && m.MonitoringID != "" {
			keep = append(keep, m.MonitoringID)
		}
	}
	if _, err := tx.Exec(ctx,
		"DELETE FROM parcel_monitoring WHERE parcel_id = $1 AND NOT (monitoring_id = ANY($2))",
		parcelID, keep,
	); err != nil {
		return 0, err
	}

	saved := 0
	for _, m := range items {
		if m == nil || m.MonitoringID == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_monitoring (
				parcel_id, acquisition_id, monitoring_id, page_no,
				status_name, company_name, source_created_at
			) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''))
			ON CONFLICT (monitoring_id) DO UPDATE SET
				parcel_id          = EXCLUDED.parcel_id,
				acquisition_id     = EXCLUDED.acquisition_id,
				page_no            = EXCLUDED.page_no,
				status_name        = EXCLUDED.status_name,
				company_name       = EXCLUDED.company_name,
				source_created_at  = EXCLUDED.source_created_at,
				updated_at         = NOW()
		`, parcelID, acquisitionID, m.MonitoringID, m.PageNo, m.StatusName, m.CompanyName, m.CreatedAt); err != nil {
			return saved, err
		}
		saved++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return saved, nil
}

func (r *landAcqRepo) ListParcelMonitorings(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelMonitoring, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, monitoring_id, COALESCE(page_no,''), COALESCE(status_name,''),
		       COALESCE(company_name,''), COALESCE(source_created_at,'')
		FROM parcel_monitoring
		WHERE parcel_id = $1
		ORDER BY source_created_at DESC NULLS LAST, monitoring_id
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParcelMonitoring
	for rows.Next() {
		m := domain.ParcelMonitoring{}
		if err := rows.Scan(&m.ID, &m.MonitoringID, &m.PageNo, &m.StatusName, &m.CompanyName, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
