package seed

import (
	"context"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"
)

// sdplatform-ын шинэ эрхийн загварт нэг системийн роль олон menu/permission
// эзэмшинэ: sd_role -> sd_role_permission(menu_id, permission_id). Permission
// key нь menu.code + ":" + sd_permission.name хэлбэрээр үүснэ.
//
// Гадаад legacy түвшингүүдэд (professional_org/mika/finance_specialist) хуучин
// resource_level нэршлийг дэмжсэн хэвээр үлдээнэ. JWT-ийн `roles` claim нь
// Role.Level()-ээр хэвийн normalize хийгддэг.

// roleDescriptions — түвшин тус бүрийн тайлбар.
var roleDescriptions = map[string]string{
	"admin":                 "Системийн администратор",
	"senior_specialist":     "Ахлах мэргэжилтэн",
	"employee":              "Мэргэжилтэн",
	"decision_registration": "Захирамж бүртгэл",
	"professional_org":      "Мэргэжлийн байгууллага",
	"mika":                  "МИКА",
	"finance_specialist":    "Санхүүгийн мэргэжилтэн",
}

func roleName(resource, level string) string { return resource + "_" + level }

func isSystemRoleLevel(level string) bool {
	switch level {
	case "admin", "senior_specialist", "employee", "decision_registration":
		return true
	default:
		return false
	}
}

func systemRoleCode(level string) string {
	switch level {
	case "admin":
		return "A100"
	case "senior_specialist":
		return "A101"
	case "employee":
		return "A102"
	case "decision_registration":
		return "A103"
	default:
		return ""
	}
}

// permissionCodes — ҮЙЛДЭЛ → sd_permission.code.
//
// ГУС-ийн бусад лавлахтай ижил хэлбэрээр (A + гурван орон) кодлоно; name нь
// үйлдлийн нэр (read/create/…) ХЭВЭЭР үлдэнэ. Эрхийн шалгалт нь sd_permission-ыг
// permission_id / name-ээр холбодог (UserRepo.GetPermissions, menu.go, role.go —
// бүгд `permission_id = … OR code = … OR name = …`) тул кодыг сольсноор
// хэрэглэгчийн эрх ӨӨРЧЛӨГДӨХГҮЙ.
var permissionCodes = map[string]string{
	"read":             "A001",
	"create":           "A002",
	"update":           "A003",
	"delete":           "A004",
	"structure:create": "A005",
}

// permissionCode — үйлдлийн код. Жагсаалтад байхгүй ШИНЭ үйлдэл нэмэгдвэл
// нэрийг нь хэвээр хэрэглэнэ (код давхцуулж санамсаргүй утга гаргахгүй).
func permissionCode(action string) string {
	if code, ok := permissionCodes[action]; ok {
		return code
	}
	return action
}

// ensureAction — үйлдлийн эрхийг (read/create/update/delete ...) баталгаажуулна.
//
// ЖИЧ: permRepo.GetByName нь нэрийг SplitPermissionKey-ээр ДАХИН таслдаг тул
// үйлдэл өөрөө цэг агуулбал ("structure:create") "create"-ийг хайж, буруу мөр
// буцаадаг. sd_permission дээр name unique БИШ учир өмнө нь энэ функц seed алхам
// бүрд шинэ давхардсан мөр үүсгэж байв. Тиймээс энд repo-гоор биш, ЯГ нэрээр
// нэг мэдэгдлээр (атом) нэмж/уншина. Давхардсан мөр аль хэдийн байвал хамгийн
// бага permission_id-г буцаана — SeedPermissionDedupe-ийн үлдээх мөртэй ижил.
func (s *Seeder) ensureAction(ctx context.Context, action string) (*domain.Permission, error) {
	code := permissionCode(action)
	p := &domain.Permission{Name: action, Action: action, Code: code}
	var created bool
	if err := s.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO sdplatform.sd_permission (code, name, created_at)
			SELECT LEFT($2::text, 45), LEFT($1::text, 250), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM sdplatform.sd_permission WHERE name = $1::text
			)
			RETURNING permission_id, created_at
		)
		SELECT permission_id, COALESCE(created_at::timestamptz, NOW()), TRUE FROM ins
		UNION ALL
		SELECT permission_id, COALESCE(created_at::timestamptz, NOW()), FALSE
		FROM sdplatform.sd_permission WHERE name = $1::text
		ORDER BY 3 DESC, 1
		LIMIT 1`, action, code).Scan(&p.ID, &p.CreatedAt, &created); err != nil {
		return nil, err
	}

	// Өмнөх seed нь code баганад ҮЙЛДЛИЙН НЭРийг бичдэг байсан (code='read').
	// Кодыг зөвхөн ТЭР тохиолдолд нөхнө — ГУС эсвэл админ өөр код тавьсан
	// бол хөндөхгүй.
	if _, err := s.db.Exec(ctx, `
		UPDATE sdplatform.sd_permission
		SET code = LEFT($2::text, 45)
		WHERE permission_id = $3
		  AND COALESCE(code, '') IN ('', $1::text)`, action, code, p.ID); err != nil {
		return nil, err
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	if created {
		slog.Info("permission (action) created", "action", action)
	}
	return p, nil
}

// ensureSystemRole — system_id=10-д харьяалагдах нэг системийн роль.
func (s *Seeder) ensureSystemRole(ctx context.Context, level string) (*domain.Role, error) {
	if r, err := s.roleRepo.GetByName(ctx, level); err == nil {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
			SELECT 10, $1, NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM sdplatform.sd_system_role
				WHERE system_id = 10 AND role_id = $1
			)`, r.ID); err != nil {
			return nil, err
		}
		return r, nil
	}
	now := time.Now()
	desc := roleDescriptions[level]
	if desc == "" {
		desc = level
	}
	r := &domain.Role{
		Code:        systemRoleCode(level),
		Name:        level,
		Resource:    "system",
		Description: desc,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.roleRepo.Create(ctx, r); err != nil {
		if existing, e := s.roleRepo.GetByName(ctx, level); e == nil {
			return existing, nil
		}
		return nil, err
	}
	slog.Info("system role created", "name", level)
	return r, nil
}

