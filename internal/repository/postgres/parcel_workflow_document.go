package postgres

import (
	"context"
	"errors"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ── Шилжилтэд ЗААВАЛ байх хавсралт (parcel_workflow_required_document) ──────

// workflowRequiredDocs — шилжилт бүрийн заавал хавсралт (ListParcelWorkflows-д).
func (r *landAcqRepo) workflowRequiredDocs(ctx context.Context) (map[int][]domain.RequiredDocument, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rd.workflow_id, dt.id, dt.type, dt.name
		FROM parcel_workflow_required_document rd
		JOIN document_type dt ON dt.id = rd.document_type_id
		ORDER BY rd.workflow_id, dt.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int][]domain.RequiredDocument{}
	for rows.Next() {
		var wid int
		var d domain.RequiredDocument
		if err := rows.Scan(&wid, &d.DocumentTypeID, &d.DocumentTypeCode, &d.DocumentTypeName); err != nil {
			return nil, err
		}
		out[wid] = append(out[wid], d)
	}
	return out, rows.Err()
}

// SetParcelWorkflowRequiredDocuments — шилжилтийн заавал хавсралтыг БҮХЭЛД
// нь солино (нэг гүйлгээнд). Шилжилт байхгүй бол ErrNotFound; нэгж талбарт
// хамааралгүй (target) хавсралтын төрөл бол ErrInvalidInput.
func (r *landAcqRepo) SetParcelWorkflowRequiredDocuments(ctx context.Context, workflowID int, documentTypeIDs []int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM parcel_workflow WHERE id = $1)`, workflowID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return domain.ErrNotFound
	}
	ids := uniqueInts(documentTypeIDs)
	if len(ids) > 0 {
		var valid int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM document_type
			WHERE id = ANY($1) AND target IN ('parcel', 'both')`, ids).Scan(&valid); err != nil {
			return err
		}
		if valid != len(ids) {
			return domain.ErrInvalidInput
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM parcel_workflow_required_document WHERE workflow_id = $1`, workflowID); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO parcel_workflow_required_document (workflow_id, document_type_id)
			SELECT $1, unnest($2::int[])`, workflowID, ids); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ParcelStatusRequirements — нэгж талбарыг ОДООГИЙН төлөвөөс statusID руу
// шилжүүлэхэд (тэр шилжилтийн тохиргоо) шаардлагатай хавсралт бүр нэгж
// талбарт (parcel_document) орсон эсэх. Тохируулсан шилжилт байхгүй бол хоосон.
//
// «Үнэлгээний тайлан» (valuation_report) нь ХУУЧИН өгөгдөлд олговрын
// valuation_report_url-аар (сонгосон урсгалын зөвшөөрөгдсөн олговор)
// бүртгэгдсэн байж болох тул түүнийг ч орсонд тооцно — өмнөх «Чөлөөлсөн»
// шалгалттай ИЖИЛ дүрэм.
func (r *landAcqRepo) ParcelStatusRequirements(ctx context.Context, acquisitionID, parcelID uuid.UUID, statusID int) ([]domain.ParcelStatusRequirement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (dt.name, dt.id) dt.id, dt.type, dt.name,
			COALESCE(latest.name, legacy.name, ''),
			COALESCE(latest.file_url, legacy.file_url, ''),
			latest.name IS NOT NULL OR legacy.name IS NOT NULL
		FROM parcel p
		JOIN parcel_workflow pw ON pw.from_status_id = p.status AND pw.to_status_id = $3
		JOIN parcel_workflow_required_document rd ON rd.workflow_id = pw.id
		JOIN document_type dt ON dt.id = rd.document_type_id
		LEFT JOIN LATERAL (
			SELECT d.name, d.file_url FROM parcel_document pd
			JOIN document d ON d.id = pd.document_id
			WHERE pd.parcel_id = p.id AND d.document_type_id = dt.id
			ORDER BY d.uploaded_at DESC NULLS LAST LIMIT 1
		) latest ON true
		LEFT JOIN LATERAL (
			SELECT 'Үнэлгээний тайлан (олговрын хавсралт)'::text AS name, c.valuation_report_url AS file_url
			FROM compensation c
			WHERE dt.type = 'valuation_report' AND c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
			  AND c.status = 'approved' AND c.valuation_report_url <> ''
			  AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
			LIMIT 1
		) legacy ON true
		WHERE p.id = $2 AND p.acquisition_id = $1
		ORDER BY dt.name, dt.id`, acquisitionID, parcelID, statusID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.ParcelStatusRequirement, error) {
		var q domain.ParcelStatusRequirement
		return q, row.Scan(&q.DocumentTypeID, &q.DocumentTypeCode, &q.DocumentTypeName, &q.DocumentName, &q.DocumentURL, &q.Present)
	})
}

func uniqueInts(in []int) []int {
	seen := make(map[int]bool, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// ParcelProgressLocked — нэгж талбарын одоогийн төлөв «Явц түгжинэ»
// (locks_progress) эсэх. Нэгж талбар олдохгүй бол ErrNotFound.
func (r *landAcqRepo) ParcelProgressLocked(ctx context.Context, parcelID uuid.UUID) (bool, error) {
	var locked bool
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(ps.locks_progress, false)
		  FROM parcel p LEFT JOIN parcel_status ps ON ps.id = p.status
		 WHERE p.id = $1`, parcelID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, domain.ErrNotFound
	}
	return locked, err
}

// GetParcelStatusRules — төлөвийн дүрэм (migration 000049). Төлөв олдохгүй бол ErrNotFound.
func (r *landAcqRepo) GetParcelStatusRules(ctx context.Context, statusID int) (*domain.ParcelStatusRules, error) {
	var rules domain.ParcelStatusRules
	err := r.db.QueryRow(ctx, `SELECT is_final, locks_progress, is_valuation_stage, is_released, requires_reason
		FROM parcel_status WHERE id = $1`, statusID).
		Scan(&rules.IsFinal, &rules.LocksProgress, &rules.IsValuationStage, &rules.IsReleased, &rules.RequiresReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rules, nil
}
