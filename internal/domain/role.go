package domain

import (
	"strings"
	"time"
)

// Role — sdplatform.sd_role.
//
// Энэ схемд эрхийн загвар нь гурван давхаргатай:
//   - sd_menu / sd_menu_role — цэс харагдах эсэх
//   - sd_role.role_type_code — НӨӨЦ (resource): land, compensation, decision, config ...
//   - sd_permission          — ҮЙЛДЭЛ (action): read, create, update, delete
//
// Тиймээс нэг роль нэг нөөцтэй ба нэр нь "<resource>_<түвшин>" хэлбэртэй
// (land_admin, land_senior_specialist ...). Үр дүнгийн эрх нь
// role.Resource + ":" + permission.Action болж бүрдэнэ.
type Role struct {
	ID          ID           `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Resource    string       `json:"resource"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `json:"permissions"`
	Menus       []Menu       `json:"menus,omitempty"`
}

// Level — ролийн түвшин (admin, senior_specialist, employee ...).
//
// JWT-ийн `roles` нэхэмжлэлд нэрийн оронд түвшин ордог тул "admin" эсэхийг
// шалгадаг маршрутууд (adminOnly, RequireAnyRole) өмнөх шигээ ажиллана:
// land_admin + compensation_admin + ... → roles: ["admin"].
func (r Role) Level() string {
	if r.Resource != "" {
		if s := strings.TrimPrefix(r.Name, r.Resource+"_"); s != r.Name && s != "" {
			return s
		}
	}
	return r.Name
}

// MaxSystemAdmins — системийн админ (хэлтэсгүй, бүх хэлтсийг хардаг цорын
// ганц роль) нийт 3-аас хэтрэхгүй.
const MaxSystemAdmins = 3

// IsSystemAdminLevel — ролийн түвшин (JWT-ийн roles) системийн админ эсэх.
func IsSystemAdminLevel(level string) bool {
	return level == "admin" || level == "Админ"
}

// IsAdminRoleName — нэр нь системийн админы нэртэй давхцаж буй эсэх.
//
// Админ эсэхийг НЭРЭЭР (Level) тодорхойлдог тул "admin" / "Админ" нэртэй
// өөр роль үүсгэвэл түүнийг авсан хэрэглэгч токендоо "admin" авч бүх
// хаалтыг давна. Иймд том/жижиг үсэг, илүү зай, "<resource>_" угтвараас үл
// хамааран ("ADMIN", " Админ ", "land_admin") бүгдийг админы нэр гэж үзнэ.
func IsAdminRoleName(name string) bool {
	n := strings.ToLower(strings.Join(strings.Fields(name), " "))
	if n == "admin" || n == "админ" {
		return true
	}
	if i := strings.LastIndex(n, "_"); i >= 0 {
		s := n[i+1:]
		return s == "admin" || s == "админ"
	}
	return false
}

// IsSystemAdmin — энэ роль системийн админы роль эсэх.
func (r Role) IsSystemAdmin() bool {
	return IsSystemAdminLevel(r.Level())
}

// RoleLevelProfessionalOrg — мэргэжлийн байгууллагын түвшин
// (compensation_professional_org).
const RoleLevelProfessionalOrg = "professional_org"

// IsAPIGrantable — энэ ролийг API-гаар хэрэглэгчид ОЛГОЖ БОЛОХ эсэх.
//
// МЭРГЭЖЛИЙН БАЙГУУЛЛАГЫН роль нь ОЛГОГДОХГҮЙ. Тэр роль нь мэргэжлийн
// байгууллагын бүртгэлд л хамаарах ба тэр бүртгэл нь хэрэглэгчийн бүтцээс
// ТУСДАА (appdb-ийн valuation_org_profile / valuation_org_employee, токены
// `valuation_org` тэмдэг). Иймд authdb-ийн ямар ч хэрэглэгч — админ ч гэсэн —
// үүнийг дамжуулан нөхөх олговрын эрх авах ёсгүй.
//
// Өмнө нь callerCanGrantRole нь "дуудагч тухайн эрхүүдийг эзэмшсэн эсэх"-ийг л
// шалгадаг байсан тул compensation:* эзэмшдэг админ энэ ролийг хэнд ч олгож
// чаддаг байв.
func (r Role) IsAPIGrantable() bool {
	return r.Level() != RoleLevelProfessionalOrg
}

