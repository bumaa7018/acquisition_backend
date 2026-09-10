package http

import (
	"strconv"
	"time"

	"acquisition_backend/internal/domain"
)

type landAcquisitionResponse struct {
	ID                  string  `json:"id"`
	PlanCode            string  `json:"plan_code"`
	PlanName            string  `json:"plan_name"`
	GeometryWKT         string  `json:"geometry_wkt"`
	AreaM2              float64 `json:"area_m2"`
	Status              int     `json:"status"`
	StartDate           *string `json:"start_date"`
	EndDate             *string `json:"end_date"`
	AcquisitionName     string  `json:"acquisition_name"`
	ImplementingOrg     string  `json:"implementing_org"`
	Reason              string  `json:"reason"`
	ResponsibleOrg      string  `json:"responsible_org"`
	FundingSource       string  `json:"funding_source"`
	GeneralCategoryID   *int    `json:"general_category_id"`
	GeneralCategoryName string  `json:"general_category_name"`
	SubCategoryID       *int    `json:"sub_category_id"`
	SubCategoryName     string  `json:"sub_category_name"`
	DecreeNumber        string  `json:"decree_number"`
	DecreeDate          *string `json:"decree_date"`
	CreatedAt           string  `json:"created_at"`
	CreatedBy           string  `json:"created_by"`
	ParcelCount         int64   `json:"parcel_count"`
	// Явц — эцсийн төлөвт шилжсэн нэгж талбарын ЭЗЛЭХ ХУВЬ (0-100) ба тоо.
	// Жагсаалт/дашбоард хоёулаа ижил тоо харуулахын тулд backend бодно.
	FinalParcelCount int64 `json:"final_parcel_count"`
	// OverlappingParcelCount — БАЙРШЛААР давхардсан нэгж талбарын тоо.
	// 0-ээс их бол чөлөөлөлт "Хээрийн судалгаа" төлөв рүү шилжихгүй тул
	// жагсаалт дээр улаанаар анхааруулна.
	OverlappingParcelCount int64            `json:"overlapping_parcel_count"`
	ProgressPercent        int              `json:"progress_percent"`
	ProfessionalOrgID      *string          `json:"professional_org_id,omitempty"`
	ProfessionalOrgName    string           `json:"professional_org_name,omitempty"`
	AUs                    []auResponse     `json:"aus"`
	Parcels                []parcelResponse `json:"parcels"`
	// Дэлгэрэнгүй (GetByID) дээр л бөглөгдөнө — тусдаа funding-sources GET API-г орлоно
	FundingSources []fundingSourceResponse `json:"funding_sources,omitempty"`
	// Төлөвлөгөөний ДЭЛГЭРЭНГҮЙ — мөн зөвхөн GetByID дээр (plan хүснэгтээс).
	// plan_code нь өөрөө нэгж талбарын дугаар боловч хуучин бүртгэлд өөр байж
	// болох тул plan_parcel_id-г ТУСАД НЬ буцаана.
	PlanParcelID     string  `json:"plan_parcel_id,omitempty"`
	PlanTypeName     string  `json:"plan_type_name,omitempty"`
	PlanGazner       string  `json:"plan_gazner,omitempty"`
	PlanAreaM2       float64 `json:"plan_area_m2,omitempty"`
	PlanStartDate    *string `json:"plan_start_date,omitempty"`
	PlanEndDate      *string `json:"plan_end_date,omitempty"`
	PlanApprovedDate *string `json:"plan_approved_date,omitempty"`
}

type auResponse struct {
	AU1Code string `json:"au1_code"`
	AU1Name string `json:"au1_name"`
	AU2Code string `json:"au2_code"`
	AU2Name string `json:"au2_name"`
	AU3Code string `json:"au3_code"`
	AU3Name string `json:"au3_name"`
}

type parcelResponse struct {
	ID                 string   `json:"id"`
	ParcelID           string   `json:"parcel_id"`
	AU1Code            string   `json:"au1_code"`
	AU2Code            string   `json:"au2_code"`
	AU3Code            string   `json:"au3_code"`
	RightType          int      `json:"right_type"`
	Landuse            string   `json:"landuse"`
	LanduseName        string   `json:"landuse_name,omitempty"`
	AreaM2             float64  `json:"area_m2"`
	AcquisitionAreaM2  float64  `json:"acquisition_area_m2"`
	RemainingAreaM2    *float64 `json:"remaining_area_m2"`
	CompensationPaid   bool     `json:"compensation_paid"`
	DbChanged          string   `json:"db_changed"`
	ChangedParcelID    string   `json:"changed_parcel_id"`
	StatusID           int      `json:"status_id"`
	StatusName         string   `json:"status_name"`
	IndependentOrgID   *string  `json:"independent_org_id,omitempty"`
	IndependentOrgName string   `json:"independent_org_name,omitempty"`
	// HolderName — газар өмчлөгч/эзэмшигчийн бүтэн нэр (ҮНДСЭН хүсэлт гаргагч).
	// HolderCount нь нийт эзэмшигчийн тоо — жагсаалт дээр "нэр +2" гэж
	// харуулахад хэрэглэгдэнэ (нэг талбар дээр олон эзэмшигч байж болно).
	HolderName  string `json:"holder_name,omitempty"`
	HolderCount int    `json:"holder_count"`
	// HasOverlap — өөр нэгж талбартай БАЙРШЛААР давхцаж байгаа эсэх.
	// Давхардал нь нэг газрын олговрыг хоёр удаа бодох эрсдэл тул дэлгэц дээр
	// улаанаар анхааруулна; OverlapCount нь давхцаж буй талбарын тоо.
	HasOverlap   bool `json:"has_overlap"`
	OverlapCount int  `json:"overlap_count"`
	// урсгал бүрийн нөхөх олговрын илгээх төлөв: valuation_type → draft|submitted|approved|returned
	ValuationStatuses map[string]string `json:"valuation_statuses,omitempty"`
	CashAmount        float64           `json:"cash_amount"`
	LandGrantAmount   float64           `json:"land_grant_amount"`
	LandGrantCount    int               `json:"land_grant_count"`
	GeometryWKT       string            `json:"geometry_wkt,omitempty"`
}

