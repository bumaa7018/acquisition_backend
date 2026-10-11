package seed

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"acquisition_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Нэгж талбарын төлөв ба ажлын урсгал — 103.87.255.222/appdb2-т тохируулсан
// бүртгэлээс (2026-10-11). id нь code-той ИЖИЛ (101–109, domain.ParcelStatus*);
// sequence нь 109 дээр — админы шинэ төлөв 110-аас.

type parcelStatusSeed struct {
	id                                                int
	code, name                                        string
	sortOrder                                         int
	color                                             string
	final, locksProgress, valuation, released, reason bool
}

var parcelStatusSeeds = []parcelStatusSeed{
	{domain.ParcelStatusPending, "101", "НӨЛӨӨЛӨЛД ӨРТСӨН НЭГЖ ТАЛБАР", 1, "#e8ecf2", false, false, false, false, false},
	{domain.ParcelStatusSurvey, "102", "СУУРИН БОЛОН ХЭЭРИЙН СУДАЛГАА", 2, "#414244", false, false, false, false, false},
	{domain.ParcelStatusAffectedOut, "103", "НӨЛӨӨЛЛӨӨС ГАРГАСАН", 3, "#3b82f6", true, true, false, false, true},
	{domain.ParcelStatusEvaluating, "104", "ҮНЭЛГЭЭ ХИЙГДСЭН", 4, "#f97316", false, false, true, false, false},
	{domain.ParcelStatusProcessing, "105", "ЗӨВШИЛЦӨХ ШАТАНД", 5, "#facc15", false, false, false, false, false},
	{domain.ParcelStatusRejected, "106", "ТАТГАЛЗСАН", 6, "#ef4444", false, false, false, false, true},
	{domain.ParcelStatusDecisionDraft, "107", "ЗАХИРАМЖИЙН ТӨСӨЛД", 7, "#3410b7", false, false, false, true, false},
	{domain.ParcelStatusPayment, "108", "ТӨЛБӨРИЙН ГҮЙЦЭТГЭЛ", 8, "#ce49d0", false, false, false, true, false},
	{domain.ParcelStatusReleased, "109", "ЧӨЛӨӨЛСӨН", 9, "#22c55e", true, true, false, true, false},
}

// legacyParcelStatusCodes — хуучин сангийн code → шинэ id (= code). Хуучин
// мөрийг ID-аар нь БИШ кодоор нь танина: орчин бүрт id өөр (appdb2-т id=6 нь
// «Судалгаа», локал сангуудад «Захирамжийн төсөлд»). Тоон код ('101'…'109')
// өөрөө шинэ id; seed-ийн анхны үгэн кодыг энд буулгана.
var legacyParcelStatusCodes = map[string]int{
	"pending":         domain.ParcelStatusPending,
	"processing":      domain.ParcelStatusProcessing,
	"evaluating":      domain.ParcelStatusEvaluating,
	"affected_out":    domain.ParcelStatusAffectedOut,
	"rejected":        domain.ParcelStatusRejected,
	"released":        domain.ParcelStatusReleased,
	"decision_draft":  domain.ParcelStatusDecisionDraft,
	"decisioin_draft": domain.ParcelStatusDecisionDraft,
}

// legacyParcelStatusRemap — шилжүүлэх мөрүүд: хуучин id → шинэ id. Тоон кодтой
// эсвэл legacyParcelStatusCodes-д буй, id нь кодоосоо зөрсөн мөр л орно; бусад
// (админы нэмсэн) төлөвийг хөндөхгүй. Хоёр мөр нэг шинэ id руу буувал түүх
// холилдох тул алдаа.
func legacyParcelStatusRemap(ctx context.Context, tx pgx.Tx) (olds, news []int, err error) {
	seeded := make(map[int]bool, len(parcelStatusSeeds))
	for _, s := range parcelStatusSeeds {
		seeded[s.id] = true
	}
	rows, err := tx.Query(ctx, `SELECT id, code FROM parcel_status`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	taken := map[int]string{}
	for rows.Next() {
		var id int
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, nil, err
		}
		n, ok := legacyParcelStatusCodes[code]
		if !ok {
			if v, convErr := strconv.Atoi(code); convErr == nil && seeded[v] {
				n, ok = v, true
			}
		}
		if !ok || n == id {
			continue
		}
		if prev, dup := taken[n]; dup {
			return nil, nil, fmt.Errorf("parcel_status: %q ба %q хоёулаа id=%d руу буух гэж байна", prev, code, n)
		}
		taken[n] = code
		olds, news = append(olds, id), append(news, n)
	}
	return olds, news, rows.Err()
}

