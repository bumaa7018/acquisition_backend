//go:build e2e

// authdb-ийн seeder-ийн E2E шалгалт — БОДИТ authdb дээр.
//
// Гол шаардлага: seeder нь authdb-д БАЙГАА мөрийг өөрчлөхгүй, зөвхөн
// өөрт дутуу мөрийг нэмнэ. ХЭРЭГЛЭГЧИЙН ЭРХ (эцсийн үр дүн) ЯГ ХЭВЭЭР гарна.
//
// Ажиллуулах: make test-e2e
package seed

import (
	"context"
	"testing"

	"acquisition_backend/internal/config"

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

// ensureAction-ы дараа эрх ХЭВЭЭР гарна.
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

	s := newTestSeeder(pool)
	for action := range permissionCodes {
		_, err := s.ensureAction(ctx, action)
		require.NoError(t, err, "action %s", action)
	}

	after := effectivePermissions(t, pool, username)
	assert.Equal(t, before, after, "эрхийн үр дүн ӨӨРЧЛӨГДӨХ ёсгүй")
}

// Байгаа sd_permission мөрийн code-ийг ХӨНДӨХГҮЙ; A001… код зөвхөн ШИНЭ мөрөнд.
func TestE2E_PermissionCode_ExistingRowsUntouched(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	s := newTestSeeder(pool)
	for action, wantCode := range permissionCodes {
		var existed bool
		var codeBefore string
		err := pool.QueryRow(ctx, `
			SELECT COALESCE(code,'') FROM sdplatform.sd_permission
			WHERE name = $1 ORDER BY permission_id LIMIT 1`, action).Scan(&codeBefore)
		existed = err == nil

		perm, err := s.ensureAction(ctx, action)
		require.NoError(t, err, "action %s", action)

		var code, name string
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COALESCE(code,''), COALESCE(name,'') FROM sdplatform.sd_permission WHERE permission_id = $1`,
			perm.ID).Scan(&code, &name))
		if existed {
			assert.Equal(t, codeBefore, code, "%s-ийн байгаа код ӨӨРЧЛӨГДӨХ ёсгүй", action)
		} else {
			assert.Equal(t, wantCode, code, "%s-ийн шинэ мөрийн код", action)
		}
		assert.Equal(t, action, name, "нэр нь үйлдлийн нэр хэвээр байх ёстой")
	}
}

// ── 2. sd_form.form_name — байгаа нэрийг хөндөхгүй ───────────

// Байгаа маягтын нэр ХЭВЭЭР; дутуу маягт латин кодчилсон нэрээр нэмэгдэнэ.
func TestE2E_FormName_ExistingRowsUntouched(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	var menus int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM sdplatform.sd_menu WHERE code = ANY($1)`, system10MenuCodes()).Scan(&menus))
	if menus == 0 {
		t.Skip("систем 10-ийн цэс seed хийгдээгүй")
	}

	codes, names := system10FormNames()
	before := map[string]string{}
	for _, code := range codes {
		var formName string
		if err := pool.QueryRow(ctx,
			`SELECT COALESCE(form_name,'') FROM sdplatform.sd_form WHERE code = $1`, code).Scan(&formName); err == nil {
			before[code] = formName
		}
	}

	require.NoError(t, newTestSeeder(pool).ensureSystem10Structure(ctx))

	for i, code := range codes {
		var formName string
		err := pool.QueryRow(ctx,
			`SELECT COALESCE(form_name,'') FROM sdplatform.sd_form WHERE code = $1`, code).Scan(&formName)
		require.NoError(t, err, "маягт %s байхгүй", code)
		if prev, ok := before[code]; ok {
			assert.Equal(t, prev, formName, "%s-ийн байгаа нэр ӨӨРЧЛӨГДӨХ ёсгүй", code)
		} else {
			assert.Equal(t, names[i], formName, "%s-ийн шинэ маягтын нэр", code)
		}
	}
}

// ── 3. Хэрэглэгчийн seed — систем 10-д хэрэглэгч байвал алгасна ─

// Систем 10-ийн ролтой хэрэглэгч аль хэдийн байвал хэрэглэгч/ажилтан/хүн
// нэмэгдэхгүй (системийн админ ч, санхүүгийн ажилтан ч).
func TestE2E_UserSeed_SkippedWhenSystemUsersExist(t *testing.T) {
	pool := openAuthPool(t)
	ctx := context.Background()

	var hasSystemUser bool
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sdplatform.sd_user u
			JOIN sdplatform.sd_user_role ur ON ur.user_id = u.user_id
			JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
			WHERE u.deleted_at IS NULL
		)`).Scan(&hasSystemUser))
	if !hasSystemUser {
		t.Skip("систем 10-ийн ролтой хэрэглэгч байхгүй — энэ тест тийм санд ажиллана")
	}

	var users, employees, persons int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.sd_user`).Scan(&users))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.hr_employee`).Scan(&employees))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM base.bs_person`).Scan(&persons))

	require.NoError(t, newTestSeeder(pool).ensureInitialUsers(ctx,
		config.AdminConfig{Username: "e2e_admin", Email: "e2e_admin@example.com", Password: "x"},
		config.FinanceConfig{Username: "e2e_finance", Email: "e2e_finance@example.com", Password: "x"}))

	// Тоо ӨӨРЧЛӨГДӨХГҮЙ байх (seed нь алгасах).
	var users2, employees2, persons2 int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.sd_user`).Scan(&users2))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM sdplatform.hr_employee`).Scan(&employees2))
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM base.bs_person`).Scan(&persons2))
	assert.Equal(t, users, users2)
	assert.Equal(t, employees, employees2)
	assert.Equal(t, persons, persons2)
}
