//go:build e2e

// base.bs_person."type" кодын E2E шалгалт — БОДИТ authdb дээр.
//
// bs_person."type" нь codelists.cl_person_type(code) рүү FK-аар холбогддог ба
// ГУС-ийн лавлахын код 10-аар үсэрдэг (10…100). Хуучин код 1/2-ыг бичвэл
// жинхэнэ санд FK зөрчил болно.
//
// БҮЛЭГЛЭЛТ: 10, 20, 30 → иргэн (individual) · бусад бүх код → хуулийн этгээд.
// Тест нь бичилт БОДИТ кодоор хийгдэж, уншилт нь эргээд зөв төрөл болж
// хөрвөж байгааг (round-trip) батална.
//
// Ажиллуулах: make test-e2e
package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsnFor := func() string {
		if v := os.Getenv("E2E_AUTH_DATABASE_URL"); v != "" {
			return v
		}
		get := func(key, def string) string {
			if v := os.Getenv(key); v != "" {
				return v
			}
			return def
		}
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			get("AUTH_DB_USER", "postgres"), get("AUTH_DB_PASSWORD", "postgres"),
			get("AUTH_DB_HOST", "localhost"), get("AUTH_DB_PORT", "5432"),
			get("AUTH_DB_NAME", "authdb"), get("AUTH_DB_SSLMODE", "disable"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsnFor())
	if err != nil {
		t.Skipf("authdb холбогдсонгүй (%v) — E2E алгаслаа", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("authdb ping бүтсэнгүй (%v) — E2E алгаслаа", err)
	}
	// Pool-ыг t.Cleanup-аар хаана: `defer pool.Close()` нь t.Cleanup-ЭЭС ӨМНӨ
	// гүйдэг тул цэвэрлэх query-үүд хаагдсан холболт дээр чимээгүй унана.
	// Энд ХАМГИЙН ЭХЭНД бүртгэснээр LIFO-гоор ХАМГИЙН СҮҮЛД хаагдана.
	t.Cleanup(pool.Close)
	return pool
}

// ensurePersonTypeCodes — тестэд шаардлагатай лавлахын кодыг бэлдэнэ
// (seeder-ийн ажиллаагүй орчинд ч тест ажиллана).
func ensurePersonTypeCodes(t *testing.T, pool *pgxpool.Pool, codes ...int) {
	t.Helper()
	ctx := context.Background()
	var added []int
	for _, code := range codes {
		tag, err := pool.Exec(ctx, `
			INSERT INTO codelists.cl_person_type (code, description, description_en)
			SELECT $1::int, 'E2E төрөл ' || $1::text, 'E2E type ' || $1::text
			WHERE NOT EXISTS (SELECT 1 FROM codelists.cl_person_type WHERE code = $1::int)`, code)
		require.NoError(t, err)
		if tag.RowsAffected() > 0 {
			added = append(added, code)
		}
	}
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, code := range added {
			if _, err := pool.Exec(cctx, `DELETE FROM codelists.cl_person_type WHERE code = $1`, code); err != nil {
				t.Errorf("лавлахын код %d цэвэрлэгдсэнгүй: %v", code, err)
			}
		}
	})
}

func strp(s string) *string { return &s }

// uniqueSuffix — person_register давхардахгүй байх (5 тэмдэгт).
func uniqueSuffix() string {
	return uuid.NewString()[:5]
}

// createPerson — HRRepo-гоор хүн бүртгэж, устгалтыг бүртгэнэ.
func createPerson(t *testing.T, repo *postgres.HRRepo, pool *pgxpool.Pool, p *domain.Person) {
	t.Helper()
	require.NoError(t, repo.CreatePerson(context.Background(), p))
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(cctx, `DELETE FROM base.bs_person WHERE person_id = $1`, p.ID); err != nil {
			t.Errorf("хүн %v цэвэрлэгдсэнгүй: %v", p.ID, err)
		}
	})
}

