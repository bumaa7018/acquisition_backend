package seed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultPermissions — системийн бүх эрх "нөөц:үйлдэл" хэлбэрээр.
// sdplatform-д нөөц нь роль дээр, үйлдэл нь эрх дээр бүртгэгддэг тул эдгээр нь
// grantLevel()-ээр (role=<нөөц>_<түвшин>, permission=<үйлдэл>) болж задарна.
var defaultPermissions = []string{
	"users:read", "users:create", "users:update", "users:delete",
	"roles:read", "roles:create", "roles:update", "roles:delete",
	"permissions:read", "permissions:create", "permissions:delete",
	"admin:read", "admin:create", "admin:update", "admin:delete",
	"hr:read", "hr:create", "hr:update", "hr:delete",
	"audit:read",
	"land:read", "land:create", "land:update", "land:delete",
	// Нөхөх олговрын (үнэлгээний) эрхүүд — гадаад ролиуд ЗӨВХӨН эдгээртэй.
	"compensation:read", "compensation:create", "compensation:update",
	// Захирамжийн төслийн эрхүүд.
	"decision:read", "decision:create", "decision:update", "decision:delete",
}

// seniorSpecialistPermissions / employeePermissions — дотоод түвшингүүдийн багц.
var seniorSpecialistPermissions = []string{
	"land:read", "land:create", "land:update",
	"decision:read", "decision:create", "decision:update",
	"users:read",
}

var employeePermissions = []string{"land:read"}

func (s *Seeder) SeedHRPermissions(ctx context.Context) error {
	if err := s.grantLevel(ctx, "admin", []string{"hr:read", "hr:create", "hr:update", "hr:delete"}); err != nil {
		return err
	}
	slog.Info("hr permissions granted", "level", "admin")
	return nil
}

