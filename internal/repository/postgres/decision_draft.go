package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DecisionDraftRepo struct {
	db *pgxpool.Pool
}

func NewDecisionDraftRepo(db *pgxpool.Pool) *DecisionDraftRepo {
	return &DecisionDraftRepo{db: db}
}

// decisionDraftSelect — жагсаалт болон нэгж бичлэг хоёулаа адилхан JOIN хийдэг
// тул SELECT/FROM хэсгийг нэг дор задална.
const decisionDraftSelect = `
	SELECT d.id, d.proposal_no, d.decree_number, d.decision_date, d.location,
	       d.duration_year, d.status, d.acquisition_id,
	       d.work_type_id, d.budget_id, d.current_progress_history_id,
	       d.confirmed_at, d.confirmed_by,
	       d.created_at, d.created_by, d.updated_at, d.updated_by,
	       COALESCE(la.acquisition_name, ''), COALESCE(la.plan_code, ''),
	       COALESCE(wt.name, ''), COALESCE(bg.name, ''),
	       (SELECT COUNT(*) FROM decision_draft_parcel ddp
	         WHERE ddp.decision_draft_id = d.id AND ddp.removed_at IS NULL),
	       COALESCE((SELECT SUM(COALESCE(NULLIF(p.acquisition_area_m2, 0), p.area_m2, 0))
	                   FROM decision_draft_parcel ddp
	                   JOIN parcel p ON p.id = ddp.parcel_id
	                  WHERE ddp.decision_draft_id = d.id AND ddp.removed_at IS NULL), 0),
	       (SELECT COUNT(*) FROM decision_draft_funding_source ddfs
	         WHERE ddfs.decision_draft_id = d.id),
	       COALESCE((SELECT string_agg(fs.organization_name || ' — ' || fs.source_type, ', '
	                                   ORDER BY fs.organization_name)
	                   FROM decision_draft_funding_source ddfs
	                   JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                  WHERE ddfs.decision_draft_id = d.id), ''),
	       COALESCE((SELECT SUM(COALESCE(ddfs.amount, 0))
	                   FROM decision_draft_funding_source ddfs
	                   JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                  WHERE ddfs.decision_draft_id = d.id
	                    AND NOT (
	                         COALESCE(fs.organization_name, '') ILIKE '%ОУ%'
	                      OR COALESCE(fs.organization_name, '') ILIKE '%СБХБ%'
	                      OR COALESCE(fs.organization_name, '') ILIKE '%тогтвортой авто зам%'
	                      OR COALESCE(fs.source_type, '') ILIKE '%ОУ%'
	                      OR COALESCE(fs.source_type, '') ILIKE '%СБХБ%'
	                      OR COALESCE(fs.source_type, '') ILIKE '%тогтвортой авто зам%'
	                      OR COALESCE(fs.currency, 'MNT') <> 'MNT'
	                    )), 0),
	       COALESCE((SELECT SUM(COALESCE(ddfs.amount, 0))
	                   FROM decision_draft_funding_source ddfs
	                   JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                  WHERE ddfs.decision_draft_id = d.id
	                    AND (
	                         COALESCE(fs.organization_name, '') ILIKE '%ОУ%'
	                      OR COALESCE(fs.organization_name, '') ILIKE '%СБХБ%'
	                      OR COALESCE(fs.organization_name, '') ILIKE '%тогтвортой авто зам%'
	                      OR COALESCE(fs.source_type, '') ILIKE '%ОУ%'
	                      OR COALESCE(fs.source_type, '') ILIKE '%СБХБ%'
	                      OR COALESCE(fs.source_type, '') ILIKE '%тогтвортой авто зам%'
	                      OR COALESCE(fs.currency, 'MNT') <> 'MNT'
	                    )), 0),
	       COALESCE((SELECT jsonb_object_agg(x.source_type, x.amount)::text
	                   FROM (
	                     SELECT COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй') AS source_type,
	                            SUM(COALESCE(ddfs.amount, 0)) AS amount
	                       FROM decision_draft_funding_source ddfs
	                       JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                      WHERE ddfs.decision_draft_id = d.id
	                      GROUP BY COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй')
	                   ) x), '{}'),
	       COALESCE((SELECT jsonb_object_agg(x.source_type, x.amount)::text
	                   FROM (
	                     SELECT COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй') AS source_type,
	                            SUM(COALESCE((
	                              SELECT SUM(c.amount)
	                                FROM compensation c
	                               WHERE c.acquisition_id = p.acquisition_id
	                                 AND c.parcel_id = p.parcel_id
	                                 AND c.compensation_type = 'cash'
	                                 AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
	                            ), 0)) AS amount
	                       FROM decision_draft_parcel ddp
	                       JOIN parcel p ON p.id = ddp.parcel_id
	                       JOIN decision_draft_funding_source ddfs ON ddfs.id = ddp.funding_link_id
	                       JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                      WHERE ddp.decision_draft_id = d.id
	                        AND ddp.removed_at IS NULL
	                      GROUP BY COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй')
	                   ) x), '{}'),
	       COALESCE((SELECT jsonb_object_agg(x.source_type, x.count)::text
	                   FROM (
	                     SELECT COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй') AS source_type,
	                            COUNT(*) AS count
	                       FROM decision_draft_parcel ddp
	                       JOIN decision_draft_funding_source ddfs ON ddfs.id = ddp.funding_link_id
	                       JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                      WHERE ddp.decision_draft_id = d.id
	                        AND ddp.removed_at IS NULL
	                      GROUP BY COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй')
	                   ) x), '{}'),
	       COALESCE((SELECT jsonb_object_agg(x.source_type, x.area)::text
	                   FROM (
	                     SELECT COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй') AS source_type,
	                            SUM(COALESCE(NULLIF(p.acquisition_area_m2, 0), p.area_m2, 0)) AS area
	                       FROM decision_draft_parcel ddp
	                       JOIN parcel p ON p.id = ddp.parcel_id
	                       JOIN decision_draft_funding_source ddfs ON ddfs.id = ddp.funding_link_id
	                       JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
	                      WHERE ddp.decision_draft_id = d.id
	                        AND ddp.removed_at IS NULL
	                      GROUP BY COALESCE(NULLIF(TRIM(fs.source_type), ''), 'Тодорхойгүй')
	                   ) x), '{}'),
	       COALESCE(ph.progress_type, ''),
	       CASE ph.progress_type
	         WHEN 'reviewing' THEN 'Хянагдаж буй'
	         WHEN 'confirming' THEN 'Баталгаажуулах'
	         ELSE ''
	       END,
	       COALESCE(ph.recipient, ''), ph.progress_date, COALESCE(ph.note, '')
	  FROM decision_draft d
	  LEFT JOIN land_acquisition la ON la.id = d.acquisition_id
	  LEFT JOIN decision_work_type wt ON wt.id = d.work_type_id
	  LEFT JOIN decision_budget    bg ON bg.id = d.budget_id
	  LEFT JOIN decision_draft_progress_history ph ON ph.id = d.current_progress_history_id`

