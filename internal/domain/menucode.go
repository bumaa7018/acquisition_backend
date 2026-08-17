package domain

// Цэсний код ↔ эрхийн нөөц.
//
// sdplatform-д цэс нь "A010001" маягийн дугаартай байдаг ба тэр баганыг ГАДААД
// систем эзэмшдэг. Аппын эрхийн түлхүүр (`land:update`) нь харин кодоос ХАМААРАХ
// ЁСГҮЙ — эс тэгвээс sdplatform цэсээ дахин дугаарлахад RequirePermission болон
// фронтендын бүх шалгалт чимээгүйхэн унана.
//
// Тиймээс код ↔ нөөцийн хөрвүүлэлт ЗӨВХӨН энд байна. Танихгүй код ирвэл кодыг нь
// шууд нөөц болгон буцаана (өөр системийн цэс саадгүй ажиллана).
var menuCodeToResource = map[string]string{
	"A010001": "land",
	"A010002": "compensation",
	"A010003": "decision",
	"A010004": "users",
	"A010005": "roles",
	"A010006": "permissions",
	"A010007": "hr",
	"A010008": "admin",
	"A010009": "audit",
}

var resourceToMenuCode = func() map[string]string {
	out := make(map[string]string, len(menuCodeToResource))
	for code, resource := range menuCodeToResource {
		out[resource] = code
	}
	return out
}()

// MenuResource — цэсний кодоос эрхийн нөөц. Танихгүй бол кодыг нь буцаана.
func MenuResource(code string) string {
	if resource, ok := menuCodeToResource[code]; ok {
		return resource
	}
	return code
}

// MenuCodeForResource — эрхийн нөөцөөс цэсний код. Танихгүй бол нөөцийг буцаана.
func MenuCodeForResource(resource string) string {
	if code, ok := resourceToMenuCode[resource]; ok {
		return code
	}
	return resource
}

// MenuPermissionKey — цэсний код + үйлдэл → "land:update".
func MenuPermissionKey(menuCode, action string) string {
	return PermissionKey(MenuResource(menuCode), action)
}