type parcelFullResponse struct {
	ID                    string   `json:"id"`
	ParcelID              string   `json:"parcel_id"`
	OldParcelID           string   `json:"old_parcel_id,omitempty"`
	AcquisitionID         string   `json:"acquisition_id"`
	ValidFrom             *string  `json:"valid_from"`
	ValidTill             *string  `json:"valid_till"`
	AU1Code               string   `json:"au1_code"`
	AU2Code               string   `json:"au2_code"`
	AU3Code               string   `json:"au3_code"`
	GeometryWKT           string   `json:"geometry_wkt"`
	AcquisitionGeomWKT    string   `json:"acquisition_geom_wkt"`
	AcquisitionAreaM2     float64  `json:"acquisition_area_m2"`
	RemainingAreaM2       *float64 `json:"remaining_area_m2"`
	RightType             int      `json:"right_type"`
	Landuse               string   `json:"landuse"`
	LanduseName           string   `json:"landuse_name,omitempty"`
	AreaM2                float64  `json:"area_m2"`
	AddressKhashaa        string   `json:"address_khashaa,omitempty"`
	AddressStreet         string   `json:"address_streetname,omitempty"`
	PropertyNo            string   `json:"property_no,omitempty"`
	DbChanged             string   `json:"db_changed"`
	ChangedParcelID       string   `json:"changed_parcel_id"`
	StatusID              int      `json:"status_id"`
	StatusName            string   `json:"status_name"`
	IndependentOrgID      *string  `json:"independent_org_id,omitempty"`
	SelectedValuationType *string  `json:"selected_valuation_type,omitempty"`
	// EstimatedValue — ТӨСӨӨЛЛИЙН үнэлгээ (₮), гараас оруулсан. null =
	// оруулаагүй. Мэргэжлийн байгууллагад ЗӨВХӨН ХАРАГДАНА.
	EstimatedValue             *float64          `json:"estimated_value"`
	EstimatedConfidencePercent *float64          `json:"estimated_confidence_percent"`
	EstimatedBaseFeePerM2      *float64          `json:"estimated_base_fee_per_m2"`
	EstimatedValueAt           *string           `json:"estimated_value_at,omitempty"`
	EstimatedValueBy           string            `json:"estimated_value_by,omitempty"`
	CreatedAt                  string            `json:"created_at"`
	CreatedBy                  string            `json:"created_by"`
	Detail                     *parcelDetailInfo `json:"detail,omitempty"`
	// Holders — БҮХ эзэмшигч (нэг хүсэлт дээр төрөл өөр олон хүн байж болно).
	// detail дотрох holder_* талбарууд нь зөвхөн үндсэн хүсэлт гаргагчийг заана.
	Holders []parcelHolderResponse `json:"holders"`
	// ГУС-аас татагдсан санхүү, эрх зүйн бүртгэлүүд
	Mortgages      []domain.ParcelMortgage      `json:"mortgages"`
	CourtDecisions []domain.ParcelCourtDecision `json:"court_decisions"`
	Invoices       []domain.ParcelInvoice       `json:"invoices"`
	Monitorings    []domain.ParcelMonitoring    `json:"monitorings"`
	// Fees — газрын төлбөрийн бодолт бүсээр. Нийт төлбөр нь payment-ийн нийлбэр.
	Fees             []domain.ParcelFeeItem         `json:"fees"`
	BasePriceFactors []domain.ParcelBasePriceFactor `json:"base_price_factors"`
	// Ownerships — УБЕГ-ийн газар өмчлөлийн бүртгэл (ХУР/XYP гарцаас татагдсан).
	// ГУС-ийн эзэмшигчийн жагсаалт (holders)-аас ТУСДАА эх сурвалж.
	Ownerships []domain.ParcelOwnership `json:"ownerships"`
	// Overlaps — БАЙРШЛААР давхцаж байгаа бусад нэгж талбарууд. Хоосон биш бол
	// дэлгэц УЛААНААР анхааруулна.
	Overlaps []domain.ParcelOverlap `json:"overlaps"`
}

