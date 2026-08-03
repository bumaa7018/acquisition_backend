package seed

import (
	"context"
	"log/slog"
	"time"

	"acquisition_backend/internal/config"
	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var defaultPermissions = []domain.Permission{
	{Name: "users:read", Description: "Read users", Resource: "users", Action: "read"},
	{Name: "users:create", Description: "Create users", Resource: "users", Action: "create"},
	{Name: "users:update", Description: "Update users", Resource: "users", Action: "update"},
	{Name: "users:delete", Description: "Delete users", Resource: "users", Action: "delete"},
	{Name: "roles:read", Description: "Read roles", Resource: "roles", Action: "read"},
	{Name: "roles:create", Description: "Create roles", Resource: "roles", Action: "create"},
	{Name: "roles:update", Description: "Update roles", Resource: "roles", Action: "update"},
	{Name: "roles:delete", Description: "Delete roles", Resource: "roles", Action: "delete"},
	{Name: "permissions:read", Description: "Read permissions", Resource: "permissions", Action: "read"},
	{Name: "permissions:create", Description: "Create permissions", Resource: "permissions", Action: "create"},
	{Name: "permissions:delete", Description: "Delete permissions", Resource: "permissions", Action: "delete"},
	{Name: "admin:read", Description: "Системийн тохиргоо харах", Resource: "admin", Action: "read"},
	{Name: "admin:create", Description: "Лавлах мэдээлэл нэмэх", Resource: "admin", Action: "create"},
	{Name: "admin:update", Description: "Лавлах мэдээлэл засах", Resource: "admin", Action: "update"},
	{Name: "admin:delete", Description: "Лавлах мэдээлэл устгах", Resource: "admin", Action: "delete"},
	{Name: "hr:read", Description: "Хүний нөөцийн бүртгэл харах", Resource: "hr", Action: "read"},
	{Name: "hr:create", Description: "Хүний нөөцийн бүртгэл нэмэх", Resource: "hr", Action: "create"},
	{Name: "hr:update", Description: "Хүний нөөцийн бүртгэл засах", Resource: "hr", Action: "update"},
	{Name: "hr:delete", Description: "Хүний нөөцийн бүртгэл устгах", Resource: "hr", Action: "delete"},
	{Name: "audit:read", Description: "Ажилтны үйлдлийн лог харах", Resource: "audit", Action: "read"},
	{Name: "land:read", Description: "Read land acquisitions", Resource: "land", Action: "read"},
	{Name: "land:create", Description: "Create land acquisitions", Resource: "land", Action: "create"},
	{Name: "land:update", Description: "Update land acquisitions", Resource: "land", Action: "update"},
	{Name: "land:delete", Description: "Delete land acquisitions", Resource: "land", Action: "delete"},
	// Нөхөх олговрын (үнэлгээний) эрхүүд — гадаад ролиуд (мэрг. байгууллага,
	// МИКА, санхүүгийн мэргэжилтэн) ЗӨВХӨН эдгээр эрхтэй ажиллана.
	{Name: "compensation:read", Description: "Нөхөх олговор/үнэлгээ харах", Resource: "compensation", Action: "read"},
	{Name: "compensation:create", Description: "Нөхөх олговор/үнэлгээ бүртгэх", Resource: "compensation", Action: "create"},
	{Name: "compensation:update", Description: "Нөхөх олговор/үнэлгээ засах, төлөв шилжүүлэх", Resource: "compensation", Action: "update"},
}

func (s *Seeder) SeedHRPermissions(ctx context.Context) error {
	if _, err := s.seedPermissions(ctx); err != nil {
		return err
	}
	role, err := s.roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return err
	}
	for _, name := range []string{"hr:read", "hr:create", "hr:update", "hr:delete"} {
		perm, err := s.permRepo.GetByName(ctx, name)
		if err != nil {
			return err
		}
		if err := s.roleRepo.AssignPermission(ctx, role.ID, perm.ID); err != nil {
			return err
		}
	}
	slog.Info("hr permissions granted", "role", "admin")
	return nil
}

