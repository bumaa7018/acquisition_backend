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

// Ажлын төрөл — захирамжийн төслийн "Ажлын төрөл" сонголтын анхны утгууд.
var decisionWorkTypeSeeds = []decisionOptionSeed{
	{"road", "Авто зам", 1},
	{"engineering_network", "Инженерийн шугам сүлжээ", 2},
	{"green_area", "Ногоон байгууламж", 3},
	{"public_building", "Нийтийн барилга байгууламж", 4},
	{"other", "Бусад", 99},
}

// Төсөв — одоо байгаа funding_sources.source_type-д хэрэглэгдэж ирсэн
// утгуудтай нийцүүлэв.
var decisionBudgetSeeds = []decisionOptionSeed{
	{"state_budget", "Улсын төсөв", 1},
	{"local_budget", "Орон нутгийн төсөв", 2},
	{"foreign_loan", "Гадаадын зээл, тусламж", 3},
	{"private", "Хувийн хөрөнгө оруулалт", 4},
	{"other", "Бусад", 99},
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
