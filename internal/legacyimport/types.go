package legacyimport

// Issue — шалгалтын үр дүн.
//
//	error   — цааш явахыг ХААНА (засаж дахин оруулна)
//	warning — оруулж болно, хэрэглэгч мэдэх ёстой
//	info    — зөвхөн мэдээлэл
type Issue struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func errIssue(code, msg string) Issue  { return Issue{Level: "error", Code: code, Message: msg} }
func warnIssue(code, msg string) Issue { return Issue{Level: "warning", Code: code, Message: msg} }
func infoIssue(code, msg string) Issue { return Issue{Level: "info", Code: code, Message: msg} }

func countLevel(issues []Issue, level string) int {
	n := 0
	for _, i := range issues {
		if i.Level == level {
			n++
		}
	}
	return n
}

func hasCode(issues []Issue, code string) bool {
	for _, i := range issues {
		if i.Code == code {
			return true
		}
	}
	return false
}

// UploadedFile — хэрэглэгчийн оруулсан файл (зөвхөн basename).
type UploadedFile struct {
	Name string
	Data []byte
}

// FileInfo — файл тус бүрийн уншилтын үр дүн.
type FileInfo struct {
	Name     string  `json:"name"`
	Features int     `json:"features"`
	Issues   []Issue `json:"issues"`
}

// ── Түр хүснэгтэд хадгалах эх мөрүүд ───────────────────────────────────────

// BoundaryRow — чөлөөлөлтийн хилийн нэг бичлэг.
type BoundaryRow struct {
	Index        int               `json:"index"`
	PlanParcelID string            `json:"planParcelID"`
	WKT          string            `json:"wkt"` // EPSG:4326
	Properties   map[string]string `json:"properties"`
}

// ParcelRow — нэгж талбарын shp-ийн нэг бичлэг (НӨЛӨӨЛӨЛД ӨРТСӨН хэсэг).
type ParcelRow struct {
	Index       int               `json:"index"`
	ParcelID    string            `json:"parcelID"`
	OldParcelID string            `json:"oldParcelID"`
	SourceCode  *int              `json:"sourceCode"`
	AreaM2      *float64          `json:"areaM2"`
	WKT         string            `json:"wkt"` // EPSG:4326
	Note        string            `json:"note"`
	Properties  map[string]string `json:"properties"`
}

// ReportRow — үнэлгээний эксэлийн нэг мөр.
type ReportRow struct {
	SourceFile            string   `json:"sourceFile"`
	SourceSheet           string   `json:"sourceSheet"`
	ExcelRow              int      `json:"excelRow"`
	RowNo                 string   `json:"rowNo"`
	ParcelID              string   `json:"parcelID"`
	OldParcelID           string   `json:"oldParcelID"`
	SheetYear             *int     `json:"sheetYear"`
	ProjectName           string   `json:"projectName"`
	ConstructionType      string   `json:"constructionType"`
	DecisionNo            string   `json:"decisionNo"`
	ListNo                string   `json:"listNo"`
	FundingSource         string   `json:"fundingSource"`
	Decree                string   `json:"decree"`
	DecreeDate            string   `json:"decreeDate"`
	HolderName            string   `json:"holderName"`
	HolderRegisterNo      string   `json:"holderRegisterNo"`
	Address               string   `json:"address"`
	AreaM2                *float64 `json:"areaM2"`
	RightTypeLabel        string   `json:"rightTypeLabel"`
	CertificateNo         string   `json:"certificateNo"`
	AffectedAreaM2        *float64 `json:"affectedAreaM2"`
	LandAmount            float64  `json:"landAmount"`
	AssetNumber           string   `json:"assetNumber"`
	RealStateAmount       float64  `json:"realStateAmount"`
	PropertyAmount        float64  `json:"propertyAmount"`
	TotalAmount           float64  `json:"totalAmount"`
	CompensationTypeLabel string   `json:"compensationTypeLabel"`
	DBChangeNote          string   `json:"dbChangeNote"`
	ChangedParcelID       string   `json:"changedParcelID"`
	Note                  string   `json:"note"`
}

func (r *ReportRow) amount() float64 {
	if r.TotalAmount > 0 {
		return r.TotalAmount
	}
	return r.LandAmount + r.RealStateAmount + r.PropertyAmount
}

// ── Дэлгэцэнд харуулах үр дүн (frontend-ийн api.ts-тэй ИЖИЛ хэлбэр) ─────────

type CategoryRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Categories struct {
	General CategoryRef  `json:"general"`
	Sub     *CategoryRef `json:"sub"`
}