// SeedHRBackfill — ажилтны бүртгэлгүй хэрэглэгч бүрт хүн/ажилтныг үүсгэж,
// hr_employee.user_id-аар холбоно (sdplatform-д хэрэглэгч↔ажилтны холбоос
// яг энэ багана дээр байдаг).
//
// bs_person.person_register нь varchar(10) тул регистрийг хэрэглэгчийн
// дугаараас богиноор («U<user_id>») үүсгэнэ.
func (s *Seeder) SeedHRBackfill(ctx context.Context) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const orgName = "Нийслэлийн Газрын алба"
	var orgID domain.ID
	if err := tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO sdplatform.sd_organization (name)
			SELECT LEFT($1::text, 250)
			WHERE NOT EXISTS (SELECT 1 FROM sdplatform.sd_organization WHERE name = LEFT($1::text, 250))
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM sdplatform.sd_organization WHERE name = LEFT($1::text, 250)
		LIMIT 1`, orgName).Scan(&orgID); err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT u.user_id, COALESCE(u.firstname, ''), COALESCE(u.lastname, ''), COALESCE(u.email, '')
		FROM sdplatform.sd_user u
		WHERE u.deleted_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM sdplatform.hr_employee e
			WHERE e.user_id = u.user_id AND COALESCE(e.is_active, 1) <> 0
		)`)
	if err != nil {
		return err
	}
	type backfillUser struct {
		id        domain.ID
		firstName string
		lastName  string
		email     string
	}
	var users []backfillUser
	for rows.Next() {
		var u backfillUser
		if err := rows.Scan(&u.id, &u.firstName, &u.lastName, &u.email); err != nil {
			rows.Close()
			return err
		}
		users = append(users, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	const defaultPosition = "Ажилтан"
	var posID domain.ID
	if err := tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO sdplatform.hr_position (name, created_at, updated_at)
			SELECT LEFT($1, 250), NOW(), NOW()
			WHERE NOT EXISTS (SELECT 1 FROM sdplatform.hr_position WHERE name = LEFT($1, 250))
			RETURNING position_id
		)
		SELECT position_id FROM ins
		UNION ALL
		SELECT position_id FROM sdplatform.hr_position WHERE name = LEFT($1, 250)
		LIMIT 1`, defaultPosition).Scan(&posID); err != nil {
		return err
	}

	for _, u := range users {
		register := fmt.Sprintf("U%d", u.id)
		var personID domain.ID
		if err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO base.bs_person ("type", "name", first_name, person_register, email_address, created_at, updated_at)
				SELECT 1, LEFT($2, 250), LEFT($3, 50), LEFT($1, 10), LEFT($4, 256), NOW(), NOW()
				WHERE NOT EXISTS (SELECT 1 FROM base.bs_person WHERE person_register = LEFT($1, 10))
				RETURNING person_id
			)
			SELECT person_id FROM ins
			UNION ALL
			SELECT person_id FROM base.bs_person WHERE person_register = LEFT($1, 10)
			LIMIT 1`, register, u.lastName, u.firstName, u.email).Scan(&personID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO sdplatform.hr_employee (
				person_id, position_id, user_id, email,
				firstname, lastname, is_active, created_at, updated_at
			)
			SELECT $1::int4, $2::int4, $3::int4, LEFT($4, 250), LEFT($5, 250), LEFT($6, 250), 1, NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM sdplatform.hr_employee
				WHERE person_id = $1 AND COALESCE(is_active, 1) <> 0
			)`, personID, posID, u.id, u.email, u.firstName, u.lastName); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("hr backfill applied", "users", len(users), "organization_id", orgID)
	return nil
}

// hrSampleDepartments — Нийслэлийн Газрын албаны бүтэц (жишээ өгөгдөл).
// parent_id хэрэглэхгүй: UI-ийн "Алба, хэлтэс" форм нь зөвхөн
// organization_id/name/code/sort_order-ыг тохируулдаг тул бүтцийг хавтгай
// байлгав — үгүй бол UI-аас засах боломжгүй мөр үүснэ.
var hrSampleDepartments = []struct {
	code string
	name string
}{
	{"ADM", "Захиргаа, удирдлагын хэлтэс"},
	{"GCH", "Газар чөлөөлөлтийн хэлтэс"},
	{"UNO", "Үнэлгээ, нөхөх олговрын хэлтэс"},
	{"GZZ", "Геодези, зураг зүйн хэлтэс"},
	{"SHO", "Санхүү, хөрөнгө оруулалтын хэлтэс"},
	{"HSH", "Хяналт, шинжилгээний хэлтэс"},
	{"MTH", "Мэдээллийн технологийн хэлтэс"},
}

// Албан тушаал → хэлтсийн харгалзаа. Тохирохгүй/хоосон тушаал нь
// hrDefaultDepartmentCode-д унана.
var hrPositionDepartment = map[string]string{
	"Ахлах мэргэжилтэн":      "GCH",
	"Ажилтан":                "ADM",
	"Санхүүгийн мэргэжилтэн": "SHO",
	"МИКА": "UNO",
	"Мэргэжлийн байгууллага":             "UNO",
	"Хөндлөнгийн мэргэжлийн байгууллага": "UNO",
}

const hrDefaultDepartmentCode = "ADM"

// SeedHRDepartments — "Алба, хэлтэс" лавлахыг жишээ өгөгдлөөр бөглөж, ажилтан
// бүрийг албан тушаалынх нь дагуу хэлтэст холбоно.
//
// Шалтгаан: hr backfill (000019) нь ажилтныг зөвхөн organization/position-той
// үүсгэсэн ба department_id-г NULL үлдээсэн. Улмаар "Алба, хэлтэс" хуудас
// хоосон (байсан 7 мөр нь E2E тестийн үлдэгдэл, бүгд deleted_at-тай) ба
// "Ажилтан" хуудсанд хэлтсийн мэдээлэл гарахгүй байв.
//
// Идемпотент: хэлтсийг (organization_id, name)-аар, ажилтны холбоосыг
// department_id IS NULL шалгуураар л хийнэ — гараар зассан өгөгдлийг
// дарж бичихгүй.
// SeedHRDepartments — "Алба, хэлтэс" лавлахыг жишээ өгөгдлөөр бөглөж, хэлтэсгүй
// ажилтан бүрийг албан тушаалынх нь дагуу хэлтэст холбоно.
//
// Ажилтны БАЙГУУЛЛАГА нь хэлтсээр дамждаг (hr_employee-д байгууллагын багана
// байхгүй) тул энэ алхам нь ажилтныг байгууллагатай холбох цорын ганц зам.
func (s *Seeder) SeedHRDepartments(ctx context.Context) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const orgName = "Нийслэлийн Газрын алба"
	var orgID domain.ID
	if err := tx.QueryRow(ctx,
		`SELECT id FROM sdplatform.sd_organization WHERE name = $1 LIMIT 1`, orgName).Scan(&orgID); err != nil {
		slog.Warn("hr departments skipped: base organization not found", "error", err)
		return nil
	}

	deptIDs := make(map[string]domain.ID, len(hrSampleDepartments))
	for _, d := range hrSampleDepartments {
		var id domain.ID
		if err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO sdplatform.hr_department (organization, code, name, created_at, updated_at)
				SELECT $1, LEFT($2, 45), LEFT($3, 250), NOW(), NOW()
				WHERE NOT EXISTS (
					SELECT 1 FROM sdplatform.hr_department
					WHERE organization = $1 AND name = LEFT($3, 250)
				)
				RETURNING department_id
			)
			SELECT department_id FROM ins
			UNION ALL
			SELECT department_id FROM sdplatform.hr_department
			WHERE organization = $1 AND name = LEFT($3, 250)
			LIMIT 1`, orgID, d.code, d.name).Scan(&id); err != nil {
			return err
		}
		deptIDs[d.code] = id
	}

	rows, err := tx.Query(ctx, `
		SELECT e.employee_id, COALESCE(pos.name, '')
		FROM sdplatform.hr_employee e
		JOIN sdplatform.hr_position pos ON pos.position_id = e.position_id
		WHERE COALESCE(e.is_active, 1) <> 0 AND e.department_id IS NULL`)
	if err != nil {
		return err
	}
	type pending struct {
		id       domain.ID
		position string
	}
	var employees []pending
	for rows.Next() {
		var e pending
		if err := rows.Scan(&e.id, &e.position); err != nil {
			rows.Close()
			return err
		}
		employees = append(employees, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range employees {
		code, ok := hrPositionDepartment[e.position]
		if !ok {
			code = hrDefaultDepartmentCode
		}
		if _, err := tx.Exec(ctx,
			`UPDATE sdplatform.hr_employee SET department_id = $1, updated_at = NOW()
			 WHERE employee_id = $2 AND department_id IS NULL`, deptIDs[code], e.id); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("hr departments seeded",
		"departments", len(hrSampleDepartments), "employees_linked", len(employees))
	return nil
}

// Гадаад ролиудын permission багц — ЗӨВХӨН compensation:* (өөр эрх олгогдохгүй).
var externalRolePermissions = map[string]struct {
	description string
	perms       []string
}{
	"professional_org":   {"Мэргэжлийн байгууллага", []string{"compensation:read", "compensation:create", "compensation:update"}},
	"mika":               {"МИКА", []string{"compensation:read", "compensation:create", "compensation:update"}},
	"finance_specialist": {"Санхүүгийн мэргэжилтэн", []string{"compensation:read", "compensation:update"}},
}

var system10AdminPermissions = []string{
	"users:read",
	"users:create",
	"roles:read",
	"permissions:read",
	"hr:read",
	"hr:create",
}

var system10RoleNames = []string{
	"admin",
	"senior_specialist",
	"employee",
}

var system10MenuSeeds = []struct {
	code      string
	name      string
	url       string
	sortOrder int
}{
	{"A010", "Газар чөлөөлөлт", "/land-acquisitions", 1},
	{"A010001", "Газар чөлөөлөлтийн жагсаалт", "/land-acquisitions", 1},
	{"A010002", "Нэгж талбар", "/land-acquisitions/:id/parcels", 2},
	{"A010003", "Хөрөнгө, үнэлгээ", "/land-acquisitions/:id/assets", 3},
	{"A010004", "Нөхөх олговор", "/land-acquisitions/:id/compensations", 4},
	{"A010005", "Баримт бичиг", "/land-acquisitions/:id/documents", 5},
}

func (s *Seeder) SeedSystem10AccessPolicy(ctx context.Context) error {
	if err := s.grantLevel(ctx, "admin", defaultPermissions); err != nil {
		return err
	}
	return s.SeedLegacySDPlatformSystem10(ctx)
}

func (s *Seeder) SeedExternalValuationOrgUsers(ctx context.Context) error {
	if err := s.SeedCompensationAccess(ctx); err != nil {
		return err
	}
	users := []struct {
		username string
		email    string
		first    string
		last     string
	}{
		{"valuation_org_1", "valuation.org1@example.mn", "Үнэлгээний", "Байгууллага 1"},
		{"valuation_org_2", "valuation.org2@example.mn", "Үнэлгээний", "Байгууллага 2"},
		{"valuation_org_3", "valuation.org3@example.mn", "Үнэлгээний", "Байгууллага 3"},
	}
	for _, u := range users {
		if err := s.seedRoleUser(ctx, "professional_org", u.username, u.email, "Testpass123!", u.first, u.last, "Мэргэжлийн байгууллага"); err != nil {
			return err
		}
	}
	return nil
}

// SeedLegacySDPlatformSystem10 — систем (system_id=10), цэсний мод болон
// маягтуудыг sdplatform-д бүртгэж, энэ системийн БҮХ роль дээр цэсийг холбоно.
//
// Эрхийн загварын 1-р давхарга нь ЦЭС (sd_menu / sd_menu_role): хэрэглэгчид
// эхлээд цэс харагдах эсэх шийдэгдэж, дараа нь роль (нөөц) ба эрх (үйлдэл)
// шалгагдана. Роль/эрх нь grantLevel()-ээр удирдагдана — энд зөвхөн цэс.
func (s *Seeder) SeedLegacySDPlatformSystem10(ctx context.Context) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var legacyTablesExist bool
	if err := tx.QueryRow(ctx, `
		SELECT to_regclass('sdplatform.sd_system') IS NOT NULL
		   AND to_regclass('sdplatform.sd_menu') IS NOT NULL
		   AND to_regclass('sdplatform.sd_menu_role') IS NOT NULL
	`).Scan(&legacyTablesExist); err != nil {
		return err
	}
	if !legacyTablesExist {
		slog.Info("sdplatform menu tables are absent; skipping menu sync")
		return nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_system (system_id, "name", description)
		VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
		ON CONFLICT (system_id) DO UPDATE SET
			"name" = EXCLUDED."name",
			description = EXCLUDED.description
	`); err != nil {
		return err
	}

	for _, m := range system10MenuSeeds {
		parentCode := ""
		if m.code != "A010" {
			parentCode = "A010"
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO sdplatform.sd_menu (code, "name", menu_url, sort_order, created_at, parent_code)
			VALUES (LEFT($1, 32), LEFT($2, 250), LEFT($3, 250), $4, NOW(), NULLIF($5, ''))
			ON CONFLICT (code) DO UPDATE SET
				"name" = EXCLUDED."name",
				menu_url = EXCLUDED.menu_url,
				sort_order = EXCLUDED.sort_order,
				parent_code = EXCLUDED.parent_code,
				updated_at = NOW()`,
			m.code, m.name, m.url, m.sortOrder, parentCode); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sdplatform.sd_menu child
		SET parent_id = parent.menu_id
		FROM sdplatform.sd_menu parent
		WHERE child.parent_code = parent.code AND child.parent_id IS DISTINCT FROM parent.menu_id`); err != nil {
		return err
	}

	// Маягтууд (sd_form) — цэс бүрт нэг.
	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_form (form_name, code, created_at)
		SELECT m."name", m.code, NOW()
		FROM sdplatform.sd_menu m
		WHERE m.code = ANY($1)
		  AND NOT EXISTS (SELECT 1 FROM sdplatform.sd_form f WHERE f.code = m.code)`,
		system10MenuCodes()); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_menu_form (form_id, menu_id, form_code, menu_code, created_at)
		SELECT f.form_id, m.menu_id, f.code, m.code, NOW()
		FROM sdplatform.sd_form f
		JOIN sdplatform.sd_menu m ON m.code = f.code
		WHERE m.code = ANY($1)
		  AND NOT EXISTS (
			SELECT 1 FROM sdplatform.sd_menu_form mf
			WHERE mf.form_code = f.code AND mf.menu_code = m.code
		  )`, system10MenuCodes()); err != nil {
		return err
	}

	// Энэ системийн бүх роль дээр цэсийг холбоно.
	tag, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at)
		SELECT m.menu_id, r.role_id, NOW()
		FROM sdplatform.sd_menu m
		CROSS JOIN sdplatform.sd_role r
		WHERE m.code = ANY($1)
		  AND EXISTS (
			SELECT 1 FROM sdplatform.sd_system_role sr
			WHERE sr.role_id = r.role_id AND sr.system_id = 10
		  )
		ON CONFLICT DO NOTHING`, system10MenuCodes())
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("sdplatform system 10 menus synced", "menus", len(system10MenuSeeds), "menu_role_links", tag.RowsAffected())
	return nil
}

