package i18n

import "strings"

type Lang string

const (
	MN Lang = "mn"
	EN Lang = "en"
)

// Detect reads Accept-Language header; default is MN.
// Browser defaults such as en-GB/en-US should not switch the Mongolian UI to English.
func Detect(header string) Lang {
	if strings.EqualFold(strings.TrimSpace(header), string(EN)) {
		return EN
	}
	return MN
}

type Key string

const (
	// Success
	MsgOK            Key = "ok"
	MsgCreated       Key = "created"
	MsgUpdated       Key = "updated"
	MsgDeleted       Key = "deleted"
	MsgSynced        Key = "synced"
	MsgAssigned      Key = "perm_assigned"
	MsgLogout        Key = "logout"
	MsgPassReset     Key = "pass_reset"
	MsgPassResetSent Key = "pass_reset_sent"

	// Generic errors
	ErrNotFound          Key = "not_found"
	ErrUnauthorized      Key = "unauthorized"
	ErrForbidden         Key = "forbidden"
	ErrConflict          Key = "conflict"
	ErrInvalidInput      Key = "invalid_input"
	ErrInternalServer    Key = "internal_server"
	ErrInvalidUUID       Key = "invalid_uuid"
	ErrInvalidAcqUUID    Key = "invalid_acq_uuid"
	ErrInvalidParcelUUID Key = "invalid_parcel_uuid"
	ErrInvalidCreds      Key = "invalid_creds"
	ErrInvalidToken      Key = "invalid_token"
	ErrExpiredToken      Key = "expired_token"
	ErrDBUnavailable     Key = "db_unavailable"

	// Land / plan errors
	ErrPlanNotFound             Key = "plan_not_found"
	ErrAcqNotFound              Key = "acq_not_found"
	ErrRequiredFields           Key = "required_fields"
	ErrStartDateFormat          Key = "start_date_format"
	ErrEndDateFormat            Key = "end_date_format"
	ErrShapefileRequired        Key = "shapefile_required"
	ErrShapefileOpen            Key = "shapefile_open"
	ErrShapefileRead            Key = "shapefile_read"
	ErrBoundaryExceeds          Key = "boundary_exceeds"
	ErrNewBoundaryExceeds       Key = "new_boundary_exceeds"
	ErrShapefileRequiredForPlan Key = "shapefile_required_for_plan"
	ErrDeleteNotNew             Key = "delete_not_new"
	WarnGeomChanged             Key = "geom_changed"
	ErrCodeRequired             Key = "code_required"

	// Status errors
	ErrStatusRequired Key = "status_required"
	ErrInvalidStatus  Key = "invalid_status"
	ErrNotAssigned    Key = "not_assigned"
	ErrApproved       Key = "approved_locked"

	// Document / file errors
	ErrFileRequired Key = "file_required"
	ErrFileTooLarge Key = "file_too_large"
	ErrFileOpen     Key = "file_open"
	ErrFileRead     Key = "file_read"
	ErrDocNotFound  Key = "doc_not_found"
)

