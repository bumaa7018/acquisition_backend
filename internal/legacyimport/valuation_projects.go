package legacyimport

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ── «ЗӨВХӨН ҮНЭЛГЭЭ»: ТӨСӨЛ БҮРИЙН ТОХИРГОО ────────────────────────────────
//
// Эксэлээс үүсэх чөлөөлөлт бүрд (бүгд заавал биш):
//
//	ангилал  — гараар сонгоно; эс бөгөөс «Бүтээн байгуулалтын төрөл»-өөс
//	мэдээлэл — хэрэгжүүлэгч/хариуцах байгууллага, огноо, НИТХ тогтоол г.м.
//	хил      — shapefile (нэг полигон); эс бөгөөс чөлөөлөлт ХИЛГҮЙ
//
// Тохиргоо session.projects-д төслийн НЭРЭЭР хадгалагдана — эксэлийг дахин
// оруулахад нэр нь ижил төсөлд хэвээр үлдэнэ.

// ProjectUpdate — PUT /projects. nil талбар өөрчлөгдөхгүй; Category.General
// хоосон бол автомат (бүтээн байгуулалтын төрлөөс) руу буцна.
type ProjectUpdate struct {
	Name     string           `json:"name"`
	Category *CategoryChoice  `json:"category"`
	Info     *AcquisitionInfo `json:"info"`
}

// land_acquisition-ий varchar(100) баганууд.
const infoNumberLimit = 100

// requireValuationProject — ноорог «Зөвхөн үнэлгээ» импорт ба түүний төсөл.
// ГУС-ын шалгалт явж байхад ч тохируулж болно (зөвхөн «Оруулах» үед хаана).
func (s *Service) requireValuationProject(ctx context.Context, id, name string) (*session, error) {
	ss, err := s.load(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	switch ss.Status {
	case "committed", "rolled_back":
		return nil, ErrCommitted
	case "committing":
		return nil, ErrBusy
	}
	if ss.Mode != ModeValuation {
		return nil, &StepError{"Төслийн тохиргоо зөвхөн «Зөвхөн үнэлгээ» импортод хийгдэнэ"}
	}
	if ss.reports == nil {
		return nil, &StepError{"Эхлээд үнэлгээний эксэлээ оруулна уу"}
	}
	for _, p := range ss.reports.Projects {
		if p.Name == name {
			return ss, nil
		}
	}
	return nil, &StepError{fmt.Sprintf("«%s» төсөл олдсонгүй — эксэлийг дахин шалгасан бол хуудсаа шинэчилнэ үү", name)}
}

// UpdateProject — төслийн ангилал ба/эсвэл чөлөөлөлтийн мэдээлэл.
func (s *Service) UpdateProject(ctx context.Context, id string, u ProjectUpdate) (*State, error) {
	ss, err := s.requireValuationProject(ctx, id, u.Name)
	if err != nil {
		return nil, err
	}
	set := ss.projects[u.Name]
	if u.Category != nil {
		if strings.TrimSpace(u.Category.General) == "" {
			set.Category = nil
		} else {
			if _, err := resolveCategories(ctx, s.db, *u.Category); err != nil {
				return nil, &StepError{err.Error()}
			}
			choice := CategoryChoice{General: strings.TrimSpace(u.Category.General), Sub: strings.TrimSpace(u.Category.Sub)}
			set.Category = &choice
		}
	}
	if u.Info != nil {
		info, err := normalizeInfo(*u.Info)
		if err != nil {
			return nil, err
		}
		set.Info = info
	}
	return s.saveProject(ctx, ss, u.Name, set)
}

// UploadProjectBoundary — төслийн хилийн shapefile (алдаатай бол хадгалахгүй).
func (s *Service) UploadProjectBoundary(ctx context.Context, id, name string, files []UploadedFile) (*State, error) {
	ss, err := s.requireValuationProject(ctx, id, name)
	if err != nil {
		return nil, err
	}
	boundary, err := s.analyzeProjectBoundary(ctx, id, name, files)
	if err != nil {
		return nil, err
	}
	set := ss.projects[name]
	set.Boundary = boundary
	return s.saveProject(ctx, ss, name, set)
}

// RemoveProjectBoundary — хилийг хасна (чөлөөлөлт хилгүй үүснэ).
func (s *Service) RemoveProjectBoundary(ctx context.Context, id, name string) (*State, error) {
	ss, err := s.requireValuationProject(ctx, id, name)
	if err != nil {
		return nil, err
	}
	set := ss.projects[name]
	set.Boundary = nil
	return s.saveProject(ctx, ss, name, set)
}

// saveProject — НЭГ төслийн түлхүүрийг солино (бусад төслийн зэрэгцээ
// засварыг дарахгүй), дараа нь төслийн жагсаалтыг (ангилал, талбай) дахин
// бодно. ГУС-ын шалгалт явж байвал тэр ажил дуусахдаа ч дахин бодно.
func (s *Service) saveProject(ctx context.Context, ss *session, name string, set ProjectSettings) (*State, error) {
	// Нөхцөлтэй — шалгалт ба бичилтийн хооронд «Оруулах» эхэлсэн бол
	// (status = committing) оруулж буй импортын тохиргоог өөрчлөхгүй.
	tag, err := s.db.Exec(ctx, `UPDATE `+stage+`.session
		SET projects = jsonb_set(COALESCE(projects, '{}'), ARRAY[$2::text], $3::jsonb), updated_at = now()
		WHERE id = $1 AND status IN ('draft', 'failed')`, ss.ID, name, jsonOf(set))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrBusy
	}
	view, err := s.buildValuationView(ctx, s.db, ss.ID, ss.reports.Files)
	if err != nil {
		return nil, err
	}
	if err := s.save(ctx, s.db, ss.ID, "reports", view); err != nil {
		return nil, err
	}
	return s.Get(ctx, ss.ID)
}

