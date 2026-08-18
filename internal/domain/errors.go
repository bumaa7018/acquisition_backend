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
	ErrAcquisitionLocked       = errors.New("acquisition is locked")
	ErrInvalidTransition       = errors.New("invalid status transition")
	ErrValuationNotApproved    = errors.New("compensation valuation is not approved yet")
	ErrNonFinalParcels         = errors.New("all parcels must be in final status before confirming acquisition")
	ErrAssetPhotoRequired      = errors.New("real estate assets must have photos before submitting valuation")
	ErrReportBeforeApproval    = errors.New("valuation must be approved before attaching the report")
	// Хавсралт нь хүсэлтийн дугаараар (app_no) татагддаг тул нэгж талбарын
	// дэлгэрэнгүй эхлээд татагдсан байх ёстой.
	ErrParcelAppNoMissing = errors.New("parcel has no application number yet, sync the parcel first")
	// Өмчлөлийн (right_type=3) газарт төлбөрийн нэхэмжлэл байдаггүй.
	ErrParcelOwnedNoPayment = errors.New("owned parcel has no land payment")
	// Эх системийн файл серверийн хаяг (.env) тохируулаагүй.
	ErrSourceFileNotConfigured = errors.New("source file server is not configured")

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
	// Чөлөөлөх хил төлөвлөгөөний нэгж талбарын хилээс хэт (50%-иас их) гарсан.
	ErrOutsidePlanExceeds = errors.New("acquisition boundary outside the plan parcel exceeds 50% of the plan area")
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
)
