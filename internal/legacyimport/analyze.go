package legacyimport

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// querier — *pgxpool.Pool, pgx.Tx аль алинд нь тохирно.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const stage = "legacy_import_stage"

// simplified — газрын зурагт харуулах WKT-ийг хөнгөлнө (≈1 м).
func simplified(col string) string {
	return "ST_AsText(ST_SimplifyPreserveTopology(" + col + ", 0.00001))"
}

const (
	planInsideWarnPct   = 90.0 // хилийн энэ хувиас бага нь төлөвлөгөөний дотор бол анхааруулна
	parcelInsideWarnPct = 99.0 // нэгж талбарын энэ хувиас бага нь хилийн дотор бол «хэсэгчлэн гадна»
	areaDiffWarnPct     = 5.0  // эксэлийн нөлөөлсөн талбай ба SHP-ийн талбайн зөрүү
)

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// readShapefiles — файлуудыг уншиж, заавал байх атрибутыг шалгана.
func readShapefiles(files []UploadedFile, required []struct {
	label string
	names []string
}) ([]FileInfo, [][]feature, []int) {
	var infos []FileInfo
	var all [][]feature
	var srids []int
	for _, set := range groupShapefiles(files) {
		info := FileInfo{Name: set.base + ".shp", Issues: set.issues()}
		if countLevel(info.Issues, "error") > 0 {
			infos = append(infos, info)
			all, srids = append(all, nil), append(srids, 0)
			continue
		}
		features, srid, err := readShapefileSet(set)
		if err != nil {
			info.Issues = append(info.Issues, errIssue("unreadable", fmt.Sprintf("%s: уншиж чадсангүй — %s", info.Name, err.Error())))
			features = nil
		} else if len(features) == 0 {
			info.Issues = append(info.Issues, errIssue("empty", info.Name+": объект (feature) байхгүй"))
			features = nil
		} else {
			for _, r := range required {
				if !hasAttr(features, r.names...) {
					info.Issues = append(info.Issues, errIssue("missing_field", fmt.Sprintf(
						"%s: «%s» атрибут олдсонгүй (%s). Файлд байгаа атрибутууд: %s",
						info.Name, r.label, strings.Join(r.names, " / "), attrNames(features))))
					features = nil
				}
			}
		}
		info.Features = len(features)
		infos = append(infos, info)
		all, srids = append(all, features), append(srids, srid)
	}
	return infos, all, srids
}

// transformSQL — эх проекцоос EPSG:4326.
const transformSQL = `CASE WHEN $W = '' THEN NULL ELSE ST_Transform(ST_SetSRID(ST_GeomFromText($W), $S::int), 4326) END`

func geomExpr(wktParam, sridParam string) string {
	return strings.NewReplacer("$W", wktParam, "$S", sridParam).Replace(transformSQL)
}

