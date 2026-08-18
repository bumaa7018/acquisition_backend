//go:build e2e

// authdb-ийн seeder-ийн E2E шалгалт — БОДИТ authdb дээр.
//
// Гол шаардлага: код/нэрийг кодчилсны ДАРАА ч ХЭРЭГЛЭГЧИЙН ЭРХ (эцсийн үр дүн)
// ЯГ ХЭВЭЭР гарах ёстой. Эрхийн шалгалт sd_permission-ыг permission_id / name-
// ээр холбодог тул code баганыг A001 болгосон нь үр дүнд нөлөөлөх ёсгүй.
//
// Ажиллуулах: make test-e2e
package seed

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── 1. sd_permission.code — A001 хэлбэр ──────────────────────

// effectivePermissions — хэрэглэгчийн ЭЦСИЙН эрх (UserRepo.GetPermissions-тэй
// ижил join логик): цэсний код + үйлдлийн нэр.
func effectivePermissions(t *testing.T, pool *pgxpool.Pool, username string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT DISTINCT m.code || ':' || p.name
		FROM sdplatform.sd_user u
		JOIN sdplatform.sd_user_role ur ON ur.user_id = u.user_id
		JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
		JOIN sdplatform.sd_role_permission rp ON rp.role_id = ur.role_id
		JOIN sdplatform.sd_menu m ON m.menu_id = rp.menu_id
		JOIN sdplatform.sd_menu_form mf ON mf.menu_id = m.menu_id OR mf.menu_code = m.code
		JOIN sdplatform.sd_form f ON f.form_id = mf.form_id OR f.code = mf.form_code
		JOIN sdplatform.sd_permission p ON p.permission_id = rp.permission_id
		JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
			AND (
				fp.permission_id = p.permission_id
				OR fp.permission_code = p.code
				OR fp.permission_code = p.name
			)
		WHERE u.username = $1
		ORDER BY 1`, username)
	require.NoError(t, err)
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

// Кодыг A001 хэлбэрт хөрвүүлсний дараа ч эрх ХЭВЭЭР гарна.
func TestE2E_PermissionCode_DoesNotChangeEffectivePermissions(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	var username string
	if err := pool.QueryRow(ctx, `
		SELECT u.username FROM sdplatform.sd_user u
		JOIN sdplatform.sd_user_role ur ON ur.user_id = u.user_id
		JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
		LIMIT 1`).Scan(&username); err != nil {
		t.Skip("систем 10-д ролтой хэрэглэгч байхгүй — шалгах юм байхгүй")
	}

	before := effectivePermissions(t, pool, username)
	require.NotEmpty(t, before, "шалгалт нь хоосон эрх дээр ач холбогдолгүй")

	// Кодыг seeder-ийн шинэ дүрмээр нөхнө.
	s := newTestSeeder(pool)
	for action := range permissionCodes {
		_, err := s.ensureAction(ctx, action)
		require.NoError(t, err, "action %s", action)
	}

	after := effectivePermissions(t, pool, username)
	assert.Equal(t, before, after, "эрхийн үр дүн ӨӨРЧЛӨГДӨХ ёсгүй")
}

// code баганад A001… хэлбэрийн утга бичигдэнэ, name нь үйлдлийн нэр хэвээр.
func TestE2E_PermissionCode_UsesNumericCode(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	s := newTestSeeder(pool)
	for action, wantCode := range permissionCodes {
		perm, err := s.ensureAction(ctx, action)
		require.NoError(t, err, "action %s", action)

		var code, name string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COALESCE(code,''), COALESCE(name,'') FROM sdplatform.sd_permission WHERE permission_id = $1`,
			perm.ID).Scan(&code, &name))
		assert.Equal(t, wantCode, code, "%s-ийн код", action)
		assert.Equal(t, action, name, "нэр нь үйлдлийн нэр хэвээр байх ёстой")
	}
}

// ── 2. sd_form.form_name — латин кодчилсон нэр ───────────────

