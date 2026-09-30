package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ── ОРУУЛСАН ИМПОРТЫГ «УСТГАХ» ──────────────────────────────────────────────
//
// ХҮРЭЭ: ЗӨВХӨН импортын бүртгэл (legacy_import_stage.owned) — «Оруулах» үед
// импорт ӨӨРӨӨ үүсгэсэн чөлөөлөлт, нэгж талбар, үнэлгээ, санхүүжилт, захирамж.
// Мөр бүрийг НЭГ БҮРЧЛЭН дуудаж устгана; өөр газраас аль хэдийн устсан бол
// логт бичээд алгасна. Алхам:
//
//  1. Үнэлгээ — нэгж талбарыг захирамжаас салгаж, олговор/хөрөнгө/үнэлгээг устгана.
//     Хоосорсон импортын санхүүжилтийн холбоос, эх үүсвэр, захирамж устна.
//  2. Нэгж талбар — нэгж талбар ба түүний БҮХ мэдээлэл (FK CASCADE), холбоосгүй
//     болсон баримт, импорт үүсгэсэн эзэмшигч (өөр газар хэрэглэгдээгүй бол).
//  3. Чөлөөлөлт — импортоос ӨӨР нэгж талбар орсон, эсвэл өөр чөлөөлөлтийн нэгж
//     талбар санхүүжилтийг нь хэрэглэж буй бол ҮЛДЭЭНЭ (хэрэглэгчид харуулна);
//     эс бөгөөс чөлөөлөлт ба импорт үүсгэсэн төлөвлөгөө устна.
//
// Бүх алхам НЭГ гүйлгээнд: гэнэтийн алдаа гарвал юу ч устахгүй, алдаа логт бичигдэнэ.
// audit_log хэзээ ч устахгүй.

// FileDeleter — файлын систем (MinIO/локал). filestorage.Storage-тэй нийцнэ.
type FileDeleter interface {
	Delete(ctx context.Context, fileURL string) error
}

// LayerRemover — дроны зургийн GeoServer давхарга. service.DroneStorage.
type LayerRemover interface {
	RemoveLayer(ctx context.Context, fileName string) error
}

// WithCleanup — «Устгах» үед файл ба дроны давхаргыг устгах хэрэгсэл.
func (s *Service) WithCleanup(files FileDeleter, layers LayerRemover) *Service {
	s.files, s.layers = files, layers
	return s
}

// RollbackAcquisition — чөлөөлөлт ба түүнийг устгах эсэх.
type RollbackAcquisition struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
	// Keep — импортоос өөр мэдээлэлтэй тул чөлөөлөлт ҮЛДЭНЭ.
	Keep       bool     `json:"keep"`
	KeepReason string   `json:"keepReason,omitempty"`
	Others     []string `json:"others"` // импортоос ӨӨР (үлдэх) нэгж талбарууд
}

// RollbackCount — бүртгэлд байгаа / санд одоо байгаа / аль хэдийн устсан.
type RollbackCount struct {
	Total   int `json:"total"`
	Exists  int `json:"exists"`
	Missing int `json:"missing"`
}

// RollbackPreview — «Устгах»-ын өмнө юу устах, юу үлдэхийг харуулна.
type RollbackPreview struct {
	Acquisitions  []RollbackAcquisition `json:"acquisitions"`
	Parcels       RollbackCount         `json:"parcels"`
	Valuations    RollbackCount         `json:"valuations"`
	Compensations int                   `json:"compensations"`
	Amount        float64               `json:"amount"`
	DecisionLinks int                   `json:"decisionLinks"`
	// AddedLater — импортын нэгж талбарт ДАРАА нь хүн нэмсэн мөр (мөн устана).
	AddedLater  map[string]int `json:"addedLater"`
	Blockers    []string       `json:"blockers"`
	CanRollback bool           `json:"canRollback"`
}

// RollbackResult — session.rollback-д хадгалагдана.
type RollbackResult struct {
	At           string         `json:"at"`
	By           string         `json:"by"`
	Acquisitions []string       `json:"acquisitions"` // устсан
	Kept         []string       `json:"kept"`         // үлдсэн (импортоос өөр мэдээлэлтэй)
	Counts       map[string]int `json:"counts"`       // "<action>:<kind>" → тоо
	Files        int            `json:"files"`
	FileErrors   int            `json:"fileErrors"`
}

// RollbackLogEntry — «Устгах»-ын нэг алхмын бичлэг.
type RollbackLogEntry struct {
	At      string `json:"at"`
	By      string `json:"by"`
	Step    string `json:"step"` // valuation | parcel | acquisition | file | rollback
	Kind    string `json:"kind"`
	RefID   string `json:"refID"`
	Label   string `json:"label"`
	Action  string `json:"action"` // deleted | detached | skipped | kept | failed
	Message string `json:"message"`
}

type ownedRef struct{ id, label string }

// manifest — импортын бүртгэл, төрлөөр.
type manifest map[string][]ownedRef

func (m manifest) ids(kind string) []string {
	out := make([]string, 0, len(m[kind]))
	for _, r := range m[kind] {
		out = append(out, r.id)
	}
	return out
}

func (m manifest) label(kind, id string) (string, bool) {
	for _, r := range m[kind] {
		if r.id == id {
			return r.label, true
		}
	}
	return "", false
}

