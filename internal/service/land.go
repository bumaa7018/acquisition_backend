package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"path"
	"strings"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/email"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/pkg/shapefile"
	"acquisition_backend/internal/repository"

	"github.com/google/uuid"
)

type LandLookupService interface {
	ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error)
	ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error)
	CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error)
	UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error)
	DeleteAcquisitionCategory(ctx context.Context, id int) error
	ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error)
}

type LandAcquisitionService interface {
	Create(ctx context.Context, req CreateLandAcquisitionRequest) (*domain.LandAcquisition, error)
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	IsAcquisitionLocked(ctx context.Context, id uuid.UUID) (bool, error)
	// AcquisitionStatus — одоогийн төлөв (нэгж талбартай ажиллах шатыг шалгахад).
	AcquisitionStatus(ctx context.Context, id uuid.UUID) (int, error)
	IsParcelAcquisitionLocked(ctx context.Context, parcelID uuid.UUID) (bool, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, *domain.BoundaryChange, error)
	// PreviewBoundaryShapefile — гараас оруулах хилийн файлыг ХАДГАЛАЛГҮЙ
	// шалгаж, талбайн зөрүүг буцаана (хадгалахаас өмнөх урьдчилсан харагдац).
	PreviewBoundaryShapefile(ctx context.Context, id uuid.UUID, shapefileData []byte) (*BoundaryAreaCheck, error)
	// ListParcelsOutsideBoundary — өгсөн хилтэй ОГТ давхцахгүй болох бүртгэлтэй
	// нэгж талбарууд (ХАДГАЛАЛГҮЙ; хил солихын өмнөх анхааруулга).
	ListParcelsOutsideBoundary(ctx context.Context, id uuid.UUID, boundaryWKT string) ([]domain.BoundaryOutsideParcel, error)
	ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error)
	// Delete — чөлөөлөлтийг SOFT delete-ээр устгана (хязгаарлалтуудыг үз:
	// landService.Delete).
	Delete(ctx context.Context, id uuid.UUID, deletedBy string) error
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error)
	// EnsureParcelInAcquisition — parcel тухайн чөлөөлөлтөд харьяалагдаж буйг
	// шалгана (IDOR-оос сэргийлж, parcel-ийн үйлдлийг зөв acquisition-д хязгаарлана).
	EnsureParcelInAcquisition(ctx context.Context, acquisitionID, parcelID uuid.UUID) error
	UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error
	// SetParcelEstimatedValue — ТӨСӨӨЛЛИЙН үнэлгээ (nil → арилгана).
	SetParcelEstimatedValue(ctx context.Context, acquisitionID, parcelID uuid.UUID, value *float64, userID string) error
	// CalculateParcelEstimatedValue — confidencePercent nil бол итгэлцүүргүй
	// (өмчлөх эрхийн газар; эрхийн төрлөөр шийдэгдэнэ). attachment ЗААВАЛ БИШ.
	CalculateParcelEstimatedValue(ctx context.Context, acquisitionID, parcelID uuid.UUID, confidencePercent, baseFeePerM2 *float64, attachment *EstimatedValueAttachment, userID string) (float64, error)
	// ListParcelEstimatedValueHistory — төсөөллийн үнэлгээний өөрчлөлтийн түүх.
	ListParcelEstimatedValueHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID) ([]*domain.ParcelEstimatedValueHistory, error)
	SyncParcel(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) error
	DiscoverParcels(ctx context.Context, acquisitionID uuid.UUID) (*domain.ParcelDiscoveryResult, error)
	GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error)
	AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy, decreeNumber string, decreeDate *time.Time) (*domain.LandAcquisition, error)
	UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged string, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error
	// ComputeAreaM2FromWKT — WKT хилийн талбайг ГУС-тай ИЖИЛ аргаар (UTM
	// проекц, бүхэл м²) тооцоод буцаана. Frontend-ийн "Хилээс тооцоолох"
	// товч үүнийг дуудна — browser дээр өөрөө тооцвол ГУС-аас зөрнө.
	ComputeAreaM2FromWKT(ctx context.Context, geomWKT string) (float64, error)
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error)
	GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error)
	GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error)
	GetChronos(ctx context.Context, acqIDs []uuid.UUID, deptID *int) (*domain.ChronosData, error)
	Suggest(ctx context.Context, q string, status int, departmentID *int) ([]*domain.LandAcquisition, error)
	// ListFilterOptions — шүүлтүүрийн dropdown-ы хөнгөн жагсаалт (List-ийн орлуулга).
	// Бүх нэвтэрсэн хэрэглэгчид нээлттэй — шүүлтгүй.
	ListFilterOptions(ctx context.Context) ([]*domain.LandAcquisitionOption, error)
	// ListAcquisitionDistricts — шүүлтүүрийн сум/дүүргийн сонголт.
	ListAcquisitionDistricts(ctx context.Context) ([]domain.AU2Option, error)
	SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *domain.ID) error
	SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *domain.ID) error
}

type LandAssetService interface {
	// AuthorizeValuationWrite — дуудагч тухайн нэгж талбарын урсгалд бичиж
	// болох эсэх (нэгж талбар чөлөөлөлтөд харьяалагдах ёстой). Хэвийн болсон
	// урсгалыг буцаана.
	AuthorizeValuationWrite(ctx context.Context, acquisitionID uuid.UUID, parcelCode, valuationType string, caller domain.ValuationCaller) (string, error)
	// AuthorizeAssetWrite / AuthorizeCompensationWrite — мөрийн өөрийн
	// нэгж талбар/урсгалаар ижил шалгалт (URL-д зөвхөн мөрийн ID ирдэг).
	AuthorizeAssetWrite(ctx context.Context, acquisitionID, assetID uuid.UUID, caller domain.ValuationCaller) (string, error)
	AuthorizeCompensationWrite(ctx context.Context, acquisitionID, compID uuid.UUID, caller domain.ValuationCaller) (string, error)
	// AuthorizeValuationSubmit — илгээх эрх (нэгж талбар UUID-аар).
	AuthorizeValuationSubmit(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string, caller domain.ValuationCaller) (string, error)
	// ArchivedValuationFileURLs — нэгж талбарын цуцлагдсан үнэлгээний архивын файлын хаягууд.
	ArchivedValuationFileURLs(ctx context.Context, parcelID uuid.UUID) ([]string, error)
	// ValuationOrgReadScope — үнэлгээний байгууллагын УНШИЖ болох урсгалууд.
	ValuationOrgReadScope(ctx context.Context, acquisitionID uuid.UUID, orgID domain.ID) (domain.ValuationOrgScope, error)
	ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error)
	CreateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error)
	UpdateAsset(ctx context.Context, u *domain.AssetUpdate) (*domain.Asset, error)
	DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID, allowedStreams []string) error
	// GetAssetAcquisition — файлын түлхүүрийн эрх шалгахад (asset-photo/<asset_id>).
	GetAssetAcquisition(ctx context.Context, assetID uuid.UUID) (uuid.UUID, error)
	// GetCompensationAcquisition — файлын түлхүүрийн эрх шалгахад
	// (compensation-report/<comp_id>). Цуцлагдсан үнэлгээний устсан мөрийг
	// архивын snapshot-оос олдог тул GetGlobalCompensation-оос ЯЛГААТАЙ.
	GetCompensationAcquisition(ctx context.Context, compID uuid.UUID) (uuid.UUID, error)
	UploadAssetPhoto(ctx context.Context, acqID, assetID uuid.UUID, filename, contentType string, data []byte) (photoURL, photoName string, err error)
	UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error
	ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error)
	UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error
	ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error)
	ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error)
	CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error
	UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error
	DeleteAssetSpecType(ctx context.Context, id int) error
	ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error)
	CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error
	UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error
	DeleteAssetCalcType(ctx context.Context, id int) error
	UpsertLandValuation(ctx context.Context, v *domain.LandValuation) (*domain.LandValuation, error)
	GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error)
	DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error
	ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error)
	ListValuationNotes(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.ValuationSectionNote, error)
	SaveValuationNotes(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string, notes map[string]string, updatedBy string) ([]*domain.ValuationSectionNote, error)
}

type LandCompensationService interface {
	ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error)
	CreateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error)
	UpdateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error)
	DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID, allowedStreams []string) error
	ApproveCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error
	RejectCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error
	UploadCompensationReport(ctx context.Context, acqID, compID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Compensation, error)
	ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error)
	CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error)
	UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error)
	GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error)
	DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error
	DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error
	ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error)
	GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error)
}

type LandPaymentService interface {
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
}

type LandDocumentService interface {
	// GetDocumentType — хавсралт байршуулах үед зөвшөөрөгдөх файлын
	// агуулгын төрлийг (жишээ нь "meeting_minutes" бол PDF-ээс гадна DOCX)
	// тодорхойлоход ашиглана.
	GetDocumentType(ctx context.Context, id int) (*domain.DocumentType, error)
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error)
	DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error
	// Нийгэм, эдийн засгийн судалгаа (чөлөөлөлтөд НЭГ бүртгэл).
	// GetAcquisitionSocioSurvey — бүртгэл байхгүй үед (nil, nil): "хоосон"
	// нь алдаа биш, дэлгэц "оруулаагүй" гэж харуулна.
	GetAcquisitionSocioSurvey(ctx context.Context, acquisitionID uuid.UUID) (*domain.AcquisitionSocioSurvey, error)
	// SaveAcquisitionSocioSurvey — тоон дүн + судалгааны PDF-ийг хадгална.
	// data хоосон бол ХАДГАЛАГДСАН файл хэвээр үлдэнэ (шинэ бүртгэлд файл
	// ЗААВАЛ — эс бөгөөс ErrSurveyFileRequired).
	SaveAcquisitionSocioSurvey(ctx context.Context, acquisitionID uuid.UUID, in SocioSurveyInput) (*domain.AcquisitionSocioSurvey, error)
	// Урьдчилан мэдэгдэх хуудсыг имэйлээр хүргүүлэх ба илгээсэн түүх.
	// PDF-ийг дуудагч (frontend) загвараас үүсгэж дамжуулна.
	SendParcelNoticeEmail(ctx context.Context, parcelID uuid.UUID, in NoticeEmailInput) (*domain.ParcelNoticeEmail, error)
	ListParcelNoticeEmails(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelNoticeEmail, error)
	// E-Mongolia-гаар мэдэгдэл хүргүүлэх ба илгээсэн түүх (2.8.5).
	// Гадаад холболт одоогоор МОК — түүхэд status="mocked" гэж бүртгэгдэнэ.
	SendParcelEMongoliaNotice(ctx context.Context, parcelID uuid.UUID, in EMongoliaNoticeInput) (*domain.ParcelEMongoliaNotice, error)
	ListParcelEMongoliaNotices(ctx context.Context, parcelID uuid.UUID) ([]domain.ParcelEMongoliaNotice, error)
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error)
	// UploadParcelDocumentWithPayment — «Нөхөх олговор олгосон баримт»-д олголтын
	// шат (60 | 40 | 100) ба огноотой; бусад төрөлд payment нь nil байна.
	UploadParcelDocumentWithPayment(ctx context.Context, parcelID uuid.UUID, filename, displayName, contentType string, data []byte, uploadedBy string, documentTypeID int, payment *DocumentPayment) (*domain.Document, error)
	DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error
	// SyncParcelDocuments — эх системээс хавсралт татах. ЗӨВХӨН ШИНИЙГ нэмнэ.
	SyncParcelDocuments(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, roles []string) (*domain.ParcelDocumentSyncResult, error)
	// SyncParcelHolders — эзэмшигчдийг app_no-гоор ТУСАД НЬ татна.
	SyncParcelHolders(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelHolderSyncResult, error)
	// Газрын төлбөр үнэлгээний бүлгийн дуудалтууд
	SyncParcelBasePrice(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelBasePrice, error)
	SyncParcelInvoices(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) ([]*domain.ParcelInvoice, error)
	// SyncParcelFee — газрын төлбөрийн бодолт. Өмчлөлийн газарт хийгдэхгүй.
	SyncParcelFee(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelFee, error)
	SyncParcelMortgages(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelSyncCountResult, error)
	SyncParcelCourtDecisions(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelSyncCountResult, error)
	SyncParcelMonitorings(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelSyncCountResult, error)
	// SyncParcelOwnerships — УБЕГ-ийн газар өмчлөлийн бүртгэл (ХУР/XYP гарц).
	// ГУС-ийн дундын сервисээс БИШ тул app_no шаардахгүй, нэгж талбарын
	// дугаараар шууд татагдана.
	SyncParcelOwnerships(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) (*domain.ParcelSyncCountResult, error)
	// OpenParcelDocumentFile — эх системээс татсан хавсралтын файлыг нээнэ
	// (харах/татах). Дуудагч Content-ийг хаана.
	OpenParcelDocumentFile(ctx context.Context, parcelID, documentID uuid.UUID) (*domain.DocumentFile, error)
}

type LandAssigneeService interface {
	SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error
	GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error)
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID domain.ID) (bool, error)
	IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID domain.ID) (bool, error)
	AcquisitionDepartmentID(ctx context.Context, acquisitionID uuid.UUID) (*int, error)
	ParcelDepartmentID(ctx context.Context, parcelID uuid.UUID) (*int, error)
	// DecisionDraftParcelIDs — захирамжид ИДЭВХТЭЙ холбогдсон нэгж талбарууд
	// (захирамжийн хавсралт файлын эрхийг шалгахад).
	DecisionDraftParcelIDs(ctx context.Context, draftID uuid.UUID) ([]uuid.UUID, error)
	IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, orgID domain.ID) (bool, error)
	IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, orgID domain.ID) (bool, error)
	// IsProfMemberOfParcelByCode — parcel_id (код)-оор parcel-ийг олж, тухайн хэрэглэгч
	// уг acquisition-д харьяалагдах parcel-ийн гишүүн мөн эсэхийг шалгана.
	// ImportValuation зэрэг parcel_id-г JSON body-гоор хүлээн авдаг route-уудад ашиглана
	// (URL param биш тул профParcelMemberByParam middleware-ээр шалгах боломжгүй).
	IsProfMemberOfParcelByCode(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, orgID domain.ID) (bool, error)
}