type parcelHolderResponse struct {
	ID            string `json:"id"`
	MainApplicant bool   `json:"main_applicant"`
	LastName      string `json:"last_name"`
	Name          string `json:"name"`
	RegisterNo    string `json:"register_no"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	// PersonType — эзэмшигчийн төрөл, жишээ нь "3: Монгол улсын хуулийн этгээд".
	// Хуулийн этгээд үед last_name хоосон, name-д байгууллагын бүтэн нэр байна.
	PersonType string `json:"person_type"`
	AppNo      string `json:"app_no"`
	// HolderRole — "holder" (ГУС-аас татагдсан эзэмшигч) эсвэл
	// "representative" (гараар бүртгэсэн итгэмжлэгдсэн төлөөлөгч).
	HolderRole string `json:"holder_role"`
	// PaymentRecipient — нөхөн төлбөр хүлээн авах эзэмшигч (нэгж талбарт НЭГ).
	PaymentRecipient bool   `json:"payment_recipient"`
	Note             string `json:"note,omitempty"`
	Address          string `json:"address,omitempty"`
}

func toParcelHolderResponse(h domain.ParcelHolder) parcelHolderResponse {
	return parcelHolderResponse{
		ID:               h.ID.String(),
		MainApplicant:    h.MainApplicant,
		LastName:         h.LastName,
		Name:             h.Name,
		RegisterNo:       h.RegisterNo,
		Phone:            h.Phone,
		Email:            h.Email,
		PersonType:       h.PersonType,
		AppNo:            h.AppNo,
		HolderRole:       h.HolderRole,
		PaymentRecipient: h.PaymentRecipient,
		Note:             h.Note,
		Address:          h.Address,
	}
}

type parcelDetailInfo struct {
	RightType           int      `json:"right_type"`
	HolderLastName      string   `json:"holder_last_name"`
	HolderName          string   `json:"holder_name"`
	HolderRegisterNo    string   `json:"holder_register_no"`
	HolderPhone         string   `json:"holder_phone"`
	HolderEmail         string   `json:"holder_email"`
	HolderType          string   `json:"holder_type"`
	HolderCivilID       string   `json:"holder_civil_id"`
	AppNo               string   `json:"app_no"`
	AppTimestamp        *string  `json:"app_timestamp"`
	AppType             string   `json:"app_type"`
	AppStatus           string   `json:"app_status"`
	DecisionNo          string   `json:"decision_no"`
	DecisionDate        *string  `json:"decision_date"`
	ContractNo          string   `json:"contract_no"`
	ContractDate        *string  `json:"contract_date"`
	ContractBegin       *string  `json:"contract_begin"`
	ContractEnd         *string  `json:"contract_end"`
	ContractPropertyNo  string   `json:"contract_property_no"`
	ContractStatus      string   `json:"contract_status"`
	CertificateNo       string   `json:"certificate_no"`
	CertificateDate     *string  `json:"certificate_date"`
	RecordNo            string   `json:"record_no"`
	RecordDate          *string  `json:"record_date"`
	RecordCertificateNo string   `json:"record_certificate_no"`
	RecordStatus        string   `json:"record_status"`
	ValuationZone       string   `json:"valuation_zone"`
	BasePricePerHa      *float64 `json:"base_price_per_ha"`
	AuctionCoeff        *float64 `json:"auction_coeff"`
	AuctionPrice        *float64 `json:"auction_price"`
}

type progressResponse struct {
	ID         string `json:"id"`
	FromStatus int    `json:"from_status"`
	ToStatus   int    `json:"to_status"`
	Note       string `json:"note"`
	ChangedBy  string `json:"changed_by"`
	ChangedAt  string `json:"changed_at"`
}

type boundaryHistoryResponse struct {
	ID                string `json:"id"`
	LandAcquisitionID string `json:"land_acquisition_id"`
	OldGeometryWKT    string `json:"old_geometry_wkt"`
	NewGeometryWKT    string `json:"new_geometry_wkt"`
	// Хилийн ТАЛБАЙ (м²) — хадгалсан геометрээс public.calculate_area_utm-ээр
	// бодогдоно (нэгж талбартай ижил аргачлал). Байршил таб дээр хилийн
	// өөрчлөлт бүрийн талбайн зөрүүг харуулахад хэрэглэгдэнэ.
	OldAreaM2 float64 `json:"old_area_m2"`
	NewAreaM2 float64 `json:"new_area_m2"`
	ChangedBy string  `json:"changed_by"`
	ChangedAt string  `json:"changed_at"`
}

// boundaryPreviewResponse — гараас оруулах хилийг ХАДГАЛАЛГҮЙ шалгасан үр дүн.
type boundaryPreviewResponse struct {
	// GeometryWKT — файлаас уншсан хил (WGS84) — газрын зураг дээр харуулна.
	GeometryWKT string `json:"geometry_wkt"`
	// AreaM2 — шинэ хилийн талбай (public.calculate_area_utm).
	AreaM2 float64 `json:"area_m2"`
	// ReferenceAreaM2 — жишиг (төлөвлөгөөний) хилийн талбай. 0 = жишиг байхгүй.
	ReferenceAreaM2 float64 `json:"reference_area_m2"`
	// DeviationPercent — |шинэ − жишиг| / жишиг × 100.
	DeviationPercent float64 `json:"deviation_percent"`
	// MaxDeviationPercent — зөвшөөрөгдөх дээд зөрүү (хувиар).
	MaxDeviationPercent float64 `json:"max_deviation_percent"`
	// Accepted — false бол хадгалах үед татгалзана.
	Accepted bool `json:"accepted"`
}

type documentResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	FileType       string `json:"file_type"`
	FileURL        string `json:"file_url"`
	SizeBytes      int64  `json:"size_bytes"`
	Note           string `json:"note"`
	UploadedBy     string `json:"uploaded_by"`
	UploadedAt     string `json:"uploaded_at"`
	DocumentTypeID int    `json:"document_type_id,omitempty"`
	// SourceDocID — эх системээс татагдсан хавсралт дээр л утгатай. Гараар
	// оруулсан хавсралтаас ялгахад хэрэглэнэ.
	SourceDocID string `json:"source_doc_id,omitempty"`
	// SourceDocCode — ГУС-ийн баримтын үүргийн код. Хавсралтыг Кадастр/
	// Төлбөр/Мониторинг гэж бүлэглэхэд хэрэглэнэ. document_type_id-тай ХАМААРАЛГҮЙ.
	SourceDocCode string `json:"source_doc_code,omitempty"`
}

func toDocumentResponse(d *domain.Document) documentResponse {
	return documentResponse{
		ID:             d.ID.String(),
		Name:           d.Name,
		FileType:       d.FileType,
		FileURL:        d.FileURL,
		SizeBytes:      d.SizeBytes,
		Note:           d.Note,
		UploadedBy:     d.UploadedBy,
		DocumentTypeID: d.DocumentTypeID,
		UploadedAt:     d.UploadedAt.Format(time.RFC3339),
		SourceDocID:    d.SourceDocID,
		SourceDocCode:  d.SourceDocCode,
	}
}

// socioSurveyResponse — нийгэм, эдийн засгийн судалгааны бүртгэл.
// FileURL нь харьцангуй зам (/api/files/...) — frontend-ийн proxy нээнэ.
type socioSurveyResponse struct {
	ID            string `json:"id"`
	AcquisitionID string `json:"acquisition_id"`
	AgreedCount   int    `json:"agreed_count"`
	RejectedCount int    `json:"rejected_count"`
	Note          string `json:"note"`
	FileURL       string `json:"file_url"`
	FileName      string `json:"file_name"`
	SizeBytes     int64  `json:"size_bytes"`
	UpdatedAt     string `json:"updated_at"`
	UpdatedBy     string `json:"updated_by"`
}

func toSocioSurveyResponse(s *domain.AcquisitionSocioSurvey) socioSurveyResponse {
	return socioSurveyResponse{
		ID:            s.ID.String(),
		AcquisitionID: s.AcquisitionID.String(),
		AgreedCount:   s.AgreedCount,
		RejectedCount: s.RejectedCount,
		Note:          s.Note,
		FileURL:       s.FileURL,
		FileName:      s.FileName,
		SizeBytes:     s.SizeBytes,
		UpdatedAt:     s.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:     s.UpdatedBy,
	}
}

// noticeEmailResponse — мэдэгдэх хуудсыг имэйлээр илгээсэн НЭГ бичлэг (түүх).
type noticeEmailResponse struct {
	ID       string `json:"id"`
	ParcelID string `json:"parcel_id"`
	ToEmail  string `json:"to_email"`
	Subject  string `json:"subject"`
	FileURL  string `json:"file_url"`
	FileName string `json:"file_name"`
	// Status — "sent" эсвэл "failed" (амжилтгүй оролдлого ч түүхэд бүртгэгдэнэ).
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	SentAt       string `json:"sent_at"`
	SentBy       string `json:"sent_by"`
}

func toNoticeEmailResponse(m domain.ParcelNoticeEmail) noticeEmailResponse {
	return noticeEmailResponse{
		ID:           m.ID.String(),
		ParcelID:     m.ParcelID.String(),
		ToEmail:      m.ToEmail,
		Subject:      m.Subject,
		FileURL:      m.FileURL,
		FileName:     m.FileName,
		Status:       m.Status,
		ErrorMessage: m.ErrorMessage,
		SentAt:       m.SentAt.Format(time.RFC3339),
		SentBy:       m.SentBy,
	}
}

// emongoliaNoticeResponse — E-Mongolia-гаар хүргүүлсэн НЭГ мэдэгдэл (түүх).
type emongoliaNoticeResponse struct {
	ID            string `json:"id"`
	ParcelID      string `json:"parcel_id"`
	RegisterNo    string `json:"register_no"`
	RecipientName string `json:"recipient_name,omitempty"`
	Subject       string `json:"subject"`
	FileURL       string `json:"file_url"`
	FileName      string `json:"file_name"`
	// Status — "sent" | "mocked" (гадаад холболтгүй түр бүртгэл) | "failed"
	Status       string `json:"status"`
	ExternalRef  string `json:"external_ref,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	SentAt       string `json:"sent_at"`
	SentBy       string `json:"sent_by"`
}