func (m manifest) has(kind, id string) bool {
	_, ok := m.label(kind, id)
	return ok
}

func loadManifest(ctx context.Context, q querier, sessionID string) (manifest, error) {
	rows, err := q.Query(ctx, `SELECT kind, ref_id::text, label FROM `+stage+`.owned
		WHERE session_id = $1 ORDER BY kind, label, ref_id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := manifest{}
	for rows.Next() {
		var kind string
		var r ownedRef
		if err := rows.Scan(&kind, &r.id, &r.label); err != nil {
			return nil, err
		}
		m[kind] = append(m[kind], r)
	}
	return m, rows.Err()
}

func collectStrings(ctx context.Context, q querier, out *[]string, sql string, args ...any) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	vals, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	*out = append((*out)[:0], vals...)
	return nil
}

func rowExists(ctx context.Context, q querier, sql string, args ...any) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (`+sql+`)`, args...).Scan(&ok)
	return ok, err
}

// ── Урьдчилсан харагдац ─────────────────────────────────────────────────────

// keepReason — чөлөөлөлтийг устгахгүй байх шалтгаан ба импортоос ӨӨР нэгж
// талбарууд. ourParcels — импортын нэгж талбарууд (2-р алхамд устах).
func keepReason(ctx context.Context, q querier, acqID string, ourParcels []string) (string, []string, error) {
	var others []string
	if err := collectStrings(ctx, q, &others, `SELECT parcel_id FROM parcel
		WHERE acquisition_id = $1 AND NOT (id = ANY($2::uuid[])) ORDER BY parcel_id`, acqID, ourParcels); err != nil {
		return "", nil, err
	}
	if len(others) > 0 {
		return fmt.Sprintf("импортоос өөр %d нэгж талбар орсон тул чөлөөлөлт болон тэдгээр нэгж талбар ҮЛДЭНЭ", len(others)), others, nil
	}
	var foreign int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM decision_draft_parcel dp
		JOIN decision_draft_funding_source f ON f.id = dp.funding_link_id
		JOIN funding_sources fs ON fs.id = f.funding_source_id
		JOIN parcel p ON p.id = dp.parcel_id
		WHERE fs.acquisition_id = $1 AND p.acquisition_id <> $1 AND NOT (dp.parcel_id = ANY($2::uuid[]))`,
		acqID, ourParcels).Scan(&foreign); err != nil {
		return "", nil, err
	}
	if foreign > 0 {
		return fmt.Sprintf("өөр чөлөөлөлтийн %d нэгж талбар энэ чөлөөлөлтийн санхүүжилтийг захирамжаар хэрэглэж байгаа тул ҮЛДЭНЭ", foreign), []string{}, nil
	}
	return "", []string{}, nil
}

// RollbackPreview — юу ч өөрчлөхгүй.
func (s *Service) RollbackPreview(ctx context.Context, id string) (*RollbackPreview, error) {
	ss, err := s.load(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	m, err := loadManifest(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	pv := &RollbackPreview{Acquisitions: []RollbackAcquisition{}, AddedLater: map[string]int{}, Blockers: []string{}}
	switch {
	case ss.Status == "rolled_back":
		pv.Blockers = append(pv.Blockers, "Энэ импортыг аль хэдийн устгасан")
	case ss.Status != "committed":
		pv.Blockers = append(pv.Blockers, "Зөвхөн системд ОРУУЛСАН импортыг устгана")
	case len(m["land_acquisition"]) == 0:
		pv.Blockers = append(pv.Blockers, "Импортын бүртгэл (устгах мэдээллийн жагсаалт) олдсонгүй — өмнөх хувилбараар оруулсан импорт")
	}

	var existingParcels []string
	if err := collectStrings(ctx, s.db, &existingParcels, `SELECT id::text FROM parcel WHERE id = ANY($1::uuid[])`, m.ids("parcel")); err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(m["parcel"]))
	for _, r := range m["parcel"] {
		codes = append(codes, r.label)
	}
	pv.Parcels = RollbackCount{Total: len(m["parcel"]), Exists: len(existingParcels), Missing: len(m["parcel"]) - len(existingParcels)}
	var valuations int
	// Үнэлгээний аль нэг мөр (олговор, хөрөнгө, газрын үнэлгээ, баталгаажуулалт) байвал «байгаа».
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM parcel p WHERE p.id = ANY($1::uuid[]) AND (
		   EXISTS (SELECT 1 FROM compensation x WHERE x.acquisition_id = p.acquisition_id AND x.parcel_id = p.parcel_id)
		OR EXISTS (SELECT 1 FROM assets x WHERE x.acquisition_id = p.acquisition_id AND x.parcel_id = p.parcel_id)
		OR EXISTS (SELECT 1 FROM land_valuation x WHERE x.acquisition_id = p.acquisition_id AND x.parcel_id = p.parcel_id)
		OR EXISTS (SELECT 1 FROM parcel_valuation_submission x WHERE x.acquisition_id = p.acquisition_id AND x.parcel_id = p.parcel_id))`,
		m.ids("valuation")).Scan(&valuations); err != nil {
		return nil, err
	}
	pv.Valuations = RollbackCount{Total: len(m["valuation"]), Exists: valuations, Missing: len(m["valuation"]) - valuations}

	var later int
	if err := s.db.QueryRow(ctx, `SELECT count(*), COALESCE(sum(amount), 0)::float8, count(*) FILTER (WHERE created_by IS DISTINCT FROM $3)
		FROM compensation c WHERE c.parcel_id = ANY($2::text[])
		  AND (c.acquisition_id = ANY($1::uuid[]) OR c.acquisition_id IN (SELECT acquisition_id FROM parcel WHERE id = ANY($4::uuid[])))`,
		m.ids("land_acquisition"), codes, Actor, existingParcels).Scan(&pv.Compensations, &pv.Amount, &later); err != nil {
		return nil, err
	}
	if later > 0 {
		pv.AddedLater["compensations"] = later
	}
	if err := s.db.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE linked_by IS DISTINCT FROM $2)
		FROM decision_draft_parcel WHERE parcel_id = ANY($1::uuid[])`, existingParcels, Actor).Scan(&pv.DecisionLinks, &later); err != nil {
		return nil, err
	}
	if later > 0 {
		pv.AddedLater["decisionLinks"] = later
	}
	var docs int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM parcel_document WHERE parcel_id = ANY($1::uuid[])`, existingParcels).Scan(&docs); err != nil {
		return nil, err
	}
	if docs > 0 {
		pv.AddedLater["documents"] = docs
	}

	for _, r := range m["land_acquisition"] {
		a := RollbackAcquisition{ID: r.id, Name: r.label, Others: []string{}}
		var name string
		switch err := s.db.QueryRow(ctx, `SELECT COALESCE(acquisition_name, '') FROM land_acquisition WHERE id = $1`, r.id).Scan(&name); {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return nil, err
		default:
			a.Exists, a.Name = true, firstNonEmpty(name, r.label)
			if a.KeepReason, a.Others, err = keepReason(ctx, s.db, r.id, existingParcels); err != nil {
				return nil, err
			}
			a.Keep = a.KeepReason != ""
		}
		pv.Acquisitions = append(pv.Acquisitions, a)
	}
	pv.CanRollback = len(pv.Blockers) == 0
	return pv, nil
}