type LandWorkflowService interface {
	ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error)
	// CreateParcelWorkflow — шилжилт ба түүнд ЗААВАЛ байх хавсралт (documentTypeIDs, заавал биш).
	CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int, documentTypeIDs []int) (*domain.ParcelWorkflow, error)
	DeleteParcelWorkflow(ctx context.Context, id int) error
	GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error)
	// UpdateParcelStatus — төлөв солих. Тухайн төлөвт тохируулсан заавал
	// хавсралт нэгж талбарт байхгүй бол шилжихгүй; attachment нь тэр дутуу
	// хавсралтыг нөхөж болно (бусад үед заавал биш).
	UpdateParcelStatus(ctx context.Context, ch domain.ParcelStatusChange, attachment *ParcelStatusAttachment) error
	// GetParcelStatusRules — төлөвийн дүрэм (шалтгаан заавал г.м.).
	GetParcelStatusRules(ctx context.Context, statusID int) (*domain.ParcelStatusRules, error)
	// SetParcelWorkflowRequiredDocuments — шилжилтэд ЗААВАЛ байх хавсралтыг солино.
	SetParcelWorkflowRequiredDocuments(ctx context.Context, workflowID int, documentTypeIDs []int) error
	// ParcelStatusRequirements — нэгж талбарт шаардлагатай хавсралт бүр орсон эсэх.
	ParcelStatusRequirements(ctx context.Context, acquisitionID, parcelID uuid.UUID, statusID int) ([]domain.ParcelStatusRequirement, error)
	ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error)
	GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error)
	// TransitionValuationSubmission — submit | approve | return.
	// attachment нь nil байж болно (хавсралт ЗААВАЛ БИШ) ба зөвхөн буцаалтад
	// хамаарна.
	TransitionValuationSubmission(ctx context.Context, in domain.ValuationTransition, attachment *ValuationAttachment) (*domain.ValuationSubmission, error)
	ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error)
	ListValuationSnapshots(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSnapshot, error)
	ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error)
	CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error)
	DeleteAcquisitionWorkflow(ctx context.Context, id int) error
}

// LandDroneService — дроны ортофото.
//
// Зураг нь ЗӨВХӨН файлын системд (MinIO) хадгалагдана: browser тийш шууд
// байршуулж, GeoServer тэндээс HTTP Range хүсэлтээр шууд уншина (COG). Backend
// растрыг өөр дээрээ хадгалахгүй — зөвхөн зам, хүрээ, давхаргыг зохицуулна.
type LandDroneService interface {
	ListDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error)
	// CreateDroneUploadURL — browser файлыг ШУУД файлын систем руу тавих
	// зөвшөөрөл. Байт API-аар дамжихгүй тул файлын хэмжээ хязгаарлагдахгүй.
	CreateDroneUploadURL(ctx context.Context, acquisitionID uuid.UUID, originalName string) (*filestorage.PresignedUpload, error)
	// RegisterDroneImage — шууд байршуулсан объектыг шалгаж бүртгэнэ.
	// Хоёр дахь буцаах утга нь GeoServer-т нийтлэх ҮЕД гарсан алдаа. Бүртгэл
	// бүтсэн (файл хадгалалтад бүтэн) тул error БИШ — гэхдээ шалтгааныг
	// хэрэглэгчид харуулахгүй бол зөвхөн серверийн лог дээр үлдэж, асуудлыг
	// олох боломжгүй болно.
	RegisterDroneImage(ctx context.Context, acquisitionID uuid.UUID, storedName, originalName string, uploadedBy *domain.ID) (*domain.DroneImage, error, error)
	RefreshDroneImages(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.DroneImage, error)
	DeleteDroneImage(ctx context.Context, acquisitionID, imageID uuid.UUID) error
	// DroneLayerName — frontend WMS дуудахад хэрэглэх давхаргын нэр ("land:drone")
	// DroneLayerName — зураг тус бүр өөрийн давхаргатай тул файлын нэрээс тооцно
	DroneLayerName(fileName string) string
}

type LandFundingService interface {
	ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error)
	// GetFinanceDashboard — санхүүгийн хяналтын самбар (он/дүүрэг/хороогоор).
	GetFinanceDashboard(ctx context.Context, filter domain.FinanceDashboardFilter) (*domain.FinanceDashboard, error)
	CreateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error)
	UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error)
	DeleteFundingSource(ctx context.Context, acquisitionID, id uuid.UUID) error
}

// LandAuthorizedRepresentativeService — итгэмжлэгдсэн төлөөлөгч нь эзэмшигчийн
// НЭГ ТӨРӨЛ (parcel_holder.holder_role='representative') болж хадгалагдана.
// Ингэснээр эзэмшигчидтэй нэг жагсаалтад орж, нөхөн төлбөр авагчийн сонголтод
// оролцоно. URL дахь parcel_id нь нэгж талбарын ДУГААР (код) байна.
type LandAuthorizedRepresentativeService interface {
	ListParcelRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) ([]domain.ParcelHolder, error)
	CreateParcelRepresentative(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, holder *domain.ParcelHolder) (*domain.ParcelHolder, error)
	DeleteParcelRepresentative(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, parcelHolderID uuid.UUID) error
	// SetParcelPaymentRecipient — нөхөн төлбөр хүлээн авах эзэмшигчийг сонгоно.
	SetParcelPaymentRecipient(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, parcelHolderID uuid.UUID) error
}

type LandService interface {
	LandLookupService
	LandAcquisitionService
	LandAssetService
	LandCompensationService
	LandPaymentService
	LandDocumentService
	LandAssigneeService
	LandWorkflowService
	LandFundingService
	LandDroneService
	LandAuthorizedRepresentativeService
}

type landService struct {
	landRepo repository.LandAcquisitionRepository
	files    filestorage.Storage
	// userRepo — auth DB-ийн хэрэглэгчид (мэрг. байгууллагын нэр харуулахад);
	// nil байж болно (тест) — тэр үед нэр хоосон үлдэнэ.
	userRepo repository.UserRepository
	// notifier — төлөв өөрчлөгдөх бүрд холбогдох хэрэглэгчдэд мэдэгдэл илгээнэ;
	// nil байж болно (тест) — тэр үед мэдэгдэл илгээгдэхгүй.
	notifier NotificationService
	// audit — ажилтны бизнес үйлдлийн лог. nil байж болно (тест).
	audit AuditService
	// parcelInfo — нэгж талбарын мэдээллийн ЦОРЫН ГАНЦ эх сурвалж (дундын
	// сервис). Тохируулаагүй бол мэдээлэл татахгүй, алдаа буцна.
	parcelInfo ParcelInfoClient
	// planProjects — middleware-ийн /plan/project. Төлөвлөгөөний ЦОРЫН ГАНЦ
	// эх сурвалж; эндээс татсаныг planStore-д хадгална.
	planProjects PlanProjectClient
	// planStore — MAIN DB-ийн plan хүснэгт. Дундын сервисээс татсан
	// төлөвлөгөө энд хадгалагдаж, хайлт/сонголт үүнээс уншина.
	planStore repository.PlanStoreRepository
	// drone — дроны ортофотог хадгалах хавтас + GeoServer-ийн мозайк.
	// Тохируулаагүй үед дроны зургийн үйлдэл ErrDroneNotConfigured буцаана.
	drone DroneStorage
	// sourceFiles — ЭХ СИСТЕМИЙН файл сервер. Татсан хавсралтын файл тэнд
	// байдаг тул харах/татахад эндээс уншина. nil үед (.env тохируулаагүй)
	// ErrSourceFileNotConfigured буцаана.
	sourceFiles SourceFileClient
	// ownershipInfo — УБЕГ-ийн өмчлөлийн бүртгэл (ХУР/XYP гарц). nil үед
	// (XYP_BASE_URL тохируулаагүй) өмчлөлийн мэдээлэл татах нь
	// ErrOwnershipNotConfigured буцаана.
	ownershipInfo OwnershipInfoClient
	// emongolia — E-Mongolia-ийн мэдэгдлийн гарц (одоогоор МОК). nil үед
	// ErrEMongoliaNotConfigured буцаана.
	emongolia EMongoliaClient
	// emailSender — мэдэгдэх хуудсыг иргэнд имэйлээр хүргүүлэхэд. nil эсвэл
	// тохируулаагүй үед ErrEmailNotConfigured буцаана (чимээгүй "илгээгдлээ"
	// гэж хэлэхгүй — мэдэгдэл хүргэсэн эсэх нь хуулийн хугацаа тоолоход хамаарна).
	emailSender email.Sender
}

func NewLandService(
	landRepo repository.LandAcquisitionRepository,
	files filestorage.Storage,
	userRepo repository.UserRepository,
	// extras — заавал бус dependency-үүд
	// (NotificationService, AuditService, DroneStorage).
	extras ...any,
) LandService {
	if files == nil {
		files = filestorage.DisabledStorage{}
	}
	s := &landService{landRepo: landRepo, files: files, userRepo: userRepo}
	for _, extra := range extras {
		switch v := extra.(type) {
		case NotificationService:
			s.notifier = v
		case AuditService:
			s.audit = v
		case ParcelInfoClient:
			s.parcelInfo = v
		case DroneStorage:
			s.drone = v
		case repository.PlanStoreRepository:
			s.planStore = v
		case SourceFileClient:
			s.sourceFiles = v
		case OwnershipInfoClient:
			s.ownershipInfo = v
		case email.Sender:
			s.emailSender = v
		case EMongoliaClient:
			s.emongolia = v
		}
		// PlanProjectClient-ийг ParcelInfoClient-тай ижил обьект хэрэгжүүлдэг тул
		// switch-ийн салангид case болгож биш, тусад нь шалгана.
		if v, ok := extra.(PlanProjectClient); ok {
			s.planProjects = v
		}
	}
	return s
}

// Create — газар чөлөөлөлтын мэдээлэл үүсгэх
//
// Алхам:
//  1. middleware /plan/project-ээс төлөвлөгөө татаж appdb-ийн plan хүснэгтэд хадгална
//  2. Төлөвлөгөөний нэгж талбарын ХИЛИЙГ чөлөөлөлтийн хил болгож ХУУЛНА
func (s *landService) ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error) {
	return s.landRepo.ListConstructionTypes(ctx)
}

func (s *landService) ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error) {
	return s.landRepo.ListAcquisitionCategories(ctx, parentID)
}

func (s *landService) CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error) {
	return s.landRepo.CreateAcquisitionCategory(ctx, name, parentID, sortOrder, departmentID)
}

func (s *landService) UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int, departmentID *int) (*domain.AcquisitionCategory, error) {
	return s.landRepo.UpdateAcquisitionCategory(ctx, id, name, sortOrder, departmentID)
}

func (s *landService) DeleteAcquisitionCategory(ctx context.Context, id int) error {
	return s.landRepo.DeleteAcquisitionCategory(ctx, id)
}

func (s *landService) ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error) {
	return s.landRepo.ListParcelStatuses(ctx)
}

// 3. Чөлөөлөх хилийн талбай, огтлолцох AU мэдээллийг appdb дээр тооцно
// 4. land_acquisition үүсгэнэ
// 5. Огтлолцох баг (au3) олж холбоно
//
// Нэгж талбарыг үүсгэх үед татахгүй (тусад нь татагдана).
func (s *landService) Create(ctx context.Context, req CreateLandAcquisitionRequest) (*domain.LandAcquisition, error) {
	// 1. Төлөвлөгөө хайх (middleware /plan/project) ба main DB-д хадгалах
	plan, err := s.resolvePlan(ctx, req.PlanCode)
	if err != nil {
		return nil, err
	}

	// 2. Чөлөөлөх хил = ТӨЛӨВЛӨГӨӨНИЙ хил. Гараас shapefile авахаа больсон тул
	//    хилийн цорын ганц эх сурвалж нь дундын сервисээс ирсэн төлөвлөгөө.
	acqWKT, err := planBoundaryWKT(plan)
	if err != nil {
		return nil, err
	}

	// 3. Хилийн шалгалт: хил нь НЭГ сум/дүүрэгт багтсан байх (validateBoundary).
	if err := s.validateBoundary(ctx, acqWKT); err != nil {
		return nil, err
	}
	acqAreaM2 := s.computeAreaM2(ctx, acqWKT)

	// 4. land_acquisition үүсгэх
	now := time.Now()
	var planID *uuid.UUID
	if plan.ID != uuid.Nil {
		planIDCopy := plan.ID
		planID = &planIDCopy
	}
	acq := &domain.LandAcquisition{
		ID:                uuid.New(),
		PlanID:            planID,
		PlanCode:          plan.PlanCode,
		PlanName:          plan.Name,
		PlanGeomWKT:       plan.BoundaryWKT,
		PlanAreaM2:        plan.AreaM2,
		PlanParcelID:      plan.ParcelID,
		PlanGazner:        plan.Gazner,
		GeometryWKT:       acqWKT,
		AreaM2:            acqAreaM2,
		Status:            domain.LandAcquisitionStatusNew,
		StartDate:         &req.StartDate,
		EndDate:           req.EndDate,
		AcquisitionName:   req.AcquisitionName,
		ImplementingOrg:   req.ImplementingOrg,
		Reason:            req.Reason,
		ResponsibleOrg:    req.ResponsibleOrg,
		FundingSource:     req.FundingSource,
		GeneralCategoryID: req.GeneralCategoryID,
		SubCategoryID:     req.SubCategoryID,
		NithDecreeNumber:  req.NithDecreeNumber,
		GroupListNumber:   req.GroupListNumber,
		CreatedAt:         now,
		CreatedBy:         req.CreatedBy,
		UpdatedAt:         now,
		UpdatedBy:         req.CreatedBy,
	}

	if err := s.landRepo.Create(ctx, acq); err != nil {
		return nil, domain.ErrInternalServer
	}

	_ = s.landRepo.CreateProgress(ctx, &domain.AcquisitionProgress{
		ID:            uuid.New(),
		AcquisitionID: acq.ID,
		FromStatus:    0,
		ToStatus:      domain.LandAcquisitionStatusNew,
		Note:          "Үүсгэгдсэн",
		ChangedBy:     req.CreatedBy,
		ChangedAt:     now,
	})

	// 5. Чөлөөлөлтийн ГЕОМЕТРЭЭР огтлолцох баг/хороо (au3)-г олж хадгална.
	// Аймаг/сум нь au3 мөрөөс гарна (FindIntersectingAU3-ыг үз).
	acq.AUs = s.saveIntersectingAUs(ctx, acq.ID, acqWKT)

	// Нэгж талбарыг ҮҮСГЭХ үед автоматаар ТАТАХГҮЙ. Шинэ чөлөөлөлт нэгж
	// талбаргүйгээр бүртгэгдэж, нэгж талбар нь хил шинэчлэх (Update) эсвэл
	// SyncParcel-ээр тусад нь татагдана.

	s.auditLog(ctx, "created", domain.AuditResourceAcquisition, acq.ID.String(), uuidPtr(acq.ID), nil, map[string]any{
		"name":      acq.AcquisitionName,
		"plan_code": acq.PlanCode,
		"status":    acq.Status,
	})
	return acq, nil
}

