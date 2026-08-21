package domain

import "time"

// Үнэлгээний (мэргэжлийн) байгууллагын бүртгэл.
//
// Байгууллагын identity, profile БОЛОН АЖИЛТАН нь appdb-д амьдарна:
//
//	valuation_org_profile  → байгууллага (id, name, регистр, лиценз ...)
//	valuation_org_employee → ажилтан (org_id → profile.id)
//	valuation_org_employee.user_id → sdplatform.sd_user (нэвтрэх эрх, заавал бус)
//
// Ажилтныг appdb-д хадгалах нь зориудын шийдэл: sdplatform-ийн HR бүтэц
// (hr_employee → hr_department → sd_organization) нь байгууллагыг
// sd_organization.id-гаар таньдаг бол чөлөөлөлт/нэгж талбар нь
// valuation_org_profile.id-гаар таньдаг. Хоёр дараалал 000002 миграцаас хойш
// салсан тул HR-аар дамжуулбал JWT-ийн org_id нь professional_org_id-тай
// хэзээ ч таарахгүй байв. Ажилтан appdb-д байхад org_id = profile.id болж
// шууд тулгагдана.
//
// sdplatform-д ЗӨВХӨН нэвтрэлтийн мөр (sd_user) нэмэгдэнэ — схемийн бүтэц
// хөндөгдөхгүй, роль ОНООГДОХГҮЙ (хандалт нь токены `valuation_org` тэмдэг ба
// аппын allowlist-ээр шийдэгддэг).

// ValuationOrgDefaultPosition — албан тушаал заагаагүй ажилтны анхдагч тушаал.
const ValuationOrgDefaultPosition = "Үнэлгээчин"

// Үнэлгээний байгууллагын цэс — АППЫН ӨӨРИЙН жагсаалт.
//
// ЯАГААД sd_menu / sd_menu_role-оос уншихгүй вэ: тэр хоёр хүснэгт нь цэсийг
// РОЛЬ-оор олгодог (sd_menu_role) бөгөөд system 10-ийн бүх роль дээр бүх цэс
// холбогддог. Үнэлгээний байгууллага роль хэрэглэхээ больсон тул цэс нь
// эрхийн загвартай нэг эх сурвалжтай байх ёстой: доорх хоёр зам нь
// middleware.ValuationOrgScope-ийн allowlist-тай ЯГ таарна.
//
// Код нь аппын өөрийн (`VO`-гаар) — sdplatform-ийн "A0101xx" дугаарлалтад
// хөндлөнгөөс оролцохгүй. ID нь sd_menu.menu_id БИШ (эдгээр мөр тэр хүснэгтэд
// байхгүй) тул зөвхөн frontend-ийн key болж ажиллана.
const (
	ValuationOrgMenuDashboardCode = "VO01"
	ValuationOrgMenuLandCode      = "VO02"
)

// ValuationOrgMenus — үнэлгээний байгууллагын ажилтанд харагдах ЦЭС.
// Дашбоард ба Чөлөөлөлт — өөр цэс байхгүй.
func ValuationOrgMenus() []*Menu {
	return []*Menu{
		{
			ID:        1,
			Code:      ValuationOrgMenuDashboardCode,
			Name:      "Дашбоард",
			URL:       "/dashboard",
			SortOrder: 1,
		},
		{
			ID:        2,
			Code:      ValuationOrgMenuLandCode,
			Name:      "Газар чөлөөлөлт",
			URL:       "/land-acquisitions",
			SortOrder: 2,
		},
	}
}

type ValuationOrg struct {
	ID               ID         `json:"id"`
	Name             string     `json:"name"`
	ShortName        string     `json:"short_name"`
	RegisterNo       string     `json:"register_no"`
	LicenseNo        string     `json:"license_no"`
	LicenseIssuedAt  *time.Time `json:"license_issued_at,omitempty"`
	LicenseExpiresAt *time.Time `json:"license_expires_at,omitempty"`
	Phone            string     `json:"phone"`
	Email            string     `json:"email"`
	Address          string     `json:"address"`
	Note             string     `json:"note"`
	IsActive         bool       `json:"is_active"`
	// EmployeeCount — ИДЭВХТЭЙ ажилтны тоо (жагсаалтын багана).
	EmployeeCount int `json:"employee_count"`
	// Employees — ЗӨВХӨН GetByID-д бөглөгдөнө (жагсаалтад ирэхгүй).
	Employees []*Employee `json:"employees,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type ValuationOrgFilter struct {
	Search   string
	IsActive *bool
	Page     int
	PageSize int
}

// ValuationOrgEmployee — байгууллагатай ХАМТ бүртгэгдэх ажилтны оролт.
//
// ID утгатай бол одоо байгаа ажилтныг засна, эс бөгөөс шинээр үүсгэнэ
// (регистрээр нь ижил ажилтан байвал түүнийг сэргээнэ). Хадгалах үед
// жагсаалтад ОРООГҮЙ ажилтнууд идэвхгүй болно — мөр устгахгүй тул түүх,
// баримт дээрх нэр хадгалагдана.
type ValuationOrgEmployee struct {
	ID           *ID    `json:"id,omitempty"`
	LastName     string `json:"last_name"`
	FirstName    string `json:"first_name"`
	RegisterNo   string `json:"register_no"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	PositionName string `json:"position_name"`

	// Нэвтрэх эрх — ЗААВАЛ БИШ. Username өгвөл sd_user үүсгэж (эсвэл одоо
	// байгааг холбож) ажилтныг түүнтэй холбоно; хоосон бол ажилтан зөвхөн
	// бүртгэлд байх ба системд нэвтрэхгүй. Password нь ЗӨВХӨН шинэ хэрэглэгч
	// үүсгэх эсвэл нууц үг солиход хэрэглэгдэнэ.
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
