package seed

import (
	"context"
	"log/slog"
)

// SeedParcelStatusFlags — нэгж талбарын төлөвийн дүрмийн өгөгдмөл утга
// (migration 000049). Өмнө КОДОНД хатуу бичигдсэн дүрмийг ЯГ хуулбарлана —
// зан төлөв өөрчлөгдөхгүй:
//
//	affected_out (Нөлөөлөгдсөн гарсан) — эцсийн, явц түгжинэ, шалтгаан заавал
//	rejected     (Татгалзсан)          — эцсийн, шалтгаан заавал (явц түгжихгүй — буцааж болно)
//	released     (Чөлөөлсөн)           — эцсийн, явц түгжинэ, захирамжид холбогдоно (is_released)
//	evaluating   (Үнэлгээ хийх)        — үнэлгээний шат
//
// Төлөвийг КОДООР нь олно (орчин бүрт id өөр байж болно). seed_history-гаар
// НЭГ л удаа ажиллах тул дараа нь админы тохируулсныг дарахгүй.
func (s *MainSeeder) SeedParcelStatusFlags(ctx context.Context) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE parcel_status ps SET
			is_final           = f.is_final,
			locks_progress     = f.locks_progress,
			is_valuation_stage = f.is_valuation_stage,
			is_released        = f.is_released,
			requires_reason    = f.requires_reason
		FROM (VALUES
			('affected_out', true,  true,  false, false, true),
			('rejected',     true,  false, false, false, true),
			('released',     true,  true,  false, true,  false),
			('evaluating',   false, false, true,  false, false)
		) AS f(code, is_final, locks_progress, is_valuation_stage, is_released, requires_reason)
		WHERE ps.code = f.code`)
	if err != nil {
		return err
	}
	slog.Info("parcel status flags seeded", "statuses", tag.RowsAffected())
	return nil
}