func toEMongoliaNoticeResponse(m domain.ParcelEMongoliaNotice) emongoliaNoticeResponse {
	return emongoliaNoticeResponse{
		ID:            m.ID.String(),
		ParcelID:      m.ParcelID.String(),
		RegisterNo:    m.RegisterNo,
		RecipientName: m.RecipientName,
		Subject:       m.Subject,
		FileURL:       m.FileURL,
		FileName:      m.FileName,
		Status:        m.Status,
		ExternalRef:   m.ExternalRef,
		ErrorMessage:  m.ErrorMessage,
		SentAt:        m.SentAt.Format(time.RFC3339),
		SentBy:        m.SentBy,
	}
}

// dashboardAcqResponse — dashboard-д харагдах ЗӨВХӨН шаардлагатай талбарууд
// (геометр, нэгж талбарын жагсаалт зэрэг хүнд өгөгдөл дашбоардад ашиглагдахгүй тул авахгүй).
type dashboardAcqResponse struct {
	ID              string `json:"id"`
	AcquisitionName string `json:"acquisition_name"`
	PlanCode        string `json:"plan_code"`
	Status          int    `json:"status"`
	ParcelCount     int64  `json:"parcel_count"`
	// Явц — жагсаалтын endpoint-той ИЖИЛ тооцоо (эцсийн 3 төлөв).
	FinalParcelCount int64 `json:"final_parcel_count"`
	ProgressPercent  int   `json:"progress_percent"`
}

type dashboardResponse struct {
	Acquisitions      []dashboardAcqResponse       `json:"acquisitions"`
	ParcelStatuses    []*domain.ParcelStatus       `json:"parcel_statuses"`
	TotalParcels      int64                        `json:"total_parcels"`
	FreedParcels      int64                        `json:"freed_parcels"`
	FreedAreaM2       float64                      `json:"freed_area_m2"`
	PlanAreaM2        float64                      `json:"plan_area_m2"`
	TotalOrders       int64                        `json:"total_orders"`
	TotalCompensation float64                      `json:"total_compensation"`
	StatusBreakdown   []domain.ParcelStatusStat    `json:"status_breakdown"`
	Timeline          []domain.TimelinePoint       `json:"timeline"`
	ValuationStatuses []domain.ValuationStatusStat `json:"valuation_statuses,omitempty"`
	ValuationTypes    []domain.ValuationTypeStat   `json:"valuation_types,omitempty"`
	FilteredAU1Codes  []string                     `json:"filtered_au1_codes"`
	FilteredAU2Codes  []string                     `json:"filtered_au2_codes"`
	FilteredAU3Codes  []string                     `json:"filtered_au3_codes"`
}

type globalParcelResponse struct {
	ID                string  `json:"id"`
	ParcelID          string  `json:"parcel_id"`
	AU1Code           string  `json:"au1_code"`
	AU2Code           string  `json:"au2_code"`
	AU3Code           string  `json:"au3_code"`
	RightType         int     `json:"right_type"`
	Landuse           string  `json:"landuse"`
	AreaM2            float64 `json:"area_m2"`
	AcquisitionAreaM2 float64 `json:"acquisition_area_m2"`
	CompensationPaid  bool    `json:"compensation_paid"`
	Status            int     `json:"status"`
	StatusID          int     `json:"status_id"`
	StatusName        string  `json:"status_name"`
	AcquisitionID     string  `json:"acquisition_id"`
	AcquisitionName   string  `json:"acquisition_name"`
	PlanCode          string  `json:"plan_code"`
	AcquisitionStatus int     `json:"acquisition_status"`
	StartDate         *string `json:"start_date"`
	EndDate           *string `json:"end_date"`
	CashAmount        float64 `json:"cash_amount"`
	LandGrantAmount   float64 `json:"land_grant_amount"`
	LandGrantCount    int     `json:"land_grant_count"`
}

