package domain

import (
	"time"

	"github.com/google/uuid"
)

type HRFilter struct {
	Search         string
	OrganizationID *uuid.UUID
	DepartmentID   *uuid.UUID
	IsActive       *bool
	Page           int
	PageSize       int
}

type Organization struct {
	ID         uuid.UUID  `json:"id"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	Name       string     `json:"name"`
	ShortName  *string    `json:"short_name,omitempty"`
	RegisterNo *string    `json:"register_no,omitempty"`
	Address    *string    `json:"address,omitempty"`
	Phone      *string    `json:"phone,omitempty"`
	Email      *string    `json:"email,omitempty"`
	ExternalID *string    `json:"external_id,omitempty"`
	Source     string     `json:"source"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type Department struct {
	ID               uuid.UUID  `json:"id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
	OrganizationName string     `json:"organization_name,omitempty"`
	ParentID         *uuid.UUID `json:"parent_id,omitempty"`
	Code             *string    `json:"code,omitempty"`
	Name             string     `json:"name"`
	SortOrder        int        `json:"sort_order"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

type Position struct {
	ID        uuid.UUID  `json:"id"`
	Code      *string    `json:"code,omitempty"`
	Name      string     `json:"name"`
	SortOrder int        `json:"sort_order"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type Person struct {
	ID         uuid.UUID  `json:"id"`
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
	ExternalID *string    `json:"external_id,omitempty"`
	Source     string     `json:"source"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	CreatedBy  *string    `json:"created_by,omitempty"`
	UpdatedBy  *string    `json:"updated_by,omitempty"`
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

type Employee struct {
	ID               uuid.UUID  `json:"id"`
	PersonID         uuid.UUID  `json:"person_id"`
	Person           *Person    `json:"person,omitempty"`
	PersonName       string     `json:"person_name,omitempty"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
	OrganizationName string     `json:"organization_name,omitempty"`
	DepartmentID     *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName   string     `json:"department_name,omitempty"`
	PositionID       uuid.UUID  `json:"position_id"`
	PositionName     string     `json:"position_name,omitempty"`
	EmployeeNo       *string    `json:"employee_no,omitempty"`
	WorkEmail        *string    `json:"work_email,omitempty"`
	WorkPhone        *string    `json:"work_phone,omitempty"`
	HiredAt          *time.Time `json:"hired_at,omitempty"`
	ReleasedAt       *time.Time `json:"released_at,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	CreatedBy        *string    `json:"created_by,omitempty"`
	UpdatedBy        *string    `json:"updated_by,omitempty"`
}