func scanDecisionDraft(row pgx.Row) (*domain.DecisionDraft, error) {
	d := &domain.DecisionDraft{}
	var fundingSourceAmountsJSON string
	var fundingSourceCompensationsJSON string
	var fundingSourceParcelCountsJSON string
	var fundingSourceParcelAreasJSON string
	err := row.Scan(
		&d.ID, &d.ProposalNo, &d.DecreeNumber, &d.DecisionDate, &d.Location,
		&d.DurationYear, &d.Status, &d.AcquisitionID,
		&d.WorkTypeID, &d.BudgetID, &d.CurrentProgressHistoryID, &d.ConfirmedAt, &d.ConfirmedBy,
		&d.CreatedAt, &d.CreatedBy, &d.UpdatedAt, &d.UpdatedBy,
		&d.AcquisitionName, &d.PlanCode,
		&d.WorkTypeName, &d.BudgetName, &d.ParcelCount, &d.ParcelAreaM2,
		&d.FundingSourceCount, &d.FundingSourceNames,
		&d.FundingLocalAmount, &d.FundingInternationalAmount, &fundingSourceAmountsJSON,
		&fundingSourceCompensationsJSON, &fundingSourceParcelCountsJSON, &fundingSourceParcelAreasJSON,
		&d.CurrentProgressType, &d.CurrentProgressTypeName,
		&d.CurrentProgressRecipient, &d.CurrentProgressDate, &d.CurrentProgressNote,
	)
	if err != nil {
		return nil, err
	}
	d.FundingSourceAmounts = map[string]float64{}
	if fundingSourceAmountsJSON != "" {
		_ = json.Unmarshal([]byte(fundingSourceAmountsJSON), &d.FundingSourceAmounts)
	}
	d.FundingSourceCompensations = map[string]float64{}
	if fundingSourceCompensationsJSON != "" {
		_ = json.Unmarshal([]byte(fundingSourceCompensationsJSON), &d.FundingSourceCompensations)
	}
	d.FundingSourceParcelCounts = map[string]int{}
	if fundingSourceParcelCountsJSON != "" {
		_ = json.Unmarshal([]byte(fundingSourceParcelCountsJSON), &d.FundingSourceParcelCounts)
	}
	d.FundingSourceParcelAreas = map[string]float64{}
	if fundingSourceParcelAreasJSON != "" {
		_ = json.Unmarshal([]byte(fundingSourceParcelAreasJSON), &d.FundingSourceParcelAreas)
	}
	return d, nil
}

