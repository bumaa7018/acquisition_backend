package postgres

import (
	"context"
	"encoding/json"
	"time"
	_ "time/tzdata" // серверт tzdata байхгүй ч Улаанбаатарын бүс ачаалагдана

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

// GetChronos — «Хронос»-ын өгөгдөл: төлөвийн бүртгэл, нэгж талбарын геометр
// (3857), чөлөөлсөн огноо, төлөвийн түүх, сар бүрийн явцын өөрчлөлт. acqIDs хоосон
// бол бүх чөлөөлөлт. Дээд тал нь domain.ChronosMaxParcels.
func (r *landAcqRepo) GetChronos(ctx context.Context, acqIDs []uuid.UUID) (*domain.ChronosData, error) {
	rows, err := r.db.Query(ctx, `
		WITH pay AS (
		        -- Олголтын баримт — НЭГ удаа нэгтгэнэ (нэгж талбар бүрт document-ыг
		        -- дахин уншвал «нэгж талбар × баримт» болж квадратаар өснө).
		        SELECT pd.parcel_id,
		               json_agg(json_build_object('date', to_char(d.payment_date, 'YYYY-MM-DD'), 'stage', d.payment_stage)
		                        ORDER BY d.payment_date)::text AS items
		          FROM document d
		          JOIN parcel_document pd ON pd.document_id = d.id
		         WHERE d.payment_stage > 0 AND d.payment_date IS NOT NULL
		         GROUP BY pd.parcel_id
		)
		SELECT p.id, p.parcel_id, p.acquisition_id, COALESCE(la.acquisition_name, ''),
		       COALESCE(p.status, 0), COALESCE(ps.name, ''), COALESCE(ps.color, ''),
		       (SELECT MIN(h.status_date) FROM parcel_status_history h
		          JOIN parcel_status rs ON rs.id = h.status_id AND rs.is_released
		         WHERE h.parcel_id = p.parcel_id AND h.acquisition_id = p.acquisition_id),
		       COALESCE(p.area_m2, 0),
		       -- Геометргүй нэгж талбар ч статистикт орно (зурагт гарахгүй) — дашбоардтай ижил тоо.
		       COALESCE(ST_AsGeoJSON(ST_SimplifyPreserveTopology(ST_Transform(p.geometry, 3857), 0.5), 1), 'null'),
		       la.created_at,
		       COALESCE((
		         SELECT TRIM(CONCAT_WS(' ', NULLIF(rh.last_name, ''), rh.name))
		           FROM parcel_holder ph
		           JOIN right_holder rh ON rh.id = ph.holder_id
		          WHERE ph.parcel_id = p.id AND ph.holder_role = 'holder'
		          ORDER BY ph.main_applicant DESC, ph.created_at
		          LIMIT 1), ''),
		       COALESCE(NULLIF(p.landuse_name, ''), p.landuse, ''), COALESCE(p.right_type, 0),
		       COALESCE(comp.land, 0), COALESCE(comp.real_state, 0), COALESCE(comp.property, 0),
		       COALESCE(comp.amount, 0), comp.first_at, comp.approved_at,
		       dd.id, ddp.linked_at, CASE WHEN dd.id IS NOT NULL THEN dd.confirmed_at END,
		       COALESCE(pay.items, '[]')
		  FROM parcel p
		  JOIN land_acquisition la ON la.id = p.acquisition_id AND la.deleted_at IS NULL
		  LEFT JOIN parcel_status ps ON ps.id = p.status
		  LEFT JOIN LATERAL (
		        SELECT SUM(c.amount) AS amount, MIN(c.created_at) AS first_at,
		               CASE WHEN bool_and(c.status = 'approved') THEN MAX(c.reviewed_at) END AS approved_at,
		               SUM(c.amount) FILTER (WHERE c.asset_id IS NULL) AS land,
		               SUM(c.amount) FILTER (WHERE c.asset_id IS NOT NULL AND a.asset_type = 'real_state') AS real_state,
		               SUM(c.amount) FILTER (WHERE c.asset_id IS NOT NULL AND COALESCE(a.asset_type, '') <> 'real_state') AS property
		          FROM compensation c
		          LEFT JOIN assets a ON a.id = c.asset_id
		         WHERE c.acquisition_id = p.acquisition_id AND c.parcel_id = p.parcel_id
		           AND c.valuation_type = COALESCE(p.selected_valuation_type, 'asset')
		  ) comp ON true
		  -- Идэвхтэй холбоос нэгж талбарт ХАМГИЙН ИХДЭЭ нэг (uq_ddp_active_parcel) —
		  -- LATERAL/ORDER BY шаардлагагүй, хүснэгт бүр нэг л удаа hash join-оор.
		  LEFT JOIN decision_draft_parcel ddp ON ddp.parcel_id = p.id AND ddp.removed_at IS NULL
		  LEFT JOIN decision_draft dd ON dd.id = ddp.decision_draft_id
		  LEFT JOIN pay ON pay.parcel_id = p.id
		 WHERE p.deleted_at IS NULL
		   AND (COALESCE(cardinality($1::uuid[]), 0) = 0 OR p.acquisition_id = ANY($1)) -- nil slice → NULL
		 ORDER BY p.parcel_id
		 LIMIT $2`, acqIDs, domain.ChronosMaxParcels+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := &domain.ChronosData{Statuses: []domain.ChronosStatus{}, Decisions: []domain.ChronosDecision{}, Parcels: []*domain.ChronosParcel{}, Months: []domain.ChronosMonth{}}
	type key struct {
		acq  uuid.UUID
		code string
	}
	byKey := map[key]*domain.ChronosParcel{}
	created := map[uuid.UUID]*time.Time{}
	months := map[string]int{}
	for rows.Next() {
		p := &domain.ChronosParcel{Events: []domain.ChronosEvent{}, Payments: []domain.ChronosPayment{}}
		var geom, payments string
		var acqCreated *time.Time
		if err := rows.Scan(&p.ID, &p.ParcelID, &p.AcquisitionID, &p.AcquisitionName,
			&p.StatusID, &p.StatusName, &p.StatusColor, &p.ReleasedAt, &p.AreaM2, &geom,
			&acqCreated, &p.HolderName, &p.LanduseName, &p.RightType,
			&p.CompLand, &p.CompRealState, &p.CompProperty,
			&p.CompAmount, &p.CompAt, &p.CompApprovedAt,
			&p.DecisionID, &p.DecisionLinkedAt, &p.DecisionConfirmedAt, &payments); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(payments), &p.Payments)
		for _, t := range []*time.Time{p.ReleasedAt, p.CompAt, p.CompApprovedAt, p.DecisionLinkedAt, p.DecisionConfirmedAt, acqCreated} {
			toUB(t)
		}
		created[p.ID] = acqCreated
		if len(out.Parcels) == domain.ChronosMaxParcels {
			out.Truncated = true
			break
		}
		p.Geometry = json.RawMessage(geom)
		out.Parcels = append(out.Parcels, p)
		byKey[key{p.AcquisitionID, p.ParcelID}] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if len(out.Parcels) > 0 {
		ids := make([]uuid.UUID, 0, len(out.Parcels))
		seen := map[uuid.UUID]bool{}
		for _, p := range out.Parcels {
			if !seen[p.AcquisitionID] {
				seen[p.AcquisitionID] = true
				ids = append(ids, p.AcquisitionID)
			}
		}
		ev, err := r.db.Query(ctx, `
			SELECT h.acquisition_id, h.parcel_id, h.status_date, h.status_id, COALESCE(ps.name, '')
			  FROM parcel_status_history h
			  LEFT JOIN parcel_status ps ON ps.id = h.status_id
			 WHERE h.acquisition_id = ANY($1)
			 ORDER BY h.status_date, h.id`, ids)
		if err != nil {
			return nil, err
		}
		defer ev.Close()
		for ev.Next() {
			var k key
			var e domain.ChronosEvent
			var at time.Time
			if err := ev.Scan(&k.acq, &k.code, &at, &e.StatusID, &e.Name); err != nil {
				return nil, err
			}
			e.Date = at.In(chronosTZ)
			if p := byKey[k]; p != nil {
				p.Events = append(p.Events, e)
				months[e.Date.Format("2006-01")]++
			}
		}
		if err := ev.Err(); err != nil {
			return nil, err
		}
	}

	st, err := r.db.Query(ctx, `SELECT id, COALESCE(code, ''), name, COALESCE(color, ''), sort_order, is_released, is_final
		FROM parcel_status ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	for st.Next() {
		var s domain.ChronosStatus
		if err := st.Scan(&s.ID, &s.Code, &s.Name, &s.Color, &s.SortOrder, &s.IsReleased, &s.IsFinal); err != nil {
			return nil, err
		}
		out.Statuses = append(out.Statuses, s)
	}
	if err := st.Err(); err != nil {
		return nil, err
	}

	// Захирамжийн төсөл — дашбоардтай ИЖИЛ хүрээ (чөлөөлөлтөөр эсвэл идэвхтэй
	// нэгж талбарын холбоосоор) + явцын түүх (сонгосон сарын байдлыг тооцоход).
	if len(out.Parcels) > 0 {
		acqIDs := make([]uuid.UUID, 0)
		seenAcq := map[uuid.UUID]bool{}
		for _, p := range out.Parcels {
			if !seenAcq[p.AcquisitionID] {
				seenAcq[p.AcquisitionID] = true
				acqIDs = append(acqIDs, p.AcquisitionID)
			}
		}
		dr, err := r.db.Query(ctx, `
			WITH ids AS (
				SELECT dd.id FROM decision_draft dd WHERE dd.acquisition_id = ANY($1)
				UNION
				SELECT ddp.decision_draft_id FROM decision_draft_parcel ddp
				  JOIN parcel p ON p.id = ddp.parcel_id
				 WHERE p.acquisition_id = ANY($1) AND ddp.removed_at IS NULL
			)
			SELECT d.id, d.created_at, CASE WHEN d.status = $2 THEN COALESCE(d.confirmed_at, d.updated_at) END,
			       COALESCE((SELECT json_agg(json_build_object('type', h.progress_type,
			                    'date', to_char(h.progress_date, 'YYYY-MM-DD"T"00:00:00Z')) ORDER BY h.progress_date, h.created_at)
			                   FROM decision_draft_progress_history h WHERE h.decision_draft_id = d.id), '[]')::text
			  FROM ids JOIN decision_draft d ON d.id = ids.id`, acqIDs, domain.DecisionDraftStatusConfirmed)
		if err != nil {
			return nil, err
		}
		defer dr.Close()
		for dr.Next() {
			d := domain.ChronosDecision{Events: []domain.ChronosDecisionEvent{}}
			var events string
			if err := dr.Scan(&d.ID, &d.CreatedAt, &d.ConfirmedAt, &events); err != nil {
				return nil, err
			}
			_ = json.Unmarshal([]byte(events), &d.Events)
			toUB(&d.CreatedAt)
			toUB(d.ConfirmedAt)
			out.Decisions = append(out.Decisions, d)
		}
		if err := dr.Err(); err != nil {
			return nil, err
		}
	}

	// Нэгж талбар бүрийн түүхийн ЭХЭНД «Хүлээгдэж буй» — огноо нь чөлөөлөлт
	// үүсгэсэн өдөр (түүх түүнээс өмнө эхэлсэн бол эхний түүхийн өдөр). Санд
	// бичихгүй — зөвхөн Хроносын дүрслэлд.
	var pending *domain.ChronosStatus
	for i := range out.Statuses {
		if out.Statuses[i].ID == domain.ParcelStatusPending {
			pending = &out.Statuses[i]
			break
		}
	}
	if pending != nil {
		for _, p := range out.Parcels {
			at := created[p.ID]
			if len(p.Events) > 0 {
				if p.Events[0].StatusID == pending.ID {
					continue
				}
				if at == nil || p.Events[0].Date.Before(*at) {
					first := p.Events[0].Date
					at = &first
				}
			}
			if at == nil {
				continue
			}
			p.Events = append([]domain.ChronosEvent{{Date: *at, StatusID: pending.ID, Name: pending.Name}}, p.Events...)
			months[at.Format("2006-01")]++
		}
	}

	for m, n := range months {
		out.Months = append(out.Months, domain.ChronosMonth{Month: m, Count: n})
	}
	sortChronosMonths(out.Months)
	return out, nil
}

// chronosTZ — Хроносын огноог Улаанбаатарын цагаар. Frontend сарыг ISO мөрийн
// эхний 7 тэмдэгтээс авдаг; сервер UTC-ээр ажиллавал орон нутгийн шөнө дундын
// огноо (04-01 00:00+08) «03-31T16:00Z» болж ӨМНӨХ сард орно.
var chronosTZ = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Ulaanbaatar"); err == nil {
		return loc
	}
	return time.FixedZone("ULAT", 8*3600)
}()

func toUB(t *time.Time) {
	if t != nil {
		*t = t.In(chronosTZ)
	}
}

func sortChronosMonths(m []domain.ChronosMonth) {
	for i := 1; i < len(m); i++ {
		for j := i; j > 0 && m[j].Month < m[j-1].Month; j-- {
			m[j], m[j-1] = m[j-1], m[j]
		}
	}
}