// Маягтын нэр латин кодоор бичигдэнэ; ХОЛБООС нь code-оор хийгддэг тул
// эрхийн үр дүн хөндөгдөхгүй.
func TestE2E_FormName_UsesLatinCode(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	var menus int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.sd_menu WHERE code = ANY($1)`, system10MenuCodes()).Scan(&menus))
	if menus == 0 {
		t.Skip("систем 10-ийн цэс seed хийгдээгүй")
	}

	codes, names := system10FormNames()
	require.NoError(t, newTestSeeder(pool).SeedLegacySDPlatformSystem10(ctx))

	for i, code := range codes {
		var formName string
		err := pool.QueryRow(ctx,
			`SELECT COALESCE(form_name,'') FROM sdplatform.sd_form WHERE code = $1`, code).Scan(&formName)
		require.NoError(t, err, "маягт %s байхгүй", code)
		assert.Equal(t, names[i], formName, "%s-ийн маягтын нэр", code)
	}
}

// ── 3. hr_department — зөвхөн 10311 ──────────────────────────

// Код 10311 байхгүй бол нэмнэ; байвал хөндөхгүй.
func TestE2E_HRDepartment_OnlySeeds10311(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	var orgID int
	if err := pool.QueryRow(ctx,
		`SELECT id FROM sdplatform.sd_organization WHERE name = 'Нийслэлийн Газрын алба' LIMIT 1`).Scan(&orgID); err != nil {
		t.Skip("үндсэн байгууллага байхгүй — seeder алгасдаг")
	}

	// Байгаа мөрийг хадгалж, тест дараа эгүүлнэ.
	var existed bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.hr_department WHERE code = $1)`,
		hrSeedDepartment.code).Scan(&existed))
	if !existed {
		t.Cleanup(func() {
			cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = pool.Exec(cctx, `
				UPDATE sdplatform.hr_employee SET department_id = NULL
				WHERE department_id IN (SELECT department_id FROM sdplatform.hr_department WHERE code = $1)`,
				hrSeedDepartment.code)
			if _, err := pool.Exec(cctx,
				`DELETE FROM sdplatform.hr_department WHERE code = $1`, hrSeedDepartment.code); err != nil {
				t.Errorf("хэлтэс цэвэрлэгдсэнгүй: %v", err)
			}
		})
	}

	// Хуучин жишээ хэлтсүүд (ADM/GCH/…) нь ӨМНӨХ seed-ээс үлдсэн байж болно —
	// seeder тэднийг ШИНЭЭР нэмэхгүй байгааг тоогоор нь шалгана.
	legacyCodes := []string{"ADM", "GCH", "UNO", "GZZ", "SHO", "HSH", "MTH"}
	var legacyBefore, totalBefore int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.hr_department WHERE code = ANY($1)`, legacyCodes).Scan(&legacyBefore))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.hr_department`).Scan(&totalBefore))

	require.NoError(t, newTestSeeder(pool).SeedHRDepartments(ctx))

	var name string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COALESCE(name,'') FROM sdplatform.hr_department WHERE code = $1`,
		hrSeedDepartment.code).Scan(&name))
	assert.Equal(t, hrSeedDepartment.name, name)

	var legacyAfter int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.hr_department WHERE code = ANY($1)`, legacyCodes).Scan(&legacyAfter))
	assert.Equal(t, legacyBefore, legacyAfter, "хуучин жишээ хэлтэс ШИНЭЭР үүсэх ёсгүй")

	// Нийт нэмэгдэл нь ХАМГИЙН ИХ 1 (зөвхөн 10311).
	var totalAfter int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.hr_department`).Scan(&totalAfter))
	assert.LessOrEqual(t, totalAfter-totalBefore, 1, "seeder ЗӨВХӨН нэг хэлтэс нэмэх ёстой")

	// Дахин ажиллуулахад давхардахгүй.
	require.NoError(t, newTestSeeder(pool).SeedHRDepartments(ctx))
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.hr_department WHERE code = $1`,
		hrSeedDepartment.code).Scan(&n))
	assert.Equal(t, 1, n)
}

// ── 4. Хэрэглэгчийн seed — зөвхөн sd_user хоосон үед ─────────

// sd_user хоосон БИШ бол хэрэглэгч/ажилтан/хүн нэмэгдэхгүй.
func TestE2E_UserSeed_SkippedWhenUsersExist(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	var users, employees, persons int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.sd_user`).Scan(&users))
	if users == 0 {
		t.Skip("sd_user хоосон — энэ тест мөртэй санд ажиллана")
	}
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.hr_employee`).Scan(&employees))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM base.bs_person`).Scan(&persons))

	empty, err := newTestSeeder(pool).usersEmpty(ctx)
	require.NoError(t, err)
	assert.False(t, empty, "sd_user мөртэй үед usersEmpty=false байх ёстой")

	// Тоо ӨӨРЧЛӨГДӨХГҮЙ байх (seed нь алгасах).
	var users2, employees2, persons2 int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.sd_user`).Scan(&users2))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.hr_employee`).Scan(&employees2))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM base.bs_person`).Scan(&persons2))
	assert.Equal(t, users, users2)
	assert.Equal(t, employees, employees2)
	assert.Equal(t, persons, persons2)
}
