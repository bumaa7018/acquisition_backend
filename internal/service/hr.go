package service

import (
	"context"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"
)

type OrganizationService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Organization, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Organization, error)
	Create(ctx context.Context, req OrganizationRequest) (*domain.Organization, error)
	Update(ctx context.Context, id domain.ID, req OrganizationRequest) (*domain.Organization, error)
	Delete(ctx context.Context, id domain.ID) error
}

type DepartmentService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Department, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Department, error)
	Create(ctx context.Context, req DepartmentRequest) (*domain.Department, error)
	Update(ctx context.Context, id domain.ID, req DepartmentRequest) (*domain.Department, error)
	Delete(ctx context.Context, id domain.ID) error
}

type PositionService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Position, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Position, error)
	Create(ctx context.Context, req PositionRequest) (*domain.Position, error)
	Update(ctx context.Context, id domain.ID, req PositionRequest) (*domain.Position, error)
	Delete(ctx context.Context, id domain.ID) error
}

type PersonService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Person, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Person, error)
	Create(ctx context.Context, req PersonRequest) (*domain.Person, error)
	Update(ctx context.Context, id domain.ID, req PersonRequest) (*domain.Person, error)
	Delete(ctx context.Context, id domain.ID) error
}

type EmployeeService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Employee, int64, error)
	GetByID(ctx context.Context, id domain.ID) (*domain.Employee, error)
	Create(ctx context.Context, req CreateEmployeeRequest) (*domain.Employee, error)
	Update(ctx context.Context, id domain.ID, req UpdateEmployeeRequest) (*domain.Employee, error)
	Delete(ctx context.Context, id domain.ID) error
}

// sd_organization-д зөвхөн name багана байдаг.
type OrganizationRequest struct {
	Name string
}

type DepartmentRequest struct {
	OrganizationID domain.ID
	ParentID       *domain.ID
	Code           *string
	Name           string
}

type PositionRequest struct {
	Code *string
	Name string
}

type PersonRequest struct {
	PersonType string
	RegisterNo string
	LastName   *string
	FirstName  *string
	LegalName  *string
	CivilID    *string
	BirthDate  *time.Time
	Gender     *string
	Phone      *string
	Email      *string
	Address    *string
}

// CreateEmployeeRequest — hr_employee-д байгууллагын багана байхгүй тул
// байгууллага нь хэлтсээр дамжина (department_id → hr_department.organization).
type CreateEmployeeRequest struct {
	PersonID     *domain.ID
	Person       *PersonRequest
	DepartmentID *domain.ID
	PositionID   domain.ID
	WorkEmail    *string
	WorkPhone    *string
	HiredAt      *time.Time
	ReleasedAt   *time.Time
}

type UpdateEmployeeRequest struct {
	PersonID     *domain.ID
	DepartmentID *domain.ID
	PositionID   *domain.ID
	WorkEmail    *string
	WorkPhone    *string
	HiredAt      *time.Time
	ReleasedAt   *time.Time
	IsActive     *bool
}

type organizationService struct {
	repo repository.OrganizationRepository
}
type departmentService struct {
	repo repository.DepartmentRepository
}
type positionService struct{ repo repository.PositionRepository }
type personService struct{ repo repository.PersonRepository }
type employeeService struct{ repo repository.EmployeeRepository }

func NewOrganizationService(repo repository.OrganizationRepository) OrganizationService {
	return &organizationService{repo: repo}
}
func NewDepartmentService(repo repository.DepartmentRepository) DepartmentService {
	return &departmentService{repo: repo}
}
func NewPositionService(repo repository.PositionRepository) PositionService {
	return &positionService{repo: repo}
}
func NewPersonService(repo repository.PersonRepository) PersonService {
	return &personService{repo: repo}
}
func NewEmployeeService(repo repository.EmployeeRepository) EmployeeService {
	return &employeeService{repo: repo}
}

// --- Organization ---

