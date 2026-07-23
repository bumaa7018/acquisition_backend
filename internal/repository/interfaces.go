package repository

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error)
	GetPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	// InvalidateSessions — тухайн хэрэглэгчийн одоо хүчинтэй бүх токеныг
	// (энэ агшингаас өмнө олгогдсоныг) хүчингүй болгоно.
	InvalidateSessions(ctx context.Context, userID uuid.UUID) error
	// IsSessionValid — тухайн issuedAt хугацаанд олгогдсон токен хэрэглэгчийн
	// эх цэгээс хойш байвал (буюу эх цэг тавиагүй бол) true.
	IsSessionValid(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error)
}

type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	GetByName(ctx context.Context, name string) (*domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*domain.Role, error)
	AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveFromUser(ctx context.Context, userID, roleID uuid.UUID) error
}

type PermissionRepository interface {
	Create(ctx context.Context, p *domain.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error)
	GetByName(ctx context.Context, name string) (*domain.Permission, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*domain.Permission, error)
}

type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type TokenRevocationRepository interface {
	RevokeToken(ctx context.Context, token *domain.RevokedToken) error
	IsTokenRevoked(ctx context.Context, jti string) (bool, error)
	DeleteExpired(ctx context.Context) error
}

// PlanRepository — auth DB-аас төлөвлөгөө, эх нэгж талбар уншина
type PlanRepository interface {
	GetByCode(ctx context.Context, planCode string) (*domain.Plan, error)
	GetSourceParcels(ctx context.Context, planCode string) ([]*domain.SourceParcel, error)
	GetSourceParcelByID(ctx context.Context, parcelID string) (*domain.SourceParcel, error)
	Suggest(ctx context.Context, q string) ([]*domain.Plan, error)
}

type LandLookupRepository interface {
	ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error)
	ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error)
	CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error)
	UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error)
	DeleteAcquisitionCategory(ctx context.Context, id int) error
	ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error)
}

type LandAcquisitionWriteRepository interface {
	Create(ctx context.Context, acq *domain.LandAcquisition) error
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	Suggest(ctx context.Context, q string, status int) ([]*domain.LandAcquisition, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	ValidateOverlap(ctx context.Context, geomWKT, planWKT string) (bool, error)
	FindIntersectingAU3(ctx context.Context, geomWKT string) ([]domain.LandAcquisitionAU, error)
	CreateAU(ctx context.Context, au *domain.LandAcquisitionAU) error
	BulkInsertParcels(ctx context.Context, acquisitionID uuid.UUID, parcels []*domain.SourceParcel, acqWKT string) error
	CreateBoundaryHistory(ctx context.Context, history *domain.LandAcquisitionBoundaryHistory) error
	ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error)
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error)
	GetParcelByCode(ctx context.Context, parcelCode string) (*domain.Parcel, error)
	GetParcelDetailByID(ctx context.Context, id uuid.UUID) (*domain.ParcelFull, error)
	UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error
	UpdateParcelAcquisitionGeom(ctx context.Context, parcelID uuid.UUID, acqWKT string) error
	SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *uuid.UUID) error
	SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *uuid.UUID) error
	UpsertHolder(ctx context.Context, src *domain.SourceParcel) (uuid.UUID, error)
	UpsertParcelRight(ctx context.Context, parcelID, acquisitionID, holderID uuid.UUID, src *domain.SourceParcel) error
	Update(ctx context.Context, acq *domain.LandAcquisition) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status int) error
	UpdateDecree(ctx context.Context, id uuid.UUID, number string, date *time.Time) error
	HasNonFinalParcels(ctx context.Context, acquisitionID uuid.UUID) (bool, error)
	UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error
	DeleteAUs(ctx context.Context, acquisitionID uuid.UUID) error
	DeleteParcelsAndAUs(ctx context.Context, acquisitionID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type LandProgressRepository interface {
	ListProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	CreateProgress(ctx context.Context, p *domain.AcquisitionProgress) error
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error)
	GetReportSummary(ctx context.Context, filter domain.ReportParcelFilter) (*domain.ReportSummary, error)
	GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error)
}

type LandAssetRepository interface {
	ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error)
	CreateAsset(ctx context.Context, b *domain.Asset) error
	UpdateAsset(ctx context.Context, b *domain.Asset) error
	DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID) (string, error)
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
	UpsertLandValuation(ctx context.Context, v *domain.LandValuation) error
	GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error)
	DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error
	ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error)
}

type LandPaymentRepository interface {
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
}

