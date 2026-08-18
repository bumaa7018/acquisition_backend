package seed

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultPermissions — системийн бүх эрх "menu/form:permission" хэлбэрээр.
var defaultPermissions = []string{
	"users:read", "users:create", "users:update", "users:delete",
	"roles:read", "roles:create", "roles:update", "roles:delete",
	"permissions:read", "permissions:create", "permissions:delete",
	"admin:read", "admin:create", "admin:update", "admin:delete",
	"hr:read", "hr:create", "hr:structure:create", "hr:update", "hr:delete",
	"audit:read",
	"land:read", "land:create", "land:update", "land:delete",
	// Нөхөх олговрын (үнэлгээний) эрхүүд — гадаад ролиуд ЗӨВХӨН эдгээртэй.
	"compensation:read", "compensation:create", "compensation:update",
	// Захирамжийн төслийн эрхүүд.
	"decision:read", "decision:create", "decision:update", "decision:delete",
}

// Дотоод системийн дүрүүдийн permission matrix.
//
// senior_specialist: тохиргоо/удирдлагаас бусад үндсэн ажлын урсгалд бүрэн
// ажиллана. users:read нь хуучин тохиргооноос үлдсэн, ажилтан сонгох UI/API-д
// хэрэгтэй туслах эрх.
var seniorSpecialistPermissions = []string{
	"land:read", "land:create", "land:update", "land:delete",
	"compensation:read", "compensation:create", "compensation:update",
	"decision:read", "decision:create", "decision:update", "decision:delete",
	"users:read",
}

var employeePermissions = []string{
	"land:read",
	"decision:read",
}

var decisionRegistrationPermissions = []string{
	"land:read",
	"decision:read", "decision:create", "decision:update", "decision:delete",
}

