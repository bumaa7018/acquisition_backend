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
