package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/go-api-template/internal/domain"
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

// PlanRepository — auth DB-аас төлөвлөгөө, эх нэгж талбар уншина
type PlanRepository interface {
	GetByCode(ctx context.Context, planCode string) (*domain.Plan, error)
	GetSourceParcels(ctx context.Context, planCode string) ([]*domain.SourceParcel, error)
	GetSourceParcelByID(ctx context.Context, parcelID string) (*domain.SourceParcel, error)
	Suggest(ctx context.Context, q string) ([]*domain.Plan, error)
}

// LandAcquisitionRepository — main DB-д газар чөлөөлөлтын мэдээлэл бичнэ
type LandAcquisitionRepository interface {
	ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error)
	ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error)
	Create(ctx context.Context, acq *domain.LandAcquisition) error
	List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error)
	Suggest(ctx context.Context, q string) ([]*domain.LandAcquisition, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error)
	ValidateWithin(ctx context.Context, innerWKT, outerWKT string) (bool, error)
	FindIntersectingAU3(ctx context.Context, geomWKT string) ([]domain.LandAcquisitionAU, error)
	CreateAU(ctx context.Context, au *domain.LandAcquisitionAU) error
	BulkInsertParcels(ctx context.Context, acquisitionID uuid.UUID, parcels []*domain.SourceParcel, acqWKT string) error
	GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error)
	GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error)
	GetParcelDetailByID(ctx context.Context, id uuid.UUID) (*domain.ParcelFull, error)
	UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error
	UpdateParcelAcquisitionGeom(ctx context.Context, parcelID uuid.UUID, acqWKT string) error
	UpsertHolder(ctx context.Context, src *domain.SourceParcel) (uuid.UUID, error)
	UpsertParcelRight(ctx context.Context, parcelID, acquisitionID, holderID uuid.UUID, src *domain.SourceParcel) error
	Update(ctx context.Context, acq *domain.LandAcquisition) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status int) error
	UpdateDecree(ctx context.Context, id uuid.UUID, number string, date *time.Time) error
	UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string) error
	DeleteParcelsAndAUs(ctx context.Context, acquisitionID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error)
	CreateProgress(ctx context.Context, p *domain.AcquisitionProgress) error
	SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error
	ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error)
	ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error)
	GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error)
	// Asset methods
	ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error)
	CreateAsset(ctx context.Context, b *domain.Asset) error
	UpdateAsset(ctx context.Context, b *domain.Asset) error
	DeleteAsset(ctx context.Context, id uuid.UUID) error
	ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error)
	AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error
	// Compensation methods
	ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.Compensation, error)
	CreateCompensation(ctx context.Context, c *domain.Compensation) error
	UpdateCompensation(ctx context.Context, c *domain.Compensation) error
	DeleteCompensation(ctx context.Context, id uuid.UUID) error
	// CompensationGrant methods
	CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error
	UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error
	GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error)
	DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error
	DeleteCompensationGrantByCompensationID(ctx context.Context, compensationID uuid.UUID) error
	// Document methods
	ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error)
	AddAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, doc *domain.Document) error
	ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error)
	AddParcelDocument(ctx context.Context, parcelID uuid.UUID, doc *domain.Document) error
	DeleteDocument(ctx context.Context, documentID uuid.UUID) error
	GetDocument(ctx context.Context, documentID uuid.UUID) (*domain.Document, error)
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
	List(ctx context.Context) ([]*domain.DocumentType, error)
	GetByID(ctx context.Context, id int) (*domain.DocumentType, error)
	Create(ctx context.Context, dt *domain.DocumentType) error
	Update(ctx context.Context, dt *domain.DocumentType) error
	Delete(ctx context.Context, id int) error
}