// outsideMongolia — хөрвүүлсний дараах шалгалт (.prj буруу байж болзошгүй).
func outsideMongolia(ctx context.Context, q querier, table, sessionID string) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM `+stage+`.`+table+`
		WHERE session_id=$1 AND geom IS NOT NULL
		  AND NOT ST_Intersects(geom, ST_MakeEnvelope(87, 41, 120, 52.5, 4326))`, sessionID).Scan(&n)
	return n, err
}

// ── 1. ЧӨЛӨӨЛӨЛТИЙН ХИЛ ─────────────────────────────────────────────────────

func (s *Service) analyzeBoundary(ctx context.Context, q querier, sessionID string, files []UploadedFile, choice CategoryChoice) (*BoundaryView, error) {
	for _, t := range []string{"boundary", "parcel", "report_row"} {
		if _, err := q.Exec(ctx, `DELETE FROM `+stage+`.`+t+` WHERE session_id=$1`, sessionID); err != nil {
			return nil, err
		}
	}
	view := &BoundaryView{Files: []FileInfo{}, Rows: []BoundaryViewRow{}, Map: []BoundaryMap{}}
	var sessionIssues []Issue
	cats, err := resolveCategories(ctx, q, choice)
	if err != nil {
		sessionIssues = append(sessionIssues, errIssue("category", err.Error()))
	}
	view.Categories = cats

	infos, sets, srids := readShapefiles(files, []struct {
		label string
		names []string
	}{
		{"Төлөвлөгөөний нэгж талбарын дугаар", planParcelFields},
	})
	if len(infos) == 0 {
		sessionIssues = append(sessionIssues, errIssue("no_shp", "Чөлөөлөлтийн хилийн .shp файл сонгогдоогүй байна"))
	}

	seen := map[string]int{}
	idx := 0
	for fi, features := range sets {
		info := &infos[fi]
		for _, f := range features {
			idx++
			row := BoundaryRow{Index: f.index, PlanParcelID: parcelKey(attr(f.props, planParcelFields...)), WKT: f.wkt, Properties: f.props}
			var issues []Issue
			var plan *PlanProject
			switch {
			case row.PlanParcelID == "":
				issues = append(issues, errIssue("no_plan_id", fmt.Sprintf("%s #%d: төлөвлөгөөний дугаар хоосон", info.Name, f.index)))
			case seen[row.PlanParcelID] > 0:
				issues = append(issues, errIssue("duplicate", fmt.Sprintf("%s: давхардсан — #%d мөрөнд аль хэдийн байна", row.PlanParcelID, seen[row.PlanParcelID])))
			default:
				seen[row.PlanParcelID] = idx
				plan, err = s.gus.PlanProject(ctx, row.PlanParcelID)
				switch {
				case err != nil:
					issues = append(issues, errIssue("gus_failed", fmt.Sprintf("%s: ГУС /plan/project дуудлага амжилтгүй — %s", row.PlanParcelID, err.Error())))
				case plan == nil:
					issues = append(issues, errIssue("plan_not_found", row.PlanParcelID+": ГУС-аас төлөвлөгөө олдсонгүй — дугаар зөв эсэхийг шалгана уу"))
				}
			}
			if row.WKT == "" {
				issues = append(issues, errIssue("no_geometry", fmt.Sprintf("%s #%d: хил (геометр) байхгүй", info.Name, f.index)))
			}

			if _, err := q.Exec(ctx, `INSERT INTO `+stage+`.boundary (session_id, idx, file_name, plan_parcel_id, row, plan, geom)
				VALUES ($1,$2,$3,$4,$5,$6, `+geomExpr("$7", "$8")+`)`,
				sessionID, idx, info.Name, row.PlanParcelID, row, plan, row.WKT, srids[fi]); err != nil {
				return nil, fmt.Errorf("хилийн мөр хадгалах: %w", err)
			}
			// Хөрвүүлсэн (4326) геометрийг эх мөрт буцааж бичнэ — «Оруулах» үүнийг хэрэглэнэ.
			if _, err := q.Exec(ctx, `UPDATE `+stage+`.boundary SET row = jsonb_set(row, '{wkt}', to_jsonb(COALESCE(ST_AsText(geom), '')))
				WHERE session_id=$1 AND idx=$2`, sessionID, idx); err != nil {
				return nil, err
			}

			vr := BoundaryViewRow{Idx: idx, File: info.Name, PlanParcelID: row.PlanParcelID, PlanFound: plan != nil}
			if row.WKT != "" {
				var planWKT string
				if plan != nil {
					planWKT = plan.BoundaryWKT
				}
				var gtype, reason string
				var valid bool
				var area, insidePct *float64
				var intersects *bool
				err := q.QueryRow(ctx, `SELECT GeometryType(b.geom), ST_IsValid(b.geom), ST_IsValidReason(b.geom),
						public.calculate_area_utm(ST_MakeValid(b.geom))::float8,
						CASE WHEN p.geom IS NULL THEN NULL ELSE ST_Intersects(ST_MakeValid(b.geom), p.geom) END,
						CASE WHEN p.geom IS NULL THEN NULL ELSE
						  (100 * ST_Area(ST_Intersection(ST_MakeValid(b.geom), p.geom)::geography)
						       / NULLIF(ST_Area(ST_MakeValid(b.geom)::geography), 0))::float8 END
					FROM `+stage+`.boundary b
					LEFT JOIN LATERAL (SELECT ST_MakeValid(ST_GeomFromText(NULLIF($3,''),4326)) AS geom) p ON true
					WHERE b.session_id=$1 AND b.idx=$2`, sessionID, idx, planWKT).
					Scan(&gtype, &valid, &reason, &area, &intersects, &insidePct)
				if err != nil {
					return nil, fmt.Errorf("хилийн геометр шалгах: %w", err)
				}
				vr.AreaM2 = area
				id := row.PlanParcelID
				if !strings.Contains(gtype, "POLYGON") {
					issues = append(issues, errIssue("not_polygon", fmt.Sprintf("%s: хил нь полигон биш (%s)", id, gtype)))
				}
				if !valid {
					issues = append(issues, warnIssue("invalid_geometry", fmt.Sprintf("%s: геометр алдаатай (%s) — автоматаар засаж (ST_MakeValid) бүртгэнэ", id, reason)))
				}
				switch {
				case plan != nil && plan.BoundaryWKT == "":
					issues = append(issues, warnIssue("plan_no_boundary", id+": төлөвлөгөөний хил ГУС-д байхгүй — хилийг тулгаж чадсангүй"))
				case intersects != nil && !*intersects:
					issues = append(issues, errIssue("outside_plan", id+": чөлөөлөлтийн хил төлөвлөгөөний хилтэй ОГТЛОЛЦОХГҮЙ — өөр төлөвлөгөөний файл эсвэл проекц буруу байж болзошгүй"))
				case insidePct != nil:
					v := round1(*insidePct)
					vr.InsidePlanPct = &v
					if v < planInsideWarnPct {
						issues = append(issues, warnIssue("partly_outside_plan", fmt.Sprintf("%s: хилийн зөвхөн %.1f%% нь төлөвлөгөөний хилийн дотор байна", id, v)))
					}
				}
			}

			planParcelID := row.PlanParcelID
			if plan != nil && plan.ParcelID != "" {
				planParcelID = plan.ParcelID
			}
			var existing bool
			if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM land_acquisition
				WHERE split_part(acquisition_name,' ',1)=$1 AND created_by=$2 AND deleted_at IS NULL)`,
				planParcelID, Actor).Scan(&existing); err != nil {
				return nil, err
			}
			if existing {
				issues = append(issues, errIssue("existing", planParcelID+": энэ чөлөөлөлт өмнө импортоор орсон — дахин оруулахгүй. Дахин оруулах бол «Хуучин мэдээлэл оруулах» жагсаалтаас тэр импортыг «Хасах»-аар устгана уу"))
			}
			if plan != nil {
				vr.Name = acquisitionName(planParcelID, plan.Gazner)
				vr.PlanCode = plan.Code
			}
			vr.Existing = existing
			vr.Issues = nonNilIssues(issues)
			if _, err := q.Exec(ctx, `UPDATE `+stage+`.boundary SET issues=$3 WHERE session_id=$1 AND idx=$2`, sessionID, idx, vr.Issues); err != nil {
				return nil, err
			}
			view.Rows = append(view.Rows, vr)
		}
	}
	if n, err := outsideMongolia(ctx, q, "boundary", sessionID); err == nil && n > 0 {
		sessionIssues = append(sessionIssues, warnIssue("outside_mongolia",
			fmt.Sprintf("%d хилийн координат Монголын хүрээнээс гадна — .prj (проекц) зөв эсэхийг шалгана уу", n)))
	}

	rows, err := q.Query(ctx, `SELECT idx, `+simplified("geom")+`,
			CASE WHEN NULLIF(plan->>'boundaryWKT','') IS NULL THEN NULL
			     ELSE `+simplified("ST_GeomFromText(plan->>'boundaryWKT',4326)")+` END
		FROM `+stage+`.boundary WHERE session_id=$1 AND geom IS NOT NULL ORDER BY idx`, sessionID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m BoundaryMap
		if err := rows.Scan(&m.Idx, &m.Boundary, &m.Plan); err != nil {
			rows.Close()
			return nil, err
		}
		view.Map = append(view.Map, m)
	}
	rows.Close()

	if len(sessionIssues) > 0 {
		view.Files = append(view.Files, FileInfo{Issues: sessionIssues})
	}
	view.Files = append(view.Files, infos...)
	var all []Issue
	for _, f := range view.Files {
		all = append(all, f.Issues...)
	}
	for _, r := range view.Rows {
		all = append(all, r.Issues...)
	}
	view.Errors, view.Warnings = countLevel(all, "error"), countLevel(all, "warning")
	view.CanProceed = view.Errors == 0 && len(view.Rows) > 0
	for i := range view.Files {
		view.Files[i].Issues = nonNilIssues(view.Files[i].Issues)
	}
	return view, rows.Err()
}

