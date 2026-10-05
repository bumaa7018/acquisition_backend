package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ── «ЗӨВХӨН ҮНЭЛГЭЭ» ГОРИМ: шалгалт (түр хүснэгтэд) ─────────────────────────
//
// Эксэлийн толгой нь бүтэн горимын үнэлгээний файлтай ИЖИЛ. Мөр бүр ТУСДАА
// (бүтэн горимынх шиг нэгтгэхгүй) — нэг нэгж талбар өөр он/төсөлд дахин
// тохиолдвол архивын адил D<n>… дугаараар бүртгэнэ. Мөрийн шийдвэр (match):
//
//	gus:info | gus:ub — ГУС-аас (эсвэл data_ub-аас) олдсон
//	excel     — ГУС-д олдоогүй, эксэлийн мэдээллээр (хилгүй) бүртгэнэ
//	failed    — ГУС дуудлага амжилтгүй, эксэлийн мэдээллээр бүртгэнэ
//	synthetic — дугааргүй, ERR… дугаараар бүртгэнэ
//	db        — санд ӨӨР бүртгэлд байгаа — ОРОХГҮЙ (бусад мэдээлэлд нөлөөлөхгүй)
//	no_project — «Бүтээн байгуулалтын ажлын нэр» хоосон — ОРОХГҮЙ

const (
	matchNoProject = "no_project"
	matchSynthetic = "synthetic"
	matchExcel     = "excel"
	matchFailed    = "failed"
	matchDB        = "db"
)

// importable — «Оруулах» үед бүртгэгдэх мөр эсэх.
func valuationImportable(match string) bool {
	return strings.HasPrefix(match, "gus:") || match == matchExcel || match == matchFailed || match == matchSynthetic
}

func (s *Service) stageValuationReports(ctx context.Context, q querier, sessionID string, files []UploadedFile) ([]FileInfo, error) {
	if _, err := q.Exec(ctx, `DELETE FROM `+stage+`.report_row WHERE session_id=$1`, sessionID); err != nil {
		return nil, err
	}
	var infos []FileInfo
	var all []ReportRow
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
		all = append(all, rows...)
		infos = append(infos, info)
	}
	if len(infos) == 0 {
		return []FileInfo{{Issues: []Issue{errIssue("no_xlsx", "Үнэлгээний .xlsx файл сонгогдоогүй байна")}}}, nil
	}

	// Төслийн нэрийг нэгтгэнэ (архивын дүрэм).
	counts := map[string]int{}
	for _, r := range all {
		if name := strings.TrimSpace(r.ProjectName); name != "" {
			counts[name]++
		}
	}
	resolve, _ := clusterNames(counts)

	for i := range all {
		row := &all[i]
		var issues []Issue
		if b := balanceAmounts(row); b.adjustment != 0 {
			kind := "задаргаа дутуу"
			if b.breakdownMissing {
				kind = "задаргаагүй"
			}
			issues = append(issues, warnIssue("balanced", fmt.Sprintf("%s: %s — газрын үнэд %s₮ нэмж «Нийт нөхөх олговор»-той тэнцүүлнэ", rowLabel(row), kind, fmtInt(b.adjustment))))
		}
		project := resolve[strings.TrimSpace(row.ProjectName)]
		var match, assigned, kind *string
		set := func(m, a, k string) { match, assigned, kind = &m, &a, &k }
		switch {
		case project == "":
			set(matchNoProject, "", "")
		case row.ParcelID == "" && row.OldParcelID == "":
			// Дугааргүй — ГУС дуудахгүй, мөрийн тогтвортой түлхүүрээс ERR… дугаар.
			id := syntheticParcelID(valuationRowKey(row))
			taken, err := parcelExists(ctx, q, id)
			if err != nil {
				return nil, err
			}
			if taken {
				set(matchDB, id, "synthetic")
			} else {
				set(matchSynthetic, id, "synthetic")
			}
		}
		if _, err := q.Exec(ctx, `INSERT INTO `+stage+`.report_row
				(session_id, idx, file_name, parcel_id, row, issues, project, match, assigned_id, id_kind)
			VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10)`,
			sessionID, i+1, row.SourceFile, row.ParcelID, row, nonNilIssues(issues), project, match, assigned, kind); err != nil {
			return nil, fmt.Errorf("үнэлгээний мөр хадгалах: %w", err)
		}
	}
	// Санд ӨӨР бүртгэлд байгаа дугаар — ГУС-аас ч хайхгүй, ОРОХГҮЙ.
	if _, err := q.Exec(ctx, `UPDATE `+stage+`.report_row r SET match = 'db', assigned_id = r.parcel_id, id_kind = 'real'
		WHERE r.session_id = $1 AND r.match IS NULL AND r.parcel_id <> ''
		  AND EXISTS (SELECT 1 FROM parcel x WHERE x.parcel_id = r.parcel_id)`, sessionID); err != nil {
		return nil, err
	}
	return infos, nil
}