// ── Устгах ──────────────────────────────────────────────────────────────────

type rollbackLog struct {
	by      string
	entries []RollbackLogEntry
	counts  map[string]int
}

func newRollbackLog(by string) *rollbackLog { return &rollbackLog{by: by, counts: map[string]int{}} }

func (l *rollbackLog) add(step, kind, ref, label, action, msg string) {
	l.entries = append(l.entries, RollbackLogEntry{At: time.Now().Format(time.RFC3339), By: l.by,
		Step: step, Kind: kind, RefID: ref, Label: label, Action: action, Message: msg})
	l.counts[action+":"+kind]++
}

type copier interface {
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
}

func (l *rollbackLog) save(ctx context.Context, q copier, sessionID string) error {
	rows := make([][]any, 0, len(l.entries))
	for _, e := range l.entries {
		at, _ := time.Parse(time.RFC3339, e.At)
		rows = append(rows, []any{sessionID, at, e.By, e.Step, e.Kind, e.RefID, e.Label, e.Action, e.Message})
	}
	_, err := q.CopyFrom(ctx, pgx.Identifier{"legacy_import_stage", "rollback_log"},
		[]string{"session_id", "created_at", "created_by", "step", "kind", "ref_id", "label", "action", "message"},
		pgx.CopyFromRows(rows))
	return err
}

// Rollback — импортын бүртгэлийн мөрүүдийг дарааллаар нь устгана.
func (s *Service) Rollback(ctx context.Context, id, by string) (*RollbackResult, error) {
	if j := s.job(id); j != nil && j.State == "running" {
		return nil, ErrBusy
	}
	result, urls, drone, err := s.rollbackTx(ctx, id, by)
	if err != nil {
		var step *StepError
		if !errors.As(err, &step) && !errors.Is(err, ErrNotFound) {
			// Гүйлгээ буцсан — юу ч устаагүй. Алдааг түүхэнд үлдээнэ.
			lg := newRollbackLog(by)
			lg.add("rollback", "session", id, "", "failed", "Устгал амжилтгүй, юу ч устгагдаагүй: "+err.Error())
			if saveErr := lg.save(context.Background(), s.db, id); saveErr != nil {
				slog.Error("legacy import rollback log", "session", id, "error", saveErr)
			}
			slog.Error("legacy import rollback failed", "session", id, "by", by, "error", err)
		}
		return nil, err
	}
	slog.Info("legacy import rolled back", "session", id, "by", by, "counts", result.Counts)
	s.cleanupFiles(id, by, result, urls, drone)
	return result, nil
}

