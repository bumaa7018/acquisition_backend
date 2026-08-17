package domain

import (
	"io"
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

// ParcelBasePrice — газрын СУУРЬ ҮНЭ (ГУС data_cama).
type ParcelBasePrice struct {
	BasePrice     float64 `json:"base_price"`
	BasePriceM2   float64 `json:"base_price_m2"`
	CalculateYear int     `json:"calculate_year"`
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
}

// ParcelStatus — нэгж талбарын төлөв (lookup)
type ParcelStatus struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
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
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
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
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
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
	CreatedAt           time.Time
	CreatedBy           string
	UpdatedAt           time.Time
	UpdatedBy           string
	AUs                 []LandAcquisitionAU
	Parcels             []*Parcel
	ParcelCount         int64
	AssignedUsers       []AcquisitionAssignee
	ProfessionalOrgID   *ID
	ProfessionalOrgName string
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

type LandAcquisitionBoundaryHistory struct {
	ID                uuid.UUID
	LandAcquisitionID uuid.UUID
	OldGeometryWKT    string
	NewGeometryWKT    string
	ChangedBy         string
	ChangedAt         time.Time
}

// LandAcquisitionFilter — газар чөлөөлөлтийн жагсаалт шүүх болон хуудаслах параметр
type LandAcquisitionFilter struct {
	PlanCode          string
	AcquisitionName   string
	Status            int // 0 = шүүлт хийхгүй; 1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED
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
	// true = зөвхөн "Илгээсэн" (submitted) төлөвтэй нөхөх олговрын үнэлгээтэй
	// нэгж талбар бүхий чөлөөлөлтүүд (санхүүгийн мэргэжилтний жагсаалт)
	HasSubmittedValuation bool
	Page                  int
	PageSize              int
}

// ParcelFilter — нэгж талбарын жагсаалт шүүх болон хуудаслах параметр
const (
	RightTypeUse       = 1 // Ашиглах
	RightTypeLease     = 2 // Эзэмших
	RightTypeOwnership = 3 // Өмчлөх
)

type ParcelFilter struct {
	ParcelID  string
	AU1Code   string
	AU2Code   string
	AU3Code   string
	RightType int
	Landuse   string
	StatusID  int
	Page      int
	PageSize  int
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
	DbChanged             bool
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
	CreatedAt             time.Time
	CreatedBy             string
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
	// Holders — БҮХ эзэмшигч (үндсэн нь эхэнд). Detail нь зөвхөн үндсэнийг агуулна.
	Holders []ParcelHolder
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
	Page            int
	PageSize        int
}

// ReportParcelFilter — тайлангийн нэгж талбар шүүх параметр
type ReportParcelFilter struct {
	AcquisitionID     string
	PlanCode          string
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
	DbChanged         bool
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

// ── Excel-ээс бөөнөөр оруулах (нэг транзакц, олон API дуудлагыг орлоно) ──

// ValuationImportAsset — импортод орох нэг хөрөнгө + түүний spec/calc/олговор
type ValuationImportAsset struct {
	AssetNumber        string
	AssetType          string
	AssetName          string
	AreaM2             float64
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
	CreatedBy     string
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
	AcquisitionID        string
	PlanCode             string
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
