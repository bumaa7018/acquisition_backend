package domain

import (
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
	ID          uuid.UUID
	PlanCode    string
	Name        string
	BoundaryWKT string // WKT текст хэлбэрт
	AreaM2      float64
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SourceParcel — нэгж талбарын эх мэдээлэл (auth DB-д хадгалагдана)
type SourceParcel struct {
	ID               uuid.UUID
	ParcelID         string
	PlanCode         string
	GeometryWKT      string
	AU1Code          string
	AU2Code          string
	AU3Code          string
	RightType        int
	Landuse          string
	AreaM2           float64
	HolderLastName   string
	HolderName       string
	HolderRegisterNo string
	HolderPhone      string
	HolderEmail      string
	HolderType       string
	HolderCivilID    string
	AppNo            string
	DecisionNo       string
	DecisionDate     *time.Time
	ContractNo       string
	ContractDate     *time.Time
	CertificateNo    string
	CertificateDate  *time.Time
	CreatedAt        time.Time
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
	ValuationStatusVoided    = "voided"    // Хүчингүй — өөр урсгал баталгаажсан үед автоматаар (эцсийн)
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
	UserID         uuid.UUID `json:"user_id"`
	UserName       string    `json:"user_name"`
	UserPosition   string    `json:"user_position"`
	AssignedBy     uuid.UUID `json:"assigned_by"`
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
	ProfessionalOrgID   *uuid.UUID
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
	PlanCode              string
	AcquisitionName       string
	Status                int // 0 = шүүлт хийхгүй; 1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED
	AU3Code               string
	StartDateFrom         *time.Time
	StartDateTo           *time.Time
	Years                 []int
	GeneralCategoryID     int
	SubCategoryID         int
	AssignedUserID        string
	ProfessionalOrgUserID string // empty = no filter; set = only acquisitions accessible by this user
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
	AreaM2                float64
	CompensationPaid      bool
	DbChanged             bool
	ChangedParcelID       string
	StatusID              int
	StatusName            string
	IndependentOrgID      *uuid.UUID
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
	RightType        int
	HolderLastName   string
	HolderName       string
	HolderRegisterNo string
	HolderPhone      string
	HolderEmail      string
	HolderType       string
	HolderCivilID    string
	AppNo            string
	AppTimestamps    *time.Time
	DecisionNo       string
	DecisionDate     *time.Time
	ContractNo       string
	ContractDate     *time.Time
	CertificateNo    string
	CertificateDate  *time.Time
	// Газрын үнэлгээний мэдээлэл
	ValuationZone  string
	BasePricePerHa *float64
	AuctionCoeff   *float64
	AuctionPrice   *float64
}

// ParcelFull — нэгж талбарын дэлгэрэнгүй мэдээлэл (parcel + parcel_detail + right_holder)
type ParcelFull struct {
	Parcel
	StatusID   int
	StatusName string
	Detail     *ParcelDetailInfo
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
	AcquisitionID     string
	PlanCode          string
	Years             []int
	GeneralCategoryID int
	SubCategoryID     int
	AssignedUserID    string
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

// DashboardSummary — дашбоардын нийлмэл хариу (бүх тооцоо DB-д хийгдсэн)
type DashboardSummary struct {
	Acquisitions      []*LandAcquisition `json:"acquisitions"`
	ParcelStatuses    []*ParcelStatus    `json:"parcel_statuses"`
	TotalParcels      int64              `json:"total_parcels"`
	FreedParcels      int64              `json:"freed_parcels"`
	FreedAreaM2       float64            `json:"freed_area_m2"`
	PlanAreaM2        float64            `json:"plan_area_m2"`
	TotalOrders       int64              `json:"total_orders"`
	TotalCompensation float64            `json:"total_compensation"`
	StatusBreakdown   []ParcelStatusStat `json:"status_breakdown"`
	Timeline          []TimelinePoint    `json:"timeline"`
	FilteredAU1Codes  []string           `json:"filtered_au1_codes"`
	FilteredAU2Codes  []string           `json:"filtered_au2_codes"`
	FilteredAU3Codes  []string           `json:"filtered_au3_codes"`
}
