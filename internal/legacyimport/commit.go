package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Actor — импортоор үүссэн мөрүүдийн created_by. CLI-тэй ИЖИЛ тул дахин
// оруулахад (CLI эсвэл вэб) давхардуулахгүй, өөрийн мөрөө л шинэчилнэ.
const Actor = "legacy-import"

// defaultBudget — санхүүжилтийн төрөл тодорхойгүй захирамжийн төсөв (migration 000044).
const defaultBudget = "Нийслэл"

// importMark — assets.notes (created_by баганагүй хүснэгт).
const importMark = "Хуучин мэдээллийн импорт"

const holderFromExcel = "Тайлангийн эксэлээс"

const valuationNote = "Хуучин мэдээллийн импорт: үнэлгээ баталгаажсан төлөвтэй орсон"

// importedParcel — «Оруулах» явцад бүртгэгдсэн (эсвэл санд байсан) нэгж талбар.
type importedParcel struct {
	id                string
	parcelID          string
	acquisitionID     string
	status            int
	statusMapped      bool
	acquisitionAreaM2 float64
}

// commitLog — мөр тус бүрийн тэмдэглэл.
type commitLog struct{ CommitLog }

const maxLogEntries = 2000

func (l *commitLog) add(level, stage, key, msg string) {
	switch level {
	case "success":
		l.Success++
		return
	case "skipped":
		l.Skipped++
	case "failed":
		l.Failed++
	case "warning":
		l.Warnings++
	}
	if len(l.Entries) < maxLogEntries {
		l.Entries = append(l.Entries, LogEntry{Level: level, Stage: stage, Key: key, Message: msg})
	}
}

// inSavepoint — нэг мөрийн алдаа бусдыг унагахгүй.
func inSavepoint(ctx context.Context, tx pgx.Tx, fn func() error) error {
	if _, err := tx.Exec(ctx, "SAVEPOINT legacy_row"); err != nil {
		return err
	}
	if err := fn(); err != nil {
		if _, rerr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT legacy_row"); rerr != nil {
			return rerr
		}
		_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT legacy_row")
		return err
	}
	_, err := tx.Exec(ctx, "RELEASE SAVEPOINT legacy_row")
	return err
}

// own — импорт ШИНЭЭР үүсгэсэн дундын мөрийг бүртгэнэ («Хасах» үед зөвхөн
// өөр газар хэрэглэгдээгүй бол устгана).
func own(ctx context.Context, q querier, sessionID, kind, refID, label string) error {
	_, err := q.Exec(ctx, `INSERT INTO `+stage+`.owned (session_id, kind, ref_id, label) VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`, sessionID, kind, refID, label)
	return err
}

