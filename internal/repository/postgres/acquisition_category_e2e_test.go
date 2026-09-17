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

// Ангилал устгах хаалт.
//
// ДҮРЭМ: ерөнхий ангилалыг устгахад дэд ангилалууд нь хамт устана. Тиймээс
// өөрт нь ЭСВЭЛ дэд ангилалд нь чөлөөлөлт холбогдсон бол ЮУ Ч устгахгүй.
//
// РЕГРЕСС: хаалт нь `la.deleted_at IS NULL`-аар шүүдэг байсан тул ЗӨӨЛӨН
// устгасан чөлөөлөлтөд холбогдсон ангилал хаалтыг давж, FK зөрчилд унаж
// 500 буцаадаг байв — яг засахыг зорьсон алдаа.
func TestE2E_DeleteAcquisitionCategory(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t)
	defer pool.Close()

	repo := postgres.NewLandAcquisitionRepo(pool)

	// newCategory — ерөнхий (parent == nil) эсвэл дэд ангилал үүсгэнэ.
	newCategory := func(t *testing.T, parent *int) int {
		t.Helper()
		var id int
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO acquisition_category (name, parent_id, sort_order)
			VALUES ($1, $2, 0) RETURNING id`,
			"E2E ангилал "+uuid.NewString()[:8], parent).Scan(&id))
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(),
				`DELETE FROM acquisition_category WHERE id = $1`, id)
		})
		return id
	}

	// newAcquisition — тухайн ангилалд холбогдсон чөлөөлөлт. softDeleted бол
	// `deleted_at` тавина (мөр УСТАХГҮЙ тул FK хүчинтэй хэвээр).
	newAcquisition := func(t *testing.T, general, sub *int, softDeleted bool) uuid.UUID {
		t.Helper()
		id := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO land_acquisition
				(id, plan_code, acquisition_name, status, general_category_id, sub_category_id,
				 deleted_at, created_at, updated_at)
			VALUES ($1, $2, 'E2E ангилалын тест', $3, $4, $5,
			        CASE WHEN $6 THEN NOW() ELSE NULL END, NOW(), NOW())`,
			id, "E2E-"+uuid.NewString()[:8], domain.LandAcquisitionStatusNew,
			general, sub, softDeleted)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(),
				`DELETE FROM land_acquisition WHERE id = $1`, id)
		})
		return id
	}

	exists := func(t *testing.T, id int) bool {
		t.Helper()
		var ok bool
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM acquisition_category WHERE id = $1)`, id).Scan(&ok))
		return ok
	}

	t.Run("холбоосгүй ангилал дэд ангилалтайгаа хамт устана", func(t *testing.T) {
		parent := newCategory(t, nil)
		child := newCategory(t, &parent)

		require.NoError(t, repo.DeleteAcquisitionCategory(ctx, parent))
		assert.False(t, exists(t, parent), "ерөнхий ангилал устана")
		assert.False(t, exists(t, child), "дэд ангилал хамт устана")
	})

	t.Run("өөрт нь холбогдсон бол устгахгүй", func(t *testing.T) {
		parent := newCategory(t, nil)
		newAcquisition(t, &parent, nil, false)

		err := repo.DeleteAcquisitionCategory(ctx, parent)
		assert.ErrorIs(t, err, domain.ErrCategoryInUse)
		assert.True(t, exists(t, parent))
	})

	t.Run("ДЭД ангилалд холбогдсон бол ерөнхийг нь ч устгахгүй", func(t *testing.T) {
		parent := newCategory(t, nil)
		child := newCategory(t, &parent)
		newAcquisition(t, &parent, &child, false)

		err := repo.DeleteAcquisitionCategory(ctx, parent)
		assert.ErrorIs(t, err, domain.ErrCategoryInUse)
		assert.True(t, exists(t, parent), "ерөнхий ангилал үлдэнэ")
		assert.True(t, exists(t, child), "дэд ангилал ч үлдэнэ")
	})

	t.Run("ЗӨӨЛӨН устгасан чөлөөлөлт ч хаалтыг барина", func(t *testing.T) {
		parent := newCategory(t, nil)
		newAcquisition(t, &parent, nil, true)

		// FK нь `deleted_at`-ыг мэддэггүй тул энэ лавлагаа хүчинтэй хэвээр.
		// 500 биш, ойлгомжтой 409 очих ёстой.
		err := repo.DeleteAcquisitionCategory(ctx, parent)
		assert.ErrorIs(t, err, domain.ErrCategoryInUse)
		assert.True(t, exists(t, parent))
	})

	t.Run("AcquisitionCount нь хаалттай ижил дүрмээр тоолно", func(t *testing.T) {
		parent := newCategory(t, nil)
		child := newCategory(t, &parent)
		newAcquisition(t, &parent, nil, false)   // идэвхтэй
		newAcquisition(t, &parent, &child, true) // зөөлөн устгасан
		newAcquisition(t, nil, &child, false)    // зөвхөн дэд ангилалаар

		list, err := repo.ListAcquisitionCategories(ctx, nil)
		require.NoError(t, err)

		var got *domain.AcquisitionCategory
		for _, c := range list {
			if c.ID == parent {
				got = c
				break
			}
		}
		require.NotNil(t, got, "үүсгэсэн ангилал жагсаалтад байх ёстой")
		assert.Equal(t, 3, got.AcquisitionCount,
			"зөөлөн устгасан болон дэд ангилалаар холбогдсон нь мөн тоологдоно")
		assert.Equal(t, 1, got.SubCount)

		// Тоо нь хаалттай ЗӨРӨХГҮЙ: 0 биш учраас устгал таслагдана.
		assert.ErrorIs(t, repo.DeleteAcquisitionCategory(ctx, parent), domain.ErrCategoryInUse)
	})
}