// SeedAuthLookups — authdb-ийн лавлах мөрүүд.
//
// Эдгээр нь урьд нь миграцад INSERT-ээр орж байсан. Миграц = зөвхөн бүтэц
// болсон тул өгөгдөл нь энд ирэв. ЗӨВХӨН байхгүй үед нэмнэ — байгаа мөрийг
// дарж бичихгүй (жинхэнэ sdplatform санд эдгээр лавлах аль хэдийн бөглөгдсөн
// байх магадлалтай).
func (s *Seeder) SeedAuthLookups(ctx context.Context) error {
	if err := s.seedPersonTypes(ctx); err != nil {
		return err
	}
	if err := s.seedLegalEntityTypes(ctx); err != nil {
		return err
	}
	// sd_system_role.system_id → sd_system FK тул систем 10 эхлээд байх ёстой.
	if _, err := s.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_system (system_id, "name", description)
		VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
		ON CONFLICT (system_id) DO NOTHING`); err != nil {
		return err
	}
	slog.Info("auth lookups seeded")
	return nil
}

// personTypes — codelists.cl_person_type-ийн БОДИТ лавлах (ГУС-аас өгсөн).
//
// Код нь 10-аар үсэрдэг (1,2 гэсэн код БАЙХГҮЙ) ба base.bs_person."type" нь
// эдгээр код рүү FK-аар холбогддог тул апп мөн ЭНЭ кодуудыг бичих ёстой
// (postgres.personTypeCitizen / personTypeLegal-ыг үз).
var personTypes = []struct {
	Code   int
	Desc   string
	DescEn string
}{
	{10, "1: Монгол улсын эрх зүйн чадамжтай иргэн", "1: Citizen of Mongolia with legal capacity"},
	{20, "2: Монгол улсын эрх зүйн чадамжгүй иргэн", "2: Citizen of Mongolia without legal capacity"},
	{30, "3: Монгол улсын хуулийн этгээд", "3: Legal entity of Mongolia"},
	{40, "4: Монгол улсын төрийн байгууллага", "4: State organisation of Mongolia"},
	{50, "5: Гадаадын иргэн", "5: Foreign citizen"},
	{60, "6: Гадаадын хуулийн этгээд", "6: Legal entity of foreign country"},
	{70, "7: Гадаадын хөрөнгө оруулалттай хуулийн этгээд", "7: Legal entity with foreign investment"},
	{80, "8: СӨХ", "8: Condominium Association"},
	{90, "9: Чөлөөт бүс", "9: Free zone"},
	{100, "10: Малчдын хоршоо", "10: Herders cooperative"},
}

// legalEntityTypes — codelists.cl_legal_entity_type-ийн БОДИТ лавлах
// (ГУС-аас өгсөн). base.bs_person.entity_type нь эдгээр id рүү FK-аар
// холбогддог.
//
// description_en нь эх өгөгдөлд хоосон тул бичихгүй — мөн локал схемд тэр
// багана байхгүй байж болно (INSERT нь зөвхөн id, description).
var legalEntityTypes = []struct {
	ID   int
	Desc string
}{
	{1, "Банк"},
	{2, "ҮЦаас ТТооцоо Төвл. Хад Төв"},
	{3, "ББСБ"},
	{4, "Сум хөгжүүлэх сан"},
	{5, "ЖДҮ-г хөгжүүлэх сан"},
	{6, "Иргэн, хуулийн этгээд"},
	{7, "Бусад"},
}

// seedLegalEntityTypes — хуулийн этгээдийн төрлийн лавлахыг ЗӨВХӨН хүснэгт
// БҮРЭН ХООСОН үед дүүргэнэ.
//
// Хүснэгт нь ГУС-ийн эзэмшилд: ганц ч мөр байвал тэр нь эх системийн өгөгдөл
// гэж үзээд ОГТ хөндөхгүй (мөр тус бүрээр нөхөхгүй — id-ийн утга эх системд
// өөр байж болзошгүй тул хагас хольсон лавлах үүсгэхээс сэргийлнэ).
func (s *Seeder) seedLegalEntityTypes(ctx context.Context) error {
	var any bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM codelists.cl_legal_entity_type)`).Scan(&any); err != nil {
		return err
	}
	if any {
		return nil // ГУС-ийн өгөгдөл байна — хөндөхгүй
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, t := range legalEntityTypes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO codelists.cl_legal_entity_type (id, description)
			VALUES ($1, LEFT($2, 250))
			ON CONFLICT (id) DO NOTHING`, t.ID, t.Desc); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("cl_legal_entity_type лавлах дүүрлээ", "rows", len(legalEntityTypes))
	return nil
}

// seedPersonTypes — хүний төрлийн лавлахыг ЗӨВХӨН байхгүй үед нэмнэ.
//
// Хүснэгт нь ГУС-ийн ЭЗЭМШИЛД байх ба гурван UNIQUE хязгаартай: code,
// description, description_en. Иймд `ON CONFLICT (code)` дангаараа хүрэлцэхгүй —
// ижил тайлбар ӨӨР кодоор бүртгэлтэй байвал бичилт unique зөрчлөөр унана.
// Гурвуулангаар шалгаж, аль нэг таарвал мөрийг ОГТ хөндөхгүй (байгаа мөрийг
// дарж бичихгүй: ГУС-ийн эх өгөгдөл нь эрхэм).
func (s *Seeder) seedPersonTypes(ctx context.Context) error {
	added := 0
	for _, pt := range personTypes {
		tag, err := s.db.Exec(ctx, `
			INSERT INTO codelists.cl_person_type (code, description, description_en)
			SELECT $1::int, $2::text, $3::text
			WHERE NOT EXISTS (
				SELECT 1 FROM codelists.cl_person_type
				WHERE code = $1::int
				   OR description = $2::text
				   OR description_en = $3::text
			)`, pt.Code, pt.Desc, pt.DescEn)
		if err != nil {
			return err
		}
		added += int(tag.RowsAffected())
	}
	if added > 0 {
		slog.Info("cl_person_type лавлах нөхөгдлөө", "added", added, "total", len(personTypes))
	}
	return nil
}

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
	slog.Info("hr backfill skipped: startup seeder must not insert sd_organization data")
	return nil
}

// hrSeedDepartment — seeder-ийн НЭГ хэлтэс. ГУС-ийн бодит код/нэр.
//
// Өмнө нь 7 жишээ хэлтэс (ADM/GCH/…) үүсгэдэг байсан — тэдгээр нь бодит
// бүтэцтэй тохирдоггүй тул хасав. Одоо ЗӨВХӨН энэ мөрийг, ЗӨВХӨН код нь
// байхгүй үед нэмнэ.
var hrSeedDepartment = struct {
	code string
	name string
}{
	code: "10311",
	name: "Нийслэлийн газар зохион байгуулалтын алба",
}

// SeedHRDepartments — "Алба, хэлтэс" лавлахад ГУС-ийн хэлтсийг (код 10311)
// бүртгэж, хэлтэсгүй ажилтныг түүнд холбоно.
//
// Ажилтны БАЙГУУЛЛАГА нь хэлтсээр дамждаг (hr_employee-д байгууллагын багана
// байхгүй) тул энэ алхам нь ажилтныг байгууллагатай холбох цорын ганц зам.
//
// Идемпотент: хэлтсийг КОДООР шалгаж, байхгүй үед л нэмнэ; ажилтны холбоосыг
// department_id IS NULL шалгуураар л хийнэ — гараар зассан өгөгдлийг дарж
// бичихгүй.
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

	// Хэлтэс — ЗӨВХӨН код нь байхгүй үед нэмнэ.
	var deptID domain.ID
	if err := tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO sdplatform.hr_department (organization, code, name, created_at, updated_at)
			SELECT $1, LEFT($2, 45), LEFT($3, 250), NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM sdplatform.hr_department WHERE code = LEFT($2, 45)
			)
			RETURNING department_id
		)
		SELECT department_id FROM ins
		UNION ALL
		SELECT department_id FROM sdplatform.hr_department WHERE code = LEFT($2, 45)
		LIMIT 1`, orgID, hrSeedDepartment.code, hrSeedDepartment.name).Scan(&deptID); err != nil {
		return err
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
		if _, err := tx.Exec(ctx,
			`UPDATE sdplatform.hr_employee SET department_id = $1, updated_at = NOW()
			 WHERE employee_id = $2 AND department_id IS NULL`, deptID, e.id); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("hr departments seeded",
		"code", hrSeedDepartment.code, "employees_linked", len(employees))
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

var system10RoleNames = []string{
	"admin",
	"senior_specialist",
	"employee",
	"decision_registration",
}

// system10MenuSeeds — цэс ба түүний МАЯГТЫН нэр.
//
// formName нь sd_form.form_name-д бичигдэнэ: кодчилсон латин нэр (эх системийн
// бусад маягттай ижил хэлбэр), цэсний кирилл нэр биш. Хэрэглээнд ямар ч
// нөлөө үзүүлэхгүй — маягтыг ҮРГЭЛЖ `code`-оор холбодог (sd_menu_form,
// sd_form_permission, menu.go/role.go/user.go бүгд f.code-оор join хийнэ).
var system10MenuSeeds = []struct {
	code      string
	name      string
	formName  string
	url       string
	sortOrder int
}{
	{"A010101", "Газар чөлөөлөлт", "land_acquisition", "/land-acquisitions", 1},
	{"A010102", "Нөхөх олговор", "compensation", "/land-acquisitions/:id/compensations", 2},
	{"A010103", "Захирамжийн төсөл", "decision_draft", "/decision-drafts", 3},
	{"A010104", "Хэрэглэгч", "user", "/users", 10},
	{"A010105", "Хэрэглэгчийн эрх", "role", "/roles", 11},
	{"A010106", "Permission лавлах", "permission", "/roles", 12},
	{"A010107", "Хүний нөөц", "hr", "/employee", 13},
	{"A010108", "Системийн тохиргоо", "system_setting", "/acquisition-category", 14},
	{"A010109", "Үйлдлийн лог", "audit_log", "/audit-logs", 15},
}

// menuScopedPermissions — "land:read" (эрхийн түлхүүр) → "A010101:read"
// (sd_form/sd_menu-гийн бодит код). Түлхүүрийн нөөц нэр код руу хөрвөнө.
func menuScopedPermissions(perms []string) []string {
	out := make([]string, 0, len(perms))
	for _, key := range perms {
		resource, action := domain.SplitPermissionKey(key)
		if resource == "" || action == "" {
			continue
		}
		out = append(out, domain.MenuCodeForResource(resource)+":"+action)
	}
	return out
}

func (s *Seeder) SeedSystem10AccessPolicy(ctx context.Context) error {
	if err := s.SeedSystemRoles(ctx); err != nil {
		return err
	}
	return s.SeedLegacySDPlatformSystem10(ctx)
}

// SeedSystemRoles — system_id=10 дээр харагдах үндсэн 4 дүрийг шинэ
// role->menu->permission бүтцээр яг тохируулна.
func (s *Seeder) SeedSystemRoles(ctx context.Context) error {
	matrix := map[string][]string{
		"admin":                 defaultPermissions,
		"senior_specialist":     seniorSpecialistPermissions,
		"employee":              employeePermissions,
		"decision_registration": decisionRegistrationPermissions,
	}
	for _, level := range system10RoleNames {
		if err := s.syncSystemLevel(ctx, level, matrix[level]); err != nil {
			return err
		}
	}
	if err := s.hideGeneratedInternalResourceRoles(ctx); err != nil {
		return err
	}
	slog.Info("system 10 roles synced", "roles", system10RoleNames)
	return nil
}

func (s *Seeder) hideGeneratedInternalResourceRoles(ctx context.Context) error {
	names := make([]string, 0, len(system10MenuSeeds)*len(system10RoleNames))
	for _, menu := range system10MenuSeeds {
		for _, level := range system10RoleNames {
			names = append(names, roleName(domain.MenuResource(menu.code), level))
		}
	}
	// Seeder нь байгаа өгөгдлийг УСТГАХГҮЙ. Хуучин үүсмэл ролиудыг system 10-оос
	// хасах ажил нь миграц 000018-д (нэг удаагийн цэвэрлэгээ) — энд зөвхөн
	// үлдсэн эсэхийг мэдээлнэ.
	var stale int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM sdplatform.sd_system_role sr
		JOIN sdplatform.sd_role r ON r.role_id = sr.role_id
		WHERE sr.system_id = 10 AND r.name = ANY($1)`, names).Scan(&stale); err != nil {
		return err
	}
	if stale > 0 {
		slog.Warn("хуучин үүсмэл роль system 10-д хэвээр байна (seeder устгахгүй)", "roles", stale)
	}
	return nil
}

// SeedExternalValuationOrgUsers — гадаад (үнэлгээний байгууллага) хэрэглэгчид.
//
// Эрхийн хэсэг (SeedCompensationAccess: роль/эрх) нь ҮРГЭЛЖ ажиллана; ХЭРЭГЛЭГЧ
// нь ЗӨВХӨН sd_user хоосон үед л үүснэ (Seed-тэй ижил дүрэм).
func (s *Seeder) SeedExternalValuationOrgUsers(ctx context.Context) error {
	if err := s.SeedCompensationAccess(ctx); err != nil {
		return err
	}
	empty, err := s.usersEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		slog.Info("гадаад хэрэглэгчийн seed алгасагдав: sd_user хоосон биш")
		return nil
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
		ON CONFLICT (system_id) DO NOTHING
	`); err != nil {
		return err
	}

	for _, m := range system10MenuSeeds {
		if _, err := tx.Exec(ctx, `
			INSERT INTO sdplatform.sd_menu (code, "name", menu_url, sort_order, created_at, parent_code)
			VALUES (LEFT($1, 32), LEFT($2, 250), LEFT($3, 250), $4, NOW(), NULLIF($5, ''))
			ON CONFLICT (code) DO NOTHING`,
			m.code, m.name, m.url, m.sortOrder, ""); err != nil {
			return err
		}
	}
	// parent_id-г ЗӨВХӨН хоосон үед бөглөнө — байгаа утгыг дарж бичихгүй.
	if _, err := tx.Exec(ctx, `
		UPDATE sdplatform.sd_menu child
		SET parent_id = parent.menu_id
		FROM sdplatform.sd_menu parent
		WHERE child.parent_code = parent.code
		  AND child.parent_id IS NULL
		  AND child.code = ANY($1)`, system10MenuCodes()); err != nil {
		return err
	}

	// Маягтууд (sd_form) — цэс бүрт нэг. Нэр нь КОДЧИЛСОН латин нэр
	// (land_acquisition г.м.) — system10MenuSeeds.formName-ыг үз.
	// Хүснэгтийн бүтэц миграцад (000008 / 000011) — seed зөвхөн мөр нэмнэ.
	formCodes, formNames := system10FormNames()
	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_form (form_name, code, created_at)
		SELECT w.form_name, w.code, NOW()
		FROM UNNEST($1::text[], $2::text[]) AS w(code, form_name)
		JOIN sdplatform.sd_menu m ON m.code = w.code
		WHERE NOT EXISTS (SELECT 1 FROM sdplatform.sd_form f WHERE f.code = w.code)`,
		formCodes, formNames); err != nil {
		return err
	}
	// Өмнөх seed нь маягтын нэрэнд ЦЭСНИЙ кирилл нэрийг бичдэг байсан —
	// ЗӨВХӨН тэр тохиолдолд кодчилсон нэрээр нөхнө (гараар зассаныг хөндөхгүй).
	if _, err := tx.Exec(ctx, `
		UPDATE sdplatform.sd_form f
		SET form_name = w.form_name
		FROM UNNEST($1::text[], $2::text[]) AS w(code, form_name)
		JOIN sdplatform.sd_menu m ON m.code = w.code
		WHERE f.code = w.code
		  AND COALESCE(f.form_name, '') IN ('', m."name")`,
		formCodes, formNames); err != nil {
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

	if _, err := tx.Exec(ctx, `
		WITH wanted AS (
			SELECT split_part(v, ':', 1) AS form_code,
			       substring(v from position(':' in v) + 1) AS permission_code
			FROM unnest($1::text[]) AS t(v)
		)
		INSERT INTO sdplatform.sd_form_permission (
			form_id, permission_id, form_code, permission_code, created_at, updated_at
		)
		SELECT DISTINCT ON (f.code, p.name)
		       f.form_id, p.permission_id, f.code, p.name, NOW(), NOW()
		FROM wanted w
		JOIN sdplatform.sd_form f ON f.code = w.form_code
		JOIN sdplatform.sd_permission p ON p.name = w.permission_code
		WHERE NOT EXISTS (
			SELECT 1
			FROM sdplatform.sd_form_permission fp
			WHERE fp.form_code = f.code
			  AND (
				fp.permission_id = p.permission_id
				OR fp.permission_code = p.code
				OR fp.permission_code = p.name
			  )
		)
		-- DISTINCT ON: sd_permission дээр ижил нэртэй олон мөр байвал маягт бүрд
		-- ЗӨВХӨН нэг (хамгийн бага permission_id-тай) мөр холбоно.
		ORDER BY f.code, p.name, p.permission_id`,
		menuScopedPermissions(defaultPermissions)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at, updated_at)
		SELECT rp.role_id, rp.permission_id, m.menu_id, COALESCE(rp.created_at, NOW()), NOW()
		FROM sdplatform.sd_role_permission rp
		JOIN sdplatform.sd_role r ON r.role_id = rp.role_id
		JOIN sdplatform.sd_menu m ON m.code = r.role_type_code
		WHERE rp.menu_id IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM sdplatform.sd_role_permission existing
			WHERE existing.role_id = rp.role_id
			  AND existing.permission_id = rp.permission_id
			  AND existing.menu_id = m.menu_id
		  )`); err != nil {
		return err
	}
	// ЖИЧ: Энд урьд нь menu_id IS NULL мөрүүдийг устгадаг байсан. Seeder нь
	// байгаа өгөгдлийг УСТГАХГҮЙ (зөвхөн байхгүйг нэмнэ) тул хассан —
	// жинхэнэ sdplatform DB-д тэдгээр мөр өөр системийнх байж болно.

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