// Update — газар чөлөөлөлтийн мэдээлэл шинэчлэх.
// Хоёрдахь буцах утга нь хил солигдсон үед л nil биш.
//
// Хил солиход ЗӨВХӨН чөлөөлөлтийн хил (ба түүнээс хамаарах талбай, сум/баг,
// хилийн түүх) өөрчлөгдөнө. НЭГЖ ТАЛБАРТ ЮУ Ч ХИЙГДЭХГҮЙ: ГУС-аас татахгүй,
// бүртгэхгүй, устгахгүй, нөлөөлөлд өртсөн хил/талбайг дахин бодохгүй.
// (Төлөвлөгөөний дугаараар солиход төлөвлөгөөний хилийг дундын сервисээс/
// ГУС-аас татна; файлаар солиход ГУС дуудагдахгүй.)
// Шинэ хилээс гарах нэгж талбарыг хэрэглэгч хадгалахаас ӨМНӨ
// (boundary-preview / boundary-outside-parcels) анхааруулгаар харна.
func (s *landService) Update(ctx context.Context, id uuid.UUID, req UpdateLandAcquisitionRequest) (*domain.LandAcquisition, *domain.BoundaryChange, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, domain.ErrNotFound
	}

	geometryChanged := false
	oldGeometryWKT := acq.GeometryWKT
	var boundaryChange *domain.BoundaryChange

	// Хил солих ХОЁР зам. Аль нь ч өгөгдөөгүй бол хил хөндөгдөхгүй; хоёуланг
	// зэрэг өгвөл аль хил хүчинтэй нь тодорхойгүй тул татгалзана.
	//
	//  1. plan_parcel_id — дугаараар төлөвлөгөөг ДАХИН хайж (Create-тэй ижил
	//     урсгал), олдсон төлөвлөгөөний хилийг шинэ хил болгож хуулна.
	//     Одоогийнхтой ижил дугаар өгсөн ч дахин хуулна — дундын сервис дээр
	//     хил засагдсан үед үүгээр шинэчилнэ.
	//  2. shapefile — хээрийн хэмжилтээр засварласан хилийг ГАРААС оруулна.
	//     Төлөвлөгөөний мэдээлэл хөндөгдөхгүй; шинэ хилийн талбай нь жишиг
	//     хилийнхээс maxBoundaryAreaDeviation-ээс их зөрж болохгүй.
	//
	// Аль ч зам дараагийн шалгалтуудыг ИЖИЛ дамжина (нэг сум/дүүрэг, нэгж
	// талбарын зөрүү) — нэг зам нөгөөгөө тойрч гарах боломжгүй.
	if req.PlanCode != "" && len(req.ShapefileData) > 0 {
		return nil, nil, domain.ErrBoundarySourceConflict
	}

	var newWKT string
	var newPlan *domain.Plan
	switch {
	case req.PlanCode != "":
		plan, err := s.resolvePlan(ctx, req.PlanCode)
		if err != nil {
			return nil, nil, err
		}
		wkt, err := planBoundaryWKT(plan)
		if err != nil {
			return nil, nil, err
		}
		newPlan, newWKT = plan, wkt

	case len(req.ShapefileData) > 0:
		wkt, err := shapefile.Parse(req.ShapefileData)
		if err != nil {
			return nil, nil, shapefileError(err, "шинэ хил")
		}
		// Гараас оруулсан хил нь ЖИШИГ хилээсээ хэт зөрөөгүй байх ёстой —
		// файл нь огт өөр газрын/масштабын хил байх эрсдэлээс хамгаална.
		if err := s.validateBoundaryAreaDeviation(ctx, acq, wkt); err != nil {
			return nil, nil, err
		}
		newWKT = wkt
	}

	if newWKT != "" {
		// Хилийн шалгалт Create-тай ИЖИЛ: шинэ хил нь нэг сум/дүүрэгт багтсан
		// байх ёстой. Үгүй бол шинэчлэлт хийгдэхгүй (шалгалтыг Update-ээр
		// тойрч гарах боломжгүй).
		if err := s.validateBoundary(ctx, newWKT); err != nil {
			return nil, nil, err
		}
		newAreaM2 := s.computeAreaM2(ctx, newWKT)

		geometryChanged = true

		// Төлөвлөгөөний мэдээлэл нь ЗӨВХӨН төлөвлөгөөний дугаараар солиход
		// шинэчлэгдэнэ. Shapefile-аар хил зассан үед төлөвлөгөө хэвээр үлдэнэ —
		// эс тэгвээс дараагийн засварын жишиг (талбайн зөрүү) алдагдана.
		if newPlan != nil {
			if newPlan.ID != uuid.Nil {
				planIDCopy := newPlan.ID
				acq.PlanID = &planIDCopy
			} else {
				acq.PlanID = nil
			}
			acq.PlanCode = newPlan.PlanCode
			acq.PlanName = newPlan.Name
			acq.PlanGeomWKT = newPlan.BoundaryWKT
			acq.PlanAreaM2 = newPlan.AreaM2
			acq.PlanParcelID = newPlan.ParcelID
			acq.PlanGazner = newPlan.Gazner
		}
		acq.GeometryWKT = newWKT
		// Шинэ хилийн талбайг автоматаар тооцно (гараас оруулсныг үл хэрэгсэнэ).
		acq.AreaM2 = newAreaM2
	}

	if req.StartDate != nil {
		acq.StartDate = req.StartDate
	}
	if req.ClearEndDate {
		acq.EndDate = nil
	} else if req.EndDate != nil {
		acq.EndDate = req.EndDate
	}
	acq.AcquisitionName = req.AcquisitionName
	acq.ImplementingOrg = req.ImplementingOrg
	acq.Reason = req.Reason
	acq.ResponsibleOrg = req.ResponsibleOrg
	acq.FundingSource = req.FundingSource
	acq.GeneralCategoryID = req.GeneralCategoryID
	acq.SubCategoryID = req.SubCategoryID
	acq.NithDecreeNumber = req.NithDecreeNumber
	acq.GroupListNumber = req.GroupListNumber
	acq.UpdatedBy = req.UpdatedBy
	// Хил хөндөөгүй үед гараас оруулсан талбайн утгыг хэрэглэнэ
	if !geometryChanged && req.AreaM2 != nil {
		acq.AreaM2 = *req.AreaM2
	}

	if err := s.landRepo.Update(ctx, acq); err != nil {
		return nil, nil, domain.ErrInternalServer
	}

	if geometryChanged {
		if err := s.landRepo.CreateBoundaryHistory(ctx, &domain.LandAcquisitionBoundaryHistory{
			ID:                uuid.New(),
			LandAcquisitionID: acq.ID,
			OldGeometryWKT:    oldGeometryWKT,
			NewGeometryWKT:    acq.GeometryWKT,
			ChangedBy:         req.UpdatedBy,
			ChangedAt:         time.Now(),
		}); err != nil {
			return nil, nil, domain.ErrInternalServer
		}

		// Шинэ хилтэй огтлолцох AU-г шинэчилнэ.
		if err := s.landRepo.DeleteAUs(ctx, acq.ID); err != nil {
			return nil, nil, domain.ErrInternalServer
		}
		acq.AUs = s.saveIntersectingAUs(ctx, acq.ID, acq.GeometryWKT)

		boundaryChange = &domain.BoundaryChange{}
	}

	audit := map[string]any{
		"name":             acq.AcquisitionName,
		"plan_code":        acq.PlanCode,
		"geometry_changed": geometryChanged,
	}
	s.auditLog(ctx, "updated", domain.AuditResourceAcquisition, acq.ID.String(), uuidPtr(acq.ID), nil, audit)
	return acq, boundaryChange, nil
}

// computeAreaM2 — чөлөөлөх хилийн талбайг (м²) shapefile-ийн геометрээс
// PostGIS-аар автоматаар тооцно. Хэрэглэгч гараас оруулах шаардлагагүй.
//
// Тооцоолол бүтэлгүйтвэл 0 буцаана — талбайн утга бүртгэлийг зогсоох шалтгаан
// биш (гэхдээ шалтгааныг логлоно).
func (s *landService) computeAreaM2(ctx context.Context, geomWKT string) float64 {
	if strings.TrimSpace(geomWKT) == "" {
		return 0
	}
	areaM2, err := s.landRepo.ComputeAreaM2(ctx, geomWKT)
	if err != nil {
		slog.Error("чөлөөлөх хилийн талбай тооцоолоход алдаа", "error", err)
		return 0
	}
	return areaM2
}

// ComputeAreaM2FromWKT — дээрхтэй ижил тооцоо, гэхдээ алдааг НУУХГҮЙ.
//
// Яагаад тусад нь: computeAreaM2 нь чөлөөлөлт бүртгэх урсгалд ашиглагддаг ба
// талбай тооцох алдаа бүртгэлийг зогсоох шалтгаан биш тул 0 буцаадаг. Харин
// хэрэглэгч "Хилээс тооцоолох" дарахад 0 гэж харуулбал төөрөгдүүлнэ — алдааг
// нь ил гаргах ёстой.
func (s *landService) ComputeAreaM2FromWKT(ctx context.Context, geomWKT string) (float64, error) {
	if strings.TrimSpace(geomWKT) == "" {
		return 0, domain.ErrInvalidInput
	}
	return s.landRepo.ComputeAreaM2(ctx, geomWKT)
}

// planBoundaryWKT — чөлөөлөлтийн хил болгон хуулах ТӨЛӨВЛӨГӨӨНИЙ хил.
//
// Хил нь дундын сервисийн /plan/project-ээс (pl_project_parcel.polygon_geom)
// ирдэг ба ХООСОН байж болно. Хоосон үед чөлөөлөлтийг хилгүйгээр бүртгэвэл
// нэгж талбар татах, талбай тооцох, газрын зураг зурах бүхэн нурах тул
// шалтгааныг тодорхой заасан алдаагаар зогсооно.
func planBoundaryWKT(plan *domain.Plan) (string, error) {
	if plan == nil {
		return "", domain.ErrPlanBoundaryMissing
	}
	wkt := strings.TrimSpace(plan.BoundaryWKT)
	if wkt == "" {
		slog.Warn("төлөвлөгөөний хил хоосон тул чөлөөлөлтийн хил үүсгэх боломжгүй",
			"plan_code", plan.PlanCode, "parcel_id", plan.ParcelID)
		return "", domain.ErrPlanBoundaryMissing
	}
	return wkt, nil
}

// maxBoundaryAreaDeviation — ГАРААС (shapefile) оруулсан чөлөөлөлтийн хилийн
// талбай нь ЖИШИГ хилийн талбайгаас зөрж болох дээд хязгаар (30%).
//
// Гараас оруулах зам нь хээрийн хэмжилтээр төлөвлөгөөний хил ЗАСВАРЛАГДСАН
// тохиолдолд зориулагдсан — засвар нь хилийн хэмжээг мэдэгдэхүйц өөрчлөх
// ёсгүй. 30%-иас их зөрөх нь ихэвчлэн огт өөр талбарын, өөр масштабын эсвэл
// зөв бус проекцтой файл оруулсны шинж тул хилийг ХҮЛЭЭЖ АВАХГҮЙ.
//
// Хязгаар нь ХАТУУ: зөрүү 30%-иас БАГА байх ёстой (яг 30% ч татгалзана).
const maxBoundaryAreaDeviation = 0.30

// boundaryReferenceWKT — талбайн зөрүүг ХЭНЭЭС хэмжихийг шийднэ.
//
// Жишиг нь ТӨЛӨВЛӨГӨӨНИЙ хил: чөлөөлөлтийн хил анх түүнээс хуулагддаг ба
// гараас оруулсан хил нь түүний хээрийн засвар байх учиртай. Ингэснээр
// дараалсан олон засвараар хил "аажмаар нүүх" (жишиг нь өөрөө шилжих)
// боломжгүй болно. Төлөвлөгөөний хил хадгалагдаагүй хуучин бүртгэл дээр
// одоогийн хил рүү шилжинэ.
func boundaryReferenceWKT(acq *domain.LandAcquisition) string {
	if acq == nil {
		return ""
	}
	if wkt := strings.TrimSpace(acq.PlanGeomWKT); wkt != "" {
		return wkt
	}
	return strings.TrimSpace(acq.GeometryWKT)
}

// BoundaryAreaCheck — гараас оруулсан хилийн талбайн зөрүүний тооцоо.
// Хадгалахаас ӨМНӨ (урьдчилан харах) ба хадгалах үед ижил кодоор бодогдоно.
type BoundaryAreaCheck struct {
	// GeometryWKT — файлаас уншсан хил (WGS84).
	GeometryWKT string
	// AreaM2 — тэр хилийн талбай (public.calculate_area_utm — нэгж талбартай
	// ИЖИЛ аргачлал).
	AreaM2 float64
	// ReferenceAreaM2 — жишиг хилийн талбай (мөн ижил аргачлалаар).
	ReferenceAreaM2 float64
	// DeviationRatio — |шинэ − жишиг| / жишиг (0.15 = 15%).
	DeviationRatio float64
	// MaxDeviationRatio — зөвшөөрөгдөх дээд зөрүү (maxBoundaryAreaDeviation).
	MaxDeviationRatio float64
	// Accepted — зөрүү хязгаарт багтсан эсэх.
	Accepted bool
	// OutsideParcels — шинэ хилтэй ОГТ давхцахгүй болох (устгагдахгүй,
	// анхааруулах) нэгж талбарууд. Зөвхөн урьдчилан харахад бөглөгдөнө.
	OutsideParcels []domain.BoundaryOutsideParcel
}