// Хавсралтын бүлгүүд — шилжилтүүдэд давтагддаг.
var (
	docsContract = []string{ // Зөвшилцөх → Захирамжийн төсөлд
		"company_certificate", "work_photo", "bank_statement", "land_certificate", "contract",
		"id_card", "power_of_attorney", "cadastral_map", "socioeconomic_study", "survey_form",
		"topographic_map", "prior_notice", "valuation_report", "valuation_source",
		"meeting_minutes", "checklist", "photo_valuation", "other",
	}
	docsPayment = append(append([]string{}, docsContract...), // + төлбөр, хүлээлцэх
		"tax_receipt", "contract_closing_act", "consent_note", "engineering_blueprint",
		"death_certificate", "compensation_paid_receipt", "inheritance_agreement", "payment_table",
		"disability_certificate", "control_card", "land_handover_act",
	)
)

type parcelTransitionSeed struct {
	from, to int
	docs     []string // заавал хавсаргах баримтын document_type.type
}

var parcelWorkflowSeeds = []parcelTransitionSeed{
	{domain.ParcelStatusPending, domain.ParcelStatusSurvey, nil},                                   // Нөлөөлөлд өртсөн → Суурин болон хээрийн судалгаа
	{domain.ParcelStatusSurvey, domain.ParcelStatusEvaluating, []string{"prior_notice", "other"}},  // → Үнэлгээ хийгдсэн
	{domain.ParcelStatusSurvey, domain.ParcelStatusAffectedOut, []string{"other"}},                 // → Нөлөөллөөс гаргасан
	{domain.ParcelStatusPending, domain.ParcelStatusEvaluating, []string{"prior_notice", "other"}}, // Нөлөөлөлд өртсөн → Үнэлгээ хийгдсэн
	{domain.ParcelStatusEvaluating, domain.ParcelStatusProcessing, []string{ /* Үнэлгээ хийгдсэн → Зөвшилцөх шатанд */
		"checklist", "company_certificate", "work_photo", "tax_receipt", "land_certificate",
		"contract", "consent_note", "engineering_blueprint", "id_card", "power_of_attorney",
		"cadastral_map", "topographic_map", "valuation_report", "valuation_source", "other",
	}},
	{domain.ParcelStatusEvaluating, domain.ParcelStatusAffectedOut, []string{"other"}},               // → Нөлөөллөөс гаргасан
	{domain.ParcelStatusProcessing, domain.ParcelStatusSurvey, []string{"consent_note", "other"}},    // Зөвшилцөх → Судалгаа
	{domain.ParcelStatusProcessing, domain.ParcelStatusAffectedOut, []string{"other"}},               // → Нөлөөллөөс гаргасан
	{domain.ParcelStatusProcessing, domain.ParcelStatusRejected, []string{"consent_note", "other"}},  // → Татгалзсан
	{domain.ParcelStatusProcessing, domain.ParcelStatusDecisionDraft, docsContract},                  // → Захирамжийн төсөлд
	{domain.ParcelStatusRejected, domain.ParcelStatusAffectedOut, []string{"other", "consent_note"}}, // Татгалзсан → Нөлөөллөөс гаргасан
	{domain.ParcelStatusRejected, domain.ParcelStatusDecisionDraft, append(append([]string{}, docsContract...), // Татгалзсан → Захирамжийн төсөлд
		"tax_receipt", "consent_note", "engineering_blueprint")},
	{domain.ParcelStatusDecisionDraft, domain.ParcelStatusProcessing, []string{"contract", "consent_note", "valuation_report", "meeting_minutes", "photo_valuation"}}, // Захирамж → Зөвшилцөх
	{domain.ParcelStatusDecisionDraft, domain.ParcelStatusEvaluating, []string{"consent_note", "valuation_report", "contract", "valuation_source", "meeting_minutes", "other"}},
	{domain.ParcelStatusDecisionDraft, domain.ParcelStatusRejected, []string{"consent_note", "other"}}, // Захирамж → Татгалзсан
	{domain.ParcelStatusDecisionDraft, domain.ParcelStatusPayment, docsPayment},                        // Захирамж → Төлбөрийн гүйцэтгэл
	{domain.ParcelStatusPayment, domain.ParcelStatusReleased, without(docsPayment, // Төлбөрийн гүйцэтгэл → Чөлөөлсөн
		"tax_receipt", "engineering_blueprint", "death_certificate", "inheritance_agreement",
		"payment_table", "disability_certificate", "power_of_attorney")},
}

