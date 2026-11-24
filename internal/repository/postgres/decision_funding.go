package postgres

import (
	"context"
	"fmt"
	"strings"

	"acquisition_backend/internal/domain"
)

// decisionFundingPaidRatio — нэгж талбарын нөхөх олговрын ОЛГОГДСОН хувь
// («Нөхөх олговор олгосон баримт»-ын шат, syncParcelCompensationPaid):
// бүрэн эсвэл 60+40 → 100%, зөвхөн 60 → 60%, зөвхөн 40 → 40%. Шатгүй ч
// хуучин импортын `compensation_paid` туг тавигдсан бол бүрэн олгосон.
const decisionFundingPaidRatio = `
	CASE WHEN p.compensation_paid_full OR (p.compensation_paid_60 AND p.compensation_paid_40) THEN 1.0
	     WHEN p.compensation_paid_60 THEN 0.6
	     WHEN p.compensation_paid_40 THEN 0.4
	     WHEN p.compensation_paid THEN 1.0
	     ELSE 0 END`

// GetDecisionFunding — захирамж ба санхүүжилтийн нэгтгэл (хяналтын самбар,
// чөлөөлөлтийн «Санхүүжилт» таб).
//
// ОН нь САНХҮҮЖИЛТИЙН он: нэгж талбар захирамжид холбогдсон санхүүжилтийн
// холбоосынхоо (decision_draft_funding_source.year) онд хамаарна — тиймээс
// «төлөвлөсөн санхүүжилт» ба түүнээс зарцуулсан / олгосон дүн ижил онд
// уялдана. Холбоосгүй (хуучин) нэгж талбар захирамжийн «Хугацаа (он)»-д.
//
// ДҮН нь захирамжийн жагсаалттай ИЖИЛ: нэгж талбарын сонгосон үнэлгээний
// бэлэн мөнгөн олговор (compensation_type = 'cash').
func (r *landAcqRepo) GetDecisionFunding(
	ctx context.Context, f domain.DecisionFundingFilter,
) (*domain.DecisionFundingSummary, error) {
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	conds := []string{"la.deleted_at IS NULL"}
	if f.DepartmentID != nil {
		conds = append(conds, fmt.Sprintf(`la.general_category_id IN (
			SELECT id FROM acquisition_category WHERE department_id = %s)`, arg(*f.DepartmentID)))
	}
	if f.AcquisitionID != "" {
		conds = append(conds, "la.id = "+arg(f.AcquisitionID)+"::uuid")
	}
	if f.PlanCode != "" {
		conds = append(conds, "la.plan_code ILIKE "+arg("%"+f.PlanCode+"%"))
	}
	if f.GeneralCategoryID != 0 {
		conds = append(conds, "la.general_category_id = "+arg(f.GeneralCategoryID))
	}
	if f.SubCategoryID != 0 {
		conds = append(conds, "la.sub_category_id = "+arg(f.SubCategoryID))
	}
	if f.AU2Code != "" {
		// Хяналтын самбартай ИЖИЛ: чөлөөлөлтийн хил ЭСВЭЛ нэгж талбарын au2.
		ph := arg(f.AU2Code)
		conds = append(conds, fmt.Sprintf(`(
			EXISTS (SELECT 1 FROM land_acquisition_au
			         WHERE land_acquisition_id = la.id AND au2_code = %[1]s)
			OR EXISTS (SELECT 1 FROM parcel p_au
			            WHERE p_au.acquisition_id = la.id AND p_au.deleted_at IS NULL
			              AND p_au.au2_code = %[1]s))`, ph))
	}
	if f.AssignedUserID != "" {
		conds = append(conds, fmt.Sprintf(`la.id IN (
			SELECT acquisition_id FROM land_acquisition_assignee WHERE user_id = %s::int4)`, arg(f.AssignedUserID)))
	}
	// Шүүлтгүй (админ, бүх чөлөөлөлт) бол санхүүжилтийг чөлөөлөлтөөр
	// хязгаарлахгүй — нэгж талбаргүй, чөлөөлөлтгүй төслийн санхүүжилт ч орно.
	scoped := len(conds) > 1
	acqCTE := `acq AS MATERIALIZED (SELECT la.id FROM land_acquisition la WHERE ` +
		strings.Join(conds, " AND ") + `)`
	acqArgs := append([]any(nil), args...)

	// Хүрээнд хамаарах захирамжийн төслүүд (чөлөөлөлтөөр эсвэл идэвхтэй нэгж
	// талбарын холбоосоор) — хяналтын самбарын захирамжийн тоололттой ИЖИЛ.
	draftScope := "TRUE"
	if scoped {
		draftScope = `ddfs.decision_draft_id IN (
			SELECT d.id FROM decision_draft d WHERE d.acquisition_id IN (SELECT id FROM acq)
			UNION
			SELECT ddp.decision_draft_id FROM decision_draft_parcel ddp
			  JOIN parcel p ON p.id = ddp.parcel_id
			 WHERE ddp.removed_at IS NULL AND p.acquisition_id IN (SELECT id FROM acq))`
	}

	out := &domain.DecisionFundingSummary{Years: []int{}, BySource: []domain.DecisionFundingSourceYear{}}

	// 1. Сонголтын онууд — оны шүүлтээс үл хамааран.
	rows, err := r.db.Query(ctx, `WITH `+acqCTE+`
		SELECT DISTINCT ddfs.year FROM decision_draft_funding_source ddfs
		 WHERE ddfs.year IS NOT NULL AND `+draftScope+`
		 ORDER BY 1 DESC`, acqArgs...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			rows.Close()
			return nil, err
		}
		out.Years = append(out.Years, y)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	yearArgs := append([]any(nil), acqArgs...)
	fundYearCond, parcelYearCond := "", ""
	if len(f.Years) > 0 {
		yearArgs = append(yearArgs, f.Years)
		ph := fmt.Sprintf("$%d::int[]", len(yearArgs))
		fundYearCond = " AND ddfs.year = ANY(" + ph + ")"
		parcelYearCond = " AND fyear = ANY(" + ph + ")"
	}

	// 2. Төлөвлөсөн санхүүжилт.
	if err := r.db.QueryRow(ctx, `WITH `+acqCTE+`
		SELECT COALESCE(SUM(ddfs.amount), 0)::float8
		  FROM decision_draft_funding_source ddfs
		 WHERE `+draftScope+fundYearCond, yearArgs...).Scan(&out.Planned); err != nil {
		return nil, err
	}

	// 2б. Эх үүсвэр × оноор (дүн бүрийн он).
	srcRows, err := r.db.Query(ctx, `WITH `+acqCTE+`
		SELECT ddfs.funding_source_id::text, ddfs.year, COALESCE(SUM(ddfs.amount), 0)::float8
		  FROM decision_draft_funding_source ddfs
		 WHERE `+draftScope+fundYearCond+`
		 GROUP BY 1, 2
		 ORDER BY 2 DESC NULLS LAST, 1`, yearArgs...)
	if err != nil {
		return nil, err
	}
	for srcRows.Next() {
		var b domain.DecisionFundingSourceYear
		if err := srcRows.Scan(&b.FundingSourceID, &b.Year, &b.Amount); err != nil {
			srcRows.Close()
			return nil, err
		}
		out.BySource = append(out.BySource, b)
	}
	srcRows.Close()
	if err := srcRows.Err(); err != nil {
		return nil, err
	}

	// 3. Захирамжид холбогдсон нэгж талбарын олговор — шатаар.
	//
	// lp — идэвхтэй холбоос бүрийн НЭГ мөр (uq_ddp_active_parcel), олговрыг
	// нэгж талбар тус бүрд нэг удаа (correlated) нэгтгэнэ — олговор×холбоос
	// JOIN хийхгүй тул мөр үржихгүй.
	err = r.db.QueryRow(ctx, fmt.Sprintf(`WITH %s,
		lp AS MATERIALIZED (
			SELECT p.acquisition_id, p.parcel_id,
			       COALESCE(p.selected_valuation_type, 'asset') AS vtype,
			       COALESCE(ddfs.year, d.duration_year) AS fyear,
			       COALESCE(d.status = %d OR ph.progress_type = '%s', false) AS issued,
			       %s AS paid_ratio
			  FROM decision_draft_parcel ddp
			  JOIN parcel p ON p.id = ddp.parcel_id AND p.deleted_at IS NULL
			  JOIN decision_draft d ON d.id = ddp.decision_draft_id
			  LEFT JOIN decision_draft_progress_history ph ON ph.id = d.current_progress_history_id
			  LEFT JOIN decision_draft_funding_source ddfs ON ddfs.id = ddp.funding_link_id
			 WHERE ddp.removed_at IS NULL
			   AND p.acquisition_id IN (SELECT id FROM acq)
		),
		amt AS (
			SELECT lp.issued, lp.paid_ratio,
			       COALESCE((SELECT SUM(c.amount) FROM compensation c
			                  WHERE c.acquisition_id = lp.acquisition_id
			                    AND c.parcel_id = lp.parcel_id
			                    AND c.compensation_type = 'cash'
			                    AND c.valuation_type = lp.vtype), 0) AS amount
			  FROM lp WHERE TRUE%s
		)
		SELECT COALESCE(SUM(amount) FILTER (WHERE issued), 0)::float8,
		       COALESCE(SUM(amount * paid_ratio) FILTER (WHERE issued), 0)::float8,
		       COUNT(*) FILTER (WHERE issued),
		       COALESCE(SUM(amount) FILTER (WHERE NOT issued), 0)::float8,
		       COALESCE(SUM(amount * paid_ratio) FILTER (WHERE NOT issued), 0)::float8,
		       COUNT(*) FILTER (WHERE NOT issued)
		  FROM amt`,
		acqCTE, domain.DecisionDraftStatusConfirmed, domain.DecisionDraftProgressConfirming,
		decisionFundingPaidRatio, parcelYearCond),
		yearArgs...,
	).Scan(
		&out.Issued.Amount, &out.Issued.Granted, &out.Issued.Parcels,
		&out.Draft.Amount, &out.Draft.Granted, &out.Draft.Parcels,
	)
	if err != nil {
		return nil, err
	}
	out.Issued.Pending = out.Issued.Amount - out.Issued.Granted
	out.Draft.Pending = out.Draft.Amount - out.Draft.Granted
	out.PlannedGranted = out.Issued.Granted + out.Draft.Granted
	return out, nil
}