// computeBoundaryAreaCheck — шинэ хил ба жишиг хилийн талбайг ИЖИЛ аргаар
// (public.calculate_area_utm — нэгж талбарын талбай тооцдогтой ижил) бодож
// зөрүүг гаргана.
//
// Талбай нь ХОЁУЛАА нэг функцээр бодогдох ёстой: өөр өөр аргаар (геодези vs
// UTM) бодвол зөрүүнд аргачлалын алдаа нэмэгдэнэ.
func (s *landService) computeBoundaryAreaCheck(
	ctx context.Context, acq *domain.LandAcquisition, newWKT string,
) (*BoundaryAreaCheck, error) {
	newArea, err := s.landRepo.ComputeAreaM2(ctx, newWKT)
	if err != nil {
		slog.Error("оруулсан хилийн талбай тооцоолоход алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	if newArea <= 0 {
		// Талбайгүй (цэг/шугам болж хураагдсан) хил — цаашид шалгах утгагүй.
		slog.Warn("оруулсан хилийн талбай тэг", "acquisition_id", acq.ID)
		return nil, domain.ErrInvalidInput
	}

	check := &BoundaryAreaCheck{
		GeometryWKT:       newWKT,
		AreaM2:            newArea,
		MaxDeviationRatio: maxBoundaryAreaDeviation,
		Accepted:          true,
	}

	refWKT := boundaryReferenceWKT(acq)
	if refWKT == "" {
		// Жишиг хил байхгүй (хилгүй бүртгэл) — зөрүү хэмжих зүйлгүй тул
		// шалгалт алгасагдана.
		slog.Warn("жишиг хил байхгүй тул талбайн зөрүүний шалгалт алгасагдав",
			"acquisition_id", acq.ID)
		return check, nil
	}
	refArea, err := s.landRepo.ComputeAreaM2(ctx, refWKT)
	if err != nil {
		slog.Error("жишиг хилийн талбай тооцоолоход алдаа", "error", err)
		return nil, domain.ErrInternalServer
	}
	if refArea <= 0 {
		slog.Warn("жишиг хилийн талбай тэг тул зөрүүний шалгалт алгасагдав",
			"acquisition_id", acq.ID)
		return check, nil
	}

	check.ReferenceAreaM2 = refArea
	check.DeviationRatio = math.Abs(newArea-refArea) / refArea
	check.Accepted = check.DeviationRatio < maxBoundaryAreaDeviation
	return check, nil
}

// validateBoundaryAreaDeviation — талбайн зөрүү хязгаарт багтаагүй бол
// domain.ErrBoundaryAreaDeviation. Хил хадгалагдахгүй.
func (s *landService) validateBoundaryAreaDeviation(
	ctx context.Context, acq *domain.LandAcquisition, newWKT string,
) error {
	check, err := s.computeBoundaryAreaCheck(ctx, acq, newWKT)
	if err != nil {
		return err
	}
	if !check.Accepted {
		slog.Warn("оруулсан хилийн талбай жишгээсээ хэт зөрсөн",
			"acquisition_id", acq.ID,
			"area_m2", check.AreaM2,
			"reference_area_m2", check.ReferenceAreaM2,
			"deviation", check.DeviationRatio)
		return domain.ErrBoundaryAreaDeviation
	}
	return nil
}

// PreviewBoundaryShapefile — гараас оруулсан хилийн файлыг ХАДГАЛАЛГҮЙ уншиж,
// талбайн зөрүү ба шинэ хилээс ГАРАХ нэгж талбаруудыг буцаана.
//
// ЯАГААД ХЭРЭГТЭЙ: хэрэглэгч хадгалахаас ӨМНӨ шинэ хилээ газрын зураг дээр
// хараад, талбайн зөрүүг тоогоор, хилээс гарах нэгж талбарыг дугаараар нь
// мэдэх ёстой (тэд устгагдахгүй ч чөлөөлөх хилийн гадна үлдэнэ). Талбайн
// тооцоо нь хадгалах замтай ЯГ ижил кодоор явагдана — дэлгэц дээрх тоо
// серверийн шийдвэрээс зөрөхгүй.
//
// Хилээс гарах талбарыг ХАДГАЛСАН геометрээр (PostGIS) тодорхойлно — ГУС руу
// хандахгүй тул хурдан. Хадгалах үед ГУС-ын хариугаар дахин тодорхойлогдож,
// хариултад дугаараараа ирнэ.
//
// Зөрүү хязгаараас хэтэрсэн ч ЭНД алдаа буцаахгүй (Accepted=false) — хэрэглэгч
// хэр их зөрснийг харах хэрэгтэй. Хадгалах үед Update татгалзана.
func (s *landService) PreviewBoundaryShapefile(
	ctx context.Context, id uuid.UUID, shapefileData []byte,
) (*BoundaryAreaCheck, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if len(shapefileData) == 0 {
		return nil, domain.ErrInvalidInput
	}
	wkt, err := shapefile.Parse(shapefileData)
	if err != nil {
		return nil, shapefileError(err, "урьдчилан харах хил")
	}
	check, err := s.computeBoundaryAreaCheck(ctx, acq, wkt)
	if err != nil {
		return nil, err
	}
	outside, err := s.landRepo.ListParcelsOutsideBoundary(ctx, id, wkt)
	if err != nil {
		// Хоосон жагсаалт буцаавал "хилээс гарах талбар алга" гэж төөрөгдүүлнэ.
		slog.Error("шинэ хилээс гарах нэгж талбарыг тодорхойлоход алдаа",
			"acquisition_id", id, "error", err)
		return nil, domain.ErrInternalServer
	}
	check.OutsideParcels = outside
	return check, nil
}

// ListParcelsOutsideBoundary — өгсөн хилтэй ОГТ давхцахгүй болох бүртгэлтэй
// нэгж талбаруудыг буцаана. Төлөвлөгөөний дугаараар хил солихын ӨМНӨ
// анхааруулга харуулахад (shapefile-ийн замд PreviewBoundaryShapefile мөн
// энэ асуулгыг ашигладаг). Юу ч хадгалахгүй, өөрчлөхгүй.
func (s *landService) ListParcelsOutsideBoundary(
	ctx context.Context, id uuid.UUID, boundaryWKT string,
) ([]domain.BoundaryOutsideParcel, error) {
	if strings.TrimSpace(boundaryWKT) == "" {
		return nil, domain.ErrInvalidInput
	}
	if _, err := s.landRepo.GetByID(ctx, id); err != nil {
		return nil, domain.ErrNotFound
	}
	outside, err := s.landRepo.ListParcelsOutsideBoundary(ctx, id, boundaryWKT)
	if err != nil {
		slog.Error("шинэ хилээс гарах нэгж талбарыг тодорхойлоход алдаа",
			"acquisition_id", id, "error", err)
		return nil, domain.ErrInternalServer
	}
	return outside, nil
}

// shapefileError — shapefile-ийн шалгалтын алдааг API-ийн domain алдаа болгож
// хөрвүүлнэ. Хэрэглэгчид ЯМАР ШАЛТГААНААР татгалзсаныг харуулах нь чухал
// (бүх алдааг нэг ерөнхий мессеж болговол файлаа юу нь болохгүй байгааг
// мэдэхгүй).
//
// Нарийн тайлбар (координат, файлын байт) нь домэйн алдаанд дамжихгүй тул
// эндээс логлоно.
func shapefileError(err error, what string) error {
	mapped := domain.ErrShapefileFormat
	switch {
	case errors.Is(err, shapefile.ErrProjection):
		mapped = domain.ErrShapefileProjection
	case errors.Is(err, shapefile.ErrOutsideCountry):
		mapped = domain.ErrShapefileOutsideCountry
	case errors.Is(err, shapefile.ErrMultipleRecords):
		mapped = domain.ErrShapefileMultiRecord
	case errors.Is(err, shapefile.ErrMultiPolygon):
		mapped = domain.ErrShapefileMultiPolygon
	case errors.Is(err, shapefile.ErrShapeType):
		mapped = domain.ErrShapefileShapeType
	}
	slog.Warn("оруулсан shapefile стандартад тохирсонгүй",
		"what", what, "reason", err.Error())
	return mapped
}

// validateBoundary — чөлөөлөх хилийг бүртгэхийн ӨМНӨ шалгана: хил нь ЗӨВХӨН
// нэг сум/дүүрэгт (au2) БҮРЭН багтах ёстой. Дамнасан эсвэл сум/дүүргийн хилээс
// гарсан бол domain.ErrAU2Crossing — хүлцэл БАЙХГҮЙ.
//
// "Төлөвлөгөөний хил дотор байх" шалгалт УСТСАН: чөлөөлөх хил нь одоо
// төлөвлөгөөний хилийн ХУУЛБАР тул өөрөөсөө гарах боломжгүй.
//
// Шалгалт бүтэлгүйтвэл бүртгэл ҮҮСЭХГҮЙ (дуудагч алдааг шууд буцаана).
func (s *landService) validateBoundary(ctx context.Context, acqWKT string) error {
	return s.validateSingleAU2(ctx, acqWKT)
}

// validateSingleAU2 — чөлөөлөх хил НЭГ сум/дүүрэгт БҮРЭН багтаж байгааг
// шалгана. Хувиар хүлцэл БАЙХГҮЙ: хилийг бүрэн бүрхсэн au2 байхгүй бол
// (хөрш сум руу хэдхэн см² оруулсан ч, эсвэл лавлахад байхгүй газар дамнасан
// ч) татгалзана.
//
// Шалгалт нь ST_CoveredBy — топологийн предикат. Тиймээс чөлөөлөх хил
// сум/дүүргийн хилийн шугамтай ЯГ давхцаж таарсан тохиолдол (хамгийн түгээмэл
// кейс) зөв "багтсан" гэж тооцогдох ба нарийвчлалын хүлцэл шаардахгүй.
//
// Шалгах өгөгдөл байхгүй үед (au2 геометр ачаалагдаагүй — scripts/
// load_admin_units.sh) бүртгэлийг ЗОГСООХГҮЙ, шалтгааныг логлож алгасна.
func (s *landService) validateSingleAU2(ctx context.Context, acqWKT string) error {
	if strings.TrimSpace(acqWKT) == "" {
		return domain.ErrInvalidInput
	}

	cov, err := s.landRepo.FindAU2Coverage(ctx, acqWKT)
	if err != nil {
		slog.Error("сум/дүүргийн (au2) давхцал шалгахад алдаа", "error", err)
		return domain.ErrInternalServer
	}
	if cov == nil || cov.ReferenceRows == 0 {
		slog.Warn("au2 лавлах геометр байхгүй тул сум/дүүргийн шалгалт алгасагдав")
		return nil
	}

	touched := make([]string, 0, len(cov.Overlaps))
	for _, o := range cov.Overlaps {
		if o.Covered {
			// Хилийг бүрэн бүрхсэн сум/дүүрэг олдлоо — дамнаагүй.
			return nil
		}
		if o.AreaM2 > 0 {
			touched = append(touched, o.Code)
		}
	}

	slog.Warn("чөлөөлөх хил сум/дүүрэг дамнасан (нэг ч au2-д бүрэн багтаагүй)",
		"au2_codes", strings.Join(touched, ","))
	return domain.ErrAU2Crossing
}

// validateParcelAcquisitionGeom — нэгж талбар дээр чөлөөлөгдөх ТАЛБАЙ/ХИЛ
// оруулахад шинэ хил нь тухайн нэгж талбарын ҮНДСЭН хил дотор 100% байхыг
// шалгана. Хувиар ХҮЛЦЭЛ БАЙХГҮЙ: үндсэн хилээс ямар нэг хэсэг гарсан бол
// domain.ErrOutsideParcelBoundary.
//
// Шалгалт нь ST_CoveredBy (топологи) тул чөлөөлөх хил нэгж талбарын хилийн
// шугамтай давхцаж таарсан (түгээмэл) кейс зөв "багтсан" гэж тооцогдоно.
//
// Талбай/хил хоёуланг ч хөндөөгүй бол шалгалт хийхгүй.
func (s *landService) validateParcelAcquisitionGeom(
	ctx context.Context,
	parcel *domain.Parcel,
	acquisitionAreaM2 *float64,
	acquisitionGeomWKT *string,
) error {
	newWKT := ""
	if acquisitionGeomWKT != nil {
		newWKT = strings.TrimSpace(*acquisitionGeomWKT)
	}
	if acquisitionAreaM2 == nil && newWKT == "" {
		return nil
	}
	if parcel == nil {
		return domain.ErrNotFound
	}

	// Чөлөөлөгдөх талбай нь нэгж талбарын үндсэн талбайгаас хэтэрч болохгүй.
	if acquisitionAreaM2 != nil && parcel.AreaM2 > 0 && *acquisitionAreaM2 > parcel.AreaM2 {
		slog.Warn("чөлөөлөгдөх талбай нэгж талбарын талбайгаас их",
			"parcel_id", parcel.ID,
			"acquisition_area_m2", *acquisitionAreaM2,
			"parcel_area_m2", parcel.AreaM2)
		return domain.ErrOutsideParcelBoundary
	}

	if newWKT == "" {
		return nil
	}
	if strings.TrimSpace(parcel.GeometryWKT) == "" {
		// Үндсэн хил татагдаагүй нэгж талбар дээр шалгах юм байхгүй.
		slog.Warn("нэгж талбарын үндсэн хил байхгүй тул хилийн шалгалт алгасагдав",
			"parcel_id", parcel.ID)
		return nil
	}

	res, err := s.landRepo.CheckContainment(ctx, newWKT, parcel.GeometryWKT)
	if err != nil {
		slog.Error("нэгж талбарын хилд багтах шалгалт хийхэд алдаа",
			"parcel_id", parcel.ID, "error", err)
		return domain.ErrInternalServer
	}
	if res == nil || !res.Covered {
		outsideM2 := 0.0
		if res != nil {
			outsideM2 = res.OutsideM2
		}
		slog.Warn("чөлөөлөх хил нэгж талбарын үндсэн хилээс гарсан",
			"parcel_id", parcel.ID, "outside_m2", outsideM2)
		return domain.ErrOutsideParcelBoundary
	}
	return nil
}

// saveIntersectingAUs — чөлөөлөлтийн ГЕОМЕТРЭЭР огтлолцох баг/хороо (au3)-г
// олж land_acquisition_au-д хадгална. Аймаг (au1) / сум-дүүрэг (au2)-ийг au3
// мөрөн дээрх код + эх хүснэгтийн нэрээр авна (FindIntersectingAU3).
//
// Чөлөөлөлт үүсэх/шинэчлэгдэхийг ЗОГСООХГҮЙ (нэмэлт мэдээлэл) — гэхдээ өмнө
// нь алдаа бүрэн залгигдаж, баг хоосон үлдсэн нь хаана ч мэдэгддэггүй байсан
// тул логлоно.
func (s *landService) saveIntersectingAUs(ctx context.Context, acquisitionID uuid.UUID, geomWKT string) []domain.LandAcquisitionAU {
	aus, err := s.landRepo.FindIntersectingAU3(ctx, geomWKT)
	if err != nil {
		slog.Error("огтлолцох au3 хайхад алдаа", "acquisition_id", acquisitionID, "error", err)
		return nil
	}
	if len(aus) == 0 {
		slog.Warn("чөлөөлөлтийн геометртэй огтлолцох au3 олдсонгүй", "acquisition_id", acquisitionID)
		return nil
	}
	saved := make([]domain.LandAcquisitionAU, 0, len(aus))
	for i := range aus {
		aus[i].LandAcquisitionID = acquisitionID
		if err := s.landRepo.CreateAU(ctx, &aus[i]); err != nil {
			slog.Error("au3 хадгалахад алдаа",
				"acquisition_id", acquisitionID, "au3_code", aus[i].AU3Code, "error", err)
			continue
		}
		saved = append(saved, aus[i])
	}
	return saved
}

// resolvePlan — төлөвлөгөөг дугаараар шийдэж, MAIN DB-ийн plan хүснэгтэд
// хадгална (чөлөөлөлт үүсгэх/хил шинэчлэх урсгалд).
//
// Эх сурвалж нь ЗӨВХӨН дундын сервис (middleware /plan/project). Тэндээс
// олдоогүй бол хадгалагдсан plan хүснэгтээс ч хайхгүй — алдаа шууд дамжиж,
// чөлөөлөлт үүсгэх урсгал тэндээ тасална.
//
// Олдсон мэдээллийг plan хүснэгтэд upsert хийж, land_acquisition.plan_id-д
// холбогдох ID-тай мөрийг буцаана.
func (s *landService) resolvePlan(ctx context.Context, planCode string) (*domain.Plan, error) {
	planCode = strings.TrimSpace(planCode)
	if planCode == "" {
		return nil, domain.ErrInvalidInput
	}

	// Төлөвлөгөөний ЦОРЫН ГАНЦ эх сурвалж нь дундын сервис. Тохируулаагүй бол
	// төсөөллийн өгөгдөл рүү унахгүй — алдаа буцаана.
	if s.planProjects == nil {
		return nil, domain.ErrInternalServer
	}
	plan, err := s.planProjects.GetPlanProject(ctx, planCode)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, domain.ErrNotFound
	}
	if plan.PlanCode == "" {
		plan.PlanCode = planCode
	}

	// appdb-д хадгалах. plan хүснэгт ЗӨВХӨН энд байх ба зөвхөн дундын
	// сервисээс татсан мэдээллийг агуулна. Upsert нь хоосон утгаар хуучныг
	// дарж бичихгүй тул хил/талбай зэрэг өмнө татагдсан мэдээлэл хадгалагдана.
	if s.planStore == nil {
		return nil, domain.ErrInternalServer
	}
	saved, err := s.planStore.Upsert(ctx, plan)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	// plan мөр нь төслийнх (plan_code) — нэгж талбар, бүтээн байгуулалт нь
	// ЭНЭ хайлтынх байх ёстой. Upsert нь хоосон gazner-ийг төслийн өөр нэгж
	// талбарынхаар нөхдөг тул ГУС-ын хариунаас шууд авна.
	saved.ParcelID = plan.ParcelID
	saved.Gazner = plan.Gazner
	return saved, nil
}