func (s *Seeder) SeedHRBackfill(ctx context.Context) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID uuid.UUID
	err = tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO organization (name, short_name, register_no, source)
			SELECT 'Нийслэлийн Газрын алба', 'НГА', 'HR-BACKFILL-ORG', 'backfill'
			WHERE NOT EXISTS (SELECT 1 FROM organization WHERE register_no='HR-BACKFILL-ORG' AND deleted_at IS NULL)
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM organization WHERE register_no='HR-BACKFILL-ORG' AND deleted_at IS NULL
		LIMIT 1`).Scan(&orgID)
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `SELECT id, username, email, first_name, last_name, COALESCE(position,'Ажилтан') FROM users WHERE deleted_at IS NULL AND employee_id IS NULL`)
	if err != nil {
		return err
	}
	type backfillUser struct {
		id           uuid.UUID
		username     string
		email        string
		firstName    string
		lastName     string
		positionName string
	}
	var users []backfillUser
	for rows.Next() {
		var u backfillUser
		if err := rows.Scan(&u.id, &u.username, &u.email, &u.firstName, &u.lastName, &u.positionName); err != nil {
			return err
		}
		users = append(users, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, u := range users {
		var posID uuid.UUID
		if err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO position (name)
				SELECT $1
				WHERE NOT EXISTS (SELECT 1 FROM position WHERE name=$1 AND deleted_at IS NULL)
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM position WHERE name=$1 AND deleted_at IS NULL
			LIMIT 1`, u.positionName).Scan(&posID); err != nil {
			return err
		}
		var personID uuid.UUID
		registerNo := "EMP-" + u.username
		if err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO person (person_type, register_no, last_name, first_name, email, source, created_by, updated_by)
				SELECT 'citizen', $1, $2, $3, $4, 'backfill', 'seed', 'seed'
				WHERE NOT EXISTS (SELECT 1 FROM person WHERE register_no=$1 AND deleted_at IS NULL)
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM person WHERE register_no=$1 AND deleted_at IS NULL
			LIMIT 1`, registerNo, u.lastName, u.firstName, u.email).Scan(&personID); err != nil {
			return err
		}
		var employeeID uuid.UUID
		employeeNo := "EMP-" + u.username
		if err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO employee (person_id, organization_id, position_id, employee_no, work_email, created_by, updated_by)
				SELECT $1, $2, $3, $4, $5, 'seed', 'seed'
				WHERE NOT EXISTS (SELECT 1 FROM employee WHERE person_id=$1 AND deleted_at IS NULL)
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM employee WHERE person_id=$1 AND deleted_at IS NULL
			LIMIT 1`, personID, orgID, posID, employeeNo, u.email).Scan(&employeeID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET employee_id=$1, position=$2, updated_at=NOW() WHERE id=$3 AND employee_id IS NULL`, employeeID, u.positionName, u.id); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("hr backfill applied")
	return nil
}

// hrSampleDepartments — Нийслэлийн Газрын албаны бүтэц (жишээ өгөгдөл).
// parent_id хэрэглэхгүй: UI-ийн "Алба, хэлтэс" форм нь зөвхөн
// organization_id/name/code/sort_order-ыг тохируулдаг тул бүтцийг хавтгай
// байлгав — үгүй бол UI-аас засах боломжгүй мөр үүснэ.
var hrSampleDepartments = []struct {
	code      string
	name      string
	sortOrder int
}{
	{"ADM", "Захиргаа, удирдлагын хэлтэс", 1},
	{"GCH", "Газар чөлөөлөлтийн хэлтэс", 2},
	{"UNO", "Үнэлгээ, нөхөх олговрын хэлтэс", 3},
	{"GZZ", "Геодези, зураг зүйн хэлтэс", 4},
	{"SHO", "Санхүү, хөрөнгө оруулалтын хэлтэс", 5},
	{"HSH", "Хяналт, шинжилгээний хэлтэс", 6},
	{"MTH", "Мэдээллийн технологийн хэлтэс", 7},
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
func (s *Seeder) SeedHRDepartments(ctx context.Context) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID uuid.UUID
	if err := tx.QueryRow(ctx,
		`SELECT id FROM organization WHERE register_no='HR-BACKFILL-ORG' AND deleted_at IS NULL LIMIT 1`,
	).Scan(&orgID); err != nil {
		// Байгууллага байхгүй бол 000019 ажиллаагүй байна — холбох ажилтан ч байхгүй.
		slog.Warn("hr departments skipped: base organization not found", "error", err)
		return nil
	}

	deptIDs := make(map[string]uuid.UUID, len(hrSampleDepartments))
	for _, d := range hrSampleDepartments {
		var id uuid.UUID
		if err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO department (organization_id, code, name, sort_order)
				SELECT $1, $2, $3, $4
				WHERE NOT EXISTS (
					SELECT 1 FROM department
					WHERE organization_id=$1 AND name=$3 AND deleted_at IS NULL
				)
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM department
			WHERE organization_id=$1 AND name=$3 AND deleted_at IS NULL
			LIMIT 1`, orgID, d.code, d.name, d.sortOrder).Scan(&id); err != nil {
			return err
		}
		deptIDs[d.code] = id
	}

	rows, err := tx.Query(ctx, `
		SELECT e.id, COALESCE(p.name, '')
		FROM employee e
		JOIN position p ON p.id = e.position_id
		WHERE e.deleted_at IS NULL AND e.department_id IS NULL AND e.organization_id = $1`, orgID)
	if err != nil {
		return err
	}
	type pending struct {
		id       uuid.UUID
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
			`UPDATE employee SET department_id=$1, updated_at=NOW(), updated_by='seed'
			 WHERE id=$2 AND department_id IS NULL`, deptIDs[code], e.id); err != nil {
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

// SeedPermissionFixes — эрхийн нэршлийн зөрчлүүдийг засна:
//  1. admin:* permission үүсгэж admin роль-д олгоно (лавлах CRUD + Тохиргоо цэс)
//  2. users:read-ийг senior_specialist-д олгоно (ажилтан/байгууллага сонгох UI)
//  3. Орфон permission-уудыг устгана (acquisition.create, valuation:create/update)
func (s *Seeder) SeedPermissionFixes(ctx context.Context) error {
	if _, err := s.seedPermissions(ctx); err != nil {
		return err
	}
	grants := map[string][]string{
		"admin":             {"admin:read", "admin:create", "admin:update", "admin:delete", "audit:read"},
		"senior_specialist": {"users:read"},
	}
	for roleName, permNames := range grants {
		role, err := s.roleRepo.GetByName(ctx, roleName)
		if err != nil {
			continue
		}
		for _, permName := range permNames {
			perm, err := s.permRepo.GetByName(ctx, permName)
			if err != nil {
				return err
			}
			_ = s.roleRepo.AssignPermission(ctx, role.ID, perm.ID)
		}
	}
	// Орфон permission-ууд: ямар ч маршрут ашиглахаа больсон
	if _, err := s.db.Exec(ctx, `
		DELETE FROM role_permissions rp USING permissions p
		WHERE rp.permission_id = p.id AND p.name IN ('acquisition.create', 'valuation:create', 'valuation:update')
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `
		DELETE FROM permissions WHERE name IN ('acquisition.create', 'valuation:create', 'valuation:update')
	`); err != nil {
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
func (s *Seeder) SeedAdminAllPermissions(ctx context.Context) error {
	permIDs, err := s.seedPermissions(ctx)
	if err != nil {
		return err
	}
	role, err := s.roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return err
	}
	for _, pid := range permIDs {
		if err := s.roleRepo.AssignPermission(ctx, role.ID, pid); err != nil {
			return err
		}
	}
	slog.Info("admin role granted all permissions", "count", len(permIDs))
	return nil
}

func (s *Seeder) SeedAuditPermission(ctx context.Context) error {
	if _, err := s.seedPermissions(ctx); err != nil {
		return err
	}
	role, err := s.roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return err
	}
	perm, err := s.permRepo.GetByName(ctx, "audit:read")
	if err != nil {
		return err
	}
	if err := s.roleRepo.AssignPermission(ctx, role.ID, perm.ID); err != nil {
		return err
	}
	slog.Info("audit permission granted", "role", role.Name, "permission", perm.Name)
	return nil
}

// SeedCompensationAccess — compensation:* permission-уудыг үүсгэж, гадаад
// ролиудад ЗӨВХӨН тэдгээрийг олгоно (бусад permission-уудыг нь хураана).
func (s *Seeder) SeedCompensationAccess(ctx context.Context) error {
	if _, err := s.seedPermissions(ctx); err != nil {
		return err
	}
	for name, cfg := range externalRolePermissions {
		role, err := s.roleRepo.GetByName(ctx, name)
		if err != nil {
			now := time.Now()
			role = &domain.Role{
				ID:          uuid.New(),
				Name:        name,
				Description: cfg.description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := s.roleRepo.Create(ctx, role); err != nil {
				return err
			}
			slog.Info("external role created", "name", name)
		}
		for _, permName := range cfg.perms {
			perm, err := s.permRepo.GetByName(ctx, permName)
			if err != nil {
				return err
			}
			_ = s.roleRepo.AssignPermission(ctx, role.ID, perm.ID)
		}
		// "Өөр эрх олгогдохгүй" — compensation:*-ээс бусад бүх permission-ийг хураана.
		if _, err := s.db.Exec(ctx, `
			DELETE FROM role_permissions rp
			USING permissions p
			WHERE rp.role_id = $1 AND rp.permission_id = p.id
			  AND p.name NOT LIKE 'compensation:%'
		`, role.ID); err != nil {
			return err
		}
		slog.Info("external role permissions synced", "role", name, "perms", cfg.perms)
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
	permIDs, err := s.seedPermissions(ctx)
	if err != nil {
		return err
	}
	adminRole, err := s.seedAdminRole(ctx, permIDs)
	if err != nil {
		return err
	}
	seniorRole, err := s.seedSeniorSpecialistRole(ctx)
	if err != nil {
		return err
	}
	employeeRole, err := s.seedEmployeeRole(ctx)
	if err != nil {
		return err
	}
	if err := s.seedAdminUser(ctx, adminCfg, adminRole.ID, seniorRole.ID); err != nil {
		return err
	}
	return s.seedEmployeeUser(ctx, employeeCfg, employeeRole.ID)
}

func (s *Seeder) seedPermissions(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, p := range defaultPermissions {
		existing, err := s.permRepo.GetByName(ctx, p.Name)
		if err == nil {
			ids = append(ids, existing.ID)
			continue
		}
		p.ID = uuid.New()
		p.CreatedAt = time.Now()
		if err := s.permRepo.Create(ctx, &p); err != nil {
			return nil, err
		}
		ids = append(ids, p.ID)
		slog.Info("permission created", "name", p.Name)
	}
	return ids, nil
}

func (s *Seeder) seedAdminRole(ctx context.Context, permIDs []uuid.UUID) (*domain.Role, error) {
	existing, err := s.roleRepo.GetByName(ctx, "admin")
	if err == nil {
		for _, pid := range permIDs {
			_ = s.roleRepo.AssignPermission(ctx, existing.ID, pid)
		}
		return existing, nil
	}

	now := time.Now()
	role := &domain.Role{
		ID:          uuid.New(),
		Name:        "admin",
		Description: "System administrator with full access",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	for _, pid := range permIDs {
		if err := s.roleRepo.AssignPermission(ctx, role.ID, pid); err != nil {
			return nil, err
		}
	}
	slog.Info("admin role created")
	return role, nil
}

func (s *Seeder) seedSeniorSpecialistRole(ctx context.Context) (*domain.Role, error) {
	permNames := []string{"land:read", "land:create", "land:update"}
	var permIDs []uuid.UUID
	for _, name := range permNames {
		perm, err := s.permRepo.GetByName(ctx, name)
		if err != nil {
			return nil, err
		}
		permIDs = append(permIDs, perm.ID)
	}

	role, err := s.roleRepo.GetByName(ctx, "senior_specialist")
	if err != nil {
		now := time.Now()
		role = &domain.Role{
			ID:          uuid.New(),
			Name:        "senior_specialist",
			Description: "Ахлах мэргэжилтэн",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.roleRepo.Create(ctx, role); err != nil {
			return nil, err
		}
		slog.Info("senior specialist role created")
	}

	for _, pid := range permIDs {
		if err := s.roleRepo.AssignPermission(ctx, role.ID, pid); err != nil {
			return nil, err
		}
	}
	return role, nil
}

func (s *Seeder) seedEmployeeRole(ctx context.Context) (*domain.Role, error) {
	perm, err := s.permRepo.GetByName(ctx, "land:read")
	if err != nil {
		return nil, err
	}

	role, err := s.roleRepo.GetByName(ctx, "employee")
	if err != nil {
		now := time.Now()
		role = &domain.Role{
			ID:          uuid.New(),
			Name:        "employee",
			Description: "Энгийн ажилтан",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.roleRepo.Create(ctx, role); err != nil {
			return nil, err
		}
		slog.Info("employee role created")
	}

	if err := s.roleRepo.AssignPermission(ctx, role.ID, perm.ID); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Seeder) seedAdminUser(ctx context.Context, cfg config.AdminConfig, roleIDs ...uuid.UUID) error {
	existing, err := s.userRepo.GetByEmail(ctx, cfg.Email)
	if err == nil {
		for _, roleID := range roleIDs {
			_ = s.roleRepo.AssignToUser(ctx, existing.ID, roleID)
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
		ID:        uuid.New(),
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
	for _, roleID := range roleIDs {
		if err := s.roleRepo.AssignToUser(ctx, user.ID, roleID); err != nil {
			return err
		}
	}
	slog.Info("admin user created", "email", cfg.Email)
	return nil
}

func (s *Seeder) seedEmployeeUser(ctx context.Context, cfg config.EmployeeConfig, roleID uuid.UUID) error {
	existing, err := s.userRepo.GetByEmail(ctx, cfg.Email)
	if err == nil {
		_ = s.roleRepo.AssignToUser(ctx, existing.ID, roleID)
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
		ID:        uuid.New(),
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
	if err := s.roleRepo.AssignToUser(ctx, user.ID, roleID); err != nil {
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
func (s *Seeder) seedRoleUser(ctx context.Context, roleName, username, email, password, firstName, lastName, position string) error {
	role, err := s.roleRepo.GetByName(ctx, roleName)
	if err != nil {
		slog.Warn("external role not found, skipping user seed", "role", roleName)
		return nil
	}

	if existing, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		_ = s.roleRepo.AssignToUser(ctx, existing.ID, role.ID)
		return nil
	}

	hashed, err := hash.Password(password)
	if err != nil {
		return err
	}
	now := time.Now()
	user := &domain.User{
		ID:        uuid.New(),
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
	if err := s.roleRepo.AssignToUser(ctx, user.ID, role.ID); err != nil {
		return err
	}
	slog.Info("external user created", "email", email, "role", roleName)
	return nil
}
