package domain

import "time"

// Захирамжийн төслийн төлөв.
const (
	DecisionDraftStatusDraft     = 1 // Төсөл
	DecisionDraftStatusConfirmed = 2 // Баталгаажсан
)

const (
	DecisionDraftProgressReviewing  = "reviewing"
	DecisionDraftProgressConfirming = "confirming"
)

// Захирамжид зөвхөн domain.ParcelStatusReleased ("Чөлөөлсөн", land.go)
// төлөвтэй нэгж талбар холбогдоно.

// DecisionOption — Захирамжийн төслийн сонголтын жагсаалтын нэгж бичлэг.
// Ажлын төрөл (decision_work_type) болон Төсөв (decision_budget) хоёулаа
// ижил бүтэцтэй тул нэг төрлөөр илэрхийлэв. Тохиргоо цэснээс удирдана.
type DecisionOption struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// DecisionDraft — захирамжийн төсөл. *Name талбарууд нь JOIN-оос ирэх
// уншихад зориулсан утга (хадгалагдахгүй).
type DecisionDraft struct {
	ID                       string     `json:"id"`
	ProposalNo               string     `json:"proposal_no"`
	DecreeNumber             string     `json:"decree_number"`
	DecisionDate             *time.Time `json:"decision_date,omitempty"`
	Location                 string     `json:"location"`
	DurationYear             *int       `json:"duration_year,omitempty"`
	Status                   int        `json:"status"`
	AcquisitionID            *string    `json:"acquisition_id,omitempty"`
	WorkTypeID               *int       `json:"work_type_id,omitempty"`
	BudgetID                 *int       `json:"budget_id,omitempty"`
	CurrentProgressHistoryID *string    `json:"current_progress_history_id,omitempty"`
	ConfirmedAt              *time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy              string     `json:"confirmed_by"`
	CreatedAt                time.Time  `json:"created_at"`
	CreatedBy                string     `json:"created_by"`
	UpdatedAt                time.Time  `json:"updated_at"`
	UpdatedBy                string     `json:"updated_by"`

	AcquisitionName string  `json:"acquisition_name"`
	PlanCode        string  `json:"plan_code"`
	WorkTypeName    string  `json:"work_type_name"`
	BudgetName      string  `json:"budget_name"`
	ParcelCount     int     `json:"parcel_count"`
	ParcelAreaM2    float64 `json:"parcel_area_m2"`

	// Санхүүгийн эх үүсвэрүүд — төслийг үүсгэсний ДАРАА олноор нэмнэ.
	// Жагсаалтад нийлбэр мэдээлэл хэрэгтэй тул тоо/нэрсийг хамт буцаана.
	FundingSourceCount         int                         `json:"funding_source_count"`
	FundingSourceNames         string                      `json:"funding_source_names"`
	FundingLocalAmount         float64                     `json:"funding_local_amount"`
	FundingInternationalAmount float64                     `json:"funding_international_amount"`
	FundingSourceAmounts       map[string]float64          `json:"funding_source_amounts"`
	FundingSourceCompensations map[string]float64          `json:"funding_source_compensation_amounts"`
	FundingSourceParcelCounts  map[string]int              `json:"funding_source_parcel_counts"`
	FundingSourceParcelAreas   map[string]float64          `json:"funding_source_parcel_areas"`
	FundingSources             []*DecisionDraftFundingLink `json:"funding_sources,omitempty"`

	CurrentProgressType      string     `json:"current_progress_type"`
	CurrentProgressTypeName  string     `json:"current_progress_type_name"`
	CurrentProgressRecipient string     `json:"current_progress_recipient"`
	CurrentProgressDate      *time.Time `json:"current_progress_date,omitempty"`
	CurrentProgressNote      string     `json:"current_progress_note"`

	// Document — баталгаажуулахад хавсаргасан захирамжийн эх баримт
	// (decision_draft.document_id). Зөвхөн дэлгэрэнгүйд бөглөгдөнө.
	Document *DecisionDraftDocument `json:"document,omitempty"`
}

// DecreeDocumentType — баталгаажсан захирамжийн хавсралтын төрөл
// (document_type.type = 'decree', «Захирамж»). Seeder нэмнэ.
const DecreeDocumentType = "decree"

