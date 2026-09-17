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

// Байгууллагыг хүчингүй болгоход ажилтнууд нь НЭВТРЭХГҮЙ болох ёстой;
// сэргээхэд өмнөх төлөвөөрөө буцаж ирнэ.
//
// РЕГРЕСС: нэвтрэлт нь зөвхөн `ve.is_active`-ыг шалгадаг байсан тул
// байгууллагыг хүчингүй болгосон ч ажилтнууд нь нэвтэрсээр байв. Түүнчлэн
// `ValuationOrgIDForUser` нь байгууллага хүчингүй үед nil буцаадаг тул токенд
// `valuation_org` тэмдэг ТАВИГДАХГҮЙ, харин `ValuationOrgScope`-ийн allowlist
// нь ЗӨВХӨН тэр тэмдэгтэй дуудагчид үйлчилдэг — өөрөөр хэлбэл байгууллагыг
// хүчингүй болгоход ажилтан нь илүү ХЯЗГААРГҮЙ токен авч, эрх шалгадаггүй бүх
// лавлах маршрутад хүрдэг байлаа.
func TestE2E_ValuationOrgLogin_InactiveOrgBlocksEmployees(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t)
	defer pool.Close()

	repo := postgres.NewValuationOrgRepo(pool)
	login := postgres.NewValuationOrgLoginRepo(pool)

	username := "e2e_" + uuid.NewString()[:8]
	org := &domain.ValuationOrg{
		Name:       "E2E нэвтрэлт " + uuid.NewString()[:8],
		RegisterNo: uuid.NewString()[:9],
		IsActive:   true,
	}
	emps := []domain.ValuationOrgEmployee{
		{
			LastName: "Тест", FirstName: "Нэвтрэгч",
			RegisterNo: uuid.NewString()[:10],
			Username:   username,
			// Hash-ийн агуулга хамаагүй: лавлагаа нь `password <> ''`-ийг л
			// шаарддаг, нууц үгийн тулгалт нь service давхаргад хийгддэг.
			Password: "$2a$10$e2etesthashvaluenotverifiedhere000000000000000000000",
		},
		{LastName: "Тест", FirstName: "Эрхгүй", RegisterNo: uuid.NewString()[:10]},
	}
	require.NoError(t, repo.Create(ctx, org, emps))
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM user_token_epoch WHERE user_source = 'valuation_org' AND user_id IN (SELECT id FROM valuation_org_employee WHERE org_id = $1)`, org.ID)
		_, _ = pool.Exec(c, `DELETE FROM valuation_org_employee WHERE org_id = $1`, org.ID)
		_, _ = pool.Exec(c, `DELETE FROM valuation_org_profile WHERE id = $1`, org.ID)
	})

	// Нэвтрэх эрхтэй ажилтны id — энэ нь user_id мөн болно.
	var empID domain.ID
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT id FROM valuation_org_employee WHERE org_id = $1 AND username = $2`,
		org.ID, username).Scan(&empID))

	epochCount := func() int {
		var n int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_token_epoch
			 WHERE user_source = 'valuation_org' AND user_id = $1`, empID).Scan(&n))
		return n
	}
	employeeIsActive := func() bool {
		var active bool
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT is_active FROM valuation_org_employee WHERE id = $1`, empID).Scan(&active))
		return active
	}

	// 1) Байгууллага идэвхтэй — нэвтэрнэ.
	u, err := login.GetActiveByLogin(ctx, username)
	require.NoError(t, err)
	assert.Equal(t, empID, u.ID)
	require.Zero(t, epochCount(), "үүсгэхэд сесс таслагдах ёсгүй")

	// 2) Байгууллагыг ХҮЧИНГҮЙ болгоно — ажилтны мөрийг хөндөхгүй.
	org.IsActive = false
	require.NoError(t, repo.Update(ctx, org, nil))

	_, err = login.GetActiveByLogin(ctx, username)
	assert.ErrorIs(t, err, domain.ErrNotFound, "хүчингүй байгууллагын ажилтан нэвтрэхгүй")

	_, err = login.GetByID(ctx, empID)
	assert.ErrorIs(t, err, domain.ErrNotFound, "refresh-ийн зам мөн хаагдана")

	orgID, err := login.ValuationOrgIDForUser(ctx, empID)
	require.NoError(t, err)
	assert.Nil(t, orgID, "хүчингүй байгууллагын org_id токенд орохгүй")

	assert.Equal(t, 1, epochCount(), "олгогдсон access токен таслагдана")
	assert.True(t, employeeIsActive(),
		"ажилтны is_active МАССААР дарж бичигдэх ёсгүй — нэвтрэх эрх нь тооцогдоно")

	// 3) СЭРГЭЭХ — ажилтан дахин нэвтэрнэ.
	org.IsActive = true
	require.NoError(t, repo.Update(ctx, org, nil))

	u, err = login.GetActiveByLogin(ctx, username)
	require.NoError(t, err, "байгууллагыг сэргээхэд ажилтан буцаж нэвтэрнэ")
	assert.Equal(t, empID, u.ID)
}