type LandCompensationRepository interface {
	ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) ([]*domain.Compensation, error)
	CreateCompensation(ctx context.Context, c *domain.Compensation) error
	UpdateCompensation(ctx context.Context, c *domain.Compensation) error
	DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error
	GetCompensation(ctx context.Context, id uuid.UUID) (*domain.Compensation, error)
	ApproveCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error
	RejectCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error
	UpdateCompensationReport(ctx context.Context, id uuid.UUID, reportURL, reportName string) error
	UpdateAssetPhoto(ctx context.Context, acquisitionID, assetID uuid.UUID, photoURL, photoName string) (string, error)
	ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error)
	ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error)
	GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error)
}

type LandCompensationGrantRepository interface {
	CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error
	UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error
	GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error)
	DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error
	DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error
}

type LandDocumentRepository interface {
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	AddAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, doc *domain.Document) error
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	AddParcelDocument(ctx context.Context, parcelID uuid.UUID, doc *domain.Document) error
	DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error
	DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error
	GetDocument(ctx context.Context, documentID uuid.UUID) (*domain.Document, error)
	GetDocumentType(ctx context.Context, id int) (*domain.DocumentType, error)
}

type LandAssigneeRepository interface {
	SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error
	GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error)
	IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error)
	IsAssignedToParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error)
	IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error)
	IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error)
}

type LandWorkflowRepository interface {
	ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error)
	CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error)
	DeleteParcelWorkflow(ctx context.Context, id int) error
	GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error)
	UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error
	InsertParcelStatusHistory(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error
	ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error)
	GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error)
	TransitionValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType, action, note, userID string) (*domain.ValuationSubmission, error)
	ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error)
	ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error)
	CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error)
	DeleteAcquisitionWorkflow(ctx context.Context, id int) error
}

type LandFundingRepository interface {
	ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error)
	GetFundingSource(ctx context.Context, id uuid.UUID) (*domain.FundingSource, error)
	CreateFundingSource(ctx context.Context, fs *domain.FundingSource) error
	UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) error
	DeleteFundingSource(ctx context.Context, id uuid.UUID) error
}

type LandAuthorizedRepresentativeRepository interface {
	ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error)
	CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) error
	DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error
}

// LandAcquisitionRepository — main DB-д газар чөлөөлөлтын мэдээлэл бичнэ.
type LandAcquisitionRepository interface {
	LandLookupRepository
	LandAcquisitionWriteRepository
	LandProgressRepository
	LandAssetRepository
	LandPaymentRepository
	LandCompensationRepository
	LandCompensationGrantRepository
	LandDocumentRepository
	LandAssigneeRepository
	LandWorkflowRepository
	LandFundingRepository
	LandAuthorizedRepresentativeRepository
}

type AcquisitionProgressStatusRepository interface {
	List(ctx context.Context) ([]*domain.AcquisitionProgressStatus, error)
	GetByID(ctx context.Context, id int) (*domain.AcquisitionProgressStatus, error)
	Create(ctx context.Context, aps *domain.AcquisitionProgressStatus) error
	Update(ctx context.Context, aps *domain.AcquisitionProgressStatus) error
	Delete(ctx context.Context, id int) error
}

type ParcelStatusRepository interface {
	List(ctx context.Context) ([]*domain.ParcelStatus, error)
	GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error)
	Create(ctx context.Context, ps *domain.ParcelStatus) error
	Update(ctx context.Context, ps *domain.ParcelStatus) error
	Delete(ctx context.Context, id int) error
}

type DocumentTypeRepository interface {
	List(ctx context.Context, target string) ([]*domain.DocumentType, error)
	GetByID(ctx context.Context, id int) (*domain.DocumentType, error)
	Create(ctx context.Context, dt *domain.DocumentType) error
	Update(ctx context.Context, dt *domain.DocumentType) error
	Delete(ctx context.Context, id int) error
}

// NotificationRepository — хэрэглэгчийн мэдэгдэл (main DB).
// Бүх унших/бичих үйлдэл recipient_id-аар шүүгдэж, хэрэглэгч зөвхөн
// өөрийн мэдэгдэлд ханддаг.
type NotificationRepository interface {
	CreateBatch(ctx context.Context, ns []*domain.Notification) error
	ListByRecipient(ctx context.Context, recipientID uuid.UUID, onlyUnread bool, page, pageSize int) ([]*domain.Notification, int64, error)
	UnreadCount(ctx context.Context, recipientID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, recipientID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, recipientID uuid.UUID) error
}

// AuditLogRepository — ажилтны бизнес үйлдлийн append-only лог.
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, int64, error)
}

// ClientRequestRepository — бусад системрүү илгээсэн HTTP хүсэлтийн append-only лог.
type ClientRequestRepository interface {
	Create(ctx context.Context, r *domain.ClientRequest) error
	List(ctx context.Context, filter domain.ClientRequestFilter) ([]*domain.ClientRequest, int64, error)
}