func nonNilIssues(issues []Issue) []Issue {
	if issues == nil {
		return []Issue{}
	}
	return issues
}

// acquisitionName — "<төлөвлөгөөний дугаар> <gazner>".
func acquisitionName(planParcelID, gazner string) string {
	clean := strings.TrimSpace(gazner)
	if planParcelID == "" || strings.HasPrefix(clean, planParcelID) {
		return clean
	}
	return strings.TrimSpace(planParcelID + " " + clean)
}

// ── 2. НЭГЖ ТАЛБАР ──────────────────────────────────────────────────────────

func (s *Service) stageParcels(ctx context.Context, q querier, sessionID string, files []UploadedFile) ([]FileInfo, error) {
	for _, t := range []string{"parcel", "report_row"} {
		if _, err := q.Exec(ctx, `DELETE FROM `+stage+`.`+t+` WHERE session_id=$1`, sessionID); err != nil {
			return nil, err
		}
	}
	infos, sets, srids := readShapefiles(files, []struct {
		label string
		names []string
	}{
		{"Нэгж талбарын дугаар", parcelIDFields},
	})
	if len(infos) == 0 {
		return []FileInfo{{Issues: []Issue{errIssue("no_shp", "Нэгж талбарын .shp файл сонгогдоогүй байна")}}}, nil
	}
	seen := map[string]int{}
	idx := 0
	for fi, features := range sets {
		info := &infos[fi]
		if features != nil && !hasAttr(features, statusCodeFields...) {
			info.Issues = append(info.Issues, warnIssue("no_status_code",
				info.Name+": төлөвийн код (code) атрибут байхгүй — бүх нэгж талбар «Хүлээгдэж буй» төлөвтэй орно"))
		}
		for _, f := range features {
			idx++
			row := ParcelRow{
				Index: f.index, ParcelID: parcelKey(attr(f.props, parcelIDFields...)),
				OldParcelID: parcelKey(attr(f.props, "old_parcel", "old_parcel_id")),
				SourceCode:  intAttr(f.props, statusCodeFields...),
				AreaM2:      parseOptionalNumber(attr(f.props, "area_m2", "area")),
				WKT:         f.wkt, Note: normalize(attr(f.props, "тайлб", "тайлбар", "note", "desc")),
				Properties: f.props,
			}
			if _, err := q.Exec(ctx, `INSERT INTO `+stage+`.parcel (session_id, idx, file_name, parcel_id, row, geom)
				VALUES ($1,$2,$3,'',$4, `+geomExpr("$5", "$6")+`)`,
				sessionID, idx, info.Name, row, row.WKT, srids[fi]); err != nil {
				return nil, fmt.Errorf("нэгж талбарын мөр хадгалах: %w", err)
			}
			// Хөрвүүлсэн WKT — зохиомол дугаар ба «Оруулах» үүнийг хэрэглэнэ.
			var wkt4326 string
			if err := q.QueryRow(ctx, `UPDATE `+stage+`.parcel SET row = jsonb_set(row, '{wkt}', to_jsonb(COALESCE(ST_AsText(geom), '')))
				WHERE session_id=$1 AND idx=$2 RETURNING row->>'wkt'`, sessionID, idx).Scan(&wkt4326); err != nil {
				return nil, err
			}
			var issues []Issue
			parcelID := row.ParcelID
			switch {
			case parcelID == "" && wkt4326 == "":
				issues = append(issues, errIssue("empty_row", fmt.Sprintf("%s #%d: дугаар ч, хил ч байхгүй — алгасагдана", info.Name, f.index)))
			case parcelID == "":
				parcelID = syntheticParcelID(wkt4326)
				issues = append(issues, warnIssue("synthetic", fmt.Sprintf("%s #%d: дугааргүй — зохиомол %s дугаараар бүртгэгдэнэ (дараа нь гараар засна)", info.Name, f.index, parcelID)))
			}
			if parcelID != "" {
				if first, dup := seen[parcelID]; dup {
					issues = append(issues, warnIssue("duplicate", fmt.Sprintf("%s: давхардсан мөр — эхнийх нь (#%d) хүчинтэй, энэ нь алгасагдана", parcelID, first)))
				} else {
					seen[parcelID] = idx
				}
				if wkt4326 == "" {
					issues = append(issues, warnIssue("no_geometry", parcelID+": хил (геометр) байхгүй — чөлөөлөлтөд хуваарилагдахгүй"))
				}
			}
			if m := mapStatusCode(row.SourceCode); !m.mapped && row.SourceCode != nil {
				issues = append(issues, warnIssue("unmapped_code", fmt.Sprintf("%s: төлөвийн код %d тохируулгад байхгүй — «Хүлээгдэж буй» болно", parcelID, *row.SourceCode)))
			}
			if _, err := q.Exec(ctx, `UPDATE `+stage+`.parcel SET parcel_id=$3, issues=$4 WHERE session_id=$1 AND idx=$2`,
				sessionID, idx, parcelID, nonNilIssues(issues)); err != nil {
				return nil, err
			}
		}
	}
	if n, err := outsideMongolia(ctx, q, "parcel", sessionID); err == nil && n > 0 {
		infos = append(infos, FileInfo{Issues: []Issue{warnIssue("outside_mongolia",
			fmt.Sprintf("%d нэгж талбарын координат Монголын хүрээнээс гадна — .prj (проекц) зөв эсэхийг шалгана уу", n))}})
	}

	// Байршил — хамгийн их давхцалтай хил, эс бөгөөс хамгийн ойрынх.
	if _, err := q.Exec(ctx, `UPDATE `+stage+`.parcel p SET
			area_m2 = public.calculate_area_utm(ST_MakeValid(p.geom)),
			boundary_idx = COALESCE(best.idx, near.idx),
			overlap_ratio = best.ratio,
			outside = best.idx IS NULL,
			nearest_distance_m = CASE WHEN best.idx IS NULL THEN near.distance END
		FROM `+stage+`.parcel p2
		LEFT JOIN LATERAL (
			SELECT b.idx, ST_Area(ST_Intersection(ST_MakeValid(b.geom), ST_MakeValid(p2.geom))::geography)
			         / NULLIF(ST_Area(ST_MakeValid(p2.geom)::geography), 0) AS ratio
			  FROM `+stage+`.boundary b
			 WHERE b.session_id = p2.session_id AND b.geom IS NOT NULL AND ST_Intersects(b.geom, p2.geom)
			   AND ST_Area(ST_Intersection(ST_MakeValid(b.geom), ST_MakeValid(p2.geom))) > 0
			 ORDER BY 2 DESC, b.idx LIMIT 1) best ON true
		LEFT JOIN LATERAL (
			SELECT b.idx, ST_Distance(ST_MakeValid(b.geom)::geography, ST_MakeValid(p2.geom)::geography) AS distance
			  FROM `+stage+`.boundary b
			 WHERE b.session_id = p2.session_id AND b.geom IS NOT NULL
			 ORDER BY 2, b.idx LIMIT 1) near ON best.idx IS NULL
		WHERE p.session_id = $1 AND p2.session_id = p.session_id AND p2.idx = p.idx AND p.geom IS NOT NULL`, sessionID); err != nil {
		return nil, fmt.Errorf("байршил тооцоолох: %w", err)
	}
	// ГУС дуудах шаардлагагүй: зохиомол, хоосон, давхардсан, САНД БҮРТГЭЛТЭЙ
	// (өөр чөлөөлөлтийн мэдээлэлд нөлөөлөхгүйн тулд импорт тэдгээрийг хөндөхгүй).
	if _, err := q.Exec(ctx, `UPDATE `+stage+`.parcel p SET gus_origin = 'skip'
		WHERE p.session_id = $1 AND (
			p.parcel_id = '' OR p.row->>'parcelID' = ''
			OR p.issues @> '[{"code":"duplicate"}]'
			OR EXISTS (SELECT 1 FROM parcel x WHERE x.parcel_id = p.parcel_id))`, sessionID); err != nil {
		return nil, err
	}
	for i := range infos {
		infos[i].Issues = nonNilIssues(infos[i].Issues)
	}
	return infos, nil
}

