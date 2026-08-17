package domain

import "time"

// Үнэлгээний (мэргэжлийн) байгууллагын бүртгэл.
//
// Байгууллага өөрөө sdplatform.sd_organization дээр амьдарна — тэнд ЗӨВХӨН
// (id, name) багана байдаг тул регистр, тусгай зөвшөөрөл зэрэг манай системд л
// хэрэгтэй талбарууд аппын өөрийн valuation_org_profile хүснэгтэд хадгалагдана.
//
// Ажилтнууд нь sdplatform-ын ТӨРӨЛХ бүтцээр бүртгэгдэнэ:
//
//	sd_organization → hr_department → hr_employee → bs_person
//	hr_employee.user_id → sd_user (нэвтрэх эрх; заавал бус)
//
// Ингэснээр нэг байгууллагад ОЛОН ажилтан байж, тус бүр өөрийн эрхээр нэвтэрч
// ажиллах боломжтой (өмнө нь байгууллага = нэг хэрэглэгч байсан).

// ValuationOrgDepartmentName — байгууллага үүсэхэд автоматаар үүсэх үндсэн
// нэгжийн нэр. hr_employee нь байгууллагатай ЗӨВХӨН хэлтсээр дамжин холбогддог
// тул (hr_employee-д organization багана байхгүй) байгууллага бүрд дор хаяж
// нэг нэгж хэрэгтэй.
const ValuationOrgDepartmentName = "Үндсэн нэгж"

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

// ValuationOrgEmployee — байгууллагатай ХАМТ бүртгэгдэх ажилтны оролт.
//
// ID утгатай бол одоо байгаа ажилтныг засна, эс бөгөөс шинээр үүсгэнэ.
// Хадгалах үед жагсаалтад ОРООГҮЙ ажилтнууд идэвхгүй болно (is_active = 0).
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