func without(list []string, drop ...string) []string {
	skip := make(map[string]bool, len(drop))
	for _, d := range drop {
		skip[d] = true
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		if !skip[v] {
			out = append(out, v)
		}
	}
	return out
}

// seedParcelStatusWorkflow — төлөв, шилжилт, шилжилтийн заавал хавсралт.
//
//	sync=false — шинэ сан: дутуугаа л нэмнэ (байгааг хөндөхгүй).
//	sync=true  — байгаа санг энэ бүртгэлтэй ТЭНЦҮҮЛНЭ: хуучин id-тай төлөвийг
//	             КОДООР нь танин (legacyParcelStatusRemap) шинэ id руу
//	             (нэгж талбар, түүх, ажлын урсгалтай нь)
//	             шилжүүлж, төлөвийг id-аар нь шинэчилнэ, жагсаалтад үгүй
//	             шилжилт/хавсралтыг устгана. Бусад (админы нэмсэн) төлөвийг
//	             хөндөхгүй.
//
// Хавсралт нь document_type-аас хамаарна — төрөл хараахан ороогүй (шинэ сангийн
// лавлах алхам) бол алгасагдаж, дараагийн sync алхамд орно.
func seedParcelStatusWorkflow(ctx context.Context, db *pgxpool.Pool, sync bool) error {
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		var (
			ids, sorts                                    []int
			codes, names, colors                          []string
			finals, locks, valuations, releaseds, reasons []bool
		)
		for _, s := range parcelStatusSeeds {
			ids, codes, names, sorts, colors = append(ids, s.id), append(codes, s.code), append(names, s.name), append(sorts, s.sortOrder), append(colors, s.color)
			finals, locks, valuations = append(finals, s.final), append(locks, s.locksProgress), append(valuations, s.valuation)
			releaseds, reasons = append(releaseds, s.released), append(reasons, s.reason)
		}
		var olds, news []int
		if sync {
			var err error
			if olds, news, err = legacyParcelStatusRemap(ctx, tx); err != nil {
				return err
			}
			// Хуучин мөрийн кодыг чөлөөлнө — шинэ мөр ижил кодоор (101…) орно.
			if _, err := tx.Exec(ctx, `UPDATE parcel_status SET code = left(code, 40) || '~' || id
				WHERE id = ANY($1::int[]) AND id <> ALL($2::int[])`, olds, news); err != nil {
				return err
			}
		}
		conflict := `DO NOTHING`
		if sync {
			conflict = `DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, sort_order = EXCLUDED.sort_order,
				color = EXCLUDED.color, is_final = EXCLUDED.is_final, locks_progress = EXCLUDED.locks_progress,
				is_valuation_stage = EXCLUDED.is_valuation_stage, is_released = EXCLUDED.is_released,
				requires_reason = EXCLUDED.requires_reason`
		}
		statuses, err := tx.Exec(ctx, `
			INSERT INTO parcel_status (id, code, name, sort_order, color,
			       is_final, locks_progress, is_valuation_stage, is_released, requires_reason)
			SELECT * FROM unnest($1::int[], $2::text[], $3::text[], $4::int[], $5::text[],
			       $6::bool[], $7::bool[], $8::bool[], $9::bool[], $10::bool[])
			ON CONFLICT (id) `+conflict,
			ids, codes, names, sorts, colors, finals, locks, valuations, releaseds, reasons)
		if err != nil {
			return err
		}
		var remapped int64
		if sync {
			// Хуучин id → шинэ id: FK-тэй бүх хүснэгт, дараа нь хуучин мөрийг устгана.
			for _, q := range []string{
				`UPDATE parcel t SET status = m.n FROM unnest($1::int[], $2::int[]) AS m(o, n) WHERE t.status = m.o`,
				`UPDATE parcel_status_history t SET status_id = m.n FROM unnest($1::int[], $2::int[]) AS m(o, n) WHERE t.status_id = m.o`,
				`UPDATE parcel_workflow t SET from_status_id = m.n FROM unnest($1::int[], $2::int[]) AS m(o, n) WHERE t.from_status_id = m.o`,
				`UPDATE parcel_workflow t SET to_status_id = m.n FROM unnest($1::int[], $2::int[]) AS m(o, n) WHERE t.to_status_id = m.o`,
				`DELETE FROM parcel_status WHERE id = ANY($1::int[]) AND id <> ALL($2::int[])`,
			} {
				tag, err := tx.Exec(ctx, q, olds, news)
				if err != nil {
					return err
				}
				remapped += tag.RowsAffected()
			}
		}
		// id-г шууд оруулсан тул sequence-ийг дэвшүүлнэ — эс бөгөөс админ шинэ
		// төлөв нэмэхэд id давхцана. 109 → дараагийнх нь 110.
		if _, err := tx.Exec(ctx, `SELECT setval('parcel_status_id_seq', GREATEST((SELECT MAX(id) FROM parcel_status), 1))`); err != nil {
			return err
		}

		var from, to, docFrom, docTo []int
		var docType []string
		for _, t := range parcelWorkflowSeeds {
			from, to = append(from, t.from), append(to, t.to)
			for _, d := range t.docs {
				docFrom, docTo, docType = append(docFrom, t.from), append(docTo, t.to), append(docType, d)
			}
		}
		var removed int64
		if sync {
			tag, err := tx.Exec(ctx, `
				DELETE FROM parcel_workflow pw
				 WHERE NOT EXISTS (SELECT 1 FROM unnest($1::int[], $2::int[]) AS w(f, t)
				                    WHERE w.f IS NOT DISTINCT FROM pw.from_status_id AND w.t = pw.to_status_id)`, from, to)
			if err != nil {
				return err
			}
			removed = tag.RowsAffected()
			tag, err = tx.Exec(ctx, `
				DELETE FROM parcel_workflow_required_document r
				 USING parcel_workflow pw, document_type dt
				 WHERE pw.id = r.workflow_id AND dt.id = r.document_type_id
				   AND NOT EXISTS (SELECT 1 FROM unnest($1::int[], $2::int[], $3::text[]) AS w(f, t, d)
				                    WHERE w.f = pw.from_status_id AND w.t = pw.to_status_id AND w.d = dt.type)`,
				docFrom, docTo, docType)
			if err != nil {
				return err
			}
			removed += tag.RowsAffected()
		}
		transitions, err := tx.Exec(ctx, `
			INSERT INTO parcel_workflow (from_status_id, to_status_id, sort_order)
			SELECT f, t, 0 FROM unnest($1::int[], $2::int[]) WITH ORDINALITY AS w(f, t, n) ORDER BY n
			ON CONFLICT (from_status_id, to_status_id) DO NOTHING`, from, to)
		if err != nil {
			return err
		}
		docs, err := tx.Exec(ctx, `
			INSERT INTO parcel_workflow_required_document (workflow_id, document_type_id)
			SELECT pw.id, dt.id
			  FROM unnest($1::int[], $2::int[], $3::text[]) AS w(f, t, d)
			  JOIN parcel_workflow pw ON pw.from_status_id = w.f AND pw.to_status_id = w.t
			  JOIN document_type dt ON dt.type = w.d
			ON CONFLICT DO NOTHING`, docFrom, docTo, docType)
		if err != nil {
			return err
		}
		slog.Info("parcel status workflow seeded", "sync", sync,
			"statuses", statuses.RowsAffected(), "remapped", remapped, "transitions", transitions.RowsAffected(),
			"required_documents", docs.RowsAffected(), "removed", removed)
		return nil
	})
}

// SyncParcelStatusWorkflow — байгаа санг appdb2-ийн төлөв/ажлын урсгалтай
// тэнцүүлж, төлөвийн id-г code руу (0–8 → 101–109) шилжүүлнэ (seed_history-гаар НЭГ л удаа — дараа нь админы засварыг дарахгүй).
func (s *MainSeeder) SyncParcelStatusWorkflow(ctx context.Context) error {
	return seedParcelStatusWorkflow(ctx, s.db, true)
}