// buildDecisionDraftWhere — шүүлтийн WHERE хэсэг болон $N аргументуудыг бүтээнэ
// (ListGlobalParcels-ийн адил динамик хэв маяг).
func buildDecisionDraftWhere(f domain.DecisionDraftFilter) (string, []any) {
	where := "1=1"
	args := []any{}
	i := 1

	add := func(clause string, val any) {
		where += fmt.Sprintf(clause, i)
		args = append(args, val)
		i++
	}

	if f.ProposalNo != "" {
		add(" AND d.proposal_no ILIKE $%d", "%"+f.ProposalNo+"%")
	}
	if f.DecreeNumber != "" {
		add(" AND d.decree_number ILIKE $%d", "%"+f.DecreeNumber+"%")
	}
	if f.Location != "" {
		add(" AND d.location ILIKE $%d", "%"+f.Location+"%")
	}
	if f.ParcelID != "" {
		add(` AND EXISTS (SELECT 1 FROM decision_draft_parcel ddp
		                   JOIN parcel p ON p.id = ddp.parcel_id
		                  WHERE ddp.decision_draft_id = d.id
		                    AND ddp.removed_at IS NULL
		                    AND p.parcel_id ILIKE $%d)`, "%"+f.ParcelID+"%")
	}
	if f.DateFrom != "" {
		add(" AND d.decision_date >= $%d::date", f.DateFrom)
	}
	if f.DateTo != "" {
		add(" AND d.decision_date <= $%d::date", f.DateTo)
	}
	if f.DurationYear > 0 {
		add(" AND d.duration_year = $%d", f.DurationYear)
	}
	if f.Status > 0 {
		add(" AND d.status = $%d", f.Status)
	}
	if f.AcquisitionID != "" {
		add(" AND d.acquisition_id = $%d::uuid", f.AcquisitionID)
	}
	if f.FundingSourceID != "" {
		add(` AND EXISTS (SELECT 1 FROM decision_draft_funding_source ddfs
		                  WHERE ddfs.decision_draft_id = d.id
		                    AND ddfs.funding_source_id = $%d::uuid)`, f.FundingSourceID)
	}
	if f.WorkTypeID > 0 {
		add(" AND d.work_type_id = $%d", f.WorkTypeID)
	}
	if f.BudgetID > 0 {
		add(" AND d.budget_id = $%d", f.BudgetID)
	}
	return where, args
}

func (r *DecisionDraftRepo) List(ctx context.Context, f domain.DecisionDraftFilter) ([]*domain.DecisionDraft, int64, error) {
	where, args := buildDecisionDraftWhere(f)

	var total int64
	countSQL := `SELECT COUNT(*) FROM decision_draft d WHERE ` + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*domain.DecisionDraft{}, 0, nil
	}

	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	listSQL := decisionDraftSelect + " WHERE " + where +
		// Шинэ нь эхэнд. Хуучин мэдээллийн импорт бүгд ИЖИЛ created_at-тай тул
		// захирамжийн огноо, дугаараар, эцэст нь id-аар тогтвортой эрэмбэлнэ.
		fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d",
			orderClause(decisionDraftSortColumns, f.SortBy, f.SortDir,
				"d.created_at DESC, d.decision_date DESC NULLS LAST, d.proposal_no DESC NULLS LAST", "d.id"),
			len(args)+1, len(args)+2)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*domain.DecisionDraft, 0, pageSize)
	for rows.Next() {
		d, err := scanDecisionDraft(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, d)
	}
	return list, total, rows.Err()
}

