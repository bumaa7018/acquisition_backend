package service

import (
	"context"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type RoleService interface {
	Create(ctx context.Context, req CreateRoleRequest) (*domain.Role, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Role, error)
	Update(ctx context.Context, id domain.ID, req UpdateRoleRequest) (*domain.Role, error)
	Delete(ctx context.Context, id domain.ID) error
	List(ctx context.Context) ([]*domain.Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID domain.ID) error
	RemovePermission(ctx context.Context, roleID, permissionID domain.ID) error
	SetMenuPermissions(ctx context.Context, roleID, menuID domain.ID, permissionIDs []domain.ID) error
	RemoveMenuPermissions(ctx context.Context, roleID, menuID domain.ID) error
	AssignToUser(ctx context.Context, userID, roleID domain.ID) error
	RemoveFromUser(ctx context.Context, userID, roleID domain.ID) error
}

type CreateRoleRequest struct {
	Name string `validate:"required"`
	// Code — заавал биш (sd_role.code, 10 тэмдэгт хүртэл). Бүх системд давтагдашгүй.
	Code        string
	Description string
}

type UpdateRoleRequest struct {
	Name        string
	Description string
}

type roleService struct {
	roleRepo repository.RoleRepository
}

func NewRoleService(roleRepo repository.RoleRepository) RoleService {
	return &roleService{roleRepo: roleRepo}
}

// errAdminRoleName — админы нэр зөвхөн системийн админы ролийнх.
var errAdminRoleName = domain.NewRuleError("«admin» / «Админ» нэр нь системийн админы ролийнх тул өөр ролийг ингэж нэрлэх боломжгүй.")

func (s *roleService) Create(ctx context.Context, req CreateRoleRequest) (*domain.Role, error) {
	// Админ эсэх нь НЭРЭЭР тодорхойлогддог тул давхардсан админ нэртэй роль
	// үүсгэвэл түүнийг авсан хэн ч бүх эрхтэй болно.
	if domain.IsAdminRoleName(req.Name) {
		return nil, errAdminRoleName
	}
	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if err := s.ensureUniqueRole(ctx, name, code, 0); err != nil {
		return nil, err
	}
	now := time.Now()
	role := &domain.Role{
		Name:        name,
		Code:        code,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) GetByID(ctx context.Context, id domain.ID) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *roleService) Update(ctx context.Context, id domain.ID, req UpdateRoleRequest) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" && req.Name != role.Name {
		// Админы ролийн нэрийг солибол админ эрх алга болно (эсвэл өөр нэрээр
		// давхардана); бусад ролийг админ болгож нэрлэвэл эрх нэмэгдэнэ.
		if role.IsSystemAdmin() || domain.IsAdminRoleName(role.Name) {
			return nil, domain.NewRuleError("Системийн админы ролийн нэрийг солих боломжгүй.")
		}
		if domain.IsAdminRoleName(req.Name) {
			return nil, errAdminRoleName
		}
		if err := s.ensureUniqueRole(ctx, strings.TrimSpace(req.Name), "", role.ID); err != nil {
			return nil, err
		}
		role.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	role.UpdatedAt = time.Now()
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) Delete(ctx context.Context, id domain.ID) error {
	return s.roleRepo.Delete(ctx, id)
}

func (s *roleService) List(ctx context.Context) ([]*domain.Role, error) {
	return s.roleRepo.List(ctx)
}

func (s *roleService) AssignPermission(ctx context.Context, roleID, permissionID domain.ID) error {
	return s.roleRepo.AssignPermission(ctx, roleID, permissionID)
}

func (s *roleService) RemovePermission(ctx context.Context, roleID, permissionID domain.ID) error {
	return s.roleRepo.RemovePermission(ctx, roleID, permissionID)
}

func (s *roleService) SetMenuPermissions(ctx context.Context, roleID, menuID domain.ID, permissionIDs []domain.ID) error {
	if roleID == 0 || menuID == 0 {
		return domain.ErrInvalidInput
	}
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return err
	}
	return s.roleRepo.SetMenuPermissions(ctx, roleID, menuID, permissionIDs)
}