// projectSettings — buildValuationView ба «Оруулах»-д (гүйлгээн дотроос ч).
func projectSettings(ctx context.Context, q querier, sessionID string) (map[string]ProjectSettings, error) {
	out := map[string]ProjectSettings{}
	err := q.QueryRow(ctx, `SELECT COALESCE(projects, '{}') FROM `+stage+`.session WHERE id = $1`, sessionID).Scan(&out)
	return out, err
}

func normalizeInfo(in AcquisitionInfo) (AcquisitionInfo, error) {
	out := AcquisitionInfo{
		ImplementingOrg:  strings.TrimSpace(in.ImplementingOrg),
		ResponsibleOrg:   strings.TrimSpace(in.ResponsibleOrg),
		Reason:           strings.TrimSpace(in.Reason),
		StartDate:        strings.TrimSpace(in.StartDate),
		EndDate:          strings.TrimSpace(in.EndDate),
		NithDecreeNumber: strings.TrimSpace(in.NithDecreeNumber),
		GroupListNumber:  strings.TrimSpace(in.GroupListNumber),
	}
	for label, v := range map[string]string{"НИТХ тогтоолын дугаар": out.NithDecreeNumber, "Бүлэг жагсаалтын дугаар": out.GroupListNumber} {
		if len([]rune(v)) > infoNumberLimit {
			return out, &StepError{fmt.Sprintf("%s %d тэмдэгтээс урт байна", label, infoNumberLimit)}
		}
	}
	var start, end time.Time
	for _, d := range []struct {
		label string
		value string
		out   *time.Time
	}{{"Эхлэх огноо", out.StartDate, &start}, {"Дуусах огноо", out.EndDate, &end}} {
		if d.value == "" {
			continue
		}
		t, err := time.Parse("2006-01-02", d.value)
		if err != nil {
			return out, &StepError{fmt.Sprintf("%s буруу (%s) — ОООО-СС-ӨӨ хэлбэрээр", d.label, d.value)}
		}
		*d.out = t
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return out, &StepError{"Дуусах огноо эхлэх огнооноос өмнө байна"}
	}
	return out, nil
}

func (i AcquisitionInfo) empty() bool { return i == AcquisitionInfo{} }