// system10FormNames — маягтын код ба КОДЧИЛСОН нэрийн хос массив.
func system10FormNames() (codes []string, names []string) {
	codes = make([]string, 0, len(system10MenuSeeds))
	names = make([]string, 0, len(system10MenuSeeds))
	for _, m := range system10MenuSeeds {
		codes = append(codes, m.code)
		names = append(names, m.formName)
	}
	return codes, names
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

// seededMultiPartActions — үйлдэл нь өөрөө цэг агуулсан эрхүүдийн үйлдлийн нэр
// ("hr:structure:create" → "structure:create").
//
// ЗӨВХӨН ийм нэрс давхардах боломжтой байсан: ensureAction-ы хуучин хайлт нэрийг
// цэгээр дахин таслдаг тул "structure:create"-ийг "create"-ээр хайж, олдохгүй
// гээд алхам бүрд шинэ мөр үүсгэдэг байв. Нэг хэсэгтэй нэрс ("read", "create")
// зөв олддог тул хэзээ ч давхардаагүй. Цэвэрлэгээг үүгээр хязгаарлаж, өөр
// системийн sd_permission мөрүүдэд хүрэхээс сэргийлнэ.
func seededMultiPartActions() []string {
	seen := map[string]bool{}
	var out []string
	for _, key := range defaultPermissions {
		_, action := domain.SplitPermissionKey(key)
		if !strings.Contains(action, ":") || seen[action] {
			continue
		}
		seen[action] = true
		out = append(out, action)
	}
	return out
}

// SeedPermissionDedupe — ensureAction-ы алдаанаас үүссэн давхардсан
// sd_permission мөрүүдийг нэгтгэнэ.
//
// Шинэ сан дээр "structure:create" 5 удаа (seed алхам болгонд нэг) үүсч, түүнээс
// улбаалан sd_form_permission болон sd_role_permission ч олшрдог байв. Ажиллах
// үеийн эрхийн шалгалт DISTINCT ашигладаг тул хандалт буруу болдоггүй, гэхдээ
// ролийн эрхийн UI-д нэг эрх олон удаа харагдана.
//
// Дараалал ЧУХАЛ: эхлээд лавлагаануудыг үлдээх мөр рүү чиглүүлж, дараа нь илүү
// мөрүүдийг устгана (sd_role_permission нь sd_permission руу ON DELETE CASCADE
// FK-тай тул эсрэгээр хийвэл олгосон эрх устана).
func (s *Seeder) SeedPermissionDedupe(ctx context.Context) error {
	actions := seededMultiPartActions()
	if len(actions) == 0 {
		return nil
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// dup — устгах мөр бүр → үлдээх мөр (нэр тус бүрийн хамгийн бага id).
	const dupCTE = `
		WITH keep AS (
			SELECT name, MIN(permission_id) AS keep_id
			FROM sdplatform.sd_permission
			WHERE name = ANY($1::text[])
			GROUP BY name
			HAVING COUNT(*) > 1
		),
		dup AS (
			SELECT p.permission_id AS dup_id, k.keep_id
			FROM sdplatform.sd_permission p
			JOIN keep k ON k.name = p.name
			WHERE p.permission_id <> k.keep_id
			  -- sdplatform схемийг бусад систем хуваалцдаг: ӨӨР системийн
			  -- ролид холбогдсон мөрийг давхардал гэж үзэн УСТГАХГҮЙ.
			  AND NOT EXISTS (
				SELECT 1
				FROM sdplatform.sd_role_permission rp
				JOIN sdplatform.sd_system_role sr
				  ON sr.role_id = rp.role_id AND sr.system_id <> 10
				WHERE rp.permission_id = p.permission_id
			  )
		)`

	if _, err := tx.Exec(ctx, dupCTE+`
		UPDATE sdplatform.sd_role_permission rp
		SET permission_id = dup.keep_id, updated_at = NOW()
		FROM dup
		WHERE rp.permission_id = dup.dup_id`, actions); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, dupCTE+`
		UPDATE sdplatform.sd_form_permission fp
		SET permission_id = dup.keep_id, updated_at = NOW()
		FROM dup
		WHERE fp.permission_id = dup.dup_id`, actions); err != nil {
		return err
	}

	// Лавлагаа чиглүүлсний дараа ижил мөрүүд үлдэнэ — эхнийхийг нь үлдээнэ.
	if _, err := tx.Exec(ctx, `
		DELETE FROM sdplatform.sd_role_permission rp
		WHERE rp.role_permission_id > (
			SELECT MIN(other.role_permission_id)
			FROM sdplatform.sd_role_permission other
			WHERE other.role_id IS NOT DISTINCT FROM rp.role_id
			  AND other.permission_id IS NOT DISTINCT FROM rp.permission_id
			  AND other.menu_id IS NOT DISTINCT FROM rp.menu_id
		)
		  AND rp.permission_id IN (
			SELECT permission_id FROM sdplatform.sd_permission WHERE name = ANY($1::text[])
		  )`, actions); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM sdplatform.sd_form_permission fp
		WHERE fp.form_permission_id > (
			SELECT MIN(other.form_permission_id)
			FROM sdplatform.sd_form_permission other
			WHERE other.form_code IS NOT DISTINCT FROM fp.form_code
			  AND other.permission_id IS NOT DISTINCT FROM fp.permission_id
		)
		  AND fp.permission_id IN (
			SELECT permission_id FROM sdplatform.sd_permission WHERE name = ANY($1::text[])
		  )`, actions); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, dupCTE+`
		DELETE FROM sdplatform.sd_permission p
		USING dup
		WHERE p.permission_id = dup.dup_id`, actions)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("duplicate permissions merged", "actions", actions, "removed", tag.RowsAffected())
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
	if err := s.grantLevel(ctx, "senior_specialist", []string{"decision:read", "decision:create", "decision:update", "decision:delete"}); err != nil {
		return err
	}
	if err := s.grantLevel(ctx, "employee", []string{"decision:read"}); err != nil {
		return err
	}
	if err := s.grantLevel(ctx, "decision_registration", []string{"decision:read", "decision:create", "decision:update", "decision:delete"}); err != nil {
		return err
	}
	slog.Info("decision permissions granted")
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
	// db — authdb (sdplatform / base / codelists схемүүд ЗӨВХӨН энд байна).
	db *pgxpool.Pool
	// appDB — аппын өөрийн сан (appdb). Аппын хүснэгтэд бичихэд л хэрэглэнэ.
	// sdplatform-ын хүснэгтүүд appdb-д ҮҮСЭХГҮЙ тул тэднийг db-ээр л хөндөнө.
	appDB    *pgxpool.Pool
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

func NewSeeder(db *pgxpool.Pool, userRepo repository.UserRepository, roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *Seeder {
	return &Seeder{db: db, appDB: db, userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo}
}

// NewSeederFor — authdb (db) болон аппын сан (appDB)-г тусад нь өгнө. Хоёр
// санд зэрэг бичих шаардлагатай seed-д хэрэглэнэ.
func NewSeederFor(db, appDB *pgxpool.Pool, userRepo repository.UserRepository, roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *Seeder {
	return &Seeder{db: db, appDB: appDB, userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo}
}

// Seed — систем 10-ийн роль/эрх + ХЭРЭГЛЭГЧИЙН seed.
//
// ХЭРЭГЛЭГЧ (sd_user), АЖИЛТАН (hr_employee), ХҮН (bs_person) нь ЗӨВХӨН
// sd_user хүснэгт БҮРЭН ХООСОН үед үүснэ: жинхэнэ санд хэрэглэгч аль хэдийн
// бүртгэлтэй байвал апп нь тест/жишээ хэрэглэгч нэмэх ёсгүй.
//
// Роль / эрх / цэс / маягт нь ҮҮНЭЭС ҮЛ ХАМААРАН өөрсдийн шалгуураар
// нэмэгдсээр байна (SeedSystemRoles).
func (s *Seeder) Seed(ctx context.Context, adminCfg config.AdminConfig, employeeCfg config.EmployeeConfig) error {
	if err := s.SeedSystemRoles(ctx); err != nil {
		return err
	}

	empty, err := s.usersEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		slog.Info("хэрэглэгчийн seed алгасагдав: sd_user хоосон биш")
		return nil
	}

	if err := s.seedAdminUser(ctx, adminCfg, "admin"); err != nil {
		return err
	}
	return s.seedEmployeeUser(ctx, employeeCfg, "employee")
}

// usersEmpty — sd_user хүснэгтэд мөр БАЙХГҮЙ эсэх (soft delete-тэй мөрийг ч
// тооцно: устгасан хэрэглэгч байгаа гэдэг нь систем аль хэдийн хэрэглэгдсэн
// гэсэн үг).
func (s *Seeder) usersEmpty(ctx context.Context) (bool, error) {
	var any bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_user)`).Scan(&any); err != nil {
		return false, err
	}
	return !any, nil
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
			existing.Position = "Мэргэжилтэн"
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
		FirstName: "Мэргэжилтэн",
		LastName:  "Хэрэглэгч",
		Position:  "Мэргэжилтэн",
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
