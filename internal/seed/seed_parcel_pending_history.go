package seed

import (
	"context"
	"log/slog"
)

// SeedParcelPendingHistory — «Хүлээгдэж буй» (code=pending) түүхгүй бүх нэгж
// талбарт АНХНЫ төлөвийн түүх нэмнэ. Огноо нь чөлөөлөлт үүсгэсэн өдөр; түүх
// түүнээс өмнө эхэлсэн (хуучин мэдээлэл) бол эхний бичлэгээс 1 секундын өмнө —
// дараалал эвдэрэхгүй. Хуучин импорт зөвхөн эцсийн төлөвийг бичдэг байсныг нөхнө.
func (s *MainSeeder) SeedParcelPendingHistory(ctx context.Context) error {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO parcel_status_history (parcel_id, acquisition_id, status_id, status_date, created_by, reason)
		SELECT p.parcel_id, p.acquisition_id, ps.id,
		       LEAST(la.created_at, COALESCE(first.at - interval '1 second', la.created_at)),
		       'system', 'Анхны төлөв'
		  FROM parcel p
		  JOIN land_acquisition la ON la.id = p.acquisition_id
		  JOIN parcel_status ps ON ps.code = 'pending'
		  LEFT JOIN LATERAL (
		        SELECT MIN(h.status_date) AS at FROM parcel_status_history h
		         WHERE h.parcel_id = p.parcel_id AND h.acquisition_id = p.acquisition_id
		  ) first ON true
		 WHERE NOT EXISTS (
		        SELECT 1 FROM parcel_status_history h
		         WHERE h.parcel_id = p.parcel_id AND h.acquisition_id = p.acquisition_id
		           AND h.status_id = ps.id)`)
	if err != nil {
		return err
	}
	slog.Info("parcel pending history seeded", "rows", tag.RowsAffected())
	return nil
}