func (s *Service) rollbackTx(ctx context.Context, id, by string) (*RollbackResult, []string, []string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM `+stage+`.session WHERE id = $1 FOR UPDATE`, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, ErrNotFound
		}
		return nil, nil, nil, err
	}
	if status != "committed" {
		return nil, nil, nil, &StepError{"Зөвхөн системд ОРУУЛСАН импортыг устгана (одоогийн төлөв: " + status + ")"}
	}
	m, err := loadManifest(ctx, tx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(m["land_acquisition"]) == 0 {
		return nil, nil, nil, &StepError{"Импортын бүртгэл (устгах мэдээллийн жагсаалт) олдсонгүй — өмнөх хувилбараар оруулсан импорт"}
	}
	lg := newRollbackLog(by)
	acqIDs := m.ids("land_acquisition")

	// Файлын замууд — устгахаас ӨМНӨ (гүйлгээ амжилттай бол, өөр мөр хэрэглээгүй бол л устгана).
	var existingParcels, urls []string
	if err := collectStrings(ctx, tx, &existingParcels, `SELECT id::text FROM parcel WHERE id = ANY($1::uuid[])`, m.ids("parcel")); err != nil {
		return nil, nil, nil, err
	}
	if err := collectStrings(ctx, tx, &urls, fileURLsSQL, acqIDs, existingParcels); err != nil {
		return nil, nil, nil, fmt.Errorf("файлын жагсаалт: %w", err)
	}

	if err := rollbackValuations(ctx, tx, m, acqIDs, lg); err != nil {
		return nil, nil, nil, fmt.Errorf("1. үнэлгээ: %w", err)
	}
	if err := rollbackParcels(ctx, tx, m, lg); err != nil {
		return nil, nil, nil, fmt.Errorf("2. нэгж талбар: %w", err)
	}
	deleted, kept, drone, err := rollbackAcquisitions(ctx, tx, m, lg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("3. чөлөөлөлт: %w", err)
	}

	result := &RollbackResult{At: time.Now().Format(time.RFC3339), By: by, Acquisitions: deleted, Kept: kept, Counts: lg.counts}
	if err := lg.save(ctx, tx, id); err != nil {
		return nil, nil, nil, fmt.Errorf("устгалын лог: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE `+stage+`.session SET status = 'rolled_back', rollback = $2::jsonb, updated_at = now()
		WHERE id = $1`, id, jsonOf(result)); err != nil {
		return nil, nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, nil, err
	}
	return result, urls, drone, nil
}