// IsSeparatedDutyLevel — үнэлгээг ХЯНАХ/БАТЛАХ түвшин (санхүү, МИКА) эсэх.
//
// Эдгээр роль нь үнэлгээг оруулагчаас ТУСДАА байх ёстой үүрэг: санхүү үнэлгээг
// батална/буцаана, МИКА өөрийн урсгалд бичнэ. Иймд тэдгээрийг ЗӨВХӨН системийн
// админ олгоно/хураана (Role.RequiresAdminGrantor).
func IsSeparatedDutyLevel(level string) bool {
	switch level {
	case "finance_specialist", "Санхүүгийн мэргэжилтэн", "Санхүү", "mika", "МИКА":
		return true
	}
	return false
}

// RequiresAdminGrantor — энэ ролийг зөвхөн системийн админ олгож/хурааж болох эсэх.
//
// "Дуудагч ролийн эрхүүдийг эзэмшсэн үү" гэсэн ерөнхий дүрэм энд хүрэлцэхгүй:
// хэлтсийн дарга, ахлах мэргэжилтэн compensation:* эрхтэй тул санхүү/МИКА
// ролийг өөрийн хэлтсийн (эсвэл шинээр нээсэн) бүртгэлд олгож, үнэлгээг
// өөрсдөө батлуулах боломжтой байв. Эдгээр ролийн хүч нь эрхийн багцад биш,
// ТҮВШИНД (токены roles) оршдог.
func (r Role) RequiresAdminGrantor() bool {
	return IsSeparatedDutyLevel(r.Level())
}

// Permission — sdplatform.sd_permission. Энэ схемд эрх нь ЗӨВХӨН үйлдэл;
// нөөц нь холбогдсон ролиос ирнэ.
type Permission struct {
	ID   ID     `json:"id"`
	Code string `json:"code"`
	// Name — ролийн дотор жагсаахад "<resource>:<action>" (өмнөх API-тай ижил),
	// эрхийн лавлах (GET /permissions) дээр зөвхөн үйлдэл.
	Name        string    `json:"name"`
	Resource    string    `json:"resource,omitempty"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// PermissionKey — "resource:action" эрхийн мөр.
func PermissionKey(resource, action string) string {
	if resource == "" {
		return action
	}
	return resource + ":" + action
}

// SplitPermissionKey — "land:read" → ("land", "read").
func SplitPermissionKey(key string) (resource, action string) {
	if i := strings.IndexByte(key, ':'); i >= 0 {
		return key[:i], key[i+1:]
	}
	return "", key
}

// NormalizeRoleName — ролийн нэрийг харьцуулахад: том/жижиг үсэг, илүү зай үл хамаарна.
func NormalizeRoleName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// reservedRoleNames — backend/frontend хаалтууд ролийг НЭРЭЭР нь таньдаг
// (токены `roles` claim). Эдгээр нэр нь системийн ролиудын түвшин ба тэдний
// монгол нэр. Өөр роль ингэж нэрлэгдвэл түүнийг авсан хэрэглэгч тэр ролийн
// хаалтуудыг (хуваарилалтын чөлөөлөлт, хуучин мэдээллийн импорт, батлах г.м.)
// эрхгүйгээр давна.
var reservedRoleNames = map[string]bool{
	"admin": true, "админ": true,
	"senior_specialist": true, "ахлах мэргэжилтэн": true,
	"department_head": true, "хэлтсийн дарга": true,
	"employee":              true,
	"decision_registration": true, "захирамж бүртгэл": true,
	"finance_specialist": true, "санхүүгийн мэргэжилтэн": true, "санхүү": true,
	"mika": true, "мика": true,
	"professional_org": true, "мэргэжлийн байгууллага": true, "мэргэжлийн байгуулл...": true,
}

// IsReservedRoleName — нэр нь системийн ролийн түвшин/нэртэй давхцах эсэх.
func IsReservedRoleName(name string) bool {
	return reservedRoleNames[NormalizeRoleName(name)]
}