// rawPersonType — DB дээр БОДИТООР бичигдсэн код.
func rawPersonType(t *testing.T, pool *pgxpool.Pool, id domain.ID) int {
	t.Helper()
	var code int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT "type" FROM base.bs_person WHERE person_id = $1`, id).Scan(&code))
	return code
}

// Иргэн → код 10 (хуучин 1 БИШ).
func TestE2E_Person_CitizenWritesCode10(t *testing.T) {
	pool := authPool(t)
	ensurePersonTypeCodes(t, pool, 10, 30)
	repo := postgres.NewHRRepo(pool)

	p := &domain.Person{
		PersonType: "citizen",
		RegisterNo: "E2E10" + uniqueSuffix(),
		LastName:   strp("Дорж"),
		FirstName:  strp("Бат"),
	}
	createPerson(t, repo, pool, p)

	assert.Equal(t, 10, rawPersonType(t, pool, p.ID), "иргэн нь код 10-аар бичигдэх ёстой")

	got, err := repo.GetPersonByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, "citizen", got.PersonType)
	require.NotNil(t, got.LastName)
	assert.Equal(t, "Дорж", *got.LastName)
}

// Хуулийн этгээд → код 40 (иргэний бүлэг 10/20/30-аас ГАДНА).
func TestE2E_Person_LegalWritesLegalCode(t *testing.T) {
	pool := authPool(t)
	ensurePersonTypeCodes(t, pool, 10, 40)
	repo := postgres.NewHRRepo(pool)

	p := &domain.Person{
		PersonType: "legal",
		RegisterNo: "E2E40" + uniqueSuffix(),
		LegalName:  strp("Тест ХХК"),
	}
	createPerson(t, repo, pool, p)

	assert.Equal(t, 40, rawPersonType(t, pool, p.ID), "хуулийн этгээд нь иргэний бүлгээс гадна кодоор бичигдэх ёстой")

	got, err := repo.GetPersonByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, "legal", got.PersonType)
	require.NotNil(t, got.LegalName)
	assert.Equal(t, "Тест ХХК", *got.LegalName)
	assert.Nil(t, got.LastName, "хуулийн этгээдэд овог байхгүй")
}

// Иргэний бүлгээс ГАДНА бүх код (40…100) "legal" болж уншигдана.
func TestE2E_Person_AllLegalCodesReadAsLegal(t *testing.T) {
	pool := authPool(t)
	legalCodes := []int{40, 50, 60, 70, 80, 90, 100}
	ensurePersonTypeCodes(t, pool, legalCodes...)
	repo := postgres.NewHRRepo(pool)
	ctx := context.Background()

	for _, code := range legalCodes {
		// Мөрийг ШУУД тэр кодоор үүсгэнэ (ГУС-аас ирсэн өгөгдлийг дуурайв).
		var id domain.ID
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO base.bs_person ("type", "name", person_register, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW()) RETURNING person_id`,
			code, fmt.Sprintf("E2E ХХК %d", code), fmt.Sprintf("E2E%05d", code)).Scan(&id))
		t.Cleanup(func() {
			cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := pool.Exec(cctx, `DELETE FROM base.bs_person WHERE person_id = $1`, id); err != nil {
				t.Errorf("хүн %v цэвэрлэгдсэнгүй: %v", id, err)
			}
		})

		got, err := repo.GetPersonByID(ctx, id)
		require.NoError(t, err, "код %d", code)
		assert.Equal(t, "legal", got.PersonType, "код %d нь хуулийн этгээд байх ёстой", code)
	}
}

// Иргэний бүлэг (10, 20, 30) "citizen" болж уншигдана.
func TestE2E_Person_CitizenCodesReadAsCitizen(t *testing.T) {
	pool := authPool(t)
	citizenCodes := []int{10, 20, 30}
	ensurePersonTypeCodes(t, pool, citizenCodes...)
	repo := postgres.NewHRRepo(pool)
	ctx := context.Background()

	for _, code := range citizenCodes {
		var id domain.ID
		require.NoError(t, pool.QueryRow(ctx, `
			INSERT INTO base.bs_person ("type", "name", first_name, person_register, created_at, updated_at)
			VALUES ($1, 'Дорж', 'Бат', $2, NOW(), NOW()) RETURNING person_id`,
			code, fmt.Sprintf("E2EC%04d", code)).Scan(&id))
		t.Cleanup(func() {
			cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := pool.Exec(cctx, `DELETE FROM base.bs_person WHERE person_id = $1`, id); err != nil {
				t.Errorf("хүн %v цэвэрлэгдсэнгүй: %v", id, err)
			}
		})

		got, err := repo.GetPersonByID(ctx, id)
		require.NoError(t, err, "код %d", code)
		assert.Equal(t, "citizen", got.PersonType, "код %d нь иргэн байх ёстой", code)
	}
}

// ── bs_person УСТГАХ БОЛОМЖГҮЙ ───────────────────────────────