// rollbackValuations — 1-р алхам: захирамжаас салгаж, үнэлгээ/олговрыг устгана.
func rollbackValuations(ctx context.Context, tx pgx.Tx, m manifest, acqIDs []string, lg *rollbackLog) error {
	touchedLinks, touchedDrafts := map[string]bool{}, map[string]bool{}
	for _, p := range m["parcel"] {
		code := p.label
		var acqID string
		err := tx.QueryRow(ctx, `SELECT acquisition_id::text FROM parcel WHERE id = $1`, p.id).Scan(&acqID)
		parcelExists := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		scope := acqIDs
		if parcelExists {
			scope = append([]string{acqID}, acqIDs...)
			// Захирамжаас салгана (FK RESTRICT funding_link-ээс өмнө).
			rows, err := tx.Query(ctx, `WITH d AS (
					DELETE FROM decision_draft_parcel WHERE parcel_id = $1
					RETURNING decision_draft_id, funding_link_id, removed_at IS NULL AS active)
				SELECT d.decision_draft_id::text, COALESCE(d.funding_link_id::text, ''), d.active,
				       COALESCE(NULLIF(dd.decree_number, ''), dd.proposal_no, '')
				  FROM d LEFT JOIN decision_draft dd ON dd.id = d.decision_draft_id`, p.id)
			if err != nil {
				return err
			}
			type link struct {
				draft, funding, decree string
				active                 bool
			}
			links, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (link, error) {
				var l link
				return l, r.Scan(&l.draft, &l.funding, &l.active, &l.decree)
			})
			if err != nil {
				return err
			}
			for _, l := range links {
				touchedDrafts[l.draft] = true
				if l.funding != "" {
					touchedLinks[l.funding] = true
				}
				state := "идэвхтэй"
				if !l.active {
					state = "түүхэн"
				}
				lg.add("valuation", "decision_link", l.draft, code, "detached", fmt.Sprintf("захирамж %s-ээс салгав (%s холбоос)", l.decree, state))
			}
		}

		// Олговор эхэлж (assets устахад compensation.asset_id SET NULL болохгүйн тулд),
		// дараа нь хөрөнгө, үнэлгээ.
		var comps, compsLater int
		var amount float64
		if err := tx.QueryRow(ctx, `WITH c AS (
				DELETE FROM compensation WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2
				RETURNING amount, created_by)
			SELECT count(*), COALESCE(sum(amount), 0)::float8, count(*) FILTER (WHERE created_by IS DISTINCT FROM $3) FROM c`,
			scope, code, Actor).Scan(&comps, &amount, &compsLater); err != nil {
			return err
		}
		other := 0
		for _, q := range []string{
			`DELETE FROM compensation_history WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
			`DELETE FROM assets WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
			`DELETE FROM land_valuation WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
			`DELETE FROM parcel_valuation_submission_history WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
			`DELETE FROM parcel_valuation_submission WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
			`DELETE FROM parcel_valuation_snapshot WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
			`DELETE FROM valuation_section_note WHERE acquisition_id = ANY($1::uuid[]) AND parcel_id = $2`,
		} {
			tag, err := tx.Exec(ctx, q, scope, code)
			if err != nil {
				return err
			}
			other += int(tag.RowsAffected())
		}
		if parcelExists {
			tag, err := tx.Exec(ctx, `DELETE FROM parcel_estimated_value_history WHERE parcel_id = $1`, p.id)
			if err != nil {
				return err
			}
			other += int(tag.RowsAffected())
		}
		imported := m.has("valuation", p.id)
		switch {
		case comps+other == 0 && imported:
			reason := "үнэлгээ олдсонгүй — өөр газраас аль хэдийн устсан"
			if !parcelExists {
				reason = "нэгж талбар ба үнэлгээ өөр газраас аль хэдийн устсан"
			}
			lg.add("valuation", "valuation", p.id, code, "skipped", reason)
		case comps+other > 0:
			msg := fmt.Sprintf("%d олговор (%s₮), бусад үнэлгээний %d мөр устгав", comps, fmtInt(amount), other)
			if compsLater > 0 {
				msg += fmt.Sprintf("; үүнээс импортын дараа нэмсэн олговор %d", compsLater)
			}
			if !imported {
				msg = "импортын дараа нэмсэн үнэлгээ: " + msg
			}
			lg.add("valuation", "valuation", p.id, code, "deleted", msg)
		}
	}

	// Санхүүжилтийн холбоос — хоосорсон импортынх устна, үлдсэнийх дүн дахин бодогдоно.
	for _, r := range m["funding_link"] {
		touchedLinks[r.id] = true
	}
	links := sortedKeys(touchedLinks)
	for _, linkID := range links {
		owned := m.has("funding_link", linkID)
		var decree string
		err := tx.QueryRow(ctx, `SELECT COALESCE(NULLIF(d.decree_number, ''), d.proposal_no, '') FROM decision_draft_funding_source f
			JOIN decision_draft d ON d.id = f.decision_draft_id WHERE f.id = $1`, linkID).Scan(&decree)
		if errors.Is(err, pgx.ErrNoRows) {
			if owned {
				lg.add("valuation", "funding_link", linkID, "", "skipped", "захирамжийн санхүүжилтийн холбоос аль хэдийн устсан")
			}
			continue
		}
		if err != nil {
			return err
		}
		used, err := rowExists(ctx, tx, `SELECT 1 FROM decision_draft_parcel WHERE funding_link_id = $1`, linkID)
		if err != nil {
			return err
		}
		switch {
		case owned && !used:
			if _, err := tx.Exec(ctx, `DELETE FROM decision_draft_funding_source WHERE id = $1`, linkID); err != nil {
				return err
			}
			lg.add("valuation", "funding_link", linkID, decree, "deleted", "захирамжийн санхүүжилтийн холбоос устгав")
		case owned:
			lg.add("valuation", "funding_link", linkID, decree, "kept", "өөр нэгж талбар хэрэглэж байгаа тул санхүүжилтийн холбоос үлдэв")
		}
	}
	if err := reconcileFundingLinks(ctx, tx, links); err != nil {
		return err
	}
	for _, r := range m["funding_source"] {
		ok, err := rowExists(ctx, tx, `SELECT 1 FROM funding_sources WHERE id = $1`, r.id)
		if err != nil {
			return err
		}
		if !ok {
			lg.add("valuation", "funding_source", r.id, r.label, "skipped", "санхүүжилтийн эх үүсвэр аль хэдийн устсан")
			continue
		}
		used, err := rowExists(ctx, tx, `SELECT 1 FROM decision_draft_funding_source WHERE funding_source_id = $1`, r.id)
		if err != nil {
			return err
		}
		if used {
			lg.add("valuation", "funding_source", r.id, r.label, "kept", "захирамжид холбоотой хэвээр тул санхүүжилтийн эх үүсвэр үлдэв")
			continue
		}
		if _, err := tx.Exec(ctx, `DELETE FROM funding_sources WHERE id = $1`, r.id); err != nil {
			return err
		}
		lg.add("valuation", "funding_source", r.id, r.label, "deleted", "санхүүжилтийн эх үүсвэр устгав")
	}
	// Импорт үүсгэсэн захирамж — хоосорсон бол устна. Санд байсан захирамжаас
	// зөвхөн холбоосыг салгасан (дээрх).
	for _, r := range m["decision_draft"] {
		touchedDrafts[r.id] = true
	}
	for _, draftID := range sortedKeys(touchedDrafts) {
		label, owned := m.label("decision_draft", draftID)
		if !owned {
			continue
		}
		ok, err := rowExists(ctx, tx, `SELECT 1 FROM decision_draft WHERE id = $1`, draftID)
		if err != nil {
			return err
		}
		if !ok {
			lg.add("valuation", "decision_draft", draftID, label, "skipped", "захирамж аль хэдийн устсан")
			continue
		}
		tag, err := tx.Exec(ctx, `DELETE FROM decision_draft d WHERE d.id = $1
			AND NOT EXISTS (SELECT 1 FROM decision_draft_parcel x WHERE x.decision_draft_id = d.id)
			AND NOT EXISTS (SELECT 1 FROM decision_draft_funding_source x WHERE x.decision_draft_id = d.id)`, draftID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			lg.add("valuation", "decision_draft", draftID, label, "deleted", "импорт үүсгэсэн захирамж устгав")
		} else {
			lg.add("valuation", "decision_draft", draftID, label, "kept", "өөр нэгж талбар/санхүүжилттэй тул захирамж үлдэв")
		}
	}
	return nil
}