func (s *landService) ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error) {
	return s.landRepo.ListBoundaryHistory(ctx, acquisitionID)
}

// Delete — зөвхөн "NEW" статустай чөлөөлөлтийг устгана.
// Delete — чөлөөлөлтийг устгана. ХЯЗГААРЛАЛТУУД (дараалалаар):
//
//  1. Зөвхөн "Шинэ" (NEW) төлөвтэй чөлөөлөлт — domain.ErrForbidden.
//  2. Доторх нэгж талбарын төлөв: "Чөлөөлсөн" (ParcelStatusReleased) нэгж
//     талбар байвал — domain.ErrHasReleasedParcels.
//  3. Баталгаажсан захирамжтай холбоо (шууд эсвэл нэгж талбараар) байвал —
//     domain.ErrConfirmedDecisionLinked.
//
// Устгалт нь SOFT delete (deleted_at) — CASCADE-ээр 23 хүснэгт цэвэрлэгдэхээс
// сэргийлнэ. Харин ХАДГАЛАЛТ дээрх файлууд (хавсралт, дроны ортофото,
// хөрөнгийн зураг, үнэлгээний тайлан) нь зай эзэлдэг тул ФОНД, хоорондоо
// зайтай (throttle) цэвэрлэгдэнэ — тэдгээрийг эргүүлэн авах боломжгүй,
// зөвхөн бүртгэл нь хадгалагдана.
func (s *landService) Delete(ctx context.Context, id uuid.UUID, deletedBy string) error {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrNotFound
	}
	if acq.Status != domain.LandAcquisitionStatusNew {
		return domain.ErrForbidden
	}

	check, err := s.landRepo.CheckAcquisitionDeletable(ctx, id)
	if err != nil {
		slog.Error("чөлөөлөлт устгах шалгалт хийхэд алдаа", "acquisition_id", id, "error", err)
		return domain.ErrInternalServer
	}
	if check == nil {
		return domain.ErrInternalServer
	}
	if check.ReleasedParcels > 0 {
		slog.Warn("чөлөөлсөн нэгж талбартай чөлөөлөлтийг устгах оролдлого",
			"acquisition_id", id,
			"released_parcels", check.ReleasedParcels,
			"parcel_count", check.ParcelCount)
		return domain.ErrHasReleasedParcels
	}
	if check.ConfirmedDecisions > 0 {
		slog.Warn("баталгаажсан захирамжтай чөлөөлөлтийг устгах оролдлого",
			"acquisition_id", id,
			"confirmed_decisions", check.ConfirmedDecisions,
			"decree_numbers", strings.Join(check.DecisionNumbers, ","))
		return domain.ErrConfirmedDecisionLinked
	}

	// Файлын замыг устгахаас ӨМНӨ уншина (дараа нь уншвал уншилтууд
	// deleted_at-аар шүүгдэх эрсдэлтэй).
	fileURLs, err := s.landRepo.ListAcquisitionFileURLs(ctx, id)
	if err != nil {
		// Файл цэвэрлэх нь устгалыг зогсоох шалтгаан биш — логлож үргэлжилнэ.
		slog.Error("устгах файлын жагсаалт уншихад алдаа", "acquisition_id", id, "error", err)
		fileURLs = nil
	}

	if err := s.landRepo.Delete(ctx, id, deletedBy); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		slog.Error("чөлөөлөлт устгахад алдаа", "acquisition_id", id, "error", err)
		return domain.ErrInternalServer
	}

	s.cleanupFilesInBackground(id, fileURLs)

	s.auditLog(ctx, "deleted", domain.AuditResourceAcquisition, id.String(), uuidPtr(id), nil, map[string]any{
		"name":            acq.AcquisitionName,
		"plan_code":       acq.PlanCode,
		"status":          acq.Status,
		"soft_delete":     true,
		"deleted_by":      deletedBy,
		"parcel_count":    check.ParcelCount,
		"parcel_statuses": check.ParcelStatusCounts,
		"file_count":      len(fileURLs),
	})
	return nil
}

const (
	// fileCleanupDelay — файл устгах хүсэлт хоорондын зай. Нэг чөлөөлөлтөд
	// мянга мянган хавсралт байж болох тул зэрэг дуудвал хадгалалтын
	// сервисийг (MinIO/S3) дүүргэж, бодит хүсэлтүүдийг удаашруулна.
	fileCleanupDelay = 200 * time.Millisecond
	// fileCleanupTimeout — фонын цэвэрлэгээний хамгийн их хугацаа. Дуусаагүй
	// файлууд хадгалалтад үлдэнэ (логлоно) — дахин устгах нь хэвийн.
	fileCleanupTimeout = 30 * time.Minute
)

// cleanupFilesInBackground — устгасан чөлөөлөлтийн файлуудыг ФОНД, хоорондоо
// fileCleanupDelay зайтай устгана.
//
// HTTP хүсэлтийн ctx-ийг ХЭРЭГЛЭХГҮЙ (хариу буцмагц cancel болно) — тусдаа
// timeout-той context үүсгэнэ. Зэрэгцээ фон ажлын тоог backgroundWriteSem-ээр
// хязгаарлана; ээлж дүүрсэн бол цэвэрлэгээг алгасаж, логлоно (файл нь хог
// болж үлдэх нь устгалыг зогсоохоос дээр).
func (s *landService) cleanupFilesInBackground(acquisitionID uuid.UUID, fileURLs []string) {
	if len(fileURLs) == 0 || s.files == nil {
		return
	}
	release, ok := tryAcquireBackgroundSlot()
	if !ok {
		slog.Warn("фон ажлын ээлж дүүрсэн тул файл цэвэрлэгээ алгасагдав",
			"acquisition_id", acquisitionID, "file_count", len(fileURLs))
		return
	}

	go func() {
		defer release()
		ctx, cancel := context.WithTimeout(context.Background(), fileCleanupTimeout)
		defer cancel()

		deleted, failed := 0, 0
		for i, fileURL := range fileURLs {
			if i > 0 {
				// Хүсэлт хоорондын зай — хадгалалтыг дүүргэхгүй.
				select {
				case <-time.After(fileCleanupDelay):
				case <-ctx.Done():
					slog.Warn("файл цэвэрлэгээний хугацаа дууслаа",
						"acquisition_id", acquisitionID,
						"deleted", deleted, "remaining", len(fileURLs)-i)
					return
				}
			}
			if err := s.files.Delete(ctx, fileURL); err != nil {
				failed++
				slog.Warn("файл устгахад алдаа",
					"acquisition_id", acquisitionID, "file_url", fileURL, "error", err)
				continue
			}
			deleted++
		}
		slog.Info("устгасан чөлөөлөлтийн файл цэвэрлэгдлээ",
			"acquisition_id", acquisitionID, "deleted", deleted, "failed", failed)
	}()
}

func (s *landService) List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error) {
	list, total, err := s.landRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

func (s *landService) GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, 0, domain.ErrNotFound
	}
	parcels, total, err := s.landRepo.GetParcels(ctx, acquisitionID, filter)
	if err != nil {
		// ЖИНХЭНЭ шалтгааныг ЗААВАЛ логлоно: ErrInternalServer нь хүсэлтийн
		// лог дээр зөвхөн "internal server error" болж гардаг тул 500-ын
		// шалтгааныг олохын тулд өгөгдлийн сан дээр гараар хайх шаардлагатай
		// болдог байв (жишээ: NULL талбайг float64 руу scan хийж унах нь
		// ЗӨВХӨН тухайн мөр орсон хуудсанд таардаг).
		slog.Error("нэгж талбарын жагсаалт уншиж чадсангүй",
			"error", err, "acquisition_id", acquisitionID,
			"page", filter.Page, "page_size", filter.PageSize, "status_id", filter.StatusID)
		return nil, 0, domain.ErrInternalServer
	}
	return parcels, total, nil
}

func (s *landService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	// professional_org_name-г repo нь app-owned valuation_org_profile-оос
	// буцаадаг. Нэрийг хэрэглэгчийн лавлах эсвэл sdplatform-оос нөхөхгүй.
	return acq, nil
}

func (s *landService) IsAcquisitionLocked(ctx context.Context, id uuid.UUID) (bool, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return false, domain.ErrNotFound
	}
	return acq.Status == domain.LandAcquisitionStatusVerified, nil
}

// AcquisitionStatus — чөлөөлөлтийн одоогийн төлөв (middleware-ийн шатны
// шалгалтад).
func (s *landService) AcquisitionStatus(ctx context.Context, id uuid.UUID) (int, error) {
	acq, err := s.landRepo.GetByID(ctx, id)
	if err != nil {
		return 0, domain.ErrNotFound
	}
	return acq.Status, nil
}

func (s *landService) IsParcelAcquisitionLocked(ctx context.Context, parcelID uuid.UUID) (bool, error) {
	parcel, err := s.landRepo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return false, domain.ErrNotFound
	}
	return s.IsAcquisitionLocked(ctx, parcel.AcquisitionID)
}

// EnsureParcelInAcquisition — parcel тухайн чөлөөлөлтөд харьяалагдаж буйг шалгана.
// Харьяалагдахгүй бол ErrNotFound буцаана (IDOR-оос сэргийлнэ).
func (s *landService) EnsureParcelInAcquisition(ctx context.Context, acquisitionID, parcelID uuid.UUID) error {
	parcel, err := s.landRepo.GetParcelByID(ctx, parcelID)
	if err != nil {
		return domain.ErrNotFound
	}
	if parcel.AcquisitionID != acquisitionID {
		return domain.ErrNotFound
	}
	return nil
}

func (s *landService) GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	// parcel тухайн чөлөөлөлтөд харьяалагдаж буйг баталгаажуулна (IDOR).
	if err := s.EnsureParcelInAcquisition(ctx, acquisitionID, parcelID); err != nil {
		return nil, domain.ErrNotFound
	}
	pf, err := s.landRepo.GetParcelDetailByID(ctx, parcelID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	// БАЙРШЛЫН давхардал — дэлгэц дээр УЛААНААР анхааруулна. Уншилт унавал
	// дэлгэрэнгүйг бүхэлд нь хаяхгүй (давхардал нь нэмэлт мэдээлэл).
	if overlaps, err := s.landRepo.ListParcelOverlaps(ctx, parcelID); err != nil {
		slog.Warn("нэгж талбарын давхардал уншихад алдаа",
			"parcel_id", parcelID, "error", err)
	} else {
		pf.Overlaps = overlaps
	}
	return pf, nil
}

