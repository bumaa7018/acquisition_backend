package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// commitValuationSession — «Зөвхөн үнэлгээ» горимын «Оруулах» (НЭГ гүйлгээ):
//
//  1. Төсөл бүрд чөлөөлөлт — ангилал, мэдээлэл, хил нь гараар тохируулснаас
//     (valuation_projects.go); тохируулаагүй бол ангилал төрлөөс, ХИЛГҮЙ
//  2. Мөр бүрд нэгж талбар — ГУС-ын үндсэн хил (газрын зурагт харагдахын тулд
//     acquisition_geom-д ч), эс бөгөөс эксэлийн мэдээллээр
//  3. Үнэлгээ/олговор/эзэмшигч/захирамж — бүтэн горимтой ИЖИЛ (applyReportRow)
//
// Бүх мөр импортын бүртгэлд (owned) орно — «Устгах» нь бүтэн горимтой ижил.
func (s *Service) commitValuationSession(ctx context.Context, sessionID string) (*CommitResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	type staged struct {
		row               ReportRow
		match             string
		source            *SourceParcel
		project, assigned string
	}
	rows, err := tx.Query(ctx, `SELECT row, COALESCE(match, ''), source, COALESCE(project, ''), COALESCE(assigned_id, '')
		FROM `+stage+`.report_row WHERE session_id=$1 ORDER BY idx`, sessionID)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (staged, error) {
		var s staged
		return s, r.Scan(&s.row, &s.match, &s.source, &s.project, &s.assigned)
	})
	if err != nil {
		return nil, err
	}

	log := &commitLog{}
	result := &CommitResult{Acquisitions: &CommitAcquisitions{Names: []string{}}, Parcels: &CommitParcels{}, Compensations: &CommitCompensations{}}

	// ── 1. Чөлөөлөлт (төсөл бүрд) ───────────────────────────────────────────
	types := map[string]map[string]int{}
	var projects []string
	for _, it := range items {
		if it.project == "" || !valuationImportable(it.match) {
			continue
		}
		if types[it.project] == nil {
			types[it.project] = map[string]int{}
			projects = append(projects, it.project)
		}
		if t := strings.TrimSpace(it.row.ConstructionType); t != "" {
			types[it.project][t]++
		}
	}
	settings, err := projectSettings(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	acqByProject := map[string]string{}
	for _, name := range projects {
		set := settings[name]
		info := set.Info
		var wkt string
		if set.Boundary != nil {
			wkt = set.Boundary.WKT
		}
		var acqID string
		var cats *Categories
		err := inSavepoint(ctx, tx, func() error {
			var err error
			// Гараар сонгосон ангилал (ID) — эс бөгөөс бүтээн байгуулалтын төрлөөс.
			if set.Category != nil {
				cats, err = resolveCategories(ctx, tx, *set.Category)
			} else {
				category, _ := categoryForConstructionType(dominantType(types[name]))
				cats, err = resolveCategoryNames(ctx, tx, category)
			}
			if err != nil {
				return err
			}
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM land_acquisition
				WHERE acquisition_name = $1 AND created_by = $2 AND deleted_at IS NULL)`, name, Actor).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return errors.New("өмнө импортоор орсон чөлөөлөлт — дахин оруулахгүй (эхлээд тэр импортыг «Устгах»-аар устгана уу)")
			}
			var subID *int
			if cats.Sub != nil {
				subID = &cats.Sub.ID
			}
			acqID = uuid.NewString()
			// Төлөвлөгөөгүй; хил нь зөвхөн гараар оруулсан shapefile-аас (эс бөгөөс ХИЛГҮЙ).
			if _, err := tx.Exec(ctx, `INSERT INTO land_acquisition (
					id, plan_code, acquisition_name, status, general_category_id, sub_category_id,
					implementing_org, reason, responsible_org, funding_source, decree_number,
					nith_decree_number, group_list_number, start_date, end_date, geometry, area_m2,
					created_at, created_by, updated_at, updated_by, legacy_import_id)
				VALUES ($1, '', $2, 1, $3, $4, $5, $6, $7, '', '', $8, $9, $10::date, $11::date,
					CASE WHEN NULLIF($12,'') IS NULL THEN NULL ELSE ST_GeomFromText($12, 4326) END,
					CASE WHEN NULLIF($12,'') IS NULL THEN NULL ELSE public.calculate_area_utm(ST_GeomFromText($12, 4326)) END,
					NOW(), $13, NOW(), $13, $14)`,
				acqID, name, cats.General.ID, subID, info.ImplementingOrg, info.Reason, info.ResponsibleOrg,
				info.NithDecreeNumber, info.GroupListNumber, nullStr(info.StartDate), nullStr(info.EndDate), wkt,
				Actor, sessionID); err != nil {
				return fmt.Errorf("чөлөөлөлт үүсгэх: %w", err)
			}
			if err := own(ctx, tx, sessionID, "land_acquisition", acqID, name); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO acquisition_progress (id, acquisition_id, from_status, to_status, note, changed_by, changed_at)
				VALUES ($1, $2, 0, 1, $3, $4, NOW())`, uuid.NewString(), acqID, "Хуучин мэдээллийн үнэлгээнээс импортлов", Actor); err != nil {
				return err
			}
			if wkt == "" {
				return nil
			}
			return insertAcquisitionAU(ctx, tx, acqID, wkt)
		})
		if err != nil {
			log.add("failed", "acquisition", name, err.Error())
			continue
		}
		acqByProject[name] = acqID
		result.Acquisitions.Created++
		result.Acquisitions.Names = append(result.Acquisitions.Names, name)
		detail := joinNonEmpty(" / ", cats.General.Name, subName(cats))
		if wkt != "" {
			detail += " · хилтэй"
		}
		log.add("success", "acquisition", name, detail)
	}
	if len(acqByProject) == 0 {
		return nil, errors.New("нэг ч чөлөөлөлт бүртгэгдсэнгүй")
	}

	// ── 2–3. Нэгж талбар ба үнэлгээ (мөр бүрд) ─────────────────────────────────
	touched := map[string]bool{}
	for _, it := range items {
		row := it.row
		label := rowLabel(&row)
		switch {
		case it.match == matchDB:
			result.Parcels.AlreadyImported++
			log.add("skipped", "compensation", label, fmt.Sprintf("санд бүртгэлтэй нэгж талбар — энэ импортод хамаарахгүй тул %s₮ орсонгүй", fmtInt(row.amount())))
			continue
		case it.match == matchNoProject:
			result.Compensations.MissingParcels++
			log.add("skipped", "compensation", label, fmt.Sprintf("«Бүтээн байгуулалтын ажлын нэр» хоосон — %s₮ орсонгүй (%s, мөр %d)", fmtInt(row.amount()), row.SourceSheet, row.ExcelRow))
			continue
		case !valuationImportable(it.match):
			return nil, fmt.Errorf("%s: ГУС-ын шалгалт дуусаагүй мөр байна", label)
		}
		acqID := acqByProject[it.project]
		if acqID == "" {
			log.add("skipped", "compensation", label, "чөлөөлөлт бүртгэгдээгүй тул орсонгүй")
			continue
		}
		var total float64
		err := inSavepoint(ctx, tx, func() error {
			src := valuationSource(&row, it.source, it.assigned)
			released := row.amount() > 0 || row.LandAmount > 0
			status := statusPending
			if released {
				// Архивын адил: олговор олгосон мөр — «Чөлөөлсөн».
				status = statusReleased
			}
			pr := ParcelRow{ParcelID: it.assigned, OldParcelID: row.OldParcelID, AreaM2: row.AreaM2, Note: row.Note}
			saved, err := upsertParcel(ctx, tx, acqID, pr, src, src.GeometryWKT, src.GeometryWKT, status)
			if err != nil {
				return err
			}
			if err := own(ctx, tx, sessionID, "parcel", saved.id, saved.parcelID); err != nil {
				return err
			}
			if err := saveDetail(ctx, tx, saved.id, acqID, src); err != nil {
				return err
			}
			if err := saveStatusHistory(ctx, tx, it.assigned, acqID, status, row.Note, compensationDate(&row)); err != nil {
				return err
			}
			saved.statusMapped = released
			r := row
			r.ParcelID = it.assigned
			if total, err = applyReportRow(ctx, tx, sessionID, &r, saved, log); err != nil {
				return err
			}
			touched[saved.id] = true
			return own(ctx, tx, sessionID, "valuation", saved.id, saved.parcelID)
		})
		if err != nil {
			log.add("failed", "compensation", label, "нэгж талбар/үнэлгээ хадгалахад алдаа: "+err.Error())
			continue
		}
		result.Parcels.Imported++
		switch {
		case it.source == nil:
			result.Parcels.GusMissing++
		case it.source.Origin == "ub":
			result.Parcels.FromUb++
		}
		result.Compensations.Applied++
		result.Compensations.TotalAmount += total
		log.add("success", "compensation", it.assigned, it.project)
	}
	// Эксэлийн мөр бүр НИТХ тогтоол / жагсаалтын дугаарыг чөлөөлөлтөд бичдэг —
	// гараар оруулсан нь ДАВУУ (хоосон бол эксэлийнх үлдэнэ).
	for name, acqID := range acqByProject {
		info := settings[name].Info
		if info.NithDecreeNumber == "" && info.GroupListNumber == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE land_acquisition SET
				nith_decree_number = COALESCE(NULLIF($2, ''), nith_decree_number),
				group_list_number = COALESCE(NULLIF($3, ''), group_list_number)
			WHERE id = $1`, acqID, info.NithDecreeNumber, info.GroupListNumber); err != nil {
			return nil, err
		}
	}
	ids := make([]string, 0, len(touched))
	for id := range touched {
		ids = append(ids, id)
	}
	if err := reconcileFunding(ctx, tx, ids); err != nil {
		return nil, err
	}
	statuses, err := syncAcquisitionStatuses(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	result.AcquisitionStatuses = statuses
	if _, err := tx.Exec(ctx, `DELETE FROM `+stage+`.report_row WHERE session_id=$1`, sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	result.Log = &log.CommitLog
	return result, nil
}

func subName(c *Categories) string {
	if c.Sub == nil {
		return ""
	}
	return c.Sub.Name
}

// valuationSource — ГУС-ын мэдээлэл (байвал), эксэл ДАВУУ: тайлангийн талбай
// нь тухайн үеийн бодит байдлыг тусгадаг тул ГУС-ийн одоогийн утгаар дарахгүй.
func valuationSource(row *ReportRow, gus *SourceParcel, assigned string) *SourceParcel {
	var src SourceParcel
	if gus != nil {
		src = *gus
	} else {
		src = SourceParcel{ParcelID: assigned, OldParcelID: row.OldParcelID, Origin: "excel"}
	}
	src.ParcelID = assigned
	if row.AreaM2 != nil && *row.AreaM2 > 0 {
		v := *row.AreaM2
		src.AreaM2 = &v
	}
	if src.PropertyNo == "" {
		src.PropertyNo = row.CertificateNo
	}
	return &src
}