// Хүний мэдээллийг устгах оролдлого — мөр ХЭВЭЭР үлдэнэ.
func TestE2E_Person_DeleteForbidden(t *testing.T) {
	pool := authPool(t)
	ensurePersonTypeCodes(t, pool, 10)
	repo := postgres.NewHRRepo(pool)
	ctx := context.Background()

	p := &domain.Person{
		PersonType: "citizen",
		RegisterNo: "E2EDL" + uniqueSuffix(),
		LastName:   strp("Дорж"),
		FirstName:  strp("Бат"),
	}
	createPerson(t, repo, pool, p)

	err := repo.DeletePerson(ctx, p.ID)
	assert.ErrorIs(t, err, domain.ErrPersonDeleteForbidden)

	// Мөр устаагүй байх ёстой.
	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM base.bs_person WHERE person_id = $1)`, p.ID).Scan(&exists))
	assert.True(t, exists, "хүний мөр устсан байна")
}

// ── DELETE нь систем 10-аар хязгаарлагдана ───────────────────

// Систем 10-д ХАМААРАХГҮЙ дүрийг устгах боломжгүй.
func TestE2E_Role_DeleteScopedToSystem10(t *testing.T) {
	pool := authPool(t)
	repo := postgres.NewRoleRepo(pool)
	ctx := context.Background()

	// ӨӨР системийн дүр (sd_system_role-д систем 10-гүй).
	var foreignID domain.ID
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_role (code, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW()) RETURNING role_id`,
		"E2E"+uniqueSuffix(), "e2e_foreign_role_"+uniqueSuffix()).Scan(&foreignID))
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(cctx, `DELETE FROM sdplatform.sd_role WHERE role_id = $1`, foreignID); err != nil {
			t.Errorf("дүр цэвэрлэгдсэнгүй: %v", err)
		}
	})

	// Устгах оролдлого — олдоогүй гэж буцаах ба мөр ХЭВЭЭР байх ёстой.
	assert.ErrorIs(t, repo.Delete(ctx, foreignID), domain.ErrNotFound,
		"өөр системийн дүр устгагдах ёсгүй")

	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_role WHERE role_id = $1)`, foreignID).Scan(&exists))
	assert.True(t, exists, "өөр системийн дүр устсан байна")
}

// Систем 10-ийн дүр нь хэвийн устана (зан төлөв хэвээр).
func TestE2E_Role_DeleteAllowedForSystem10(t *testing.T) {
	pool := authPool(t)
	repo := postgres.NewRoleRepo(pool)
	ctx := context.Background()

	var ourID domain.ID
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_role (code, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW()) RETURNING role_id`,
		"E2E"+uniqueSuffix(), "e2e_our_role_"+uniqueSuffix()).Scan(&ourID))
	_, err := pool.Exec(ctx, `
		INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
		VALUES (10, $1, NOW())`, ourID)
	require.NoError(t, err)
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cctx, `DELETE FROM sdplatform.sd_system_role WHERE role_id = $1`, ourID)
		_, _ = pool.Exec(cctx, `DELETE FROM sdplatform.sd_role WHERE role_id = $1`, ourID)
	})

	require.NoError(t, repo.Delete(ctx, ourID), "систем 10-ийн дүр устах ёстой")

	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_role WHERE role_id = $1)`, ourID).Scan(&exists))
	assert.False(t, exists)
}

// Систем 10-д хэрэглэгддэггүй эрх (permission) устгагдахгүй.
func TestE2E_Permission_DeleteScopedToSystem10(t *testing.T) {
	pool := authPool(t)
	repo := postgres.NewPermissionRepo(pool)
	ctx := context.Background()

	// Ямар ч ролид холбогдоогүй, өөр системийн эрх.
	var foreignID domain.ID
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_permission (code, name, created_at)
		VALUES ($1, $2, NOW()) RETURNING permission_id`,
		"E2E"+uniqueSuffix(), "e2e_foreign_perm_"+uniqueSuffix()).Scan(&foreignID))
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(cctx,
			`DELETE FROM sdplatform.sd_permission WHERE permission_id = $1`, foreignID); err != nil {
			t.Errorf("эрх цэвэрлэгдсэнгүй: %v", err)
		}
	})

	assert.ErrorIs(t, repo.Delete(ctx, foreignID), domain.ErrNotFound,
		"систем 10-д хамааралгүй эрх устгагдах ёсгүй")

	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_permission WHERE permission_id = $1)`,
		foreignID).Scan(&exists))
	assert.True(t, exists, "өөр системийн эрх устсан байна")
}