func (s *organizationService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Organization, int64, error) {
	return s.repo.ListOrganizations(ctx, f)
}
func (s *organizationService) GetByID(ctx context.Context, id domain.ID) (*domain.Organization, error) {
	return s.repo.GetOrganizationByID(ctx, id)
}
func (s *organizationService) Create(ctx context.Context, req OrganizationRequest) (*domain.Organization, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	o := &domain.Organization{Name: strings.TrimSpace(req.Name)}
	if err := s.repo.CreateOrganization(ctx, o); err != nil {
		return nil, err
	}
	return s.repo.GetOrganizationByID(ctx, o.ID)
}
func (s *organizationService) Update(ctx context.Context, id domain.ID, req OrganizationRequest) (*domain.Organization, error) {
	o, err := s.repo.GetOrganizationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		o.Name = strings.TrimSpace(req.Name)
	}
	if err := s.repo.UpdateOrganization(ctx, o); err != nil {
		return nil, err
	}
	return s.repo.GetOrganizationByID(ctx, id)
}
func (s *organizationService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.DeleteOrganization(ctx, id)
}

// --- Department ---

func (s *departmentService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Department, int64, error) {
	return s.repo.ListDepartments(ctx, f)
}
func (s *departmentService) GetByID(ctx context.Context, id domain.ID) (*domain.Department, error) {
	return s.repo.GetDepartmentByID(ctx, id)
}
func (s *departmentService) Create(ctx context.Context, req DepartmentRequest) (*domain.Department, error) {
	if req.OrganizationID == 0 || strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	d := &domain.Department{
		OrganizationID: req.OrganizationID,
		ParentID:       req.ParentID,
		Code:           req.Code,
		Name:           strings.TrimSpace(req.Name),
	}
	if err := s.repo.CreateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return s.repo.GetDepartmentByID(ctx, d.ID)
}
func (s *departmentService) Update(ctx context.Context, id domain.ID, req DepartmentRequest) (*domain.Department, error) {
	d, err := s.repo.GetDepartmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.OrganizationID != 0 {
		d.OrganizationID = req.OrganizationID
	}
	if strings.TrimSpace(req.Name) != "" {
		d.Name = strings.TrimSpace(req.Name)
	}
	d.ParentID, d.Code = req.ParentID, req.Code
	if err := s.repo.UpdateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return s.repo.GetDepartmentByID(ctx, id)
}
func (s *departmentService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.DeleteDepartment(ctx, id)
}

// --- Position ---