// parcelExists — энэ дугаартай нэгж талбар САНД БАЙГАА эсэх (устгагдсан ч).
// Байвал импорт ХӨНДӨХГҮЙ: өөр чөлөөлөлтийн мэдээлэлд нөлөөлөхгүйн тулд.
func parcelExists(ctx context.Context, q querier, parcelID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM parcel WHERE parcel_id = $1)`, parcelID).Scan(&ok)
	return ok, err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// commitSession — ТҮР ХҮСНЭГТЭЭС СИСТЕМ РҮҮ, НЭГ гүйлгээнд. Алдаа гарвал бүгд буцна.
func (s *Service) commitSession(ctx context.Context, sessionID string, choice CategoryChoice) (*CommitResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cats, err := resolveCategories(ctx, tx, choice)
	if err != nil {
		return nil, err
	}
	log := &commitLog{}
	result := &CommitResult{}

	acqs, byIdx, err := commitAcquisitions(ctx, tx, sessionID, cats, log)
	if err != nil {
		return nil, err
	}
	if len(byIdx) == 0 {
		return nil, errors.New("нэг ч чөлөөлөлт бүртгэгдсэнгүй")
	}
	result.Acquisitions = acqs
	acqIDs := make([]string, 0, len(byIdx))
	for _, id := range byIdx {
		acqIDs = append(acqIDs, id)
	}

	parcels, pstats, err := commitParcels(ctx, tx, sessionID, byIdx, log)
	if err != nil {
		return nil, err
	}
	result.Parcels = pstats

	comps, err := commitReports(ctx, tx, sessionID, acqIDs, parcels, log)
	if err != nil {
		return nil, err
	}
	result.Compensations = comps

	statuses, err := syncAcquisitionStatuses(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	result.AcquisitionStatuses = statuses

	for _, t := range []string{"report_row", "parcel", "boundary"} {
		if _, err := tx.Exec(ctx, `DELETE FROM `+stage+`.`+t+` WHERE session_id=$1`, sessionID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	result.Log = &log.CommitLog
	return result, nil
}

// ── 1. Чөлөөлөлт ────────────────────────────────────────────────────────────

func commitAcquisitions(ctx context.Context, tx pgx.Tx, sessionID string, cats *Categories, log *commitLog) (*CommitAcquisitions, map[int]string, error) {
	type staged struct {
		idx  int
		row  BoundaryRow
		plan *PlanProject
	}
	rows, err := tx.Query(ctx, `SELECT idx, row, plan FROM `+stage+`.boundary WHERE session_id=$1 ORDER BY idx`, sessionID)
	if err != nil {
		return nil, nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (staged, error) {
		var s staged
		return s, r.Scan(&s.idx, &s.row, &s.plan)
	})
	if err != nil {
		return nil, nil, err
	}
	out := &CommitAcquisitions{Names: []string{}}
	byIdx := map[int]string{}
	var subID *int
	if cats.Sub != nil {
		subID = &cats.Sub.ID
	}
	for _, it := range items {
		if it.plan == nil || it.row.WKT == "" {
			log.add("skipped", "acquisition", it.row.PlanParcelID, "төлөвлөгөө эсвэл хил байхгүй")
			continue
		}
		plan := it.plan
		planParcelID := plan.ParcelID
		if planParcelID == "" {
			planParcelID = it.row.PlanParcelID
		}
		name := acquisitionName(planParcelID, plan.Gazner)
		var acqID string
		err := inSavepoint(ctx, tx, func() error {
			var planID string
			// Санд байгаа төлөвлөгөөг ӨӨРЧЛӨХГҮЙ — байхгүй бол л үүсгэж эзэмшинэ.
			err := tx.QueryRow(ctx, `INSERT INTO plan (
					id, plan_code, project_id, name, plan_type_name, boundary_wkt, area_m2, status,
					start_data, end_data, approved_date, parcel_id, gazner)
				VALUES ($1, $2, NULLIF($3,''), $4, NULLIF($5,''),
					CASE WHEN NULLIF($6,'') IS NULL THEN NULL ELSE ST_GeomFromText($6, 4326) END,
					CASE WHEN NULLIF($6,'') IS NULL THEN NULL ELSE public.calculate_area_utm(ST_GeomFromText($6, 4326)) END,
					0, $7, $8, $9, NULLIF($10,''), NULLIF($11,''))
				ON CONFLICT (plan_code) DO NOTHING
				RETURNING id`,
				uuid.NewString(), plan.Code, plan.ProjectID, plan.Name, plan.PlanTypeName, plan.BoundaryWKT,
				nullStr(plan.StartData), nullStr(plan.EndData), nullStr(plan.ApprovedDate), plan.ParcelID, plan.Gazner,
			).Scan(&planID)
			switch {
			case err == nil:
				if err := own(ctx, tx, sessionID, "plan", planID, plan.Code); err != nil {
					return err
				}
			case errors.Is(err, pgx.ErrNoRows):
				if err := tx.QueryRow(ctx, `SELECT id FROM plan WHERE plan_code = $1`, plan.Code).Scan(&planID); err != nil {
					return fmt.Errorf("төлөвлөгөө олох: %w", err)
				}
			default:
				return fmt.Errorf("төлөвлөгөө хадгалах: %w", err)
			}

			start, end := nullStr(parseDate(plan.StartData)), nullStr(parseDate(plan.EndData))
			// Өмнө импортоор орсон чөлөөлөлтийг ДАХИН оруулахгүй, ШИНЭЧЛЭХГҮЙ —
			// импорт бүр зөвхөн ӨӨРИЙН үүсгэсэн мөрийг эзэмшинэ («Хасах» боломжтой).
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM land_acquisition
				WHERE split_part(acquisition_name, ' ', 1) = $1 AND created_by = $2 AND deleted_at IS NULL)`,
				planParcelID, Actor).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return errors.New("өмнө импортоор орсон чөлөөлөлт — дахин оруулахгүй (эхлээд тэр импортыг «Хасах»-аар устгана уу)")
			}
			acqID = uuid.NewString()
			if _, err := tx.Exec(ctx, `INSERT INTO land_acquisition (
					id, plan_id, plan_code, plan_name, plan_geom, plan_area_m2, geometry, area_m2,
					status, start_date, end_date, acquisition_name,
					implementing_org, reason, responsible_org, funding_source,
					decree_number, decree_date, general_category_id, sub_category_id,
					created_at, created_by, updated_at, updated_by, legacy_import_id)
				VALUES ($1, $2, $3, $4,
					CASE WHEN NULLIF($5,'') IS NULL THEN NULL ELSE ST_GeomFromText($5, 4326) END,
					CASE WHEN NULLIF($5,'') IS NULL THEN NULL ELSE public.calculate_area_utm(ST_GeomFromText($5, 4326)) END,
					ST_GeomFromText($6, 4326), public.calculate_area_utm(ST_GeomFromText($6, 4326)),
					1, $7, $8, $9, '', '', '', '', '', NULL, $11, $12, NOW(), $10, NOW(), $10, $13)`,
				acqID, planID, plan.Code, plan.Name, plan.BoundaryWKT, it.row.WKT, start, end, name, Actor,
				cats.General.ID, subID, sessionID); err != nil {
				return fmt.Errorf("чөлөөлөлт үүсгэх: %w", err)
			}
			if err := own(ctx, tx, sessionID, "land_acquisition", acqID, name); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO acquisition_progress (id, acquisition_id, from_status, to_status, note, changed_by, changed_at)
				VALUES ($1, $2, 0, 1, $3, $4, NOW())`, uuid.NewString(), acqID, "Хуучин мэдээллээс импортлов", Actor); err != nil {
				return err
			}
			return insertAcquisitionAU(ctx, tx, acqID, it.row.WKT)
		})
		if err != nil {
			log.add("failed", "acquisition", planParcelID, err.Error())
			continue
		}
		byIdx[it.idx] = acqID
		out.Names = append(out.Names, name)
		out.Created++
		log.add("success", "acquisition", planParcelID, name)
	}
	return out, byIdx, nil
}

// insertAcquisitionAU — хилтэй (EPSG:4326 WKT) огтлолцох баг/хороо.
func insertAcquisitionAU(ctx context.Context, q querier, acqID, wkt string) error {
	_, err := q.Exec(ctx, `INSERT INTO land_acquisition_au (
			land_acquisition_id, au1_code, au1_name, au2_code, au2_name, au3_code, au3_name,
			created_at, created_by, updated_at, updated_by)
		SELECT $1, COALESCE(a3.au1_code,''), COALESCE(a1.name,''), COALESCE(a3.au2_code,''), COALESCE(a2.name,''),
		       a3.code, COALESCE(a3.name,''), NOW(), $3, NOW(), $3
		  FROM au3 a3 LEFT JOIN au1 a1 ON a1.code = a3.au1_code LEFT JOIN au2 a2 ON a2.code = a3.au2_code
		 WHERE a3.geometry IS NOT NULL AND ST_Intersects(a3.geometry, ST_MakeValid(ST_GeomFromText($2, 4326)))
		ON CONFLICT (land_acquisition_id, au3_code) DO NOTHING`, acqID, wkt, Actor)
	return err
}

// ── 2. Нэгж талбар ──────────────────────────────────────────────────────────

// shpSource — ГУС-д олдоогүй нэгж талбарын мэдээлэл (SHP-ийн атрибутаас).
// Үндсэн хил ГУС-аас л ирнэ — SHP нь НӨЛӨӨЛСӨН хэсэг тул энд хоосон.
func shpSource(row ParcelRow, au [3]string) *SourceParcel {
	p := row.Properties
	return &SourceParcel{
		ParcelID: row.ParcelID, OldParcelID: row.OldParcelID,
		AU1Code: au[0], AU2Code: au[1], AU3Code: au[2],
		Landuse:        parcelKey(attr(p, "landuse")),
		AddressKhashaa: normalize(attr(p, "address_kh", "address_khashaa")),
		AddressStreet:  normalize(attr(p, "address_st", "address_streetname")),
		PropertyNo:     normalize(attr(p, "property_n", "property_no")),
		ValidFrom:      parseDate(attr(p, "valid_from")), ValidTill: parseDate(attr(p, "valid_till")),
		Origin: "shp",
	}
}

func commitParcels(ctx context.Context, tx pgx.Tx, sessionID string, acqByIdx map[int]string, log *commitLog) (map[string]*importedParcel, *CommitParcels, error) {
	type staged struct {
		parcelID    string
		row         ParcelRow
		boundaryIdx *int
		outside     bool
		source      *SourceParcel
		issues      []Issue
		au          [3]string
	}
	rows, err := tx.Query(ctx, `SELECT p.parcel_id, p.row, p.boundary_idx, p.outside, p.source, p.issues,
			COALESCE(b.row->'properties'->>'au1', b.row->'properties'->>'au1_code', ''),
			COALESCE(b.row->'properties'->>'au2', b.row->'properties'->>'au2_code', ''),
			COALESCE(b.row->'properties'->>'au3', b.row->'properties'->>'au3_code', '')
		FROM `+stage+`.parcel p
		LEFT JOIN `+stage+`.boundary b ON b.session_id = p.session_id AND b.idx = p.boundary_idx
		WHERE p.session_id=$1 ORDER BY p.idx`, sessionID)
	if err != nil {
		return nil, nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (staged, error) {
		var s staged
		return s, r.Scan(&s.parcelID, &s.row, &s.boundaryIdx, &s.outside, &s.source, &s.issues, &s.au[0], &s.au[1], &s.au[2])
	})
	if err != nil {
		return nil, nil, err
	}
	parcels := map[string]*importedParcel{}
	stats := &CommitParcels{}
	for _, it := range items {
		if it.parcelID == "" || hasCode(it.issues, "duplicate") || hasCode(it.issues, "empty_row") {
			continue
		}
		exists, err := parcelExists(ctx, tx, it.parcelID)
		if err != nil {
			return nil, nil, err
		}
		if exists {
			stats.AlreadyImported++
			log.add("skipped", "parcel", it.parcelID, "санд бүртгэлтэй нэгж талбар — энэ импортод хамаарахгүй тул хөндсөнгүй")
			continue
		}
		if it.boundaryIdx == nil || acqByIdx[*it.boundaryIdx] == "" {
			log.add("skipped", "parcel", it.parcelID, "хил (геометр) байхгүй эсвэл чөлөөлөлт бүртгэгдээгүй тул хуваарилах боломжгүй")
			continue
		}
		acqID := acqByIdx[*it.boundaryIdx]
		source := it.source
		if source == nil {
			source = shpSource(it.row, it.au)
			stats.GusMissing++
			log.add("warning", "parcel", it.parcelID, "ГУС-д олдсонгүй — SHP-ийн мэдээллээр бүртгэв (үндсэн хил, эрхийн төрөл дутуу)")
		} else if source.Origin == "ub" {
			stats.FromUb++
		}
		if it.outside {
			stats.Nearest++
		}
		mapping := mapStatusCode(it.row.SourceCode)
		row := it.row
		row.ParcelID = it.parcelID
		var saved *importedParcel
		err = inSavepoint(ctx, tx, func() error {
			var e error
			saved, e = upsertParcel(ctx, tx, acqID, row, source, row.WKT, source.GeometryWKT, mapping.status)
			if e != nil {
				return e
			}
			if e := own(ctx, tx, sessionID, "parcel", saved.id, saved.parcelID); e != nil {
				return e
			}
			if e := saveDetail(ctx, tx, saved.id, acqID, source); e != nil {
				return e
			}
			return saveStatusHistory(ctx, tx, it.parcelID, acqID, mapping.status, joinNonEmpty("; ", mapping.label, row.Note), "")
		})
		if err != nil {
			log.add("failed", "parcel", it.parcelID, "нэгж талбар хадгалахад алдаа: "+err.Error())
			continue
		}
		saved.statusMapped = mapping.mapped
		parcels[it.parcelID] = saved
		stats.Imported++
		log.add("success", "parcel", it.parcelID, "")
	}
	return parcels, stats, nil
}

// upsertParcel — geometry = ГУС-ийн үндсэн хил, acquisition_geom = нөлөөлсөн хэсэг.
func upsertParcel(ctx context.Context, q querier, acqID string, row ParcelRow, s *SourceParcel, affectedWKT, baseWKT string, status int) (*importedParcel, error) {
	old := s.OldParcelID
	if old == "" {
		old = row.OldParcelID
	}
	var p importedParcel
	err := q.QueryRow(ctx, `WITH raw AS (
			SELECT ST_GeomFromText(NULLIF($4, ''), 4326) AS base_geom, ST_GeomFromText(NULLIF($20, ''), 4326) AS affected_geom
		), geom AS (
			SELECT CASE WHEN base_geom IS NULL THEN NULL WHEN GeometryType(base_geom) = 'POLYGON' THEN base_geom
			            ELSE ST_GeometryN(ST_CollectionExtract(base_geom, 3), 1) END::geometry(Polygon, 4326) AS base_geom,
			       CASE WHEN affected_geom IS NULL THEN NULL WHEN GeometryType(affected_geom) = 'POLYGON' THEN affected_geom
			            ELSE ST_GeometryN(ST_CollectionExtract(affected_geom, 3), 1) END::geometry(Polygon, 4326) AS affected_geom
			  FROM raw
		), calc AS (
			SELECT base_geom,
			       CASE WHEN affected_geom IS NOT NULL AND NOT ST_IsEmpty(affected_geom) THEN affected_geom END AS affected_geom,
			       COALESCE(NULLIF($16::numeric, 0), public.calculate_area_utm(base_geom), 0) AS area_m2,
			       COALESCE(CASE WHEN affected_geom IS NOT NULL AND NOT ST_IsEmpty(affected_geom)
			                     THEN public.calculate_area_utm(affected_geom) END, 0) AS overlap_area
			  FROM geom
		)
		INSERT INTO parcel (
			id, parcel_id, old_parcel_id, acquisition_id, valid_from, valid_till, au1_code, au2_code, au3_code,
			geometry, acquisition_geom, acquisition_area_m2, remaining_area_m2,
			right_type, landuse, landuse_name, area_m2, address_khashaa, address_streetname, property_no,
			status, compensation_paid, db_changed, created_at, created_by, updated_at, updated_by, deleted_at)
		SELECT $1, $2, NULLIF($5, ''), $3, $6::date, $7::date, NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
		       calc.base_geom, calc.affected_geom, NULLIF(calc.overlap_area, 0),
		       CASE WHEN calc.area_m2 > 0 THEN GREATEST(calc.area_m2 - calc.overlap_area, 0) END,
		       $11, NULLIF($12, ''), NULLIF($13, ''), NULLIF(calc.area_m2, 0),
		       NULLIF($14, ''), NULLIF($15, ''), NULLIF($17, ''),
		       $18, false, '', NOW(), $19, NOW(), $19, NULL
		  FROM calc
		RETURNING id, parcel_id, acquisition_id, COALESCE(acquisition_area_m2, 0)::float8`,
		uuid.NewString(), row.ParcelID, acqID, baseWKT, truncate(old, oldParcelIDMax), nullStr(s.ValidFrom), nullStr(s.ValidTill),
		s.AU1Code, s.AU2Code, s.AU3Code, s.RightType, s.Landuse, s.LanduseName, s.AddressKhashaa, s.AddressStreet,
		s.AreaM2, s.PropertyNo, status, Actor, affectedWKT,
	).Scan(&p.id, &p.parcelID, &p.acquisitionID, &p.acquisitionAreaM2)
	if err != nil {
		return nil, fmt.Errorf("parcel хадгалах: %w", err)
	}
	p.status = status
	return &p, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return ""
	}
	return s
}

func saveDetail(ctx context.Context, q querier, parcelUUID, acqID string, s *SourceParcel) error {
	if _, err := q.Exec(ctx, `DELETE FROM parcel_detail WHERE parcel_id = $1 AND created_by = $2`, parcelUUID, Actor); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `INSERT INTO parcel_detail (
			id, parcel_id, acquisition_id, right_type, holder_id, app_no, app_timestamps, app_type, app_status,
			decision_no, decision_date, contract_no, contract_date, contract_begin, contract_end,
			contract_property_no, contract_status, certificate_no, certificate_date,
			record_no, record_date, record_certificate_no, record_status,
			valuation_zone, base_price_per_ha, auction_coeff, auction_price, created_at, created_by, updated_at, updated_by)
		VALUES ($1, $2, $3, $4,
			(SELECT holder_id FROM parcel_holder WHERE parcel_id = $2 AND holder_role = 'holder'
			  ORDER BY main_applicant DESC, created_at ASC LIMIT 1),
			NULLIF($5,''), $6::date, NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), $10::date,
			NULLIF($11,''), $12::date, $13::date, $14::date, NULLIF($15,''), NULLIF($16,''),
			NULLIF($17,''), $18::date, NULLIF($19,''), $20::date, NULLIF($21,''), NULLIF($22,''),
			NULL, NULL, NULL, NULL, NOW(), $23, NOW(), $23)`,
		uuid.NewString(), parcelUUID, acqID, s.RightType, s.AppNo, nullStr(s.AppTimestamp), s.AppType, s.AppStatus,
		s.DecisionNo, nullStr(s.DecisionDate), s.ContractNo, nullStr(s.ContractDate), nullStr(s.ContractBegin),
		nullStr(s.ContractEnd), s.ContractPropertyNo, s.ContractStatus, s.CertificateNo, nullStr(s.CertificateDate),
		s.RecordNo, nullStr(s.RecordDate), s.RecordCertificateNo, s.RecordStatus, Actor)
	return err
}

// saveStatusHistory — statusDate хоосон бол өнөөдөр.
func saveStatusHistory(ctx context.Context, q querier, parcelID, acqID string, status int, reason, statusDate string) error {
	if _, err := q.Exec(ctx, `DELETE FROM parcel_status_history WHERE parcel_id = $1 AND created_by = $2`, parcelID, Actor); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, status_date, created_by, reason)
		VALUES ($1, $2, $3, COALESCE($6::timestamptz, NOW()), $4, NULLIF($5, ''))`,
		parcelID, acqID, status, Actor, strings.TrimSpace(reason), nullStr(statusDate))
	return err
}

