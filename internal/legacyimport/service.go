package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Хэрэглэгчид харагдах алдаа — handler HTTP код руу хөрвүүлнэ.
var (
	ErrNotFound  = errors.New("импорт олдсонгүй")
	ErrCommitted = errors.New("энэ импорт аль хэдийн оруулагдсан")
	ErrBusy      = errors.New("энэ импорт дээр ажил явагдаж байна — дуусахыг хүлээнэ үү")
)

// StepError — алхмыг хийх нөхцөл хангагдаагүй (409).
type StepError struct{ Message string }

func (e *StepError) Error() string { return e.Message }

// Service — хуучин мэдээллийн импорт. Процесс даяар НЭГ (ГУС-ын дараалал,
// арын ажлуудын дараалал нь нийтлэг).
type Service struct {
	db  *pgxpool.Pool
	gus *GusClient

	jobsMu sync.Mutex
	jobs   map[string]*Job
	// queue — ГУС-ын шалгалт ба «Оруулах» НЭГ дараалалд (зэрэгцүүлэхгүй).
	queue chan func()

	// «Хасах» үед файл ба дроны давхарга цэвэрлэх (заавал биш).
	files  FileDeleter
	layers LayerRemover
}

func NewService(db *pgxpool.Pool, gus GusConfig) *Service {
	s := &Service{db: db, gus: NewGusClient(gus), jobs: map[string]*Job{}, queue: make(chan func(), 64)}
	go func() {
		for job := range s.queue {
			job()
		}
	}()
	return s
}

// jobTimeout — нэг ажлын дээд хугацаа (ГУС 1 сек завсарлагатай тул мянга гаруй
// нэгж талбарт хангалттай).
const jobTimeout = 3 * time.Hour

func (s *Service) enqueue(sessionID, kind string, run func(ctx context.Context, job *Job) error) {
	job := &Job{Kind: kind, State: "running"}
	s.jobsMu.Lock()
	s.jobs[sessionID] = job
	s.jobsMu.Unlock()
	s.queue <- func() {
		s.gus.ClearFailures()
		ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		defer cancel()
		err := run(ctx, job)
		s.jobsMu.Lock()
		defer s.jobsMu.Unlock()
		if err != nil {
			job.State, job.Error = "failed", err.Error()
			slog.Error("legacy import job failed", "session", sessionID, "kind", kind, "error", err)
			return
		}
		job.State = "done"
	}
}

func (s *Service) job(sessionID string) *Job {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	if j, ok := s.jobs[sessionID]; ok {
		cp := *j
		return &cp
	}
	return nil
}

func (s *Service) setProgress(job *Job, done, total int) {
	s.jobsMu.Lock()
	job.Done, job.Total = done, total
	s.jobsMu.Unlock()
}

// ── Session ─────────────────────────────────────────────────────────────────

type session struct {
	SessionInfo
	boundary *BoundaryView
	parcels  *ParcelView
	reports  *ReportView
	result   *CommitResult
	projects map[string]ProjectSettings // «Зөвхөн үнэлгээ»: төслийн тохиргоо
}

func (s *Service) load(ctx context.Context, q querier, id string) (*session, error) {
	var ss session
	var created, updated time.Time
	err := q.QueryRow(ctx, `SELECT id::text, created_by, created_at, updated_at, status, mode, category, boundary, parcels, reports, result, rollback,
			COALESCE(projects, '{}')
		FROM `+stage+`.session WHERE id = $1`, id).
		Scan(&ss.ID, &ss.CreatedBy, &created, &updated, &ss.Status, &ss.Mode, &ss.Category, &ss.boundary, &ss.parcels, &ss.reports, &ss.result, &ss.Rollback,
			&ss.projects)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ss.CreatedAt, ss.UpdatedAt = created.Format(time.RFC3339), updated.Format(time.RFC3339)
	return &ss, nil
}

func (s *Service) save(ctx context.Context, q querier, id, column string, value any) error {
	switch column {
	case "category", "boundary", "parcels", "reports", "result":
	default:
		return fmt.Errorf("unknown column %s", column)
	}
	var arg any
	if value != nil {
		arg = jsonOf(value)
	}
	_, err := q.Exec(ctx, `UPDATE `+stage+`.session SET `+column+` = $2::jsonb, updated_at = now() WHERE id = $1`, id, arg)
	return err
}