func rowLabel(r *ReportRow) string {
	if r.ParcelID != "" {
		return r.ParcelID
	}
	if r.OldParcelID != "" {
		return r.OldParcelID
	}
	return fmt.Sprintf("%s мөр %d", r.SourceSheet, r.ExcelRow)
}

// lookupValuationInGus — дугаартай мөрийг ГУС-аас (шинэ → хуучин, info → ub)
// хайж, системд бүртгэх дугаарыг олгоно. Мөрийн ДАРААЛЛААР: эхний бүртгэл
// бодит дугаараа, дараагийнх нь D<n>… авна.
func (s *Service) lookupValuationInGus(ctx context.Context, q querier, sessionID string, progress func(done, total int)) error {
	type item struct {
		idx           int
		parcelID, old string
		row           ReportRow
	}
	rows, err := q.Query(ctx, `SELECT idx, parcel_id, COALESCE(row->>'oldParcelID',''), row FROM `+stage+`.report_row
		WHERE session_id=$1 AND match IS NULL ORDER BY idx`, sessionID)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (item, error) {
		var it item
		return it, r.Scan(&it.idx, &it.parcelID, &it.old, &it.row)
	})
	if err != nil {
		return err
	}
	occurrences := map[string]int{}
	progress(0, len(items))
	for i, it := range items {
		match := matchExcel
		var source *SourceParcel
		hit, err := s.gus.ParcelByEitherID(ctx, it.parcelID, it.old)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		switch {
		case err != nil:
			match = matchFailed
		case hit != nil:
			match, source = "gus:"+hit.Source.Origin, &hit.Source
		}
		// ГУС-ын ОДООГИЙН дугаар (хуучнаар хайсан үед чухал; varchar(12)-т багтах ёстой).
		resolved := it.parcelID
		if source != nil && source.ParcelID != "" && len(source.ParcelID) <= parcelIDLimit {
			resolved = source.ParcelID
		}
		assigned, kind := resolved, "real"
		switch {
		case resolved == "" || len(resolved) > parcelIDLimit:
			assigned, kind = syntheticParcelID(valuationRowKey(&it.row)), "synthetic"
		default:
			taken, err := parcelExists(ctx, q, resolved)
			if err != nil {
				return err
			}
			if taken && occurrences[resolved] == 0 {
				// ГУС-ын дугаараар санд ӨӨР бүртгэлд байна — ОРОХГҮЙ.
				match = matchDB
				break
			}
			occurrences[resolved]++
			if n := occurrences[resolved]; n > 1 {
				assigned, kind = duplicateParcelID(resolved, n, valuationRowKey(&it.row)), "duplicate"
			}
		}
		if match != matchDB && kind != "real" {
			if taken, err := parcelExists(ctx, q, assigned); err != nil {
				return err
			} else if taken {
				match = matchDB // өмнөх импортоор орсон ижил мөр
			}
		}
		if _, err := q.Exec(ctx, `UPDATE `+stage+`.report_row SET match=$3, source=$4, assigned_id=$5, id_kind=$6
			WHERE session_id=$1 AND idx=$2`, sessionID, it.idx, match, source, assigned, kind); err != nil {
			return err
		}
		progress(i+1, len(items))
	}
	return nil
}