func system10MenuCodes() []string {
	out := make([]string, 0, len(system10MenuSeeds))
	for _, m := range system10MenuSeeds {
		out = append(out, m.code)
	}
	return out
}

// SeedPermissionFixes — эрхийн нэршлийн зөрчлүүдийг засна:
//  1. admin:* permission үүсгэж admin роль-д олгоно (лавлах CRUD + Тохиргоо цэс)
//  2. users:read-ийг senior_specialist-д олгоно (ажилтан/байгууллага сонгох UI)
//  3. Орфон permission-уудыг устгана (acquisition.create, valuation:create/update)
//
// SeedPermissionFixes — лавлах CRUD (admin:*), аудит, мөн ахлах мэргэжилтний
// users:read эрхийг баталгаажуулна.
func (s *Seeder) SeedPermissionFixes(ctx context.Context) error {
	if err := s.grantLevel(ctx, "admin", []string{"admin:read", "admin:create", "admin:update", "admin:delete", "audit:read"}); err != nil {
		return err
	}
	if err := s.grantLevel(ctx, "senior_specialist", []string{"users:read"}); err != nil {
		return err
	}
	slog.Info("permission fixes applied")
	return nil
}

// SeedAdminAllPermissions — admin роль-д defaultPermissions-ийн БҮХ эрхийг
// олгоно (идемпотент).
//
// Яагаад: compensation:* эрхүүд admin роль үүссэний ДАРАА нэмэгдсэн тул
// (000008_compensation_access) admin-д олгогдоогүй үлдсэн. Үүнээс болж
// "эрх нэмэгдүүлэлтээс сэргийлэх" шалгалт (роль/эрх олгохдоо дуудагч өөрөө
// тухайн эрхийг эзэмшсэн байх ёстой) админд МИКА / мэргэжлийн байгууллага /
// санхүүгийн роль олгохыг хориглож байсан. admin бол "full access" роль тул
// шинэ эрх нэмэгдэх бүрд түүнд олгогдох ёстой.
// SeedAdminAllPermissions — admin түвшинд БҮХ эрхийг олгоно (идемпотент).
// Шинэ эрх нэмэгдэх бүрд админ түүнийг эзэмших ёстой, эс тэгвээс "эрх
// нэмэгдүүлэлтээс сэргийлэх" шалгалт админд тухайн роль олгохыг хориглоно.
func (s *Seeder) SeedAdminAllPermissions(ctx context.Context) error {
	if err := s.grantLevel(ctx, "admin", defaultPermissions); err != nil {
		return err
	}
	slog.Info("admin level granted all permissions", "count", len(defaultPermissions))
	return nil
}

