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
	ErrNotFound Key = "not_found"
	// ErrRouteNotFound — API-д байхгүй хаяг/метод руу хүсэлт ирсэн.
	ErrRouteNotFound     Key = "route_not_found"
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
	ErrTooManyRequests   Key = "too_many_requests"

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
	ErrAU2Crossing              Key = "au2_crossing"
	ErrOutsidePlanExceeds       Key = "outside_plan_exceeds"
	ErrOutsideParcelBoundary    Key = "outside_parcel_boundary"
	ErrShapefileProjection      Key = "shapefile_projection"
	ErrShapefileOutsideCountry  Key = "shapefile_outside_country"
	ErrShapefileMultiRecord     Key = "shapefile_multi_record"
	ErrShapefileShapeType       Key = "shapefile_shape_type"
	ErrShapefileMultiPolygon    Key = "shapefile_multipolygon"
	ErrShapefileFormat          Key = "shapefile_format"
	ErrHasReleasedParcels       Key = "has_released_parcels"
	ErrConfirmedDecisionLinked  Key = "confirmed_decision_linked"
	ErrShapefileRequiredForPlan Key = "shapefile_required_for_plan"
	ErrDeleteNotNew             Key = "delete_not_new"
	WarnGeomChanged             Key = "geom_changed"
	ErrCodeRequired             Key = "code_required"

	// Status errors
	ErrStatusRequired          Key = "status_required"
	ErrInvalidStatus           Key = "invalid_status"
	ErrNotAssigned             Key = "not_assigned"
	ErrApproved                Key = "approved_locked"
	ErrValuationReportRequired Key = "valuation_report_required"
	ErrAcqLocked               Key = "acq_locked"
	ErrNonFinalParcels         Key = "non_final_parcels"
	ErrAssetPhotoRequired      Key = "asset_photo_required"
	ErrReportBeforeApproval    Key = "report_before_approval"
	ErrParcelAppNoMissing      Key = "parcel_app_no_missing"
	ErrParcelOwnedNoPayment    Key = "parcel_owned_no_payment"
	ErrSourceFileUnavailable   Key = "source_file_unavailable"

	// Document / file errors
	ErrFileRequired           Key = "file_required"
	ErrFileTooLarge           Key = "file_too_large"
	ErrDroneFileType          Key = "drone_file_type"
	ErrDroneUploadIncomplete  Key = "drone_upload_incomplete"
	ErrDroneAlreadyRegistered Key = "drone_already_registered"
	ErrDroneTooLarge          Key = "drone_too_large"
	ErrFileOpen               Key = "file_open"
	ErrFileRead               Key = "file_read"
	ErrFileType               Key = "file_type"
	ErrDocNotFound            Key = "doc_not_found"

	// Захирамжийн төслийн алдаа
	ErrDecisionConfirmed     Key = "decision_confirmed"
	ErrParcelNotReleased     Key = "parcel_not_released"
	ErrParcelAlreadyLinked   Key = "parcel_already_linked"
	ErrDecisionAcqMismatch   Key = "decision_acq_mismatch"
	ErrDecreeRequired        Key = "decree_required"
	ErrFundingSourceRequired Key = "funding_source_required"
	ErrFundingSourceInUse    Key = "funding_source_in_use"
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

	// "Олдсонгүй" дангаараа юу олдоогүйг хэлдэггүй тул тодорхой болгов —
	// энэ мессежийг frontend шууд хэрэглэгчид харуулна.
	ErrNotFound:          {"Мэдээлэл олдсонгүй", "Requested data was not found"},
	ErrRouteNotFound:     {"Мэдээлэл олдсонгүй", "Requested endpoint was not found"},
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
	ErrTooManyRequests:   {"Хэт олон хүсэлт илгээгдсэн байна", "Too many requests"},

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
	ErrAU2Crossing:              {"Сум/дүүрэг дамнах боломжгүй. Чөлөөлөх хил нь зөвхөн нэг сум/дүүргийн хил дотор байх шаардлагатай.", "Cannot span multiple districts. The acquisition boundary must lie within a single district (au2)."},
	ErrOutsidePlanExceeds:       {"Чөлөөлөх хил төлөвлөгөөний нэгж талбарын хилээс хэт их (50%-иас дээш) гарсан байна.", "The acquisition boundary extends beyond the plan parcel boundary by more than 50% of the plan area."},
	ErrOutsideParcelBoundary:    {"Чөлөөлөх хил/талбай нь нэгж талбарын үндсэн хил дотор байх шаардлагатай. Оруулсан хил үндсэн хилээс гарсан байна.", "The acquisition geometry must lie inside the parcel's own boundary."},
	ErrShapefileProjection:      {"Оруулсан файлын проекц зөрүүтэй. Хил нь WGS84 (EPSG:4326) координатын системд, градусаар байх шаардлагатай.", "Invalid projection: the boundary must be in WGS84 (EPSG:4326) degrees."},
	ErrShapefileOutsideCountry:  {"Оруулсан хилийн координат Монгол Улсын хүрээнд байхгүй. Файлын проекц эсвэл координатыг шалгана уу.", "The boundary coordinates fall outside Mongolia. Check the file's projection."},
	ErrShapefileMultiRecord:     {"Оруулсан файлд нэгээс олон полигон байна. Нэг полигон агуулсан файл оруулна уу.", "The file contains more than one polygon. Upload a file with exactly one polygon."},
	ErrShapefileShapeType:       {"Оруулсан файлын геометр нь полигон биш байна (Polygon төрөл шаардлагатай). Цэг (Point) эсвэл шугам (PolyLine) төрөл дэмжигдэхгүй.", "The geometry must be a polygon; point and line types are not supported."},
	ErrShapefileMultiPolygon:    {"Оруулсан файлд салангид олон полигон (MultiPolygon) байна. Хил нь нэг полигон байх шаардлагатай (цоорхой бүхий полигон зөвшөөрөгдөнө).", "MultiPolygon is not supported: the boundary must be a single polygon (holes are allowed)."},
	ErrShapefileFormat:          {"Оруулсан файл ESRI Shapefile (.shp) бүтцэд тохирохгүй эсвэл эвдэрсэн байна.", "The file is not a valid ESRI Shapefile (.shp)."},
	ErrShapefileRequiredForPlan: {"Төлөвлөгөө солих бол шинэ shapefile заавал шаардлагатай", "Shapefile is required when changing plan"},
	ErrDeleteNotNew:             {"Зөвхөн NEW статустай чөлөөлөлтийг устгах боломжтой", "Only acquisitions with NEW status can be deleted"},
	ErrHasReleasedParcels:       {"Чөлөөлсөн төлөвтэй нэгж талбар байгаа тул чөлөөлөлтийг устгах боломжгүй.", "The acquisition has released parcels and cannot be deleted."},
	ErrConfirmedDecisionLinked:  {"Баталгаажсан захирамжтай холбогдсон тул чөлөөлөлтийг устгах боломжгүй.", "The acquisition is linked to a confirmed decision and cannot be deleted."},
	WarnGeomChanged:             {"Чөлөөлөлтийн хил шинэчлэгдлээ. Хуучин холбогдсон нэгж талбарууд хэвээр үлдэж, шинэ хилтэй давхцах дутуу нэгж талбарууд нэмэгдлээ", "The acquisition boundary was updated. Existing linked parcels were kept, and missing parcels intersecting the new boundary were added"},
	ErrCodeRequired:             {"code параметр шаардлагатай", "code parameter is required"},

	ErrStatusRequired:          {"to_status шаардлагатай", "to_status is required"},
	ErrInvalidStatus:           {"Буруу статус", "Invalid status"},
	ErrNotAssigned:             {"Энэ чөлөөлөлтэд ажиллах эрх байхгүй", "You are not assigned to this acquisition"},
	ErrApproved:                {"Санхүүгийн мэргэжилтэн зөвшөөрсөн мэдээллийг өөрчлөх боломжгүй", "Approved record cannot be modified"},
	ErrValuationReportRequired: {"Үнэлгээний тайлан хавсаргаагүй байна. Чөлөөлсөн болгохын өмнө тайлан оруулах шаардлагатай.", "Valuation report required before releasing the parcel."},
	ErrAcqLocked:               {"Баталгаажсан чөлөөлөлтийн мэдээллийг өөрчлөх боломжгүй", "Confirmed acquisition cannot be modified"},
	ErrNonFinalParcels:         {"Бүх нэгж талбар Чөлөөлсөн, Нөлөөллөөс гарсан, Татгалзсан төлөвийн аль нэг болсон байх шаардлагатай.", "All parcels must be Released, Affected out, or Rejected before confirming the acquisition."},
	ErrAssetPhotoRequired:      {"Үл хөдлөх төрөлтэй хөрөнгө бүрт зураг (PDF) хавсаргасны дараа илгээх боломжтой.", "Every real estate asset must have a photo attached before submitting the valuation."},
	ErrReportBeforeApproval:    {"Хөрөнгийн үнэлгээ баталгаажсаны дараа үнэлгээний тайлан хавсаргах боломжтой.", "The valuation report can be attached only after the valuation is approved."},
	ErrParcelAppNoMissing:      {"Нэгж талбарын хүсэлтийн дугаар (app_no) байхгүй тул хавсралт татах боломжгүй. Эхлээд нэгж талбарын мэдээллийг шинэчилнэ үү.", "The parcel has no application number yet. Sync the parcel details first."},
	ErrParcelOwnedNoPayment:    {"Өмчлөлийн газар тул төлбөрийн мэдээлэл байхгүй.", "This parcel is privately owned, so it has no land payment."},
	ErrSourceFileUnavailable:   {"Эх системийн файл сервер тохируулагдаагүй тул хавсралтын файлыг харах боломжгүй.", "The source file server is not configured, so the attachment cannot be opened."},

	ErrFileRequired:           {"file талбар шаардлагатай", "file field is required"},
	ErrFileTooLarge:           {"Файл 50MB-аас хэтэрсэн байна", "file exceeds 50MB limit"}, // upload_validation.go лимиттэй ижил
	ErrDroneFileType:          {"Зөвхөн GeoTIFF (.tif / .tiff) файл зөвшөөрөгдөнө", "only GeoTIFF (.tif/.tiff) files are allowed"},
	ErrDroneUploadIncomplete:  {"Зураг файлын системд бүрэн ороогүй байна. Дахин байршуулна уу", "uploaded image was not found in file storage"},
	ErrDroneAlreadyRegistered: {"Зураг аль хэдийн бүртгэгдсэн байна", "image is already registered"},
	ErrDroneTooLarge:          {"Зургийн хэмжээ хэтэрсэн байна. Дээд хязгаар 500 MB", "image exceeds the maximum allowed size of 500 MB"},
	ErrFileOpen:               {"Файл нээхэд алдаа гарлаа", "failed to open file"},
	ErrFileRead:               {"Файл уншихад алдаа гарлаа", "failed to read file"},
	ErrFileType:               {"Зөвхөн PDF файл зөвшөөрөгдөнө", "only PDF files are allowed"},
	ErrDocNotFound:            {"Баримт олдсонгүй", "document not found"},

	ErrDecisionConfirmed:     {"Баталгаажсан захирамжийн төсөлд өөрчлөлт оруулах боломжгүй", "confirmed decision draft cannot be modified"},
	ErrParcelNotReleased:     {"Зөвхөн 'Чөлөөлсөн' төлөвтэй нэгж талбарыг захирамжид холбоно", "only released parcels can be linked to a decision draft"},
	ErrParcelAlreadyLinked:   {"Тухайн нэгж талбар өөр захирамжид аль хэдийн холбогдсон байна", "parcel is already linked to another decision draft"},
	ErrDecisionAcqMismatch:   {"Захирамжид өөр чөлөөлөлтөд харьяалагдах нэгж талбар холбох боломжгүй", "parcel belongs to a different land acquisition"},
	ErrDecreeRequired:        {"Захирамжийн дугаар болон огноог заавал бөглөнө", "decree number and date are required"},
	ErrFundingSourceRequired: {"Эхлээд захирамжид санхүүгийн эх үүсвэр нэмээд, нэгж талбар бүрт эх үүсвэрээ сонгоно уу", "add a funding source to the decision draft and select one for the parcel"},
	ErrFundingSourceInUse:    {"Энэ эх үүсвэрийг нэгж талбар ашиглаж байгаа тул хасах боломжгүй", "funding source is in use by a linked parcel"},
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
