package legacyimport

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"
)

// paidStage — архивын хүснэгтийн дүнгээс (тухайн НЗД-ын захирамжаар олгосон)
// нөхөх олговрын олгосон шатыг бодно: олгосон ÷ батлагдсан нийт олговор.
//
//	≥ 100% (бүхэл төгрөгийн зөрүүг тооцохгүй) → бүрэн
//	≥  60%                                   → 60%
//	бусад                                    → олгоогүй
//
// 40% нь 60%-ийн ДАРААХ шат тул ганц дүнгээс тогтоогдохгүй.
func paidStage(paid, total float64) int {
	if paid <= 0 || total <= 0 {
		return 0
	}
	switch {
	case paid >= total-1:
		return domain.PaymentStageFull
	case paid*100 >= total*domain.PaymentStage60:
		return domain.PaymentStage60
	}
	return 0
}

// savePaidStage — олгосон шатыг нэгж талбарт (60 / 40 / бүрэн, огноотой)
// бичнэ; хувийг жагсаалт эндээс бодно (domain.CompensationPaidPercent).
// «Нөхөх олговор олгосон баримт» хавсаргасан нэгж талбарыг ХӨНДӨХГҮЙ — баримт
// нь эх сурвалж (syncParcelCompensationPaid).
func savePaidStage(ctx context.Context, q querier, acqID, parcelID string, paid float64, date string) error {
	var total float64
	if err := q.QueryRow(ctx, `SELECT COALESCE(SUM(amount), 0)::float8 FROM compensation
		WHERE acquisition_id = $1 AND parcel_id = $2 AND status = 'approved'`, acqID, parcelID).Scan(&total); err != nil {
		return err
	}
	stage := paidStage(paid, total)
	_, err := q.Exec(ctx, `UPDATE parcel p SET
			compensation_paid_60      = ($3::int = 60),
			compensation_paid_40      = false,
			compensation_paid_full    = ($3::int = 100),
			compensation_paid_60_at   = CASE WHEN $3::int = 60  THEN $4::date END,
			compensation_paid_40_at   = NULL,
			compensation_paid_full_at = CASE WHEN $3::int = 100 THEN $4::date END
		WHERE p.acquisition_id = $1 AND p.parcel_id = $2
		  AND NOT EXISTS (SELECT 1 FROM parcel_document pd JOIN document d ON d.id = pd.document_id
		                   WHERE pd.parcel_id = p.id AND d.payment_stage IS NOT NULL)`,
		acqID, parcelID, stage, nullStr(date))
	return err
}

// planApprovedDate — төлөвлөгөөний `approved_date` (текст) → YYYY-MM-DD;
// огноо биш эсвэл байхгүй өдөр (02-31) бол хоосон.
func planApprovedDate(raw string) string {
	d := parseDate(raw)
	if d == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", d); err != nil {
		return ""
	}
	return d
}
