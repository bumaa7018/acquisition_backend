package domain

import (
	"encoding/json"
	"io"
	"math"
	"time"

	"github.com/google/uuid"
)

// Plan статус (auth DB)
const (
	PlanStatusDraft    = 1 // 1=DRAFT
	PlanStatusActive   = 2 // 2=ACTIVE
	PlanStatusArchived = 3 // 3=ARCHIVED
)

// LandAcquisition статус (main DB)
const (
	LandAcquisitionStatusNew       = 1 // Шинэ
	LandAcquisitionStatusField     = 2 // Хээрийн судалгаа
	LandAcquisitionStatusVerified  = 3 // Баталгаажсан
	LandAcquisitionStatusCancelled = 4 // Цуцлагдсан
)

// Parcel статус (main DB)
const (
	ParcelStatusPending     = 0 // Хүлээгдэж буй
	ParcelStatusProcessing  = 1 // Зөвшилцөх шатандаа
	ParcelStatusEvaluating  = 2 // Үнэлгээ хийх
	ParcelStatusAffectedOut = 3 // Нөлөөлөгдсөн гарсан
	ParcelStatusRejected    = 4 // Татгалзсан
	ParcelStatusReleased    = 5 // Чөлөөлсөн
)

// AcquisitionStatusLabels — статусын монгол нэрс
var AcquisitionStatusLabels = map[int]string{
	LandAcquisitionStatusNew:       "Шинэ",
	LandAcquisitionStatusField:     "Хээрийн судалгаа",
	LandAcquisitionStatusVerified:  "Баталгаажсан",
	LandAcquisitionStatusCancelled: "Цуцлагдсан",
}

// AcquisitionStatusOrder — дэс дугаарын эрэмбэ
var AcquisitionStatusOrder = []int{
	LandAcquisitionStatusNew,
	LandAcquisitionStatusField,
	LandAcquisitionStatusVerified,
	LandAcquisitionStatusCancelled,
}

// StatusOption — боломжтой явцын мэдээлэл
type StatusOption struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// Plan — газар зохион байгуулалтын төлөвлөгөө (auth DB-д хадгалагдана)
type Plan struct {
	ID           uuid.UUID
	ProjectID    string
	Code         string
	PlanCode     string
	Name         string
	PlanTypeName string
	BoundaryWKT  string // WKT текст хэлбэрт
	AreaM2       float64
	Status       int
	StartData    *string
	EndData      *string
	ApprovedDate *string
	// ParcelID — төлөвлөгөөний нэгж талбарын дугаар
	// (data_plan.pl_project_parcel.parcel_id). Төлөвлөгөөг ҮҮГЭЭР хайдаг.
	ParcelID string
	// Gazner — тухайн төлөвлөгөөний нэгж талбар дээр хийгдэх бүтээн
	// байгуулалтын ажлын нэр, ж: "Дамбадаржаа дэд төвийн бүтээн байгуулалтын ажил".
	Gazner    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SourceParcel — нэгж талбарын эх мэдээлэл (auth DB-д хадгалагдана)
type SourceParcel struct {
	ID                  uuid.UUID  `json:"id"`
	ParcelID            string     `json:"parcel_id"`
	OldParcelID         string     `json:"old_parcel_id"`
	PlanCode            string     `json:"plan_code"`
	GeometryWKT         string     `json:"geometry_wkt"`
	AU1Code             string     `json:"au1_code"`
	AU2Code             string     `json:"au2_code"`
	AU3Code             string     `json:"au3_code"`
	RightType           int        `json:"right_type"`
	Landuse             string     `json:"landuse"`
	LanduseName         string     `json:"landuse_name"`
	AreaM2              float64    `json:"area_m2"`
	AddressKhashaa      string     `json:"address_khashaa"`
	AddressStreet       string     `json:"address_streetname"`
	ValidFrom           *time.Time `json:"valid_from"`
	ValidTill           *time.Time `json:"valid_till"`
	PropertyNo          string     `json:"property_no"`
	HolderLastName      string     `json:"holder_last_name"`
	HolderName          string     `json:"holder_name"`
	HolderRegisterNo    string     `json:"holder_register_no"`
	HolderPhone         string     `json:"holder_phone"`
	HolderEmail         string     `json:"holder_email"`
	HolderType          string     `json:"holder_type"`
	HolderCivilID       string     `json:"holder_civil_id"`
	AppNo               string     `json:"app_no"`
	AppTimestamp        *time.Time `json:"app_timestamp"`
	AppType             string     `json:"app_type"`
	AppStatus           string     `json:"app_status"`
	DecisionNo          string     `json:"decision_no"`
	DecisionDate        *time.Time `json:"decision_date"`
	ContractNo          string     `json:"contract_no"`
	ContractDate        *time.Time `json:"contract_date"`
	ContractBegin       *time.Time `json:"contract_begin"`
	ContractEnd         *time.Time `json:"contract_end"`
	ContractPropertyNo  string     `json:"contract_property_no"`
	ContractStatus      string     `json:"contract_status"`
	CertificateNo       string     `json:"certificate_no"`
	CertificateDate     *time.Time `json:"certificate_date"`
	RecordNo            string     `json:"record_no"`
	RecordDate          *time.Time `json:"record_date"`
	RecordCertificateNo string     `json:"record_certificate_no"`
	RecordStatus        string     `json:"record_status"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ParcelBasePrice — газрын СУУРЬ ҮНЭ + үнэлгээний ХҮЧИН ЗҮЙЛийн задаргаа
// (ГУС data_cama, GET /parcel/base-price/{parcel_id}/factors).
//
// Status=false нь транспортын алдаа БИШ — эх систем нь суурь үнэ олдоогүй ч
// status=true буцаадаг тул практикт үргэлж true байна.
type ParcelBasePrice struct {
	Status    bool                    `json:"status"`
	ParcelID  string                  `json:"parcel_id"`
	BasePrice float64                 `json:"base_price"`
	Factors   []ParcelBasePriceFactor `json:"factor_values"`
}

// ParcelBasePriceFactor — нэгж талбарын НЭГ үнэлгээний хүчин зүйлийн утга.
type ParcelBasePriceFactor struct {
	FactorID       int      `json:"factor_id"`
	FactorCode     string   `json:"factor_code"`
	FactorName     string   `json:"factor_name"`
	FactorDataType string   `json:"factor_data_type"`
	FactorValue    *float64 `json:"factor_value"`
	YearValue      *int     `json:"year_value"`
	InActive       *bool    `json:"in_active"`
	// ClosetsParcelID — эх хүснэгтийн баганын нэр ЯГ ингэж бичигдсэн
	// (closets ≠ closest). ГУС-тай зөрөхгүйн тулд ЗАСААГҮЙ.
	ClosetsParcelID string `json:"closets_parcel_id"`
}

// ParcelFee — ГАЗРЫН ТӨЛБӨРийн бодолт (ГУС, POST /parcel/fee). Төлбөрийн бүс
// тус бүрээр задарна; нийт төлбөр нь Items-ийн payment-ийн нийлбэр.
//
// Status=false нь ТРАНСПОРТЫН алдаа БИШ: "төлбөр бодох боломжгүй" гэсэн
// бизнесийн дүгнэлт, шалтгаан нь Msg-д монголоор ирнэ.
type ParcelFee struct {
	Status       bool            `json:"status"`
	Msg          string          `json:"msg"`
	ParcelID     string          `json:"parcel_id"`
	AreaM2       float64         `json:"area_m2"`
	TotalPayment float64         `json:"total_payment"`
	Items        []ParcelFeeItem `json:"items"`
}

// ParcelFeeItem — нэг төлбөрийн бүсийн бодолт.
type ParcelFeeItem struct {
	ZoneID            string  `json:"zone_id"`
	ZoneType          string  `json:"zone_type"`
	ZoneNo            string  `json:"zone_no"`
	ZoneName          string  `json:"zone_name"`
	Area              float64 `json:"area"`
	LanduseArea       float64 `json:"landuse_area"`
	ZoneArea          float64 `json:"zone_area"`
	BaseFeeID         string  `json:"base_fee_id"`
	ResolutionID      string  `json:"resolution_id"`
	ConfidencePercent float64 `json:"confidence_percent"`
	SubsidizedArea    float64 `json:"subsidized_area"`
	SubsidizedFeeRate float64 `json:"subsidized_fee_rate"`
	BasePriceM2       float64 `json:"base_price_m2"`
	BaseFeePerM2      float64 `json:"base_fee_per_m2"`
	Landuses          string  `json:"landuses"`
	Value             float64 `json:"value"`
	Payment           float64 `json:"payment"`
}

// ParcelInvoice — газрын төлбөрийн нэхэмжлэл (ГУС data_estimate).
type ParcelInvoice struct {
	InvoiceNo   string  `json:"invoice_no"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	PaidAmount  float64 `json:"paid_amount"`
	StatusID    int     `json:"status_id"`
	StatusName  string  `json:"status_name"`
}

// ParcelMortgage — барьцааны бүртгэл. SourceParentID утгатай бол ДАВХАР
// барьцаа — эцэг барьцааныхаа SourceID-г заана.
type ParcelMortgage struct {
	ID                  uuid.UUID `json:"id"`
	SourceID            string    `json:"source_id"`
	SourceParentID      string    `json:"source_parent_id"`
	AppNo               string    `json:"app_no"`
	StartMortgagePeriod string    `json:"start_mortgage_period"`
	EndMortgagePeriod   string    `json:"end_mortgage_period"`
	LoanContractNo      string    `json:"loan_contract_no"`
	MortgageContractNo  string    `json:"mortgage_contract_no"`
	LoanContractAmount  float64   `json:"loan_contract_amount"`
	MonetaryUnitValue   float64   `json:"monetary_unit_value"`
	UnitType            string    `json:"unit_type"`
	MortgageType        string    `json:"mortgage_type"`
	StatusID            string    `json:"status_id"`
	StatusName          string    `json:"status_name"`
}

// ParcelCourtDecision — нэгж талбарт хамаарах шүүхийн шийдвэр.
type ParcelCourtDecision struct {
	ID              uuid.UUID `json:"id"`
	SourceID        string    `json:"source_id"`
	AppNo           string    `json:"app_no"`
	StartPeriod     string    `json:"start_period"`
	EndPeriod       string    `json:"end_period"`
	CourtStatus     string    `json:"court_status"`
	StatusName      string    `json:"status_name"`
	CourtDecisionNo string    `json:"court_decision_no"`
}

// ParcelMonitoring — хянан баталгааны мэдээлэл (data_monitoring).
type ParcelMonitoring struct {
	ID           uuid.UUID `json:"id"`
	MonitoringID string    `json:"monitoring_id"`
	PageNo       string    `json:"page_no"`
	StatusName   string    `json:"status_name"`
	CompanyName  string    `json:"company_name"`
	CreatedAt    string    `json:"created_at"`
}

// ParcelOwnership — УБЕГ-ийн газар өмчлөлийн бүртгэлийн НЭГ мөр
// (XYP /xyp/ownership/citizen/parcel).
//
// Нэг нэгж талбар дээр хамтран өмчлөгч олон байх ба өмчлөлийн ТҮҮХ (өв
// залгамжлал, өмчлөгч өөрчлөгдөх) тусад нь мөр болж ирдэг тул нэг register_no
// хэд хэдэн огноотой давтагдаж болно.
type ParcelOwnership struct {
	ID uuid.UUID `json:"id"`
	// SourceParcelID — УБЕГ дээрх нэгж талбарын дугаар (prprty_parcel_id).
	SourceParcelID string `json:"source_parcel_id"`
	// RecordDate — бүртгэлийн огноо, YYYYMMDD (эх системийн хэлбэрээр).
	RecordDate string `json:"record_date"`
	// CertificateNo — улсын бүртгэлийн/гэрчилгээний дугаар (ubd).
	CertificateNo string  `json:"certificate_no"`
	LanduseName   string  `json:"landuse_name"`
	Area          float64 `json:"area"`
	RegisterNo    string  `json:"register_no"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	// OwnershipType — өмчлөлийн хэлбэр (ж: "Өмчилсөн", "Өв залгамжлуулсан").
	OwnershipType string `json:"ownership_type"`
	OwnershipCode string `json:"ownership_code"`
	// ServiceName — бүртгэлийг үүсгэсэн улсын бүртгэлийн үйлчилгээ.
	ServiceName       string `json:"service_name"`
	ServiceCode       string `json:"service_code"`
	AimagCityName     string `json:"aimag_city_name"`
	AimagCode         string `json:"aimag_code"`
	SoumDistrictName  string `json:"soum_district_name"`
	SoumCode          string `json:"soum_code"`
	BagKhorooName     string `json:"bag_khoroo_name"`
	BagCode           string `json:"bag_code"`
	AddressStreetName string `json:"address_street_name"`
	BairNumber        string `json:"bair_number"`
	Toot              string `json:"toot"`
	FullAddress       string `json:"full_address"`
}

// ParcelSyncCountResult — татаж хадгалсан бичлэгийн тоо.
type ParcelSyncCountResult struct {
	Found int `json:"found"`
	Saved int `json:"saved"`
}

// ErrParcelOwned — өмчлөлийн (right_type=3) газарт төлбөр байхгүй.
// ParcelHolderSyncResult — эзэмшигчийн мэдээлэл татсаны үр дүн.
//   - Found   — ГУС-аас ирсэн нийт
//   - Saved   — хадгалагдсан (регистрийн давхардал хассаны дараах)
//   - Skipped — регистр давхцсан тул алгасагдсан
type ParcelHolderSyncResult struct {
	Found   int `json:"found"`
	Saved   int `json:"saved"`
	Skipped int `json:"skipped"`
}

// ParcelDiscoveryResult — чөлөөлөх ХИЛЭЭР дундын сервисээс нэгж талбарын
// ДУГААРУУДЫГ татаж бүртгэсний үр дүн.
//
// Дугаар бүрийг бүртгэхийн өмнө аль хэдийн бүртгэгдсэн эсэхийг шалгадаг тул
// дахин дуудахад давхардал үүсэхгүй:
//   - Created  — шинээр бүртгэгдсэн
//   - Existing — өмнө нь ЭНЭ чөлөөлөлтөд бүртгэгдсэн байсан
//   - Skipped  — ӨӨР чөлөөлөлтөд бүртгэгдсэн тул алгасагдсан
//     (parcel.parcel_id глобал unique)
type ParcelDiscoveryResult struct {
	Total    int `json:"total"`
	Created  int `json:"created"`
	Existing int `json:"existing"`
	Skipped  int `json:"skipped"`
	// Энэ чөлөөлөлтөд бүртгэлтэй болсон дугаарууд (created + existing)
	ParcelIDs []string `json:"parcel_ids"`
	// RemovedOutside — чөлөөлөх ХИЛД БАГТААГҮЙ тул хасагдсан нэгж талбарын тоо
	// (зөвхөн "Шинэ" төлөвт автоматаар хасагдана).
	RemovedOutside int `json:"removed_outside"`
	// OverlappingParcels — БАЙРШЛААР давхардсан нэгж талбарын тоо. 0-ээс их бол
	// "Хээрийн судалгаа" төлөв рүү шилжих боломжгүй.
	OverlappingParcels int `json:"overlapping_parcels"`
}

// BoundaryChange — чөлөөлөх ХИЛ солигдоход нэгж талбарт гарсан өөрчлөлт.
// Хил хөндөгдөөгүй үед nil (Update-ийн хариултад geometry_changed-ийн үүрэг
// хийнэ).
type BoundaryChange struct {
	// Added — шинэ хилээр НЭМЭГДСЭН (шинээр бүртгэгдсэн) нэгж талбар
	Added int `json:"added"`
	// Removed — шинэ хилд ороогүй тул БҮРМӨСӨН устгагдсан нэгж талбар
	Removed int `json:"removed"`
}

// ParcelRef — нэгж талбарын хөнгөн лавлагаа (хил солиход ГУС-аас олдсон
// дугаартай зөрүүлж, хасагдах талбаруудыг тодорхойлоход хэрэглэгдэнэ).
type ParcelRef struct {
	ID       uuid.UUID
	ParcelID string
	Status   int
}

// ConstructionType — бүтээн байгуулалтын төрөл (lookup, legacy)
type ConstructionType struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// AcquisitionCategory — чөлөөлөлтийн ангилал (ерөнхий болон дэд)
type AcquisitionCategory struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ParentID  *int   `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
	// DepartmentID — ХАРИУЦСАН АЛБА (authdb.sdplatform.hr_department).
	// Өөр сангийн бүртгэл тул FK байхгүй, нэрийг дэлгэц /departments-ээс
	// тааруулна. nil = тодорхойлоогүй.
	DepartmentID *int `json:"department_id"`
	// AcquisitionCount — ЭНЭ ангилалд холбогдсон чөлөөлөлтийн тоо
	// (ерөнхий ангилалд дэд ангилалуудаараа дамжсан нь ч орно).
	//
	// ЯАГААД: ангилалыг устгахад `land_acquisition`-ийн FK зөрчигдөж,
	// хэрэглэгчид "Дотоод алдаа гарлаа" гэсэн ойлгомжгүй 500 буцдаг байв.
	// Тоог урьдчилж харуулснаар устгахын ӨМНӨ анхааруулна.
	AcquisitionCount int `json:"acquisition_count"`
	// SubCount — дэд ангилалын тоо (зөвхөн ерөнхий ангилалд).
	SubCount int `json:"sub_count"`
}

// ParcelStatus — нэгж талбарын төлөв (lookup)
type ParcelStatus struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	// Color — `#rrggbb`. Төлөвийн өнгөний ЦОРЫН ГАНЦ эх сурвалж: жагсаалтын
	// тэмдэглэгээ, дашбоардын диаграм, мөн газрын зураг дээрх төлөвийн
	// давхарга (GeoServer-ийн SLD руу `env=fill:<өнгө>`-өөр дамжина) бүгд
	// эндээс авна. Хоосон бол дуудагч тал саарал өгөгдмөл хэрэглэнэ —
	// шинээр нэмсэн төлөв өнгө сонгоогүй байж болно.
	Color string `json:"color"`
}

