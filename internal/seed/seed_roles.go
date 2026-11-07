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
// Гадаад legacy түвшингүүдэд (mika/finance_specialist) хуучин resource_level
// нэршлийг дэмжсэн хэвээр үлдээнэ. JWT-ийн `roles` claim нь Role.Level()-ээр
// хэвийн normalize хийгддэг.

// roleDescriptions — түвшин тус бүрийн тайлбар.
var roleDescriptions = map[string]string{
	"admin":                 "Системийн администратор",
	"senior_specialist":     "Ахлах мэргэжилтэн",
	"department_head":       "Хэлтсийн дарга",
	"employee":              "Мэргэжилтэн",
	"decision_registration": "Захирамж бүртгэл",
	"mika":                  "МИКА",
	"finance_specialist":    "Санхүүгийн мэргэжилтэн",
}

func roleName(resource, level string) string { return resource + "_" + level }

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
	case "department_head":
		return "A104"
	default:
		return ""
	}
}

// permissionCodes — ҮЙЛДЭЛ → sd_permission.code.
//
// ГУС-ийн бусад лавлахтай ижил хэлбэрээр (A + гурван орон) кодлоно; name нь
// үйлдлийн нэр (read/create/…) ХЭВЭЭР үлдэнэ. Эрхийн шалгалт нь sd_permission-ыг
// permission_id / name-ээр холбодог (UserRepo.GetPermissions, menu.go, role.go —
// бүгд `permission_id = … OR code = … OR name = …`).
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
// буцаадаг. sd_permission дээр name unique БИШ учир энд ЯГ нэрээр нэг
// мэдэгдлээр (атом) нэмж/уншина. Давхардсан мөр аль хэдийн байвал хамгийн
// бага permission_id-г буцаана.
//
// Байгаа мөрийг ХӨНДӨХГҮЙ: sd_permission-ийн үйлдлийн мөрүүд (read, create …)
// бусад системтэй ХУВААЛЦАГДДАГ. A001 хэлбэрийн код ЗӨВХӨН шинээр нэмэх
// мөрөнд бичигдэнэ.
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
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	if created {
		slog.Info("permission (action) created", "action", action)
	}
	return p, nil
}

// roleSpec — систем 10-д байх ёстой нэг роль ба түүний АНХНЫ эрх.
type roleSpec struct {
	name        string
	code        string // sd_role.code (UNIQUE); хоосон бол NULL
	resource    string // sd_role.role_type_code
	description string
	// menuScoped — эрх цэс тус бүрээр (sd_role_permission.menu_id) олгогдоно.
	// false бол хуучин гадаад ролиудын адил menu_id NULL.
	menuScoped bool
	perms      []string
}

// ensureRole — ролийг систем 10 дотроос НЭРЭЭР нь хайна; байвал ЮУ Ч ХИЙХГҮЙ
// (эрх/цэсийг нь админ UI-аас өөрчилсөн байж болно). Байхгүй бол нэг
// гүйлгээгээр: роль + систем 10 холбоос + анхны эрх + системийн бүх цэс.
//
// Өөр системийн ижил нэртэй ролийг ОГТ ХӨНДӨХГҮЙ — хайлт систем 10-оор
// шүүгдэнэ. sd_role.code (A100 …) өөр системд эзлэгдсэн бол кодгүй үүсгэнэ.
func (s *Seeder) ensureRole(ctx context.Context, spec roleSpec) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Олон instance зэрэг асахад ижил роль хоёр үүсэхээс сэргийлнэ.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('acquisition_backend:seed_role'))`); err != nil {
		return err
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sdplatform.sd_role r
			WHERE r.name = $1
			  AND EXISTS (
				SELECT 1 FROM sdplatform.sd_system_role sr
				WHERE sr.role_id = r.role_id AND sr.system_id = 10
			  )
		)`, spec.name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	code := spec.code
	if code != "" {
		var taken bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM sdplatform.sd_role WHERE code = $1)`, code).Scan(&taken); err != nil {
			return err
		}
		if taken {
			slog.Warn("role code taken by another system, creating without code", "name", spec.name, "code", code)
			code = ""
		}
	}

	var roleID domain.ID
	if err := tx.QueryRow(ctx, `
		INSERT INTO sdplatform.sd_role (code, name, role_type_code, description, created_at, updated_at)
		VALUES (LEFT(NULLIF($1, ''), 10), LEFT($2, 250), LEFT(NULLIF($3, ''), 45), $4, NOW(), NOW())
		RETURNING role_id`, code, spec.name, spec.resource, spec.description).Scan(&roleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_system_role (system_id, role_id, created_at)
		VALUES (10, $1, NOW())`, roleID); err != nil {
		return err
	}

	if spec.menuScoped {
		// Нөөц бүрийн үйлдлүүдийг цэсийнх нь маягтын эрхээр дамжуулан олгоно.
		var resources []string
		byResource := map[string][]string{}
		for _, key := range spec.perms {
			resource, action := domain.SplitPermissionKey(key)
			if resource == "" || action == "" {
				continue
			}
			if _, ok := byResource[resource]; !ok {
				resources = append(resources, resource)
			}
			byResource[resource] = append(byResource[resource], action)
		}
		for _, resource := range resources {
			// $1::int — заавал: DISTINCT нь төрөл нь тодорхойгүй параметрийг text
			// гэж шийддэг (42P08).
			if _, err := tx.Exec(ctx, `
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
				  AND p.name = ANY($3)`,
				roleID, domain.MenuCodeForResource(resource), byResource[resource]); err != nil {
				return err
			}
		}
	} else {
		for _, key := range spec.perms {
			_, action := domain.SplitPermissionKey(key)
			if action == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO sdplatform.sd_role_permission (role_id, permission_id, menu_id, created_at, updated_at)
				SELECT $1::int, p.permission_id, NULL, NOW(), NOW()
				FROM sdplatform.sd_permission p
				WHERE p.name = $2
				ORDER BY p.permission_id
				LIMIT 1`, roleID, action); err != nil {
				return err
			}
		}
	}

	// Системийн бүх цэсийг ролийн цэсний жагсаалтад холбоно (хуучин
	// 000024_system10_access_policy алхмын үр дүнтэй ижил).
	if _, err := tx.Exec(ctx, `
		INSERT INTO sdplatform.sd_menu_role (menu_id, role_id, created_at, updated_at)
		SELECT m.menu_id, $1, NOW(), NOW()
		FROM sdplatform.sd_menu m
		WHERE m.code = ANY($2)
		ON CONFLICT DO NOTHING`, roleID, system10MenuCodes()); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("system 10 role created", "name", spec.name, "role_id", roleID)
	return nil
}