// lookupParcelsInGus — нэгж талбар бүрийг ГУС-аас хайж хариуг хадгална.
func (s *Service) lookupParcelsInGus(ctx context.Context, q querier, sessionID string, progress func(done, total int)) error {
	type item struct {
		idx           int
		parcelID, old string
	}
	rows, err := q.Query(ctx, `SELECT idx, parcel_id, COALESCE(row->>'oldParcelID','') FROM `+stage+`.parcel
		WHERE session_id=$1 AND gus_origin IS NULL ORDER BY idx`, sessionID)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (item, error) {
		var it item
		return it, r.Scan(&it.idx, &it.parcelID, &it.old)
	})
	if err != nil {
		return err
	}
	progress(0, len(items))
	for i, it := range items {
		origin := "missing"
		var source *SourceParcel
		hit, err := s.gus.ParcelByEitherID(ctx, it.parcelID, it.old)
		switch {
		case ctx.Err() != nil:
			return ctx.Err()
		case err != nil:
			origin = "failed"
		case hit != nil:
			origin, source = hit.Source.Origin, &hit.Source
		}
		if _, err := q.Exec(ctx, `UPDATE `+stage+`.parcel SET gus_origin=$3, source=$4 WHERE session_id=$1 AND idx=$2`,
			sessionID, it.idx, origin, source); err != nil {
			return err
		}
		progress(i+1, len(items))
	}
	return nil
}

