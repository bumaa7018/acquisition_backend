package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"acquisition_backend/internal/domain"
)

// GetFinanceDashboard — САНХҮҮГИЙН хяналтын самбарын нэгтгэл.
//
// Эх сурвалж: `compensation` (нөхөх олговор) + `compensation_grant` (олгосон
// бичилт), нэгж талбараар дүүрэг (au2) / хороо (au3)-той холбогдоно.
//
// ОН: олговрын ӨӨРИЙН огноо (`compensation_date` → `reviewed_at` →
// `created_at`) — гүйлгээ аль онд хамаарахыг тэр заана. Нэгж талбарын статусын
// он (хяналтын самбарын бусад хэсэг) нь ӨӨР ойлголт тул эндээс зааглав.
//
// ГҮЙЦЭТГЭЛ (системд ачаалал өгөхгүй байх):
//   - Суурь хүснэгтүүдийг НЭГ л удаа уншина: `MATERIALIZED` CTE-д хадгалаад
//     дөрвөн зүсэлтийг (нийт / он / дүүрэг / хороо) `UNION ALL`-аар түүн дээр
//     hash-аар бодно. Бодит хэмжилт: 796мс (GROUPING SETS) → 121мс.
//   - Нэрийн лавлах (au2/au3) нь ЖИЖИГ тул нэгтгэсний ДАРАА, зөвхөн гарч ирсэн
//     кодуудаар нэг удаа татагдана — их хүснэгттэй JOIN хийхгүй.
//   - Шүүлтүүрийн он нь `compensation`-ийн огнооноос шууд (нэгж талбартай
//     холбохгүй) — хөнгөн DISTINCT.
//   - `land_acquisition`-тай JOIN ХИЙХГҮЙ: чөлөөлөлт soft delete хийгдэхэд
//     түүний нэгж талбарууд мөн `deleted_at`-тай болдог (000003 миграц) тул
//     `p.deleted_at IS NULL` хангалттай.
func (r *landAcqRepo) GetFinanceDashboard(
	ctx context.Context, filter domain.FinanceDashboardFilter,
) (*domain.FinanceDashboard, error) {
	args := []any{}
	where := []string{"p.deleted_at IS NULL"}
	// Дугаарлалтыг тусад нь тоолуураар барихгүй: `len(args)+1` нь ДАРААГИЙН
	// байрлагчийн дугаар тул шүүлтүүр нэмэхэд дугаар салах эрсдэлгүй.
	next := func() int { return len(args) + 1 }

	if filter.DepartmentID != nil {
		where = append(where, fmt.Sprintf(`c.acquisition_id IN (
			SELECT la.id FROM land_acquisition la
			JOIN acquisition_category ac ON ac.id = la.general_category_id
			WHERE ac.department_id = $%d)`, next()))
		args = append(args, *filter.DepartmentID)
	}

	if len(filter.Years) > 0 {
		ph := make([]string, len(filter.Years))
		for j, y := range filter.Years {
			ph[j] = fmt.Sprintf("$%d", next())
			args = append(args, y)
		}
		where = append(where, fmt.Sprintf(
			`EXTRACT(YEAR FROM COALESCE(c.compensation_date, c.reviewed_at, c.created_at))::int IN (%s)`,
			strings.Join(ph, ",")))
	}
	if filter.AU2Code != "" {
		where = append(where, fmt.Sprintf("COALESCE(p.au2_code,'') = $%d", next()))
		args = append(args, filter.AU2Code)
	}
	if filter.AU3Code != "" {
		where = append(where, fmt.Sprintf("COALESCE(p.au3_code,'') = $%d", next()))
		args = append(args, filter.AU3Code)
	}
	if filter.AcquisitionID != "" {
		where = append(where, fmt.Sprintf("c.acquisition_id = $%d::uuid", next()))
		args = append(args, filter.AcquisitionID)
	}

	out := &domain.FinanceDashboard{
		ByYear:     []domain.FinanceBucket{},
		ByDistrict: []domain.FinanceBucket{},
		ByKhoroo:   []domain.FinanceBucket{},
		Years:      []int{},
	}

	// НЭГ асуулга — бүх зүсэлт (нийт / он / дүүрэг / хороо).
	//
	// `MATERIALIZED` CTE + `UNION ALL`: суурь хүснэгтүүд НЭГ л удаа уншигдаж,
	// дөрвөн нэгтгэл нь түр хадгалагдсан мөрүүд дээр hash-аар бодогдоно.
	// (`GROUPING SETS` нь нэг асуулга ч гэсэн эрэмбэлэлт шаардаж 6 дахин удаан
	// байсан: 796мс → 121мс бодит хэмжилтээр.)
	rows, err := r.db.Query(ctx, `
		WITH f AS MATERIALIZED (
			SELECT p.id AS parcel_uuid,
			       EXTRACT(YEAR FROM COALESCE(c.compensation_date, c.reviewed_at, c.created_at))::int AS comp_year,
			       COALESCE(NULLIF(p.au2_code,''), '—') AS au2_code,
			       COALESCE(NULLIF(p.au3_code,''), '—') AS au3_code,
			       c.acquisition_id,
			       CASE WHEN c.status = 'approved' THEN COALESCE(c.amount,0) ELSE 0 END AS approved_amount,
			       CASE WHEN c.status <> 'approved' THEN COALESCE(c.amount,0) ELSE 0 END AS pending_amount,
			       COALESCE(cg.amount, 0) AS granted_amount,
			       CASE
			         WHEN c.asset_id IS NULL THEN 'land'
			         WHEN a.asset_type = 'real_state' THEN 'real_state'
			         ELSE 'property'
			       END AS category
			FROM compensation c
			JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
			LEFT JOIN compensation_grant cg ON cg.compensation_id = c.id
			LEFT JOIN assets a ON a.id = c.asset_id
			WHERE `+strings.Join(where, " AND ")+`
		)
		SELECT 'total' AS bucket, '' AS k,
		       COALESCE(SUM(approved_amount),0), COALESCE(SUM(granted_amount),0),
		       COALESCE(SUM(pending_amount),0), COUNT(DISTINCT parcel_uuid)
		FROM f
		UNION ALL
		SELECT 'year', COALESCE(comp_year::text,''),
		       COALESCE(SUM(approved_amount),0), COALESCE(SUM(granted_amount),0),
		       COALESCE(SUM(pending_amount),0), COUNT(DISTINCT parcel_uuid)
		FROM f GROUP BY comp_year
		UNION ALL
		SELECT 'district', au2_code,
		       COALESCE(SUM(approved_amount),0), COALESCE(SUM(granted_amount),0),
		       COALESCE(SUM(pending_amount),0), COUNT(DISTINCT parcel_uuid)
		FROM f GROUP BY au2_code
		UNION ALL
		SELECT 'khoroo', au3_code,
		       COALESCE(SUM(approved_amount),0), COALESCE(SUM(granted_amount),0),
		       COALESCE(SUM(pending_amount),0), COUNT(DISTINCT parcel_uuid)
		FROM f GROUP BY au3_code
		UNION ALL
		SELECT 'category', category,
		       COALESCE(SUM(approved_amount),0), COALESCE(SUM(granted_amount),0),
		       COALESCE(SUM(pending_amount),0), COUNT(DISTINCT parcel_uuid)
		FROM f GROUP BY category`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bucket, key string
		b := domain.FinanceBucket{}
		if err := rows.Scan(&bucket, &key,
			&b.Amount, &b.Granted, &b.Pending, &b.Parcels); err != nil {
			return nil, err
		}
		b.Key, b.Name = key, key
		switch bucket {
		case "total":
			out.TotalAmount, out.GrantedAmount = b.Amount, b.Granted
			out.PendingAmount, out.TotalParcels = b.Pending, b.Parcels
		case "year":
			if key == "" {
				continue // огноогүй олговор — оны график дээр харуулахгүй
			}
			out.ByYear = append(out.ByYear, b)
		case "district":
			out.ByDistrict = append(out.ByDistrict, b)
		case "khoroo":
			out.ByKhoroo = append(out.ByKhoroo, b)
		case "category":
			switch key {
			case "land":
				out.LandAmount = b.Amount
			case "real_state":
				out.RealStateAmount = b.Amount
			default:
				out.PropertyAmount = b.Amount
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.financeStages(ctx, out, where, args); err != nil {
		return nil, err
	}

	sortBuckets(out.ByYear, true)
	sortBuckets(out.ByDistrict, false)
	sortBuckets(out.ByKhoroo, false)
	if err := r.fillAUNames(ctx, out); err != nil {
		return nil, err
	}

	if err := r.financeFilterOptions(ctx, out, filter); err != nil {
		return nil, err
	}

	// Шүүлтүүрийн он — ШҮҮЛТГҮЙ (сонголт нарийсахгүй), нэгж талбартай холбохгүй.
	yearRows, err := r.db.Query(ctx, `
		SELECT DISTINCT EXTRACT(YEAR FROM COALESCE(compensation_date, reviewed_at, created_at))::int AS y
		FROM compensation
		WHERE COALESCE(compensation_date, reviewed_at, created_at) IS NOT NULL
		ORDER BY y DESC`)
	if err != nil {
		return nil, err
	}
	defer yearRows.Close()
	for yearRows.Next() {
		var y int
		if err := yearRows.Scan(&y); err != nil {
			return nil, err
		}
		out.Years = append(out.Years, y)
	}
	return out, yearRows.Err()
}

// sortBuckets — оноор ӨСӨХ (график зүүнээс баруун тийш), бусад нь ДҮНГЭЭР буурах.
func sortBuckets(list []domain.FinanceBucket, byKeyAsc bool) {
	for a := 1; a < len(list); a++ {
		for b := a; b > 0; b-- {
			less := list[b].Amount > list[b-1].Amount
			if byKeyAsc {
				less = list[b].Key < list[b-1].Key
			}
			if !less {
				break
			}
			list[b], list[b-1] = list[b-1], list[b]
		}
	}
}

// fillAUNames — дүүрэг/хорооны НЭРийг зөвхөн гарч ирсэн кодуудаар татна
// (лавлах хоосон эсвэл код олдохгүй бол код нь нэр болж үлдэнэ).
func (r *landAcqRepo) fillAUNames(ctx context.Context, out *domain.FinanceDashboard) error {
	load := func(table string, list []domain.FinanceBucket) error {
		if len(list) == 0 {
			return nil
		}
		codes := make([]string, 0, len(list))
		for _, b := range list {
			codes = append(codes, b.Key)
		}
		rows, err := r.db.Query(ctx,
			fmt.Sprintf(`SELECT code, name FROM %s WHERE code = ANY($1)`, table), codes)
		if err != nil {
			return err
		}
		defer rows.Close()
		names := map[string]string{}
		for rows.Next() {
			var code, name string
			if err := rows.Scan(&code, &name); err != nil {
				return err
			}
			names[code] = name
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for idx := range list {
			if name, ok := names[list[idx].Key]; ok && name != "" {
				list[idx].Name = name
			}
		}
		return nil
	}
	if err := load("au2", out.ByDistrict); err != nil {
		return err
	}
	return load("au3", out.ByKhoroo)
}

// financeStages — нэгж талбарын ГҮЙЦЭТГЭЛИЙН шатууд ба санхүүжилтийн эх
// үүсвэрийн задаргаа.
//
// Гүйцэтгэл: нэгж талбарын түвшний нэгтгэл НЭГ асуулгаар — олговрын дүнг
// нэгж талбараар урьдчилан хураангуйлаад (pamt), захирамжийн холбоос болон
// үнэлгээний төлөвийг ЖИЖИГ дэд асуулгаар нэгтгэнэ.
func (r *landAcqRepo) financeStages(
	ctx context.Context, out *domain.FinanceDashboard, where []string, args []any,
) error {
	// ГҮЙЦЭТГЭЛ: холбоосуудыг ПРЕ-НЭГТГЭЖ (decree / subs) LEFT JOIN-оор авна.
	// Нэгж талбар бүрд `EXISTS` дэд асуулга ажиллуулах хувилбар 3.8 СЕКУНД
	// авдаг байсныг 63мс болгов (бодит хэмжилт).
	rows, err := r.db.Query(ctx, `
		WITH f AS MATERIALIZED (
			SELECT p.id AS parcel_uuid, p.parcel_id,
			       CASE WHEN c.status = 'approved' THEN COALESCE(c.amount,0) ELSE 0 END AS approved_amount,
			       CASE WHEN c.status <> 'approved' THEN COALESCE(c.amount,0) ELSE 0 END AS pending_amount,
			       COALESCE(cg.amount, 0) AS granted_amount
			FROM compensation c
			JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
			LEFT JOIN compensation_grant cg ON cg.compensation_id = c.id
			WHERE `+strings.Join(where, " AND ")+`
		), pamt AS (
			SELECT parcel_uuid, MIN(parcel_id) AS code,
			       SUM(approved_amount) AS approved, SUM(granted_amount) AS granted,
			       SUM(pending_amount) AS pending
			FROM f GROUP BY parcel_uuid
		), decree AS (
			-- ЗАХИРАМЖ холбогдсон (баталгаажсан, дугаартай) нэгж талбарууд.
			SELECT DISTINCT ddp.parcel_id AS parcel_uuid
			FROM decision_draft_parcel ddp
			JOIN decision_draft dd ON dd.id = ddp.decision_draft_id
			WHERE ddp.removed_at IS NULL AND dd.status = 2
			  AND COALESCE(dd.decree_number,'') <> ''
		), subs AS (
			SELECT parcel_id AS code,
			       bool_or(status = 'approved') AS approved_sub,
			       bool_or(status = 'submitted') AS submitted_sub
			FROM parcel_valuation_submission GROUP BY parcel_id
		)
		SELECT stage, COUNT(*),
		       COALESCE(SUM(CASE WHEN stage = 'granted' THEN granted ELSE approved + pending END), 0)
		FROM (
			SELECT pa.approved, pa.granted, pa.pending,
			       CASE
			         WHEN d.parcel_uuid IS NOT NULL THEN 'granted'
			         WHEN pa.approved > 0 OR s.approved_sub THEN 'approved'
			         WHEN s.submitted_sub THEN 'submitted'
			         ELSE 'pending'
			       END AS stage
			FROM pamt pa
			LEFT JOIN decree d ON d.parcel_uuid = pa.parcel_uuid
			LEFT JOIN subs s ON s.code = pa.code
		) x
		GROUP BY stage`, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var stage string
		st := domain.FinanceStage{}
		if err := rows.Scan(&stage, &st.Count, &st.Amount); err != nil {
			rows.Close()
			return err
		}
		switch stage {
		case "granted":
			out.StageGranted = st
		case "approved":
			out.StageApproved = st
		case "submitted":
			out.StageSubmitted = st
		default:
			out.StagePending = st
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// ЧӨЛӨӨЛӨЛТИЙН нэгтгэл — НЭГ асуулгаас гурван зүсэлт:
	//   acq    — ЯВЖ БУЙ (баталгаажаагүй) чөлөөлөлт бүрийн дүн (чартад),
	//   status — төлөвөөр (карт),
	//   budget — захирамжийн ТӨСВӨӨР (карт).
	acqArgs := append(append([]any{}, args...),
		domain.LandAcquisitionStatusNew, domain.LandAcquisitionStatusField)
	acqRows, err := r.db.Query(ctx, fmt.Sprintf(`
		WITH f AS MATERIALIZED (
			SELECT p.id AS parcel_uuid, c.acquisition_id,
			       CASE WHEN c.status = 'approved' THEN COALESCE(c.amount,0) ELSE 0 END AS approved_amount,
			       CASE WHEN c.status <> 'approved' THEN COALESCE(c.amount,0) ELSE 0 END AS pending_amount,
			       COALESCE(cg.amount, 0) AS granted_amount
			FROM compensation c
			JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
			LEFT JOIN compensation_grant cg ON cg.compensation_id = c.id
			WHERE `+strings.Join(where, " AND ")+`
		), agg AS (
			SELECT f.acquisition_id,
			       COUNT(DISTINCT f.parcel_uuid) AS parcels,
			       SUM(f.approved_amount) AS amount,
			       SUM(f.granted_amount) AS granted,
			       SUM(f.pending_amount) AS pending
			FROM f GROUP BY f.acquisition_id
		), budget_per_acq AS (
			-- Чөлөөлөлт бүрийн ТӨСӨВ (захирамжийн төслөөс). Эхлээд DISTINCT
			-- болгож жижигрүүлнэ: шууд JOIN хийвэл захирамжийн тоогоор мөр
			-- үржиж, асуулга 312мс → 46мс-ийн зөрүү үүсгэдэг.
			SELECT DISTINCT dd.acquisition_id,
			       COALESCE(NULLIF(TRIM(b.name),''), 'Тодорхойгүй') AS budget
			FROM decision_draft dd
			LEFT JOIN decision_budget b ON b.id = dd.budget_id
			WHERE dd.acquisition_id IS NOT NULL
		), joined AS (
			SELECT a.*, la.acquisition_name, la.plan_code, la.status
			FROM agg a
			JOIN land_acquisition la ON la.id = a.acquisition_id AND la.deleted_at IS NULL
		)
		SELECT 'acq' AS kind, acquisition_id::text,
		       COALESCE(NULLIF(acquisition_name,''), plan_code, acquisition_id::text),
		       amount, granted, pending, parcels, 1
		FROM joined
		WHERE status IN ($%[1]d, $%[2]d)   -- Шинэ, Хээрийн судалгаа (баталгаажаагүй)
		UNION ALL
		SELECT 'status', status::text, '',
		       COALESCE(SUM(amount),0), COALESCE(SUM(granted),0), COALESCE(SUM(pending),0),
		       COALESCE(SUM(parcels),0), COUNT(*)
		FROM joined GROUP BY status
		UNION ALL
		SELECT 'budget', bp.budget, '',
		       COALESCE(SUM(j.amount),0), COALESCE(SUM(j.granted),0), COALESCE(SUM(j.pending),0),
		       COALESCE(SUM(j.parcels),0), COUNT(DISTINCT j.acquisition_id)
		FROM joined j
		JOIN budget_per_acq bp ON bp.acquisition_id = j.acquisition_id
		GROUP BY 2
		ORDER BY 4 DESC`, len(args)+1, len(args)+2), acqArgs...)
	if err != nil {
		return err
	}
	out.TopAcquisitions = []domain.FinanceBucket{}
	out.AcquisitionStatus = []domain.FinanceBucket{}
	out.AcquisitionBudgets = []domain.FinanceBucket{}
	for acqRows.Next() {
		var kind, key, name string
		b := domain.FinanceBucket{}
		if err := acqRows.Scan(&kind, &key, &name,
			&b.Amount, &b.Granted, &b.Pending, &b.Parcels, &b.Count); err != nil {
			acqRows.Close()
			return err
		}
		b.Key = key
		b.Name = name
		if b.Name == "" {
			b.Name = key
		}
		switch kind {
		case "acq":
			out.TopAcquisitions = append(out.TopAcquisitions, b)
		case "status":
			if code, convErr := strconv.Atoi(key); convErr == nil {
				if label, ok := domain.AcquisitionStatusLabels[code]; ok {
					b.Name = label
				}
			}
			out.AcquisitionStatus = append(out.AcquisitionStatus, b)
			out.TotalAcquisitions += b.Count
		case "budget":
			out.AcquisitionBudgets = append(out.AcquisitionBudgets, b)
		}
	}
	acqRows.Close()
	if err := acqRows.Err(); err != nil {
		return err
	}

	// САНХҮҮЖИЛТИЙН эх үүсвэрийн ТӨРЛӨӨР — хамрах хүрээнд орсон чөлөөлөлтүүдээр.
	// (Чөлөөлөлтийн ЖАГСААЛТ энд буцаагдахгүй: тэр нь "Газар чөлөөлөлт" цэсэнд
	// бүрэн хэлбэрээрээ байгаа — самбар зөвхөн нэгтгэсэн дүн харуулна.)
	typeRows, err := r.db.Query(ctx, `
		WITH f AS MATERIALIZED (
			SELECT DISTINCT c.acquisition_id
			FROM compensation c
			JOIN parcel p ON p.acquisition_id = c.acquisition_id AND p.parcel_id = c.parcel_id
			WHERE `+strings.Join(where, " AND ")+`
		)
		SELECT COALESCE(NULLIF(TRIM(fs.source_type),''), 'Тодорхойгүй') AS t,
		       COALESCE(SUM(fs.amount),0)
		FROM funding_sources fs
		JOIN f ON f.acquisition_id = fs.acquisition_id
		GROUP BY t
		ORDER BY 2 DESC`, args...)
	if err != nil {
		return err
	}
	defer typeRows.Close()
	out.FundingByType = []domain.FinanceBucket{}
	for typeRows.Next() {
		b := domain.FinanceBucket{}
		if err := typeRows.Scan(&b.Key, &b.Amount); err != nil {
			return err
		}
		b.Name = b.Key
		out.FundingTotal += b.Amount
		out.FundingByType = append(out.FundingByType, b)
	}
	return typeRows.Err()
}

// financeFilterOptions — дүүрэг (бүрэн жагсаалт) ба хороо (сонгосон дүүргээр)
// сонголтууд. Он/чөлөөлөлтийн шүүлтээс ХАМААРАХГҮЙ: эс бөгөөс сонгосон
// шүүлтээ буцааж өөрчлөх боломжгүй болно.
//
// Хөнгөн: зөвхөн `parcel`-ийн индекслэгдсэн (au1, au2, au3) багануудаар
// DISTINCT, нэрийг жижиг лавлахаас LEFT JOIN-оор.
func (r *landAcqRepo) financeFilterOptions(
	ctx context.Context, out *domain.FinanceDashboard, filter domain.FinanceDashboardFilter,
) error {
	districts, err := r.db.Query(ctx, `
		SELECT DISTINCT COALESCE(NULLIF(p.au2_code,''), '—') AS code,
		       COALESCE(NULLIF(a.name,''), COALESCE(NULLIF(p.au2_code,''), '—')) AS name
		FROM parcel p
		LEFT JOIN au2 a ON a.code = p.au2_code
		WHERE p.deleted_at IS NULL
		ORDER BY name`)
	if err != nil {
		return err
	}
	out.DistrictOptions = []domain.FinanceOption{}
	for districts.Next() {
		o := domain.FinanceOption{}
		if err := districts.Scan(&o.Code, &o.Name); err != nil {
			districts.Close()
			return err
		}
		out.DistrictOptions = append(out.DistrictOptions, o)
	}
	districts.Close()
	if err := districts.Err(); err != nil {
		return err
	}

	khorooArgs := []any{}
	khorooWhere := "p.deleted_at IS NULL"
	if filter.AU2Code != "" {
		khorooWhere += " AND COALESCE(p.au2_code,'') = $1"
		khorooArgs = append(khorooArgs, filter.AU2Code)
	}
	khoroos, err := r.db.Query(ctx, `
		SELECT DISTINCT COALESCE(NULLIF(p.au3_code,''), '—') AS code,
		       COALESCE(NULLIF(a.name,''), COALESCE(NULLIF(p.au3_code,''), '—')) AS name
		FROM parcel p
		LEFT JOIN au3 a ON a.code = p.au3_code
		WHERE `+khorooWhere+`
		ORDER BY name`, khorooArgs...)
	if err != nil {
		return err
	}
	defer khoroos.Close()
	out.KhorooOptions = []domain.FinanceOption{}
	for khoroos.Next() {
		o := domain.FinanceOption{}
		if err := khoroos.Scan(&o.Code, &o.Name); err != nil {
			return err
		}
		out.KhorooOptions = append(out.KhorooOptions, o)
	}
	return khoroos.Err()
}