// DecisionDraftDocument — захирамжийн хавсралт файл.
type DecisionDraftDocument struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	FileType   string    `json:"file_type"`
	FileURL    string    `json:"file_url"`
	SizeBytes  int64     `json:"size_bytes"`
	UploadedBy string    `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// ValidFundingYear — санхүүжилтийн он зөв мужид байгаа эсэх.
func ValidFundingYear(year int) bool {
	return year >= 2000 && year <= 2100
}

// DecisionDraftFundingLink — захирамж ↔ санхүүгийн эх үүсвэрийн холбоос.
type DecisionDraftFundingLink struct {
	ID              string `json:"id"`
	DecisionDraftID string `json:"decision_draft_id"`
	FundingSourceID string `json:"funding_source_id"`
	// Year — санхүүжилтийн ОН. Нэг эх үүсвэр он бүрт өөр дүнтэй (тусдаа
	// холбоос) байж болно. Хуучин мөрд seeder бөглөнө.
	Year      *int      `json:"year"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`

	// funding_sources-оос JOIN-оор ирэх мэдээлэл
	AcquisitionID    string   `json:"acquisition_id"`
	AcquisitionName  string   `json:"acquisition_name"`
	OrganizationName string   `json:"organization_name"`
	SourceType       string   `json:"source_type"`
	Amount           *float64 `json:"amount,omitempty"`
	Currency         string   `json:"currency"`
	Note             string   `json:"note"`
}

// DecisionDraftProgressHistory — захирамжийн төслийн явцын append-only түүх.
// Захирамжийн дугаарыг энд хадгалахгүй, зөвхөн decision_draft дээр байна.
type DecisionDraftProgressHistory struct {
	ID               string `json:"id"`
	DecisionDraftID  string `json:"decision_draft_id"`
	ProgressType     string `json:"progress_type"`
	ProgressTypeName string `json:"progress_type_name"`
	Recipient        string `json:"recipient"`
	// ReviewUnitID — «Хянагдаж буй газар» (decision_review_unit). Recipient нь
	// түүний нэрийн хуулбар; хуучин (чөлөөт текст) бичлэгт nil байж болно.
	ReviewUnitID *int      `json:"review_unit_id"`
	ProgressDate time.Time `json:"progress_date"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
}

// DecisionDraftParcel — захирамж ↔ нэгж талбарын холбоос. RemovedAt != nil
// бол хасагдсан (түүхэнд үлдсэн) холбоос.
type DecisionDraftParcel struct {
	ID              string     `json:"id"`
	DecisionDraftID string     `json:"decision_draft_id"`
	ParcelUUID      string     `json:"parcel_uuid"`
	LinkedAt        time.Time  `json:"linked_at"`
	LinkedBy        string     `json:"linked_by"`
	RemovedAt       *time.Time `json:"removed_at,omitempty"`
	RemovedBy       string     `json:"removed_by"`

	// Тухайн нэгж талбарыг санхүүжүүлэх эх үүсвэр — холбох үед сонгоно,
	// дараа нь солих боломжтой. decision_draft_funding_source руу заана.
	FundingLinkID       *string `json:"funding_link_id,omitempty"`
	FundingSourceID     string  `json:"funding_source_id,omitempty"`
	FundingOrganization string  `json:"funding_organization,omitempty"`
	FundingSourceType   string  `json:"funding_source_type,omitempty"`
	FundingYear         *int    `json:"funding_year,omitempty"`

	// Нэгж талбарын JOIN мэдээлэл
	ParcelID           string  `json:"parcel_id"`
	AcquisitionID      string  `json:"acquisition_id"`
	AcquisitionName    string  `json:"acquisition_name"`
	AreaM2             float64 `json:"area_m2"`
	AcquisitionAreaM2  float64 `json:"acquisition_area_m2"`
	CompensationAmount float64 `json:"compensation_amount"`
	ParcelStatus       int     `json:"parcel_status"`
	ParcelStatusName   string  `json:"parcel_status_name"`
	Landuse            string  `json:"landuse"`

	// ListByParcel-д захирамжийн мэдээллийг хамт буцаана
	ProposalNo     string     `json:"proposal_no,omitempty"`
	DecreeNumber   string     `json:"decree_number,omitempty"`
	DecisionDate   *time.Time `json:"decision_date,omitempty"`
	DecisionStatus int        `json:"decision_status,omitempty"`
	WorkTypeName   string     `json:"work_type_name,omitempty"`
	BudgetName     string     `json:"budget_name,omitempty"`
	Location       string     `json:"location,omitempty"`
	DurationYear   *int       `json:"duration_year,omitempty"`

	CurrentProgressType      string     `json:"current_progress_type,omitempty"`
	CurrentProgressTypeName  string     `json:"current_progress_type_name,omitempty"`
	CurrentProgressRecipient string     `json:"current_progress_recipient,omitempty"`
	CurrentProgressDate      *time.Time `json:"current_progress_date,omitempty"`
	CurrentProgressNote      string     `json:"current_progress_note,omitempty"`
}