func (s *roleService) RemoveMenuPermissions(ctx context.Context, roleID, menuID domain.ID) error {
	if roleID == 0 || menuID == 0 {
		return domain.ErrInvalidInput
	}
	return s.roleRepo.RemoveMenuPermissions(ctx, roleID, menuID)
}

// AssignToUser — роль олгоно.
//
// Мэргэжлийн байгууллагын ролийг ЭНД БҮРЭН ХААНА (domain.Role.IsAPIGrantable).
// Шалгалт handler-т биш service-д байх нь чухал: роль олгодог зам хоёр байдаг
// (хэрэглэгч үүсгэх, POST /users/:id/roles) ба цаашид нэмэгдэж болзошгүй тул
// бүгдийн дайрдаг ганц цэг дээр хаана.
func (s *roleService) AssignToUser(ctx context.Context, userID, roleID domain.ID) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if !role.IsAPIGrantable() {
		return domain.ErrForbidden
	}
	// Админы нэртэй боловч эрхгүй (хоосон) роль, эсвэл системийн админ биш
	// атлаа админ нэртэй роль хэнд ч олгогдохгүй.
	if domain.IsAdminRoleName(role.Name) && (!role.IsSystemAdmin() || len(role.Permissions) == 0) {
		return domain.ErrForbidden
	}
	return s.roleRepo.AssignToUser(ctx, userID, roleID)
}

func (s *roleService) RemoveFromUser(ctx context.Context, userID, roleID domain.ID) error {
	return s.roleRepo.RemoveFromUser(ctx, userID, roleID)
}

// maxRoleCodeLen — sd_role.code баганын урт (repo LEFT(..., 10)-тэй ижил).
const maxRoleCodeLen = 10

// ensureUniqueRole — ролийн нэр, код давхардахгүйг КОДООР шалгана (санд
// хязгаарлалт нэмэхгүй — authdb-ийн бүтцийг өөрчлөхгүй).
//
// Нэр: том/жижиг үсэг, илүү зай үл харгалзан
//   - системийн ролийн түвшин/монгол нэртэй (IsReservedRoleName) давхцахгүй —
//     хаалтууд ролийг НЭРЭЭР таньдаг тул ийм нэртэй роль тэр ролийн эрхийг авна;
//   - систем 10-ийн өөр ролийн НЭР эсвэл ТҮВШИН (токенд орох нэр)-тэй
//     давхцахгүй (жишээ нь "mika" нь compensation_mika-ийн түвшин).
//
// Код: бүх системийн sd_role дотор давхцахгүй, 10 тэмдэгтээс хэтрэхгүй.
// selfID — нэр солих үед тухайн роль өөрөө (0 = шинэ).
func (s *roleService) ensureUniqueRole(ctx context.Context, name, code string, selfID domain.ID) error {
	if domain.IsReservedRoleName(name) {
		return domain.NewRuleError("«" + name + "» нь системийн ролийн нэр тул өөр ролийг ингэж нэрлэх боломжгүй.")
	}
	want := domain.NormalizeRoleName(name)
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.ID == selfID {
			continue
		}
		if domain.NormalizeRoleName(r.Name) == want || domain.NormalizeRoleName(r.Level()) == want {
			return domain.NewRuleError("«" + name + "» нэртэй роль аль хэдийн бүртгэлтэй байна.")
		}
	}
	if code == "" {
		return nil
	}
	if len([]rune(code)) > maxRoleCodeLen {
		return domain.NewRuleError("Ролийн код 10 тэмдэгтээс хэтрэхгүй.")
	}
	taken, err := s.roleRepo.CodeTaken(ctx, code, selfID)
	if err != nil {
		return err
	}
	if taken {
		return domain.NewRuleError("«" + code + "» кодтой роль аль хэдийн бүртгэлтэй байна.")
	}
	return nil
}
