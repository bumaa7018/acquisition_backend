package seed

import (
	"context"
	"log/slog"
	"time"

	"acquisition_backend/internal/domain"
)

// sdplatform-ын эрхийн загварт нөөц (resource) нь РОЛЬ дээр
// (sd_role.role_type_code), үйлдэл (action) нь ЭРХ дээр (sd_permission.name)
// бүртгэгддэг. Тиймээс "land:read" гэсэн нэг эрх нь
//
//	role  = land_<түвшин>  (role_type_code = "land")
//	perm  = read
//
// гэсэн хос болж задарна. Нэг хэрэглэгч олон нөөцийн роль авснаар өмнөх
// "admin" гэсэн нэг роль эзэмшиж байсантай ЯГ ижил эрхийн багцтай болно —
// JWT-ийн `roles` нэхэмжлэлд ролийн ТҮВШИН ордог тул (land_admin → admin)
// маршрутын шалгалтууд ч хэвээр ажиллана.

// roleDescriptions — түвшин тус бүрийн тайлбар.
var roleDescriptions = map[string]string{
	"admin":              "Системийн администратор",
	"senior_specialist":  "Ахлах мэргэжилтэн",
	"employee":           "Энгийн ажилтан",
	"professional_org":   "Мэргэжлийн байгууллага",
	"mika":               "МИКА",
	"finance_specialist": "Санхүүгийн мэргэжилтэн",
}

func roleName(resource, level string) string { return resource + "_" + level }

// ensureAction — үйлдлийн эрхийг (read/create/update/delete ...) баталгаажуулна.
func (s *Seeder) ensureAction(ctx context.Context, action string) (*domain.Permission, error) {
	if p, err := s.permRepo.GetByName(ctx, action); err == nil {
		return p, nil
	}
	p := &domain.Permission{Name: action, Action: action, Code: action, CreatedAt: time.Now()}
	if err := s.permRepo.Create(ctx, p); err != nil {
		// Зэрэг ажиллахад давхардвал дахин уншина.
		if existing, e := s.permRepo.GetByName(ctx, action); e == nil {
			return existing, nil
		}
		return nil, err
	}
	slog.Info("permission (action) created", "action", action)
	return p, nil
}

// ensureRole — <resource>_<level> роль байгаа эсэхийг баталгаажуулна.
func (s *Seeder) ensureRole(ctx context.Context, resource, level string) (*domain.Role, error) {
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

// grantLevel — өгсөн түвшинд "resource:action" эрхүүдийг олгоно.
func (s *Seeder) grantLevel(ctx context.Context, level string, perms []string) error {
	for _, key := range perms {
		resource, action := domain.SplitPermissionKey(key)
		if resource == "" || action == "" {
			continue
		}
		role, err := s.ensureRole(ctx, resource, level)
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

// revokeOtherResources — тухайн түвшний ролиудаас зөвшөөрөгдсөн нөөцөөс
// бусдыг нь хасна (гадаад ролиуд зөвхөн compensation-той байх ёстой).
func (s *Seeder) revokeOtherResources(ctx context.Context, level string, allowed map[string]bool) error {
	roles, err := s.rolesForLevel(ctx, level)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if allowed[r.Resource] {
			continue
		}
		if _, err := s.db.Exec(ctx,
			`DELETE FROM sdplatform.sd_role_permission WHERE role_id = $1`, r.ID); err != nil {
			return err
		}
	}
	return nil
}
