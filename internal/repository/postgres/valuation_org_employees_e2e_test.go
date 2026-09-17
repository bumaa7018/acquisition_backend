//go:build e2e

package postgres_test

import (
	"context"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Байгууллагыг ХООСОН ажилтны жагсаалттай хадгалахад байгаа ажилтнууд
// идэвхтэй хэвээр үлдэх ёстой.
//
// РЕГРЕСС: өмнө нь `employees: []` нь "бүгдийг хас" гэж тайлбарлагдаж, тухайн
// байгууллагын БҮХ ажилтныг `is_active=false` болгоод сессийг нь тасалдаг
// байв. Дэлгэцийн маягт дэлгэрэнгүй ачаалагдахаас өмнө хадгалахад яг хоосон
// массив илгээдэг тул нэг даралт бүх ажилтныг системээс гаргаж, `is_active`
// шаарддаг нэвтрэлт ч боломжгүй болгож байлаа.
func TestE2E_ValuationOrgUpdate_EmptyEmployeeListKeepsEmployees(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t)
	defer pool.Close()

	repo := postgres.NewValuationOrgRepo(pool)

	org := &domain.ValuationOrg{
		Name:       "E2E үнэлгээ " + uuid.NewString()[:8],
		RegisterNo: uuid.NewString()[:9],
		IsActive:   true,
	}
	emps := []domain.ValuationOrgEmployee{
		{LastName: "Тест", FirstName: "Ажилтан1", RegisterNo: uuid.NewString()[:10]},
		{LastName: "Тест", FirstName: "Ажилтан2", RegisterNo: uuid.NewString()[:10]},
	}
	require.NoError(t, repo.Create(ctx, org, emps))
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM valuation_org_employee WHERE org_id = $1`, org.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM user_token_epoch WHERE user_source = 'valuation_org' AND user_id IN (SELECT id FROM valuation_org_employee WHERE org_id = $1)`, org.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM valuation_org_profile WHERE id = $1`, org.ID)
	})

	activeCount := func() int {
		var n int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM valuation_org_employee WHERE org_id = $1 AND is_active`,
			org.ID).Scan(&n))
		return n
	}
	epochCount := func() int {
		var n int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_token_epoch
			 WHERE user_source = 'valuation_org'
			   AND user_id IN (SELECT id FROM valuation_org_employee WHERE org_id = $1)`,
			org.ID).Scan(&n))
		return n
	}

	require.Equal(t, 2, activeCount(), "үүсгэсний дараа хоёр ажилтан идэвхтэй байх ёстой")
	require.Zero(t, epochCount(), "үүсгэхэд сесс таслагдах ёсгүй")

	// 1) ХООСОН жагсаалт — зөвхөн байгууллагын мэдээлэл засагдана.
	org.Phone = "99119911"
	require.NoError(t, repo.Update(ctx, org, []domain.ValuationOrgEmployee{}))
	assert.Equal(t, 2, activeCount(), "хоосон жагсаалт ажилтныг унтраах ёсгүй")
	assert.Zero(t, epochCount(), "хоосон жагсаалт сессийг таслах ёсгүй")

	// 2) nil жагсаалт — мөн адил хөндөхгүй (өмнөх зан төлөв хэвээр).
	org.Phone = "99229922"
	require.NoError(t, repo.Update(ctx, org, nil))
	assert.Equal(t, 2, activeCount(), "nil жагсаалт ажилтныг унтраах ёсгүй")
	assert.Zero(t, epochCount())

	// 3) ЗОРИУДААР хасах — нэг ажилтныг л үлдээвэл нөгөө нь идэвхгүй болж,
	//    сесс нь таслагдана. Энэ зан төлөв ХЭВЭЭР байх ёстой.
	var keep domain.ValuationOrgEmployee
	var keepID domain.ID
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT id, last_name, first_name, register_no
		  FROM valuation_org_employee
		 WHERE org_id = $1 AND is_active
		 ORDER BY id
		 LIMIT 1`, org.ID).
		Scan(&keepID, &keep.LastName, &keep.FirstName, &keep.RegisterNo))
	keep.ID = &keepID
	require.NoError(t, repo.Update(ctx, org, []domain.ValuationOrgEmployee{keep}))
	assert.Equal(t, 1, activeCount(), "жагсаалтад ороогүй ажилтан идэвхгүй болно")
	assert.Equal(t, 1, epochCount(), "хасагдсан ажилтны сесс таслагдана")
}