// SetParcelEstimatedValue — ТӨСӨӨЛЛИЙН үнэлгээ (мэргэжилтэн/админ гараас).
//
// value=nil → үнэлгээг АРИЛГАНА. Сөрөг утга хүлээж авахгүй.
func (s *landService) SetParcelEstimatedValue(
	ctx context.Context, acquisitionID, parcelID uuid.UUID, value *float64, userID string,
) error {
	if value != nil && (*value < 0 || math.IsNaN(*value) || math.IsInf(*value, 0)) {
		return domain.ErrInvalidInput
	}
	// Арилгалт/шууд бичилт нь ч ӨӨРЧЛӨЛТ — түүхэнд бичигдэнэ (value=nil бол
	// "арилгасан" гэж уншигдана). Хавсралт байхгүй: файл нь тооцооллын
	// гэрчилгээ тул тооцоолох урсгалд л хамаарна.
	if err := s.landRepo.SetParcelEstimatedValue(ctx, domain.ParcelEstimatedValueChange{
		AcquisitionID: acquisitionID,
		ParcelID:      parcelID,
		Value:         value,
		UserID:        userID,
	}); err != nil {
		return err
	}
	s.auditLog(ctx, "estimated_value_set", domain.AuditResourceParcel, parcelID.String(),
		uuidPtr(acquisitionID), uuidPtr(parcelID), map[string]any{
			"estimated_value": value,
			"user_id":         userID,
		})
	return nil
}

func (s *landService) UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error {
	if err := s.landRepo.UpdateParcelValuation(ctx, parcelID, zone, basePricePerHa, auctionCoeff, auctionPrice); err != nil {
		return domain.ErrInternalServer
	}
	return nil
}

// GetProgress — статус өөрчлөлтийн түүхийг буцаана
func (s *landService) GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error) {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	list, err := s.landRepo.ListProgress(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return list, nil
}

// GetAvailableStatuses — одоогийн статусаас шилжих боломжтой статусуудыг буцаана.
// DB-д ажлын урсгал тохируулсан бол тэрийг ашиглана, эс бол hard-coded эрэмбийг ашиглана.
func (s *landService) GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	rules, _ := s.landRepo.ListAcquisitionWorkflows(ctx)
	if len(rules) > 0 {
		var opts []domain.StatusOption
		for _, r := range rules {
			if r.FromStatusID != nil && *r.FromStatusID == acq.Status {
				opts = append(opts, domain.StatusOption{
					ID:    r.ToStatusID,
					Label: domain.AcquisitionStatusLabels[r.ToStatusID],
				})
			}
		}
		return opts, nil
	}

	var opts []domain.StatusOption
	for _, st := range domain.AcquisitionStatusOrder {
		if st > acq.Status {
			opts = append(opts, domain.StatusOption{
				ID:    st,
				Label: domain.AcquisitionStatusLabels[st],
			})
		}
	}
	return opts, nil
}

// AdvanceStatus — сонгосон статус руу шилжүүлнэ.
// DB-д ажлын урсгал тохируулсан бол тэрийг шалгана, эс бол hard-coded дүрмийг ашиглана.
func (s *landService) AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy, decreeNumber string, decreeDate *time.Time) (*domain.LandAcquisition, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	rules, _ := s.landRepo.ListAcquisitionWorkflows(ctx)
	if len(rules) > 0 {
		allowed := false
		for _, r := range rules {
			if r.FromStatusID != nil && *r.FromStatusID == acq.Status && r.ToStatusID == toStatus {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, domain.ErrInvalidInput
		}
	} else {
		if toStatus <= acq.Status {
			return nil, domain.ErrInvalidInput
		}
		if _, ok := domain.AcquisitionStatusLabels[toStatus]; !ok {
			return nil, domain.ErrInvalidInput
		}
	}

	// "Баталгаажсан" руу шилжихэд НЗД-ын захирамжийн дугаар/огноо ШААРДАХГҮЙ
	// (хэрэглэгчийн шийдвэр, 2026-10-29). Илгээсэн бол доор хадгална.
	//
	// "ХЭЭРИЙН СУДАЛГАА" руу шилжихэд нэгж талбарын БҮРДЭЛ зөв байх ёстой:
	// нэгж талбартай ажиллах (үнэлгээ, олговор, төлөв) энэ төлөвөөс эхэлдэг тул
	// дараа нь бүрдлийг засах нь бичигдсэн өгөгдлийг хөндөнө.
	if toStatus == domain.LandAcquisitionStatusField {
		count, err := s.landRepo.CountParcels(ctx, acquisitionID)
		if err != nil {
			return nil, domain.ErrInternalServer
		}
		if count == 0 {
			return nil, domain.ErrParcelsNotDiscovered
		}
		// Давхардлыг ДАХИН тооцно: хил/нэгж талбар сүүлд өөрчлөгдсөн байж
		// болох тул хуучин тооцоонд найдахгүй.
		overlapping, err := s.landRepo.DetectParcelOverlaps(ctx, acquisitionID)
		if err != nil {
			return nil, domain.ErrInternalServer
		}
		if overlapping > 0 {
			slog.Warn("давхардалтай нэгж талбартай тул Хээрийн судалгаа руу шилжсэнгүй",
				"acquisition_id", acquisitionID, "parcels", overlapping)
			return nil, domain.ErrParcelsOverlap
		}
	}
	if toStatus == domain.LandAcquisitionStatusVerified {
		hasNonFinalParcels, err := s.landRepo.HasNonFinalParcels(ctx, acquisitionID)
		if err != nil {
			return nil, domain.ErrInternalServer
		}
		if hasNonFinalParcels {
			return nil, domain.ErrNonFinalParcels
		}
	}

	progress := &domain.AcquisitionProgress{
		ID:            uuid.New(),
		AcquisitionID: acquisitionID,
		FromStatus:    acq.Status,
		ToStatus:      toStatus,
		Note:          note,
		ChangedBy:     changedBy,
		ChangedAt:     time.Now(),
	}
	if err := s.landRepo.CreateProgress(ctx, progress); err != nil {
		return nil, domain.ErrInternalServer
	}

	if err := s.landRepo.UpdateStatus(ctx, acquisitionID, toStatus); err != nil {
		return nil, domain.ErrInternalServer
	}

	if decreeNumber != "" && decreeDate != nil {
		if err := s.landRepo.UpdateDecree(ctx, acquisitionID, decreeNumber, decreeDate); err != nil {
			return nil, domain.ErrInternalServer
		}
		acq.DecreeNumber = decreeNumber
		acq.DecreeDate = decreeDate
	}

	// Мэдэгдэл: статус амжилттай шилжсэний дараа (энэ мөчид acq.Status
	// хуучин утгаараа тул from→to хоёуланг нь дамжуулж чадна).
	s.notifyAcquisitionStatusChanged(ctx, acq, toStatus)
	s.auditLog(ctx, "status_changed", domain.AuditResourceAcquisition, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"from_status":   acq.Status,
		"to_status":     toStatus,
		"note":          note,
		"decree_number": decreeNumber,
	})

	acq.Status = toStatus
	return acq, nil
}

// UpdateParcelMeta — нэгж талбарын мэдээллийн сангийн өөрчлөлтийн мэдээллийг шинэчлэнэ
//
// Чөлөөлөгдөх талбай/хил өөрчилж байвал шинэ хил нь нэгж талбарын ҮНДСЭН хил
// дотор байхыг ХАТУУ шалгана (validateParcelAcquisitionGeom).
func (s *landService) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged string, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
	parcel, _ := s.landRepo.GetParcelByID(ctx, parcelID)
	if err := s.validateParcelAcquisitionGeom(ctx, parcel, acquisitionAreaM2, acquisitionGeomWKT); err != nil {
		return err
	}
	if err := s.landRepo.UpdateParcelMeta(ctx, parcelID, dbChanged, changedParcelID, acquisitionAreaM2, acquisitionGeomWKT); err != nil {
		return err
	}
	acqID := uuid.Nil
	if parcel != nil {
		acqID = parcel.AcquisitionID
	}
	s.auditLog(ctx, "meta_updated", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acqID), uuidPtr(parcelID), map[string]any{
		"db_changed":          dbChanged,
		"changed_parcel_id":   changedParcelID,
		"acquisition_area_m2": acquisitionAreaM2,
		"geometry_changed":    acquisitionGeomWKT != nil,
	})
	return nil
}

// SetParcelCompensation — нэгж талбарын нөхөн төлбөрийн төлөв шинэчлэнэ
func (s *landService) SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error {
	parcel, _ := s.landRepo.GetParcelByID(ctx, parcelID)
	if err := s.landRepo.SetParcelCompensation(ctx, parcelID, paid); err != nil {
		return err
	}
	acqID := uuid.Nil
	parcelCode := ""
	if parcel != nil {
		acqID = parcel.AcquisitionID
		parcelCode = parcel.ParcelID
	}
	s.auditLog(ctx, "compensation_flag_set", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acqID), uuidPtr(parcelID), map[string]any{
		"paid":        paid,
		"parcel_code": parcelCode,
	})
	return nil
}

// ListGlobalParcels — бүх чөлөөлөлтийн нэгж талбарын нэгдсэн жагсаалт
func (s *landService) ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error) {
	list, total, err := s.landRepo.ListGlobalParcels(ctx, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

// ListReportParcels — тайлангийн нэгж талбарын жагсаалт (нөхөн төлбөр, эзэмшигч нэгтгэсэн)
func (s *landService) ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error) {
	list, total, err := s.landRepo.ListReportParcels(ctx, filter)
	if err != nil {
		return nil, 0, domain.ErrInternalServer
	}
	return list, total, nil
}

// GetReportSummary — тайлангийн жагсаалтын дээд хэсгийн статистикийн карт
// (нийт чөлөөлөлт/нэгж талбар, статус болон оны задаргаа, нийт талбай/олговор)
func (s *landService) GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error) {
	summary, err := s.landRepo.GetReportSummary(ctx, filter)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return summary, nil
}

// DiscoverParcels — чөлөөлөх ХИЛЭЭР дундын сервисээс (/parcels/by/acquisition)
// нэгж талбаруудыг 100-ийн БАГЦААР татаж, main DB-д бүртгэнэ.
//
// Багц бүр дугаар + БҮРТГЭЛИЙН баганыг (геометр, талбай, зориулалт, хаяг,
// эрхийн хугацаа) агуулна тул нэгж талбар нь тэр даруй бүрэн бүртгэгдэнэ —
// өмнө нь эдгээрийг дугаар тус бүрээр /parcel/info-аас татдаг байсан (N+1).
//
// ЭРХ ЗҮЙН дэлгэрэнгүй (хүсэлт, шийдвэр, гэрээ, улсын бүртгэл, эзэмшигч) нь
// ЭНД татагдахгүй — түүнийг дараа нь дугаар тус бүрээр SyncParcel татна.
//
// Дугаар бүрийг бүртгэхийн өмнө байгаа эсэхийг шалгадаг тул дахин дуудахад
// давхардал үүсэхгүй (repo дээрх InsertParcelCodes). ДАХИН дуудахад байгаа
// нэгж талбарын бүртгэлийн багана нь ирсэн мэдээллээр ШИНЭЧЛЭГДЭНЭ — эх
// системийн өөрчлөлт тусах ба хуучин хувилбараар зөвхөн дугаараар
// бүртгэгдсэн мөрүүд нөхөгдөнө.
func (s *landService) DiscoverParcels(ctx context.Context, acquisitionID uuid.UUID) (*domain.ParcelDiscoveryResult, error) {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if strings.TrimSpace(acq.GeometryWKT) == "" {
		// Хилгүй чөлөөлөлтөөс нэгж талбар хайх боломжгүй.
		return nil, domain.ErrInvalidInput
	}
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}

	parcels, err := s.listParcelsByAcquisitionBatched(ctx, acq.GeometryWKT)
	if err != nil {
		slog.Error("чөлөөлөх хилээр нэгж талбар татахад алдаа",
			"acquisition_id", acquisitionID, "error", err)
		return nil, domain.ErrInternalServer
	}

	res, err := s.landRepo.InsertParcelCodes(ctx, acquisitionID, parcels, acq.GeometryWKT)
	if err != nil {
		slog.Error("нэгж талбарын дугаар бүртгэхэд алдаа",
			"acquisition_id", acquisitionID, "error", err)
		// Хэсэг нь амжилттай бүртгэгдсэн бол үр дүнг буцаана — бүх багцыг
		// хаяхгүй (нэг дугаарын алдаанаас болж жагсаалт хоосон үлдэхгүй).
		if res == nil || len(res.ParcelIDs) == 0 {
			return nil, domain.ErrInternalServer
		}
	}
	if res == nil {
		res = &domain.ParcelDiscoveryResult{}
	}

	// "ШИНЭ" төлөвт хилээс ГАРСАН нэгж талбарыг ШУУД хасна.
	//
	// Хил солигдоход (Update) хуучин хилийн нэгж талбарууд үлдэж болно. "Шинэ"
	// төлөвт нэгж талбартай ажиллаж эхлээгүй тул устгах нь ямар ч үнэлгээ /
	// олговрыг хөндөхгүй — иймд баталгаажуулалт хүлээхгүй. Бусад төлөвт
	// автоматаар устгахгүй: тэнд өгөгдөл бичигдсэн байж болно (хил солих
	// урсгал нь Update дээр тусад нь шалгагддаг).
	if acq.Status == domain.LandAcquisitionStatusNew {
		if removed, err := s.landRepo.DeleteParcelsOutsideBoundary(ctx, acquisitionID); err != nil {
			slog.Error("хилээс гарсан нэгж талбарыг хасахад алдаа",
				"acquisition_id", acquisitionID, "error", err)
		} else if len(removed) > 0 {
			res.RemovedOutside = len(removed)
			slog.Info("хилд багтаагүй нэгж талбар хасагдлаа",
				"acquisition_id", acquisitionID, "parcel_ids", removed)
			s.auditLog(ctx, "parcels_pruned", domain.AuditResourceAcquisition,
				acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
					"removed":    len(removed),
					"parcel_ids": removed,
					"reason":     "хилд багтаагүй (Шинэ төлөв)",
				})
		}
	}

	// БАЙРШЛЫН ДАВХАРДЛЫГ нэгж талбар татсан мөчид тооцно — "Хээрийн судалгаа"
	// төлөв рүү шилжихэд шалгагддаг тул тэр мөчид ШИНЭ байх ёстой.
	if n, err := s.landRepo.DetectParcelOverlaps(ctx, acquisitionID); err != nil {
		slog.Error("нэгж талбарын давхардал тооцоход алдаа",
			"acquisition_id", acquisitionID, "error", err)
	} else {
		res.OverlappingParcels = n
		if n > 0 {
			slog.Warn("байршлаар давхардсан нэгж талбар олдлоо",
				"acquisition_id", acquisitionID, "parcels", n)
		}
	}
	return res, nil
}