// ── 3. Үнэлгээ ба нөхөх олговор ─────────────────────────────────────────────

func commitReports(ctx context.Context, tx pgx.Tx, sessionID string, acqIDs []string, parcels map[string]*importedParcel, log *commitLog) (*CommitCompensations, error) {
	type staged struct {
		row    ReportRow
		match  *string
		source *SourceParcel
	}
	rows, err := tx.Query(ctx, `SELECT row, match, source FROM `+stage+`.report_row WHERE session_id=$1 ORDER BY idx`, sessionID)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (staged, error) {
		var s staged
		return s, r.Scan(&s.row, &s.match, &s.source)
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	// Хуучин дугаараар таарах SHP-ийн нэгж талбар.
	byOld := map[string]*importedParcel{}
	orows, err := tx.Query(ctx, `SELECT parcel_id, row->>'oldParcelID' FROM `+stage+`.parcel
		WHERE session_id=$1 AND COALESCE(row->>'oldParcelID','') <> ''`, sessionID)
	if err != nil {
		return nil, err
	}
	for orows.Next() {
		var id, old string
		if err := orows.Scan(&id, &old); err != nil {
			orows.Close()
			return nil, err
		}
		if p := parcels[id]; p != nil {
			byOld[old] = p
		}
	}
	orows.Close()

	out := &CommitCompensations{}
	touched := map[string]bool{}
	for _, it := range items {
		row := it.row
		kind, _, _ := strings.Cut(derefOr(it.match, "missing"), ":")
		var parcel *importedParcel
		switch kind {
		case "staged":
			parcel = parcels[row.ParcelID]
			if parcel == nil && row.OldParcelID != "" {
				parcel = byOld[row.OldParcelID]
			}
		case "db":
			// Санд өмнө нь бүртгэлтэй нэгж талбар — өөр чөлөөлөлтийн мэдээлэлд
			// нөлөөлөхгүйн тулд энэ импорт үнэлгээ бичихгүй.
			log.add("skipped", "compensation", row.ParcelID, fmt.Sprintf("санд бүртгэлтэй нэгж талбар — энэ импортод хамаарахгүй тул %s₮ орсонгүй", fmtInt(row.amount())))
			continue
		case "gus":
			exists, err := parcelExists(ctx, tx, row.ParcelID)
			if err != nil {
				return nil, err
			}
			if parcel = parcels[row.ParcelID]; parcel == nil && it.source != nil && !exists {
				p, err := importReportParcel(ctx, tx, sessionID, acqIDs, row, it.source)
				if err != nil {
					log.add("failed", "parcel", row.ParcelID, "ГУС-аас нэгж талбар бүртгэхэд алдаа: "+err.Error())
				}
				parcel = p
			}
		}
		if parcel == nil {
			out.MissingParcels++
			reason := "нэгж талбар олдоогүй"
			if kind == "staged" || kind == "gus" {
				reason = "нэгж талбар санд бүртгэлтэй тул энэ импортод хамаарахгүй"
			}
			log.add("skipped", "compensation", row.ParcelID, fmt.Sprintf("%s — %s₮ орсонгүй (%s, мөр %d)", reason, fmtInt(row.amount()), row.SourceSheet, row.ExcelRow))
			continue
		}
		parcels[parcel.parcelID] = parcel
		row.ParcelID = parcel.parcelID
		var total float64
		err := inSavepoint(ctx, tx, func() error {
			var e error
			if total, e = applyReportRow(ctx, tx, sessionID, &row, parcel, log); e != nil {
				return e
			}
			return own(ctx, tx, sessionID, "valuation", parcel.id, parcel.parcelID)
		})
		if err != nil {
			log.add("failed", "compensation", row.ParcelID, "нөхөх олговор хадгалахад алдаа: "+err.Error())
			continue
		}
		out.Applied++
		out.TotalAmount += total
		touched[parcel.id] = true
		log.add("success", "compensation", row.ParcelID, "")
	}
	ids := make([]string, 0, len(touched))
	for id := range touched {
		ids = append(ids, id)
	}
	if err := reconcileFunding(ctx, tx, ids); err != nil {
		return nil, err
	}
	return out, nil
}

// importReportParcel — нэгж талбарын SHP-д байхгүй, ГУС-аас үндсэн хилтэй
// олдсон нэгж талбар. Чөлөөлөлтийг үндсэн хилийн давхцлаар (эс бөгөөс хамгийн
// ойроор) сонгож, нөлөөлсөн хэсгийг хил ба чөлөөлөлтийн огтлолцлоор бодно.
func importReportParcel(ctx context.Context, tx pgx.Tx, sessionID string, acqIDs []string, row ReportRow, src *SourceParcel) (*importedParcel, error) {
	var acqID string
	err := tx.QueryRow(ctx, `SELECT la.id FROM land_acquisition la
		WHERE la.id = ANY($1::uuid[]) AND la.geometry IS NOT NULL
		ORDER BY ST_Area(ST_Intersection(ST_MakeValid(la.geometry), ST_MakeValid(ST_GeomFromText($2, 4326)))) DESC,
		         ST_Distance(ST_MakeValid(la.geometry)::geography, ST_MakeValid(ST_GeomFromText($2, 4326))::geography), la.id
		LIMIT 1`, acqIDs, src.GeometryWKT).Scan(&acqID)
	if err != nil {
		return nil, err
	}
	var affected string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(ST_AsText(ST_CollectionExtract(ST_Intersection(
			ST_MakeValid(geometry), ST_MakeValid(ST_GeomFromText($2,4326))), 3)), '')
		FROM land_acquisition WHERE id=$1`, acqID, src.GeometryWKT).Scan(&affected); err != nil {
		return nil, err
	}
	pr := ParcelRow{ParcelID: row.ParcelID, OldParcelID: row.OldParcelID, AreaM2: row.AreaM2, Note: row.Note}
	var saved *importedParcel
	err = inSavepoint(ctx, tx, func() error {
		var e error
		if saved, e = upsertParcel(ctx, tx, acqID, pr, src, affected, src.GeometryWKT, statusPending); e != nil {
			return e
		}
		if e := own(ctx, tx, sessionID, "parcel", saved.id, saved.parcelID); e != nil {
			return e
		}
		if e := saveDetail(ctx, tx, saved.id, acqID, src); e != nil {
			return e
		}
		return saveStatusHistory(ctx, tx, row.ParcelID, acqID, statusPending, row.Note, "")
	})
	if err != nil {
		return nil, err
	}
	saved.statusMapped = false
	return saved, nil
}

// compensationDate — архивын sheet-ийн он давуу; эс бөгөөс захирамжийн огноо.
func compensationDate(r *ReportRow) string {
	if r.SheetYear == nil {
		return r.DecreeDate
	}
	if r.DecreeDate != "" && strings.HasPrefix(r.DecreeDate, strconv.Itoa(*r.SheetYear)) {
		return r.DecreeDate
	}
	return fmt.Sprintf("%d-12-31", *r.SheetYear)
}

func balanceNote(b balancedAmounts) string {
	if b.adjustment == 0 {
		return ""
	}
	reason := "задаргааны нийлбэр нийт дүнгээс дутуу байсан"
	if b.breakdownMissing {
		reason = "тайлан дээр задаргаа байгаагүй"
	}
	return fmt.Sprintf("«Нийт нөхөх олговор»-той тэнцүүлэв: газрын үнэд %s₮ нэмэв (%s)", fmtInt(b.adjustment), reason)
}

// applyReportRow — тайлангийн мөрийг үнэлгээ/олговор/хөрөнгө/эзэмшигч болгоно.
func applyReportRow(ctx context.Context, tx pgx.Tx, sessionID string, row *ReportRow, parcel *importedParcel, log *commitLog) (float64, error) {
	acqID := parcel.acquisitionID
	var rightType *int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(pd.right_type, p.right_type) FROM parcel p
		LEFT JOIN parcel_detail pd ON pd.parcel_id = p.id WHERE p.id = $1
		ORDER BY pd.right_type NULLS LAST LIMIT 1`, parcel.id).Scan(&rightType); err != nil {
		return 0, err
	}

	owners, ownerErr := splitOwners(row.HolderName, row.HolderRegisterNo)
	if ownerErr != nil {
		log.add("warning", "compensation", row.ParcelID, "өмчлөгч бүртгэгдсэнгүй — "+ownerErr.Error())
	}
	oldID := row.OldParcelID
	if len(oldID) > oldParcelIDMax {
		log.add("warning", "compensation", row.ParcelID, fmt.Sprintf("хуучин дугаар %s (%d орон) хадгалагдсангүй", oldID, len(oldID)))
		oldID = ""
	}
	if issue, err := saveReportLinks(ctx, tx, sessionID, row, parcel); err != nil {
		return 0, err
	} else if issue != "" {
		log.add("warning", "compensation", row.ParcelID, issue)
	}

	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM compensation WHERE acquisition_id = $1 AND parcel_id = $2 AND created_by = $3`, []any{acqID, row.ParcelID, Actor}},
		{`DELETE FROM assets WHERE acquisition_id = $1 AND parcel_id = $2 AND notes = $3`, []any{acqID, row.ParcelID, importMark}},
		{`DELETE FROM land_valuation WHERE acquisition_id = $1 AND parcel_id = $2 AND valuation_type = 'asset'`, []any{acqID, row.ParcelID}},
	} {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			return 0, err
		}
	}

	compType, _ := compensationType(row.CompensationTypeLabel)
	date := compensationDate(row)
	balanced := balanceAmounts(row)
	note := joinNonEmpty(" | ", row.Decree, row.Note)

	var landArea *float64
	if row.AffectedAreaM2 != nil {
		landArea = row.AffectedAreaM2
	} else if parcel.acquisitionAreaM2 > 0 {
		v := parcel.acquisitionAreaM2
		landArea = &v
	}
	if balanced.landAmount > 0 || row.CertificateNo != "" || landArea != nil {
		area, price := 0.0, 0.0
		if landArea != nil {
			area = *landArea
			if area > 0 {
				price = balanced.landAmount / area
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO land_valuation (
				id, acquisition_id, parcel_id, land_area_m2, base_price_per_m2, ownership_cert_no,
				appraiser_org_name, appraiser_reg_no, appraiser_state_reg_no, appraiser_director, appraiser_license,
				appraiser_address, appraiser_contact, source_file_name, source_file_hash, valuation_type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, '', '', '', '', '', '', '', '', $6, '', 'asset', NOW(), NOW())`,
			uuid.NewString(), acqID, row.ParcelID, area, price, row.SourceFile); err != nil {
			return 0, err
		}
	}
	insertComp := func(assetID *string, target string, amount float64, compNote string) error {
		_, err := tx.Exec(ctx, `INSERT INTO compensation (
				id, acquisition_id, parcel_id, asset_id, compensation_type, coverage_percent, amount, compensation_date,
				note, target_type, status, review_note, reviewed_by, reviewed_at, valuation_report_url,
				valuation_report_name, valuation_type, created_at, created_by, updated_at)
			VALUES ($1, $2, $3, $4, $5, 100, $6, $7::date, $8, $9, 'approved', '', $10, NOW(), '', '', 'asset', NOW(), $10, NOW())`,
			uuid.NewString(), acqID, row.ParcelID, assetID, compType, amount, nullStr(date), compNote, target, Actor)
		return err
	}
	if balanced.landAmount > 0 {
		if err := insertComp(nil, "parcel", balanced.landAmount, joinNonEmpty(" | ", note, balanceNote(balanced))); err != nil {
			return 0, err
		}
	}
	realNumbers := assetNumbers(row.AssetNumber)
	if len(realNumbers) == 0 && row.RealStateAmount > 0 {
		realNumbers = []string{""}
	}
	var propertyNumbers []string
	if row.PropertyAmount > 0 {
		propertyNumbers = []string{""}
	}
	for _, g := range []struct {
		kind, name string
		amount     float64
		numbers    []string
	}{
		{"real_state", "Үл хөдлөх хөрөнгө", row.RealStateAmount, realNumbers},
		{"property", "Эд хөрөнгө", row.PropertyAmount, propertyNumbers},
	} {
		combined := ""
		if len(g.numbers) > 1 {
			combined = "Хамтын нийлбэр үнэлгээ; хөрөнгө тус бүрийн задаргаа байхгүй: " + strings.Join(g.numbers, ", ")
		}
		for i, number := range g.numbers {
			assetID := uuid.NewString()
			if _, err := tx.Exec(ctx, `INSERT INTO assets (
					id, acquisition_id, parcel_id, asset_number, asset_type, asset_name, floor_count, area_m2,
					owner_name, address, notes, unit, capacity, description, unit_price, valuation_type)
				VALUES ($1,$2,$3,$4,$5,$6,0,0,$7,$8,$9,'','',$10,0,'asset')`,
				assetID, acqID, row.ParcelID, number, g.kind, g.name, row.HolderName, row.Address, importMark, combined); err != nil {
				return 0, err
			}
			// Нийлбэр дүнг эхний хөрөнгийн холбоосоор НЭГ удаа хадгална.
			if i == 0 && g.amount > 0 {
				if err := insertComp(&assetID, "asset", g.amount, joinNonEmpty(" | ", note, combined)); err != nil {
					return 0, err
				}
			}
		}
	}

	var total float64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0)::float8 FROM compensation
		WHERE acquisition_id=$1 AND parcel_id=$2 AND valuation_type='asset'`, acqID, row.ParcelID).Scan(&total); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE parcel SET
			compensation_paid = $2::boolean,
			old_parcel_id     = COALESCE(NULLIF($9::text, ''), old_parcel_id),
			db_changed        = COALESCE(NULLIF($3::text, ''), db_changed),
			changed_parcel_id = COALESCE(NULLIF($4::text, ''), changed_parcel_id),
			right_type        = COALESCE($5::integer, right_type),
			area_m2           = COALESCE($6::numeric, NULLIF(area_m2, 0)),
			acquisition_area_m2 = COALESCE($8::numeric, acquisition_area_m2),
			remaining_area_m2 = GREATEST(COALESCE($6::numeric, NULLIF(area_m2, 0), 0) - COALESCE($8::numeric, acquisition_area_m2, 0), 0),
			updated_at = NOW(), updated_by = $7
		WHERE parcel_id = $1`,
		row.ParcelID, total > 0, row.DBChangeNote, row.ChangedParcelID, rightType, row.AreaM2, Actor, row.AffectedAreaM2, oldID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE parcel_detail SET right_type = COALESCE($2::integer, right_type) WHERE parcel_id = $1`,
		parcel.id, rightType); err != nil {
		return 0, err
	}
	status := parcel.status
	if !parcel.statusMapped {
		guessed, ok := statusFromNote(row.Note)
		if !ok {
			guessed, ok = statusFromNote(row.Decree)
		}
		if ok && guessed != parcel.status {
			if _, err := tx.Exec(ctx, `UPDATE parcel SET status = $2 WHERE parcel_id = $1`, row.ParcelID, guessed); err != nil {
				return 0, err
			}
			status = guessed
			parcel.status = guessed
		}
	}
	if err := saveStatusHistory(ctx, tx, row.ParcelID, acqID, status, joinNonEmpty("; ", row.Note, row.Decree), date); err != nil {
		return 0, err
	}
	if ownerErr == nil && len(owners) > 0 {
		if err := saveExcelHolders(ctx, tx, sessionID, parcel.id, acqID, row, owners); err != nil {
			return 0, err
		}
	}
	if total > 0 {
		ok, err := approveValuation(ctx, tx, parcel.id, row.ParcelID, acqID)
		if err != nil {
			return 0, err
		}
		if !ok {
			log.add("warning", "compensation", row.ParcelID, "үнэлгээний төлөвийг хүн аль хэдийн өөрчилсөн тул баталгаажуулалт хийгдсэнгүй")
		}
	}
	return total, nil
}

var reLegalEntity = regexp.MustCompile(`(?i)(ххк|төхк|тöхк|\bхк\b|ххн|нөхөрлөл|компани|corporation|llc)`)

func saveExcelHolders(ctx context.Context, tx pgx.Tx, sessionID, parcelUUID, acqID string, row *ReportRow, owners []owner) error {
	if _, err := tx.Exec(ctx, `DELETE FROM parcel_holder WHERE parcel_id=$1 AND holder_role='holder' AND created_by=$2`, parcelUUID, Actor); err != nil {
		return err
	}
	first := ""
	for i, o := range owners {
		company := reLegalEntity.MatchString(o.full)
		last, name, kind := cut(o.lastName, 100), cut(firstNonEmpty(o.name, o.lastName), 100), "citizen"
		if company {
			last, name, kind = "", cut(o.full, 100), "company"
		}
		// Санд байгаа хүнийг (регистрээр) ӨӨРЧЛӨХГҮЙ — байхгүй бол л үүсгэж эзэмшинэ.
		var holderID string
		err := tx.QueryRow(ctx, `INSERT INTO right_holder (
				id, last_name, name, register_no, address, holder_type, created_at, created_by, updated_at, updated_by)
			VALUES ($1, NULLIF($2,''), $3, NULLIF($4,''), NULLIF($5,''), $6, NOW(), $7, NOW(), $7)
			ON CONFLICT (register_no) DO NOTHING
			RETURNING id`, uuid.NewString(), last, name, o.registerNo, row.Address, kind, Actor).Scan(&holderID)
		switch {
		case err == nil:
			if err := own(ctx, tx, sessionID, "right_holder", holderID, name); err != nil {
				return err
			}
		case errors.Is(err, pgx.ErrNoRows):
			if err := tx.QueryRow(ctx, `SELECT id FROM right_holder WHERE register_no = $1`, o.registerNo).Scan(&holderID); err != nil {
				return err
			}
		default:
			return err
		}
		if first == "" {
			first = holderID
		}
		if _, err := tx.Exec(ctx, `INSERT INTO parcel_holder (
				id, parcel_id, acquisition_id, holder_id, app_no, main_applicant, holder_role, payment_recipient, note,
				created_at, created_by, updated_at, updated_by)
			SELECT $1,$2,$3,$4,NULL,$5,'holder',$5,$6,NOW(),$7,NOW(),$7
			WHERE NOT EXISTS (SELECT 1 FROM parcel_holder WHERE parcel_id=$2 AND holder_id=$4 AND holder_role='holder')`,
			uuid.NewString(), parcelUUID, acqID, holderID, i == 0, holderFromExcel, Actor); err != nil {
			return err
		}
	}
	if first == "" {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE parcel_detail SET holder_id = $2, updated_at = NOW(), updated_by = $3 WHERE parcel_id = $1`,
		parcelUUID, first, Actor); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO parcel_detail (id, parcel_id, acquisition_id, holder_id, created_by, updated_by, right_type)
		SELECT $1,$2,$3,$4,$5,$5,right_type FROM parcel WHERE id=$2 AND NOT EXISTS (SELECT 1 FROM parcel_detail WHERE parcel_id=$2)`,
		uuid.NewString(), parcelUUID, acqID, first, Actor)
	return err
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// saveReportLinks — чөлөөлөлт/төлөвлөгөөний нэмэлт мэдээлэл, хаяг, санхүүжилт,
// захирамж. Буцаах string нь холбоос үүсээгүй шалтгаан (анхааруулга).
func saveReportLinks(ctx context.Context, tx pgx.Tx, sessionID string, row *ReportRow, parcel *importedParcel) (string, error) {
	acqID := parcel.acquisitionID
	// Төлөвлөгөөний «Бүтээн байгуулалт» — ЗӨВХӨН энэ импортын үүсгэсэн төлөвлөгөөнд.
	if _, err := tx.Exec(ctx, `UPDATE plan SET gazner=$2, updated_at=NOW()
		WHERE id=(SELECT plan_id FROM land_acquisition WHERE id=$1) AND NULLIF($2,'') IS NOT NULL
		  AND id IN (SELECT ref_id FROM `+stage+`.owned WHERE session_id=$3 AND kind='plan')`, acqID, row.ProjectName, sessionID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE land_acquisition SET
			nith_decree_number=COALESCE(NULLIF($2,''),nith_decree_number),
			group_list_number=COALESCE(NULLIF($3,''),group_list_number),
			funding_source=CASE WHEN NULLIF($4,'') IS NULL THEN funding_source
				WHEN COALESCE(funding_source,'')='' THEN btrim($4)
				WHEN EXISTS (SELECT 1 FROM unnest(string_to_array(funding_source,'; ')) AS existing
				             WHERE lower(btrim(existing)) = lower(btrim($4))) THEN funding_source
				ELSE funding_source || '; ' || btrim($4) END,
			updated_at=NOW(), updated_by=$5
		WHERE id=$1`, acqID, row.DecisionNo, row.ListNo, row.FundingSource, Actor); err != nil {
		return "", err
	}
	addr := parseAddress(row.Address)
	street := addr.street
	if street == "" && addr.khashaa == "" {
		street = addr.raw
	}
	if _, err := tx.Exec(ctx, `UPDATE parcel SET address_streetname=COALESCE(NULLIF($2,''),address_streetname),
			address_khashaa=COALESCE(NULLIF($3,''),address_khashaa), property_no=COALESCE(NULLIF($4,''),property_no)
		WHERE id=$1`, parcel.id, street, addr.khashaa, row.CertificateNo); err != nil {
		return "", err
	}
	if addr.district != "" || addr.au3 != "" {
		rows, err := tx.Query(ctx, `SELECT au1_code, au2_code, au3_code FROM land_acquisition_au
			WHERE land_acquisition_id=$1
			  AND ($2='' OR lower(au2_name) LIKE '%' || lower($2) || '%')
			  AND ($3='' OR substring(au3_name from '[0-9]+')=$3)`, acqID, addr.district, addr.au3)
		if err != nil {
			return "", err
		}
		type au struct{ a1, a2, a3 string }
		aus, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (au, error) {
			var v au
			return v, r.Scan(&v.a1, &v.a2, &v.a3)
		})
		if err != nil {
			return "", err
		}
		if len(aus) == 1 {
			if _, err := tx.Exec(ctx, `UPDATE parcel SET au1_code=$2, au2_code=$3, au3_code=$4 WHERE id=$1`,
				parcel.id, aus[0].a1, aus[0].a2, aus[0].a3); err != nil {
				return "", err
			}
		}
	}

	var fundingID *string
	if funding := normalize(row.FundingSource); funding != "" {
		parts := regexp.MustCompile(`\s*—\s*|\s+-\s+`).Split(funding, -1)
		organization := parts[0]
		sourceType := strings.Join(parts[1:], " — ")
		if sourceType == "" {
			sourceType = organization
		}
		rows, err := tx.Query(ctx, `SELECT id FROM funding_sources WHERE acquisition_id=$1
			AND lower(btrim(organization_name))=lower(btrim($2)) AND lower(btrim(source_type))=lower(btrim($3))`,
			acqID, organization, sourceType)
		if err != nil {
			return "", err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return "", err
		}
		if len(ids) > 1 {
			return "", fmt.Errorf("санхүүгийн эх үүсвэр давхардсан: %s", row.FundingSource)
		}
		if len(ids) == 1 {
			fundingID = &ids[0]
		} else {
			id := uuid.NewString()
			if _, err := tx.Exec(ctx, `INSERT INTO funding_sources(id,acquisition_id,organization_name,source_type,currency,note,created_by)
				VALUES($1,$2,$3,$4,'MNT',$5,$6)`, id, acqID, organization, sourceType,
				fmt.Sprintf("Хуучин мэдээллийн импорт: %s / %s", row.SourceFile, row.SourceSheet), Actor); err != nil {
				return "", err
			}
			if err := own(ctx, tx, sessionID, "funding_source", id, organization); err != nil {
				return "", err
			}
			fundingID = &id
		}
	}

	decree, err := parseDecree(row.Decree)
	if err != nil {
		return err.Error(), nil
	}
	if decree == nil {
		return "", nil
	}
	if decree.date == "" {
		return fmt.Sprintf("Захирамж %s: огноогүй тул аль оных нь тодорхойгүй — холбоос үүсгэсэнгүй", decree.number), nil
	}
	year, _ := strconv.Atoi(decree.date[:4])
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, fmt.Sprintf("legacy-decree:%s:%d", decree.number, year)); err != nil {
		return "", err
	}
	var draftID string
	err = tx.QueryRow(ctx, `SELECT id FROM decision_draft
		WHERE regexp_replace(replace(upper(decree_number),'A','А'),'[-/[:space:]]','','g') = regexp_replace($1,'[-/[:space:]]','','g')
		  AND decision_date IS NOT NULL AND EXTRACT(YEAR FROM decision_date) = $2
		ORDER BY created_at, id LIMIT 1`, decree.number, year).Scan(&draftID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		draftID = uuid.NewString()
		workType, err := decisionOption(ctx, tx, "decision_work_type", row.ConstructionType)
		if err != nil {
			return "", err
		}
		budget, err := decisionOption(ctx, tx, "decision_budget", row.FundingSource)
		if err != nil {
			return "", err
		}
		if budget == nil {
			// Санхүүжилтийн төрөл (төсөв) тодорхойгүй — өгөгдмөл нь «Нийслэл».
			if budget, err = decisionOption(ctx, tx, "decision_budget", defaultBudget); err != nil {
				return "", err
			}
		}
		location := firstNonEmpty(row.ProjectName, row.Address)
		if _, err := tx.Exec(ctx, `INSERT INTO decision_draft
				(id,proposal_no,decree_number,decision_date,location,duration_year,status,acquisition_id,
				 confirmed_at,confirmed_by,created_by,updated_by,work_type_id,budget_id)
			VALUES($1,$2,$3,$4::date,$5,$6,2,$7,NOW(),$8,$8,$8,$9,$10)`,
			draftID, fmt.Sprintf("%d/%s", year, decree.number), decree.number, decree.date, location, year,
			acqID, Actor, workType, budget); err != nil {
			return "", err
		}
		if err := own(ctx, tx, sessionID, "decision_draft", draftID, decree.number); err != nil {
			return "", err
		}
	case err != nil:
		return "", err
		// default: санд байгаа захирамж — ӨӨРЧЛӨХГҮЙ, зөвхөн энэ импортын нэгж
		// талбарыг холбоно (холбоос нь нэгж талбартайгаа хамт устна).
	}

	var fundingLinkID *string
	if fundingID != nil {
		var id string
		var inserted bool
		// xmax = 0 — шинээр үүссэн мөр (ON CONFLICT-оор олдсон биш).
		if err := tx.QueryRow(ctx, `INSERT INTO decision_draft_funding_source (decision_draft_id,funding_source_id,created_by)
			VALUES($1,$2,$3)
			ON CONFLICT(decision_draft_id,funding_source_id) DO UPDATE SET funding_source_id=EXCLUDED.funding_source_id
			RETURNING id, (xmax = 0)`, draftID, *fundingID, Actor).Scan(&id, &inserted); err != nil {
			return "", err
		}
		if inserted {
			if err := own(ctx, tx, sessionID, "funding_link", id, decree.number); err != nil {
				return "", err
			}
		}
		fundingLinkID = &id
	}

	// Нэг нэгж талбар олон захирамжид орсон байж болно — хамгийн СҮҮЛИЙН
	// огноотойг идэвхтэй болгож, бусдыг түүхэнд үлдээнэ (uq_ddp_active_parcel).
	var currentID, currentDraft, currentDate string
	err = tx.QueryRow(ctx, `SELECT p.id, p.decision_draft_id, COALESCE(to_char(d.decision_date,'YYYY-MM-DD'),'')
		FROM decision_draft_parcel p JOIN decision_draft d ON d.id=p.decision_draft_id
		WHERE p.parcel_id=$1 AND p.removed_at IS NULL LIMIT 1`, parcel.id).Scan(&currentID, &currentDraft, &currentDate)
	hasCurrent := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if hasCurrent && currentDraft != draftID {
		var pastID string
		perr := tx.QueryRow(ctx, `SELECT id FROM decision_draft_parcel WHERE parcel_id=$1 AND decision_draft_id=$2 LIMIT 1`,
			parcel.id, draftID).Scan(&pastID)
		if perr != nil && !errors.Is(perr, pgx.ErrNoRows) {
			return "", perr
		}
		if decree.date > currentDate {
			if _, err := tx.Exec(ctx, `UPDATE decision_draft_parcel SET removed_at=NOW(), removed_by=$2 WHERE id=$1`, currentID, Actor); err != nil {
				return "", err
			}
			if pastID != "" {
				_, err = tx.Exec(ctx, `UPDATE decision_draft_parcel SET removed_at=NULL, removed_by='',
					funding_link_id=COALESCE($2::uuid, funding_link_id) WHERE id=$1`, pastID, fundingLinkID)
			} else {
				_, err = tx.Exec(ctx, `INSERT INTO decision_draft_parcel(decision_draft_id,parcel_id,funding_link_id,linked_by)
					VALUES($1,$2,$3,$4)`, draftID, parcel.id, fundingLinkID, Actor)
			}
			return "", err
		}
		if pastID == "" {
			_, err = tx.Exec(ctx, `INSERT INTO decision_draft_parcel (decision_draft_id,parcel_id,funding_link_id,linked_by,removed_at,removed_by)
				VALUES($1,$2,$3,$4,NOW(),$4)`, draftID, parcel.id, fundingLinkID, Actor)
		}
		return "", err
	}
	if hasCurrent {
		_, err = tx.Exec(ctx, `UPDATE decision_draft_parcel SET funding_link_id=COALESCE($2::uuid,funding_link_id) WHERE id=$1`,
			currentID, fundingLinkID)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO decision_draft_parcel(decision_draft_id,parcel_id,funding_link_id,linked_by)
			VALUES($1,$2,$3,$4)`, draftID, parcel.id, fundingLinkID, Actor)
	}
	return "", err
}

func decisionOption(ctx context.Context, tx pgx.Tx, table, value string) (*int, error) {
	v := normalize(value)
	if v == "" {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `SELECT id FROM `+table+` WHERE lower(trim(name))=lower($1) OR lower(trim(code))=lower($1)`, v)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return nil, err
	}
	if len(ids) > 1 {
		return nil, fmt.Errorf("%s: лавлах давхардсан: %s", table, value)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &ids[0], nil
}

// reconcileFunding — давтан импортод дүн нэмэхгүй: холбогдсон олговроос дахин нийлүүлнэ.
func reconcileFunding(ctx context.Context, tx pgx.Tx, parcelUUIDs []string) error {
	if len(parcelUUIDs) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE decision_draft_funding_source f SET amount=COALESCE((
			SELECT SUM(c.amount) FROM decision_draft_parcel dp JOIN parcel p ON p.id=dp.parcel_id
			  JOIN compensation c ON c.parcel_id=p.parcel_id AND c.acquisition_id=p.acquisition_id
			WHERE dp.funding_link_id=f.id AND dp.removed_at IS NULL AND c.valuation_type='asset'),0)
		WHERE f.created_by=$1 AND f.id IN (SELECT funding_link_id FROM decision_draft_parcel
		                                   WHERE parcel_id=ANY($2::uuid[]) AND removed_at IS NULL)`, Actor, parcelUUIDs); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE funding_sources f SET amount=(
			SELECT COALESCE(SUM(amount),0) FROM decision_draft_funding_source WHERE funding_source_id=f.id)
		WHERE f.created_by=$1 AND f.id IN (SELECT d.funding_source_id FROM decision_draft_funding_source d
		  JOIN decision_draft_parcel p ON p.funding_link_id=d.id WHERE p.parcel_id=ANY($2::uuid[]) AND p.removed_at IS NULL)`,
		Actor, parcelUUIDs)
	return err
}

// approveValuation — импортын үнэлгээг БАТАЛГААЖСАН болгоно. Хүний оруулсан
// төлөв байвал хөндөхгүй (false).
func approveValuation(ctx context.Context, tx pgx.Tx, parcelUUID, parcelID, acqID string) (bool, error) {
	tag, err := tx.Exec(ctx, `INSERT INTO parcel_valuation_submission (
			acquisition_id, parcel_id, valuation_type, status, submitted_by, submitted_at, reviewed_by, reviewed_at, last_note)
		VALUES ($1, $2, 'asset', 'approved', $3, NOW(), $3, NOW(), $4)
		ON CONFLICT (acquisition_id, parcel_id, valuation_type) DO UPDATE SET
			status = 'approved', submitted_by = EXCLUDED.submitted_by, submitted_at = NOW(),
			reviewed_by = EXCLUDED.reviewed_by, reviewed_at = NOW(), last_note = EXCLUDED.last_note, updated_at = NOW()
		WHERE parcel_valuation_submission.status = 'draft' OR parcel_valuation_submission.reviewed_by = $3`,
		acqID, parcelID, Actor, valuationNote)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM parcel_valuation_submission_history
		WHERE acquisition_id = $1 AND parcel_id = $2 AND valuation_type = 'asset' AND created_by = $3`, acqID, parcelID, Actor); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO parcel_valuation_submission_history (
			acquisition_id, parcel_id, valuation_type, action, from_status, to_status, note, created_by)
		VALUES ($1, $2, 'asset', 'submit', 'draft', 'submitted', $3, $4),
		       ($1, $2, 'asset', 'approve', 'submitted', 'approved', $3, $4)`, acqID, parcelID, valuationNote, Actor); err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `UPDATE parcel SET selected_valuation_type = 'asset', updated_at = NOW(), updated_by = $2
		WHERE id = $1 AND (selected_valuation_type IS NULL OR selected_valuation_type = 'asset')`, parcelUUID, Actor)
	return err == nil, err
}

// syncAcquisitionStatuses — импортын чөлөөлөлт: явц 100% → 3 Баталгаажсан,
// бусад үед → 2 Хээрийн судалгаа. Цуцлагдсаныг (4) хөндөхгүй.
func syncAcquisitionStatuses(ctx context.Context, tx pgx.Tx, sessionID string) (*AcquisitionStatuses, error) {
	rows, err := tx.Query(ctx, `WITH progress AS (
			SELECT la.id, la.status AS from_status,
			       CASE WHEN COUNT(p.id) > 0 AND COUNT(p.id) = COUNT(p.id) FILTER (WHERE COALESCE(p.status, 0) IN (SELECT id FROM parcel_status WHERE is_final))
			            THEN 3 ELSE 2 END AS to_status
			  FROM land_acquisition la LEFT JOIN parcel p ON p.acquisition_id = la.id
			 WHERE la.legacy_import_id = $2 AND la.deleted_at IS NULL AND COALESCE(la.status, 0) <> 4
			 GROUP BY la.id, la.status
		), updated AS (
			UPDATE land_acquisition la SET status = pr.to_status, updated_at = NOW(), updated_by = $1
			  FROM progress pr WHERE la.id = pr.id AND la.status IS DISTINCT FROM pr.to_status
			RETURNING la.id, pr.from_status, pr.to_status
		), history AS (
			INSERT INTO acquisition_progress (acquisition_id, from_status, to_status, note, changed_by, changed_at)
			SELECT id, COALESCE(from_status, 0), to_status,
			       CASE WHEN to_status = 3 THEN 'Хуучин мэдээллийн импорт: бүх нэгж талбар эцсийн төлөвт (явц 100%)'
			            ELSE 'Хуучин мэдээллийн импорт: явц 100% хүрээгүй' END, $1, NOW()
			  FROM updated
		)
		SELECT pr.to_status, (u.id IS NOT NULL) FROM progress pr LEFT JOIN updated u ON u.id = pr.id`, Actor, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &AcquisitionStatuses{}
	for rows.Next() {
		var to int
		var changed bool
		if err := rows.Scan(&to, &changed); err != nil {
			return nil, err
		}
		if to == 3 {
			out.Verified++
		} else {
			out.Field++
		}
		if changed {
			out.Changed++
		}
	}
	return out, rows.Err()
}