var messages = map[Key][2]string{
	// [0]=MN  [1]=EN
	MsgOK:            {"Амжилттай", "OK"},
	MsgCreated:       {"Амжилттай үүсгэлээ", "Created successfully"},
	MsgUpdated:       {"Амжилттай шинэчлэлээ", "Updated successfully"},
	MsgDeleted:       {"Амжилттай устгалаа", "Deleted successfully"},
	MsgSynced:        {"Нэгж талбарын мэдээлэл амжилттай татагдлаа", "Parcel data synced successfully"},
	MsgAssigned:      {"Эрх амжилттай олгогдлоо", "Permission assigned successfully"},
	MsgLogout:        {"Амжилттай гарлаа", "Logged out successfully"},
	MsgPassReset:     {"Нууц үг амжилттай шинэчлэгдлээ", "Password reset successfully"},
	MsgPassResetSent: {"Имэйл хаяг бүртгэлтэй бол reset холбоос илгээгдлээ", "If the email exists, a reset link has been sent"},

	ErrNotFound:          {"Олдсонгүй", "Not found"},
	ErrUnauthorized:      {"Нэвтрэх шаардлагатай", "Unauthorized"},
	ErrForbidden:         {"Хандах эрх байхгүй", "Forbidden"},
	ErrConflict:          {"Давхардсан бичлэг", "Already exists"},
	ErrInvalidInput:      {"Буруу оролт", "Invalid input"},
	ErrInternalServer:    {"Дотоод алдаа гарлаа", "Internal server error"},
	ErrInvalidUUID:       {"Буруу UUID формат", "Invalid UUID format"},
	ErrInvalidAcqUUID:    {"Буруу acquisition UUID формат", "Invalid acquisition UUID format"},
	ErrInvalidParcelUUID: {"Буруу parcel UUID формат", "Invalid parcel UUID format"},
	ErrInvalidCreds:      {"Нэвтрэх нэр эсвэл нууц үг буруу", "Invalid credentials"},
	ErrInvalidToken:      {"Токен хүчингүй", "Invalid token"},
	ErrExpiredToken:      {"Токений хугацаа дууссан", "Token expired"},
	ErrDBUnavailable:     {"Мэдээллийн сан давчдаж байна", "Database unavailable"},

	ErrPlanNotFound:             {"Төлөвлөгөө олдсонгүй", "Plan not found"},
	ErrAcqNotFound:              {"Чөлөөлөлт олдсонгүй", "Land acquisition not found"},
	ErrRequiredFields:           {"plan_code, start_date, end_date заавал шаардлагатай", "plan_code, start_date, end_date are required"},
	ErrStartDateFormat:          {"start_date формат буруу (YYYY-MM-DD)", "Invalid start_date format (YYYY-MM-DD)"},
	ErrEndDateFormat:            {"end_date формат буруу (YYYY-MM-DD)", "Invalid end_date format (YYYY-MM-DD)"},
	ErrShapefileRequired:        {"shapefile файл шаардлагатай", "shapefile file is required"},
	ErrShapefileOpen:            {"shapefile нээхэд алдаа", "Failed to open shapefile"},
	ErrShapefileRead:            {"shapefile уншихад алдаа", "Failed to read shapefile"},
	ErrBoundaryExceeds:          {"Чөлөөлөх хил нь төлөвлөгөөний хилээс гарсан эсвэл shapefile формат буруу", "Acquisition boundary exceeds plan boundary or invalid shapefile"},
	ErrNewBoundaryExceeds:       {"Шинэ хил нь төлөвлөгөөний хилээс гарсан эсвэл shapefile формат буруу", "New boundary exceeds plan boundary or invalid shapefile"},
	ErrShapefileRequiredForPlan: {"Төлөвлөгөө солих бол шинэ shapefile заавал шаардлагатай", "Shapefile is required when changing plan"},
	ErrDeleteNotNew:             {"Зөвхөн NEW статустай чөлөөлөлтийг устгах боломжтой", "Only acquisitions with NEW status can be deleted"},
	WarnGeomChanged:             {"Чөлөөлөлтийн хил шинэчлэгдлээ. Хуучин холбогдсон нэгж талбарууд хэвээр үлдэж, шинэ хилтэй давхцах дутуу нэгж талбарууд нэмэгдлээ", "The acquisition boundary was updated. Existing linked parcels were kept, and missing parcels intersecting the new boundary were added"},
	ErrCodeRequired:             {"code параметр шаардлагатай", "code parameter is required"},

	ErrStatusRequired: {"to_status шаардлагатай", "to_status is required"},
	ErrInvalidStatus:  {"Буруу статус", "Invalid status"},
	ErrNotAssigned:    {"Энэ чөлөөлөлтэд ажиллах эрх байхгүй", "You are not assigned to this acquisition"},
	ErrApproved:       {"Санхүүгийн мэргэжилтэн зөвшөөрсөн мэдээллийг өөрчлөх боломжгүй", "Approved record cannot be modified"},

	ErrFileRequired: {"file талбар шаардлагатай", "file field is required"},
	ErrFileTooLarge: {"Файл 10MB-аас хэтэрсэн байна", "file exceeds 10MB limit"},
	ErrFileOpen:     {"Файл нээхэд алдаа гарлаа", "failed to open file"},
	ErrFileRead:     {"Файл уншихад алдаа гарлаа", "failed to read file"},
	ErrDocNotFound:  {"Баримт олдсонгүй", "document not found"},
}

// T returns the translated message for the given key and language.
func T(l Lang, k Key) string {
	if pair, ok := messages[k]; ok {
		if l == EN {
			return pair[1]
		}
		return pair[0]
	}
	return string(k)
}