// FundingSourceOption — захирамжийн төслийн "Санхүүгийн эх үүсвэр" сонголт.
// funding_sources нь чөлөөлөлт тус бүрийн дэд хүснэгт бөгөөд захирамжийн
// чөлөөлөлт нь нэгж талбар холбогдох хүртэл тодорхойгүй тул сонголтын
// жагсаалтад бүх чөлөөлөлтийн эх үүсвэрийг нэрийнх нь хамт харуулна.
type FundingSourceOption struct {
	ID               string   `json:"id"`
	AcquisitionID    string   `json:"acquisition_id"`
	AcquisitionName  string   `json:"acquisition_name"`
	PlanCode         string   `json:"plan_code"`
	OrganizationName string   `json:"organization_name"`
	SourceType       string   `json:"source_type"`
	Amount           *float64 `json:"amount,omitempty"`
	Currency         string   `json:"currency"`
}

// ParcelLinkInfo — захирамжид холбохоос өмнөх шалгалтад шаардлагатай нэгж
// талбарын хамгийн бага мэдээлэл.
type ParcelLinkInfo struct {
	ID            string
	ParcelID      string
	Status        int
	AcquisitionID string
	// IsReleased — «Захирамжид холбогдоно» (parcel_status.is_released) төлөв.
	IsReleased bool
}

// DecisionDraftFilter — жагсаалтын шүүлт. ParcelID нь холбогдсон нэгж
// талбарын дугаараар (ILIKE) шүүнэ.
type DecisionDraftFilter struct {
	ProposalNo   string
	DecreeNumber string
	Location     string
	ParcelID     string
	DateFrom     string
	DateTo       string
	DurationYear int
	// FundingYear — санхүүжилтийн он: тэр онтой холбоостой төсөл л гарна, мөн
	// санхүүжилтийн нэгтгэл (дүн, эх үүсвэрээрх задаргаа) ЗӨВХӨН тэр оных.
	FundingYear     int
	Status          int
	AcquisitionID   string
	FundingSourceID string
	WorkTypeID      int
	BudgetID        int
	SortBy          string // decisionDraftSortColumns-ийн түлхүүр
	SortDir         string // asc | desc
	Page            int
	PageSize        int
}

// ── Захирамж ба санхүүжилтийн нэгтгэл (хяналтын самбар, чөлөөлөлтийн
// «Санхүүжилт» таб) ─────────────────────────────────────────────────────────

// DecisionFundingFilter — нэгтгэлийн шүүлт. Years нь САНХҮҮЖИЛТИЙН он
// (decision_draft_funding_source.year): нэгж талбар өөрийн санхүүжилтийн
// холбоосын онд хамаарна. Хоосон бол бүх он.
type DecisionFundingFilter struct {
	// DepartmentID — утгатай бол ЗӨВХӨН тухайн хэлтсийн чөлөөлөлт.
	DepartmentID      *int
	Years             []int
	AcquisitionID     string
	PlanCode          string
	GeneralCategoryID int
	SubCategoryID     int
	AU2Code           string
	AssignedUserID    string
}

// DecisionFundingBucket — захирамжийн шатны нөхөх олговор (нэгж талбарын
// сонгосон үнэлгээний бэлэн мөнгөн олговор). Granted — нэгж талбарын олголтын
// гүйцэтгэлээр (60 / 40 / бүрэн) ОЛГОГДСОН хэсэг, Pending = Amount − Granted.
type DecisionFundingBucket struct {
	Amount  float64 `json:"amount"`
	Granted float64 `json:"granted"`
	Pending float64 `json:"pending"`
	Parcels int64   `json:"parcels"`
}

// DecisionFundingSummary — сонгосон оны төлөвлөсөн санхүүжилт ба түүнээс
// захирамжаар зарцуулагдсан / олгогдсон дүн.
type DecisionFundingSummary struct {
	// Years — хүрээнд байгаа санхүүжилтийн бүх он (шүүлтийн сонголт).
	Years []int `json:"years"`
	// Planned — төлөвлөсөн санхүүжилт (холбоосын дүнгийн нийлбэр).
	Planned float64 `json:"planned"`
	// PlannedGranted — тэр санхүүжилтээр холбогдсон нэгж талбарын ОЛГОСОН дүн.
	PlannedGranted float64 `json:"planned_granted"`
	// Issued — захирамж гарсан (Баталгаажсан, эсвэл Баталгаажуулах шатанд).
	Issued DecisionFundingBucket `json:"issued"`
	// Draft — захирамжийн төсөлд (Төсөл / Хянагдаж буй).
	Draft DecisionFundingBucket `json:"draft"`
	// BySource — төлөвлөсөн санхүүжилт эх үүсвэр × оноор (чөлөөлөлтийн
	// «Санхүүжилтын эх үүсвэрүүд» хүснэгтэд дүн бүрийн он).
	BySource []DecisionFundingSourceYear `json:"by_source"`
}

// DecisionFundingSourceYear — нэг эх үүсвэрийн нэг оны төлөвлөсөн дүн.
type DecisionFundingSourceYear struct {
	FundingSourceID string  `json:"funding_source_id"`
	Year            *int    `json:"year"`
	Amount          float64 `json:"amount"`
}