func (s *Service) buildParcelView(ctx context.Context, q querier, sessionID string, files []FileInfo) (*ParcelView, error) {
	rows, err := q.Query(ctx, `SELECT p.idx, p.file_name, p.parcel_id, p.row, p.area_m2::float8, p.boundary_idx,
			p.overlap_ratio::float8, p.outside, p.nearest_distance_m::float8, p.gus_origin, p.issues,
			EXISTS (SELECT 1 FROM parcel x WHERE x.parcel_id = p.parcel_id AND p.parcel_id <> '')
		FROM `+stage+`.parcel p WHERE p.session_id=$1 ORDER BY p.idx`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	view := &ParcelView{Files: files, Rows: []ParcelViewRow{}, Map: []ParcelMap{},
		Stats: ParcelStats{UnmappedCodes: map[string]int{},
			Gus: map[string]int{"info": 0, "ub": 0, "missing": 0, "failed": 0, "skip": 0, "pending": 0}}}
	for rows.Next() {
		var r ParcelViewRow
		var row ParcelRow
		var ratio *float64
		var gus *string
		if err := rows.Scan(&r.Idx, &r.File, &r.ParcelID, &row, &r.AreaM2, &r.BoundaryIdx, &ratio,
			&r.Outside, &r.NearestDistanceM, &gus, &r.Issues, &r.AlreadyImported); err != nil {
			return nil, err
		}
		base := append([]Issue(nil), r.Issues...)
		m := mapStatusCode(row.SourceCode)
		r.OldParcelID, r.SourceCode, r.Status, r.StatusMapped = row.OldParcelID, row.SourceCode, m.status, m.mapped
		r.Synthetic = hasCode(base, "synthetic")
		r.Gus = "pending"
		if gus != nil {
			r.Gus = *gus
		}
		if ratio != nil {
			v := round1(*ratio * 100)
			r.InsidePct = &v
		}
		switch {
		case r.Outside:
			view.Stats.Outside++
			dist := 0.0
			if r.NearestDistanceM != nil {
				dist = *r.NearestDistanceM
			}
			r.Issues = append(r.Issues, warnIssue("outside", fmt.Sprintf(
				"%s: аль ч чөлөөлөлтийн хилтэй огтлолцохгүй — %s м зайтай хамгийн ойр чөлөөлөлтөд хамааруулна", r.ParcelID, fmtInt(dist))))
		case r.InsidePct != nil && *r.InsidePct < parcelInsideWarnPct:
			view.Stats.PartlyOutside++
			r.Issues = append(r.Issues, infoIssue("partly_outside", fmt.Sprintf("%s: %.1f%% нь л чөлөөлөлтийн хилийн дотор", r.ParcelID, *r.InsidePct)))
		}
		if r.AlreadyImported {
			view.Stats.AlreadyImported++
			r.Issues = append(r.Issues, warnIssue("already_imported",
				r.ParcelID+": санд өөр бүртгэлд байгаа нэгж талбар — бусад мэдээлэлд нөлөөлөхгүйн тулд энэ импортод ОРОХГҮЙ (хөндөхгүй)"))
		}
		switch r.Gus {
		case "ub":
			r.Issues = append(r.Issues, warnIssue("gus_ub", r.ParcelID+": /parcel/info-д олдсонгүй — data_ub-аас зөвхөн үндсэн мэдээлэл (эрхийн төрөл, гэрээ дутуу)"))
		case "missing":
			r.Issues = append(r.Issues, warnIssue("gus_missing", r.ParcelID+": ГУС-д олдсонгүй — зөвхөн SHP-ийн мэдээллээр орно (үндсэн хил, эрхийн төрөл дутуу)"))
		case "failed":
			r.Issues = append(r.Issues, warnIssue("gus_failed", r.ParcelID+": ГУС дуудлага амжилтгүй — «Оруулах» үед дахин оролдоно"))
		}
		if r.Synthetic {
			view.Stats.Synthetic++
		}
		if hasCode(base, "duplicate") {
			view.Stats.Duplicates++
		}
		if hasCode(base, "no_geometry") {
			view.Stats.NoGeometry++
		}
		if !m.mapped && row.SourceCode != nil {
			view.Stats.UnmappedCodes[fmt.Sprint(*row.SourceCode)]++
		}
		view.Stats.Gus[r.Gus]++
		r.Issues = nonNilIssues(r.Issues)
		view.Rows = append(view.Rows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	view.Stats.Total = len(view.Rows)

	mrows, err := q.Query(ctx, `SELECT idx, parcel_id, `+simplified("geom")+`, outside FROM `+stage+`.parcel
		WHERE session_id=$1 AND geom IS NOT NULL AND NOT (issues @> '[{"code":"duplicate"}]') ORDER BY idx LIMIT 5000`, sessionID)
	if err != nil {
		return nil, err
	}
	for mrows.Next() {
		var m ParcelMap
		if err := mrows.Scan(&m.Idx, &m.ParcelID, &m.WKT, &m.Outside); err != nil {
			mrows.Close()
			return nil, err
		}
		view.Map = append(view.Map, m)
	}
	mrows.Close()

	var all, fileIssues []Issue
	for _, f := range files {
		fileIssues = append(fileIssues, f.Issues...)
	}
	all = append(all, fileIssues...)
	for _, r := range view.Rows {
		all = append(all, r.Issues...)
	}
	view.Errors, view.Warnings = countLevel(all, "error"), countLevel(all, "warning")
	view.GusDone = view.Stats.Gus["pending"] == 0
	// Мөрийн алдаа (хоосон мөр) нь зөвхөн тэр мөрийг алгасна — файл уншигдсан л бол цааш.
	view.CanProceed = countLevel(fileIssues, "error") == 0 && len(view.Rows) > 0
	return view, nil
}

// ── 3. ҮНЭЛГЭЭНИЙ ФАЙЛ ─────────────────────────────────────────────────────

func (s *Service) stageReports(ctx context.Context, q querier, sessionID string, files []UploadedFile) ([]FileInfo, error) {
	if _, err := q.Exec(ctx, `DELETE FROM `+stage+`.report_row WHERE session_id=$1`, sessionID); err != nil {
		return nil, err
	}
	var infos []FileInfo
	var batches [][]ReportRow
	for _, f := range files {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xlsx") || strings.HasPrefix(f.Name, "~$") {
			continue
		}
		info := FileInfo{Name: f.Name, Issues: []Issue{}}
		rows, notes, err := readReport(f.Name, f.Data)
		for _, n := range notes {
			info.Issues = append(info.Issues, warnIssue("sheet_skipped", n))
		}
		switch {
		case err != nil:
			info.Issues = append(info.Issues, errIssue("unreadable", f.Name+": "+err.Error()))
		case len(rows) == 0:
			info.Issues = append(info.Issues, errIssue("empty", f.Name+": нэгж талбарын мөр олдсонгүй"))
		}
		info.Features = len(rows)
		batches = append(batches, rows)
		infos = append(infos, info)
	}
	if len(infos) == 0 {
		return []FileInfo{{Issues: []Issue{errIssue("no_xlsx", "Үнэлгээний .xlsx файл сонгогдоогүй байна")}}}, nil
	}
	merged, dups := mergeReports(batches)
	for i := range merged {
		row := &merged[i]
		var issues []Issue
		if src, ok := dups[row.ParcelID]; ok && row.ParcelID != "" {
			issues = append(issues, warnIssue("duplicate", fmt.Sprintf("%s: %d газар давхардсан (%s) — нэг мөр болгон нэгтгэв", row.ParcelID, len(src), strings.Join(src, "; "))))
		}
		if b := balanceAmounts(row); b.adjustment != 0 {
			kind := "задаргаа дутуу"
			if b.breakdownMissing {
				kind = "задаргаагүй"
			}
			issues = append(issues, warnIssue("balanced", fmt.Sprintf("%s: %s — газрын үнэд %s₮ нэмж «Нийт нөхөх олговор»-той тэнцүүлнэ", row.ParcelID, kind, fmtInt(b.adjustment))))
		}
		if row.ParcelID == "" {
			issues = append(issues, warnIssue("no_parcel_id", fmt.Sprintf("%s мөр %d: нэгж талбарын дугааргүй", row.SourceSheet, row.ExcelRow)))
		}
		if _, err := q.Exec(ctx, `INSERT INTO `+stage+`.report_row (session_id, idx, file_name, parcel_id, row, issues) VALUES ($1,$2,$3,$4,$5,$6)`,
			sessionID, i+1, row.SourceFile, row.ParcelID, row, nonNilIssues(issues)); err != nil {
			return nil, fmt.Errorf("үнэлгээний мөр хадгалах: %w", err)
		}
	}
	// SHP-ийн нэгж талбар (шинэ/хуучин дугаараар), дараа нь сан.
	for _, sql := range []string{
		`UPDATE ` + stage + `.report_row r SET match = 'staged'
		  WHERE r.session_id = $1 AND r.parcel_id <> '' AND EXISTS (
		    SELECT 1 FROM ` + stage + `.parcel p WHERE p.session_id = r.session_id
		       AND (p.parcel_id = r.parcel_id OR (NULLIF(r.row->>'oldParcelID','') IS NOT NULL
		            AND p.row->>'oldParcelID' = r.row->>'oldParcelID')))`,
		`UPDATE ` + stage + `.report_row r SET match = 'db'
		  WHERE r.session_id = $1 AND r.match IS NULL AND r.parcel_id <> '' AND EXISTS (
		    SELECT 1 FROM parcel x JOIN land_acquisition la ON la.id = x.acquisition_id
		     WHERE x.parcel_id = r.parcel_id AND x.deleted_at IS NULL AND la.deleted_at IS NULL)`,
		`UPDATE ` + stage + `.report_row SET match = 'missing' WHERE session_id = $1 AND match IS NULL AND parcel_id = ''`,
	} {
		if _, err := q.Exec(ctx, sql, sessionID); err != nil {
			return nil, err
		}
	}
	return infos, nil
}

// lookupReportsInGus — SHP-д ч, санд ч байхгүй мөрүүдийг ГУС-аас хайна.
func (s *Service) lookupReportsInGus(ctx context.Context, q querier, sessionID string, progress func(done, total int)) error {
	type item struct {
		idx           int
		parcelID, old string
	}
	rows, err := q.Query(ctx, `SELECT idx, parcel_id, COALESCE(row->>'oldParcelID','') FROM `+stage+`.report_row
		WHERE session_id=$1 AND match IS NULL ORDER BY idx`, sessionID)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (item, error) {
		var it item
		return it, r.Scan(&it.idx, &it.parcelID, &it.old)
	})
	if err != nil {
		return err
	}
	progress(0, len(items))
	for i, it := range items {
		match := "missing"
		var source *SourceParcel
		hit, err := s.gus.ParcelByEitherID(ctx, it.parcelID, it.old)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Үндсэн хилтэй хариуг л ашиглана — чөлөөлөлтөд байршлаар хуваарилна.
		if err == nil && hit != nil && hit.Source.GeometryWKT != "" {
			match, source = "gus:"+hit.Source.Origin, &hit.Source
		}
		if _, err := q.Exec(ctx, `UPDATE `+stage+`.report_row SET match=$3, source=$4 WHERE session_id=$1 AND idx=$2`,
			sessionID, it.idx, match, source); err != nil {
			return err
		}
		progress(i+1, len(items))
	}
	return nil
}

func (s *Service) buildReportView(ctx context.Context, q querier, sessionID string, files []FileInfo) (*ReportView, error) {
	rows, err := q.Query(ctx, `SELECT r.idx, r.file_name, r.parcel_id, r.row, r.match, r.issues,
			(SELECT p.area_m2::float8 FROM `+stage+`.parcel p
			  WHERE p.session_id = r.session_id AND p.parcel_id = r.parcel_id LIMIT 1)
		FROM `+stage+`.report_row r WHERE r.session_id=$1 ORDER BY r.idx`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	view := &ReportView{Files: files, Rows: []ReportViewRow{}, ParcelsWithoutReport: []ParcelRef{}}
	st := &view.Stats
	for rows.Next() {
		var v ReportViewRow
		var row ReportRow
		var match *string
		if err := rows.Scan(&v.Idx, &v.File, &v.ParcelID, &row, &match, &v.Issues, &v.ShpAreaM2); err != nil {
			return nil, err
		}
		kind, origin, _ := strings.Cut(derefOr(match, "pending"), ":")
		amount := row.amount()
		st.TotalAmount += amount
		switch kind {
		case "staged":
			st.Staged++
		case "db":
			st.DB++
			v.Issues = append(v.Issues, warnIssue("db_parcel",
				v.ParcelID+": санд өөр бүртгэлд байгаа нэгж талбар — энэ мөрийн үнэлгээ ОРОХГҮЙ (бусад мэдээлэлд нөлөөлөхгүй)"))
		case "gus":
			st.Gus++
			o := origin
			v.GusOrigin = &o
			if origin == "ub" {
				st.GusFromUb++
				v.Issues = append(v.Issues, warnIssue("gus_ub", v.ParcelID+": нэгж талбарын SHP-д байхгүй — data_ub-аас үндсэн мэдээллээр шинээр бүртгэгдэнэ"))
			} else {
				v.Issues = append(v.Issues, infoIssue("gus_info", v.ParcelID+": нэгж талбарын SHP-д байхгүй — ГУС-аас татаж шинээр бүртгэнэ"))
			}
		case "missing":
			st.Missing++
			st.MissingAmount += amount
			label := v.ParcelID
			if label == "" {
				label = fmt.Sprintf("мөр %d", row.ExcelRow)
			}
			v.Issues = append(v.Issues, warnIssue("not_found", fmt.Sprintf(
				"%s: SHP, сан, ГУС аль алинд нь олдсонгүй — энэ мөрийн үнэлгээ ОРОХГҮЙ (%s₮)", label, fmtInt(amount))))
		default:
			st.Pending++
		}
		if row.AffectedAreaM2 != nil && v.ShpAreaM2 != nil && *v.ShpAreaM2 > 0 {
			diff := math.Abs(*row.AffectedAreaM2 - *v.ShpAreaM2)
			if pct := diff * 100 / *v.ShpAreaM2; pct > areaDiffWarnPct && diff > 1 {
				st.AreaDiffs++
				v.Issues = append(v.Issues, warnIssue("area_diff", fmt.Sprintf("%s: эксэлийн нөлөөлсөн талбай %s м², SHP-ийнх %s м² (%.0f%% зөрүү)",
					v.ParcelID, fmtInt(*row.AffectedAreaM2), fmtInt(*v.ShpAreaM2), pct)))
			}
		}
		if hasCode(v.Issues, "balanced") {
			st.Balanced++
		}
		if hasCode(v.Issues, "duplicate") {
			st.Duplicates++
		}
		v.Match, v.Sheet, v.ExcelRow = kind, row.SourceSheet, row.ExcelRow
		v.OldParcelID, v.HolderName, v.TotalAmount, v.AffectedAreaM2 = row.OldParcelID, row.HolderName, amount, row.AffectedAreaM2
		v.Issues = nonNilIssues(v.Issues)
		view.Rows = append(view.Rows, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	st.Rows = len(view.Rows)

	wrows, err := q.Query(ctx, `SELECT p.parcel_id, p.file_name FROM `+stage+`.parcel p
		WHERE p.session_id = $1 AND p.parcel_id <> '' AND NOT (p.issues @> '[{"code":"duplicate"}]')
		  AND NOT EXISTS (SELECT 1 FROM `+stage+`.report_row r WHERE r.session_id = p.session_id
		                   AND (r.parcel_id = p.parcel_id OR (NULLIF(p.row->>'oldParcelID','') IS NOT NULL
		                        AND r.row->>'oldParcelID' = p.row->>'oldParcelID')))
		ORDER BY p.idx`, sessionID)
	if err != nil {
		return nil, err
	}
	for wrows.Next() {
		var ref ParcelRef
		if err := wrows.Scan(&ref.ParcelID, &ref.File); err != nil {
			wrows.Close()
			return nil, err
		}
		view.ParcelsWithoutReport = append(view.ParcelsWithoutReport, ref)
	}
	wrows.Close()
	st.WithoutReport = len(view.ParcelsWithoutReport)

	var fileIssues, all []Issue
	for _, f := range files {
		fileIssues = append(fileIssues, f.Issues...)
	}
	all = append(all, fileIssues...)
	for _, r := range view.Rows {
		all = append(all, r.Issues...)
	}
	view.Errors = countLevel(all, "error")
	view.Warnings = countLevel(all, "warning")
	if st.WithoutReport > 0 {
		view.Warnings++
	}
	view.GusDone = st.Pending == 0
	view.CanProceed = countLevel(fileIssues, "error") == 0 && len(view.Rows) > 0
	return view, nil
}

func derefOr(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// resolveCategories — сонгосон ангилал acquisition_category-д байгаа эсэх.
func resolveCategories(ctx context.Context, q querier, choice CategoryChoice) (*Categories, error) {
	var general CategoryRef
	err := q.QueryRow(ctx, `SELECT id, name FROM acquisition_category WHERE id::text = $1 AND parent_id IS NULL`,
		strings.TrimSpace(choice.General)).Scan(&general.ID, &general.Name)
	if err != nil {
		return nil, fmt.Errorf("ерөнхий ангилал сонгоно уу (id=%q олдсонгүй)", choice.General)
	}
	out := &Categories{General: general}
	if sub := strings.TrimSpace(choice.Sub); sub != "" {
		var ref CategoryRef
		if err := q.QueryRow(ctx, `SELECT id, name FROM acquisition_category WHERE id::text = $1 AND parent_id = $2`,
			sub, general.ID).Scan(&ref.ID, &ref.Name); err != nil {
			return nil, fmt.Errorf("«%s»-ийн дэд ангилал id=%s олдсонгүй", general.Name, sub)
		}
		out.Sub = &ref
	}
	return out, nil
}

// jsonOf — шалгалтын үр дүнг session-ийн jsonb багана руу.
func jsonOf(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