// Тусад нь хүчингүй болгосон ажилтан нь байгууллагыг сэргээхэд ДАГАЖ
// сэргэхгүй. Энэ нь `is_active`-ыг массаар бичихийн оронд тооцоолж байгаагийн
// гол шалтгаан: массаар сэргээвэл зориудаар хасагдсан ажилтан эргэж амилна.
func TestE2E_ValuationOrgLogin_OrgRestoreKeepsIndividualDeactivation(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t)
	defer pool.Close()

	repo := postgres.NewValuationOrgRepo(pool)
	login := postgres.NewValuationOrgLoginRepo(pool)

	const pwHash = "$2a$10$e2etesthashvaluenotverifiedhere000000000000000000000"
	stayName := "e2e_stay_" + uuid.NewString()[:8]
	goneName := "e2e_gone_" + uuid.NewString()[:8]

	org := &domain.ValuationOrg{
		Name:       "E2E сэргээлт " + uuid.NewString()[:8],
		RegisterNo: uuid.NewString()[:9],
		IsActive:   true,
	}
	stay := domain.ValuationOrgEmployee{
		LastName: "Тест", FirstName: "Үлдэх",
		RegisterNo: uuid.NewString()[:10], Username: stayName, Password: pwHash,
	}
	gone := domain.ValuationOrgEmployee{
		LastName: "Тест", FirstName: "Хасагдах",
		RegisterNo: uuid.NewString()[:10], Username: goneName, Password: pwHash,
	}
	require.NoError(t, repo.Create(ctx, org, []domain.ValuationOrgEmployee{stay, gone}))
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM user_token_epoch WHERE user_source = 'valuation_org' AND user_id IN (SELECT id FROM valuation_org_employee WHERE org_id = $1)`, org.ID)
		_, _ = pool.Exec(c, `DELETE FROM valuation_org_employee WHERE org_id = $1`, org.ID)
		_, _ = pool.Exec(c, `DELETE FROM valuation_org_profile WHERE id = $1`, org.ID)
	})

	// "Үлдэх"-ийг ганцаарчлан жагсаалтад үлдээж, "Хасагдах"-ыг ЗОРИУДААР хасна.
	var stayID domain.ID
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT id FROM valuation_org_employee WHERE org_id = $1 AND username = $2`,
		org.ID, stayName).Scan(&stayID))
	stay.ID = &stayID
	require.NoError(t, repo.Update(ctx, org, []domain.ValuationOrgEmployee{stay}))

	_, err := login.GetActiveByLogin(ctx, goneName)
	require.ErrorIs(t, err, domain.ErrNotFound, "хасагдсан ажилтан нэвтрэхгүй")

	// Байгууллагыг хүчингүй болгоод СЭРГЭЭНЭ.
	org.IsActive = false
	require.NoError(t, repo.Update(ctx, org, nil))
	org.IsActive = true
	require.NoError(t, repo.Update(ctx, org, nil))

	_, err = login.GetActiveByLogin(ctx, stayName)
	assert.NoError(t, err, "үлдсэн ажилтан сэргэнэ")

	_, err = login.GetActiveByLogin(ctx, goneName)
	assert.ErrorIs(t, err, domain.ErrNotFound,
		"зориудаар хасагдсан ажилтан байгууллага сэргэхэд ДАГАЖ сэргэх ёсгүй")
}