type parcelPaymentResponse struct {
	ID        string  `json:"id"`
	ParcelID  string  `json:"parcel_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	PaidAt    *string `json:"paid_at"`
	Note      string  `json:"note"`
	CreatedAt string  `json:"created_at"`
	CreatedBy string  `json:"created_by"`
}

type assetResponse struct {
	ID            string  `json:"id"`
	AcquisitionID string  `json:"acquisition_id"`
	ParcelID      string  `json:"parcel_id"`
	AssetNumber   string  `json:"asset_number"`
	AssetType     string  `json:"asset_type"`
	ValuationType string  `json:"valuation_type"`
	AssetName     string  `json:"asset_name"`
	FloorCount    int     `json:"floor_count"`
	AreaM2        float64 `json:"area_m2"`
	OwnerName     string  `json:"owner_name"`
	Address       string  `json:"address"`
	Notes         string  `json:"notes"`
	Unit          string  `json:"unit"`
	Capacity      string  `json:"capacity"`
	Description   string  `json:"description"`
	UnitPrice     float64 `json:"unit_price"`
	PhotoPdfURL   string  `json:"photo_pdf_url"`
	PhotoPdfName  string  `json:"photo_pdf_name"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func toAssetResponse(b *domain.Asset) assetResponse {
	return assetResponse{
		ID:            b.ID.String(),
		AcquisitionID: b.AcquisitionID.String(),
		ParcelID:      b.ParcelID,
		AssetNumber:   b.AssetNumber,
		AssetType:     b.AssetType,
		ValuationType: b.ValuationType,
		AssetName:     b.AssetName,
		FloorCount:    b.FloorCount,
		AreaM2:        b.AreaM2,
		OwnerName:     b.OwnerName,
		Address:       b.Address,
		Notes:         b.Notes,
		Unit:          b.Unit,
		Capacity:      b.Capacity,
		Description:   b.Description,
		UnitPrice:     b.UnitPrice,
		PhotoPdfURL:   b.PhotoPdfURL,
		PhotoPdfName:  b.PhotoPdfName,
		CreatedAt:     b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     b.UpdatedAt.Format(time.RFC3339),
	}
}

type assetSpecResponse struct {
	ID         string `json:"id"`
	AssetID    string `json:"asset_id"`
	SpecTypeID int    `json:"spec_type_id"`
	SpecCode   string `json:"spec_code"`
	SpecName   string `json:"spec_name"`
	Value      string `json:"value"`
}

type assetCalcResponse struct {
	ID         string  `json:"id"`
	AssetID    string  `json:"asset_id"`
	CalcTypeID int     `json:"calc_type_id"`
	CalcCode   string  `json:"calc_code"`
	CalcName   string  `json:"calc_name"`
	CalcGroup  string  `json:"calc_group"`
	Unit       string  `json:"unit"`
	Value      float64 `json:"value"`
}

func toAssetSpecResponse(s *domain.AssetSpec) assetSpecResponse {
	return assetSpecResponse{
		ID: s.ID.String(), AssetID: s.AssetID.String(),
		SpecTypeID: s.SpecTypeID, SpecCode: s.SpecCode, SpecName: s.SpecName, Value: s.Value,
	}
}

func toAssetCalcResponse(c *domain.AssetCalculation) assetCalcResponse {
	return assetCalcResponse{
		ID: c.ID.String(), AssetID: c.AssetID.String(),
		CalcTypeID: c.CalcTypeID, CalcCode: c.CalcCode, CalcName: c.CalcName, CalcGroup: c.CalcGroup,
		Unit: c.Unit, Value: c.Value,
	}
}

type landValuationResponse struct {
	ID                  string  `json:"id"`
	AcquisitionID       string  `json:"acquisition_id"`
	ParcelID            string  `json:"parcel_id"`
	LandAreaM2          float64 `json:"land_area_m2"`
	BasePricePerM2      float64 `json:"base_price_per_m2"`
	TotalValue          float64 `json:"total_value"`
	OwnershipCertNo     string  `json:"ownership_cert_no"`
	AppraiserOrgName    string  `json:"appraiser_org_name"`
	AppraiserRegNo      string  `json:"appraiser_reg_no"`
	AppraiserStateRegNo string  `json:"appraiser_state_reg_no"`
	AppraiserDirector   string  `json:"appraiser_director"`
	AppraiserLicense    string  `json:"appraiser_license"`
	AppraiserAddress    string  `json:"appraiser_address"`
	AppraiserContact    string  `json:"appraiser_contact"`
	SourceFileName      string  `json:"source_file_name"`
	SourceFileHash      string  `json:"source_file_hash"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

func toLandValuationResponse(v *domain.LandValuation) landValuationResponse {
	return landValuationResponse{
		ID:                  v.ID.String(),
		AcquisitionID:       v.AcquisitionID.String(),
		ParcelID:            v.ParcelID,
		LandAreaM2:          v.LandAreaM2,
		BasePricePerM2:      v.BasePricePerM2,
		TotalValue:          v.LandAreaM2 * v.BasePricePerM2,
		OwnershipCertNo:     v.OwnershipCertNo,
		AppraiserOrgName:    v.AppraiserOrgName,
		AppraiserRegNo:      v.AppraiserRegNo,
		AppraiserStateRegNo: v.AppraiserStateRegNo,
		AppraiserDirector:   v.AppraiserDirector,
		AppraiserLicense:    v.AppraiserLicense,
		AppraiserAddress:    v.AppraiserAddress,
		AppraiserContact:    v.AppraiserContact,
		SourceFileName:      v.SourceFileName,
		SourceFileHash:      v.SourceFileHash,
		CreatedAt:           v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           v.UpdatedAt.Format(time.RFC3339),
	}
}

type compensationGrantResponse struct {
	ID             string  `json:"id"`
	AcquisitionID  string  `json:"acquisition_id"`
	CompensationID string  `json:"compensation_id"`
	Amount         float64 `json:"amount"`
	GrantDate      *string `json:"grant_date"`
	Note           string  `json:"note"`
	LandAreaM2     float64 `json:"land_area_m2"`
	LandPrice      float64 `json:"land_price"`
	LandLocation   string  `json:"land_location"`
	LandPurpose    string  `json:"land_purpose"`
	LandUseType    string  `json:"land_use_type"`
	ParcelNumber   string  `json:"parcel_number"`
	DecreeNumber   string  `json:"decree_number"`
	CreatedAt      string  `json:"created_at"`
	CreatedBy      string  `json:"created_by"`
	UpdatedAt      string  `json:"updated_at"`
}

func toCompensationGrantResponse(g *domain.CompensationGrant) compensationGrantResponse {
	r := compensationGrantResponse{
		ID:             g.ID.String(),
		AcquisitionID:  g.AcquisitionID.String(),
		CompensationID: g.CompensationID.String(),
		Amount:         g.Amount,
		Note:           g.Note,
		LandAreaM2:     g.LandAreaM2,
		LandPrice:      g.LandPrice,
		LandLocation:   g.LandLocation,
		LandPurpose:    g.LandPurpose,
		LandUseType:    g.LandUseType,
		ParcelNumber:   g.ParcelNumber,
		DecreeNumber:   g.DecreeNumber,
		CreatedAt:      g.CreatedAt.Format(time.RFC3339),
		CreatedBy:      g.CreatedBy,
		UpdatedAt:      g.UpdatedAt.Format(time.RFC3339),
	}
	if g.GrantDate != nil {
		s := g.GrantDate.Format("2006-01-02")
		r.GrantDate = &s
	}
	return r
}

type compensationResponse struct {
	ID                  string                     `json:"id"`
	AcquisitionID       string                     `json:"acquisition_id"`
	TargetType          string                     `json:"target_type"`
	ValuationType       string                     `json:"valuation_type"`
	ParcelID            string                     `json:"parcel_id"`
	AssetID             *string                    `json:"asset_id"`
	CompensationType    string                     `json:"compensation_type"`
	CoveragePercent     float64                    `json:"coverage_percent"`
	Amount              float64                    `json:"amount"`
	CompensationDate    *string                    `json:"compensation_date"`
	Note                string                     `json:"note"`
	Grant               *compensationGrantResponse `json:"grant,omitempty"`
	Status              string                     `json:"status"`
	ReviewNote          string                     `json:"review_note"`
	ReviewedBy          string                     `json:"reviewed_by"`
	ReviewedAt          *string                    `json:"reviewed_at"`
	ValuationReportURL  string                     `json:"valuation_report_url"`
	ValuationReportName string                     `json:"valuation_report_name"`
	CreatedAt           string                     `json:"created_at"`
	CreatedBy           string                     `json:"created_by"`
	UpdatedAt           string                     `json:"updated_at"`
}

func toCompensationResponse(c *domain.Compensation) compensationResponse {
	r := compensationResponse{
		ID:                  c.ID.String(),
		AcquisitionID:       c.AcquisitionID.String(),
		TargetType:          c.TargetType,
		ValuationType:       c.ValuationType,
		ParcelID:            c.ParcelID,
		CompensationType:    c.CompensationType,
		CoveragePercent:     c.CoveragePercent,
		Amount:              c.Amount,
		Note:                c.Note,
		Status:              c.Status,
		ReviewNote:          c.ReviewNote,
		ReviewedBy:          c.ReviewedBy,
		ValuationReportURL:  c.ValuationReportURL,
		ValuationReportName: c.ValuationReportName,
		CreatedAt:           c.CreatedAt.Format(time.RFC3339),
		CreatedBy:           c.CreatedBy,
		UpdatedAt:           c.UpdatedAt.Format(time.RFC3339),
	}
	if c.AssetID != nil {
		s := c.AssetID.String()
		r.AssetID = &s
	}
	if c.CompensationDate != nil {
		s := c.CompensationDate.Format("2006-01-02")
		r.CompensationDate = &s
	}
	if c.ReviewedAt != nil {
		s := c.ReviewedAt.Format(time.RFC3339)
		r.ReviewedAt = &s
	}
	if c.Grant != nil {
		gr := toCompensationGrantResponse(c.Grant)
		r.Grant = &gr
	}
	return r
}

type compensationHistoryResponse struct {
	ID               string  `json:"id"`
	CompensationID   string  `json:"compensation_id"`
	CompensationType string  `json:"compensation_type"`
	CoveragePercent  float64 `json:"coverage_percent"`
	Amount           float64 `json:"amount"`
	CompensationDate *string `json:"compensation_date"`
	Note             string  `json:"note"`
	Status           string  `json:"status"`
	ReviewNote       string  `json:"review_note"`
	ReviewedBy       string  `json:"reviewed_by"`
	ReviewedAt       *string `json:"reviewed_at"`
	ArchivedAt       string  `json:"archived_at"`
}

func toCompensationHistoryResponse(h *domain.CompensationHistory) compensationHistoryResponse {
	r := compensationHistoryResponse{
		ID:               h.ID.String(),
		CompensationID:   h.CompensationID.String(),
		CompensationType: h.CompensationType,
		CoveragePercent:  h.CoveragePercent,
		Amount:           h.Amount,
		Note:             h.Note,
		Status:           h.Status,
		ReviewNote:       h.ReviewNote,
		ReviewedBy:       h.ReviewedBy,
		ArchivedAt:       h.ArchivedAt.Format(time.RFC3339),
	}
	if h.CompensationDate != nil {
		s := h.CompensationDate.Format("2006-01-02")
		r.CompensationDate = &s
	}
	if h.ReviewedAt != nil {
		s := h.ReviewedAt.Format(time.RFC3339)
		r.ReviewedAt = &s
	}
	return r
}

func toGlobalParcelResponse(p *domain.GlobalParcel) globalParcelResponse {
	r := globalParcelResponse{
		ID:                p.ID.String(),
		ParcelID:          p.ParcelID,
		AU1Code:           p.AU1Code,
		AU2Code:           p.AU2Code,
		AU3Code:           p.AU3Code,
		RightType:         p.RightType,
		Landuse:           p.Landuse,
		AreaM2:            p.AreaM2,
		AcquisitionAreaM2: p.AcquisitionAreaM2,
		CompensationPaid:  p.CompensationPaid,
		Status:            p.StatusID,
		StatusID:          p.StatusID,
		StatusName:        p.StatusName,
		AcquisitionID:     p.AcquisitionID.String(),
		AcquisitionName:   p.AcquisitionName,
		PlanCode:          p.PlanCode,
		AcquisitionStatus: p.AcquisitionStatus,
		CashAmount:        p.CashAmount,
		LandGrantAmount:   p.LandGrantAmount,
		LandGrantCount:    p.LandGrantCount,
	}
	if p.StartDate != nil {
		s := p.StartDate.Format("2006-01-02")
		r.StartDate = &s
	}
	if p.EndDate != nil {
		e := p.EndDate.Format("2006-01-02")
		r.EndDate = &e
	}
	return r
}

func toLandAcqResponse(acq *domain.LandAcquisition) landAcquisitionResponse {
	r := landAcquisitionResponse{
		ID:                     acq.ID.String(),
		PlanCode:               acq.PlanCode,
		PlanName:               acq.PlanName,
		GeometryWKT:            acq.GeometryWKT,
		AreaM2:                 acq.AreaM2,
		Status:                 acq.Status,
		AcquisitionName:        acq.AcquisitionName,
		ImplementingOrg:        acq.ImplementingOrg,
		Reason:                 acq.Reason,
		ResponsibleOrg:         acq.ResponsibleOrg,
		FundingSource:          acq.FundingSource,
		GeneralCategoryID:      acq.GeneralCategoryID,
		GeneralCategoryName:    acq.GeneralCategoryName,
		SubCategoryID:          acq.SubCategoryID,
		SubCategoryName:        acq.SubCategoryName,
		DecreeNumber:           acq.DecreeNumber,
		CreatedAt:              acq.CreatedAt.Format(time.RFC3339),
		CreatedBy:              acq.CreatedBy,
		ParcelCount:            acq.ParcelCount,
		FinalParcelCount:       acq.FinalParcelCount,
		OverlappingParcelCount: acq.OverlappingParcelCount,
		ProgressPercent:        acq.ProgressPercent(),
		PlanParcelID:           acq.PlanParcelID,
		PlanTypeName:           acq.PlanTypeName,
		PlanGazner:             acq.PlanGazner,
		PlanAreaM2:             acq.PlanAreaM2,
		PlanStartDate:          acq.PlanStartDate,
		PlanEndDate:            acq.PlanEndDate,
		PlanApprovedDate:       acq.PlanApprovedDate,
	}
	if acq.StartDate != nil {
		s := acq.StartDate.Format("2006-01-02")
		r.StartDate = &s
	}
	if acq.EndDate != nil {
		e := acq.EndDate.Format("2006-01-02")
		r.EndDate = &e
	}
	if acq.DecreeDate != nil {
		s := acq.DecreeDate.Format("2006-01-02")
		r.DecreeDate = &s
	}
	if acq.ProfessionalOrgID != nil {
		s := strconv.FormatInt(int64(*acq.ProfessionalOrgID), 10)
		r.ProfessionalOrgID = &s
	}
	r.ProfessionalOrgName = acq.ProfessionalOrgName
	for _, au := range acq.AUs {
		r.AUs = append(r.AUs, auResponse{
			AU1Code: au.AU1Code, AU1Name: au.AU1Name,
			AU2Code: au.AU2Code, AU2Name: au.AU2Name,
			AU3Code: au.AU3Code, AU3Name: au.AU3Name,
		})
	}
	for _, p := range acq.Parcels {
		r.Parcels = append(r.Parcels, parcelResponse{
			ID: p.ID.String(), ParcelID: p.ParcelID,
			AU1Code: p.AU1Code, AU2Code: p.AU2Code, AU3Code: p.AU3Code,
			RightType: p.RightType, Landuse: p.Landuse, LanduseName: p.LanduseName, AreaM2: p.AreaM2,
			AcquisitionAreaM2: p.AcquisitionAreaM2, RemainingAreaM2: p.RemainingAreaM2,
			CompensationPaid: p.CompensationPaid, DbChanged: p.DbChanged,
			ChangedParcelID: p.ChangedParcelID,
			CashAmount:      p.CashAmount, LandGrantAmount: p.LandGrantAmount,
			LandGrantCount: p.LandGrantCount,
		})
	}
	return r
}

func toParcelResponse(p *domain.Parcel) parcelResponse {
	r := parcelResponse{
		ID:                 p.ID.String(),
		ParcelID:           p.ParcelID,
		AU1Code:            p.AU1Code,
		AU2Code:            p.AU2Code,
		AU3Code:            p.AU3Code,
		RightType:          p.RightType,
		Landuse:            p.Landuse,
		LanduseName:        p.LanduseName,
		AreaM2:             p.AreaM2,
		AcquisitionAreaM2:  p.AcquisitionAreaM2,
		RemainingAreaM2:    p.RemainingAreaM2,
		CompensationPaid:   p.CompensationPaid,
		DbChanged:          p.DbChanged,
		ChangedParcelID:    p.ChangedParcelID,
		StatusID:           p.StatusID,
		StatusName:         p.StatusName,
		GeometryWKT:        p.GeometryWKT,
		IndependentOrgName: p.IndependentOrgName,
		HolderName:         p.HolderName,
		HolderCount:        p.HolderCount,
		HasOverlap:         p.HasOverlap,
		OverlapCount:       p.OverlapCount,
		ValuationStatuses:  p.ValuationStatuses,
		CashAmount:         p.CashAmount,
		LandGrantAmount:    p.LandGrantAmount,
		LandGrantCount:     p.LandGrantCount,
	}
	if p.IndependentOrgID != nil {
		s := strconv.FormatInt(int64(*p.IndependentOrgID), 10)
		r.IndependentOrgID = &s
	}
	return r
}

func toParcelFullResponse(pf *domain.ParcelFull) parcelFullResponse {
	r := parcelFullResponse{
		Mortgages:                  pf.Mortgages,
		CourtDecisions:             pf.CourtDecisions,
		Invoices:                   pf.Invoices,
		Monitorings:                pf.Monitorings,
		Fees:                       pf.Fees,
		BasePriceFactors:           pf.BasePriceFactors,
		Ownerships:                 pf.Ownerships,
		Overlaps:                   pf.Overlaps,
		EstimatedValue:             pf.EstimatedValue,
		EstimatedConfidencePercent: pf.EstimatedConfidencePercent,
		EstimatedBaseFeePerM2:      pf.EstimatedBaseFeePerM2,
		EstimatedValueBy:           pf.EstimatedValueBy,
		EstimatedValueAt:           timePtrRFC3339(pf.EstimatedValueAt),
		ID:                         pf.ID.String(),
		ParcelID:                   pf.ParcelID,
		OldParcelID:                pf.OldParcelID,
		AcquisitionID:              pf.AcquisitionID.String(),
		AU1Code:                    pf.AU1Code,
		AU2Code:                    pf.AU2Code,
		AU3Code:                    pf.AU3Code,
		GeometryWKT:                pf.GeometryWKT,
		AcquisitionGeomWKT:         pf.AcquisitionGeomWKT,
		AcquisitionAreaM2:          pf.AcquisitionAreaM2,
		RemainingAreaM2:            pf.RemainingAreaM2,
		RightType:                  pf.RightType,
		Landuse:                    pf.Landuse,
		LanduseName:                pf.LanduseName,
		AreaM2:                     pf.AreaM2,
		AddressKhashaa:             pf.AddressKhashaa,
		AddressStreet:              pf.AddressStreet,
		PropertyNo:                 pf.PropertyNo,
		DbChanged:                  pf.DbChanged,
		ChangedParcelID:            pf.ChangedParcelID,
		StatusID:                   pf.StatusID,
		StatusName:                 pf.StatusName,
		CreatedAt:                  pf.CreatedAt.Format(time.RFC3339),
		CreatedBy:                  pf.CreatedBy,
	}
	if pf.IndependentOrgID != nil {
		s := strconv.FormatInt(int64(*pf.IndependentOrgID), 10)
		r.IndependentOrgID = &s
	}
	if pf.SelectedValuationType != nil {
		r.SelectedValuationType = pf.SelectedValuationType
	}
	if pf.ValidFrom != nil {
		s := pf.ValidFrom.Format("2006-01-02")
		r.ValidFrom = &s
	}
	if pf.ValidTill != nil {
		s := pf.ValidTill.Format("2006-01-02")
		r.ValidTill = &s
	}
	if pf.Detail != nil {
		d := pf.Detail
		di := &parcelDetailInfo{
			RightType:           d.RightType,
			HolderLastName:      d.HolderLastName,
			HolderName:          d.HolderName,
			HolderRegisterNo:    d.HolderRegisterNo,
			HolderPhone:         d.HolderPhone,
			HolderEmail:         d.HolderEmail,
			HolderType:          d.HolderType,
			HolderCivilID:       d.HolderCivilID,
			AppNo:               d.AppNo,
			AppType:             d.AppType,
			AppStatus:           d.AppStatus,
			DecisionNo:          d.DecisionNo,
			ContractNo:          d.ContractNo,
			ContractPropertyNo:  d.ContractPropertyNo,
			ContractStatus:      d.ContractStatus,
			CertificateNo:       d.CertificateNo,
			RecordNo:            d.RecordNo,
			RecordCertificateNo: d.RecordCertificateNo,
			RecordStatus:        d.RecordStatus,
		}
		if d.AppTimestamps != nil {
			s := d.AppTimestamps.Format(time.RFC3339)
			di.AppTimestamp = &s
		}
		if d.DecisionDate != nil {
			s := d.DecisionDate.Format("2006-01-02")
			di.DecisionDate = &s
		}
		if d.ContractDate != nil {
			s := d.ContractDate.Format("2006-01-02")
			di.ContractDate = &s
		}
		if d.ContractBegin != nil {
			s := d.ContractBegin.Format("2006-01-02")
			di.ContractBegin = &s
		}
		if d.ContractEnd != nil {
			s := d.ContractEnd.Format("2006-01-02")
			di.ContractEnd = &s
		}
		if d.CertificateDate != nil {
			s := d.CertificateDate.Format("2006-01-02")
			di.CertificateDate = &s
		}
		if d.RecordDate != nil {
			s := d.RecordDate.Format("2006-01-02")
			di.RecordDate = &s
		}
		di.ValuationZone = d.ValuationZone
		di.BasePricePerHa = d.BasePricePerHa
		di.AuctionCoeff = d.AuctionCoeff
		di.AuctionPrice = d.AuctionPrice
		r.Detail = di
	}
	r.Holders = make([]parcelHolderResponse, 0, len(pf.Holders))
	for _, h := range pf.Holders {
		r.Holders = append(r.Holders, toParcelHolderResponse(h))
	}
	return r
}

type reportParcelRowResponse struct {
	ParcelID            string  `json:"parcel_id"`
	AreaM2              float64 `json:"area_m2"`
	AcquisitionAreaM2   float64 `json:"acquisition_area_m2"`
	RemainingAreaM2     float64 `json:"remaining_area_m2"`
	RightType           int     `json:"right_type"`
	DbChanged           string  `json:"db_changed"`
	ChangedParcelID     string  `json:"changed_parcel_id"`
	Status              int     `json:"status"`
	StatusName          string  `json:"status_name"`
	StatusDate          *string `json:"status_date"`
	AcquisitionID       string  `json:"acquisition_id"`
	AcquisitionName     string  `json:"acquisition_name"`
	PlanCode            string  `json:"plan_code"`
	GeneralCategoryName string  `json:"general_category_name"`
	SubCategoryName     string  `json:"sub_category_name"`
	DecreeNumber        string  `json:"decree_number"`
	DecreeDate          *string `json:"decree_date"`
	HolderLastName      string  `json:"holder_last_name"`
	HolderName          string  `json:"holder_name"`
	HolderRegisterNo    string  `json:"holder_register_no"`
	LandComp            float64 `json:"land_comp"`
	RealStateComp       float64 `json:"real_state_comp"`
	PropertyComp        float64 `json:"property_comp"`
	TotalComp           float64 `json:"total_comp"`
}

type fundingSourceResponse struct {
	ID               string   `json:"id"`
	AcquisitionID    string   `json:"acquisition_id"`
	OrganizationName string   `json:"organization_name"`
	SourceType       string   `json:"source_type"`
	Amount           *float64 `json:"amount,omitempty"`
	Currency         string   `json:"currency"`
	Note             string   `json:"note"`
	CreatedAt        string   `json:"created_at"`
	CreatedBy        string   `json:"created_by"`
}

func toFundingSourceResponse(fs *domain.FundingSource) fundingSourceResponse {
	return fundingSourceResponse{
		ID:               fs.ID.String(),
		AcquisitionID:    fs.AcquisitionID.String(),
		OrganizationName: fs.OrganizationName,
		SourceType:       fs.SourceType,
		Amount:           fs.Amount,
		Currency:         fs.Currency,
		Note:             fs.Note,
		CreatedAt:        fs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy:        fs.CreatedBy,
	}
}

func toReportParcelRowResponse(r *domain.ReportParcelRow) reportParcelRowResponse {
	resp := reportParcelRowResponse{
		ParcelID:            r.ParcelID,
		AreaM2:              r.AreaM2,
		AcquisitionAreaM2:   r.AcquisitionAreaM2,
		RemainingAreaM2:     r.RemainingAreaM2,
		RightType:           r.RightType,
		DbChanged:           r.DbChanged,
		ChangedParcelID:     r.ChangedParcelID,
		Status:              r.Status,
		StatusName:          r.StatusName,
		AcquisitionID:       r.AcquisitionID.String(),
		AcquisitionName:     r.AcquisitionName,
		PlanCode:            r.PlanCode,
		GeneralCategoryName: r.GeneralCategoryName,
		SubCategoryName:     r.SubCategoryName,
		DecreeNumber:        r.DecreeNumber,
		HolderLastName:      r.HolderLastName,
		HolderName:          r.HolderName,
		HolderRegisterNo:    r.HolderRegisterNo,
		LandComp:            r.LandComp,
		RealStateComp:       r.RealStateComp,
		PropertyComp:        r.PropertyComp,
		TotalComp:           r.LandComp + r.RealStateComp + r.PropertyComp,
	}
	if r.DecreeDate != nil {
		s := r.DecreeDate.Format("2006-01-02")
		resp.DecreeDate = &s
	}
	if r.StatusDate != nil {
		s := r.StatusDate.Format("2006-01-02")
		resp.StatusDate = &s
	}
	return resp
}

type reportStatusStatResponse struct {
	Status     int    `json:"status"`
	StatusName string `json:"status_name"`
	Count      int    `json:"count"`
}

type reportYearStatResponse struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

type reportSummaryResponse struct {
	AcquisitionCount      int                        `json:"acquisition_count"`
	ParcelCount           int                        `json:"parcel_count"`
	TotalAreaM2           float64                    `json:"total_area_m2"`
	TotalParcelAreaM2     float64                    `json:"total_parcel_area_m2"`
	TotalCompensation     float64                    `json:"total_compensation"`
	LandCompensation      float64                    `json:"land_compensation"`
	RealStateCompensation float64                    `json:"real_state_compensation"`
	PropertyCompensation  float64                    `json:"property_compensation"`
	OtherCompensation     float64                    `json:"other_compensation"`
	StatusBreakdown       []reportStatusStatResponse `json:"status_breakdown"`
	YearBreakdown         []reportYearStatResponse   `json:"year_breakdown"`
}

func toReportSummaryResponse(s *domain.ReportSummary) reportSummaryResponse {
	resp := reportSummaryResponse{
		AcquisitionCount:      s.AcquisitionCount,
		ParcelCount:           s.ParcelCount,
		TotalAreaM2:           s.TotalAreaM2,
		TotalParcelAreaM2:     s.TotalParcelAreaM2,
		TotalCompensation:     s.TotalCompensation,
		LandCompensation:      s.LandCompensation,
		RealStateCompensation: s.RealStateCompensation,
		PropertyCompensation:  s.PropertyCompensation,
		OtherCompensation:     s.OtherCompensation,
		StatusBreakdown:       make([]reportStatusStatResponse, 0, len(s.StatusBreakdown)),
		YearBreakdown:         make([]reportYearStatResponse, 0, len(s.YearBreakdown)),
	}
	for _, st := range s.StatusBreakdown {
		resp.StatusBreakdown = append(resp.StatusBreakdown, reportStatusStatResponse{
			Status:     st.Status,
			StatusName: st.StatusName,
			Count:      st.Count,
		})
	}
	for _, y := range s.YearBreakdown {
		resp.YearBreakdown = append(resp.YearBreakdown, reportYearStatResponse{Year: y.Year, Count: y.Count})
	}
	return resp
}

type globalCompensationResponse struct {
	compensationResponse
	AcquisitionName  string `json:"acquisition_name"`
	HolderName       string `json:"holder_name"`
	HolderLastName   string `json:"holder_last_name"`
	HolderRegisterNo string `json:"holder_register_no"`
	HolderPhone      string `json:"holder_phone"`
}

func toGlobalCompensationResponse(gc *domain.GlobalCompensation) globalCompensationResponse {
	return globalCompensationResponse{
		compensationResponse: toCompensationResponse(&gc.Compensation),
		AcquisitionName:      gc.AcquisitionName,
		HolderName:           gc.HolderName,
		HolderLastName:       gc.HolderLastName,
		HolderRegisterNo:     gc.HolderRegisterNo,
		HolderPhone:          gc.HolderPhone,
	}
}

// timePtrRFC3339 — сонголттой хугацааг JSON-д тохирсон текст болгоно (nil → nil).
func timePtrRFC3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.Format(time.RFC3339)
	return &v
}
