package seed

import (
	"context"
	"log/slog"
)

// decisionOptionSeed — Тохиргооноос удирдагдах сонголтын анхны утга.
type decisionOptionSeed struct {
	code      string
	name      string
	sortOrder int
}

// Ажлын төрөл — захирамжийн төслийн "Ажлын төрөл" сонголтын анхны утгууд
// (Газрын тухай хуулийн 10.2-т заасан нийтийн хэрэгцээ; appdb-ийн бодит
// тохиргоо, 2026-10-07).
var decisionWorkTypeSeeds = []decisionOptionSeed{
	{"10.2.1", "Боловсролын үйлчилгээ", 1},
	{"10.2.2", "Эрүүл мэндийн үйлчилгээ", 2},
	{"10.2.3", "Нийгмийн халамжийн үйлчилгээ", 3},
	{"10.2.4", "Нийтийн номын сан, соёлын байгууллага", 4},
	{"10.2.5", "Спортын байгууламж", 5},
	{"10.2.6", "Эрчим хүч", 6},
	{"10.2.7", "Дулаан хангамж", 7},
	{"10.2.8", "Ус хангамж, ариутгах татуурга, цэвэрлэх байгууламж", 8},
	{"10.2.9", "Харилцаа холбооны шугам, сүлжээ", 9},
	{"10.2.10", "Авто болон төмөр зам, гүүр, нүхэн гарц, гарам", 10},
	{"10.2.11", "Гамшгаас хамгаалах үйл ажиллагаа", 11},
	{"10.2.12", "Гэмт хэрэгтэй тэмцэх, нийтийн хэв журам хангах үйл ажиллагаа", 12},
	{"10.2.13", "Хог хаягдлыг зохистой хаях, хадгалах, устгах", 13},
	{"10.2.14", "Тохижилт, цэцэрлэгжүүлэлт", 14},
	{"10.2.15", "Явган зам, дугуйн зам", 15},
	{"10.2.16", "Нийтийн хэрэгцээний авто зогсоол", 16},
	{"10.2.17", "Нийтийн тээвэр, дүүжин тээвэр", 17},
	{"10.2.18", "Гэр хорооллыг орон сууцжуулах", 18},
}

// Төсөв — захирамжийн төслийн "Төсөв" сонголтын анхны утгууд (appdb-ийн
// бодит тохиргоо, 2026-10-07).
var decisionBudgetSeeds = []decisionOptionSeed{
	{"state_budget", "Улсын төсөв", 1},
	{"local_budget", "Дүүргийн төсөв", 2},
	{"foreign_loan", "Гадаадын зээл, тусламж", 3},
	{"capital", "Нийслэлийн төсөв", 4},
	{"other", "Бусад", 5},
}

// SeedDecisionOptions — Ажлын төрөл / Төсвийн анхны сонголтуудыг оруулна.
// Идэмпотент: code давхцвал алгасна (хэрэглэгчийн гараар өөрчилсөн нэрийг
// дарж бичихгүй).
func (s *MainSeeder) SeedDecisionOptions(ctx context.Context) error {
	for _, table := range []struct {
		name  string
		seeds []decisionOptionSeed
	}{
		{"decision_work_type", decisionWorkTypeSeeds},
		{"decision_budget", decisionBudgetSeeds},
	} {
		for _, o := range table.seeds {
			if _, err := s.db.Exec(ctx,
				`INSERT INTO `+table.name+` (code, name, sort_order)
				 VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
				o.code, o.name, o.sortOrder); err != nil {
				return err
			}
		}
	}
	slog.Info("seeded decision draft options")
	return nil
}

// SeedDecisionReviewUnitsFromRecipients — өмнө «Хэнд» талбарт чөлөөт текстээр
// бичсэн хүлээн авагчийг «Төсөл хянах нэгж» болгон оруулж, явцын бичлэгийг
// түүнтэй холбоно (migration 000050). Байгаа нэгжийг (нэрээр) дахин үүсгэхгүй.
func (s *MainSeeder) SeedDecisionReviewUnitsFromRecipients(ctx context.Context) error {
	tag, err := s.db.Exec(ctx, `
		WITH names AS (
			SELECT DISTINCT btrim(recipient) AS name
			  FROM decision_draft_progress_history
			 WHERE progress_type = 'reviewing' AND btrim(recipient) <> ''
			   AND NOT EXISTS (SELECT 1 FROM decision_review_unit u WHERE u.name = btrim(recipient))
		), numbered AS (
			SELECT name, row_number() OVER (ORDER BY name) AS n FROM names
		)
		INSERT INTO decision_review_unit (code, name, sort_order)
		SELECT 'unit_' || (COALESCE((SELECT MAX(id) FROM decision_review_unit), 0) + n), name,
		       COALESCE((SELECT MAX(sort_order) FROM decision_review_unit), 0) + n
		  FROM numbered
		ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return err
	}
	linked, err := s.db.Exec(ctx, `
		UPDATE decision_draft_progress_history h
		   SET review_unit_id = u.id
		  FROM decision_review_unit u
		 WHERE h.review_unit_id IS NULL AND h.progress_type = 'reviewing'
		   AND u.name = btrim(h.recipient)`)
	if err != nil {
		return err
	}
	slog.Info("decision review units seeded", "units", tag.RowsAffected(), "linked", linked.RowsAffected())
	return nil
}
