package domain

import "errors"

var (
	ErrNotFound                = errors.New("not found")
	ErrAlreadyExists           = errors.New("already exists")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrTooManyRequests         = errors.New("too many requests")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("token expired")
	ErrInvalidInput            = errors.New("invalid input")
	ErrInternalServer          = errors.New("internal server error")
	ErrDecreeRequired          = errors.New("decree number and date are required")
	ErrApproved                = errors.New("approved record cannot be modified")
	ErrValuationReportRequired = errors.New("valuation report required before releasing parcel")
	// ErrStatusAttachmentRequired — "Татгалзсан" төлөвт шилжихэд хавсралт заавал.
	ErrStatusAttachmentRequired          = errors.New("attachment required to reject parcel")
	ErrValuationCancelAttachmentRequired = errors.New("attachment required to cancel valuation")
	ErrAcquisitionLocked                 = errors.New("acquisition is locked")
	ErrInvalidTransition                 = errors.New("invalid status transition")
	ErrValuationNotApproved              = errors.New("compensation valuation is not approved yet")
	ErrNonFinalParcels                   = errors.New("all parcels must be in final status before confirming acquisition")
	ErrAssetPhotoRequired                = errors.New("real estate assets must have photos before submitting valuation")
	ErrReportBeforeApproval              = errors.New("valuation must be approved before attaching the report")
	// Хавсралт нь хүсэлтийн дугаараар (app_no) татагддаг тул нэгж талбарын
	// дэлгэрэнгүй эхлээд татагдсан байх ёстой.
	ErrParcelAppNoMissing = errors.New("parcel has no application number yet, sync the parcel first")
	// Өмчлөлийн (right_type=3) газарт төлбөрийн нэхэмжлэл байдаггүй.
	ErrParcelOwnedNoPayment = errors.New("owned parcel has no land payment")
	// Эх системийн файл серверийн хаяг (.env) тохируулаагүй.
	ErrSourceFileNotConfigured = errors.New("source file server is not configured")
	// ХУР (XYP) гарцын хаяг (.env: XYP_BASE_URL) тохируулаагүй тул УБЕГ-ийн
	// өмчлөлийн мэдээлэл татах боломжгүй.
	ErrOwnershipNotConfigured = errors.New("xyp ownership gateway is not configured")
	// Нийгэм, эдийн засгийн судалгааны эх баримт (PDF) заавал хавсрагдана.
	ErrSurveyFileRequired = errors.New("socio-economic survey file is required")
	// Имэйл хаяг буруу форматтай.
	ErrInvalidEmail = errors.New("invalid email address")
	// SMTP тохируулаагүй (.env: EMAIL_HOST) тул имэйл илгээх боломжгүй.
	ErrEmailNotConfigured = errors.New("email is not configured")
	// SMTP сервер илгээлтийг татгалзсан (хаяг олдсонгүй, холбогдсонгүй г.м).
	ErrEmailSendFailed = errors.New("email sending failed")
	// E-Mongolia мэдэгдэлд хүлээн авагчийн регистрийн дугаар шаардлагатай.
	ErrRegisterNoRequired = errors.New("recipient register number is required")
	// E-Mongolia-ийн гарц тохируулаагүй (мок ч ажиллахгүй үед).
	ErrEMongoliaNotConfigured = errors.New("e-mongolia gateway is not configured")
	// E-Mongolia-ийн гарц илгээлтийг татгалзсан.
	ErrEMongoliaSendFailed = errors.New("e-mongolia notice sending failed")

	// Газрын зургийн СУУРЬ зургийн тохиргоо (app_setting: basemap)
	ErrInvalidBasemapType    = errors.New("basemap type must be xyz, wms or image")
	ErrInvalidBasemapURL     = errors.New("basemap url must be an http(s) address")
	ErrInvalidBasemapTileURL = errors.New("tile url must contain {x}, {y} and {z} placeholders")
	ErrBasemapLayerRequired  = errors.New("wms basemap requires a layer name")
	ErrBasemapExtentRequired = errors.New("image basemap requires a valid extent")

	// Захирамжийн төсөл
	ErrDecisionConfirmed   = errors.New("confirmed decision draft cannot be modified")
	ErrParcelNotReleased   = errors.New("only released parcels can be linked to a decision draft")
	ErrParcelAlreadyLinked = errors.New("parcel is already linked to another decision draft")
	ErrDecisionAcqMismatch = errors.New("parcel belongs to a different land acquisition")
	// Нэгж талбар холбохын өмнө захирамжид эх үүсвэр нэмсэн байх ба талбар
	// бүрт эх үүсвэр сонгосон байх ёстой.
	ErrFundingSourceRequired = errors.New("a funding source must be added and selected first")
	ErrFundingSourceInUse    = errors.New("funding source is in use by a linked parcel")

	// Хилийн шалгалт (чөлөөлөлт үүсгэх / хил солих / нэгж талбарын хил)
	//
	// Чөлөөлөх хил нь нэг сум/дүүрэгт (au2) багтаагүй — дамнасан эсвэл
	// сум/дүүргийн хилээс гарсан.
	ErrAU2Crossing = errors.New("acquisition boundary crosses administrative unit (au2) borders")
	// Чөлөөлөлтийн хил нь ҮҮСГЭХ үедээ ТӨЛӨВЛӨГӨӨНИЙ хилээс хуулагддаг тул
	// төлөвлөгөөнд хил ирээгүй бол үүсгэх/солих боломжгүй.
	ErrPlanBoundaryMissing = errors.New("plan parcel has no boundary geometry")
	// Гараас (shapefile) оруулсан чөлөөлөлтийн хилийн талбай нь ЖИШИГ хилийн
	// (төлөвлөгөөний хил, эс байвал одоогийн хил) талбайгаас хэт их зөрсөн —
	// хээрийн хэмжилтийн засвар биш, огт өөр газрын/масштабын файл байх
	// магадлалтай (үзнэ үү: service.maxBoundaryAreaDeviation).
	ErrBoundaryAreaDeviation = errors.New("boundary area deviates too much from the reference boundary")
	// Хил солих хоёр зам (төлөвлөгөөний дугаар / shapefile) нэг хүсэлтэд зэрэг
	// ирсэн — аль хил хүчинтэй нь тодорхойгүй тул аль нэгийг нь сонгуулна.
	ErrBoundarySourceConflict = errors.New("provide either a plan parcel id or a shapefile, not both")
	// base.bs_person (хүний бүртгэл) нь ГУС-ийн эх өгөгдөл — УСТГАХ БОЛОМЖГҮЙ.
	ErrPersonDeleteForbidden = errors.New("person records cannot be deleted")

	// Нэгж талбарын чөлөөлөх хил/талбай нь тухайн нэгж талбарын ҮНДСЭН
	// хилээс гарсан (хатуу шалгалт).
	ErrOutsideParcelBoundary = errors.New("acquisition geometry must be inside the parcel boundary")

	// Оруулсан shapefile нь appdb-ийн стандартаас зөрсөн
	//
	// Проекц нь WGS84 (EPSG:4326) биш — координат градус биш эсвэл .prj өөр
	// координатын систем зарласан.
	ErrShapefileProjection = errors.New("shapefile projection must be WGS84 (EPSG:4326)")
	// Координат нь Монгол Улсын хүрээнд байхгүй.
	ErrShapefileOutsideCountry = errors.New("shapefile coordinates are outside Mongolia")
	// Нэгээс олон полигон бичлэгтэй файл (аль нь гэдэг тодорхойгүй).
	ErrShapefileMultiRecord = errors.New("shapefile must contain exactly one polygon")
	// Polygon биш геометр (шугам, цэг гэх мэт).
	ErrShapefileShapeType = errors.New("shapefile geometry must be a polygon")
	// Салангид олон полигон (MultiPolygon) — geometry(Polygon,4326) багананд
	// хадгалагдахгүй.
	ErrShapefileMultiPolygon = errors.New("multipolygon is not supported, a single polygon is required")
	// Формат/бүтэц эвдэрсэн эсвэл shapefile биш.
	ErrShapefileFormat = errors.New("invalid shapefile format")

	// Чөлөөлөлт устгах хязгаарлалт
	//
	// "Чөлөөлсөн" төлөвтэй нэгж талбар бүхий чөлөөлөлтийг устгах боломжгүй.
	ErrHasReleasedParcels = errors.New("acquisition has released parcels and cannot be deleted")
	// Баталгаажсан захирамжтай холбогдсон чөлөөлөлтийг устгах боломжгүй.
	ErrConfirmedDecisionLinked = errors.New("acquisition is linked to a confirmed decision draft")

	// Чөлөөлөх ХИЛ солих хязгаарлалт
	//
	// Шинэ хил нь "Чөлөөлсөн" төлөвтэй нэгж талбарыг гадна орхиж байвал хилийг
	// хүлээж авахгүй: тэр талбарууд шинэ хилээр хасагдах бөгөөд аль хэдийн
	// чөлөөлөгдсөн (олговор төлөгдсөн) бүртгэлийг устгах боломжгүй.
	ErrReleasedParcelOutsideBoundary = errors.New("new boundary excludes released parcels")

	// "ХЭЭРИЙН СУДАЛГАА" төлөв рүү шилжих хязгаарлалт
	//
	// Нэгж талбартай ажиллах (үнэлгээ, олговор, төлөв) нь ЗӨВХӨН "Хээрийн
	// судалгаа" төлөвөөс эхэлдэг. Тиймээс тэр төлөв рүү шилжихийн ӨМНӨ нэгж
	// талбарын бүрдэл ЗӨВ байх ёстой — дараа нь засах нь аль хэдийн бичигдсэн
	// үнэлгээ/олговрыг хөндөнө.

	// Нэгж талбар огт татагдаагүй бол хээрийн судалгаа хийх зүйл байхгүй.
	ErrParcelsNotDiscovered = errors.New("parcels must be discovered before field survey")
	// Нэгж талбарууд хоорондоо БАЙРШЛААР давхцаж байвал шилжүүлэхгүй: нэг
	// газрын нөхөх олговор хоёр удаа бодогдох эрсдэлтэй.
	ErrParcelsOverlap = errors.New("parcels overlap each other by location")
	// Нэгж талбартай ажиллах үйлдлийг "Шинэ" төлөвт хийх боломжгүй.
	ErrAcquisitionNotFieldStage = errors.New("parcel work requires field survey status")
)