// reconcileFundingLinks — үлдсэн импортын санхүүжилтийн дүнг дахин нийлүүлнэ.
func reconcileFundingLinks(ctx context.Context, tx pgx.Tx, links []string) error {
	if len(links) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE decision_draft_funding_source f SET amount = COALESCE((
			SELECT SUM(c.amount) FROM decision_draft_parcel dp JOIN parcel p ON p.id = dp.parcel_id
			  JOIN compensation c ON c.parcel_id = p.parcel_id AND c.acquisition_id = p.acquisition_id
			WHERE dp.funding_link_id = f.id AND dp.removed_at IS NULL AND c.valuation_type = 'asset'), 0)
		WHERE f.id = ANY($1::uuid[]) AND f.created_by = $2`, links, Actor); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE funding_sources fs SET amount = (
			SELECT COALESCE(SUM(amount), 0) FROM decision_draft_funding_source WHERE funding_source_id = fs.id)
		WHERE fs.created_by = $2 AND fs.id IN (SELECT funding_source_id FROM decision_draft_funding_source WHERE id = ANY($1::uuid[]))`,
		links, Actor)
	return err
}

// rollbackParcels — 2-р алхам: нэгж талбар ба түүний бүх мэдээлэл.
func rollbackParcels(ctx context.Context, tx pgx.Tx, m manifest, lg *rollbackLog) error {
	var docIDs []string
	for _, p := range m["parcel"] {
		var acqID string
		err := tx.QueryRow(ctx, `SELECT acquisition_id::text FROM parcel WHERE id = $1`, p.id).Scan(&acqID)
		if errors.Is(err, pgx.ErrNoRows) {
			lg.add("parcel", "parcel", p.id, p.label, "skipped", "нэгж талбар өөр газраас аль хэдийн устсан")
			continue
		}
		if err != nil {
			return err
		}
		var docs []string
		if err := collectStrings(ctx, tx, &docs, `SELECT document_id::text FROM parcel_document WHERE parcel_id = $1`, p.id); err != nil {
			return err
		}
		docIDs = append(docIDs, docs...)
		// FK-гүй хүснэгтүүд.
		if _, err := tx.Exec(ctx, `DELETE FROM notification WHERE parcel_id = $1`, p.id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM authorized_representatives WHERE acquisition_id = $1 AND parcel_id = $2`, acqID, p.label); err != nil {
			return err
		}
		// Үлдсэн бүх мэдээлэл FK CASCADE-аар (эзэмшигчийн холбоос, дэлгэрэнгүй,
		// төлөвийн түүх, баримтын холбоос, төлбөр, хураамж …).
		if _, err := tx.Exec(ctx, `DELETE FROM parcel WHERE id = $1`, p.id); err != nil {
			return err
		}
		msg := "нэгж талбар ба түүний бүх мэдээлэл устгав"
		if len(docs) > 0 {
			msg += fmt.Sprintf(" (баримтын холбоос %d)", len(docs))
		}
		lg.add("parcel", "parcel", p.id, p.label, "deleted", msg)
	}
	if err := deleteOrphanDocuments(ctx, tx, docIDs, "parcel", lg); err != nil {
		return err
	}
	for _, h := range m["right_holder"] {
		ok, err := rowExists(ctx, tx, `SELECT 1 FROM right_holder WHERE id = $1`, h.id)
		if err != nil {
			return err
		}
		if !ok {
			lg.add("parcel", "right_holder", h.id, h.label, "skipped", "эзэмшигч аль хэдийн устсан")
			continue
		}
		tag, err := tx.Exec(ctx, `DELETE FROM right_holder r WHERE r.id = $1
			AND NOT EXISTS (SELECT 1 FROM parcel_holder x WHERE x.holder_id = r.id)
			AND NOT EXISTS (SELECT 1 FROM parcel_detail x WHERE x.holder_id = r.id)`, h.id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			lg.add("parcel", "right_holder", h.id, h.label, "deleted", "импорт үүсгэсэн эзэмшигч устгав")
		} else {
			lg.add("parcel", "right_holder", h.id, h.label, "kept", "өөр нэгж талбарт холбоотой тул эзэмшигч үлдэв")
		}
	}
	return nil
}

// deleteOrphanDocuments — ямар ч холбоосгүй болсон баримт.
func deleteOrphanDocuments(ctx context.Context, tx pgx.Tx, docIDs []string, step string, lg *rollbackLog) error {
	if len(docIDs) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `DELETE FROM document d WHERE d.id = ANY($1::uuid[])
			AND NOT EXISTS (SELECT 1 FROM land_acquisition_document x WHERE x.document_id = d.id)
			AND NOT EXISTS (SELECT 1 FROM acquisition_document x WHERE x.document_id = d.id)
			AND NOT EXISTS (SELECT 1 FROM parcel_document x WHERE x.document_id = d.id)
		RETURNING d.id::text, COALESCE(d.name, '')`, docIDs)
	if err != nil {
		return err
	}
	type doc struct{ id, name string }
	docs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (doc, error) {
		var d doc
		return d, r.Scan(&d.id, &d.name)
	})
	if err != nil {
		return err
	}
	for _, d := range docs {
		lg.add(step, "document", d.id, d.name, "deleted", "холбоосгүй болсон баримт устгав")
	}
	return nil
}

// rollbackAcquisitions — 3-р алхам. Буцаах: устсан, үлдсэн нэр, дроны файлууд.
func rollbackAcquisitions(ctx context.Context, tx pgx.Tx, m manifest, lg *rollbackLog) ([]string, []string, []string, error) {
	deleted, kept, drone := []string{}, []string{}, []string{}
	for _, r := range m["land_acquisition"] {
		var name string
		err := tx.QueryRow(ctx, `SELECT COALESCE(acquisition_name, '') FROM land_acquisition WHERE id = $1`, r.id).Scan(&name)
		if errors.Is(err, pgx.ErrNoRows) {
			lg.add("acquisition", "land_acquisition", r.id, r.label, "skipped", "чөлөөлөлт өөр газраас аль хэдийн устсан")
			continue
		}
		if err != nil {
			return nil, nil, nil, err
		}
		name = firstNonEmpty(name, r.label)
		// 2-р алхмын дараа үлдсэн нэгж талбар бүгд импортынх БИШ.
		reason, others, err := keepReason(ctx, tx, r.id, []string{})
		if err != nil {
			return nil, nil, nil, err
		}
		if reason != "" {
			msg := reason
			if len(others) > 0 {
				msg += ": " + strings.Join(others, ", ")
			}
			lg.add("acquisition", "land_acquisition", r.id, name, "kept", msg)
			kept = append(kept, name)
			continue
		}
		var docs, files []string
		if err := collectStrings(ctx, tx, &docs, `SELECT document_id::text FROM land_acquisition_document WHERE land_acquisition_id = $1
			UNION SELECT document_id::text FROM acquisition_document WHERE acquisition_id = $1`, r.id); err != nil {
			return nil, nil, nil, err
		}
		if err := collectStrings(ctx, tx, &files, `SELECT file_name FROM drone_image WHERE acquisition_id = $1`, r.id); err != nil {
			return nil, nil, nil, err
		}
		// FK-гүй хүснэгтүүд.
		for _, q := range []string{
			`DELETE FROM compensation_history WHERE acquisition_id = $1`,
			`DELETE FROM notification WHERE acquisition_id = $1`,
			`DELETE FROM parcel_valuation_snapshot WHERE acquisition_id = $1`,
			`DELETE FROM valuation_section_note WHERE acquisition_id = $1`,
		} {
			if _, err := tx.Exec(ctx, q, r.id); err != nil {
				return nil, nil, nil, err
			}
		}
		// Энэ чөлөөлөлтийн ХООСОН захирамж (FK нь SET NULL тул өнчрөх байсан).
		// Өөр нэгж талбар эсвэл өөр чөлөөлөлтийн санхүүжилттэй бол үлдэнэ.
		tag, err := tx.Exec(ctx, `DELETE FROM decision_draft d WHERE d.acquisition_id = $1
			AND NOT EXISTS (SELECT 1 FROM decision_draft_parcel x WHERE x.decision_draft_id = d.id)
			AND NOT EXISTS (SELECT 1 FROM decision_draft_funding_source x JOIN funding_sources fs ON fs.id = x.funding_source_id
			                WHERE x.decision_draft_id = d.id AND fs.acquisition_id IS DISTINCT FROM $1)`, r.id)
		if err != nil {
			return nil, nil, nil, err
		}
		if n := tag.RowsAffected(); n > 0 {
			lg.add("acquisition", "decision_draft", r.id, name, "deleted", fmt.Sprintf("чөлөөлөлтийн хоосон захирамж %d устгав", n))
		}
		// Үлдсэн бүх мэдээлэл FK CASCADE-аар (санхүүжилт, явц, баг/хороо, дрон, баримтын холбоос …).
		if _, err := tx.Exec(ctx, `DELETE FROM land_acquisition WHERE id = $1`, r.id); err != nil {
			return nil, nil, nil, err
		}
		lg.add("acquisition", "land_acquisition", r.id, name, "deleted", "чөлөөлөлт ба түүний бүх мэдээлэл устгав")
		deleted = append(deleted, name)
		drone = append(drone, files...)
		if err := deleteOrphanDocuments(ctx, tx, docs, "acquisition", lg); err != nil {
			return nil, nil, nil, err
		}
	}
	for _, p := range m["plan"] {
		ok, err := rowExists(ctx, tx, `SELECT 1 FROM plan WHERE id = $1`, p.id)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			lg.add("acquisition", "plan", p.id, p.label, "skipped", "төлөвлөгөө аль хэдийн устсан")
			continue
		}
		tag, err := tx.Exec(ctx, `DELETE FROM plan p WHERE p.id = $1
			AND NOT EXISTS (SELECT 1 FROM land_acquisition x WHERE x.plan_id = p.id)`, p.id)
		if err != nil {
			return nil, nil, nil, err
		}
		if tag.RowsAffected() > 0 {
			lg.add("acquisition", "plan", p.id, p.label, "deleted", "импорт үүсгэсэн төлөвлөгөө устгав")
		} else {
			lg.add("acquisition", "plan", p.id, p.label, "kept", "өөр чөлөөлөлт хэрэглэж байгаа тул төлөвлөгөө үлдэв")
		}
	}
	return deleted, kept, drone, nil
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// fileURLsSQL — хүрээн дэх файлын замууд ($1 чөлөөлөлт, $2 нэгж талбар).
// Устгалын ДАРАА өөр мөрөнд хэрэглэгдсээр байвал (үлдсэн чөлөөлөлт г.м.) устахгүй.
const fileURLsSQL = `
	SELECT d.file_url FROM document d WHERE d.id IN (
		SELECT document_id FROM land_acquisition_document WHERE land_acquisition_id = ANY($1::uuid[])
		UNION SELECT document_id FROM acquisition_document WHERE acquisition_id = ANY($1::uuid[])
		UNION SELECT document_id FROM parcel_document WHERE parcel_id = ANY($2::uuid[]))
	UNION SELECT file_url FROM drone_image WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT file_url FROM acquisition_socio_survey WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT photo_pdf_url FROM assets WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT valuation_report_url FROM compensation WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT file_url FROM parcel_emongolia_notice WHERE parcel_id = ANY($2::uuid[])
	UNION SELECT attachment_url FROM parcel_estimated_value_history WHERE acquisition_id = ANY($1::uuid[]) OR parcel_id = ANY($2::uuid[])
	UNION SELECT file_url FROM parcel_notice_email WHERE parcel_id = ANY($2::uuid[])
	UNION SELECT attachment_url FROM parcel_status_history WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT report_url FROM parcel_valuation_snapshot WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT source_url FROM parcel_valuation_snapshot WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT f->>'url' FROM parcel_valuation_snapshot s,
	      jsonb_array_elements(CASE WHEN jsonb_typeof(s.files) = 'array' THEN s.files ELSE '[]'::jsonb END) f
	      WHERE s.acquisition_id = ANY($1::uuid[])
	UNION SELECT attachment_url FROM parcel_valuation_submission WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT draft_report_url FROM parcel_valuation_submission WHERE acquisition_id = ANY($1::uuid[])
	UNION SELECT attachment_url FROM parcel_valuation_submission_history WHERE acquisition_id = ANY($1::uuid[])`

// fileReferencedSQL — устгалын ДАРАА энэ зам өөр мөрөнд хэрэглэгдсээр байгаа эсэх.
const fileReferencedSQL = `SELECT EXISTS (
	SELECT 1 FROM document WHERE file_url = $1
	UNION ALL SELECT 1 FROM drone_image WHERE file_url = $1
	UNION ALL SELECT 1 FROM acquisition_socio_survey WHERE file_url = $1
	UNION ALL SELECT 1 FROM assets WHERE photo_pdf_url = $1
	UNION ALL SELECT 1 FROM compensation WHERE valuation_report_url = $1
	UNION ALL SELECT 1 FROM parcel_emongolia_notice WHERE file_url = $1
	UNION ALL SELECT 1 FROM parcel_estimated_value_history WHERE attachment_url = $1
	UNION ALL SELECT 1 FROM parcel_notice_email WHERE file_url = $1
	UNION ALL SELECT 1 FROM parcel_status_history WHERE attachment_url = $1
	UNION ALL SELECT 1 FROM parcel_valuation_snapshot WHERE report_url = $1 OR source_url = $1 OR files @> jsonb_build_array(jsonb_build_object('url', $1::text))
	UNION ALL SELECT 1 FROM parcel_valuation_submission WHERE attachment_url = $1 OR draft_report_url = $1
	UNION ALL SELECT 1 FROM parcel_valuation_submission_history WHERE attachment_url = $1)`

// cleanupFiles — гүйлгээ амжилттай болсны ДАРАА, өөр мөрөнд хэрэглэгдээгүй файл
// ба дроны давхаргыг устгаж логт бичнэ.
func (s *Service) cleanupFiles(id, by string, result *RollbackResult, urls, drone []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	lg := newRollbackLog(by)
	for _, u := range urls {
		if strings.TrimSpace(u) == "" || s.files == nil {
			continue
		}
		var used bool
		if err := s.db.QueryRow(ctx, fileReferencedSQL, u).Scan(&used); err != nil || used {
			continue
		}
		if err := s.files.Delete(ctx, u); err != nil {
			result.FileErrors++
			lg.add("file", "file", "", u, "failed", "файл устгаж чадсангүй: "+err.Error())
			continue
		}
		result.Files++
		lg.add("file", "file", "", u, "deleted", "файл устгав")
	}
	if s.layers != nil {
		for _, name := range drone {
			if used, err := rowExists(ctx, s.db, `SELECT 1 FROM drone_image WHERE file_name = $1`, name); err != nil || used {
				continue
			}
			if err := s.layers.RemoveLayer(ctx, name); err != nil {
				lg.add("file", "drone_layer", "", name, "failed", "дроны давхарга устгаж чадсангүй: "+err.Error())
				continue
			}
			lg.add("file", "drone_layer", "", name, "deleted", "дроны давхарга устгав")
		}
	}
	if len(lg.entries) > 0 {
		if err := lg.save(ctx, s.db, id); err != nil {
			slog.Error("legacy import rollback file log", "session", id, "error", err)
		}
	}
	if _, err := s.db.Exec(ctx, `UPDATE `+stage+`.session SET rollback = $2::jsonb WHERE id = $1`, id, jsonOf(result)); err != nil {
		slog.Error("legacy import rollback result", "session", id, "error", err)
	}
}

// RollbackLog — устгалын түүх (эхнээс нь).
func (s *Service) RollbackLog(ctx context.Context, id string) ([]RollbackLogEntry, error) {
	rows, err := s.db.Query(ctx, `SELECT created_at, created_by, step, kind, ref_id, label, action, message
		FROM `+stage+`.rollback_log WHERE session_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (RollbackLogEntry, error) {
		var e RollbackLogEntry
		var at time.Time
		err := r.Scan(&at, &e.By, &e.Step, &e.Kind, &e.RefID, &e.Label, &e.Action, &e.Message)
		e.At = at.Format(time.RFC3339)
		return e, err
	})
}