func (r *DecisionDraftRepo) GetByID(ctx context.Context, id string) (*domain.DecisionDraft, error) {
	d, err := scanDecisionDraft(r.db.QueryRow(ctx, decisionDraftSelect+" WHERE d.id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *DecisionDraftRepo) Create(ctx context.Context, d *domain.DecisionDraft) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO decision_draft
		    (proposal_no, decree_number, decision_date, location, duration_year,
		     status, acquisition_id, work_type_id, budget_id,
		     created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)
		RETURNING id, created_at, updated_at`,
		d.ProposalNo, d.DecreeNumber, d.DecisionDate, d.Location, d.DurationYear,
		d.Status, d.AcquisitionID, d.WorkTypeID, d.BudgetID,
		d.CreatedBy,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *DecisionDraftRepo) Update(ctx context.Context, d *domain.DecisionDraft) error {
	res, err := r.db.Exec(ctx, `
		UPDATE decision_draft
		   SET proposal_no   = $1,
		       location      = $2,
		       duration_year = $3,
		       work_type_id  = $4,
		       budget_id     = $5,
		       updated_by    = $6,
		       updated_at    = NOW()
		 WHERE id = $7 AND status = $8`,
		d.ProposalNo, d.Location, d.DurationYear,
		d.WorkTypeID, d.BudgetID, d.UpdatedBy, d.ID, domain.DecisionDraftStatusDraft)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Confirm — "Явц нэмэх": захирамжийн дугаар/огноог бөглөж баталгаажуулна.
// status = 1 нөхцөлтэй тул давхар баталгаажуулах оролдлого чимээгүй өнгөрөхгүй.
//
// doc != nil бол НЭГ гүйлгээнд: хавсралтын document мөр үүсч, захирамжид
// (decision_draft.document_id) холбогдож, захирамжид ИДЭВХТЭЙ холбогдсон бүх
// нэгж талбарын хавсралт (parcel_document) болно — нэг файл, олон холбоос.
func (r *DecisionDraftRepo) Confirm(ctx context.Context, id, decreeNumber, decisionDate, note, actor string, doc *domain.Document) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	res, err := tx.Exec(ctx, `
		UPDATE decision_draft
		   SET decree_number = $1,
		       decision_date = $2::date,
		       status        = $3,
		       confirmed_at  = NOW(),
		       confirmed_by  = $4,
		       updated_by    = $4,
		       updated_at    = NOW()
		 WHERE id = $5 AND status = $6`,
		decreeNumber, decisionDate, domain.DecisionDraftStatusConfirmed, actor,
		id, domain.DecisionDraftStatusDraft)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	var historyID string
	err = tx.QueryRow(ctx, `
		INSERT INTO decision_draft_progress_history
		    (decision_draft_id, progress_type, recipient, progress_date, note, created_by)
		VALUES ($1, $2, '', $3::date, $4, $5)
		RETURNING id`,
		id, domain.DecisionDraftProgressConfirming, decisionDate, note, actor).Scan(&historyID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE decision_draft SET current_progress_history_id = $1 WHERE id = $2`,
		historyID, id)
	if err != nil {
		return err
	}
	if doc != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO document (id, name, file_type, file_url, size_bytes, note,
			                      uploaded_by, uploaded_at, document_type_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::int, 0))`,
			doc.ID, doc.Name, doc.FileType, doc.FileURL, doc.SizeBytes, doc.Note,
			doc.UploadedBy, doc.UploadedAt, doc.DocumentTypeID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE decision_draft SET document_id = $1 WHERE id = $2`, doc.ID, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parcel_document (parcel_id, document_id)
			SELECT DISTINCT dp.parcel_id, $2::uuid
			  FROM decision_draft_parcel dp
			 WHERE dp.decision_draft_id = $1 AND dp.removed_at IS NULL
			ON CONFLICT DO NOTHING`, id, doc.ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DecreeDocumentTypeID — «Захирамж» (decree) хавсралтын төрлийн id. Seeder
// нэмээгүй бол domain.ErrNotFound.
func (r *DecisionDraftRepo) DecreeDocumentTypeID(ctx context.Context) (int, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`SELECT id FROM document_type WHERE type = $1`, domain.DecreeDocumentType).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrNotFound
	}
	return id, err
}

// GetDocument — захирамжийн хавсралт файл; хавсралтгүй бол (nil, nil).
func (r *DecisionDraftRepo) GetDocument(ctx context.Context, draftID string) (*domain.DecisionDraftDocument, error) {
	d := &domain.DecisionDraftDocument{}
	err := r.db.QueryRow(ctx, `
		SELECT doc.id::text, doc.name, doc.file_type, doc.file_url, doc.size_bytes,
		       doc.uploaded_by, doc.uploaded_at
		  FROM decision_draft dd
		  JOIN document doc ON doc.id = dd.document_id
		 WHERE dd.id = $1`, draftID).
		Scan(&d.ID, &d.Name, &d.FileType, &d.FileURL, &d.SizeBytes, &d.UploadedBy, &d.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DecisionDraftRepo) AddProgressHistory(ctx context.Context, h *domain.DecisionDraftProgressHistory) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// «Хянагдаж буй газар» сонгосон бол recipient = нэгжийн нэр (түүх, тайланд).
	err = tx.QueryRow(ctx, `
		INSERT INTO decision_draft_progress_history
		    (decision_draft_id, progress_type, recipient, progress_date, note, created_by, review_unit_id)
		SELECT $1, $2, COALESCE(u.name, $3), $4::date, $5, $6, u.id
		  FROM (SELECT 1) one
		  LEFT JOIN decision_review_unit u ON u.id = $7
		 WHERE $7::int IS NULL OR u.id IS NOT NULL
		RETURNING id, created_at, recipient`,
		h.DecisionDraftID, h.ProgressType, h.Recipient, h.ProgressDate, h.Note, h.CreatedBy, h.ReviewUnitID,
	).Scan(&h.ID, &h.CreatedAt, &h.Recipient)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidInput // сонгосон нэгж олдсонгүй
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE decision_draft
		   SET current_progress_history_id = $1,
		       updated_by = $2,
		       updated_at = NOW()
		 WHERE id = $3`,
		h.ID, h.CreatedBy, h.DecisionDraftID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DecisionDraftRepo) ListProgressHistory(ctx context.Context, draftID string) ([]*domain.DecisionDraftProgressHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, decision_draft_id, progress_type,
		       CASE progress_type
		         WHEN 'reviewing' THEN 'Хянагдаж буй'
		         WHEN 'confirming' THEN 'Баталгаажуулах'
		         ELSE progress_type
		       END AS progress_type_name,
		       recipient, review_unit_id, progress_date, note, created_at, created_by
		  FROM decision_draft_progress_history
		 WHERE decision_draft_id = $1
		 ORDER BY progress_date DESC, created_at DESC`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*domain.DecisionDraftProgressHistory{}
	for rows.Next() {
		h := &domain.DecisionDraftProgressHistory{}
		if err := rows.Scan(
			&h.ID, &h.DecisionDraftID, &h.ProgressType, &h.ProgressTypeName,
			&h.Recipient, &h.ReviewUnitID, &h.ProgressDate, &h.Note, &h.CreatedAt, &h.CreatedBy,
		); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func (r *DecisionDraftRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM decision_draft WHERE id = $1 AND status = $2`,
		id, domain.DecisionDraftStatusDraft)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetAcquisition — эхний нэгж талбар холбогдоход захирамжийн чөлөөлөлтийг