func (s *positionService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Position, int64, error) {
	return s.repo.ListPositions(ctx, f)
}
func (s *positionService) GetByID(ctx context.Context, id domain.ID) (*domain.Position, error) {
	return s.repo.GetPositionByID(ctx, id)
}
func (s *positionService) Create(ctx context.Context, req PositionRequest) (*domain.Position, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	p := &domain.Position{Code: req.Code, Name: strings.TrimSpace(req.Name)}
	if err := s.repo.CreatePosition(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPositionByID(ctx, p.ID)
}
func (s *positionService) Update(ctx context.Context, id domain.ID, req PositionRequest) (*domain.Position, error) {
	p, err := s.repo.GetPositionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		p.Name = strings.TrimSpace(req.Name)
	}
	p.Code = req.Code
	if err := s.repo.UpdatePosition(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPositionByID(ctx, id)
}
func (s *positionService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.DeletePosition(ctx, id)
}

// --- Person ---

func personFromRequest(req PersonRequest) *domain.Person {
	return &domain.Person{
		PersonType: req.PersonType,
		RegisterNo: strings.TrimSpace(req.RegisterNo),
		LastName:   req.LastName,
		FirstName:  req.FirstName,
		LegalName:  req.LegalName,
		CivilID:    req.CivilID,
		BirthDate:  req.BirthDate,
		Gender:     req.Gender,
		Phone:      req.Phone,
		Email:      req.Email,
		Address:    req.Address,
	}
}

func validatePerson(p *domain.Person) error {
	if p == nil || strings.TrimSpace(p.RegisterNo) == "" {
		return domain.ErrInvalidInput
	}
	// bs_person.person_register нь varchar(10) — түүнээс урт регистр багтахгүй.
	if len([]rune(p.RegisterNo)) > 10 {
		return domain.ErrInvalidInput
	}
	switch p.PersonType {
	case "citizen":
		if p.LastName == nil || strings.TrimSpace(*p.LastName) == "" || p.FirstName == nil || strings.TrimSpace(*p.FirstName) == "" {
			return domain.ErrInvalidInput
		}
	case "legal":
		if p.LegalName == nil || strings.TrimSpace(*p.LegalName) == "" {
			return domain.ErrInvalidInput
		}
	default:
		return domain.ErrInvalidInput
	}
	return nil
}

func (s *personService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Person, int64, error) {
	return s.repo.ListPersons(ctx, f)
}
func (s *personService) GetByID(ctx context.Context, id domain.ID) (*domain.Person, error) {
	return s.repo.GetPersonByID(ctx, id)
}
func (s *personService) Create(ctx context.Context, req PersonRequest) (*domain.Person, error) {
	p := personFromRequest(req)
	if err := validatePerson(p); err != nil {
		return nil, err
	}
	if err := s.repo.CreatePerson(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPersonByID(ctx, p.ID)
}
func (s *personService) Update(ctx context.Context, id domain.ID, req PersonRequest) (*domain.Person, error) {
	p := personFromRequest(req)
	p.ID = id
	if err := validatePerson(p); err != nil {
		return nil, err
	}
	if err := s.repo.UpdatePerson(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPersonByID(ctx, id)
}
func (s *personService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.DeletePerson(ctx, id)
}

// --- Employee ---

func employeeFromCreate(req CreateEmployeeRequest) *domain.Employee {
	return &domain.Employee{
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		WorkEmail:    req.WorkEmail,
		WorkPhone:    req.WorkPhone,
		HiredAt:      req.HiredAt,
		ReleasedAt:   req.ReleasedAt,
		IsActive:     true,
	}
}

func (s *employeeService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Employee, int64, error) {
	return s.repo.ListEmployees(ctx, f)
}
func (s *employeeService) GetByID(ctx context.Context, id domain.ID) (*domain.Employee, error) {
	return s.repo.GetEmployeeByID(ctx, id)
}
func (s *employeeService) Create(ctx context.Context, req CreateEmployeeRequest) (*domain.Employee, error) {
	if req.PositionID == 0 || (req.PersonID == nil && req.Person == nil) {
		return nil, domain.ErrInvalidInput
	}
	e := employeeFromCreate(req)
	var p *domain.Person
	if req.Person != nil {
		p = personFromRequest(*req.Person)
		if err := validatePerson(p); err != nil {
			return nil, err
		}
	} else {
		e.PersonID = *req.PersonID
	}
	if err := s.repo.CreateEmployeeWithPerson(ctx, e, p); err != nil {
		return nil, err
	}
	return s.repo.GetEmployeeByID(ctx, e.ID)
}
func (s *employeeService) Update(ctx context.Context, id domain.ID, req UpdateEmployeeRequest) (*domain.Employee, error) {
	e, err := s.repo.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.PersonID != nil {
		e.PersonID = *req.PersonID
	}
	if req.DepartmentID != nil {
		e.DepartmentID = req.DepartmentID
	}
	if req.PositionID != nil {
		e.PositionID = *req.PositionID
	}
	e.WorkEmail, e.WorkPhone, e.HiredAt, e.ReleasedAt = req.WorkEmail, req.WorkPhone, req.HiredAt, req.ReleasedAt
	if req.IsActive != nil {
		e.IsActive = *req.IsActive
	}
	if e.PersonID == 0 || e.PositionID == 0 {
		return nil, domain.ErrInvalidInput
	}
	if err := s.repo.UpdateEmployee(ctx, e); err != nil {
		return nil, err
	}
	return s.repo.GetEmployeeByID(ctx, id)
}
func (s *employeeService) Delete(ctx context.Context, id domain.ID) error {
	return s.repo.DeleteEmployee(ctx, id)
}