type BoundaryViewRow struct {
	Idx           int      `json:"idx"`
	File          string   `json:"file"`
	PlanParcelID  string   `json:"planParcelID"`
	Name          string   `json:"name"`
	PlanCode      string   `json:"planCode"`
	PlanFound     bool     `json:"planFound"`
	AreaM2        *float64 `json:"areaM2"`
	InsidePlanPct *float64 `json:"insidePlanPct"`
	Existing      bool     `json:"existing"`
	Issues        []Issue  `json:"issues"`
}

type BoundaryMap struct {
	Idx      int     `json:"idx"`
	Boundary string  `json:"boundary"`
	Plan     *string `json:"plan"`
}

type BoundaryView struct {
	Categories *Categories       `json:"categories"`
	Files      []FileInfo        `json:"files"`
	Rows       []BoundaryViewRow `json:"rows"`
	Map        []BoundaryMap     `json:"map"`
	Errors     int               `json:"errors"`
	Warnings   int               `json:"warnings"`
	CanProceed bool              `json:"canProceed"`
}

type ParcelViewRow struct {
	Idx          int    `json:"idx"`
	File         string `json:"file"`
	ParcelID     string `json:"parcelID"`
	OldParcelID  string `json:"oldParcelID"`
	Synthetic    bool   `json:"synthetic"`
	SourceCode   *int   `json:"sourceCode"`
	Status       int    `json:"status"`
	StatusName   string `json:"statusName"`
	StatusMapped bool   `json:"statusMapped"`
	// StatusUnmatched — SHP-ийн код системийн төлөвтэй таараагүй (эхний төлөвт орно).
	StatusUnmatched  bool     `json:"statusUnmatched"`
	AreaM2           *float64 `json:"areaM2"`
	BoundaryIdx      *int     `json:"boundaryIdx"`
	InsidePct        *float64 `json:"insidePct"`
	Outside          bool     `json:"outside"`
	NearestDistanceM *float64 `json:"nearestDistanceM"`
	AlreadyImported  bool     `json:"alreadyImported"`
	Gus              string   `json:"gus"`
	Issues           []Issue  `json:"issues"`
}

type ParcelStats struct {
	Total         int            `json:"total"`
	Synthetic     int            `json:"synthetic"`
	Duplicates    int            `json:"duplicates"`
	NoGeometry    int            `json:"noGeometry"`
	UnmappedCodes map[string]int `json:"unmappedCodes"`
	// StatusUnmatched — төлөвийн код таараагүй нэгж талбарын тоо.
	StatusUnmatched int `json:"statusUnmatched"`
	// InitialStatusName — таараагүй / кодгүй нэгж талбар орох эхний төлөв.
	InitialStatusName string         `json:"initialStatusName"`
	AlreadyImported   int            `json:"alreadyImported"`
	Outside           int            `json:"outside"`
	PartlyOutside     int            `json:"partlyOutside"`
	Gus               map[string]int `json:"gus"`
}

type ParcelMap struct {
	Idx      int    `json:"idx"`
	ParcelID string `json:"parcelID"`
	WKT      string `json:"wkt"`
	Outside  bool   `json:"outside"`
}

type ParcelView struct {
	Files      []FileInfo      `json:"files"`
	Rows       []ParcelViewRow `json:"rows"`
	Stats      ParcelStats     `json:"stats"`
	Map        []ParcelMap     `json:"map"`
	Errors     int             `json:"errors"`
	Warnings   int             `json:"warnings"`
	GusDone    bool            `json:"gusDone"`
	CanProceed bool            `json:"canProceed"`
}

type ReportViewRow struct {
	Idx            int      `json:"idx"`
	File           string   `json:"file"`
	Sheet          string   `json:"sheet"`
	ExcelRow       int      `json:"excelRow"`
	ParcelID       string   `json:"parcelID"`
	OldParcelID    string   `json:"oldParcelID"`
	HolderName     string   `json:"holderName"`
	TotalAmount    float64  `json:"totalAmount"`
	AffectedAreaM2 *float64 `json:"affectedAreaM2"`
	ShpAreaM2      *float64 `json:"shpAreaM2"`
	Match          string   `json:"match"`
	GusOrigin      *string  `json:"gusOrigin"`
	Issues         []Issue  `json:"issues"`
	// «Зөвхөн үнэлгээ» горим: төсөл (чөлөөлөлт), системд бүртгэгдэх дугаар.
	Project    string `json:"project,omitempty"`
	AssignedID string `json:"assignedID,omitempty"`
	IDKind     string `json:"idKind,omitempty"` // real | duplicate | synthetic
}