// analyzeProjectBoundary — нэг shapefile-ийн бүх полигоныг EPSG:4326 руу
// хөрвүүлж НЭГТГЭНЭ. land_acquisition.geometry нь Polygon тул нэгтгэсэн хил
// НЭГ полигон байх ёстой. Төслийн нэгж талбарууд (ГУС-ын хил) хилтэй
// огтлолцохгүй бол анхааруулна — өөр төслийн файл байж болзошгүй.
func (s *Service) analyzeProjectBoundary(ctx context.Context, sessionID, name string, files []UploadedFile) (*ProjectBoundary, error) {
	infos, sets, srids := readShapefiles(files, nil)
	switch {
	case len(infos) == 0:
		return nil, &StepError{"Хилийн .shp файл сонгогдоогүй байна"}
	case len(infos) > 1:
		return nil, &StepError{fmt.Sprintf("Нэг төсөлд НЭГ shapefile оруулна уу (%d ирсэн)", len(infos))}
	}
	info := infos[0]
	for _, i := range info.Issues {
		if i.Level == "error" {
			return nil, &StepError{i.Message}
		}
	}
	var wkts []string
	for _, f := range sets[0] {
		if f.wkt != "" {
			wkts = append(wkts, f.wkt)
		}
	}
	if len(wkts) == 0 {
		return nil, &StepError{info.Name + ": хил (геометр) байхгүй"}
	}
	var (
		parts     int
		valid, mn bool
		wkt       string
		area      float64
	)
	err := s.db.QueryRow(ctx, `WITH src AS (
			SELECT ST_Transform(ST_SetSRID(ST_GeomFromText(w), $2::int), 4326) AS g FROM unnest($1::text[]) w
		), u AS (
			SELECT ST_Union(ST_CollectionExtract(ST_MakeValid(g), 3)) AS geom, bool_and(ST_IsValid(g)) AS valid FROM src
		)
		SELECT COALESCE(ST_NumGeometries(geom), 0), valid,
			COALESCE(ST_Intersects(geom, ST_MakeEnvelope(87, 41, 120, 52.5, 4326)), false),
			CASE WHEN ST_NumGeometries(geom) = 1 THEN ST_AsText(ST_GeometryN(geom, 1)) ELSE '' END,
			CASE WHEN ST_NumGeometries(geom) = 1 THEN public.calculate_area_utm(ST_GeometryN(geom, 1))::float8 ELSE 0 END
		FROM u`, wkts, srids[0]).Scan(&parts, &valid, &mn, &wkt, &area)
	if err != nil {
		return nil, &StepError{fmt.Sprintf("%s: геометрийг уншиж чадсангүй — %s", info.Name, err.Error())}
	}
	switch {
	case parts == 0:
		return nil, &StepError{info.Name + ": полигон хил олдсонгүй"}
	case parts > 1:
		return nil, &StepError{fmt.Sprintf("%s: хил %d тусдаа хэсэгтэй — чөлөөлөлтийн хил НЭГ полигон байх ёстой (хэсгүүдийг GIS-д нэгтгэнэ үү)", info.Name, parts)}
	case !mn:
		return nil, &StepError{info.Name + ": координат Монголын хүрээнээс гадна — .prj (проекц) зөв эсэхийг шалгана уу"}
	}
	issues := []Issue{}
	for _, i := range info.Issues {
		if i.Level == "warning" {
			issues = append(issues, i)
		}
	}
	if !valid {
		issues = append(issues, warnIssue("invalid_geometry", info.Name+": геометр алдаатай байсан — автоматаар засав (ST_MakeValid)"))
	}
	if len(wkts) > 1 {
		issues = append(issues, infoIssue("merged", fmt.Sprintf("%s: %d полигоныг нэг хил болгон нэгтгэв", info.Name, len(wkts))))
	}
	var outside, total int
	if err := s.db.QueryRow(ctx, `SELECT
			count(*) FILTER (WHERE NOT ST_Intersects(ST_MakeValid(ST_GeomFromText(source->>'geometryWKT', 4326)), ST_GeomFromText($3, 4326)))::int,
			count(*)::int
		FROM `+stage+`.report_row
		WHERE session_id = $1 AND project = $2 AND NULLIF(source->>'geometryWKT', '') IS NOT NULL`,
		sessionID, name, wkt).Scan(&outside, &total); err != nil {
		return nil, err
	}
	switch {
	case total > 0 && outside == total:
		issues = append(issues, warnIssue("parcels_outside", fmt.Sprintf("Төслийн %d нэгж талбарын НЭГ Ч хилтэй огтлолцохгүй — өөр төслийн хил эсвэл проекц буруу байж болзошгүй", total)))
	case outside > 0:
		issues = append(issues, warnIssue("parcels_outside", fmt.Sprintf("Төслийн %d нэгж талбараас %d нь хилийн гадна", total, outside)))
	}
	return &ProjectBoundary{File: info.Name, WKT: wkt, AreaM2: round1(area), Issues: issues}, nil
}
