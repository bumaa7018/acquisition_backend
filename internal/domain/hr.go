package domain

import (
	"time"
)

// HR лавлах нь sdplatform схемийн хүснэгтүүд дээр ШУУД буудаг. Тэдгээрт
// байхгүй талбар (sort_order, source, external_id, deleted_at, employee_no ...)
// энэ загварт ОГТ БАЙХГҮЙ — нэмэлт хүснэгт үүсгэдэггүй тул өөр DB руу шууд
// холбоход хэвийн ажиллана.
//
//	Organization -> sdplatform.sd_organization  (id, name)
//	Department   -> sdplatform.hr_department    (department_id, code, name, parent_id, organization, created_at, updated_at)
//	Position     -> sdplatform.hr_position      (position_id, code, name, created_at, updated_at)
//	Person       -> base.bs_person              (person_id, type, name, first_name, gender, date_of_birth, person_register, phone, email_address, address_town_or_local_name, civil_id, created_at, updated_at)
//	Employee     -> sdplatform.hr_employee      (employee_id, person_id, department_id, position_id, email, mobile_phone, hired_date, terminated_date, is_active, created_at, updated_at)

type HRFilter struct {
	Search         string
	OrganizationID *ID
	DepartmentID   *ID
	IsActive       *bool
	Page           int
	PageSize       int
}

// Organization — sd_organization-д ЗӨВХӨН id, name багана байдаг.
type Organization struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

type Department struct {
	ID               ID        `json:"id"`
	OrganizationID   ID        `json:"organization_id"`
	OrganizationName string    `json:"organization_name,omitempty"`
	ParentID         *ID       `json:"parent_id,omitempty"`
	Code             *string   `json:"code,omitempty"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Position struct {
	ID        ID        `json:"id"`
	Code      *string   `json:"code,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Person — base.bs_person.
//
// person_type нь codelists.cl_person_type код (1=иргэн, 2=хуулийн этгээд).
// Овог (иргэн) болон байгууллагын нэр (хуулийн этгээд) нь нэг л "name" баганад
// хадгалагдана. Хүйс нь int (1=M, 2=F).
type Person struct {
	ID         ID         `json:"id"`
	PersonType string     `json:"person_type"`
	RegisterNo string     `json:"register_no"`
	LastName   *string    `json:"last_name,omitempty"`
	FirstName  *string    `json:"first_name,omitempty"`
	LegalName  *string    `json:"legal_name,omitempty"`
	CivilID    *string    `json:"civil_id,omitempty"`
	BirthDate  *time.Time `json:"birth_date,omitempty"`
	Gender     *string    `json:"gender,omitempty"`
	Phone      *string    `json:"phone,omitempty"`
	Email      *string    `json:"email,omitempty"`
	Address    *string    `json:"address,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (p *Person) DisplayName() string {
	if p == nil {
		return ""
	}
	if p.PersonType == "legal" && p.LegalName != nil {
		return *p.LegalName
	}
	name := ""
	if p.LastName != nil {
		name = *p.LastName
	}
	if p.FirstName != nil {
		if name != "" {
			name += " "
		}
		name += *p.FirstName
	}
	return name
}

// Employee — hr_employee.
//
// hr_employee-д байгууллагын багана БАЙХГҮЙ тул байгууллага нь хэлтсээр
// дамжина: hr_employee.department_id → hr_department.organization. Тиймээс
// OrganizationID/OrganizationName нь зөвхөн УНШИХ талбарууд.
type Employee struct {
	ID       ID `json:"id"`
	PersonID ID `json:"person_id"`
	// UserID — ажилтны нэвтрэх эрх (hr_employee.user_id). Нэвтрэх эрхгүй
	// ажилтанд nil. Чөлөөлөлтөд ажилтан хуваарилах / ажилтнаар шүүх үйлдэл
	// нь land_acquisition_assignee.user_id-тай ажилладаг тул ажилтныг сонгоод
	// ХЭРЭГЛЭГЧИЙН ID-г нь дамжуулах шаардлагатай.
	UserID *ID `json:"user_id,omitempty"`
	// Username — ажилтны нэвтрэх нэр (sd_user.username). Нэвтрэх эрхгүй бол
	// хоосон. Засварлах цонхонд ажилтан ямар эрхээр нэвтэрдгийг харуулахад
	// хэрэгтэй — үгүй бол зассан үед хоосон нэр илгээгдэх эрсдэлтэй.
	//
	// ЗӨВХӨН үнэлгээний байгууллагын ажилтны хариултад бөглөгдөнө: HR-ийн
	// ерөнхий жагсаалт нь өөр САНГИЙН sd_user-ийг join хийж чадахгүй.
	Username         string     `json:"username,omitempty"`
	Person           *Person    `json:"person,omitempty"`
	PersonName       string     `json:"person_name,omitempty"`
	OrganizationID   *ID        `json:"organization_id,omitempty"`
	OrganizationName string     `json:"organization_name,omitempty"`
	DepartmentID     *ID        `json:"department_id,omitempty"`
	DepartmentName   string     `json:"department_name,omitempty"`
	PositionID       ID         `json:"position_id"`
	PositionName     string     `json:"position_name,omitempty"`
	WorkEmail        *string    `json:"work_email,omitempty"`
	WorkPhone        *string    `json:"work_phone,omitempty"`
	HiredAt          *time.Time `json:"hired_at,omitempty"`
	ReleasedAt       *time.Time `json:"released_at,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Menu — sdplatform.sd_menu. Роль тус бүрт харагдах цэсийг sd_menu_role-оор
// холбоно (цэс холбох үйлдэл).
type Menu struct {
	ID          ID           `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	ParentID    *ID          `json:"parent_id,omitempty"`
	ParentCode  string       `json:"parent_code,omitempty"`
	URL         string       `json:"menu_url,omitempty"`
	Icon        string       `json:"menu_icon,omitempty"`
	SortOrder   int          `json:"sort_order"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `json:"permissions,omitempty"`
}
