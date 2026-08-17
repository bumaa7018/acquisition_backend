package domain

import "time"

// Үнэлгээний (мэргэжлийн) байгууллагын бүртгэл.
//
// Мэргэжлийн байгууллагын identity болон бүх profile мэдээлэл appdb-ийн
// valuation_org_profile хүснэгтэд хадгалагдана. Энэ model нь ерөнхий HR
// лавлахын байгууллагаас тусдаа бөгөөд profile CRUD нь HR өгөгдөлд хүрэхгүй.
//
// Ажилтны бүртгэл нь тусдаа HR модулийн үүрэг бөгөөд энд автоматаар үүсэхгүй:
//
//	sd_organization → hr_department → hr_employee → bs_person
//	hr_employee.user_id → sd_user (нэвтрэх эрх; заавал бус)
//
// Profile endpoint нь HR хүснэгтүүдэд бичихгүй.

// Deprecated: profile-only бүртгэл үндсэн HR нэгж автоматаар үүсгэхгүй.
// ValuationOrgDepartmentName — байгууллага үүсэхэд автоматаар үүсэх үндсэн
// нэгжийн нэр. hr_employee нь байгууллагатай ЗӨВХӨН хэлтсээр дамжин холбогддог
// тул (hr_employee-д organization багана байхгүй) байгууллага бүрд дор хаяж
// нэг нэгж хэрэгтэй.
const ValuationOrgDepartmentName = "Үндсэн нэгж"

// Deprecated: profile-only бүртгэл анхдагч албан тушаал үүсгэхгүй.
// ValuationOrgPositionName — ажилтанд албан тушаал заагаагүй үед хэрэглэх
// анхдагч албан тушаал (hr_employee.position_id заавал утгатай байх ёстой).
const ValuationOrgPositionName = "Үнэлгээчин"

type ValuationOrg struct {
	ID               ID          `json:"id"`
	Name             string      `json:"name"`
	ShortName        string      `json:"short_name"`
	RegisterNo       string      `json:"register_no"`
	LicenseNo        string      `json:"license_no"`
	LicenseIssuedAt  *time.Time  `json:"license_issued_at,omitempty"`
	LicenseExpiresAt *time.Time  `json:"license_expires_at,omitempty"`
	Phone            string      `json:"phone"`
	Email            string      `json:"email"`
	Address          string      `json:"address"`
	Note             string      `json:"note"`
	IsActive         bool        `json:"is_active"`
	EmployeeCount    int         `json:"employee_count"`
	Employees        []*Employee `json:"employees,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type ValuationOrgFilter struct {
	Search   string
	IsActive *bool
	Page     int
	PageSize int
}

// ValuationOrgEmployee — хуучин response/request compatibility type.
//
// ID утгатай бол одоо байгаа ажилтныг засна, эс бөгөөс шинээр үүсгэнэ.
// Мэргэжлийн байгууллагын profile endpoint ажилтан бүртгэхгүй.
type ValuationOrgEmployee struct {
	ID           *ID    `json:"id,omitempty"`
	LastName     string `json:"last_name"`
	FirstName    string `json:"first_name"`
	RegisterNo   string `json:"register_no"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	PositionName string `json:"position_name"`

	// Нэвтрэх эрх — заавал биш. Username өгвөл sd_user үүсгэж (эсвэл одоо
	// байгааг холбож) ажилтныг түүнтэй холбоно. Password нь ЗӨВХӨН шинэ
	// хэрэглэгч үүсгэх эсвэл нууц үг солиход хэрэглэгдэнэ.
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
