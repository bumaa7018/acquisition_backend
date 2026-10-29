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
	ErrPlanNotFound            Key = "plan_not_found"
	ErrAcqNotFound             Key = "acq_not_found"
	ErrRequiredFields          Key = "required_fields"
	ErrStartDateFormat         Key = "start_date_format"
	ErrEndDateFormat           Key = "end_date_format"
	ErrShapefileRead           Key = "shapefile_read"
	ErrPlanBoundaryMissing     Key = "plan_boundary_missing"
	ErrBoundaryExceeds         Key = "boundary_exceeds"
	ErrNewBoundaryExceeds      Key = "new_boundary_exceeds"
	ErrAU2Crossing             Key = "au2_crossing"
	ErrOutsideParcelBoundary   Key = "outside_parcel_boundary"
	ErrBoundaryAreaDeviation   Key = "boundary_area_deviation"
	ErrBoundarySourceConflict  Key = "boundary_source_conflict"
	ErrShapefileRequired       Key = "shapefile_required"
	ErrShapefileProjection     Key = "shapefile_projection"
	ErrShapefileOutsideCountry Key = "shapefile_outside_country"
	ErrShapefileMultiRecord    Key = "shapefile_multi_record"
	ErrShapefileShapeType      Key = "shapefile_shape_type"
	ErrShapefileMultiPolygon   Key = "shapefile_multipolygon"
	ErrShapefileFormat         Key = "shapefile_format"
	ErrPersonDeleteForbidden   Key = "person_delete_forbidden"
	ErrHasReleasedParcels      Key = "has_released_parcels"
	ErrCategoryInUse           Key = "category_in_use"
	ErrConfirmedDecisionLinked Key = "confirmed_decision_linked"
	ErrDeleteNotNew            Key = "delete_not_new"
	WarnGeomChanged            Key = "geom_changed"
	ErrCodeRequired            Key = "code_required"

	// Status errors
	ErrStatusRequired                    Key = "status_required"
	ErrInvalidStatus                     Key = "invalid_status"
	ErrNotAssigned                       Key = "not_assigned"
	ErrApproved                          Key = "approved_locked"
	ErrValuationReportRequired           Key = "valuation_report_required"
	ErrParcelProgressLocked              Key = "parcel_progress_locked"
	ErrStatusAttachmentRequired          Key = "status_attachment_required"
	ErrValuationCancelAttachmentRequired Key = "valuation_cancel_attachment_required"
	ErrAcqLocked                         Key = "acq_locked"
	ErrNonFinalParcels                   Key = "non_final_parcels"
	ErrParcelsNotDiscovered              Key = "parcels_not_discovered"
	ErrParcelsOverlap                    Key = "parcels_overlap"
	ErrNotFieldStage                     Key = "acq_not_field_stage"
	ErrAssetPhotoRequired                Key = "asset_photo_required"
	ErrValuationSubmittedLocked          Key = "valuation_submitted_locked"
	ErrSubmitReportRequired              Key = "submit_report_required"
	ErrSubmitWorkPhotoRequired           Key = "submit_work_photo_required"
	ErrReportBeforeApproval              Key = "report_before_approval"
	ErrParcelAppNoMissing                Key = "parcel_app_no_missing"
	ErrParcelOwnedNoPayment              Key = "parcel_owned_no_payment"
	ErrSourceFileUnavailable             Key = "source_file_unavailable"
	ErrOwnershipUnavailable              Key = "ownership_unavailable"
	ErrSurveyFileRequired                Key = "survey_file_required"
	ErrInvalidEmail                      Key = "invalid_email"
	ErrEmailNotConfigured                Key = "email_not_configured"
	ErrEmailSendFailed                   Key = "email_send_failed"
	ErrExternalAPIDisabled               Key = "external_api_disabled"
	ErrExternalFilterRequired            Key = "external_filter_required"
	ErrRegisterNoRequired                Key = "register_no_required"
	ErrEMongoliaUnavailable              Key = "emongolia_unavailable"
	ErrEMongoliaSendFailed               Key = "emongolia_send_failed"
	ErrBasemapType                       Key = "basemap_type"
	ErrBasemapURL                        Key = "basemap_url"
	ErrBasemapTileURL                    Key = "basemap_tile_url"
	ErrBasemapLayer                      Key = "basemap_layer"
	ErrBasemapExtent                     Key = "basemap_extent"

	// Document / file errors
	ErrFileRequired           Key = "file_required"
	ErrFileTooLarge           Key = "file_too_large"
	ErrLegacyFileTooLarge     Key = "legacy_file_too_large"
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
	ErrDecisionAttachment    Key = "decision_attachment_required"
	ErrDecisionDocLocked     Key = "decision_document_locked"
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

	ErrPlanNotFound:            {"Төлөвлөгөө олдсонгүй", "Plan not found"},
	ErrAcqNotFound:             {"Чөлөөлөлт олдсонгүй", "Land acquisition not found"},
	ErrRequiredFields:          {"plan_code, start_date, end_date заавал шаардлагатай", "plan_code, start_date, end_date are required"},
	ErrStartDateFormat:         {"start_date формат буруу (YYYY-MM-DD)", "Invalid start_date format (YYYY-MM-DD)"},
	ErrEndDateFormat:           {"end_date формат буруу (YYYY-MM-DD)", "Invalid end_date format (YYYY-MM-DD)"},
	ErrShapefileRead:           {"shapefile уншихад алдаа", "Failed to read shapefile"},
	ErrPlanBoundaryMissing:     {"Төлөвлөгөөний нэгж талбарын хил байхгүй байна. Чөлөөлөлтийн хил нь төлөвлөгөөний хилээс хуулагддаг тул үргэлжлүүлэх боломжгүй.", "The plan parcel has no boundary. The acquisition boundary is copied from the plan, so it cannot continue."},
	ErrBoundaryExceeds:         {"Төлөвлөгөөний хил шалгалтад тэнцсэнгүй", "The plan boundary failed validation"},
	ErrNewBoundaryExceeds:      {"Төлөвлөгөөний шинэ хил шалгалтад тэнцсэнгүй", "The new plan boundary failed validation"},
	ErrAU2Crossing:             {"Сум/дүүрэг дамнах боломжгүй. Чөлөөлөх хил нь зөвхөн нэг сум/дүүргийн хил дотор байх шаардлагатай.", "Cannot span multiple districts. The acquisition boundary must lie within a single district (au2)."},
	ErrOutsideParcelBoundary:   {"Чөлөөлөх хил/талбай нь нэгж талбарын үндсэн хил дотор байх шаардлагатай. Оруулсан хил үндсэн хилээс гарсан байна.", "The acquisition geometry must lie inside the parcel's own boundary."},
	ErrBoundaryAreaDeviation:   {"Оруулсан хилийн талбай нь жишиг хилийн (төлөвлөгөөний хил) талбайгаас 30%-иас их зөрсөн байна. Хилийн зөрүү 30%-иас бага байх шаардлагатай — файлын хил, проекцийг шалгана уу.", "The uploaded boundary area deviates by 30% or more from the reference (plan) boundary. The deviation must stay below 30%."},
	ErrBoundarySourceConflict:  {"Хил солих хоёр аргыг зэрэг ашиглах боломжгүй. Төлөвлөгөөний дугаараар хайх эсвэл хилийн файл оруулах аль нэгийг сонгоно уу.", "Cannot change the boundary from a plan and a shapefile at the same time. Choose one."},
	ErrShapefileRequired:       {"Хилийн файл (.shp) оруулна уу.", "A boundary shapefile (.shp) is required."},
	ErrShapefileProjection:     {"Оруулсан файлын проекц зөрүүтэй. Хил нь WGS84 (EPSG:4326) координатын системд, градусаар байх шаардлагатай.", "Invalid projection: the boundary must be in WGS84 (EPSG:4326) degrees."},
	ErrShapefileOutsideCountry: {"Оруулсан хилийн координат Монгол Улсын хүрээнд байхгүй. Файлын проекц эсвэл координатыг шалгана уу.", "The boundary coordinates fall outside Mongolia. Check the file's projection."},
	ErrShapefileMultiRecord:    {"Оруулсан файлд нэгээс олон полигон байна. Нэг полигон агуулсан файл оруулна уу.", "The file contains more than one polygon. Upload a file with exactly one polygon."},
	ErrShapefileShapeType:      {"Оруулсан файлын геометр нь полигон биш байна (Polygon төрөл шаардлагатай). Цэг (Point) эсвэл шугам (PolyLine) төрөл дэмжигдэхгүй.", "The geometry must be a polygon; point and line types are not supported."},
	ErrShapefileMultiPolygon:   {"Оруулсан файлд салангид олон полигон (MultiPolygon) байна. Хил нь нэг полигон байх шаардлагатай (цоорхой бүхий полигон зөвшөөрөгдөнө).", "MultiPolygon is not supported: the boundary must be a single polygon (holes are allowed)."},
	ErrShapefileFormat:         {"Оруулсан файл ESRI Shapefile (.shp) бүтцэд тохирохгүй эсвэл эвдэрсэн байна.", "The file is not a valid ESRI Shapefile (.shp)."},
	ErrDeleteNotNew:            {"Зөвхөн NEW статустай чөлөөлөлтийг устгах боломжтой", "Only acquisitions with NEW status can be deleted"},
	ErrPersonDeleteForbidden:   {"Хүний мэдээллийг устгах боломжгүй. Ажилтныг чөлөөлөх бол ажилтны бүртгэлийг идэвхгүй болгоно.", "Person records cannot be deleted; deactivate the employee instead."},
	ErrHasReleasedParcels:      {"Чөлөөлсөн төлөвтэй нэгж талбар байгаа тул чөлөөлөлтийг устгах боломжгүй.", "The acquisition has released parcels and cannot be deleted."},
	ErrCategoryInUse:           {"Энэ ангилалд (эсвэл дэд ангилалд нь) чөлөөлөлт холбогдсон тул устгах боломжгүй. Устгасан чөлөөлөлт ч холбоосоо хадгалдаг. Эхлээд тэдгээрийн ангилалыг өөрчилнө үү.", "The category (or one of its sub-categories) is used by acquisitions and cannot be deleted; deleted acquisitions still hold the reference. Reassign them first."},
	ErrConfirmedDecisionLinked: {"Баталгаажсан захирамжтай холбогдсон тул чөлөөлөлтийг устгах боломжгүй.", "The acquisition is linked to a confirmed decision and cannot be deleted."},
	WarnGeomChanged:            {"Чөлөөлөлтийн хил шинэчлэгдлээ. Нэгж талбарт өөрчлөлт ороогүй", "The acquisition boundary was updated. Parcels were not changed"},
	ErrCodeRequired:            {"code параметр шаардлагатай", "code parameter is required"},

	ErrStatusRequired:                    {"to_status шаардлагатай", "to_status is required"},
	ErrInvalidStatus:                     {"Буруу статус", "Invalid status"},
	ErrNotAssigned:                       {"Энэ чөлөөлөлтэд ажиллах эрх байхгүй", "You are not assigned to this acquisition"},
	ErrApproved:                          {"Санхүүгийн мэргэжилтэн зөвшөөрсөн мэдээллийг өөрчлөх боломжгүй", "Approved record cannot be modified"},
	ErrParcelProgressLocked:              {"Нэгж талбарын одоогийн төлөвт явц түгжигдсэн тул солих боломжгүй («Явц түгжинэ» дүрэм).", "Parcel progress is locked by its current status."},
	ErrValuationReportRequired:           {"Үнэлгээний тайлан хавсаргаагүй байна. Чөлөөлсөн болгохын өмнө тайлан оруулах шаардлагатай.", "Valuation report required before releasing the parcel."},
	ErrStatusAttachmentRequired:          {"Татгалзсан төлөвт шилжихэд шалтгаанаас гадна хавсралт заавал оруулна.", "An attachment is required to reject the parcel."},
	ErrValuationCancelAttachmentRequired: {"Үнэлгээ цуцлахад тайлбар болон PDF хавсралт заавал оруулна.", "A PDF attachment is required to cancel the valuation."},
	ErrAcqLocked:                         {"Баталгаажсан чөлөөлөлтийн мэдээллийг өөрчлөх боломжгүй", "Confirmed acquisition cannot be modified"},
	ErrNonFinalParcels:                   {"Бүх нэгж талбар Чөлөөлсөн, Нөлөөллөөс гарсан, Татгалзсан төлөвийн аль нэг болсон байх шаардлагатай.", "All parcels must be Released, Affected out, or Rejected before confirming the acquisition."},
	ErrParcelsNotDiscovered:              {"Нэгж талбар татагдаагүй байна. \"Хээрийн судалгаа\" төлөвт шилжихийн тулд чөлөөлөх хилээр нэгж талбаруудыг бүрэн татсан байх шаардлагатай.", "No parcels have been discovered. Discover all parcels within the boundary before moving to field survey."},
	ErrParcelsOverlap:                    {"Нэгж талбарууд хоорондоо байршлаар давхцаж байна. Давхардлыг арилгах хүртэл \"Хээрийн судалгаа\" төлөвт шилжих боломжгүй — нэг газрын нөхөх олговор хоёр удаа бодогдох эрсдэлтэй.", "Parcels overlap each other by location. Resolve the overlaps before moving to field survey."},
	ErrNotFieldStage:                     {"Нэгж талбартай ажиллахын тулд чөлөөлөлт \"Хээрийн судалгаа\" төлөвт байх шаардлагатай.", "Working with parcels requires the acquisition to be in field survey status."},
	ErrAssetPhotoRequired:                {"Үл хөдлөх төрөлтэй хөрөнгө бүрт зураг (PDF) хавсаргасны дараа илгээх боломжтой.", "Every real estate asset must have a photo attached before submitting the valuation."},
	ErrValuationSubmittedLocked:          {"Үнэлгээг илгээсэн/баталгаажсан тул засварлах боломжгүй. Буцаагдсан эсвэл илгээгээгүй төлөвт засна.", "The valuation has been submitted or approved and can no longer be edited."},
	ErrSubmitReportRequired:              {"Үнэлгээ илгээхийн өмнө \"Үнэлгээний тайлан\"-г хавсаргана уу.", "Attach the valuation report before submitting."},
	ErrSubmitWorkPhotoRequired:           {"Үнэлгээ илгээхийн өмнө \"Ажлын зураг\"-ийг хавсаргана уу.", "Attach the work photos before submitting."},
	ErrReportBeforeApproval:              {"Хөрөнгийн үнэлгээ баталгаажсаны дараа үнэлгээний тайлан хавсаргах боломжтой.", "The valuation report can be attached only after the valuation is approved."},
	ErrParcelAppNoMissing:                {"Нэгж талбарын хүсэлтийн дугаар (app_no) байхгүй тул хавсралт татах боломжгүй. Эхлээд нэгж талбарын мэдээллийг шинэчилнэ үү.", "The parcel has no application number yet. Sync the parcel details first."},
	ErrParcelOwnedNoPayment:              {"Өмчлөлийн газар тул төлбөрийн мэдээлэл байхгүй.", "This parcel is privately owned, so it has no land payment."},
	ErrSourceFileUnavailable:             {"Эх системийн файл сервер тохируулагдаагүй тул хавсралтын файлыг харах боломжгүй.", "The source file server is not configured, so the attachment cannot be opened."},
	ErrOwnershipUnavailable:              {"ХУР системийн холболт тохируулагдаагүй тул УБЕГ-ийн өмчлөлийн мэдээлэл татах боломжгүй.", "The XYP gateway is not configured, so the state ownership registry cannot be fetched."},
	ErrSurveyFileRequired:                {"Нийгэм, эдийн засгийн судалгааны файлыг (PDF) заавал хавсаргана.", "The socio-economic survey file (PDF) is required."},
	ErrInvalidEmail:                      {"Имэйл хаяг буруу байна.", "The email address is invalid."},
	ErrEmailNotConfigured:                {"Имэйл серверийн тохиргоо байхгүй тул мэдэгдэл илгээх боломжгүй.", "Email is not configured, so the notice cannot be sent."},
	ErrEmailSendFailed:                   {"Имэйл илгээхэд алдаа гарлаа. Хаягийг шалгаад дахин оролдоно уу.", "Sending the email failed. Check the address and try again."},
	ErrExternalAPIDisabled:               {"Гадаад системийн API тохируулагдаагүй байна.", "The external system API is not configured."},
	ErrExternalFilterRequired:            {"parcel_id, acquisition_id эсвэл plan_code шүүлтүүрээс дор хаяж нэгийг заана уу.", "Provide at least one filter: parcel_id, acquisition_id or plan_code."},
	ErrRegisterNoRequired:                {"Хүлээн авагчийн регистрийн дугаарыг оруулна уу.", "The recipient register number is required."},
	ErrEMongoliaUnavailable:              {"E-Mongolia-ийн холболт тохируулагдаагүй байна.", "The E-Mongolia gateway is not configured."},
	ErrEMongoliaSendFailed:               {"E-Mongolia-руу мэдэгдэл илгээхэд алдаа гарлаа.", "Sending the notice to E-Mongolia failed."},
	ErrBasemapType:                       {"Суурь зургийн төрөл xyz (tile), wms эсвэл image байх ёстой.", "Basemap type must be xyz (tile), wms or image."},
	ErrBasemapURL:                        {"Суурь зургийн хаяг http:// эсвэл https://-ээр эхлэх ёстой.", "The basemap URL must start with http:// or https://."},
	ErrBasemapTileURL:                    {"Tile хаягт {x}, {y}, {z} орлуулагч байх ёстой (ж: https://.../{z}/{y}/{x}.png).", "The tile URL must contain {x}, {y} and {z} placeholders."},
	ErrBasemapLayer:                      {"WMS суурь зурагт давхаргын нэр (layers) шаардлагатай.", "A WMS basemap requires a layer name."},
	ErrBasemapExtent:                     {"Зургийн хүрээг (minLon, minLat, maxLon, maxLat) зөв оруулна уу.", "Provide a valid image extent (minLon, minLat, maxLon, maxLat)."},

	ErrFileRequired:           {"file талбар шаардлагатай", "file field is required"},
	ErrFileTooLarge:           {"Файл 20MB-аас хэтэрсэн байна", "file exceeds 20MB limit"},   // upload_validation.go лимиттэй ижил
	ErrLegacyFileTooLarge:     {"Файл 100MB-аас хэтэрсэн байна", "file exceeds 100MB limit"}, // maxLegacyDocumentUploadSize-тай ижил
	ErrDroneFileType:          {"Зөвхөн GeoTIFF (.tif / .tiff) файл зөвшөөрөгдөнө", "only GeoTIFF (.tif/.tiff) files are allowed"},
	ErrDroneUploadIncomplete:  {"Зураг файлын системд бүрэн ороогүй байна. Дахин байршуулна уу", "uploaded image was not found in file storage"},
	ErrDroneAlreadyRegistered: {"Зураг аль хэдийн бүртгэгдсэн байна", "image is already registered"},
	ErrDroneTooLarge:          {"Зургийн хэмжээ хэтэрсэн байна. Дээд хязгаар 500 MB", "image exceeds the maximum allowed size of 500 MB"},
	ErrFileOpen:               {"Файл нээхэд алдаа гарлаа", "failed to open file"},
	ErrFileRead:               {"Файл уншихад алдаа гарлаа", "failed to read file"},
	ErrFileType:               {"Зөвхөн PDF файл зөвшөөрөгдөнө", "only PDF files are allowed"},
	ErrDocNotFound:            {"Баримт олдсонгүй", "document not found"},

	ErrDecisionConfirmed:     {"Баталгаажсан захирамжийн төсөлд өөрчлөлт оруулах боломжгүй", "confirmed decision draft cannot be modified"},
	ErrDecisionAttachment:    {"Захирамжийг баталгаажуулахад хавсралт файл (PDF) заавал хавсаргана", "an attachment file (PDF) is required to confirm the decision"},
	ErrDecisionDocLocked:     {"Баталгаажсан захирамжийн хавсралтыг нэгж талбараас устгах боломжгүй", "a confirmed decision's attachment cannot be deleted from a parcel"},
	ErrParcelNotReleased:     {"Зөвхөн «Захирамжид холбогдоно» дүрэмтэй төлөвтэй (жишээ нь 'Чөлөөлсөн') нэгж талбарыг захирамжид холбоно", "only released parcels can be linked to a decision draft"},
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