// valuationProjects — төсөл бүрийн ангилал (зонхилох төрлөөр) ба тоо.
func (s *Service) buildValuationView(ctx context.Context, q querier, sessionID string, files []FileInfo) (*ReportView, error) {
	rows, err := q.Query(ctx, `SELECT r.idx, r.file_name, r.parcel_id, r.row, r.match, r.issues,
			COALESCE(r.project, ''), COALESCE(r.assigned_id, ''), COALESCE(r.id_kind, '')
		FROM `+stage+`.report_row r WHERE r.session_id=$1 ORDER BY r.idx`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	view := &ReportView{Files: files, Rows: []ReportViewRow{}, ParcelsWithoutReport: []ParcelRef{}, Projects: []ValuationProject{}}
	st := &view.Stats
	type agg struct {
		p     *ValuationProject
		types map[string]int
	}
	projects := map[string]*agg{}
	var order []string
	variants := map[string]map[string]bool{}
	for rows.Next() {
		var v ReportViewRow
		var row ReportRow
		var match *string
		if err := rows.Scan(&v.Idx, &v.File, &v.ParcelID, &row, &match, &v.Issues, &v.Project, &v.AssignedID, &v.IDKind); err != nil {
			return nil, err
		}
		m := derefOr(match, "pending")
		kind, origin, _ := strings.Cut(m, ":")
		amount := row.amount()
		st.TotalAmount += amount
		label := rowLabel(&row)
		switch kind {
		case "gus":
			st.Gus++
			o := origin
			v.GusOrigin = &o
			if origin == "ub" {
				st.GusFromUb++
				v.Issues = append(v.Issues, warnIssue("gus_ub", label+": ГУС-д олдоогүй — data_ub-аас үндсэн мэдээллээр бүртгэнэ"))
			}
		case matchExcel, matchFailed:
			st.Excel++
			msg := label + ": ГУС-д олдсонгүй — эксэлийн мэдээллээр (үндсэн хилгүй) бүртгэнэ"
			if kind == matchFailed {
				msg = label + ": ГУС дуудлага амжилтгүй — эксэлийн мэдээллээр бүртгэнэ (дахин шалгаж болно)"
			}
			v.Issues = append(v.Issues, warnIssue("gus_missing", msg))
		case matchSynthetic:
			v.Issues = append(v.Issues, warnIssue("synthetic", fmt.Sprintf("%s мөр %d: нэгж талбарын дугааргүй — %s дугаараар бүртгэнэ", row.SourceSheet, row.ExcelRow, v.AssignedID)))
		case matchDB:
			st.DB++
			st.MissingAmount += amount
			v.Issues = append(v.Issues, warnIssue("db_parcel", label+": санд өөр бүртгэлд байгаа нэгж талбар — энэ мөрийн үнэлгээ ОРОХГҮЙ (бусад мэдээлэлд нөлөөлөхгүй)"))
		case matchNoProject:
			st.Missing++
			st.MissingAmount += amount
			v.Issues = append(v.Issues, warnIssue("no_project", fmt.Sprintf("%s: «Бүтээн байгуулалтын ажлын нэр» хоосон — чөлөөлөлт тодорхойгүй тул ОРОХГҮЙ (%s₮)", label, fmtInt(amount))))
		default:
			st.Pending++
		}
		if v.IDKind == "synthetic" && kind != matchSynthetic && kind != matchDB {
			v.Issues = append(v.Issues, warnIssue("synthetic", fmt.Sprintf("%s: бодит дугаар тогтоогдсонгүй — %s дугаараар бүртгэнэ", label, v.AssignedID)))
		}
		if v.IDKind == "synthetic" && kind != matchDB {
			st.Synthetic++
		}
		if v.IDKind == "duplicate" && kind != matchDB {
			st.Duplicates++
			v.Issues = append(v.Issues, warnIssue("duplicate", fmt.Sprintf("%s: файлд дахин тохиолдсон (өөр он/төсөл) — %s дугаараар тусад нь бүртгэнэ", label, v.AssignedID)))
		}
		if hasCode(v.Issues, "balanced") {
			st.Balanced++
		}
		if v.Project != "" {
			a := projects[v.Project]
			if a == nil {
				a = &agg{p: &ValuationProject{Name: v.Project, Variants: []string{}, Issues: []Issue{}}, types: map[string]int{}}
				projects[v.Project] = a
				order = append(order, v.Project)
				variants[v.Project] = map[string]bool{}
			}
			a.p.Rows++
			a.p.Amount += amount
			if valuationImportable(m) {
				a.p.Parcels++
			}
			if t := strings.TrimSpace(row.ConstructionType); t != "" {
				a.types[t]++
			}
			if name := strings.TrimSpace(row.ProjectName); name != "" && !variants[v.Project][name] {
				variants[v.Project][name] = true
				a.p.Variants = append(a.p.Variants, name)
			}
		}
		v.Match, v.Sheet, v.ExcelRow = kind, row.SourceSheet, row.ExcelRow
		v.OldParcelID, v.HolderName, v.TotalAmount, v.AffectedAreaM2 = row.OldParcelID, row.HolderName, amount, row.AffectedAreaM2
		v.Issues = nonNilIssues(v.Issues)
		view.Rows = append(view.Rows, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	st.Rows = len(view.Rows)

	settings, err := projectSettings(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}
	var projectIssues []Issue
	for _, name := range order {
		a := projects[name]
		p := a.p
		p.ConstructionType = dominantType(a.types)
		set := settings[name]
		if err := applyProjectCategory(ctx, q, p, set.Category); err != nil {
			return nil, err
		}
		p.HasInfo = !set.Info.empty()
		if set.Boundary != nil {
			area := set.Boundary.AreaM2
			p.AreaM2 = &area
			for _, i := range set.Boundary.Issues {
				if i.Level == "warning" {
					p.Issues = append(p.Issues, warnIssue(i.Code, fmt.Sprintf("«%s» хил: %s", name, i.Message)))
				}
			}
		}
		if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM land_acquisition
			WHERE acquisition_name = $1 AND created_by = $2 AND deleted_at IS NULL)`, name, Actor).Scan(&p.Existing); err != nil {
			return nil, err
		}
		if p.Existing {
			p.Issues = append(p.Issues, errIssue("existing", fmt.Sprintf("«%s»: өмнө импортоор орсон чөлөөлөлт — дахин оруулахгүй (эхлээд тэр импортыг «Устгах»-аар устгана уу)", name)))
		}
		if len(p.Variants) > 1 {
			p.Issues = append(p.Issues, infoIssue("name_merged", fmt.Sprintf("«%s»: %d өөр бичиглэлийг нэг төсөл болгон нэгтгэв", name, len(p.Variants))))
		}
		projectIssues = append(projectIssues, p.Issues...)
		view.Projects = append(view.Projects, *p)
	}
	st.Projects = len(view.Projects)

	var fileIssues, all []Issue
	for _, f := range files {
		fileIssues = append(fileIssues, f.Issues...)
	}
	all = append(all, fileIssues...)
	all = append(all, projectIssues...)
	for _, r := range view.Rows {
		all = append(all, r.Issues...)
	}
	view.Errors = countLevel(all, "error")
	view.Warnings = countLevel(all, "warning")
	view.GusDone = st.Pending == 0
	view.CanProceed = countLevel(fileIssues, "error") == 0 && countLevel(projectIssues, "error") == 0 &&
		len(view.Rows) > 0 && len(view.Projects) > 0
	return view, nil
}

// applyProjectCategory — гараар сонгосон (ID-аар) ангилал, эс бөгөөс
// бүтээн байгуулалтын төрлөөс. Санд алга бол алдаа (issue) — импорт зогсоно.
func applyProjectCategory(ctx context.Context, q querier, p *ValuationProject, chosen *CategoryChoice) error {
	if chosen != nil {
		cats, err := resolveCategories(ctx, q, *chosen)
		if err != nil {
			p.Issues = append(p.Issues, errIssue("category_missing", fmt.Sprintf("«%s»: сонгосон ангилал — %s", p.Name, err.Error())))
			return nil
		}
		p.Category, p.CategoryMatched, p.CategoryChosen = categoryName{General: cats.General.Name}, true, true
		if cats.Sub != nil {
			p.Category.Sub = cats.Sub.Name
		}
		p.CategoryIDs = cats.choice()
		return nil
	}
	p.Category, p.CategoryMatched = categoryForConstructionType(p.ConstructionType)
	if !p.CategoryMatched {
		p.Issues = append(p.Issues, warnIssue("category_fallback", fmt.Sprintf("«%s»: бүтээн байгуулалтын төрөл «%s» ангилалд таарсангүй — «%s / %s»-д орно (ангиллыг гараар сонгож болно)",
			p.Name, p.ConstructionType, p.Category.General, p.Category.Sub)))
	}
	cats, err := resolveCategoryNames(ctx, q, p.Category)
	if err != nil {
		p.Issues = append(p.Issues, errIssue("category_missing", fmt.Sprintf("«%s»: %s — ангиллыг гараар сонгоно уу", p.Name, err.Error())))
		return nil
	}
	p.CategoryIDs = cats.choice()
	return nil
}

// choice — ID-аар (frontend-ийн сонгох талбарын утга).
func (c *Categories) choice() *CategoryChoice {
	out := &CategoryChoice{General: strconv.Itoa(c.General.ID)}
	if c.Sub != nil {
		out.Sub = strconv.Itoa(c.Sub.ID)
	}
	return out
}

func dominantType(types map[string]int) string {
	keys := make([]string, 0, len(types))
	for k := range types {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if types[keys[i]] != types[keys[j]] {
			return types[keys[i]] > types[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// resolveCategoryNames — ангиллыг НЭРЭЭР (орчин бүрт id өөр байж болно).
func resolveCategoryNames(ctx context.Context, q querier, c categoryName) (*Categories, error) {
	var general CategoryRef
	err := q.QueryRow(ctx, `SELECT id, name FROM acquisition_category WHERE parent_id IS NULL AND name = $1 ORDER BY id LIMIT 1`,
		c.General).Scan(&general.ID, &general.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("«%s» ангилал санд алга", c.General)
	}
	if err != nil {
		return nil, err
	}
	out := &Categories{General: general}
	if c.Sub != "" {
		var sub CategoryRef
		err := q.QueryRow(ctx, `SELECT id, name FROM acquisition_category WHERE parent_id = $1 AND name = $2 ORDER BY id LIMIT 1`,
			general.ID, c.Sub).Scan(&sub.ID, &sub.Name)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("«%s / %s» дэд ангилал санд алга", c.General, c.Sub)
		}
		if err != nil {
			return nil, err
		}
		out.Sub = &sub
	}
	return out, nil
}

// uploadValuation — «Зөвхөн үнэлгээ»: эксэлийг түр хүснэгтэд, ГУС-ын хайлт арын ажлаар.
func (s *Service) uploadValuation(ctx context.Context, id string, files []UploadedFile) (*State, error) {
	infos, err := s.stageValuationReports(ctx, s.db, id, files)
	if err != nil {
		return nil, err
	}
	view, err := s.buildValuationView(ctx, s.db, id, infos)
	if err != nil {
		return nil, err
	}
	if err := s.save(ctx, s.db, id, "reports", view); err != nil {
		return nil, err
	}
	s.enqueue(id, "reports", func(ctx context.Context, job *Job) error {
		if err := s.lookupValuationInGus(ctx, s.db, id, func(d, t int) { s.setProgress(job, d, t) }); err != nil {
			return err
		}
		view, err := s.buildValuationView(ctx, s.db, id, infos)
		if err != nil {
			return err
		}
		return s.save(ctx, s.db, id, "reports", view)
	})
	return s.Get(ctx, id)
}

// valuationSummary — «Зөвхөн үнэлгээ» горимын нэгтгэл: төсөл бүр = чөлөөлөлт.
func valuationSummary(ss *session) *Summary {
	out := &Summary{Acquisitions: []SummaryAcquisition{}, Blockers: []string{}, Cautions: []string{}}
	switch ss.Status {
	case "committed", "rolled_back":
		out.Blockers = append(out.Blockers, "Энэ импорт аль хэдийн оруулагдсан")
	case "committing":
		out.Blockers = append(out.Blockers, "Оруулж байна…")
	}
	r := ss.reports
	if r == nil {
		out.Blockers = append(out.Blockers, "Үнэлгээний файл оруулаагүй")
		return out
	}
	if !r.CanProceed {
		out.Blockers = append(out.Blockers, "Үнэлгээний файлд алдаа байна")
	}
	if !r.GusDone {
		out.Blockers = append(out.Blockers, "Нэгж талбарыг ГУС-аас шалгаж дуусаагүй")
	}
	st := r.Stats
	if st.DB > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d мөрийн нэгж талбар санд өөр бүртгэлд байгаа тул тэдгээрийн үнэлгээ орохгүй", st.DB))
	}
	if st.Missing > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d мөрийн «Бүтээн байгуулалтын ажлын нэр» хоосон тул орохгүй", st.Missing))
	}
	if st.Excel > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбар ГУС-д олдоогүй — эксэлийн мэдээллээр (үндсэн хилгүй) бүртгэгдэнэ", st.Excel))
	}
	if st.GusFromUb > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбар зөвхөн data_ub-аас олдсон — эрхийн төрөл, гэрээ дутуу", st.GusFromUb))
	}
	if st.Synthetic > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d мөр нэгж талбарын дугааргүй — ERR… дугаараар бүртгэгдэнэ", st.Synthetic))
	}
	if st.Duplicates > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d мөрийн дугаар файлд дахин тохиолдсон — D<n>… дугаараар тусад нь бүртгэгдэнэ", st.Duplicates))
	}
	unmatched := 0
	for i, p := range r.Projects {
		if !p.CategoryMatched {
			unmatched++
		}
		out.Acquisitions = append(out.Acquisitions, SummaryAcquisition{Idx: i + 1, Name: p.Name, Existing: p.Existing,
			AreaM2: p.AreaM2, Parcels: p.Parcels, ReportRows: p.Rows})
	}
	if unmatched > 0 {
		out.Cautions = append(out.Cautions, fmt.Sprintf("%d төслийн бүтээн байгуулалтын төрөл ангилалд таарсангүй — «%s / %s»-д орно",
			unmatched, fallbackCategory.General, fallbackCategory.Sub))
	}
	out.Reports = &st
	out.CanCommit = len(out.Blockers) == 0
	return out
}
