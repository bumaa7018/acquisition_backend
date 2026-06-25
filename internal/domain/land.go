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

// ConstructionType — бүтээн байгуулалтын төрөл (lookup)
type ConstructionType struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// ParcelStatus — нэгж талбарын төлөв (lookup)
type ParcelStatus struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// LandAcquisition — газар чөлөөлөлтын мэдээлэл (main DB-д хадгалагдана)
type LandAcquisition struct {
	ID                   uuid.UUID
	PlanID               *uuid.UUID
	PlanCode             string
	PlanName             string
	PlanGeomWKT          string
	PlanAreaM2           float64
	GeometryWKT          string
	AreaM2               float64
	Status               int
	StartDate            *time.Time
	EndDate              *time.Time
	AcquisitionName      string
	ImplementingOrg      string
	Reason               string
	ResponsibleOrg       string
	FundingSource        string
	ConstructionTypeID   *int
	ConstructionTypeName string
	DecreeNumber         string
	DecreeDate           *time.Time
	CreatedAt            time.Time
	CreatedBy            string
	UpdatedAt            time.Time
	UpdatedBy            string
	AUs                  []LandAcquisitionAU
	Parcels              []*Parcel
	ParcelCount          int64
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

// LandAcquisitionFilter — газар чөлөөлөлтийн жагсаалт шүүх болон хуудаслах параметр
type LandAcquisitionFilter struct {
	PlanCode      string
	Status        int // 0 = шүүлт хийхгүй; 1=NEW, 2=ACTIVE, 3=CLOSED, 4=CANCELLED
	AU3Code       string
	StartDateFrom *time.Time
	StartDateTo   *time.Time
	Years         []int
	Page          int
	PageSize      int
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
	Page      int
	PageSize  int
}

// Parcel — нэгж талбар (main DB-д хадгалагдана)
type Parcel struct {
	ID                 uuid.UUID
	ParcelID           string
	OldParcelID        string
	AcquisitionID      uuid.UUID
	ValidFrom          *time.Time
	ValidTill          *time.Time
	AU1Code            string
	AU2Code            string
	AU3Code            string
	GeometryWKT        string
	AcquisitionGeomWKT string  // чөлөөлөлтийн бүстэй давхцах хэсгийн геометр
	AcquisitionAreaM2  float64 // давхцах хэсгийн талбай (м²)
	RemainingAreaM2    *float64
	RightType          int
	Landuse            string
	AreaM2             float64
	CompensationPaid   bool
	DbChanged          bool
	ChangedParcelID    string
	CreatedAt          time.Time
	CreatedBy          string
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
	Detail *ParcelDetailInfo
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
	AcquisitionID    string
	PlanCode         string
	AcquisitionName  string
	AU3Code          string
	RightType        int
	Landuse          string
	Years            []int  // start_date-ийн он (олон утга дэмжинэ)
	CompensationType string // 'cash' | 'land_grant' | '' (бүгд)
	Page             int
	PageSize         int
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
	AcquisitionID        uuid.UUID
	AcquisitionName      string
	PlanCode             string
	ConstructionTypeName string
	DecreeNumber         string
	DecreeDate           *time.Time
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
	ParcelID string
	Page     int
	PageSize int
}

// Asset — хөрөнгийн мэдээлэл
type Asset struct {
	ID            uuid.UUID `json:"id"`
	AcquisitionID uuid.UUID `json:"acquisition_id"`
	ParcelID      string    `json:"parcel_id"`
	AssetNumber   string    `json:"asset_number"`
	AssetType     string    `json:"asset_type"`
	AssetName     string    `json:"asset_name"`
	FloorCount    int       `json:"floor_count"`
	AreaM2        float64   `json:"area_m2"`
	OwnerName     string    `json:"owner_name"`
	Address       string    `json:"address"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	TargetType       string             `json:"target_type"` // 'parcel' | 'asset'
	ParcelID         string             `json:"parcel_id"`
	AssetID          *uuid.UUID         `json:"asset_id"`
	CompensationType string             `json:"compensation_type"`
	CoveragePercent  float64            `json:"coverage_percent"`
	Amount           float64            `json:"amount"`
	CompensationDate *time.Time         `json:"compensation_date"`
	Note             string             `json:"note"`
	Grant            *CompensationGrant `json:"grant,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	CreatedBy        string             `json:"created_by"`
	UpdatedAt        time.Time          `json:"updated_at"`
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

// DashboardFilter — дашбоардын шүүлтүүрийн параметр
type DashboardFilter struct {
	AcquisitionID string
	PlanCode      string
	Years         []int
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
}
