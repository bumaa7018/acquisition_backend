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
		"admin":             {"admin:read", "admin:create", "admin:update", "admin:delete"},
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
