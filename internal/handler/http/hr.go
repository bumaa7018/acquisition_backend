package http

import (
	"net/http"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type HRHandler struct {
	orgSvc service.OrganizationService
	depSvc service.DepartmentService
	posSvc service.PositionService
	perSvc service.PersonService
	empSvc service.EmployeeService
}

func NewHRHandler(org service.OrganizationService, dep service.DepartmentService, pos service.PositionService, per service.PersonService, emp service.EmployeeService) *HRHandler {
	return &HRHandler{orgSvc: org, depSvc: dep, posSvc: pos, perSvc: per, empSvc: emp}
}

// sd_organization-д зөвхөн name багана байдаг.
type organizationBody struct {
	Name string `json:"name"`
}

type departmentBody struct {
	OrganizationID domain.ID  `json:"organization_id"`
	ParentID       *domain.ID `json:"parent_id"`
	Code           *string    `json:"code"`
	Name           string     `json:"name"`
}

type positionBody struct {
	Code *string `json:"code"`
	Name string  `json:"name"`
}

type personBody struct {
	PersonType string  `json:"person_type"`
	RegisterNo string  `json:"register_no"`
	LastName   *string `json:"last_name"`
	FirstName  *string `json:"first_name"`
	LegalName  *string `json:"legal_name"`
	CivilID    *string `json:"civil_id"`
	BirthDate  *string `json:"birth_date"`
	Gender     *string `json:"gender"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Address    *string `json:"address"`
}

// hr_employee-д байгууллагын багана байхгүй — байгууллага нь хэлтсээр дамжина.
type employeeBody struct {
	PersonID     *domain.ID  `json:"person_id"`
	Person       *personBody `json:"person"`
	DepartmentID *domain.ID  `json:"department_id"`
	PositionID   domain.ID   `json:"position_id"`
	WorkEmail    *string     `json:"work_email"`
	WorkPhone    *string     `json:"work_phone"`
	HiredAt      *string     `json:"hired_at"`
	ReleasedAt   *string     `json:"released_at"`
	IsActive     *bool       `json:"is_active"`
}

func parseOptionalDate(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	if t, err := time.Parse("2006-01-02", *raw); err == nil {
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func personReqFromBody(b personBody) (service.PersonRequest, error) {
	birthDate, err := parseOptionalDate(b.BirthDate)
	if err != nil {
		return service.PersonRequest{}, err
	}
	return service.PersonRequest{PersonType: b.PersonType, RegisterNo: b.RegisterNo, LastName: b.LastName, FirstName: b.FirstName, LegalName: b.LegalName, CivilID: b.CivilID, BirthDate: birthDate, Gender: b.Gender, Phone: b.Phone, Email: b.Email, Address: b.Address}, nil
}

func employeeReqFromBody(b employeeBody) (service.CreateEmployeeRequest, error) {
	hiredAt, err := parseOptionalDate(b.HiredAt)
	if err != nil {
		return service.CreateEmployeeRequest{}, err
	}
	releasedAt, err := parseOptionalDate(b.ReleasedAt)
	if err != nil {
		return service.CreateEmployeeRequest{}, err
	}
	var p *service.PersonRequest
	if b.Person != nil {
		req, err := personReqFromBody(*b.Person)
		if err != nil {
			return service.CreateEmployeeRequest{}, err
		}
		p = &req
	}
	return service.CreateEmployeeRequest{PersonID: b.PersonID, Person: p, DepartmentID: b.DepartmentID, PositionID: b.PositionID, WorkEmail: b.WorkEmail, WorkPhone: b.WorkPhone, HiredAt: hiredAt, ReleasedAt: releasedAt}, nil
}

func hrFilter(c *gin.Context) (domain.HRFilter, bool) {
	f := domain.HRFilter{Search: c.Query("search"), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", 20)}
	if v := c.Query("organization_id"); v != "" {
		id, ok := parseID(v)
		if !ok {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return f, false
		}
		f.OrganizationID = &id
	}
	if v := c.Query("department_id"); v != "" {
		id, ok := parseID(v)
		if !ok {
			response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
			return f, false
		}
		f.DepartmentID = &id
	}
	if v := c.Query("is_active"); v != "" {
		active := v == "true"
		f.IsActive = &active
	}
	return f, true
}

func (h *HRHandler) ListOrganizations(c *gin.Context) {
	f, ok := hrFilter(c)
	if !ok {
		return
	}
	list, total, err := h.orgSvc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, f.Page, f.PageSize)
}
func (h *HRHandler) GetOrganization(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	item, err := h.orgSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) CreateOrganization(c *gin.Context) {
	var b organizationBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.orgSvc.Create(c.Request.Context(), service.OrganizationRequest(b))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}
func (h *HRHandler) UpdateOrganization(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b organizationBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.orgSvc.Update(c.Request.Context(), id, service.OrganizationRequest(b))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) DeleteOrganization(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.orgSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *HRHandler) ListDepartments(c *gin.Context) {
	f, ok := hrFilter(c)
	if !ok {
		return
	}
	list, total, err := h.depSvc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, f.Page, f.PageSize)
}
func (h *HRHandler) CreateDepartment(c *gin.Context) {
	var b departmentBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.depSvc.Create(c.Request.Context(), service.DepartmentRequest(b))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}
func (h *HRHandler) UpdateDepartment(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b departmentBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.depSvc.Update(c.Request.Context(), id, service.DepartmentRequest(b))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) DeleteDepartment(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.depSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *HRHandler) ListPositions(c *gin.Context) {
	f, ok := hrFilter(c)
	if !ok {
		return
	}
	list, total, err := h.posSvc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, f.Page, f.PageSize)
}
func (h *HRHandler) CreatePosition(c *gin.Context) {
	var b positionBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.posSvc.Create(c.Request.Context(), service.PositionRequest(b))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}
func (h *HRHandler) UpdatePosition(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b positionBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.posSvc.Update(c.Request.Context(), id, service.PositionRequest(b))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) DeletePosition(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.posSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *HRHandler) ListPersons(c *gin.Context) {
	f, ok := hrFilter(c)
	if !ok {
		return
	}
	list, total, err := h.perSvc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, f.Page, f.PageSize)
}
func (h *HRHandler) CreatePerson(c *gin.Context) {
	var b personBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	req, err := personReqFromBody(b)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.perSvc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}
func (h *HRHandler) UpdatePerson(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b personBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	req, err := personReqFromBody(b)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.perSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) DeletePerson(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.perSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *HRHandler) ListEmployees(c *gin.Context) {
	f, ok := hrFilter(c)
	if !ok {
		return
	}
	// Системийн админаас бусад нь ЗӨВХӨН өөрийн хэлтсийн ажилтныг (хэлтсийн
	// дарга хэрэглэгч үүсгэхдээ ажилтан сонгоно).
	if !middleware.IsSystemAdmin(c) {
		f.DepartmentID = middleware.CallerDepartmentID(c)
		if f.DepartmentID == nil {
			response.Paginated(c, []*domain.Employee{}, 0, f.Page, f.PageSize)
			return
		}
	}
	list, total, err := h.empSvc.List(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, list, total, f.Page, f.PageSize)
}
func (h *HRHandler) GetEmployee(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	item, err := h.empSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) CreateEmployee(c *gin.Context) {
	var b employeeBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	req, err := employeeReqFromBody(b)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	item, err := h.empSvc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, item)
}
func (h *HRHandler) UpdateEmployee(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var b employeeBody
	if err := c.ShouldBindJSON(&b); err != nil {
		response.AbortValidation(c, err)
		return
	}
	hiredAt, err := parseOptionalDate(b.HiredAt)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	releasedAt, err := parseOptionalDate(b.ReleasedAt)
	if err != nil {
		response.AbortValidation(c, err)
		return
	}
	req := service.UpdateEmployeeRequest{PersonID: b.PersonID, DepartmentID: b.DepartmentID, PositionID: &b.PositionID, WorkEmail: b.WorkEmail, WorkPhone: b.WorkPhone, HiredAt: hiredAt, ReleasedAt: releasedAt, IsActive: b.IsActive}
	item, err := h.empSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}
func (h *HRHandler) DeleteEmployee(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.empSvc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
