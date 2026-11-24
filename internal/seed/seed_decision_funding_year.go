package seed

import (
	"context"
	"log/slog"
)

// SeedDecisionFundingYear — migration 000065-аас өмнө үүссэн санхүүжилтийн
// холбоосын онг бөглөнө: захирамжийн төслийн «Хугацаа (он)», эс бөгөөс
// захирамжийн огнооны он, эс бөгөөс холбоос үүссэн он.
//
// Өмнө (төсөл, эх үүсвэр) давхардахгүй байсан тул он бөглөхөд шинэ дүрэм
// (төсөл, эх үүсвэр, он) зөрчигдөхгүй. Онтой мөрийг хөндөхгүй — идемпотент.
func (s *MainSeeder) SeedDecisionFundingYear(ctx context.Context) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE decision_draft_funding_source f
		   SET year = COALESCE(d.duration_year,
		                       EXTRACT(YEAR FROM d.decision_date)::int,
		                       EXTRACT(YEAR FROM f.created_at)::int)
		  FROM decision_draft d
		 WHERE d.id = f.decision_draft_id
		   AND f.year IS NULL`)
	if err != nil {
		return err
	}
	slog.Info("decision funding years backfilled", "links", tag.RowsAffected())
	return nil
}