func (s *Seeder) SeedAuditPermission(ctx context.Context) error {
	if err := s.grantLevel(ctx, "admin", []string{"audit:read"}); err != nil {
		return err
	}
	slog.Info("audit permission granted", "level", "admin")
	return nil
}

// SeedDecisionPermissions — decision:* эрхүүдийг үүсгэж олгоно.
//
// Захирамжийн төслийн маршрутууд land:*-аас decision:* руу шилжсэн тул:
//  1. admin роль-д БҮХ decision:* эрхийг олгоно (админ бол full access роль,
//     мөн эрх нэмэгдүүлэлтийн шалгалт нь өөрт байхгүй эрхийг бусдад олгуулдаггүй).
//  2. Одоо land:<action> эрхтэй БҮХ роль-д харгалзах decision:<action>-ийг
//     олгоно — өмнө нь захирамжийн төсөлтэй ажиллаж байсан мэргэжилтнүүд
//     (ахлах мэргэжилтэн, UI-аас үүсгэсэн ролиуд) эрхээ алдахгүй.
//
// Гадаад ролиуд (мэрг. байгууллага, МИКА, санхүү) land:* эрхгүй тул (зөвхөн
// compensation:*) энэ алхмаар decision:* авахгүй.
// SeedDecisionPermissions — захирамжийн төслийн эрхүүд.
// Админд бүгдийг, land:<action> эрхтэй түвшингүүдэд харгалзах
// decision:<action>-ийг олгоно (өмнөх зан төлөв хэвээр).
func (s *Seeder) SeedDecisionPermissions(ctx context.Context) error {
	if err := s.grantLevel(ctx, "admin", []string{"decision:read", "decision:create", "decision:update", "decision:delete"}); err != nil {
		return err
	}
	// land:<action> → decision:<action> тусгал: land нөөцийн роль бүрийн
	// түвшинд харгалзах decision ролийг үүсгэж, ижил үйлдлүүдийг олгоно.
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return err
	}
	mirrored := 0
	for _, r := range roles {
		if r.Resource != "land" {
			continue
		}
		var perms []string
		for _, p := range r.Permissions {
			perms = append(perms, domain.PermissionKey("decision", p.Action))
		}
		if len(perms) == 0 {
			continue
		}
		if err := s.grantLevel(ctx, r.Level(), perms); err != nil {
			return err
		}
		mirrored += len(perms)
	}
	slog.Info("decision permissions granted", "mirrored_from_land", mirrored)
	return nil
}

