package service

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/hash"
	"acquisition_backend/internal/repository"
)

type UserService interface {
	GetByID(ctx context.Context, id domain.ID) (*domain.User, error)
	// GetBySource — хэрэглэгчийг ЭХ СУРВАЛЖТАЙ нь хайна (domain.UserSource*).
	// Хоёр сангийн user_id мөргөлддөг тул ӨӨРИЙН профайл татахад заавал энэ.
	GetBySource(ctx context.Context, source string, id domain.ID) (*domain.User, error)
	List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error)
	Create(ctx context.Context, req CreateUserRequest) (*domain.User, error)
	Update(ctx context.Context, id domain.ID, req UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, id domain.ID) error
	// ChangePassword — админ бусад хэрэглэгчийн нууц үгийг сэргээнэ. Нууц үг
	// солигдсоны дараа тухайн хэрэглэгчийн бүх сесс хүчингүй болно.
	ChangePassword(ctx context.Context, id domain.ID, newPassword string) error
	// InvalidateSessions — хэрэглэгчийн бүх токеныг хүчингүй болгоно. Роль/эрх
	// нь JWT-д агуулагддаг тул эрх өөрчлөгдсөний дараа шууд хүчинтэй болгоход.
	InvalidateSessions(ctx context.Context, id domain.ID) error
	// GetPermissions — хэрэглэгчийн ролиудаас цуглуулсан эрхийн нэрс. Дуудагч
	// өөрөөсөө өндөр эрхтэй хэрэглэгчийг өөрчлөхийг хориглоход хэрэглэнэ.
	GetPermissions(ctx context.Context, id domain.ID) ([]string, error)
	// DepartmentOf — хэрэглэгчийн хэлтэс (хэлтсийн даргын хамрах хүрээг
	// шалгахад). Хэлтэсгүй бол nil.
	DepartmentOf(ctx context.Context, id domain.ID) (*domain.ID, error)
	// EnsureSystemAdminSlot — хэрэглэгчид (userID nil бол шинэ хэрэглэгчид)
	// системийн админ роль олгоход дээд хязгаар (domain.MaxSystemAdmins)
	// хэтрэхгүй эсэхийг шалгана. Аль хэдийн админ бол дахин тоологдохгүй.
	EnsureSystemAdminSlot(ctx context.Context, userID *domain.ID) error
}

type CreateUserRequest struct {
	Username   string `validate:"required"`
	Email      string `validate:"omitempty"`
	Password   string `validate:"required,min=8"`
	FirstName  string `validate:"required"`
	LastName   string `validate:"required"`
	Position   string
	EmployeeID *domain.ID
	Employee   *CreateEmployeeRequest
	IsActive   *bool
	// RequireDepartment — ажилтан ЗААВАЛ хэлтэстэй байна (системийн админ
	// роль олгохгүй бол handler тавина).
	RequireDepartment bool
	// ScopeDepartmentID — утгатай бол ажилтны хэлтэс ЯГ үүнтэй таарна
	// (хэлтсийн дарга зөвхөн өөрийн хэлтэст хэрэглэгч үүсгэнэ).
	ScopeDepartmentID *domain.ID
}

// checkDepartment — үүсгэх гэж буй хэрэглэгчийн ажилтны хэлтсийг шалгана.
func (req CreateUserRequest) checkDepartment(deptID *domain.ID) error {
	if (req.RequireDepartment || req.ScopeDepartmentID != nil) && deptID == nil {
		return domain.ErrDepartmentRequired
	}
	if req.ScopeDepartmentID != nil && *deptID != *req.ScopeDepartmentID {
		return domain.ErrOutsideDepartment
	}
	return nil
}

type UpdateUserRequest struct {
	Username  string `validate:"omitempty"`
	Email     string `validate:"omitempty"`
	FirstName string
	LastName  string
	Position  string
	IsActive  *bool
}

type userService struct {
	userRepo     repository.UserRepository
	employeeRepo repository.EmployeeRepository
	// loginSources — ӨӨРИЙН профайлыг (GetBySource) шийдэх эх сурвалжууд.
	// [0] нь ҮРГЭЛЖ authdb. AuthService-ийнхтэй ижил загвар.
	loginSources []repository.LoginUserRepository
}

// NewUserService — userRepo-гоос гадна нэмэлт репог хүлээж авна:
//
//	repository.EmployeeRepository   — HR ажилтны бүртгэл
//	repository.LoginUserRepository  — НЭМЭЛТ нэвтрэлтийн эх сурвалж (appdb)
func NewUserService(userRepo repository.UserRepository, extras ...any) UserService {
	s := &userService{
		userRepo:     userRepo,
		loginSources: []repository.LoginUserRepository{userRepo},
	}
	for _, extra := range extras {
		switch v := extra.(type) {
		case repository.EmployeeRepository:
			if s.employeeRepo == nil {
				s.employeeRepo = v
			}
		case repository.LoginUserRepository:
			s.loginSources = append(s.loginSources, v)
		}
	}
	return s
}

