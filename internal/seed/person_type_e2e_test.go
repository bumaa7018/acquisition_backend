//go:build e2e

// cl_person_type лавлахын seed-ийн E2E шалгалт — БОДИТ authdb дээр.
//
// Ажиллуулах:
//
//	make test-e2e
//	# эсвэл
//	E2E_AUTH_DATABASE_URL='postgres://postgres:pass@localhost:5432/authdb?sslmode=disable' \
//	  go test -tags e2e ./internal/seed/ -v
//
// Хүснэгт нь ГУС-ийн ЭЗЭМШИЛД: гурван UNIQUE (code, description, description_en)
// байдаг тул seed нь БАЙХГҮЙ үед л нэмэх, байгааг ХӨНДӨХГҮЙ байх ёстой.
package seed

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authDSN() string {
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

func openAuthPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, authDSN())
	if err != nil {
		t.Skipf("authdb холбогдсонгүй (%v) — E2E алгаслаа", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("authdb ping бүтсэнгүй (%v) — E2E алгаслаа", err)
	}
	var exists bool
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema='codelists' AND table_name='cl_person_type'
		)`).Scan(&exists))
	if !exists {
		pool.Close()
		t.Skip("codelists.cl_person_type байхгүй — authdb миграц ажиллуулна уу")
	}
	// Pool-ыг t.Cleanup-аар хаана: `defer pool.Close()` нь t.Cleanup-ЭЭС ӨМНӨ
	// гүйдэг тул эгүүлэх query-үүд хаагдсан холболт дээр чимээгүй унана.
	t.Cleanup(pool.Close)
	return pool
}

// seededCodes — тестийн бүх код (цэвэрлэхэд).
func seededCodes() []int {
	codes := make([]int, 0, len(personTypes))
	for _, pt := range personTypes {
		codes = append(codes, pt.Code)
	}
	return codes
}

// snapshot — тестийн эхэнд байсан мөрүүдийг хадгалж, дараа нь эгүүлнэ.
func snapshotPersonTypes(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	type row struct {
		code         int
		desc, descEn string
		descEnIsNull bool
	}
	rows, err := pool.Query(ctx,
		`SELECT code, description, COALESCE(description_en, ''), description_en IS NULL
		 FROM codelists.cl_person_type`)
	require.NoError(t, err)
	var before []row
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.code, &r.desc, &r.descEn, &r.descEnIsNull))
		before = append(before, r)
	}
	rows.Close()
	require.NoError(t, rows.Err())

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// Тестийн нэмсэн мөрүүдийг арилгана...
		if _, err := pool.Exec(cctx,
			`DELETE FROM codelists.cl_person_type WHERE code = ANY($1)`, seededCodes()); err != nil {
			t.Errorf("лавлах цэвэрлэгдсэнгүй: %v", err)
		}
		// ...дараа нь эхний байдлыг эгүүлнэ.
		for _, r := range before {
			var descEn any
			if !r.descEnIsNull {
				descEn = r.descEn
			}
			_, _ = pool.Exec(cctx, `
				INSERT INTO codelists.cl_person_type (code, description, description_en)
				VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`, r.code, r.desc, descEn)
		}
	})
}

// newTestSeeder — зөвхөн cl_person_type-д хүрэх seeder (repo-ууд шаардлагагүй).
func newTestSeeder(pool *pgxpool.Pool) *Seeder {
	return NewSeeder(pool)
}

// Хоосон лавлах — 10 мөр бүгд нэмэгдэнэ.
func TestE2E_SeedPersonTypes_InsertsWhenMissing(t *testing.T) {
	pool := openAuthPool(t)
	snapshotPersonTypes(t, pool)
	ctx := context.Background()

	// ГУС-ийн 10 кодыг цэвэрлээд seed-ийг ажиллуулна.
	_, err := pool.Exec(ctx, `DELETE FROM codelists.cl_person_type WHERE code = ANY($1)`, seededCodes())
	require.NoError(t, err)

	require.NoError(t, newTestSeeder(pool).seedPersonTypes(ctx))

	for _, pt := range personTypes {
		var desc, descEn string
		err := pool.QueryRow(ctx,
			`SELECT description, COALESCE(description_en,'') FROM codelists.cl_person_type WHERE code = $1`,
			pt.Code).Scan(&desc, &descEn)
		require.NoError(t, err, "код %d нэмэгдээгүй", pt.Code)
		assert.Equal(t, pt.Desc, desc)
		assert.Equal(t, pt.DescEn, descEn)
	}
}

// Дахин ажиллуулахад давхардахгүй (идемпотент).
func TestE2E_SeedPersonTypes_Idempotent(t *testing.T) {
	pool := openAuthPool(t)
	snapshotPersonTypes(t, pool)
	ctx := context.Background()

	s := newTestSeeder(pool)
	require.NoError(t, s.seedPersonTypes(ctx))

	var before int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM codelists.cl_person_type`).Scan(&before))

	// Гурван удаа дахин — мөрийн тоо өөрчлөгдөх ёсгүй.
	for i := 0; i < 3; i++ {
		require.NoError(t, s.seedPersonTypes(ctx), "%d-р дуудалт", i+1)
	}

	var after int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM codelists.cl_person_type`).Scan(&after))
	assert.Equal(t, before, after, "дахин ажиллуулахад мөр нэмэгдэх ёсгүй")
}

// ГУС-ийн БАЙГАА мөрийг ХӨНДӨХГҮЙ: ижил код өөр тайлбартай байвал дарж бичихгүй.
func TestE2E_SeedPersonTypes_DoesNotOverwriteExisting(t *testing.T) {
	pool := openAuthPool(t)
	snapshotPersonTypes(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `DELETE FROM codelists.cl_person_type WHERE code = ANY($1)`, seededCodes())
	require.NoError(t, err)

	// ГУС дээр код 30 нь ӨӨР тайлбартай байлаа гэж үзье.
	const guasDesc = "ГУС-ийн эх тайлбар 30"
	_, err = pool.Exec(ctx, `
		INSERT INTO codelists.cl_person_type (code, description, description_en)
		VALUES (30, $1, 'GUAS source 30')`, guasDesc)
	require.NoError(t, err)

	require.NoError(t, newTestSeeder(pool).seedPersonTypes(ctx))

	var desc string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT description FROM codelists.cl_person_type WHERE code = 30`).Scan(&desc))
	assert.Equal(t, guasDesc, desc, "байгаа мөр дарж бичигдэх ёсгүй")

	// Бусад 9 код нь хэвийн нэмэгдсэн байх.
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM codelists.cl_person_type WHERE code = ANY($1)`, seededCodes()).Scan(&n))
	assert.Equal(t, len(personTypes), n)
}

// Ижил ТАЙЛБАР өөр кодоор бүртгэлтэй байвал unique зөрчил гаргахгүй, алгасна.
func TestE2E_SeedPersonTypes_SkipsOnDescriptionClash(t *testing.T) {
	pool := openAuthPool(t)
	snapshotPersonTypes(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `DELETE FROM codelists.cl_person_type WHERE code = ANY($1)`, seededCodes())
	require.NoError(t, err)

	// Код 111 нь бидний код 80-ийн тайлбарыг аль хэдийн эзэмшсэн.
	clash := personTypes[7] // 80 · СӨХ
	_, err = pool.Exec(ctx, `
		INSERT INTO codelists.cl_person_type (code, description, description_en)
		VALUES (111, $1, $2)`, clash.Desc, clash.DescEn)
	require.NoError(t, err)
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(cctx, `DELETE FROM codelists.cl_person_type WHERE code = 111`)
	})

	// UNIQUE (description) зөрчилгүйгээр өнгөрөх ёстой.
	require.NoError(t, newTestSeeder(pool).seedPersonTypes(ctx))

	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM codelists.cl_person_type WHERE code = 80)`).Scan(&exists))
	assert.False(t, exists, "тайлбар давхцсан код нэмэгдэх ёсгүй")
}