// ParcelStatusHistory — нэгж талбарын статус солигдсон түүх
type ParcelStatusHistory struct {
	ID            int       `json:"id"`
	ParcelID      string    `json:"parcel_id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	StatusID      int       `json:"status_id"`
	StatusName    string    `json:"status_name"`
	StatusDate    time.Time `json:"status_date"`
	CreatedBy     string    `json:"created_by"`
	// Reason — төлөв солих ШАЛТГААН. "Нөлөөлөгдсөн гарсан"/"Татгалзсан"
	// үед заавал, бусад төлөвт хоосон байж болно.
	Reason string `json:"reason"`
	// Тухайн шилжилтийн ХАВСРАЛТ. "Татгалзсан" үед ЗААВАЛ. Файл нь нэгж
	// талбарын баримт болж мөн бүртгэгддэг тул "Хавсралт" хэсэгт ч харагдана.
	AttachmentURL  string `json:"attachment_url"`
	AttachmentName string `json:"attachment_name"`
}

// ParcelStatusChange — нэгж талбарын төлөв солих НЭГ үйлдэл.
//
// Хавсралт нэмэгдсэнээр аргументын тоо олширсон тул НЭГ бүтцээр дамжина
// (дуудагч бүр ижил талбар бөглөнө, дараалал андуурагдахгүй).
type ParcelStatusChange struct {
	ParcelID      uuid.UUID
	AcquisitionID uuid.UUID
	StatusID      int
	CreatedBy     string
	Reason        string
	// MinIO-д хадгалагдсаны дараах харьцангуй зам ба харагдах нэр.
	AttachmentURL  string
	AttachmentName string
}

// Нөхөх олговрын үнэлгээний илгээх/зөвшөөрөх төлөв (нэгж талбарын статусаас тусдаа).
const (
	ValuationStatusDraft     = "draft"     // Ноорог — засах боломжтой
	ValuationStatusSubmitted = "submitted" // Илгээсэн — засах хаагдсан, санхүү хянана
	ValuationStatusApproved  = "approved"  // Баталгаажсан — эцсийн, засах хаагдсан
	ValuationStatusReturned  = "returned"  // Буцаагдсан — засах боломжтой, дахин илгээнэ
	ValuationStatusRejected  = "rejected"  // Татгалзсан — өөр урсгал баталгаажсан үед автоматаар (эцсийн)
)

// Үнэлгээний урсгалын төрөл (нэгж талбар бүрт 3 тусдаа урсгал).
const (
	ValuationTypeAsset       = "asset"       // Үндсэн мэрг. байгууллагын үнэлгээ
	ValuationTypeIndependent = "independent" // Хөндлөнгийн байгууллагын үнэлгээ
	ValuationTypeMika        = "mika"        // МИКА-гийн үнэлгээ
)

// ValuationStreamAllowed — тухайн мөрийн урсгалд дуудагч ажиллаж болох эсэх.
//
// allowed хоосон = хязгаарлалтгүй (дотоод ахлах мэргэжилтэн/админ). Гадаад
// ролиуд зөвхөн ӨӨРИЙН урсгалдаа ажиллана — эс бөгөөс МИКА мэргэжлийн
// байгууллагын үнэлгээг (эсрэгээр нь ч) устгах боломжтой болно.
//
// Хоосон valuation_type нь түүхэн өгөгдөлд "asset" гэсэн утгатай (INSERT дээр
// COALESCE(NULLIF(...),'asset') хийгддэгтэй ижил).
func ValuationStreamAllowed(stream string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	if stream == "" {
		stream = ValuationTypeAsset
	}
	for _, a := range allowed {
		if a == stream {
			return true
		}
	}
	return false
}

// ValuationSubmission — нэгж талбарын нөхөх олговрын үнэлгээний илгээлтийн төлөв
type ValuationSubmission struct {
	ID            uuid.UUID  `json:"id"`
	AcquisitionID uuid.UUID  `json:"acquisition_id"`
	ParcelID      string     `json:"parcel_id"`
	ValuationType string     `json:"valuation_type"`
	Status        string     `json:"status"`
	SubmittedBy   string     `json:"submitted_by"`
	SubmittedAt   *time.Time `json:"submitted_at"`
	ReviewedBy    string     `json:"reviewed_by"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	LastNote      string     `json:"last_note"`
	// ХАМГИЙН СҮҮЛИЙН үйлдлийн хавсралт (`last_note`-той ижил зарчим).
	// Хоосон = хавсралтгүй. Хавсралт нь ЗААВАЛ БИШ.
	AttachmentURL  string `json:"attachment_url"`
	AttachmentName string `json:"attachment_name"`
	// БАТАЛГААЖААГҮЙ үнэлгээний тайлан — ИЛГЭЭХ үед хавсаргана (PDF/Word).
	// Буцаах/цуцлах үйлдэлд ХӨНДӨГДӨХГҮЙ (attachment_*-аас ялгаатай) тул
	// хянагч бүх шатанд уншиж чадна.
	DraftReportURL  string    `json:"draft_report_url"`
	DraftReportName string    `json:"draft_report_name"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ValuationTransition — үнэлгээний илгээх/зөвшөөрөх/буцаах шилжилтийн оролт.
//
// Бүтэц болгосон шалтгаан: шилжилт нь 8 параметртэй болсон (үүнээс 4 нь
// текст) — дараалалаар дамжуулбал хольж эндүүрэх нь цаг хугацааны асуудал.
type ValuationTransition struct {
	AcquisitionID uuid.UUID
	ParcelID      uuid.UUID
	ValuationType string
	// Action — submit | approve | return | cancel
	Action string
	// Note — ЗААВАЛ (бүх шилжилтэд шаардагдана)
	Note   string
	UserID string
	// ХАВСРАЛТ — "буцаах" (return) үед сонголттой, "цуцлах" (cancel) үед
	// заавал хүлээн авна. Хоосон бол return хавсралтгүй явж болно.
	AttachmentURL  string
	AttachmentName string
	// Баталгаажаагүй үнэлгээний тайлан — ЗӨВХӨН submit үед утгатай.
	DraftReportURL  string
	DraftReportName string
}

// ValuationSubmissionHistory — төлөв шилжилт бүрийн түүх (тайлбартай)
type ValuationSubmissionHistory struct {
	ID            int       `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	ParcelID      string    `json:"parcel_id"`
	ValuationType string    `json:"valuation_type"`
	Action        string    `json:"action"`
	FromStatus    string    `json:"from_status"`
	ToStatus      string    `json:"to_status"`
	Note          string    `json:"note"`
	// Тухайн үйлдлийн хавсралт. Түүх нь ЭХ СУРВАЛЖ: дараагийн буцаалт
	// submission-ийн attachment-ыг дарж бичсэн ч хуучин хавсралт энд үлдэнэ.
	AttachmentURL  string    `json:"attachment_url"`
	AttachmentName string    `json:"attachment_name"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// ValuationSnapshot — баталгаажсан үнэлгээг хүчингүй болгоход хадгалсан дэлгэрэнгүй.
type ValuationSnapshot struct {
	ID            uuid.UUID       `json:"id"`
	AcquisitionID uuid.UUID       `json:"acquisition_id"`
	ParcelID      string          `json:"parcel_id"`
	ValuationType string          `json:"valuation_type"`
	SubmissionID  *uuid.UUID      `json:"submission_id,omitempty"`
	Note          string          `json:"note"`
	CancelledBy   string          `json:"cancelled_by"`
	CancelledAt   time.Time       `json:"cancelled_at"`
	SubmittedBy   string          `json:"submitted_by"`
	SubmittedAt   *time.Time      `json:"submitted_at"`
	ReviewedBy    string          `json:"reviewed_by"`
	ReviewedAt    *time.Time      `json:"reviewed_at"`
	LandValuation json.RawMessage `json:"land_valuation"`
	Assets        json.RawMessage `json:"assets"`
	Compensations json.RawMessage `json:"compensations"`
	TotalAmount   float64         `json:"total_amount"`
	// ReportURL/ReportName — цуцлах үед хүчинтэй байсан ҮНЭЛГЭЭНИЙ ТАЙЛАН.
	// Цуцлалтын дараа тайлан нь нэгж талбарын хавсралтаас хасагддаг
	// (шинэ үнэлгээний тайлантай андуурагдахгүйн тулд) тул зөвхөн эндээс
	// татагдана. Файл нь хадгалалтад ХЭВЭЭР үлдэнэ.
	ReportURL  string `json:"report_url"`
	ReportName string `json:"report_name"`
	// Цуцлах үед хавсралтаас хасагдсан ЭХ ХҮСНЭГТ (Excel) — түүхээс татагдана.
	SourceURL  string `json:"source_url"`
	SourceName string `json:"source_name"`
	// Files — цуцлах мөчид ЭНЭ үнэлгээнд хамаарч байсан файлуудын БҮРЭН
	// жагсаалт (Баталгаажсан тайлан, Тайлан, Үнэлгээний хүснэгт). Зөвхөн
	// түүхийн дэлгэрэнгүйд харагдана. Хуучин snapshot дээр хоосон байх тул
	// хэрэглэгчийн тал ReportURL/SourceURL руу ухарна.
	Files []ValuationSnapshotFile `json:"files"`
}

// ValuationSnapshotFile — архивын НЭГ файлын холбоос.
type ValuationSnapshotFile struct {
	Label string `json:"label"`
	Name  string `json:"name"`
	URL   string `json:"url"`
}

// ParcelEstimatedValueChange — ТӨСӨӨЛЛИЙН үнэлгээний НЭГ өөрчлөлт.
//
// `parcel` дээрх талбарууд нь зөвхөн ОДООГИЙН байдал тул өөрчлөлт бүр
// `parcel_estimated_value_history`-д мөн бичигдэнэ (эс бөгөөс хэн, хэзээ,
// ямар итгэлцүүрээр тооцсон нь дарагдана). Иймд хадгалах үйлдэл нь энэ НЭГ
// бүтцээр дамжина — хоёр газарт өөр өөр аргумент бичихээс сэргийлнэ.
type ParcelEstimatedValueChange struct {
	AcquisitionID uuid.UUID
	ParcelID      uuid.UUID
	// Value nil = үнэлгээг АРИЛГАХ.
	Value *float64
	// ConfidencePercent nil = итгэлцүүр хэрэглээгүй (өмчлөх эрхийн газар).
	ConfidencePercent *float64
	BaseFeePerM2      *float64
	// AreaM2 — тооцоололд хэрэглэсэн нөлөөлөлд өртсөн талбай.
	AreaM2 float64
	// Хавсралт (ЗААВАЛ БИШ) — MinIO-д хадгалагдсаны дараах харьцангуй зам.
	AttachmentURL  string
	AttachmentName string
	UserID         string
}

// ParcelEstimatedValueHistory — төсөөллийн үнэлгээний өөрчлөлтийн бүртгэл.
type ParcelEstimatedValueHistory struct {
	ID            uuid.UUID `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	ParcelID      uuid.UUID `json:"parcel_id"`
	// Value nil = тэр үйлдлээр үнэлгээ АРИЛГАГДСАН.
	Value             *float64  `json:"value"`
	ConfidencePercent *float64  `json:"confidence_percent"`
	BaseFeePerM2      *float64  `json:"base_fee_per_m2"`
	AreaM2            float64   `json:"area_m2"`
	AttachmentURL     string    `json:"attachment_url"`
	AttachmentName    string    `json:"attachment_name"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
}

// ParcelWorkflow — нэгж талбарын ажлын урсгалын шилжилт
type ParcelWorkflow struct {
	ID             int    `json:"id"`
	FromStatusID   *int   `json:"from_status_id"`
	ToStatusID     int    `json:"to_status_id"`
	FromStatusName string `json:"from_status_name"`
	ToStatusName   string `json:"to_status_name"`
	SortOrder      int    `json:"sort_order"`
}

// AcquisitionWorkflow — газар чөлөөлөлтийн ажлын урсгалын шилжилт
type AcquisitionWorkflow struct {
	ID             int    `json:"id"`
	FromStatusID   *int   `json:"from_status_id"`
	ToStatusID     int    `json:"to_status_id"`
	FromStatusName string `json:"from_status_name"`
	ToStatusName   string `json:"to_status_name"`
	SortOrder      int    `json:"sort_order"`
}

// AcquisitionAssignee — чөлөөлөлтэд хуваарилагдсан ажилтан
type AcquisitionAssignee struct {
	UserID         ID        `json:"user_id"`
	UserName       string    `json:"user_name"`
	UserPosition   string    `json:"user_position"`
	AssignedBy     ID        `json:"assigned_by"`
	AssignedByName string    `json:"assigned_by_name"`
	AssignedAt     time.Time `json:"assigned_at"`
}

// LandAcquisition — газар чөлөөлөлтын мэдээлэл (main DB-д хадгалагдана)
type LandAcquisition struct {
	ID                  uuid.UUID
	PlanID              *uuid.UUID
	PlanCode            string
	PlanName            string
	PlanGeomWKT         string
	PlanAreaM2          float64
	GeometryWKT         string
	AreaM2              float64
	Status              int
	StartDate           *time.Time
	EndDate             *time.Time
	AcquisitionName     string
	ImplementingOrg     string
	Reason              string
	ResponsibleOrg      string
	FundingSource       string
	GeneralCategoryID   *int
	GeneralCategoryName string
	SubCategoryID       *int
	SubCategoryName     string
	DecreeNumber        string
	DecreeDate          *time.Time
	// НИТХ-ын тогтоолын дугаар ба тогтоолын хавсралтын бүлэг жагсаалтын
	// дугаар — DecreeNumber (НЗД-ын ЗАХИРАМЖ) -аас ӨӨР баримт, зэрэгцэн
	// хадгалагдана. "" = бүртгээгүй.
	NithDecreeNumber string
	GroupListNumber  string
	CreatedAt        time.Time
	CreatedBy        string
	UpdatedAt        time.Time
	UpdatedBy        string
	AUs              []LandAcquisitionAU
	Parcels          []*Parcel
	ParcelCount      int64
	// FinalParcelCount — ЭЦСИЙН төлөвт (Нөлөөлөгдсөн гарсан / Татгалзсан /
	// Чөлөөлсөн) шилжсэн нэгж талбарын тоо. Чөлөөлөлтийн ЯВЦЫГ үүгээр
	// хэмжинэ: чөлөөлөлт "Баталгаажсан" болохын нөхцөл нь БҮХ талбар эцсийн
	// төлөвт байх явдал (HasNonFinalParcels-тай ижил шалгуур).
	FinalParcelCount int64
	// OverlappingParcelCount — БАЙРШЛААР давхардсан нэгж талбарын тоо
	// (parcel_overlap). 0-ээс их бол чөлөөлөлт "Хээрийн судалгаа" төлөв рүү
	// шилжихгүй — иймд жагсаалт дээр анхааруулга харуулна.
	OverlappingParcelCount int64
	AssignedUsers          []AcquisitionAssignee
	ProfessionalOrgID      *ID
	ProfessionalOrgName    string
	// Төлөвлөгөөний ДЭЛГЭРЭНГҮЙ — plan хүснэгтээс (LEFT JOIN plan_code-оор).
	// ЗӨВХӨН GetByID дээр дүүрнэ: жагсаалтад хэрэггүй бөгөөд мөр тутам join
	// хийх нь илүү. Төлөвлөгөө нь ГУС-аас татагдан plan хүснэгтэд хадгалагдсан
	// байдаг тул дахин ГУС руу хандах шаардлагагүй.
	PlanParcelID     string
	PlanTypeName     string
	PlanGazner       string
	PlanStartDate    *string
	PlanEndDate      *string
	PlanApprovedDate *string
}

// AU2Option — шүүлтүүрийн СУМ/ДҮҮРГИЙН сонголт.
//
// Эх сурвалж нь land_acquisition_au: чөлөөлөлт бүртгэхэд хилээр нь тодорхойлж
// хадгалдаг. Тиймээс жагсаалтад чөлөөлөлт БАЙГАА дүүргүүд л гарна — хоосон
// сонголт харагдахгүй.
type AU2Option struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ProgressPercent — чөлөөлөлтийн явц (0-100). Нэгж талбаргүй бол 0.
func (a *LandAcquisition) ProgressPercent() int {
	if a == nil || a.ParcelCount <= 0 {
		return 0
	}
	pct := int(math.Round(float64(a.FinalParcelCount) * 100 / float64(a.ParcelCount)))
	if pct > 100 {
		return 100
	}
	return pct
}

// LandAcquisitionOption — шүүлтүүрийн dropdown-д зориулсан ХӨНГӨН бүтэц.
//
// ЯАГААД ТУСДАА: үндсэн жагсаалт (List) нь мөр тутамд
// `(SELECT COUNT(*) FROM parcel ...)`, acquisition_category болон
// valuation_org_profile join хийж, 25+ багана буцаадаг. Тайлан/нэгж
// талбарын хуудасны "Төлөвлөгөө"/"Газар чөлөөлөлт" dropdown-ууд түүнийг
// page_size=200-аар дуудаж байсан бөгөөд өгөгдөл нэмэгдэх тутам хүндэрч,
// 30 секундын timeout-д ороод "Серверт холбогдоход алдаа гарлаа" гэж
// хэрэглэгчид харагддаг байв. Dropdown-д зөвхөн эдгээр 4 талбар хэрэгтэй.
type LandAcquisitionOption struct {
	ID              uuid.UUID
	AcquisitionName string
	PlanCode        string
	PlanName        string
}

// LandAcquisitionAU — чөлөөлөлт ↔ захиргааны нэгж холбоос
type LandAcquisitionAU struct {
	LandAcquisitionID uuid.UUID
	AU1Code           string
	AU1Name           string
	AU2Code           string
	AU2Name           string
	AU3Code           string
	AU3Name           string
}

// AU2Overlap — чөлөөлөх хил нэг сум/дүүрэгтэй (au2) давхцсан хэсэг.
type AU2Overlap struct {
	Code string
	Name string
	// Covered — чөлөөлөх хил ЭНЭ сум/дүүрэгт БҮРЭН багтсан эсэх
	// (ST_CoveredBy). Хилийн шугам дээр таарсныг багтсанд тооцно.
	Covered bool
	// AreaM2 — давхцсан хэсгийн талбай (м², geography). Зөвхөн лог/шалтгаан
	// тайлбарлахад — шалгалт нь Covered-ээр хийгдэнэ.
	AreaM2 float64
}

// AU2Coverage — чөлөөлөх хилийг сум/дүүргийн (au2) хилтэй харьцуулсан дүн.
// Чөлөөлөлт нь ЗӨВХӨН нэг сум/дүүрэгт багтах ёстой тул бүртгэхийн өмнө
// хилийг үүгээр шалгана.
type AU2Coverage struct {
	// ReferenceRows — геометртэй au2 мөрийн тоо. 0 бол лавлах өгөгдөл
	// ачаалагдаагүй тул шалгах боломжгүй (scripts/load_admin_units.sh).
	ReferenceRows int
	// Overlaps — давхцсан сум/дүүргүүд (давхцсан талбайгаар буурахаар).
	Overlaps []AU2Overlap
}

// AcquisitionDeleteCheck — чөлөөлөлтийг устгах боломжтой эсэхийг шалгасан дүн
// (нэг DB дуудалтаар цуглуулна).
type AcquisitionDeleteCheck struct {
	// ParcelCount — чөлөөлөлтөд бүртгэлтэй нэгж талбарын НИЙТ тоо.
	ParcelCount int
	// ReleasedParcels — "Чөлөөлсөн" (ParcelStatusReleased) төлөвтэй нэгж
	// талбарын тоо. 0-ээс их бол устгах боломжгүй.
	ReleasedParcels int
	// ParcelStatusCounts — төлөв → тоо. Устгалыг зогсоох шалтгааныг логлож,
	// хэрэглэгчид тайлбарлахад.
	ParcelStatusCounts map[int]int
	// ConfirmedDecisions — баталгаажсан захирамжийн тоо (чөлөөлөлт дээр шууд
	// холбогдсон, эсвэл нэгж талбараар холбогдсон).
	ConfirmedDecisions int
	// DecisionNumbers — баталгаажсан захирамжийн дугаарууд (логт).
	DecisionNumbers []string
}

// GeomContainment — хил (geom) нь өөр хил (container) дотор багтаж байгаа
// эсэхийн дүн.
type GeomContainment struct {
	// Covered — ST_CoveredBy: geom нь container-т БҮРЭН багтсан эсэх. Хилийн
	// шугам дээр давхцаж таарсныг багтсанд тооцно (тиймээс геометрийн
	// нарийвчлалын хүлцэл шаардахгүй).
	Covered bool
	// OutsideM2 — container-ээс ГАДНА гарсан хэсгийн талбай (м²).
	OutsideM2 float64
	// ContainerM2 — container-ийн бүтэн талбай (м²).
	ContainerM2 float64
}

type LandAcquisitionBoundaryHistory struct {
	ID                uuid.UUID
	LandAcquisitionID uuid.UUID
	OldGeometryWKT    string
	NewGeometryWKT    string
	// OldAreaM2 / NewAreaM2 — хадгалсан геометрээс УНШИХ үед бодогддог талбай
	// (public.calculate_area_utm — нэгж талбартай ижил аргачлал). Хүснэгтэд
	// багана нэмээгүй: талбай нь геометрээс бүрэн гардаг тул хадгалах нь
	// давхардал болно (хоёр эх сурвалж зөрөх эрсдэл).
	OldAreaM2 float64
	NewAreaM2 float64
	ChangedBy string
	ChangedAt time.Time
}

// LandAcquisitionFilter — газар чөлөөлөлтийн жагсаалт шүүх болон хуудаслах параметр
type LandAcquisitionFilter struct {
	PlanCode        string
	AcquisitionName string
	Status          int // 0 = шүүлт хийхгүй; 1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED
	// AU2Code — СУМ/ДҮҮРГИЙН код. Чөлөөлөлт бүртгэхэд хилээр нь тодорхойлогдож
	// land_acquisition_au-д хадгалагддаг тул тэр хүснэгтээр шүүнэ. Нэг
	// чөлөөлөлт хэд хэдэн багтай (au3) байж болох ч дүүрэг нь ихэвчлэн нэг.
	AU2Code           string
	AU3Code           string
	StartDateFrom     *time.Time
	StartDateTo       *time.Time
	Years             []int
	GeneralCategoryID int
	SubCategoryID     int
	AssignedUserID    string
	// ProfessionalOrgID — хоосон бол шүүлт байхгүй; утгатай бол тухайн
	// БАЙГУУЛЛАГАД оноогдсон чөлөөлөлтүүд л буцна (үндсэн гүйцэтгэгчээр
	// эсвэл нэгж талбарын хараат бус үнэлгээчнээр). Байгууллагын аль ч
	// ажилтан ижил жагсаалт харна.
	ProfessionalOrgID string
	// Санхүүгийн жагсаалт: хянах эсвэл баталгаажсан үнэлгээг цуцлах боломжтой чөлөөлөлтүүд.
	HasReviewableValuation bool
	// SortBy / SortDir — хүснэгтийн баганаар эрэмбэлэх. SortBy нь repository
	// дахь ЗӨВШӨӨРӨГДСӨН баганын түлхүүр (танигдаагүй бол үндсэн эрэмбэ),
	// SortDir нь "asc" | "desc".
	SortBy   string
	SortDir  string
	Page     int
	PageSize int
}

// ParcelFilter — нэгж талбарын жагсаалт шүүх болон хуудаслах параметр
const (
	RightTypeUse       = 1 // Ашиглах
	RightTypeLease     = 2 // Эзэмших
	RightTypeOwnership = 3 // Өмчлөх
)

// RightTypeName — эрхийн төрлийн монгол нэр. Гадаад системд код дангаараа
// уншигдахгүй тул нэрийг ХАМТ явуулна (frontend-ийн RIGHT_TYPE_LABELS-тэй ижил).
func RightTypeName(rightType int) string {
	switch rightType {
	case RightTypeUse:
		return "Ашиглах"
	case RightTypeLease:
		return "Эзэмших"
	case RightTypeOwnership:
		return "Өмчлөх"
	default:
		return ""
	}
}

type ParcelFilter struct {
	ParcelID  string
	AU1Code   string
	AU2Code   string
	AU3Code   string
	RightType int
	Landuse   string
	StatusID  int
	// ProfessionalOrgID — ҮНЭЛГЭЭНИЙ БАЙГУУЛЛАГЫН хүрээ (хоосон бол шүүлт
	// байхгүй). Утгатай бол нэгж талбар ЗӨВХӨН дараах тохиолдолд харагдана:
	//
	//	чөлөөлөлтийн үндсэн байгууллага (professional_org_id) = тухайн байгууллага
	//	ЭСВЭЛ нэгж талбарын хөндлөнгийн байгууллага (independent_org_id) = тухайн байгууллага
	//
	// Хөндлөнгийн үнэлгээгээр нэг л нэгж талбарт холбогдсон байгууллага
	// чөлөөлөлтийн БУСАД (өөр байгууллагын) нэгж талбарыг харах ёсгүй.
	ProfessionalOrgID string
	// HasOverlap — байршлаар давхардсан эсэхээр шүүнэ.
	//   nil   — шүүлт байхгүй (бүгд)
	//   true  — ЗӨВХӨН давхардалтай
	//   false — ЗӨВХӨН давхардалгүй
	HasOverlap *bool
	// SortBy / SortDir — хүснэгтийн баганаар эрэмбэлэх (LandAcquisitionFilter-тэй ижил).
	SortBy   string
	SortDir  string
	Page     int
	PageSize int
}

// Parcel — нэгж талбар (main DB-д хадгалагдана)
type Parcel struct {
	ID                    uuid.UUID
	ParcelID              string
	OldParcelID           string
	AcquisitionID         uuid.UUID
	ValidFrom             *time.Time
	ValidTill             *time.Time
	AU1Code               string
	AU2Code               string
	AU3Code               string
	GeometryWKT           string
	AcquisitionGeomWKT    string  // чөлөөлөлтийн бүстэй давхцах хэсгийн геометр
	AcquisitionAreaM2     float64 // давхцах хэсгийн талбай (м²)
	RemainingAreaM2       *float64
	RightType             int
	Landuse               string
	LanduseName           string
	AreaM2                float64
	AddressKhashaa        string
	AddressStreet         string
	PropertyNo            string
	CompensationPaid      bool
	DbChanged             string
	ChangedParcelID       string
	StatusID              int
	StatusName            string
	IndependentOrgID      *ID
	IndependentOrgName    string
	SelectedValuationType *string           // санхүү зөвшөөрсөн үндсэн үнэлгээний урсгал (asset/independent/mika)
	ValuationStatuses     map[string]string // урсгал бүрийн илгээх төлөв: valuation_type → draft|submitted|approved|returned
	CashAmount            float64
	LandGrantAmount       float64
	LandGrantCount        int
	// HolderName — ГАЗАР ӨМЧЛӨГЧ/ЭЗЭМШИГЧийн бүтэн нэр (овог + нэр).
	//
	// Нэг нэгж талбар дээр ОЛОН эзэмшигч байж болох тул энэ нь ҮНДСЭН хүсэлт
	// гаргагч (parcel_holder.main_applicant); HolderCount нь нийт тоо.
	// Жагсаалт дээр "нэр +2" гэж харуулахад хоёулаа хэрэгтэй.
	HolderName  string
	HolderCount int
	// HasOverlap — тухайн нэгж талбар ӨӨР нэгж талбартай БАЙРШЛААР давхцаж
	// байгаа эсэх (parcel_overlap). OverlapCount нь давхцаж байгаа талбарын тоо.
	// Давхардал нь нэг газрын нөхөх олговрыг хоёр удаа бодох эрсдэл тул
	// жагсаалт дээр тод харагдах ёстой.
	HasOverlap   bool
	OverlapCount int
	// EstimatedValue — ТӨСӨӨЛЛИЙН үнэлгээ (₮). Мэргэжилтэн/админ ГААРАС
	// оруулна; албан ёсны үнэлгээ (land_valuation, assets) хийгдэхээс өмнөх
	// урьдчилсан тооцоо. nil = оруулаагүй (0-ээс ЯЛГААТАЙ).
	EstimatedValue             *float64
	EstimatedConfidencePercent *float64
	EstimatedBaseFeePerM2      *float64
	EstimatedValueAt           *time.Time
	EstimatedValueBy           string
	CreatedAt                  time.Time
	CreatedBy                  string
}

// ParcelOverlap — НЭГ нэгж талбарын өөр нэгж талбартай байршлын давхардал.
//
// Хоёр чиглэлээр хадгалагддаг тул (A→B, B→A) нэг талбарын давхардлыг
// parcel_id-гаар шүүхэд бүрэн жагсаалт гарна.
type ParcelOverlap struct {
	// OtherParcelID — давхцаж байгаа нэгж талбарын UUID (дэлгэц түүн рүү холбоно)
	OtherParcelID uuid.UUID `json:"other_parcel_uuid"`
	// OtherParcelCode — давхцаж байгаа нэгж талбарын ДУГААР
	OtherParcelCode string `json:"other_parcel_id"`
	// OverlapAreaM2 — давхцах хэсгийн талбай (calculate_area_utm)
	OverlapAreaM2 float64 `json:"overlap_area_m2"`
	// OtherStatusID/OtherStatusName — давхцаж буй талбарын төлөв (аль нь
	// чөлөөлөгдсөнийг шийдвэрлэхэд хэрэгтэй)
	OtherStatusID   int       `json:"other_status_id"`
	OtherStatusName string    `json:"other_status_name"`
	DetectedAt      time.Time `json:"detected_at"`
}

// ParcelDetailInfo — parcel_detail + right_holder холбосон мэдээлэл
type ParcelDetailInfo struct {
	RightType           int
	HolderLastName      string
	HolderName          string
	HolderRegisterNo    string
	HolderPhone         string
	HolderEmail         string
	HolderType          string
	HolderCivilID       string
	AppNo               string
	AppTimestamps       *time.Time
	AppType             string
	AppStatus           string
	DecisionNo          string
	DecisionDate        *time.Time
	ContractNo          string
	ContractDate        *time.Time
	ContractBegin       *time.Time
	ContractEnd         *time.Time
	ContractPropertyNo  string
	ContractStatus      string
	CertificateNo       string
	CertificateDate     *time.Time
	RecordNo            string
	RecordDate          *time.Time
	RecordCertificateNo string
	RecordStatus        string
	// Газрын үнэлгээний мэдээлэл
	ValuationZone  string
	BasePricePerHa *float64
	AuctionCoeff   *float64
	AuctionPrice   *float64
}

// ParcelHolder — нэгж талбарын эзэмшигч. Нэг хүсэлт (app_no) дээр эзэмшигчийн
// төрөл өөр хэд хэдэн хүн байж болох тул parcel_holder-д БҮГД хадгалагдана.
//
// Дундын сервисийн /application/persons хариултаас:
//
//	name       → LastName (овог/эцгийн нэр)
//	first_name → Name     (өөрийн нэр)
//
// ХУУЛИЙН ЭТГЭЭД (компани) үед first_name хоосон ирдэг тул байгууллагын бүтэн
// нэр Name-д бичигдэж, LastName хоосон үлдэнэ.
type ParcelHolder struct {
	ID uuid.UUID
	// MainApplicant — үндсэн хүсэлт гаргагч. parcel_detail.holder_id-д
	// ЭНЭ хүн холбогдоно (хуучин, нэг эзэмшигчтэй дэлгэцүүд тэрийг уншина).
	MainApplicant bool
	LastName      string
	Name          string
	RegisterNo    string
	Phone         string
	Email         string
	// PersonType — эзэмшигчийн төрөл (cl_person_type), жишээ нь
	// "2: Монгол улсын иргэн", "3: Монгол улсын хуулийн этгээд".
	// right_holder.holder_type-д хадгалагдана.
	PersonType string
	AppNo      string
	// HolderRole — ГУС-аас татагдсан эзэмшигч эсвэл гараар бүртгэсэн
	// итгэмжлэгдсэн төлөөлөгч эсэх (ParcelHolderRole* тогтмолууд).
	HolderRole string
	// PaymentRecipient — нөхөн төлбөрийг ХҮЛЭЭН АВАХ эзэмшигч. Нэгж талбар
	// тутамд зөвхөн НЭГ мөр TRUE (DB-д partial unique index-ээр хамгаалагдсан).
	PaymentRecipient bool
	// Note — итгэмжлэгдсэн төлөөлөгчийн тэмдэглэл
	Note string
	// Address — right_holder.address (төлөөлөгч бүртгэхэд хадгалагдана)
	Address string
}

// Нэгж талбарын эзэмшигчийн төрөл.
const (
	// ParcelHolderRoleHolder — ГУС-аас /application/persons-оор татагдсан
	// эзэмшигч. Дахин sync хийхэд бүгд шинэчлэгдэнэ.
	ParcelHolderRoleHolder = "holder"
	// ParcelHolderRoleRepresentative — итгэмжлэгдсэн төлөөлөгч. Гараар
	// бүртгэгддэг тул sync хийхэд ХАДГАЛАГДАЖ үлдэнэ.
	ParcelHolderRoleRepresentative = "representative"
)

// ParcelHolderPersonTypeRepresentative — right_holder.holder_type-д бичигдэх
// утга. ГУС-аас ирдэг "2: Монгол улсын иргэн" зэрэг утгуудтай нэг талбарт
// хамт хадгалагдах тул дэлгэц дээр шууд уншигдахаар монголоор бичнэ.
const ParcelHolderPersonTypeRepresentative = "Итгэмжлэгдсэн төлөөлөгч"

// ParcelGeometryResult — нэгж талбарын геометрээс PostGIS-аар тооцоолсон
// утгууд (чөлөөлөлтийн бүстэй давхцуулсны дараа).
type ParcelGeometryResult struct {
	// ParcelAreaM2 — нэгж талбарын геометрийн бодит талбай (м²)
	ParcelAreaM2 float64
	// AcquisitionAreaM2 — ЧӨЛӨӨЛӨГДӨХ талбайн хэмжээ (давхцах хэсэг, м²)
	AcquisitionAreaM2 float64
	// RemainingAreaM2 — үлдэх талбай (м²)
	RemainingAreaM2 float64
	// HasOverlap — чөлөөлөлтийн бүстэй давхцсан эсэх
	HasOverlap bool
}

// ParcelFull — нэгж талбарын дэлгэрэнгүй мэдээлэл (parcel + parcel_detail + right_holder)
type ParcelFull struct {
	Parcel
	StatusID   int
	StatusName string
	Detail     *ParcelDetailInfo
	// ГУС-аас татагдсан санхүү, эрх зүйн бүртгэлүүд
	Mortgages      []ParcelMortgage
	CourtDecisions []ParcelCourtDecision
	Invoices       []ParcelInvoice
	Monitorings    []ParcelMonitoring
	// Fees — газрын төлбөрийн бодолт бүсээр (ГУС-аас татагдаж хадгалагдсан)
	Fees []ParcelFeeItem
	// BasePriceFactors — суурь үнийн хүчин зүйлийн задаргаа
	BasePriceFactors []ParcelBasePriceFactor
	// Holders — БҮХ эзэмшигч (үндсэн нь эхэнд). Detail нь зөвхөн үндсэнийг агуулна.
	Holders []ParcelHolder
	// Ownerships — УБЕГ-ийн өмчлөлийн бүртгэл (ГУС-аас БИШ, XYP-ээс татагдана)
	Ownerships []ParcelOwnership
	// Overlaps — БАЙРШЛААР давхцаж байгаа бусад нэгж талбарууд. Хоосон биш бол
	// дэлгэц дээр УЛААНААР анхааруулна.
	Overlaps []ParcelOverlap
}

// AcquisitionProgress — газар чөлөөлөлтийн статус өөрчлөлтийн түүх
type AcquisitionProgress struct {
	ID            uuid.UUID
	AcquisitionID uuid.UUID
	FromStatus    int
	ToStatus      int
	Note          string
	ChangedBy     string
	ChangedAt     time.Time
}

// AcquisitionSocioSurvey — нийгэм, эдийн засгийн судалгааны дүн.
//
// Чөлөөлөлтөд НЭГ бүртгэл (acquisition_id unique): дахин оруулахад мөр
// орлогдож, солигдсон файл хадгалалтаас цэвэрлэгдэнэ. Судалгааны PDF нь
// ЗААВАЛ байна (FileURL хоосон бүртгэл байхгүй).
type AcquisitionSocioSurvey struct {
	ID            uuid.UUID
	AcquisitionID uuid.UUID
	// AgreedCount / RejectedCount — зөвшөөрсөн / татгалзсан иргэн, хуулийн
	// этгээдийн тоо.
	AgreedCount   int
	RejectedCount int
	Note          string
	FileURL       string
	FileName      string
	SizeBytes     int64
	CreatedAt     time.Time
	CreatedBy     string
	UpdatedAt     time.Time
	UpdatedBy     string
}

// ParcelNoticeEmail — урьдчилан мэдэгдэх хуудсыг имэйлээр илгээсэн НЭГ бичлэг.
//
// Нэгж талбар дээр илгээлт олон удаа гарч болох тул түүх хэлбэрээр
// хуримтлагдана. Status нь "sent" эсвэл "failed" (алдаатай оролдлогыг ч
// бүртгэнэ — илгээгдээгүйг илгээсэн гэж эндүүрэхээс сэргийлнэ).
type ParcelNoticeEmail struct {
	ID            uuid.UUID
	ParcelID      uuid.UUID
	AcquisitionID uuid.UUID
	ToEmail       string
	Subject       string
	FileURL       string
	FileName      string
	SizeBytes     int64
	Status        string
	ErrorMessage  string
	SentAt        time.Time
	SentBy        string
}

const (
	NoticeEmailStatusSent   = "sent"
	NoticeEmailStatusFailed = "failed"
)

// ParcelEMongoliaNotice — мэдэгдэх хуудсыг E-Mongolia-гаар хүргүүлсэн НЭГ
// бичлэг (төлөвлөгөөний 2.8.5).
//
// Хүлээн авагчийг РЕГИСТРИЙН дугаараар тодорхойлно (имэйл биш). ExternalRef нь
// гадаад системийн лавлагааны дугаар — мок үед "MOCK-" угтвартай.
type ParcelEMongoliaNotice struct {
	ID            uuid.UUID
	ParcelID      uuid.UUID
	AcquisitionID uuid.UUID
	RegisterNo    string
	RecipientName string
	Subject       string
	FileURL       string
	FileName      string
	SizeBytes     int64
	Status        string
	ExternalRef   string
	ErrorMessage  string
	SentAt        time.Time
	SentBy        string
}

const (
	EMongoliaStatusSent = "sent"
	// EMongoliaStatusMocked — гадаад систем ХОЛБОГДООГҮЙ (түр мок). Иргэнд
	// хүрсэн гэж ТООЦОХГҮЙ.
	EMongoliaStatusMocked = "mocked"
	EMongoliaStatusFailed = "failed"
)

// Document — үндсэн баримт бичиг (URL-аар хадгалагдана)
type Document struct {
	ID             uuid.UUID
	Name           string
	FileType       string
	FileURL        string
	SizeBytes      int64
	Note           string
	UploadedBy     string
	UploadedAt     time.Time
	DocumentTypeID int
	// SourceDocID — эх системийн баримтын дугаар (ct_document.id). Гараар
	// оруулсан хавсралт дээр хоосон. Дахин татахад давхардлыг үүгээр таньдаг.
	SourceDocID string
	SourceFtpID string
	// SourceDocCode — ГУС-ийн баримтын үүргийн код (ж: "1"=Өргөдөл,
	// "11"=Кадастрын зураг). Хавсралтыг бүлэглэхэд ашиглана. Гараар оруулсан
	// дээр хоосон. document_type_id-тай ХАМААРАЛГҮЙ — дугаарлалт нь өөр.
	SourceDocCode string
}

// DocumentFile — татаж/харуулахаар нээсэн файлын урсгал. Дуудагч Content-ийг
// ХААХ хариуцлагатай. SizeBytes нь мэдэгдэхгүй үед -1.
type DocumentFile struct {
	Name        string
	ContentType string
	SizeBytes   int64
	Content     io.ReadCloser
}

// SourceDocument — дундын сервисийн /application/documents-ийн нэг мөр:
// хүсэлтэд хавсаргасан эх системийн баримт.
type SourceDocument struct {
	AppNo   string
	DocName string
	DocCode string
	DocID   string
	FileURL string
	FtpID   string
}

// ParcelDocumentSyncResult — хавсралт татсаны үр дүн. Татах нь ЗӨВХӨН НЭМЭХ
// үйлдэл тул хадгалагдсан хавсралт устгагдахгүй.
type ParcelDocumentSyncResult struct {
	// Found — эх системд олдсон баримтын тоо
	Found int
	// Saved — ШИНЭЭР бүртгэгдсэн баримтын тоо
	Saved int
	// Skipped — өмнө нь татагдсан тул алгасагдсан баримтын тоо
	Skipped int
	// Documents — нэгж талбарын одоогийн БҮХ хавсралт (шинэчилсний дараах)
	Documents []*Document
}

// GlobalParcel — parcel joined with acquisition info
type GlobalParcel struct {
	Parcel
	Status            int
	AcquisitionName   string
	PlanCode          string
	AcquisitionStatus int
	StartDate         *time.Time
	EndDate           *time.Time
	CashAmount        float64
	LandGrantAmount   float64
	LandGrantCount    int
}

// GlobalParcelFilter — глобал нэгж талбарын жагсаалт шүүх болон хуудаслах параметр
type GlobalParcelFilter struct {
	ParcelID        string
	RightType       int
	Landuse         string
	AU3Code         string
	AcquisitionName string
	AcquisitionID   string
	PlanCode        string
	Status          int
	StatusID        int
	UnlinkedOnly    bool
	Years           []int
	// SortBy / SortDir — хүснэгтийн баганаар эрэмбэлэх (LandAcquisitionFilter-тэй ижил).
	SortBy   string
	SortDir  string
	Page     int
	PageSize int
}

// ReportParcelFilter — тайлангийн нэгж талбар шүүх параметр
type ReportParcelFilter struct {
	// ParcelID — нэгж талбарын ДУГААР (код). Гадаад API-д нэг талбарын
	// мэдээллийг дугаараар шүүхэд хэрэглэгдэнэ; хоосон бол шүүлт хийхгүй.
	ParcelID string
	// PlanParcelID — ЧӨЛӨӨЛӨЛТИЙН төлөвлөгөөний нэгж талбарын дугаар
	// (plan.parcel_id — чөлөөлөлтийг үүгээр үүсгэдэг). Тухайн төлөвлөгөөний
	// нэгж талбараар үүссэн БҮХ чөлөөлөлтийн мөрүүд орно.
	PlanParcelID  string
	AcquisitionID string
	// PlanCode — ХЭСЭГЧИЛСЭН таарал (тайлангийн хайлт). Гадаад API нь ЯГ
	// таарлыг шаарддаг тул PlanCodeExact-ыг хэрэглэнэ.
	PlanCode          string
	PlanCodeExact     string
	AcquisitionName   string
	AU3Code           string
	RightType         int
	Landuse           string
	Years             []int  // start_date-ийн он (олон утга дэмжинэ)
	CompensationType  string // 'cash' | 'land_grant' | '' (бүгд)
	GeneralCategoryID int
	SubCategoryID     int
	Page              int
	PageSize          int
}

// ReportParcelRow — тайлангийн нэг мөрийн бүх мэдээлэл (нэг JOIN query-аас)
type ReportParcelRow struct {
	// Нэгж талбар
	ParcelID          string
	AreaM2            float64
	AcquisitionAreaM2 float64
	RemainingAreaM2   float64
	RightType         int
	DbChanged         string
	ChangedParcelID   string
	Status            int
	StatusName        string
	// StatusDate — сүүлд статус өөрчлөгдсөн огноо (parcel_status_history-ээс);
	// тайлангийн жагсаалтын дээд хэсэгт оноор нь бүлэглэхэд ашиглана.
	StatusDate *time.Time
	// Чөлөөлөлт
	AcquisitionID       uuid.UUID
	AcquisitionName     string
	PlanCode            string
	GeneralCategoryName string
	SubCategoryName     string
	DecreeNumber        string
	DecreeDate          *time.Time
	// Эзэмшигч
	HolderLastName   string
	HolderName       string
	HolderRegisterNo string
	// Нөхөн төлбөр (нэгтгэсэн)
	LandComp      float64
	RealStateComp float64
	PropertyComp  float64
}

// ReportStatusStat — тайлангийн шүүлтэд тохирох нэгж талбаруудыг статусаар бүлэглэсэн тоо
type ReportStatusStat struct {
	Status     int
	StatusName string
	Count      int
}

// ReportYearStat — тайлангийн шүүлтэд тохирох чөлөөлөлтүүдийг (сүүлийн статус
// өөрчлөлтийн оноор) бүлэглэсэн тоо
type ReportYearStat struct {
	Year  int
	Count int
}

// ReportSummary — тайлангийн жагсаалтын дээд хэсгийн статистикийн карт (бүх
// хуудаслагдсан үр дүнгээр, нэг л дуудлагаар тооцоологдоно)
type ReportSummary struct {
	AcquisitionCount int
	ParcelCount      int
	// TotalAreaM2 — нөлөөлөлд өртсөн (acquisition_area_m2) талбайн нийлбэр.
	TotalAreaM2 float64
	// TotalParcelAreaM2 — нэгж талбаруудын үндсэн (бүтэн, acquisition-аас өмнөх) талбайн
	// хэмжээний нийлбэр (p.area_m2) — "Нийт чөлөөлөлт" картад харуулахад ашиглана.
	TotalParcelAreaM2 float64
	TotalCompensation float64
	// Нөхөх олговрын төрлөөр задаргаа (ListReportParcels-ийн land_comp/real_state_comp/
	// property_comp-той адил ангилал) — картын доод хэсэгт харуулахад ашиглана.
	LandCompensation      float64
	RealStateCompensation float64
	PropertyCompensation  float64
	// OtherCompensation — target_type='asset' боловч холбогдох asset нь устсан/
	// төрөлгүй зэрэг шалтгаанаар дээрх 3 ангилалд ороогүй үлдэгдэл (TotalCompensation -
	// Land - RealState - Property). Задаргааны нийлбэр нь нийт дүнтэй яг тэнцүү байхын
	// баталгаа (тайлангийн картаас "зөрж байна" гэсэн тайлагдсан зөрүүг арилгасан).
	OtherCompensation float64
	StatusBreakdown   []ReportStatusStat
	YearBreakdown     []ReportYearStat
}

// AssetFilter — хөрөнгийн жагсаалт шүүх болон хуудаслах параметр
type AssetFilter struct {
	ParcelID      string
	ValuationType string // хоосон = бүх урсгал
	Page          int
	PageSize      int
}

// Asset — хөрөнгийн үндсэн мэдээлэл
type Asset struct {
	ID            uuid.UUID `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	ParcelID      string    `json:"parcel_id"`
	AssetNumber   string    `json:"asset_number"`
	AssetType     string    `json:"asset_type"`
	ValuationType string    `json:"valuation_type"`
	AssetName     string    `json:"asset_name"`
	FloorCount    int       `json:"floor_count"`
	AreaM2        float64   `json:"area_m2"`
	OwnerName     string    `json:"owner_name"`
	Address       string    `json:"address"`
	Notes         string    `json:"notes"`
	Unit          string    `json:"unit"`
	Capacity      string    `json:"capacity"`
	Description   string    `json:"description"`
	UnitPrice     float64   `json:"unit_price"`    // нэгж үнэ (Бусад хөрөнгө хүснэгтээс)
	PhotoPdfURL   string    `json:"photo_pdf_url"` // зургуудыг нэгтгэсэн PDF
	PhotoPdfName  string    `json:"photo_pdf_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AssetSpecType — барилгын үзүүлэлтийн төрөл (asset_spec_type хүснэгт)
type AssetSpecType struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// AssetCalcType — тооцооллын мэдээллийн төрөл (asset_calc_type хүснэгт)
type AssetCalcType struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	DefaultUnit string `json:"default_unit"`
	SortOrder   int    `json:"sort_order"`
	Grp         string `json:"grp"` // бүлгийн нэр (Итгэлцүүр г.м) — байхгүй бол хоосон
}

// AssetSpec — барилгын нэг үзүүлэлтийн утга (asset_spec хүснэгт)
type AssetSpec struct {
	ID         uuid.UUID `json:"id"`
	AssetID    uuid.UUID `json:"asset_id"`
	SpecTypeID int       `json:"spec_type_id"`
	SpecCode   string    `json:"spec_code"`
	SpecName   string    `json:"spec_name"`
	Value      string    `json:"value"`
}

// AssetCalculation — барилгын нэг тооцооллын утга (asset_calculation хүснэгт)
type AssetCalculation struct {
	ID         uuid.UUID `json:"id"`
	AssetID    uuid.UUID `json:"asset_id"`
	CalcTypeID int       `json:"calc_type_id"`
	CalcCode   string    `json:"calc_code"`
	CalcName   string    `json:"calc_name"`
	CalcGroup  string    `json:"calc_group"` // бүлгийн нэр (asset_calc_type.grp-ээс)
	Unit       string    `json:"unit"`
	Value      float64   `json:"value"`
}

// LandValuation — газрын үнэлгээ (мэргэжлийн байгуулгын үнэлгээчин оруулна)
type LandValuation struct {
	ID             uuid.UUID `json:"id"`
	AcquisitionID  uuid.UUID `json:"acquisition_id"`
	ParcelID       string    `json:"parcel_id"`
	ValuationType  string    `json:"valuation_type"`
	LandAreaM2     float64   `json:"land_area_m2"`
	BasePricePerM2 float64   `json:"base_price_per_m2"`
	// Excel импортоор ирдэг нэмэлт мэдээлэл (үнэлгээний тайлангийн метадата)
	OwnershipCertNo     string    `json:"ownership_cert_no"`      // өмчлөх эрхийн гэрчилгээ
	AppraiserOrgName    string    `json:"appraiser_org_name"`     // үнэлгээ хийсэн байгууллага
	AppraiserRegNo      string    `json:"appraiser_reg_no"`       // регистрийн дугаар
	AppraiserStateRegNo string    `json:"appraiser_state_reg_no"` // улсын бүртгэлийн дугаар
	AppraiserDirector   string    `json:"appraiser_director"`     // захирал
	AppraiserLicense    string    `json:"appraiser_license"`      // тусгай зөвшөөрөл
	AppraiserAddress    string    `json:"appraiser_address"`
	AppraiserContact    string    `json:"appraiser_contact"`
	SourceFileName      string    `json:"source_file_name"` // эх Excel файлын нэр
	SourceFileHash      string    `json:"source_file_hash"` // давхардал шалгах SHA-256
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ValuationSectionNote — үнэлгээний хүснэгт (3.1–5.1) бүрийн ТАЙЛБАР.
// Excel-ийн "ТАЙЛБАР:" мөрөөс импортлогдоно, дэлгэц дээр гараас засагдана.
// SectionKey нь frontend-ийн ValuationSectionKey-тэй ижил утгатай (land_valuation,
// building_cost, other_assets, …) — бүртгэл болгож хатуулаагүй тул загварт
// шинэ хүснэгт нэмэгдэхэд миграц шаардахгүй.
type ValuationSectionNote struct {
	ID            uuid.UUID `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	ParcelID      string    `json:"parcel_id"`
	ValuationType string    `json:"valuation_type"`
	SectionKey    string    `json:"section_key"`
	Note          string    `json:"note"`
	// Excel дэх хүснэгтийн таних мэдээлэл — дэлгэц дугаар/нэр/дарааллыг үүнээс авна.
	SectionNo  string    `json:"section_no"`  // "3.3"
	TableLabel string    `json:"table_label"` // "Хүснэгт-3"
	Title      string    `json:"title"`       // "Газрын үнэлгээ"
	SortOrder  int       `json:"sort_order"`  // файл дахь дараалал
	UpdatedBy  string    `json:"updated_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ValuationSectionInput — импортын үед нэг хүснэгтийн тайлбар + таних мэдээлэл.
type ValuationSectionInput struct {
	Key        string
	Note       string
	SectionNo  string
	TableLabel string
	Title      string
	SortOrder  int
}

// ── Excel-ээс бөөнөөр оруулах (нэг транзакц, олон API дуудлагыг орлоно) ──

// ValuationImportAsset — импортод орох нэг хөрөнгө + түүний spec/calc/олговор
type ValuationImportAsset struct {
	AssetNumber        string
	AssetType          string
	AssetName          string
	AreaM2             float64
	FloorCount         int // "Давхрын тоо" үзүүлэлтээс (барилгын тодорхойлолтын хүснэгт)
	Unit               string
	Capacity           string
	Description        string
	OwnerName          string
	Notes              string
	UnitPrice          float64
	Specs              []AssetSpec        // зөвхөн SpecTypeID + Value ашиглана
	Calculations       []AssetCalculation // зөвхөн CalcTypeID + Unit + Value ашиглана
	CompensationAmount float64            // >0 бол cash нөхөн олговор үүсгэнэ
}

// ValuationImport — бүхэл нэгж талбарын үнэлгээг нэг дор оруулах хүсэлт
type ValuationImport struct {
	ParcelID      string
	ValuationType string // asset/independent/mika — импорт орох урсгал
	Replace       bool   // true бол уг талбар+урсгалын хуучин хөрөнгө/олговрыг эхлээд устгана
	Land          LandValuation
	Assets        []ValuationImportAsset
	// SectionNotes — хүснэгт бүрийн ТАЙЛБАР (section_key → бичвэр). Хоосон бол
	// хуучин тайлбарыг ХЭВЭЭР үлдээнэ (импорт нь тайлбарыг санамсаргүй арчихгүй).
	SectionNotes map[string]string
	// Sections — хүснэгтийн тайлбар + дугаар/нэр/дараалал. SectionNotes-оос
	// ДЭЛГЭРЭНГҮЙ хувилбар нь; хоёул ирвэл Sections давамгайлна.
	Sections  []ValuationSectionInput
	CreatedBy string
}

// ValuationImportResult — импортын дүн
type ValuationImportResult struct {
	DeletedAssets int `json:"deleted_assets"`
	DeletedComps  int `json:"deleted_comps"`
	CreatedAssets int `json:"created_assets"`
	CreatedComps  int `json:"created_comps"`
	// Replace үед устгагдсан мөрүүдийн файлууд — service хадгалалтаас цэвэрлэнэ
	OrphanedFileURLs []string `json:"-"`
}

// CompensationGrant — газар олгох олговрын бичлэг
type CompensationGrant struct {
	ID             uuid.UUID  `json:"id"`
	AcquisitionID  uuid.UUID  `json:"acquisition_id"`
	CompensationID uuid.UUID  `json:"compensation_id"`
	Amount         float64    `json:"amount"`
	GrantDate      *time.Time `json:"grant_date"`
	Note           string     `json:"note"`
	LandAreaM2     float64    `json:"land_area_m2"`
	LandPrice      float64    `json:"land_price"`
	LandLocation   string     `json:"land_location"`
	LandPurpose    string     `json:"land_purpose"`
	LandUseType    string     `json:"land_use_type"`
	ParcelNumber   string     `json:"parcel_number"`
	DecreeNumber   string     `json:"decree_number"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      string     `json:"created_by"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Compensation — нөхөх олговрын бичлэг
type Compensation struct {
	ID               uuid.UUID          `json:"id"`
	AcquisitionID    uuid.UUID          `json:"acquisition_id"`
	TargetType       string             `json:"target_type"`    // 'parcel' | 'asset'
	ValuationType    string             `json:"valuation_type"` // asset/independent/mika урсгал
	ParcelID         string             `json:"parcel_id"`
	AssetID          *uuid.UUID         `json:"asset_id"`
	CompensationType string             `json:"compensation_type"`
	CoveragePercent  float64            `json:"coverage_percent"`
	Amount           float64            `json:"amount"`
	CompensationDate *time.Time         `json:"compensation_date"`
	Note             string             `json:"note"`
	Grant            *CompensationGrant `json:"grant,omitempty"`
	// Хянах workflow
	Status     string     `json:"status"`      // pending | approved | rejected
	ReviewNote string     `json:"review_note"` // татгалзсан шалтгаан
	ReviewedBy string     `json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	// Үнэлгээний тайлан (санхүүгийн ажилтан зөвшөөрсний дараа мэргэжлийн байгууллага оруулна)
	ValuationReportURL  string    `json:"valuation_report_url"`
	ValuationReportName string    `json:"valuation_report_name"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           string    `json:"created_by"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// GlobalCompensation — нөхөн төлбөр + acquisition_name + парцелийн эзэмшигчийн мэдээлэл
type GlobalCompensation struct {
	Compensation
	AcquisitionName  string `json:"acquisition_name"`
	HolderName       string `json:"holder_name"`
	HolderLastName   string `json:"holder_last_name"`
	HolderRegisterNo string `json:"holder_register_no"`
	HolderPhone      string `json:"holder_phone"`
}

// GlobalCompensationFilter — глобал нөхөн төлбөрийн жагсаалт шүүлт
type GlobalCompensationFilter struct {
	Status        string // pending | approved | rejected | ""
	Search        string // parcel_id эсвэл holder_name
	AcquisitionID string
	Page          int
	PageSize      int
}

// CompensationHistory — татгалзсан үнэлгээний түүх
type CompensationHistory struct {
	ID               uuid.UUID  `json:"id"`
	CompensationID   uuid.UUID  `json:"compensation_id"`
	AcquisitionID    uuid.UUID  `json:"acquisition_id"`
	TargetType       string     `json:"target_type"`
	ParcelID         string     `json:"parcel_id"`
	AssetID          *uuid.UUID `json:"asset_id"`
	CompensationType string     `json:"compensation_type"`
	CoveragePercent  float64    `json:"coverage_percent"`
	Amount           float64    `json:"amount"`
	CompensationDate *time.Time `json:"compensation_date"`
	Note             string     `json:"note"`
	Status           string     `json:"status"`
	ReviewNote       string     `json:"review_note"`
	ReviewedBy       string     `json:"reviewed_by"`
	ReviewedAt       *time.Time `json:"reviewed_at"`
	ArchivedAt       time.Time  `json:"archived_at"`
}

// ParcelPayment — нэгж талбарт хийсэн төлбөрийн бичлэг
type ParcelPayment struct {
	ID        uuid.UUID
	ParcelID  uuid.UUID
	Amount    float64
	Currency  string
	PaidAt    *time.Time
	Note      string
	CreatedAt time.Time
	CreatedBy string
}

// FundingSource — санхүүжилтийн эх үүсвэр
type FundingSource struct {
	ID               uuid.UUID `json:"id"`
	AcquisitionID    uuid.UUID `json:"acquisition_id"`
	OrganizationName string    `json:"organization_name"`
	SourceType       string    `json:"source_type"`
	Amount           *float64  `json:"amount,omitempty"`
	Currency         string    `json:"currency"`
	Note             string    `json:"note"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
}

// DroneImage — чөлөөлөлтийн дроны ортофото (.tif).
//
// FileURL нь файлын систем (MinIO) дээрх object-ийн зам — зургийн ЭХ СУРВАЛЖ.
//
// FileName нь файлын системд хадгалагдсан нэр. Зураг тус бүр GeoServer дээр
// ӨӨРИЙН давхаргатай бөгөөд давхаргын нэр нь эндээс тооцогдоно.
type DroneImage struct {
	ID            uuid.UUID `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	FileName      string    `json:"file_name"`
	FileURL       string    `json:"file_url"`
	OriginalName  string    `json:"original_name"`
	ContentType   string    `json:"content_type"`
	SizeBytes     int64     `json:"size_bytes"`
	MinX          *float64  `json:"min_x,omitempty"`
	MinY          *float64  `json:"min_y,omitempty"`
	MaxX          *float64  `json:"max_x,omitempty"`
	MaxY          *float64  `json:"max_y,omitempty"`
	Published     bool      `json:"published"`
	UploadedAt    time.Time `json:"uploaded_at"`
	UploadedBy    *ID       `json:"uploaded_by,omitempty"`
}

// AuthorizedRepresentative — нэгж талбарын итгэмжлэгдсэн төлөөлөгч
type AuthorizedRepresentative struct {
	ID            uuid.UUID `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	ParcelID      string    `json:"parcel_id"`
	LastName      string    `json:"last_name"`
	FirstName     string    `json:"first_name"`
	RegisterNo    string    `json:"register_no"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	Address       string    `json:"address"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     string    `json:"created_by"`
}

// DashboardFilter — дашбоардын шүүлтүүрийн параметр
type DashboardFilter struct {
	AcquisitionID string
	PlanCode      string
	// AU2Code — сум/дүүргийн код (land_acquisition_au-аар шүүнэ)
	AU2Code              string
	Years                []int
	Status               int
	GeneralCategoryID    int
	SubCategoryID        int
	AssignedUserID       string
	ProfessionalOrgID    string
	FilterByParcelStatus bool
	ParcelStatusID       int
}

// ParcelStatusStat — нэгж талбарын төлөвийн статистик
type ParcelStatusStat struct {
	StatusID int     `json:"status_id"`
	Name     string  `json:"name"`
	Count    int64   `json:"count"`
	AreaM2   float64 `json:"area_m2"`
}

// TimelinePoint — огноогоор нэгж талбарын тоо
type TimelinePoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ValuationStatusStat struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type ValuationTypeStat struct {
	ValuationType string  `json:"valuation_type"`
	Count         int64   `json:"count"`
	Amount        float64 `json:"amount"`
}

// DashboardSummary — дашбоардын нийлмэл хариу (бүх тооцоо DB-д хийгдсэн)
type DashboardSummary struct {
	Acquisitions      []*LandAcquisition    `json:"acquisitions"`
	ParcelStatuses    []*ParcelStatus       `json:"parcel_statuses"`
	TotalParcels      int64                 `json:"total_parcels"`
	FreedParcels      int64                 `json:"freed_parcels"`
	FreedAreaM2       float64               `json:"freed_area_m2"`
	PlanAreaM2        float64               `json:"plan_area_m2"`
	TotalOrders       int64                 `json:"total_orders"`
	TotalCompensation float64               `json:"total_compensation"`
	StatusBreakdown   []ParcelStatusStat    `json:"status_breakdown"`
	Timeline          []TimelinePoint       `json:"timeline"`
	ValuationStatuses []ValuationStatusStat `json:"valuation_statuses"`
	ValuationTypes    []ValuationTypeStat   `json:"valuation_types"`
	FilteredAU1Codes  []string              `json:"filtered_au1_codes"`
	FilteredAU2Codes  []string              `json:"filtered_au2_codes"`
	FilteredAU3Codes  []string              `json:"filtered_au3_codes"`
}