// discoveryBatchSize — дундын сервисээс НЭГ дуудалтад авах нэгж талбарын тоо.
// Дундын сервисийн ХАТУУ дээд хязгаараас (maxParcelsByAcquisitionLimit = 200)
// БАГА байх ЁСТОЙ: эс бөгөөс
// хариулт чимээгүй хумигдаж, доорх "limit-ээс бага ирлээ = дууслаа" дүгнэлт
// эрт зогсоно.
const discoveryBatchSize = 100

// maxDiscoveryBatches — гогцооны ХАМГААЛАЛТ (100 × 200 = 20 000 нэгж талбар).
const maxDiscoveryBatches = 200

// discoveryBatchPause — багц хоорондын зай: эх системийн (ГУС) өгөгдлийн санг
// дараалсан хүнд SELECT-ээр дарахгүйн тулд.
const discoveryBatchPause = 100 * time.Millisecond

// listParcelsByAcquisitionBatched — чөлөөлөх ХИЛЭЭР давхцах нэгж талбаруудыг
// 100-ийн БАГЦААР, keyset cursor-оор (өмнөх багцын сүүлийн parcel_id) татна:
// 250 талбар = 3 дуудалт (100, 100, 50).
//
// Нэг том SELECT-ийн оронд жижиг багц явуулах нь эх системийн DB-ийн нэг
// удаагийн ачааллыг болон хариултын хэмжээг барина (мөр бүрд геометрийн WKT
// байдаг тул хариулт хурдан томордог).
//
// Зогсох нөхцөл:
//  1. Багц limit-ээс БАГА ирвэл — дууссан.
//  2. Багцаас ШИНЭ дугаар нэг ч нэмэгдээгүй бол — cursor хөдлөөгүй (cursor-ыг
//     үл хэрэгсэх хуучин дундын сервис) тул мөнхийн гогцоо гарахгүй.
//  3. maxDiscoveryBatches — хатуу тоолуур.
//
// ЭХНИЙ багц унавал алдаа буцна (юу ч бүртгэгдэхгүй). ДАРАА-гийн багц унавал
// хуримтлагдсаныг буцаана: дахин дуудахад ON CONFLICT DO NOTHING тул дутууг
// нөхөх боломжтой.
func (s *landService) listParcelsByAcquisitionBatched(
	ctx context.Context, acquisitionWKT string,
) ([]*domain.SourceParcel, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}

	var (
		after   string
		out     []*domain.SourceParcel
		seen    = map[string]struct{}{}
		batches int
	)
	for batches = 0; batches < maxDiscoveryBatches; batches++ {
		if batches > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(discoveryBatchPause):
			}
		}

		page, err := s.parcelInfo.ListParcelsByAcquisition(ctx, acquisitionWKT, after, discoveryBatchSize)
		if err != nil {
			if len(out) == 0 {
				return nil, err
			}
			slog.Error("нэгж талбарын багц татахад алдаа, хуримтлагдсанаар үргэлжилнэ",
				"batch", batches, "after", after, "collected", len(out), "error", err)
			return out, nil
		}

		added := 0
		for _, sp := range page {
			if sp == nil {
				continue
			}
			code := strings.TrimSpace(sp.ParcelID)
			if code == "" {
				continue
			}
			// Дундын сервис parcel_id-аар эрэмбэлж буцаадаг тул сүүлийн дугаар
			// нь дараагийн багцын cursor болно.
			after = code
			if _, dup := seen[code]; dup {
				continue
			}
			seen[code] = struct{}{}
			sp.ParcelID = code
			out = append(out, sp)
			added++
		}

		if len(page) < discoveryBatchSize {
			break
		}
		if added == 0 {
			slog.Warn("нэгж талбарын багц шинэ дугаар нэмсэнгүй, гогцоо зогсоно",
				"batch", batches, "after", after, "collected", len(out))
			break
		}
	}
	if batches >= maxDiscoveryBatches {
		slog.Warn("нэгж талбарын багцын хязгаарт хүрлээ",
			"batches", batches, "collected", len(out))
	}
	return out, nil
}

// SyncParcel — нэг нэгж талбарын дэлгэрэнгүй мэдээллийг дундын сервисээс татаж
// хадгална. Чөлөөлөлтийн бүстэй давхцах геометрийг PostGIS-р тооцоолно.
//
// Алхам:
//  1. /parcel/info/{parcel_id} — нэгж талбарын дэлгэрэнгүй
//  2. Дэлгэрэнгүйг хадгална (parcel + parcel_detail)
//  3. Дэлгэрэнгүй дээрх app_no-гоор /application/persons — ЭЗЭМШИГЧИД
//     (нэг хүсэлт дээр төрөл өөр олон хүн байж болно) → бүгдийг хадгална
//
// Sync нь ОРЛУУЛАХ (replace) үйлдэл: хуучин parcel_detail болон эзэмшигчдийн
// мөрүүд УСТГАГДАЖ, дундын сервисийн одоогийн мэдээллээр шинээр үүснэ. Ингэснээр
// эх системд байхаа больсон эзэмшигч/гэрээ хоцрогдож үлдэхгүй.
//
// /parcel/info нь БҮРТГЭЛИЙН баганыг (геометр, талбай, зориулалт, хаяг,
// эрхийн хугацаа) мөн буцаадаг тул нэгж талбарын ДУТУУ мэдээлэл ЭНД
// нөхөгдөнө — багцаар (discovery) бүртгэгдээгүй, эсвэл хуучин хувилбараар
// зөвхөн дугаараар бүртгэгдсэн мөрүүд бүрэн болно. Ирсэн утга л дарж бичнэ:
// хоосон/0 ирвэл хадгалагдсан утга үлдэнэ.
//
// Хэрэглэгчийн оруулсан өгөгдөл (үнэлгээ, нөхөх олговор, хөрөнгө, төлөв,
// баримт) хөндөгдөхгүй.
func (s *landService) SyncParcel(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) error {
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return domain.ErrNotFound
	}

	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return domain.ErrNotFound
	}

	if s.parcelInfo == nil {
		return domain.ErrInternalServer
	}
	src, err := s.parcelInfo.GetParcelInfo(ctx, parcelCode)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return domain.ErrInternalServer
	}

	holderID, err := s.landRepo.UpsertHolder(ctx, src)
	if err != nil {
		return domain.ErrInternalServer
	}

	if err := s.landRepo.ReplaceParcelDetail(ctx, p.ID, acq.ID, holderID, src); err != nil {
		return err
	}

	// Ирсэн геометрээр чөлөөлөгдөх талбайн хил/хэмжээ, үлдэх талбай, AU кодыг тооцно.
	geom := s.applyParcelGeometry(ctx, p.ID, src.GeometryWKT, acq.GeometryWKT)

	// ЭЗЭМШИГЧ ЭНД ТАТАГДАХГҮЙ — тусад нь SyncParcelHolders-оор татагдана
	// (дэлгэц дээр "Иргэн, хуулийн этгээдийн мэдээлэл" гэсэн тусдаа алхам).
	// Бөөн импортын урсгал сэргэхэд /sync-ийн ДАРАА /sync/holders-ыг мөн
	// дуудах ёстойг анхаарна уу.

	audit := map[string]any{
		"parcel_code": parcelCode,
		"app_no":      src.AppNo,
	}
	if geom != nil {
		audit["acquisition_area_m2"] = geom.AcquisitionAreaM2
		audit["remaining_area_m2"] = geom.RemainingAreaM2
	}
	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(), uuidPtr(acq.ID), uuidPtr(p.ID), audit)
	return nil
}

// applyParcelGeometry — эх системээс ирсэн нэгж талбарын геометрийг хадгалж,
// чөлөөлөлтийн бүстэй давхцах хэсгийг тооцно:
//
//   - чөлөөлөгдөх талбайн ХИЛ (parcel.acquisition_geom)
//   - чөлөөлөгдөх талбайн ХЭМЖЭЭ (parcel.acquisition_area_m2)
//   - үлдэх талбай (parcel.remaining_area_m2)
//   - геометрээр харьяалах аймаг/сум/багийн код
//
// ХЭВИЙН зам нь ИРСЭН геометрээр: /parcel/info геометрийг (geometry_wkt)
// discovery-тэй ижил хэллэгээр буцаадаг тул нэгж талбарын хил ЭНД нөхөгдөж,
// давхцал шинэчлэгдэнэ. Геометр ирээгүй үед (танигдахгүй WKT, эсвэл хуучин
// дундын сервис) хадгалагдсан геометрээр л давхцлыг тооцно. АУ кодын нөхөлт
// нь ХОЁР замд ч ажиллах ёстой — тэр нь хадгалагдсан геометрээс уншдаг.
//
// Тооцоолол sync-ийг ЗОГСООХГҮЙ — дэлгэрэнгүй аль хэдийн хадгалагдсан.
func (s *landService) applyParcelGeometry(ctx context.Context, parcelID uuid.UUID, parcelWKT, acqWKT string) *domain.ParcelGeometryResult {
	var result *domain.ParcelGeometryResult

	if strings.TrimSpace(parcelWKT) == "" {
		slog.Debug("геометр ирсэнгүй — хадгалагдсанаар давхцлыг тооцно",
			"parcel_id", parcelID)
		if err := s.landRepo.UpdateParcelAcquisitionGeom(ctx, parcelID, acqWKT); err != nil {
			slog.Error("чөлөөлөгдөх талбайг тооцоолоход алдаа", "parcel_id", parcelID, "error", err)
		}
	} else {
		res, err := s.landRepo.ApplyParcelGeometry(ctx, parcelID, parcelWKT, acqWKT)
		if err != nil {
			// Геометр невалид байж болно — чимээгүй үлдэхгүйн тулд логлоод,
			// ХАДГАЛАГДСАН геометрээр давхцлыг нөхөхийг оролдоно.
			slog.Error("нэгж талбарын геометрээр чөлөөлөгдөх талбайг тооцоолоход алдаа",
				"parcel_id", parcelID, "error", err)
			if err := s.landRepo.UpdateParcelAcquisitionGeom(ctx, parcelID, acqWKT); err != nil {
				slog.Error("чөлөөлөгдөх талбайг хадгалагдсан геометрээр тооцоолоход алдаа",
					"parcel_id", parcelID, "error", err)
			}
		} else {
			result = res
			if !result.HasOverlap {
				slog.Warn("нэгж талбар чөлөөлөлтийн бүстэй давхцахгүй",
					"parcel_id", parcelID, "parcel_area_m2", result.ParcelAreaM2)
			}
		}
	}

	// Аймаг/сум/баг — эх систем зөвхөн au2-ыг буцаадаг тул геометрээс нөхнө.
	if au, err := s.landRepo.FillParcelAUCodesFromGeometry(ctx, parcelID); err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.Error("нэгж талбарын АУ кодыг тооцоолоход алдаа", "parcel_id", parcelID, "error", err)
		} else {
			slog.Warn("нэгж талбарын геометртэй огтлолцох баг/хороо олдсонгүй", "parcel_id", parcelID)
		}
	} else {
		slog.Debug("нэгж талбарын АУ код тодорхойлогдов", "parcel_id", parcelID,
			"au1", au.AU1Code, "au2", au.AU2Code, "au3", au.AU3Code)
	}
	return result
}