// ── cl_legal_entity_type — ЗӨВХӨН хоосон үед дүүргэнэ ────────

// snapshotLegalEntityTypes — тестийн эхний байдлыг эгүүлнэ.
func snapshotLegalEntityTypes(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	type row struct {
		id   int
		desc string
	}
	rows, err := pool.Query(ctx,
		`SELECT id, COALESCE(description, '') FROM codelists.cl_legal_entity_type`)
	require.NoError(t, err)
	var before []row
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.id, &r.desc))
		before = append(before, r)
	}
	rows.Close()
	require.NoError(t, rows.Err())

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := pool.Exec(cctx, `DELETE FROM codelists.cl_legal_entity_type`); err != nil {
			t.Errorf("cl_legal_entity_type цэвэрлэгдсэнгүй: %v", err)
			return
		}
		for _, r := range before {
			_, _ = pool.Exec(cctx, `
				INSERT INTO codelists.cl_legal_entity_type (id, description)
				VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, r.id, r.desc)
		}
	})
}

// Хоосон бол 7 мөр бүгд нэмэгдэнэ.
func TestE2E_SeedLegalEntityTypes_FillsWhenEmpty(t *testing.T) {
	pool := openAuthPool(t)
	snapshotLegalEntityTypes(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `DELETE FROM codelists.cl_legal_entity_type`)
	require.NoError(t, err)

	require.NoError(t, newTestSeeder(pool).seedLegalEntityTypes(ctx))

	for _, want := range legalEntityTypes {
		var desc string
		err := pool.QueryRow(ctx,
			`SELECT COALESCE(description,'') FROM codelists.cl_legal_entity_type WHERE id = $1`,
			want.ID).Scan(&desc)
		require.NoError(t, err, "id %d нэмэгдээгүй", want.ID)
		assert.Equal(t, want.Desc, desc)
	}

	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM codelists.cl_legal_entity_type`).Scan(&n))
	assert.Equal(t, len(legalEntityTypes), n)
}