// SeedCompensationAccess — compensation:* permission-уудыг үүсгэж, гадаад
// ролиудад ЗӨВХӨН тэдгээрийг олгоно (бусад permission-уудыг нь хураана).
// SeedCompensationAccess — гадаад түвшингүүдэд ЗӨВХӨН compensation:* эрхийг
// олгоно (бусад нөөцийн ролийг нь хоослоно).
func (s *Seeder) SeedCompensationAccess(ctx context.Context) error {
	allowed := map[string]bool{"compensation": true}
	for level, cfg := range externalRolePermissions {
		if err := s.grantLevel(ctx, level, cfg.perms); err != nil {
			return err
		}
		if err := s.revokeOtherResources(ctx, level, allowed); err != nil {
			return err
		}
		slog.Info("external level permissions synced", "level", level, "perms", cfg.perms)
	}
	return nil
}

type Seeder struct {
	db       *pgxpool.Pool
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

func NewSeeder(db *pgxpool.Pool, userRepo repository.UserRepository, roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *Seeder {
	return &Seeder{db: db, userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo}
}

func (s *Seeder) Seed(ctx context.Context, adminCfg config.AdminConfig, employeeCfg config.EmployeeConfig) error {
	if err := s.grantLevel(ctx, "admin", defaultPermissions); err != nil {
		return err
	}
	if err := s.grantLevel(ctx, "senior_specialist", seniorSpecialistPermissions); err != nil {
		return err
	}
	if err := s.grantLevel(ctx, "employee", employeePermissions); err != nil {
		return err
	}
	if err := s.seedAdminUser(ctx, adminCfg, "admin", "senior_specialist"); err != nil {
		return err
	}
	return s.seedEmployeeUser(ctx, employeeCfg, "employee")
}

func (s *Seeder) seedAdminUser(ctx context.Context, cfg config.AdminConfig, levels ...string) error {
	existing, err := s.userRepo.GetByEmail(ctx, cfg.Email)
	if err == nil {
		if err := s.assignLevelToUser(ctx, existing.ID, levels...); err != nil {
			return err
		}
		if existing.Position == "" {
			existing.Position = "Ахлах мэргэжилтэн"
			existing.UpdatedAt = time.Now()
			_ = s.userRepo.Update(ctx, existing)
		}
		return nil
	}

	hashed, err := hash.Password(cfg.Password)
	if err != nil {
		return err
	}

	now := time.Now()
	user := &domain.User{
		Username:  "admin",
		Email:     cfg.Email,
		Password:  hashed,
		FirstName: "Admin",
		LastName:  "User",
		Position:  "Ахлах мэргэжилтэн",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	if err := s.assignLevelToUser(ctx, user.ID, levels...); err != nil {
		return err
	}
	slog.Info("admin user created", "email", cfg.Email)
	return nil
}

func (s *Seeder) seedEmployeeUser(ctx context.Context, cfg config.EmployeeConfig, level string) error {
	existing, err := s.userRepo.GetByEmail(ctx, cfg.Email)
	if err == nil {
		if err := s.assignLevelToUser(ctx, existing.ID, level); err != nil {
			return err
		}
		if existing.Position == "" {
			existing.Position = "Ажилтан"
			existing.UpdatedAt = time.Now()
			_ = s.userRepo.Update(ctx, existing)
		}
		return nil
	}

	hashed, err := hash.Password(cfg.Password)
	if err != nil {
		return err
	}

	now := time.Now()
	user := &domain.User{
		Username:  cfg.Username,
		Email:     cfg.Email,
		Password:  hashed,
		FirstName: "Энгийн",
		LastName:  "Ажилтан",
		Position:  "Ажилтан",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	if err := s.assignLevelToUser(ctx, user.ID, level); err != nil {
		return err
	}
	slog.Info("employee user created", "email", cfg.Email)
	return nil
}

// SeedExternalUsers — санхүүгийн мэргэжилтэн болон мэргэжлийн байгууллагын
// анхны хэрэглэгчийг үүсгэнэ. Роль (finance_specialist, professional_org) нь
// SeedCompensationAccess-ээр аль хэдийн үүссэн байх ёстой.
func (s *Seeder) SeedExternalUsers(ctx context.Context, finCfg config.FinanceConfig, profCfg config.ProfessionalOrgConfig) error {
	if err := s.seedRoleUser(ctx, "finance_specialist",
		finCfg.Username, finCfg.Email, finCfg.Password,
		"Санхүүгийн", "мэргэжилтэн", "Санхүүгийн мэргэжилтэн"); err != nil {
		return err
	}
	return s.seedRoleUser(ctx, "professional_org",
		profCfg.Username, profCfg.Email, profCfg.Password,
		"Мэргэжлийн", "байгууллага", "Мэргэжлийн байгууллага")
}

// seedRoleUser — өгсөн роль-той нэг хэрэглэгчийг idempotent байдлаар үүсгэнэ.
// Роль байхгүй бол чимээгүй алгасна (seed дараалал буруу тохиолдолд ч эвдрэхгүй).
func (s *Seeder) seedRoleUser(ctx context.Context, level, username, email, password, firstName, lastName, position string) error {
	roles, err := s.rolesForLevel(ctx, level)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		slog.Warn("external level has no roles, skipping user seed", "level", level)
		return nil
	}

	if existing, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return s.assignLevelToUser(ctx, existing.ID, level)
	}

	hashed, err := hash.Password(password)
	if err != nil {
		return err
	}
	now := time.Now()
	user := &domain.User{
		Username:  username,
		Email:     email,
		Password:  hashed,
		FirstName: firstName,
		LastName:  lastName,
		Position:  position,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	if err := s.assignLevelToUser(ctx, user.ID, level); err != nil {
		return err
	}
	slog.Info("external user created", "email", email, "level", level)
	return nil
}
