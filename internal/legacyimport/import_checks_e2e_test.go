//go:build e2e

package legacyimport

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// assertPaidAndInitialHistory — импортын дараа:
//
//  1. Батлагдсан олговортой нэгж талбар бүр хүснэгтийн дүнгээр «бүрэн»
//     олгосон (архивын мөр = захирамжаар олгосон олговор); олговоргүйд шат алга.
//  2. Нэгж талбар бүр яг НЭГ «Анхны төлөв»-тэй бөгөөд тэр нь хамгийн эхний явц.
//  3. Түүний огноо = төлөвлөгөө батлагдсан огноо (planDate; хоосон бол
//     чөлөөлөлт үүсгэсэн огноо), эцсийн төлөв түүнээс өмнө бол 1 секундын өмнө.
func assertPaidAndInitialHistory(t *testing.T, ctx context.Context, pool *pgxpool.Pool, acqs []string, planDate string) {
	t.Helper()
	const imported = `FROM parcel p WHERE p.acquisition_id = ANY($1::uuid[]) AND p.created_by = $2`
	const hasComp = ` AND EXISTS (SELECT 1 FROM compensation c WHERE c.acquisition_id = p.acquisition_id
		AND c.parcel_id = p.parcel_id AND c.status = 'approved' AND c.amount > 0)`
	withComp := count(t, ctx, pool, `SELECT count(*) `+imported+hasComp, acqs, Actor)
	require.Positive(t, withComp, "олговортой нэгж талбар байх ёстой")
	require.Equal(t, withComp, count(t, ctx, pool, `SELECT count(*) `+imported+hasComp+`
		AND p.compensation_paid_full AND NOT p.compensation_paid_60 AND NOT p.compensation_paid_40`, acqs, Actor),
		"хүснэгтийн дүнгээр бүрэн олгосон")
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) `+imported+` AND NOT `+hasComp[5:]+`
		AND (p.compensation_paid_full OR p.compensation_paid_60 OR p.compensation_paid_40)`, acqs, Actor),
		"олговоргүй нэгж талбарт олгосон шат байхгүй")

	const initial = `h.parcel_id = p.parcel_id AND h.acquisition_id = p.acquisition_id
		AND h.status_id = 101 AND h.created_by = $2 AND h.reason LIKE 'Анхны төлөв%'`
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) `+imported+`
		AND (SELECT count(*) FROM parcel_status_history h WHERE `+initial+`) <> 1`, acqs, Actor),
		"нэгж талбар бүр яг нэг «Анхны төлөв»")
	require.Zero(t, count(t, ctx, pool, `SELECT count(*) FROM parcel p
		JOIN land_acquisition la ON la.id = p.acquisition_id
		JOIN parcel_status_history h ON `+initial+`
		LEFT JOIN LATERAL (SELECT min(x.status_date) AS at FROM parcel_status_history x
		                    WHERE x.parcel_id = p.parcel_id AND x.acquisition_id = p.acquisition_id AND x.id <> h.id) nxt ON true
		WHERE p.acquisition_id = ANY($1::uuid[]) AND p.created_by = $2
		  AND h.status_date IS DISTINCT FROM CASE
		        WHEN nxt.at IS NULL THEN COALESCE($3::date::timestamptz, la.created_at)
		        ELSE LEAST(COALESCE($3::date::timestamptz, la.created_at), nxt.at - interval '1 second') END`,
		acqs, Actor, nullStr(planDate)), "«Анхны төлөв»-ийн огноо — төлөвлөгөө батлагдсан / чөлөөлөлт үүсгэсэн огноо, хамгийн эхэнд")
}