// ГАНЦ мөр байвал хүснэгтийг ОГТ хөндөхгүй.
func TestE2E_SeedLegalEntityTypes_SkipsWhenNotEmpty(t *testing.T) {
	pool := openAuthPool(t)
	snapshotLegalEntityTypes(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `DELETE FROM codelists.cl_legal_entity_type`)
	require.NoError(t, err)

	// ГУС дээр ганц мөр байлаа гэж үзье (id ч, тайлбар ч бидний жагсаалттай өөр).
	const guasDesc = "ГУС-ийн эх төрөл"
	_, err = pool.Exec(ctx,
		`INSERT INTO codelists.cl_legal_entity_type (id, description) VALUES (99, $1)`, guasDesc)
	require.NoError(t, err)

	require.NoError(t, newTestSeeder(pool).seedLegalEntityTypes(ctx))

	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM codelists.cl_legal_entity_type`).Scan(&n))
	assert.Equal(t, 1, n, "мөртэй хүснэгтэд seeder бичих ёсгүй")

	var desc string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT description FROM codelists.cl_legal_entity_type WHERE id = 99`).Scan(&desc))
	assert.Equal(t, guasDesc, desc)
}

// Дахин ажиллуулахад мөр нэмэгдэхгүй (2-р дуудалт нь "хоосон биш" тул алгасна).
func TestE2E_SeedLegalEntityTypes_Idempotent(t *testing.T) {
	pool := openAuthPool(t)
	snapshotLegalEntityTypes(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `DELETE FROM codelists.cl_legal_entity_type`)
	require.NoError(t, err)

	s := newTestSeeder(pool)
	for i := 0; i < 3; i++ {
		require.NoError(t, s.seedLegalEntityTypes(ctx), "%d-р дуудалт", i+1)
	}

	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM codelists.cl_legal_entity_type`).Scan(&n))
	assert.Equal(t, len(legalEntityTypes), n)
}
