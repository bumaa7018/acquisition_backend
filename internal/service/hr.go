package service

import (
	"context"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

type OrganizationService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Organization, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	Create(ctx context.Context, req OrganizationRequest) (*domain.Organization, error)
	Update(ctx context.Context, id uuid.UUID, req OrganizationRequest) (*domain.Organization, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type DepartmentService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Department, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Department, error)
	Create(ctx context.Context, req DepartmentRequest) (*domain.Department, error)
	Update(ctx context.Context, id uuid.UUID, req DepartmentRequest) (*domain.Department, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PositionService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Position, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Position, error)
	Create(ctx context.Context, req PositionRequest) (*domain.Position, error)
	Update(ctx context.Context, id uuid.UUID, req PositionRequest) (*domain.Position, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PersonService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Person, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	Create(ctx context.Context, req PersonRequest) (*domain.Person, error)
	Update(ctx context.Context, id uuid.UUID, req PersonRequest) (*domain.Person, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type EmployeeService interface {
	List(ctx context.Context, filter domain.HRFilter) ([]*domain.Employee, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)
	Create(ctx context.Context, req CreateEmployeeRequest) (*domain.Employee, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateEmployeeRequest) (*domain.Employee, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type OrganizationRequest struct {
	ParentID   *uuid.UUID
	Name       string
	ShortName  *string
	RegisterNo *string
	Address    *string
	Phone      *string
	Email      *string
	ExternalID *string
	Source     string
	IsActive   *bool
}

type DepartmentRequest struct {
	OrganizationID uuid.UUID
	ParentID       *uuid.UUID
	Code           *string
	Name           string
	SortOrder      int
	IsActive       *bool
}

type PositionRequest struct {
	Code      *string
	Name      string
	SortOrder int
	IsActive  *bool
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
	ExternalID *string
	Source     string
	IsActive   *bool
	CreatedBy  *string
	UpdatedBy  *string
}

type CreateEmployeeRequest struct {
	PersonID       *uuid.UUID
	Person         *PersonRequest
	OrganizationID uuid.UUID
	DepartmentID   *uuid.UUID
	PositionID     uuid.UUID
	EmployeeNo     *string
	WorkEmail      *string
	WorkPhone      *string
	HiredAt        *time.Time
	ReleasedAt     *time.Time
	IsActive       *bool
	CreatedBy      *string
	UpdatedBy      *string
}

type UpdateEmployeeRequest struct {
	PersonID       *uuid.UUID
	OrganizationID *uuid.UUID
	DepartmentID   *uuid.UUID
	PositionID     *uuid.UUID
	EmployeeNo     *string
	WorkEmail      *string
	WorkPhone      *string
	HiredAt        *time.Time
	ReleasedAt     *time.Time
	IsActive       *bool
	UpdatedBy      *string
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

func activeValue(v *bool) bool { return v == nil || *v }

func (s *organizationService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Organization, int64, error) {
	return s.repo.ListOrganizations(ctx, f)
}
func (s *organizationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	return s.repo.GetOrganizationByID(ctx, id)
}
func (s *organizationService) Create(ctx context.Context, req OrganizationRequest) (*domain.Organization, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	o := &domain.Organization{ParentID: req.ParentID, Name: strings.TrimSpace(req.Name), ShortName: req.ShortName, RegisterNo: req.RegisterNo, Address: req.Address, Phone: req.Phone, Email: req.Email, ExternalID: req.ExternalID, Source: req.Source, IsActive: activeValue(req.IsActive)}
	if err := s.repo.CreateOrganization(ctx, o); err != nil {
		return nil, err
	}
	return s.repo.GetOrganizationByID(ctx, o.ID)
}
func (s *organizationService) Update(ctx context.Context, id uuid.UUID, req OrganizationRequest) (*domain.Organization, error) {
	o, err := s.repo.GetOrganizationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		o.Name = strings.TrimSpace(req.Name)
	}
	o.ParentID, o.ShortName, o.RegisterNo, o.Address, o.Phone, o.Email, o.ExternalID = req.ParentID, req.ShortName, req.RegisterNo, req.Address, req.Phone, req.Email, req.ExternalID
	if req.Source != "" {
		o.Source = req.Source
	}
	if req.IsActive != nil {
		o.IsActive = *req.IsActive
	}
	if err := s.repo.UpdateOrganization(ctx, o); err != nil {
		return nil, err
	}
	return s.repo.GetOrganizationByID(ctx, id)
}
func (s *organizationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteOrganization(ctx, id)
}

func (s *departmentService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Department, int64, error) {
	return s.repo.ListDepartments(ctx, f)
}
func (s *departmentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	return s.repo.GetDepartmentByID(ctx, id)
}
func (s *departmentService) Create(ctx context.Context, req DepartmentRequest) (*domain.Department, error) {
	if req.OrganizationID == uuid.Nil || strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	d := &domain.Department{OrganizationID: req.OrganizationID, ParentID: req.ParentID, Code: req.Code, Name: strings.TrimSpace(req.Name), SortOrder: req.SortOrder, IsActive: activeValue(req.IsActive)}
	if err := s.repo.CreateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return s.repo.GetDepartmentByID(ctx, d.ID)
}
func (s *departmentService) Update(ctx context.Context, id uuid.UUID, req DepartmentRequest) (*domain.Department, error) {
	d, err := s.repo.GetDepartmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.OrganizationID != uuid.Nil {
		d.OrganizationID = req.OrganizationID
	}
	if strings.TrimSpace(req.Name) != "" {
		d.Name = strings.TrimSpace(req.Name)
	}
	d.ParentID, d.Code, d.SortOrder = req.ParentID, req.Code, req.SortOrder
	if req.IsActive != nil {
		d.IsActive = *req.IsActive
	}
	if err := s.repo.UpdateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return s.repo.GetDepartmentByID(ctx, id)
}
func (s *departmentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDepartment(ctx, id)
}

func (s *positionService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Position, int64, error) {
	return s.repo.ListPositions(ctx, f)
}
func (s *positionService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Position, error) {
	return s.repo.GetPositionByID(ctx, id)
}
func (s *positionService) Create(ctx context.Context, req PositionRequest) (*domain.Position, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	p := &domain.Position{Code: req.Code, Name: strings.TrimSpace(req.Name), SortOrder: req.SortOrder, IsActive: activeValue(req.IsActive)}
	if err := s.repo.CreatePosition(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPositionByID(ctx, p.ID)
}
func (s *positionService) Update(ctx context.Context, id uuid.UUID, req PositionRequest) (*domain.Position, error) {
	p, err := s.repo.GetPositionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		p.Name = strings.TrimSpace(req.Name)
	}
	p.Code, p.SortOrder = req.Code, req.SortOrder
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}
	if err := s.repo.UpdatePosition(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPositionByID(ctx, id)
}
func (s *positionService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePosition(ctx, id)
}

func personFromRequest(req PersonRequest) *domain.Person {
	return &domain.Person{PersonType: req.PersonType, RegisterNo: strings.TrimSpace(req.RegisterNo), LastName: req.LastName, FirstName: req.FirstName, LegalName: req.LegalName, CivilID: req.CivilID, BirthDate: req.BirthDate, Gender: req.Gender, Phone: req.Phone, Email: req.Email, Address: req.Address, ExternalID: req.ExternalID, Source: req.Source, IsActive: activeValue(req.IsActive), CreatedBy: req.CreatedBy, UpdatedBy: req.UpdatedBy}
}

func validatePerson(p *domain.Person) error {
	if p == nil || strings.TrimSpace(p.RegisterNo) == "" {
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
func (s *personService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error) {
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
func (s *personService) Update(ctx context.Context, id uuid.UUID, req PersonRequest) (*domain.Person, error) {
	p := personFromRequest(req)
	p.ID = id
	if err := validatePerson(p); err != nil {
		return nil, err
	}
	p.UpdatedBy = req.UpdatedBy
	if err := s.repo.UpdatePerson(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPersonByID(ctx, id)
}
func (s *personService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePerson(ctx, id)
}

func employeeFromCreate(req CreateEmployeeRequest) *domain.Employee {
	return &domain.Employee{OrganizationID: req.OrganizationID, DepartmentID: req.DepartmentID, PositionID: req.PositionID, EmployeeNo: req.EmployeeNo, WorkEmail: req.WorkEmail, WorkPhone: req.WorkPhone, HiredAt: req.HiredAt, ReleasedAt: req.ReleasedAt, IsActive: activeValue(req.IsActive), CreatedBy: req.CreatedBy, UpdatedBy: req.UpdatedBy}
}

func (s *employeeService) List(ctx context.Context, f domain.HRFilter) ([]*domain.Employee, int64, error) {
	return s.repo.ListEmployees(ctx, f)
}
func (s *employeeService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	return s.repo.GetEmployeeByID(ctx, id)
}
func (s *employeeService) Create(ctx context.Context, req CreateEmployeeRequest) (*domain.Employee, error) {
	if req.OrganizationID == uuid.Nil || req.PositionID == uuid.Nil || (req.PersonID == nil && req.Person == nil) {
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
func (s *employeeService) Update(ctx context.Context, id uuid.UUID, req UpdateEmployeeRequest) (*domain.Employee, error) {
	e, err := s.repo.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.PersonID != nil {
		e.PersonID = *req.PersonID
	}
	if req.OrganizationID != nil {
		e.OrganizationID = *req.OrganizationID
	}
	if req.DepartmentID != nil {
		e.DepartmentID = req.DepartmentID
	}
	if req.PositionID != nil {
		e.PositionID = *req.PositionID
	}
	e.EmployeeNo, e.WorkEmail, e.WorkPhone, e.HiredAt, e.ReleasedAt = req.EmployeeNo, req.WorkEmail, req.WorkPhone, req.HiredAt, req.ReleasedAt
	if req.IsActive != nil {
		e.IsActive = *req.IsActive
	}
	e.UpdatedBy = req.UpdatedBy
	if e.PersonID == uuid.Nil || e.OrganizationID == uuid.Nil || e.PositionID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if err := s.repo.UpdateEmployee(ctx, e); err != nil {
		return nil, err
	}
	return s.repo.GetEmployeeByID(ctx, id)
}
func (s *employeeService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteEmployee(ctx, id)
}