type ReportStats struct {
	Rows          int     `json:"rows"`
	Staged        int     `json:"staged"`
	DB            int     `json:"db"`
	Gus           int     `json:"gus"`
	GusFromUb     int     `json:"gusFromUb"`
	Missing       int     `json:"missing"`
	Pending       int     `json:"pending"`
	WithoutReport int     `json:"withoutReport"`
	TotalAmount   float64 `json:"totalAmount"`
	MissingAmount float64 `json:"missingAmount"`
	AreaDiffs     int     `json:"areaDiffs"`
	Balanced      int     `json:"balanced"`
	Duplicates    int     `json:"duplicates"`
	// OwnerIssues — өмчлөгчийн овог нэр/регистр дутуу эсвэл зөрүүтэй мөр.
	OwnerIssues int `json:"ownerIssues"`
	// «Зөвхөн үнэлгээ» горим.
	Excel     int `json:"excel"`     // ГУС-д олдоогүй — эксэлийн мэдээллээр бүртгэнэ
	Synthetic int `json:"synthetic"` // дугааргүй — ERR… дугаар
	Projects  int `json:"projects"`
}

// ValuationProject — «Зөвхөн үнэлгээ» горимын нэг төсөл = нэг чөлөөлөлт (хилгүй).
type ValuationProject struct {
	Name             string       `json:"name"`
	Variants         []string     `json:"variants"`
	Rows             int          `json:"rows"`
	Parcels          int          `json:"parcels"` // санд байхгүй — бүртгэгдэх нэгж талбар
	Amount           float64      `json:"amount"`
	ConstructionType string       `json:"constructionType"`
	Category         categoryName `json:"category"`
	CategoryMatched  bool         `json:"categoryMatched"`
	// CategoryIDs — Category-ийн ID (сонгох талбарын анхны утга); санд алга бол nil.
	CategoryIDs    *CategoryChoice `json:"categoryIds"`
	CategoryChosen bool            `json:"categoryChosen"` // гараар сонгосон (төрлөөс биш)
	HasInfo        bool            `json:"hasInfo"`        // чөлөөлөлтийн мэдээлэл бөглөсөн
	AreaM2         *float64        `json:"areaM2"`         // хилийн shapefile-аас; хилгүй бол nil
	Existing       bool            `json:"existing"`       // өмнө импортоор орсон чөлөөлөлт
	Issues         []Issue         `json:"issues"`
}

// ProjectSettings — «Зөвхөн үнэлгээ»: төсөл (чөлөөлөлт) бүрийн ГАРААР
// тохируулсан мэдээлэл. Бүгд заавал биш.
type ProjectSettings struct {
	Category *CategoryChoice  `json:"category,omitempty"` // ID-аар; nil бол бүтээн байгуулалтын төрлөөс
	Info     AcquisitionInfo  `json:"info"`
	Boundary *ProjectBoundary `json:"boundary,omitempty"`
}

// AcquisitionInfo — чөлөөлөлтийн «Төслийн мэдээлэл» (land_acquisition-ий баганууд).
type AcquisitionInfo struct {
	ImplementingOrg  string `json:"implementingOrg"`
	ResponsibleOrg   string `json:"responsibleOrg"`
	Reason           string `json:"reason"`
	StartDate        string `json:"startDate"` // YYYY-MM-DD
	EndDate          string `json:"endDate"`
	NithDecreeNumber string `json:"nithDecreeNumber"`
	GroupListNumber  string `json:"groupListNumber"`
}

// ProjectBoundary — шалгалт давсан хилийн shapefile (нэг полигон).
type ProjectBoundary struct {
	File   string  `json:"file"`
	WKT    string  `json:"wkt"` // EPSG:4326
	AreaM2 float64 `json:"areaM2"`
	Issues []Issue `json:"issues"` // зөвхөн анхааруулга (алдаатай хил хадгалагдахгүй)
}

type ParcelRef struct {
	ParcelID string `json:"parcelID"`
	File     string `json:"file"`
}

type ReportView struct {
	Files                []FileInfo         `json:"files"`
	Rows                 []ReportViewRow    `json:"rows"`
	ParcelsWithoutReport []ParcelRef        `json:"parcelsWithoutReport"`
	Projects             []ValuationProject `json:"projects,omitempty"`
	Stats                ReportStats        `json:"stats"`
	Errors               int                `json:"errors"`
	Warnings             int                `json:"warnings"`
	GusDone              bool               `json:"gusDone"`
	CanProceed           bool               `json:"canProceed"`
}

type SummaryAcquisition struct {
	Idx          int      `json:"idx"`
	PlanParcelID string   `json:"planParcelID"`
	Name         string   `json:"name"`
	AreaM2       *float64 `json:"areaM2"`
	Existing     bool     `json:"existing"`
	Parcels      int      `json:"parcels"`
	Outside      int      `json:"outside"`
	ReportRows   int      `json:"reportRows"`
}

type Summary struct {
	Acquisitions         []SummaryAcquisition `json:"acquisitions"`
	Parcels              *ParcelStats         `json:"parcels"`
	Reports              *ReportStats         `json:"reports"`
	ParcelsWithoutReport int                  `json:"parcelsWithoutReport"`
	Blockers             []string             `json:"blockers"`
	Cautions             []string             `json:"cautions"`
	CanCommit            bool                 `json:"canCommit"`
}