// SyncParcelHolders — нэгж талбарын эзэмшигчдийг (иргэн, хуулийн этгээд)
// хадгалагдсан app_no-гоор ГУС-аас татаж шинэчилнэ.
//
// SyncParcel-аас ТУСАД НЬ дуудагдана: хэрэглэгч зөвхөн эзэмшигчийг шинэчлэх
// боломжтой байх ёстой. app_no нь /sync-ээр хадгалагддаг тул түүнийг эхлээд
// ажиллуулсан байх шаардлагатай (эс бөгөөс 422).
func (s *landService) SyncParcelHolders(
	ctx context.Context,
	acquisitionID uuid.UUID,
	parcelCode string,
) (*domain.ParcelHolderSyncResult, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	acq, err := s.landRepo.GetByID(ctx, acquisitionID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	full, err := s.landRepo.GetParcelDetailByID(ctx, p.ID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var appNo string
	if full.Detail != nil {
		appNo = strings.TrimSpace(full.Detail.AppNo)
	}
	if appNo == "" {
		slog.Warn("нэгж талбарт app_no байхгүй тул эзэмшигч татагдсангүй",
			"parcel_id", p.ID, "parcel_code", parcelCode)
		return nil, domain.ErrParcelAppNoMissing
	}

	holders, err := s.parcelInfo.ListApplicationPersons(ctx, appNo)
	if err != nil {
		slog.Error("эзэмшигчийн мэдээлэл татахад алдаа",
			"parcel_id", p.ID, "app_no", appNo, "error", err)
		return nil, domain.ErrInternalServer
	}

	found := len(holders)
	// Регистр давхцсаныг хадгалахгүй алгасна (нэг хүн хэд хэдэн үүрэгтэй ирж болно).
	holders, skippedRegisterNos := dedupeHoldersByRegisterNo(holders)
	for _, h := range holders {
		if h != nil && h.AppNo == "" {
			h.AppNo = appNo
		}
	}

	mainHolderID, err := s.landRepo.ReplaceParcelHolders(ctx, p.ID, acq.ID, appNo, holders)
	if err != nil {
		slog.Error("эзэмшигчдийг хадгалахад алдаа",
			"parcel_id", p.ID, "count", len(holders), "error", err)
		return nil, domain.ErrInternalServer
	}
	if err := s.landRepo.SetParcelDetailHolder(ctx, p.ID, mainHolderID); err != nil {
		slog.Error("үндсэн эзэмшигчийг холбоход алдаа",
			"parcel_id", p.ID, "holder_id", mainHolderID, "error", err)
	}

	s.auditLog(ctx, "synced", domain.AuditResourceParcel, p.ID.String(),
		uuidPtr(acq.ID), uuidPtr(p.ID), map[string]any{
			"parcel_code":  parcelCode,
			"app_no":       appNo,
			"holder_count": len(holders),
		})

	return &domain.ParcelHolderSyncResult{
		Found:   found,
		Saved:   len(holders),
		Skipped: len(skippedRegisterNos),
	}, nil
}

// SyncParcelDocuments — нэгж талбарын ХАВСРАЛТЫГ эх системээс татна.
//
// Хүсэлтийн дугаар (app_no) нь нэгж талбарын дэлгэрэнгүйгээс авагдана —
// эхлээд SyncParcel хийгдсэн байх ёстой.
//
// ЗӨВХӨН НЭМЭХ үйлдэл: эх системд байгаа боловч энд байхгүй баримт л шинээр
// бүртгэгдэнэ. Гараар оруулсан болон өмнө татсан хавсралт УСТГАГДАХГҮЙ,
// ДАРАГДАХГҮЙ — тиймээс дахин дуудахад аюулгүй (идемпотент).
//
// roles — баримтын үүргийн код (doc_code). Хоосон бол бүх хавсралт татагдана.
func (s *landService) SyncParcelDocuments(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, roles []string) (*domain.ParcelDocumentSyncResult, error) {
	if s.parcelInfo == nil {
		return nil, domain.ErrInternalServer
	}
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.resolveParcelRow(ctx, acquisitionID, parcelCode)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	full, err := s.landRepo.GetParcelDetailByID(ctx, p.ID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var appNo string
	if full.Detail != nil {
		appNo = strings.TrimSpace(full.Detail.AppNo)
	}
	if appNo == "" {
		// Хүсэлтийн дугааргүй бол ямар хавсралт татахыг тодорхойлох аргагүй.
		slog.Warn("нэгж талбарт app_no байхгүй тул хавсралт татагдсангүй",
			"parcel_id", p.ID, "parcel_code", parcelCode)
		return nil, domain.ErrParcelAppNoMissing
	}

	srcDocs, err := s.parcelInfo.ListApplicationDocuments(ctx, appNo, roles)
	if err != nil {
		slog.Error("хавсралт татахад алдаа",
			"parcel_id", p.ID, "app_no", appNo, "error", err)
		return nil, domain.ErrInternalServer
	}

	docs := make([]*domain.Document, 0, len(srcDocs))
	now := time.Now()
	for _, src := range srcDocs {
		if src == nil {
			continue
		}
		docs = append(docs, &domain.Document{
			ID:            uuid.New(),
			Name:          sourceDocumentName(src),
			FileType:      fileTypeFromURL(src.FileURL),
			FileURL:       strings.TrimSpace(src.FileURL),
			UploadedBy:    "system",
			UploadedAt:    now,
			SourceDocID:   src.DocID,
			SourceFtpID:   src.FtpID,
			SourceDocCode: strings.TrimSpace(src.DocCode),
		})
	}

	saved, err := s.landRepo.AddSourceParcelDocuments(ctx, p.ID, docs)
	if err != nil {
		slog.Error("хавсралт хадгалахад алдаа",
			"parcel_id", p.ID, "app_no", appNo, "count", len(docs), "error", err)
		return nil, domain.ErrInternalServer
	}

	list, err := s.landRepo.ListParcelDocuments(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	s.auditLog(ctx, "synced", domain.AuditResourceDocument, p.ID.String(),
		uuidPtr(acquisitionID), uuidPtr(p.ID), map[string]any{
			"parcel_code": parcelCode,
			"app_no":      appNo,
			"roles":       roles,
			"found":       len(docs),
			"saved":       saved,
		})

	return &domain.ParcelDocumentSyncResult{
		Found:     len(docs),
		Saved:     saved,
		Skipped:   len(docs) - saved,
		Documents: list,
	}, nil
}

// OpenParcelDocumentFile — эх системээс татсан хавсралтын ФАЙЛЫГ нээнэ.
//
// Файл нь эх системийн файл сервер дээр байдаг (энд зөвхөн зам + ftp_id
// хадгалагдана) тул API нь тэндээс уншиж дамжуулна. Дуудагч Content-ийг
// ХААХ хариуцлагатай.
//
// Зөвхөн ЭХ СИСТЕМИЙН хавсралтад зориулагдсан: гараар оруулсан файл нь
// файлын систем (MinIO) дээр байдаг ба frontend-ийн /api/files-ээр тараагдана.
func (s *landService) OpenParcelDocumentFile(ctx context.Context, parcelID, documentID uuid.UUID) (*domain.DocumentFile, error) {
	if s.sourceFiles == nil {
		return nil, domain.ErrSourceFileNotConfigured
	}

	doc, err := s.landRepo.GetParcelDocument(ctx, parcelID, documentID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.SourceDocID) == "" {
		// Эх системийнх биш — энэ зам түүнд зориулагдаагүй.
		return nil, domain.ErrNotFound
	}

	src, err := s.sourceFiles.Open(ctx, doc.SourceFtpID, doc.FileURL)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalidInput) ||
			errors.Is(err, domain.ErrSourceFileNotConfigured) {
			return nil, err
		}
		slog.Error("эх системээс хавсралт унших боломжгүй",
			"parcel_id", parcelID, "document_id", documentID,
			"ftp_id", doc.SourceFtpID, "error", err)
		return nil, domain.ErrInternalServer
	}

	return &domain.DocumentFile{
		Name:        downloadFileName(doc),
		ContentType: src.ContentType,
		SizeBytes:   src.SizeBytes,
		Content:     src.Content,
	}, nil
}

// downloadFileName — татахад харагдах нэр: баримтын нэр + эх файлын өргөтгөл.
// Эх зам дахь нэр нь hash (…-6a7e697d329b7.pdf) тул хэрэглэгчид ойлгомжгүй.
func downloadFileName(doc *domain.Document) string {
	name := strings.TrimSpace(doc.Name)
	if name == "" {
		name = "document"
	}
	// Файлын нэрэнд орох боломжгүй тэмдэгтүүдийг цэвэрлэнэ.
	name = strings.NewReplacer("/", "-", "\\", "-", "\"", "", "\r", "", "\n", "").Replace(name)

	ext := strings.TrimSpace(doc.FileType)
	if ext == "" {
		ext = fileTypeFromURL(doc.FileURL)
	}
	if ext == "" || strings.HasSuffix(strings.ToLower(name), "."+strings.ToLower(ext)) {
		return name
	}
	return name + "." + ext
}

// sourceDocumentName — харагдах нэр: үүргийн тайлбар, байхгүй бол код.
func sourceDocumentName(src *domain.SourceDocument) string {
	if name := strings.TrimSpace(src.DocName); name != "" {
		return name
	}
	if code := strings.TrimSpace(src.DocCode); code != "" {
		return "Хавсралт " + code
	}
	return "Хавсралт"
}

// fileTypeFromURL — файлын өргөтгөл. Тодорхойгүй бол хоосон (баганад NOT NULL
// хязгаар байхгүй ч хоосон мөр эмх цэгцтэй).
func fileTypeFromURL(fileURL string) string {
	fileURL = strings.TrimSpace(fileURL)
	if i := strings.IndexAny(fileURL, "?#"); i >= 0 {
		fileURL = fileURL[:i]
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(fileURL)), ".")
	if len(ext) > 10 {
		return ""
	}
	return ext
}

// dedupeHoldersByRegisterNo — регистрийн дугаараар давхцсан эзэмшигчийг
// алгасна: эхний тохиолдол хадгалагдаж, дараачийнх нь хаягдана.
//
// Эрэмбэ хадгалагдана (үндсэн хүсэлт гаргагч эхэнд ирдэг тул давхцалд
// ҮНДСЭН нь үлдэнэ). Регистргүй хүнийг давхцал гэж үзэхгүй — тэднийг
// нэр+утсаар тааруулна.
//
// Хоёрдахь буцах утга нь алгасагдсан регистрийн дугаарууд (логлоход).
func dedupeHoldersByRegisterNo(holders []*domain.ParcelHolder) ([]*domain.ParcelHolder, []string) {
	kept := make([]*domain.ParcelHolder, 0, len(holders))
	var skipped []string
	seen := make(map[string]struct{}, len(holders))

	for _, h := range holders {
		if h == nil {
			continue
		}
		registerNo := strings.ToUpper(strings.TrimSpace(h.RegisterNo))
		if registerNo != "" {
			if _, dup := seen[registerNo]; dup {
				skipped = append(skipped, h.RegisterNo)
				continue
			}
			seen[registerNo] = struct{}{}
		}
		kept = append(kept, h)
	}
	return kept, skipped
}

func (s *landService) GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	return s.landRepo.GetDashboard(ctx, filter)
}

func (s *landService) GetChronos(ctx context.Context, acqIDs []uuid.UUID, deptID *int) (*domain.ChronosData, error) {
	return s.landRepo.GetChronos(ctx, acqIDs, deptID)
}

func (s *landService) Suggest(ctx context.Context, q string, status int, departmentID *int) ([]*domain.LandAcquisition, error) {
	return s.landRepo.Suggest(ctx, q, status, departmentID)
}

func (s *landService) ListFilterOptions(ctx context.Context) ([]*domain.LandAcquisitionOption, error) {
	return s.landRepo.ListFilterOptions(ctx)
}

func (s *landService) ListAcquisitionDistricts(ctx context.Context) ([]domain.AU2Option, error) {
	return s.landRepo.ListAcquisitionDistricts(ctx)
}

func (s *landService) SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error {
	if _, err := s.landRepo.GetByID(ctx, acquisitionID); err != nil {
		return domain.ErrNotFound
	}
	// Мэдэгдэлд зөвхөн ШИНЭЭР нэмэгдсэн хүмүүсийг ялгахын тулд өмнөх
	// жагсаалтыг сольхоос нь өмнө авч үлдэнэ.
	previous, _ := s.landRepo.GetAssignees(ctx, acquisitionID)
	if err := s.landRepo.SetAssignees(ctx, acquisitionID, assignees); err != nil {
		return err
	}
	s.notifyAssigneesSet(ctx, acquisitionID, previous, assignees)
	s.auditLog(ctx, "assignees_set", domain.AuditResourceAcquisition, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"previous_count": len(previous),
		"current_count":  len(assignees),
		"assignees":      assignees,
	})
	return nil
}

func (s *landService) GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error) {
	return s.landRepo.GetAssignees(ctx, acquisitionID)
}

func (s *landService) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID domain.ID) (bool, error) {
	return s.landRepo.IsAssigned(ctx, acquisitionID, userID)
}

func (s *landService) IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID domain.ID) (bool, error) {
	return s.landRepo.IsAssignedToParcel(ctx, parcelID, userID)
}

func (s *landService) AcquisitionDepartmentID(ctx context.Context, acquisitionID uuid.UUID) (*int, error) {
	return s.landRepo.AcquisitionDepartmentID(ctx, acquisitionID)
}

func (s *landService) ParcelDepartmentID(ctx context.Context, parcelID uuid.UUID) (*int, error) {
	return s.landRepo.ParcelDepartmentID(ctx, parcelID)
}

func (s *landService) DecisionDraftParcelIDs(ctx context.Context, draftID uuid.UUID) ([]uuid.UUID, error) {
	return s.landRepo.DecisionDraftParcelIDs(ctx, draftID)
}

func (s *landService) IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, orgID domain.ID) (bool, error) {
	return s.landRepo.IsProfMemberOfAcquisition(ctx, acquisitionID, orgID)
}

func (s *landService) IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, orgID domain.ID) (bool, error) {
	return s.landRepo.IsProfMemberOfParcel(ctx, parcelID, orgID)
}

func (s *landService) IsProfMemberOfParcelByCode(ctx context.Context, acquisitionID uuid.UUID, parcelCode string, orgID domain.ID) (bool, error) {
	p, err := s.landRepo.GetParcelByCode(ctx, parcelCode)
	if err != nil {
		return false, err
	}
	if p.AcquisitionID != acquisitionID {
		return false, domain.ErrNotFound
	}
	return s.landRepo.IsProfMemberOfParcel(ctx, p.ID, orgID)
}

// SetAcquisitionProfessionalOrg — orgID нь ҮНЭЛГЭЭНИЙ БАЙГУУЛЛАГЫН ID
// (өмнө нь хэрэглэгчийн ID байсан). nil бол оноолт салгагдана.
func (s *landService) SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgID *domain.ID) error {
	// Эсрэг чиглэл: аль нэг нэгж талбарын хөндлөнгийн байгууллагыг үндсэн
	// болговол тэр нэгж талбар дээр нэг байгууллага өөрийгөө хянана.
	if orgID != nil {
		n, err := s.landRepo.CountParcelsWithIndependentOrg(ctx, acquisitionID, *orgID)
		if err != nil {
			return err
		}
		if n > 0 {
			return domain.NewRuleError(fmt.Sprintf("Энэ байгууллага %d нэгж талбарын хөндлөнгийн үнэлгээний байгууллага тул үндсэн байгууллагаар сонгох боломжгүй.", n))
		}
	}
	if err := s.landRepo.SetAcquisitionProfessionalOrg(ctx, acquisitionID, orgID); err != nil {
		return err
	}
	s.notifyOrgAssigned(ctx, acquisitionID, nil, orgID, false)
	s.auditLog(ctx, "professional_org_set", domain.AuditResourceAcquisition, acquisitionID.String(), uuidPtr(acquisitionID), nil, map[string]any{
		"org_id": orgID,
	})
	return nil
}

// SetParcelIndependentOrg — orgID нь хараат бус үнэлгээний БАЙГУУЛЛАГЫН ID.
func (s *landService) SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgID *domain.ID) error {
	// Хөндлөнгийн үнэлгээ нь ҮНДСЭН үнэлгээг хараат бусаар шалгадаг тул
	// нэг байгууллага хоёуланд нь байж болохгүй (өөрийн үнэлгээгээ өөрөө хянана).
	if orgID != nil {
		acq, err := s.landRepo.GetByID(ctx, acqID)
		if err != nil {
			return domain.ErrNotFound
		}
		if acq.ProfessionalOrgID != nil && *acq.ProfessionalOrgID == *orgID {
			return domain.NewRuleError("Хөндлөнгийн үнэлгээний байгууллага нь чөлөөлөлтийн үндсэн үнэлгээний байгууллагаас өөр байх ёстой.")
		}
	}
	if err := s.landRepo.SetParcelIndependentOrg(ctx, acqID, parcelID, orgID); err != nil {
		return err
	}
	s.notifyOrgAssigned(ctx, acqID, &parcelID, orgID, true)
	s.auditLog(ctx, "independent_org_set", domain.AuditResourceParcel, parcelID.String(), uuidPtr(acqID), uuidPtr(parcelID), map[string]any{
		"org_id": orgID,
	})
	return nil
}
