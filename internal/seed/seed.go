package seed

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"

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

// systemRolePermissions — системийн 4 үндсэн ролийн анхны эрх. ЗӨВХӨН роль
// ШИНЭЭР үүсэх үед олгогдоно (ensureRole-ийг үз).
var systemRolePermissions = map[string][]string{
	"admin":                 defaultPermissions,
	"senior_specialist":     seniorSpecialistPermissions,
	"employee":              employeePermissions,
	"decision_registration": decisionRegistrationPermissions,
}

// externalRoles — гадаад ролиуд, ЗӨВХӨН compensation:* эрхтэй. Роль нь
// "compensation_<level>" нэртэй, эрх нь цэсгүй (menu_id NULL) холбогдоно.
//
// professional_org ЭНД БАЙХГҮЙ: үнэлгээний байгууллага системийн үндсэн
// хэрэглэгчээс тусдаа явдаг тул роль/эрх ОГТ хэрэглэхээ больсон. Тэдний
// хандалт нь appdb-ийн valuation_org_employee (токены `valuation_org` тэмдэг)
// ба аппын маршрутын allowlist-ээр шийдэгддэг — internal/middleware/
// valuation_org.go-г үз.
var externalRoles = []struct {
	level string
	perms []string
}{
	{"mika", []string{"compensation:read", "compensation:create", "compensation:update"}},
	{"finance_specialist", []string{"compensation:read", "compensation:update"}},
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

// EnsureAuthDB — authdb-д ЭНЭ СИСТЕМД (system_id=10) шаардлагатай мөрүүд
// байгаа эсэхийг шалгаж, ДУТУУГ Л нэмнэ. Startup БҮРТ ажиллана.
//
// authdb нь бусад системтэй хуваалцсан, аль хэдийн ажиллаж байгаа сан
// (sdplatform). Иймд:
//   - Байгаа мөрийг UPDATE / DELETE хийхгүй.
//   - Өөр системийн роль, хэрэглэгч, цэс, эрхийн холбоосод хүрэхгүй: ролийг
//     ЗӨВХӨН систем 10 дотроос нэрээр нь хайна.
//   - Роль байхгүй үед л анхны эрх/цэстэй нь үүсгэнэ. Байгаа ролийн эрхийг
//     дахин нөхөхгүй — админ UI-аас зориуд хассан эрх буцаж ирэхгүй.
//   - Хэрэглэгч: систем 10-д нэвтэрч чадах хэрэглэгч байхгүй бол системийн
//     админ; sd_user БҮРЭН ХООСОН (шинэ сан) бол мөн санхүүгийн мэргэжилтэн.
//
// Яагаад appdb-ийн seed_history-гүй вэ: authdb-ийг өөр сан руу солиход
// түүх нь appdb-д "хийгдсэн" хэвээр үлдэж, шинэ санд систем 10-ийн роль/цэс
// огт үүсэхгүй байв. Шалгалт нь өөрөө идемпотент тул түүх шаардлагагүй.
//
// Шинэ эрх нэмбэл: шинэ санд роль үүсэхдээ автоматаар авна; БАЙГАА санд
// олгох бол startup.go-д нэг удаагийн (seed_history-тэй) алхам нэмнэ.
func (s *Seeder) EnsureAuthDB(ctx context.Context, adminCfg config.AdminConfig, financeCfg config.FinanceConfig) error {
	if err := s.SeedAuthLookups(ctx); err != nil {
		return fmt.Errorf("auth lookups: %w", err)
	}
	if err := s.ensureSystem10Structure(ctx); err != nil {
		return fmt.Errorf("system 10 menus/forms: %w", err)
	}
	for _, level := range system10RoleNames {
		desc := roleDescriptions[level]
		if desc == "" {
			desc = level
		}
		if err := s.ensureRole(ctx, roleSpec{
			name:        level,
			code:        systemRoleCode(level),
			resource:    "system",
			description: desc,
			menuScoped:  true,
			perms:       systemRolePermissions[level],
		}); err != nil {
			return fmt.Errorf("role %s: %w", level, err)
		}
	}
	for _, r := range externalRoles {
		desc := roleDescriptions[r.level]
		if desc == "" {
			desc = r.level
		}
		if err := s.ensureRole(ctx, roleSpec{
			name:        roleName("compensation", r.level),
			resource:    "compensation",
			description: desc + " — compensation",
			perms:       r.perms,
		}); err != nil {
			return fmt.Errorf("role %s: %w", r.level, err)
		}
	}
	return s.ensureInitialUsers(ctx, adminCfg, financeCfg)
}

// ensureSystem10Structure — систем (system_id=10), цэс, маягт, маягтын эрх.
// Бүгд ЗӨВХӨН байхгүй үед нэмэгдэнэ; байгаа мөрийн нэр/холбоосыг хөндөхгүй.
//
// Эрхийн загварын 1-р давхарга нь ЦЭС (sd_menu / sd_menu_role): хэрэглэгчид
// эхлээд цэс харагдах эсэх шийдэгдэж, дараа нь роль (нөөц) ба эрх (үйлдэл)
// шалгагдана. Роль ↔ цэс/эрхийн холбоос нь ensureRole-д.
func (s *Seeder) ensureSystem10Structure(ctx context.Context) error {
	// Үйлдлүүд (sd_permission) эхэлж — маягтын эрх тэдэн рүү холбогдоно.
	seen := map[string]bool{}
	for _, key := range defaultPermissions {
		_, action := domain.SplitPermissionKey(key)
		if action == "" || seen[action] {
			continue
		}
		seen[action] = true
		if _, err := s.ensureAction(ctx, action); err != nil {
			return err
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_system (system_id, "name", description)
		VALUES (10, 'Газар чөлөөлөлт', 'Газар чөлөөлөлтийн систем')
		ON CONFLICT (system_id) DO NOTHING
	`); err != nil {
		return err
	}

	// Цэсүүд бүгд үндсэн түвшний (parent_code хоосон).
	for _, m := range system10MenuSeeds {
		if _, err := tx.Exec(ctx, `
			INSERT INTO sdplatform.sd_menu (code, "name", menu_url, sort_order, created_at, parent_code)
			VALUES (LEFT($1, 32), LEFT($2, 250), LEFT($3, 250), $4, NOW(), NULLIF($5, ''))
			ON CONFLICT (code) DO NOTHING`,
			m.code, m.name, m.url, m.sortOrder, ""); err != nil {
			return err
		}
	}

	// Маягтууд (sd_form) — цэс бүрт нэг. Нэр нь КОДЧИЛСОН латин нэр
	// (land_acquisition г.м.) — system10MenuSeeds.formName-ыг үз.
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
	return tx.Commit(ctx)
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

// Seeder — authdb (sdplatform / base / codelists схемүүд) дээр ажиллана.
type Seeder struct {
	db *pgxpool.Pool
}

func NewSeeder(db *pgxpool.Pool) *Seeder {
	return &Seeder{db: db}
}

// initialUser — хоосон authdb дээр үүсэх нэг хэрэглэгч ба түүний систем 10-ийн
// роль.
type initialUser struct {
	username, email, password string
	firstName, lastName       string
	roleName                  string
}

// systemAdminFirstName — системийн админ бүртгэлийн нэр. Бүртгэл нь ХҮН БИШ
// (системийн дүр) тул овог хоосон, нэрэнд хүний мэдээлэл бичихгүй; ажилтан
// (hr_employee) / хүн (bs_person)-тэй холбогдохгүй.
const systemAdminFirstName = "Системийн админ"

// ensureInitialUsers — анхны хэрэглэгчид:
//
//  1. СИСТЕМИЙН АДМИН (ADMIN_USERNAME / ADMIN_EMAIL / ADMIN_PASSWORD) — систем
//     10-д нэвтэрч чадах (систем 10-ийн ролтой) хэрэглэгч НЭГ Ч БАЙХГҮЙ үед.
//     Ажиллаж байгаа sdplatform санд холбогдоход ч үүснэ: тэнд хэрэглэгчид бий
//     ч хэн нь ч энэ системийн рольгүй тул өөрөөр нэвтэрч эрх олгох хүн байхгүй.
//     Нэр/имэйл нь өөр хүнд бүртгэлтэй бол тэр бүртгэлийг БУЛААХГҮЙ (роль
//     олгохгүй), шинээр ч үүсгэхгүй — нэвтрэх нэр давхцаж, аль бүртгэлээр
//     нэвтрэх нь тодорхойгүй болно. Оронд нь ERROR лог бичнэ.
//  2. САНХҮҮГИЙН МЭРГЭЖИЛТЭН — ЗӨВХӨН sd_user БҮРЭН ХООСОН (шинэ сан) үед.
//
// Бусад жишээ хэрэглэгч (employee, prof_org) үүсгэхгүй — админ API-аар үүсгэнэ.
func (s *Seeder) ensureInitialUsers(ctx context.Context, adminCfg config.AdminConfig, financeCfg config.FinanceConfig) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Олон instance зэрэг асахад хэрэглэгч давхар үүсэхээс сэргийлнэ.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('acquisition_backend:seed_admin'))`); err != nil {
		return err
	}
	var anyUser, anySystemUser bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM sdplatform.sd_user),
		       EXISTS (
				SELECT 1
				FROM sdplatform.sd_user u
				JOIN sdplatform.sd_user_role ur ON ur.user_id = u.user_id
				JOIN sdplatform.sd_system_role sr ON sr.role_id = ur.role_id AND sr.system_id = 10
				WHERE u.deleted_at IS NULL
		       )`).Scan(&anyUser, &anySystemUser); err != nil {
		return err
	}

	var users []initialUser
	if !anySystemUser {
		adminUsername := strings.TrimSpace(adminCfg.Username)
		if adminUsername == "" {
			adminUsername = "admin"
		}
		var taken bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sdplatform.sd_user
				WHERE deleted_at IS NULL
				  AND (lower(username) IN (lower($1), lower($2))
				       OR lower(email) IN (lower($1), lower($2)))
			)`, adminUsername, strings.TrimSpace(adminCfg.Email)).Scan(&taken); err != nil {
			return err
		}
		if taken {
			slog.Error("системийн админ үүссэнгүй: ADMIN_USERNAME эсвэл ADMIN_EMAIL өөр хэрэглэгчид бүртгэлтэй байна. "+
				"Систем 10-д нэвтэрч чадах хэрэглэгч алга — .env-д давхцахгүй ADMIN_USERNAME / ADMIN_EMAIL тохируулаад дахин асаана уу",
				"username", adminUsername, "email", adminCfg.Email)
		} else {
			users = append(users, initialUser{adminUsername, adminCfg.Email, adminCfg.Password,
				systemAdminFirstName, "", "admin"})
		}
	}
	if !anyUser {
		users = append(users, initialUser{financeCfg.Username, financeCfg.Email, financeCfg.Password,
			"Санхүүгийн", "мэргэжилтэн", roleName("compensation", "finance_specialist")})
	}
	if len(users) == 0 {
		return nil
	}

	for _, u := range users {
		hashed, err := hash.Password(u.password)
		if err != nil {
			return err
		}
		var userID domain.ID
		if err := tx.QueryRow(ctx, `
			INSERT INTO sdplatform.sd_user (username, email, password, firstname, lastname, created_at, updated_at, is_active)
			VALUES (LEFT($1, 50), LEFT(NULLIF(BTRIM($2::text), ''), 250), LEFT($3, 150), LEFT($4, 50), LEFT(NULLIF($5, ''), 50), NOW()::date, NOW(), true)
			RETURNING user_id`, u.username, u.email, hashed, u.firstName, u.lastName).Scan(&userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO sdplatform.sd_user_role (user_id, role_id, created_at, updated_at)
			SELECT $1, r.role_id, NOW(), NOW()
			FROM sdplatform.sd_role r
			WHERE r.name = $2
			  AND EXISTS (
				SELECT 1 FROM sdplatform.sd_system_role sr
				WHERE sr.role_id = r.role_id AND sr.system_id = 10
			  )
			ON CONFLICT DO NOTHING`, userID, u.roleName); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	for _, u := range users {
		slog.Info("initial user created", "username", u.username, "email", u.email, "role", u.roleName)
	}
	return nil
}