func (s *Service) setStatus(ctx context.Context, id, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE `+stage+`.session SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

func (s *Service) requireDraft(ctx context.Context, id string) (*session, error) {
	ss, err := s.load(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if ss.Status == "committed" || ss.Status == "rolled_back" {
		return nil, ErrCommitted
	}
	if j := s.job(id); ss.Status == "committing" || (j != nil && j.State == "running") {
		return nil, ErrBusy
	}
	return ss, nil
}

// ── Нийтийн үйлдлүүд ────────────────────────────────────────────────────────

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.Query(ctx, `SELECT id, name, parent_id FROM acquisition_category ORDER BY COALESCE(parent_id, 0), sort_order, id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Category, error) {
		var c Category
		return c, r.Scan(&c.ID, &c.Name, &c.ParentID)
	})
}

// ListFilter — импортын жагсаалтын хуудаслалт ба чөлөөлөлтийн нэрээр хайлт.
type ListFilter struct {
	// Query — чөлөөлөлтийн нэрсийн аль нэгэнд агуулагдах текст (том/жижиг үсэг ялгахгүй).
	Query    string
	Page     int
	PageSize int
}

const maxListPageSize = 100

// likeEscaper — хэрэглэгчийн текстийн %, _ тэмдэгтийг ILIKE-ийн загвар биш,
// жирийн тэмдэгт болгоно.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// List — оруулсан/амжилтгүй/устгасан импортууд (хуудаслалттай) ба нийт тоо.
// Ноорог (оруулаагүй) жагсаалтад харагдахгүй — удаан хөндөгдөөгүйг нь энд
// цэвэрлэнэ (purgeDrafts).
func (s *Service) List(ctx context.Context, f ListFilter) ([]SessionListItem, int64, error) {
	if err := s.purgeDrafts(ctx); err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > maxListPageSize {
		f.PageSize = maxListPageSize
	}
	pattern := ""
	if q := strings.TrimSpace(f.Query); q != "" {
		pattern = "%" + likeEscaper.Replace(q) + "%"
	}

	// names — жагсаалтад харуулдаг ЯГ тэр нэрс (хайлт ч түүгээр).
	const base = `WITH list AS (
		SELECT s.*,
			-- Оруулсан чөлөөлөлтийн нэрс; оруулж амжаагүй (амжилтгүй) бол үүсэх байсных.
			CASE WHEN jsonb_array_length(COALESCE(result->'acquisitions'->'names', '[]')) > 0 THEN result->'acquisitions'->'names'
			     WHEN mode = 'valuation' THEN jsonb_path_query_array(COALESCE(reports, '{}'), '$.projects[*].name')
			     ELSE jsonb_path_query_array(COALESCE(boundary, '{}'), '$.rows[*].name') END AS names
		FROM ` + stage + `.session s WHERE s.status <> 'draft'
	), filtered AS (
		SELECT * FROM list
		 WHERE $1 = '' OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(names) n WHERE n ILIKE $1)
	)`

	var total int64
	if err := s.db.QueryRow(ctx, base+` SELECT count(*) FROM filtered`, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx, base+` SELECT id::text, created_by, created_at, updated_at, status, mode,
			CASE WHEN mode = 'valuation' THEN COALESCE((reports->'stats'->>'projects')::int, 0)
			     ELSE jsonb_array_length(COALESCE(boundary->'rows','[]')) END,
			names,
			COALESCE((parcels->'stats'->>'total')::int, 0),
			COALESCE((reports->'stats'->>'rows')::int, 0),
			s.status = 'committed' AND EXISTS (SELECT 1 FROM `+stage+`.owned o WHERE o.session_id = s.id AND o.kind = 'land_acquisition'),
			rollback
		FROM filtered s ORDER BY updated_at DESC, id LIMIT $2 OFFSET $3`,
		pattern, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (SessionListItem, error) {
		var it SessionListItem
		var created, updated time.Time
		err := r.Scan(&it.ID, &it.CreatedBy, &created, &updated, &it.Status, &it.Mode, &it.Acquisitions, &it.AcquisitionNames,
			&it.Parcels, &it.ReportRows, &it.Deletable, &it.Rollback)
		if it.AcquisitionNames == nil {
			it.AcquisitionNames = []string{}
		}
		it.CreatedAt, it.UpdatedAt = created.Format(time.RFC3339), updated.Format(time.RFC3339)
		return it, err
	})
	return items, total, err
}

// draftIdleTTL — ноорог импорт ингэж удаан хөндөгдөөгүй бол устгагдана. Шууд
// (0) биш: өөр хэрэглэгч яг одоо ажиллаж буй нооргийг жагсаалт нээхэд устгахгүй.
// Өөрийн нооргоос «Бүх импорт» руу гарахад дэлгэц тэр даруй устгадаг.
const draftIdleTTL = time.Hour

// purgeDrafts — удаан хөндөгдөөгүй, арын ажилгүй нооргийг устгана (түр
// хүснэгтийн мөрүүд нь CASCADE-аар хамт устана).
func (s *Service) purgeDrafts(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `SELECT id::text FROM `+stage+`.session
		WHERE status = 'draft' AND updated_at < now() - make_interval(secs => $1)`, draftIdleTTL.Seconds())
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil && !errors.Is(err, ErrBusy) && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

// Create — шинэ импорт. mode: full (хил + нэгж талбар + үнэлгээ) | valuation (зөвхөн үнэлгээ).
func (s *Service) Create(ctx context.Context, createdBy, mode string) (string, error) {
	switch mode {
	case "":
		mode = ModeFull
	case ModeFull, ModeValuation:
	default:
		return "", &StepError{"Импортын горим буруу: " + mode}
	}
	id := uuid.NewString()
	_, err := s.db.Exec(ctx, `INSERT INTO `+stage+`.session (id, created_by, mode) VALUES ($1, $2, $3)`, id, createdBy, mode)
	return id, err
}

// Импортын горим.
const (
	ModeFull      = "full"
	ModeValuation = "valuation"
)

var errValuationMode = &StepError{"«Зөвхөн үнэлгээ» импортод хил, нэгж талбарын файл оруулахгүй — үнэлгээний эксэлээ оруулна уу"}

func (s *Service) Delete(ctx context.Context, id string) error {
	ss, err := s.load(ctx, s.db, id)
	if err != nil {
		return err
	}
	// Оруулсан импорт нь системд юу орсны ТҮҮХ — зөвхөн харна, устгахгүй.
	switch ss.Status {
	case "committed", "rolled_back":
		return ErrCommitted
	case "committing":
		return ErrBusy
	}
	// ГУС-ын шалгалт явж байхад устгавал ажил устсан импорт руу бичнэ.
	if j := s.job(id); j != nil && j.State == "running" {
		return ErrBusy
	}
	// Шалгалт ба устгалын хооронд төлөв өөрчлөгдөхөөс хамгаалж нөхцөлтэй устгана.
	if _, err := s.db.Exec(ctx, `DELETE FROM `+stage+`.session WHERE id = $1 AND status IN ('draft', 'failed')`, id); err != nil {
		return err
	}
	s.jobsMu.Lock()
	delete(s.jobs, id)
	s.jobsMu.Unlock()
	return nil
}

// Progress — арын ажлын явц (хөнгөн): дэлгэц үүгээр л давтан шалгана.
type Progress struct {
	Status string `json:"status"`
	Job    *Job   `json:"job"`
}

func (s *Service) Progress(ctx context.Context, id string) (*Progress, error) {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM `+stage+`.session WHERE id = $1`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &Progress{Status: status, Job: s.job(id)}, nil
}

func (s *Service) Get(ctx context.Context, id string) (*State, error) {
	ss, err := s.load(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	summary, err := s.summary(ctx, ss)
	if err != nil {
		return nil, err
	}
	log, err := s.RollbackLog(ctx, id)
	if err != nil {
		return nil, err
	}
	return &State{Session: ss.SessionInfo, Boundary: ss.boundary, Parcels: ss.parcels, Reports: ss.reports,
		Result: ss.result, Summary: *summary, Job: s.job(id), RollbackLog: log, ProjectSettings: ss.projects}, nil
}

// UploadBoundary — 1-р алхам. Хил солигдвол дараагийн алхмуудын шалгалт арилна.
func (s *Service) UploadBoundary(ctx context.Context, id string, files []UploadedFile, choice CategoryChoice) (*State, error) {
	ss, err := s.requireDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if ss.Mode == ModeValuation {
		return nil, errValuationMode
	}
	view, err := s.analyzeBoundary(ctx, s.db, id, files, choice)
	if err != nil {
		return nil, err
	}
	for col, v := range map[string]any{"category": choice, "boundary": view, "parcels": nil, "reports": nil} {
		if err := s.save(ctx, s.db, id, col, v); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

// UploadParcels — 2-р алхам. ГУС-ын шалгалт арын ажлаар явна.
func (s *Service) UploadParcels(ctx context.Context, id string, files []UploadedFile) (*State, error) {
	ss, err := s.requireDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if ss.Mode == ModeValuation {
		return nil, errValuationMode
	}
	if ss.boundary == nil || !ss.boundary.CanProceed {
		return nil, &StepError{"Эхлээд чөлөөлөлтийн хилийн файлыг алдаагүй оруулна уу"}
	}
	infos, err := s.stageParcels(ctx, s.db, id, files)
	if err != nil {
		return nil, err
	}
	view, err := s.buildParcelView(ctx, s.db, id, infos)
	if err != nil {
		return nil, err
	}
	if err := s.save(ctx, s.db, id, "parcels", view); err != nil {
		return nil, err
	}
	if err := s.save(ctx, s.db, id, "reports", nil); err != nil {
		return nil, err
	}
	s.enqueue(id, "parcels", func(ctx context.Context, job *Job) error {
		if err := s.lookupParcelsInGus(ctx, s.db, id, func(d, t int) { s.setProgress(job, d, t) }); err != nil {
			return err
		}
		view, err := s.buildParcelView(ctx, s.db, id, infos)
		if err != nil {
			return err
		}
		return s.save(ctx, s.db, id, "parcels", view)
	})
	return s.Get(ctx, id)
}

// UploadReports — 3-р алхам.
func (s *Service) UploadReports(ctx context.Context, id string, files []UploadedFile) (*State, error) {
	ss, err := s.requireDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if ss.Mode == ModeValuation {
		return s.uploadValuation(ctx, id, files)
	}
	if ss.boundary == nil || !ss.boundary.CanProceed || ss.parcels == nil || !ss.parcels.CanProceed {
		return nil, &StepError{"Эхлээд чөлөөлөлтийн хил ба нэгж талбарын файлыг оруулна уу"}
	}
	infos, err := s.stageReports(ctx, s.db, id, files)
	if err != nil {
		return nil, err
	}
	view, err := s.buildReportView(ctx, s.db, id, infos)
	if err != nil {
		return nil, err
	}
	if err := s.save(ctx, s.db, id, "reports", view); err != nil {
		return nil, err
	}
	s.enqueue(id, "reports", func(ctx context.Context, job *Job) error {
		if err := s.lookupReportsInGus(ctx, s.db, id, func(d, t int) { s.setProgress(job, d, t) }); err != nil {
			return err
		}
		view, err := s.buildReportView(ctx, s.db, id, infos)
		if err != nil {
			return err
		}
		return s.save(ctx, s.db, id, "reports", view)
	})
	return s.Get(ctx, id)
}

// Commit — 5-р алхам. Арын ажлаар НЭГ гүйлгээнд оруулна.
func (s *Service) Commit(ctx context.Context, id string) error {
	ss, err := s.requireDraft(ctx, id)
	if err != nil {
		return err
	}
	summary, err := s.summary(ctx, ss)
	if err != nil {
		return err
	}
	if !summary.CanCommit {
		return &StepError{joinNonEmpty("; ", summary.Blockers...)}
	}
	// Атомик шилжилт — зэрэг ирсэн хоёр «Оруулах»-ын зөвхөн нэг нь эхэлнэ.
	tag, err := s.db.Exec(ctx, `UPDATE `+stage+`.session SET status = 'committing', updated_at = now()
		WHERE id = $1 AND status IN ('draft', 'failed')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBusy
	}
	choice := CategoryChoice{}
	if ss.Category != nil {
		choice = *ss.Category
	}
	s.enqueue(id, "commit", func(ctx context.Context, _ *Job) error {
		var result *CommitResult
		var err error
		if ss.Mode == ModeValuation {
			result, err = s.commitValuationSession(ctx, id)
		} else {
			result, err = s.commitSession(ctx, id, choice)
		}
		// Хэрэглэгчийн хүсэлт цуцлагдсан ч үр дүнг заавал хадгална.
		saveCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err != nil {
			_ = s.save(saveCtx, s.db, id, "result", CommitResult{Error: err.Error()})
			_ = s.setStatus(saveCtx, id, "failed")
			return err
		}
		result.CommittedAt = time.Now().Format(time.RFC3339)
		if err := s.save(saveCtx, s.db, id, "result", result); err != nil {
			return err
		}
		return s.setStatus(saveCtx, id, "committed")
	})
	return nil
}

// ── 4. Нэгтгэл ──────────────────────────────────────────────────────────────

func (s *Service) summary(ctx context.Context, ss *session) (*Summary, error) {
	if ss.Mode == ModeValuation {
		return valuationSummary(ss), nil
	}
	type counts struct{ parcels, outside, reportRows int }
	byIdx := map[int]counts{}
	rows, err := s.db.Query(ctx, `SELECT b.idx,
			COUNT(p.idx) FILTER (WHERE NOT (p.issues @> '[{"code":"duplicate"}]'))::int,
			COUNT(p.idx) FILTER (WHERE p.outside)::int,
			(SELECT COUNT(DISTINCT r.idx)::int FROM `+stage+`.report_row r
			   JOIN `+stage+`.parcel p2 ON p2.session_id = r.session_id AND p2.parcel_id = r.parcel_id
			  WHERE r.session_id = b.session_id AND p2.boundary_idx = b.idx)
		FROM `+stage+`.boundary b
		LEFT JOIN `+stage+`.parcel p ON p.session_id = b.session_id AND p.boundary_idx = b.idx
		WHERE b.session_id = $1 GROUP BY b.session_id, b.idx`, ss.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var idx int
		var c counts
		if err := rows.Scan(&idx, &c.parcels, &c.outside, &c.reportRows); err != nil {
			rows.Close()
			return nil, err
		}
		byIdx[idx] = c
	}
	rows.Close()

	out := &Summary{Acquisitions: []SummaryAcquisition{}, Blockers: []string{}, Cautions: []string{}}
	switch ss.Status {
	case "committed":
		out.Blockers = append(out.Blockers, "Энэ импорт аль хэдийн оруулагдсан")
	case "committing":
		out.Blockers = append(out.Blockers, "Оруулж байна…")
	}
	switch {
	case ss.boundary == nil:
		out.Blockers = append(out.Blockers, "Чөлөөлөлтийн хилийн файл оруулаагүй")
	case !ss.boundary.CanProceed:
		out.Blockers = append(out.Blockers, "Чөлөөлөлтийн хилийн файлд алдаа байна")
	}
	if ss.parcels == nil {
		out.Blockers = append(out.Blockers, "Нэгж талбарын файл оруулаагүй")
	} else {
		p := ss.parcels
		if !p.CanProceed {
			out.Blockers = append(out.Blockers, "Нэгж талбарын файлд алдаа байна")
		}
		if !p.GusDone {
			out.Blockers = append(out.Blockers, "Нэгж талбарыг ГУС-аас шалгаж дуусаагүй")
		}
		if p.Stats.Outside > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбар чөлөөлөлтийн хилийн гадна — хамгийн ойр чөлөөлөлтөд хамаарна", p.Stats.Outside))
		}
		if n := p.Stats.Gus["missing"] + p.Stats.Gus["failed"]; n > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбар ГУС-аас татагдаагүй — зөвхөн SHP-ийн мэдээллээр орно", n))
		}
		if n := p.Stats.AlreadyImported; n > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбар санд өөр бүртгэлд байгаа тул энэ импортод орохгүй", n))
		}
		if n := p.Stats.Gus["ub"]; n > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбар зөвхөн data_ub-аас олдсон — эрхийн төрөл, гэрээ дутуу", n))
		}
		st := p.Stats
		out.Parcels = &st
	}
	if ss.reports == nil {
		out.Cautions = append(out.Cautions, "Үнэлгээний файл оруулаагүй — үнэлгээ, нөхөх олговор орохгүй")
	} else {
		r := ss.reports
		if !r.CanProceed {
			out.Blockers = append(out.Blockers, "Үнэлгээний файлд алдаа байна")
		}
		if !r.GusDone {
			out.Blockers = append(out.Blockers, "Үнэлгээний нэгж талбарыг ГУС-аас шалгаж дуусаагүй")
		}
		if r.Stats.Missing > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d үнэлгээний мөрийн нэгж талбар олдсонгүй — тэдгээрийн %s₮ ОРОХГҮЙ", r.Stats.Missing, fmtInt(r.Stats.MissingAmount)))
		}
		if r.Stats.DB > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d үнэлгээний мөрийн нэгж талбар санд өөр бүртгэлд байгаа тул тэдгээрийн үнэлгээ орохгүй", r.Stats.DB))
		}
		if r.Stats.WithoutReport > 0 {
			out.Cautions = append(out.Cautions, fmt.Sprintf("%d нэгж талбарт үнэлгээний мөр байхгүй", r.Stats.WithoutReport))
		}
		if r.Stats.OwnerIssues > 0 {
			out.Cautions = append(out.Cautions, ownerCaution(r.Stats.OwnerIssues))
		}
		st := r.Stats
		out.Reports = &st
		out.ParcelsWithoutReport = r.Stats.WithoutReport
	}
	if ss.boundary != nil {
		for _, row := range ss.boundary.Rows {
			c := byIdx[row.Idx]
			out.Acquisitions = append(out.Acquisitions, SummaryAcquisition{
				Idx: row.Idx, PlanParcelID: row.PlanParcelID, Name: row.Name, AreaM2: row.AreaM2,
				Existing: row.Existing, Parcels: c.parcels, Outside: c.outside, ReportRows: c.reportRows,
			})
		}
	}
	out.CanCommit = len(out.Blockers) == 0
	return out, nil
}