// GetBySource — ӨӨРИЙН профайл (`/users/me`).
//
// source-гүйгээр ЗӨВХӨН authdb-аас хайвал appdb дээрх хэрэглэгч (үнэлгээний
// байгууллагын ажилтан) олдохгүй 404 өгнө, эсвэл БҮР АЮУЛТАЙ нь — ижил
// дугаартай ӨӨР хүний (жишээ нь admin) бүртгэл буцна: хоёр сангийн user_id
// мөргөлддөг. Иймд токены `src` тэмдгээр чиглүүлнэ.
func (s *userService) GetBySource(ctx context.Context, source string, id domain.ID) (*domain.User, error) {
	if source == "" {
		source = domain.UserSourceAuth
	}
	if source == domain.UserSourceAuth {
		return s.GetByID(ctx, id)
	}
	for _, src := range s.loginSources {
		if src.Source() != source {
			continue
		}
		return src.GetByID(ctx, id)
	}
	// Танигдаагүй эх сурвалж — authdb руу fallback ХИЙХГҮЙ (өөр хүн рүү унана).
	return nil, domain.ErrNotFound
}

func (s *userService) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *userService) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, total, nil
}

func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
	if s.employeeRepo != nil && req.EmployeeID == nil && req.Employee == nil {
		return nil, domain.ErrInvalidInput
	}
	existing, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	hashed, err := hash.Password(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashed,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Position:  req.Position,
		IsActive:  req.IsActive == nil || *req.IsActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if s.employeeRepo != nil {
		if req.EmployeeID != nil {
			if existing, err := s.userRepo.GetByEmployeeID(ctx, *req.EmployeeID); err == nil && existing != nil {
				return nil, domain.ErrAlreadyExists
			} else if err != nil && err != domain.ErrNotFound {
				return nil, err
			}
			employee, err := s.employeeRepo.GetEmployeeByID(ctx, *req.EmployeeID)
			if err != nil {
				return nil, err
			}
			if err := req.checkDepartment(employee.DepartmentID); err != nil {
				return nil, err
			}
			applyEmployeeToUser(user, employee)
			if err := s.userRepo.Create(ctx, user); err != nil {
				return nil, err
			}
		} else {
			employee := employeeFromCreate(*req.Employee)
			if err := req.checkDepartment(employee.DepartmentID); err != nil {
				return nil, err
			}
			var person *domain.Person
			if req.Employee.Person != nil {
				person = personFromRequest(*req.Employee.Person)
				if err := validatePerson(person); err != nil {
					return nil, err
				}
			} else if req.Employee.PersonID != nil {
				employee.PersonID = *req.Employee.PersonID
			} else {
				return nil, domain.ErrInvalidInput
			}
			if employee.PositionID == 0 {
				return nil, domain.ErrInvalidInput
			}
			if err := s.userRepo.CreateWithEmployee(ctx, user, employee, person); err != nil {
				return nil, err
			}
			user.EmployeeID = &employee.ID
		}
	} else {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
	}
	user.Password = ""
	return user, nil
}

func applyEmployeeToUser(user *domain.User, employee *domain.Employee) {
	user.EmployeeID = &employee.ID
	user.Employee = employee
	user.Position = employee.PositionName
	if employee.Person == nil {
		return
	}
	if employee.Person.PersonType == "legal" {
		user.LastName = "Хуулийн этгээд"
		if employee.Person.LegalName != nil {
			user.FirstName = *employee.Person.LegalName
		}
		return
	}
	if employee.Person.LastName != nil {
		user.LastName = *employee.Person.LastName
	}
	if employee.Person.FirstName != nil {
		user.FirstName = *employee.Person.FirstName
	}
}

func (s *userService) Update(ctx context.Context, id domain.ID, req UpdateUserRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Position != "" {
		user.Position = req.Position
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (s *userService) Delete(ctx context.Context, id domain.ID) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}
	// Устгагдсан хэрэглэгчийн гарт байгаа токен хүчинтэй хэвээр байвал soft-delete
	// нь хандалтыг зогсоохгүй тул сессийг нь тасална.
	return s.userRepo.InvalidateSessions(ctx, id)
}

func (s *userService) ChangePassword(ctx context.Context, id domain.ID, newPassword string) error {
	// Хэрэглэгч байгаа (мөн устгагдаагүй) эсэхийг эхлээд шалгана.
	if _, err := s.userRepo.GetByID(ctx, id); err != nil {
		return err
	}
	hashed, err := hash.Password(newPassword)
	if err != nil {
		return err
	}
	if err := s.userRepo.UpdatePassword(ctx, id, hashed); err != nil {
		return err
	}
	// Хуучин нууц үгээр олгогдсон токенууд хүчинтэй хэвээр байвал нууц үг солих
	// нь халдлагыг зогсоохгүй — иймд бүх сессийг хүчингүй болгоно.
	return s.userRepo.InvalidateSessions(ctx, id)
}

func (s *userService) InvalidateSessions(ctx context.Context, id domain.ID) error {
	return s.userRepo.InvalidateSessions(ctx, id)
}

func (s *userService) DepartmentOf(ctx context.Context, id domain.ID) (*domain.ID, error) {
	return s.userRepo.DepartmentIDForUser(ctx, id)
}

func (s *userService) EnsureSystemAdminSlot(ctx context.Context, userID *domain.ID) error {
	n, err := s.userRepo.CountSystemAdmins(ctx, userID)
	if err != nil {
		return err
	}
	if n >= domain.MaxSystemAdmins {
		return domain.ErrSystemAdminLimit
	}
	return nil
}

func (s *userService) GetPermissions(ctx context.Context, id domain.ID) ([]string, error) {
	return s.userRepo.GetPermissions(ctx, id)
}