// CommitResult — «Оруулах»-ын үр дүн. Амжилтгүй бол зөвхөн Error.
type CommitResult struct {
	Error               string               `json:"error,omitempty"`
	CommittedAt         string               `json:"committedAt,omitempty"`
	Acquisitions        *CommitAcquisitions  `json:"acquisitions,omitempty"`
	Parcels             *CommitParcels       `json:"parcels,omitempty"`
	Compensations       *CommitCompensations `json:"compensations,omitempty"`
	AcquisitionStatuses *AcquisitionStatuses `json:"acquisitionStatuses,omitempty"`
	Log                 *CommitLog           `json:"log,omitempty"`
}

type CommitAcquisitions struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Names   []string `json:"names"`
}

type CommitParcels struct {
	Imported        int `json:"imported"`
	AlreadyImported int `json:"alreadyImported"`
	GusMissing      int `json:"gusMissing"`
	FromUb          int `json:"fromUb"`
	Nearest         int `json:"nearest"`
	// StatusUnmatched — төлөвийн код системд таараагүй, эхний төлөвт орсон.
	StatusUnmatched int `json:"statusUnmatched"`
}

type CommitCompensations struct {
	Applied        int     `json:"applied"`
	TotalAmount    float64 `json:"totalAmount"`
	MissingParcels int     `json:"missingParcels"`
}

type AcquisitionStatuses struct {
	Verified int `json:"verified"`
	Field    int `json:"field"`
	Changed  int `json:"changed"`
}

// CommitLog — «Оруулах»-ын мөр тус бүрийн тэмдэглэл (сан дотор хадгалагдана).
type CommitLog struct {
	Success  int        `json:"success"`
	Skipped  int        `json:"skipped"`
	Failed   int        `json:"failed"`
	Warnings int        `json:"warnings"`
	Entries  []LogEntry `json:"entries,omitempty"`
}

type LogEntry struct {
	Level   string `json:"level"`
	Stage   string `json:"stage"`
	Key     string `json:"key"`
	Message string `json:"message"`
}

// Job — ГУС-ын шалгалт / «Оруулах»-ын арын ажил.
type Job struct {
	Kind  string `json:"kind"`  // parcels | reports | commit
	State string `json:"state"` // running | done | failed
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Error string `json:"error,omitempty"`
}

type SessionInfo struct {
	ID        string `json:"id"`
	CreatedBy string `json:"createdBy"`
	// CreatedByID — үүсгэсэн хэрэглэгчийн дугаар (sd_user). Хуучин импортод хоосон.
	CreatedByID *int64          `json:"createdById,omitempty"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Status      string          `json:"status"`
	Mode        string          `json:"mode"` // full | valuation
	Category    *CategoryChoice `json:"category"`
	Rollback    *RollbackResult `json:"rollback,omitempty"`
}

type CategoryChoice struct {
	General string `json:"general"`
	Sub     string `json:"sub"`
}

// State — нэг импортын бүх алхмын үр дүн (GET /legacy-import/sessions/:id).
type State struct {
	Session  SessionInfo   `json:"session"`
	Boundary *BoundaryView `json:"boundary"`
	Parcels  *ParcelView   `json:"parcels"`
	Reports  *ReportView   `json:"reports"`
	Result   *CommitResult `json:"result"`
	Summary  Summary       `json:"summary"`
	Job      *Job          `json:"job"`
	// ProjectSettings — «Зөвхөн үнэлгээ»: төслийн нэр → гараар тохируулсан мэдээлэл.
	ProjectSettings map[string]ProjectSettings `json:"projectSettings"`
	// RollbackLog — «Устгах»-ын түүх (устгасан/алгассан/үлдээсэн мөр бүр).
	RollbackLog []RollbackLogEntry `json:"rollbackLog"`
}

type SessionListItem struct {
	ID           string `json:"id"`
	CreatedBy    string `json:"createdBy"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	Status       string `json:"status"`
	Mode         string `json:"mode"`
	Acquisitions int    `json:"acquisitions"`
	// AcquisitionNames — оруулсан чөлөөлөлтийн нэрс (оруулаагүй бол хилийн мөрийнх).
	AcquisitionNames []string `json:"acquisitionNames"`
	Parcels          int      `json:"parcels"`
	ReportRows       int      `json:"reportRows"`
	// Deletable — оруулсан бөгөөд импортын бүртгэл (устгах жагсаалт) нь байгаа эсэх.
	Deletable bool            `json:"deletable"`
	Rollback  *RollbackResult `json:"rollback,omitempty"`
}

type Category struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID *int   `json:"parentId"`
}