// бөглөнө (acquisition_id IS NULL нөхцөлтэй тул давхар бичихгүй).
func (r *DecisionDraftRepo) SetAcquisition(ctx context.Context, id, acquisitionID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE decision_draft SET acquisition_id = $1 WHERE id = $2 AND acquisition_id IS NULL`,
		acquisitionID, id)
	return err
}

const decisionDraftParcelSelect = `
	SELECT ddp.id, ddp.decision_draft_id, ddp.parcel_id, ddp.linked_at,
	       ddp.linked_by, ddp.removed_at, ddp.removed_by,
	       ddp.funding_link_id, COALESCE(ddfs.funding_source_id::text, ''),
	       COALESCE(fs.organization_name, ''), COALESCE(fs.source_type, ''),
	       p.parcel_id, p.acquisition_id, COALESCE(la.acquisition_name, ''),
	       COALESCE(p.area_m2, 0), COALESCE(p.acquisition_area_m2, 0),
	       COALESCE((
	         SELECT SUM(c.amount)
	           FROM compensation c
	          WHERE c.acquisition_id = p.acquisition_id
	            AND c.parcel_id = p.parcel_id
	            AND c.compensation_type = 'cash'
	            AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
	       ), 0),
	       p.status, COALESCE(ps.name, ''), COALESCE(p.landuse, '')
	  FROM decision_draft_parcel ddp
	  JOIN parcel p ON p.id = ddp.parcel_id
	  LEFT JOIN land_acquisition la ON la.id = p.acquisition_id
	  LEFT JOIN parcel_status    ps ON ps.id = p.status
	  LEFT JOIN decision_draft_funding_source ddfs ON ddfs.id = ddp.funding_link_id
	  LEFT JOIN funding_sources fs ON fs.id = ddfs.funding_source_id`

func scanDecisionDraftParcel(rows pgx.Rows) (*domain.DecisionDraftParcel, error) {
	l := &domain.DecisionDraftParcel{}
	err := rows.Scan(
		&l.ID, &l.DecisionDraftID, &l.ParcelUUID, &l.LinkedAt,
		&l.LinkedBy, &l.RemovedAt, &l.RemovedBy,
		&l.FundingLinkID, &l.FundingSourceID, &l.FundingOrganization, &l.FundingSourceType,
		&l.ParcelID, &l.AcquisitionID, &l.AcquisitionName,
		&l.AreaM2, &l.AcquisitionAreaM2, &l.CompensationAmount,
		&l.ParcelStatus, &l.ParcelStatusName, &l.Landuse,
	)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// ListParcels — захирамжид холбогдсон нэгж талбарууд. Идэвхтэй холбоос эхэнд,
// хасагдсан (түүх) нь ард.
func (r *DecisionDraftRepo) ListParcels(ctx context.Context, draftID string) ([]*domain.DecisionDraftParcel, error) {
	rows, err := r.db.Query(ctx,
		decisionDraftParcelSelect+`
		 WHERE ddp.decision_draft_id = $1
		 ORDER BY (ddp.removed_at IS NOT NULL), ddp.linked_at DESC`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DecisionDraftParcel
	for rows.Next() {
		l, err := scanDecisionDraftParcel(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// ListByParcel — нэгж талбарын "Захирамж" таб. Захирамжийн мэдээллийг хамт
// буцаана.
func (r *DecisionDraftRepo) ListByParcel(ctx context.Context, parcelUUID string) ([]*domain.DecisionDraftParcel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ddp.id, ddp.decision_draft_id, ddp.parcel_id, ddp.linked_at,
		       ddp.linked_by, ddp.removed_at, ddp.removed_by,
		       p.parcel_id, p.acquisition_id, COALESCE(la.acquisition_name, ''),
		       COALESCE(p.area_m2, 0), COALESCE(p.acquisition_area_m2, 0),
		       COALESCE((
		         SELECT SUM(c.amount)
		           FROM compensation c
		          WHERE c.acquisition_id = p.acquisition_id
		            AND c.parcel_id = p.parcel_id
		            AND c.compensation_type = 'cash'
		            AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
		       ), 0),
		       p.status, COALESCE(ps.name, ''), COALESCE(p.landuse, ''),
		       d.proposal_no, d.decree_number, d.decision_date, d.status,
		       COALESCE(wt.name, ''), COALESCE(bg.name, ''), d.location, d.duration_year,
		       COALESCE(ph.progress_type, ''),
		       CASE ph.progress_type
		         WHEN 'reviewing' THEN 'Хянагдаж буй'
		         WHEN 'confirming' THEN 'Баталгаажуулах'
		         ELSE ''
		       END,
		       COALESCE(ph.recipient, ''), ph.progress_date, COALESCE(ph.note, '')
		  FROM decision_draft_parcel ddp
		  JOIN decision_draft d ON d.id = ddp.decision_draft_id
		  JOIN parcel p ON p.id = ddp.parcel_id
		  LEFT JOIN land_acquisition   la ON la.id = p.acquisition_id
		  LEFT JOIN parcel_status      ps ON ps.id = p.status
		  LEFT JOIN decision_work_type wt ON wt.id = d.work_type_id
		  LEFT JOIN decision_budget    bg ON bg.id = d.budget_id
		  LEFT JOIN decision_draft_progress_history ph ON ph.id = d.current_progress_history_id
		 WHERE ddp.parcel_id = $1
		 ORDER BY (ddp.removed_at IS NOT NULL), ddp.linked_at DESC`, parcelUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DecisionDraftParcel
	for rows.Next() {
		l := &domain.DecisionDraftParcel{}
		if err := rows.Scan(
			&l.ID, &l.DecisionDraftID, &l.ParcelUUID, &l.LinkedAt,
			&l.LinkedBy, &l.RemovedAt, &l.RemovedBy,
			&l.ParcelID, &l.AcquisitionID, &l.AcquisitionName,
			&l.AreaM2, &l.AcquisitionAreaM2, &l.CompensationAmount,
			&l.ParcelStatus, &l.ParcelStatusName, &l.Landuse,
			&l.ProposalNo, &l.DecreeNumber, &l.DecisionDate, &l.DecisionStatus,
			&l.WorkTypeName, &l.BudgetName, &l.Location, &l.DurationYear,
			&l.CurrentProgressType, &l.CurrentProgressTypeName,
			&l.CurrentProgressRecipient, &l.CurrentProgressDate, &l.CurrentProgressNote,
		); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// GetParcelInfo — холбохоос өмнөх шалгалтад хэрэгтэй хамгийн бага мэдээлэл.
func (r *DecisionDraftRepo) GetParcelInfo(ctx context.Context, parcelUUID string) (*domain.ParcelLinkInfo, error) {
	info := &domain.ParcelLinkInfo{}
	err := r.db.QueryRow(ctx,
		`SELECT p.id, p.parcel_id, p.status, p.acquisition_id, COALESCE(ps.is_released, false)
		   FROM parcel p LEFT JOIN parcel_status ps ON ps.id = p.status
		  WHERE p.id = $1`, parcelUUID).
		Scan(&info.ID, &info.ParcelID, &info.Status, &info.AcquisitionID, &info.IsReleased)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return info, nil
}

// LinkParcel — идэвхтэй холбоос үүсгэнэ. uq_ddp_active_parcel partial unique
// index-ийн зөрчлийг ErrParcelAlreadyLinked болгож хөрвүүлнэ (гэнэтийн
// зэрэгцээ хүсэлтээс хамгаална).
func (r *DecisionDraftRepo) LinkParcel(ctx context.Context, draftID, parcelUUID, fundingLinkID, actor string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO decision_draft_parcel (decision_draft_id, parcel_id, funding_link_id, linked_by)
		VALUES ($1, $2, $3, $4) RETURNING id`, draftID, parcelUUID, fundingLinkID, actor).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "uq_ddp_active_parcel") {
			return "", domain.ErrParcelAlreadyLinked
		}
		return "", err
	}
	return id, nil
}

// UnlinkParcel — түүх үлдээж хасна (мөр устгахгүй).
func (r *DecisionDraftRepo) UnlinkParcel(ctx context.Context, draftID, linkID, actor string) error {
	res, err := r.db.Exec(ctx, `
		UPDATE decision_draft_parcel
		   SET removed_at = NOW(), removed_by = $1
		 WHERE id = $2 AND decision_draft_id = $3 AND removed_at IS NULL`,
		actor, linkID, draftID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListDraftFundingSources — захирамжид нэмэгдсэн санхүүгийн эх үүсвэрүүд.
func (r *DecisionDraftRepo) ListDraftFundingSources(ctx context.Context, draftID string) ([]*domain.DecisionDraftFundingLink, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ddfs.id, ddfs.decision_draft_id, ddfs.funding_source_id,
		       ddfs.created_at, ddfs.created_by,
		       COALESCE(fs.acquisition_id::text, ''), COALESCE(la.acquisition_name, ''),
		       fs.organization_name, fs.source_type, ddfs.amount,
		       COALESCE(fs.currency, 'MNT'), COALESCE(fs.note, '')
		  FROM decision_draft_funding_source ddfs
		  JOIN funding_sources fs ON fs.id = ddfs.funding_source_id
		  LEFT JOIN land_acquisition la ON la.id = fs.acquisition_id
		 WHERE ddfs.decision_draft_id = $1
		 ORDER BY ddfs.created_at DESC, ddfs.id`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*domain.DecisionDraftFundingLink{}
	for rows.Next() {
		l := &domain.DecisionDraftFundingLink{}
		if err := rows.Scan(
			&l.ID, &l.DecisionDraftID, &l.FundingSourceID,
			&l.CreatedAt, &l.CreatedBy,
			&l.AcquisitionID, &l.AcquisitionName,
			&l.OrganizationName, &l.SourceType, &l.Amount, &l.Currency, &l.Note,
		); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// AddFundingSource — эх үүсвэр нэмнэ. Давхардвал UNIQUE зөрчлийг тодорхой
// алдаа болгож хөрвүүлнэ.
func (r *DecisionDraftRepo) AddFundingSource(ctx context.Context, draftID, fundingSourceID string, amount float64, actor string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO decision_draft_funding_source
		    (decision_draft_id, funding_source_id, amount, created_by)
		VALUES ($1, $2, $3, $4)`, draftID, fundingSourceID, amount, actor)
	if err != nil {
		// Постгрес хязгаарлалтын нэрийг 63 тэмдэгтээр таслах тул бүтэн нэрээр
		// биш, баттай агуулагдах угтвараар шалгана.
		if strings.Contains(err.Error(), "decision_draft_funding_source_decision_draft_id") {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// FundingLinkBelongsTo — сонгосон эх үүсвэрийн холбоос ЯГ энэ захирамжийнх
// эсэхийг шалгана (өөр захирамжийн эх үүсвэрийг оноохоос сэргийлнэ).
func (r *DecisionDraftRepo) FundingLinkBelongsTo(ctx context.Context, draftID, fundingLinkID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM decision_draft_funding_source
		                 WHERE id = $1 AND decision_draft_id = $2)`,
		fundingLinkID, draftID).Scan(&exists)
	return exists, err
}

// FundingLinkInUse — тухайн эх үүсвэрийг идэвхтэй нэгж талбар ашиглаж байгаа
// эсэх. Ашиглагдаж байвал захирамжаас хасахыг зөвшөөрөхгүй.
func (r *DecisionDraftRepo) FundingLinkInUse(ctx context.Context, fundingLinkID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM decision_draft_parcel
		                 WHERE funding_link_id = $1 AND removed_at IS NULL)`,
		fundingLinkID).Scan(&exists)
	return exists, err
}

// SetParcelFundingLink — холбогдсон нэгж талбарын санхүүгийн эх үүсвэрийг солино.
func (r *DecisionDraftRepo) SetParcelFundingLink(ctx context.Context, draftID, linkID, fundingLinkID string) error {
	res, err := r.db.Exec(ctx, `
		UPDATE decision_draft_parcel
		   SET funding_link_id = $1
		 WHERE id = $2 AND decision_draft_id = $3 AND removed_at IS NULL`,
		fundingLinkID, linkID, draftID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// BindFundingSourceToAcquisition — захирамжийн төсөлд нэгж талбар холбоход
// сонгосон санхүүгийн эх үүсвэрийг ТУХАЙН ЧӨЛӨӨЛӨЛТӨД хадгална.
//
// Чөлөөлөлт бүртгэхэд эх үүсвэр асуухаа больсон тул чөлөөлөлтийн санхүүжилтийн
// мэдээлэл ЗӨВХӨН эндээс бүрдэнэ. Хоёр тохиолдол:
//
//   - Эх үүсвэр нь чөлөөлөлтгүй (бие даасан, acquisition_id IS NULL) бол шууд
//     тухайн чөлөөлөлтөд холбоно.
//   - ӨӨР чөлөөлөлтийнх бол зөөж авахгүй (тэр чөлөөлөлтийн санхүүжилт эвдэрнэ)
//     — харин ижил нэр/төрлийн мөр байхгүй бол хуулбарыг үүсгэнэ.
//
// Идемпотент: дахин дуудахад давхардал үүсгэхгүй.
func (r *DecisionDraftRepo) BindFundingSourceToAcquisition(ctx context.Context, fundingLinkID, acquisitionID string) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE funding_sources fs
		   SET acquisition_id = $2
		  FROM decision_draft_funding_source ddfs
		 WHERE ddfs.id = $1
		   AND fs.id = ddfs.funding_source_id
		   AND fs.acquisition_id IS NULL
	`, fundingLinkID, acquisitionID); err != nil {
		return err
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO funding_sources (
			acquisition_id, organization_name, source_type, amount, currency, note, created_by
		)
		SELECT $2, src.organization_name, src.source_type, src.amount, src.currency, src.note, src.created_by
		  FROM decision_draft_funding_source ddfs
		  JOIN funding_sources src ON src.id = ddfs.funding_source_id
		 WHERE ddfs.id = $1
		   AND src.acquisition_id IS NOT NULL
		   AND src.acquisition_id <> $2
		   AND NOT EXISTS (
		       SELECT 1 FROM funding_sources dup
		        WHERE dup.acquisition_id = $2
		          AND dup.organization_name = src.organization_name
		          AND dup.source_type = src.source_type
		   )
	`, fundingLinkID, acquisitionID)
	return err
}

func (r *DecisionDraftRepo) RemoveFundingSource(ctx context.Context, draftID, linkID string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM decision_draft_funding_source WHERE id = $1 AND decision_draft_id = $2`,
		linkID, draftID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CreateFundingSourceOption — захирамжийн төслөөс ГАРААР шинэ санхүүгийн эх
// үүсвэр бүртгэнэ. Чөлөөлөлтгүй (acquisition_id IS NULL) бие даасан мөр
// болж funding_sources хүснэгтэд бичигдэнэ.
func (r *DecisionDraftRepo) CreateFundingSourceOption(ctx context.Context, o *domain.FundingSourceOption, note, actor string) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO funding_sources
		    (acquisition_id, organization_name, source_type, amount, currency, note, created_by)
		VALUES (NULL, $1, $2, $3, $4, $5, $6)
		RETURNING id`,
		o.OrganizationName, o.SourceType, o.Amount, o.Currency, note, actor,
	).Scan(&o.ID)
}

// ListFundingSourceOptions — бүх чөлөөлөлтийн санхүүгийн эх үүсвэрийг
// сонголтын жагсаалт болгон буцаана. funding_sources-т GET маршрут байхгүй
// (чөлөөлөлтийн дэлгэрэнгүйн хамт л ирдэг) бөгөөд захирамжийн төслийг
// үүсгэх үед аль чөлөөлөлтөд харьяалагдах нь хараахан тодорхойгүй тул
// бүгдийг чөлөөлөлтийн нэрийнх нь хамт харуулна.
func (r *DecisionDraftRepo) ListFundingSourceOptions(ctx context.Context) ([]*domain.FundingSourceOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT fs.id, COALESCE(fs.acquisition_id::text, ''), COALESCE(la.acquisition_name, ''),
		       COALESCE(la.plan_code, ''), fs.organization_name, fs.source_type,
		       fs.amount, COALESCE(fs.currency, 'MNT')
		  FROM funding_sources fs
		  LEFT JOIN land_acquisition la ON la.id = fs.acquisition_id
		 ORDER BY la.acquisition_name NULLS LAST, fs.organization_name, fs.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*domain.FundingSourceOption{}
	for rows.Next() {
		o := &domain.FundingSourceOption{}
		if err := rows.Scan(&o.ID, &o.AcquisitionID, &o.AcquisitionName,
			&o.PlanCode, &o.OrganizationName, &o.SourceType, &o.Amount, &o.Currency); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// ActiveLinkOf — нэгж талбар аль захирамжид идэвхтэй холбогдсоныг буцаана
// (олдоогүй бол "" ба nil).
func (r *DecisionDraftRepo) ActiveLinkOf(ctx context.Context, parcelUUID string) (string, error) {
	var draftID string
	err := r.db.QueryRow(ctx,
		`SELECT decision_draft_id FROM decision_draft_parcel
		  WHERE parcel_id = $1 AND removed_at IS NULL LIMIT 1`, parcelUUID).Scan(&draftID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return draftID, nil
}