// ensureResourceRole — legacy/static external role: <resource>_<level>.
func (s *Seeder) ensureResourceRole(ctx context.Context, resource, level string) (*domain.Role, error) {
	name := roleName(resource, level)
	if r, err := s.roleRepo.GetByName(ctx, name); err == nil {
		return r, nil
	}
	now := time.Now()
	desc := roleDescriptions[level]
	if desc == "" {
		desc = level
	}
	r := &domain.Role{
		Name:        name,
		Resource:    resource,
		Description: desc + " — " + resource,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.roleRepo.Create(ctx, r); err != nil {
		if existing, e := s.roleRepo.GetByName(ctx, name); e == nil {
			return existing, nil
		}
		return nil, err
	}
	slog.Info("role created", "name", name, "resource", resource)
	return r, nil
}

// grantLevel — системийн үндсэн role-д menu permission олгоно. Гадаад legacy
// түвшин бол resource_level role-г хэвээр дэмжинэ.
func (s *Seeder) grantLevel(ctx context.Context, level string, perms []string) error {
	if isSystemRoleLevel(level) {
		return s.grantSystemLevel(ctx, level, perms)
	}
	return s.grantResourceLevel(ctx, level, perms)
}

func (s *Seeder) grantResourceLevel(ctx context.Context, level string, perms []string) error {
	for _, key := range perms {
		resource, action := domain.SplitPermissionKey(key)
		if resource == "" || action == "" {
			continue
		}
		role, err := s.ensureResourceRole(ctx, resource, level)
		if err != nil {
			return err
		}
		perm, err := s.ensureAction(ctx, action)
		if err != nil {
			return err
		}
		if err := s.roleRepo.AssignPermission(ctx, role.ID, perm.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) grantSystemLevel(ctx context.Context, level string, perms []string) error {
	role, err := s.ensureSystemRole(ctx, level)
	if err != nil {
		return err
	}
	byResource := map[string][]string{}
	for _, key := range perms {
		resource, action := domain.SplitPermissionKey(key)
		if resource == "" || action == "" {
			continue
		}
		if _, err := s.ensureAction(ctx, action); err != nil {
			return err
		}
		byResource[resource] = append(byResource[resource], action)
	}
	for resource, actions := range byResource {
		if err := s.grantMenuActions(ctx, role.ID, domain.MenuCodeForResource(resource), actions); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) grantMenuActions(ctx context.Context, roleID domain.ID, menuCode string, actions []string) error {
	if len(actions) == 0 {
		return nil
	}
	if _, err := s.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at, updated_at)
		SELECT m.menu_id, $1, NOW(), NOW()
		FROM sdplatform.sd_menu m
		WHERE m.code = $2
		ON CONFLICT DO NOTHING`, roleID, menuCode); err != nil {
		return err
	}
	// DISTINCT — нэг form дээр ижил эрхийн олон sd_form_permission мөр байвал
	// (NOT EXISTS нь мэдэгдлийн ӨМНӨХ төлвийг хардаг тул) ижил role_permission
	// мөр олон удаа орох эрсдэлтэй.
	//
	// $1::int — заавал. DISTINCT нь төрөл нь тодорхойгүй параметрийг text гэж
	// шийддэг бол доорх NOT EXISTS дэх rp.role_id = $1 нь бүхэл тоо гэж дүгнэдэг:
	// ил cast хийхгүй бол "inconsistent types deduced for parameter $1" (42P08) алдаа өгнө.
	_, err := s.db.Exec(ctx, `
		INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at, updated_at)
		SELECT DISTINCT $1::int, p.permission_id, m.menu_id, NOW(), NOW()
		FROM sdplatform.sd_menu m
		JOIN sdplatform.sd_form f ON f.code = m.code
		JOIN sdplatform.sd_form_permission fp ON fp.form_code = f.code
		JOIN sdplatform.sd_permission p ON (
			p.permission_id = fp.permission_id
			OR p.code = fp.permission_code
			OR p.name = fp.permission_code
		)
		WHERE m.code = $2
		  AND p.name = ANY($3)
		  AND NOT EXISTS (
			SELECT 1 FROM sdplatform.sd_role_permission rp
			WHERE rp.role_id = $1
			  AND rp.permission_id = p.permission_id
			  AND rp.menu_id = m.menu_id
		  )`, roleID, menuCode, actions)
	return err
}

// syncSystemLevel — системийн ролийн эрхийг баталгаажуулна.
//
// ЖИЧ: Урьд нь энэ функц эрхийг эхлээд УСТГААД дахин олгодог байсан. Seeder нь
// байгаа өгөгдлийг устгах/дарж бичихгүй (зөвхөн байхгүйг нэмнэ) дүрэмтэй тул
// устгах алхмуудыг хассан. grantSystemLevel өөрөө NOT EXISTS шалгалттай тул
// идемпотент хэвээр.
func (s *Seeder) syncSystemLevel(ctx context.Context, level string, perms []string) error {
	if _, err := s.ensureSystemRole(ctx, level); err != nil {
		return err
	}
	return s.grantSystemLevel(ctx, level, perms)
}

// rolesForLevel — тухайн түвшний бүх нөөцийн роль (хэрэглэгчид олгоход).
func (s *Seeder) rolesForLevel(ctx context.Context, level string) ([]*domain.Role, error) {
	all, err := s.roleRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []*domain.Role
	for _, r := range all {
		if r.Level() == level {
			out = append(out, r)
		}
	}
	return out, nil
}

// assignLevelToUser — түвшний бүх ролийг хэрэглэгчид олгоно.
func (s *Seeder) assignLevelToUser(ctx context.Context, userID domain.ID, levels ...string) error {
	for _, level := range levels {
		roles, err := s.rolesForLevel(ctx, level)
		if err != nil {
			return err
		}
		for _, r := range roles {
			if err := s.roleRepo.AssignToUser(ctx, userID, r.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// revokeOtherResources — гадаад ролиуд зөвхөн compensation нөөцтэй байх ёстой.
//
// ЖИЧ: Урьд нь энэ функц илүү эрхийг ШУУД УСТГАДАГ байсан. Seeder нь байгаа
// өгөгдлийг устгахгүй дүрэмтэй болсон тул зөвхөн АНХААРУУЛГА бичнэ — жинхэнэ
// sdplatform DB-д тэдгээр роль өөр системийнх байж болзошгүй. Seeder өөрөө
// эдгээр ролид compensation-оос өөр эрх ОЛГОДОГГҮЙ тул шинэ суулгацад энэ
// тохиолдол гарахгүй; гарвал гараар шалгаж засна.
func (s *Seeder) revokeOtherResources(ctx context.Context, level string, allowed map[string]bool) error {
	roles, err := s.rolesForLevel(ctx, level)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if allowed[r.Resource] {
			continue
		}
		var extra int
		if err := s.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM sdplatform.sd_role_permission WHERE role_id = $1`, r.ID).Scan(&extra); err != nil {
			return err
		}
		if extra > 0 {
			slog.Warn("гадаад роль зөвшөөрөгдөөгүй нөөцийн эрхтэй байна (seeder устгахгүй)",
				"role", r.Name, "resource", r.Resource, "permissions", extra)
		}
	}
	return nil
}
