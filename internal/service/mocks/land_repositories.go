package mocks

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockPlanRepository — PlanRepository-ийн mock
type MockPlanRepository struct{ mock.Mock }

func (m *MockPlanRepository) GetByCode(ctx context.Context, planCode string) (*domain.Plan, error) {
	args := m.Called(ctx, planCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Plan), args.Error(1)
}

func (m *MockPlanRepository) GetSourceParcels(ctx context.Context, planCode string) ([]*domain.SourceParcel, error) {
	args := m.Called(ctx, planCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.SourceParcel), args.Error(1)
}

func (m *MockPlanRepository) GetSourceParcelByID(ctx context.Context, parcelID string) (*domain.SourceParcel, error) {
	args := m.Called(ctx, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SourceParcel), args.Error(1)
}

func (m *MockPlanRepository) Suggest(ctx context.Context, q string) ([]*domain.Plan, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Plan), args.Error(1)
}

// MockLandAcquisitionRepository — LandAcquisitionRepository-ийн mock
type MockLandAcquisitionRepository struct{ mock.Mock }

func (m *MockLandAcquisitionRepository) ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.ConstructionType), args.Error(1)
}

func (m *MockLandAcquisitionRepository) ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.ParcelStatus), args.Error(1)
}

func (m *MockLandAcquisitionRepository) ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error) {
	args := m.Called(ctx, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AcquisitionCategory), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error) {
	args := m.Called(ctx, name, parentID, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionCategory), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error) {
	args := m.Called(ctx, id, name, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionCategory), args.Error(1)
}

func (m *MockLandAcquisitionRepository) DeleteAcquisitionCategory(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) Create(ctx context.Context, acq *domain.LandAcquisition) error {
	return m.Called(ctx, acq).Error(0)
}

func (m *MockLandAcquisitionRepository) List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.LandAcquisition), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandAcquisitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandAcquisition), args.Error(1)
}

func (m *MockLandAcquisitionRepository) ValidateOverlap(ctx context.Context, geomWKT, planWKT string) (bool, error) {
	args := m.Called(ctx, geomWKT, planWKT)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandAcquisitionRepository) FindIntersectingAU3(ctx context.Context, geomWKT string) ([]domain.LandAcquisitionAU, error) {
	args := m.Called(ctx, geomWKT)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LandAcquisitionAU), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateAU(ctx context.Context, au *domain.LandAcquisitionAU) error {
	return m.Called(ctx, au).Error(0)
}

func (m *MockLandAcquisitionRepository) BulkInsertParcels(ctx context.Context, acquisitionID uuid.UUID, parcels []*domain.SourceParcel, acqWKT string) error {
	return m.Called(ctx, acquisitionID, parcels, acqWKT).Error(0)
}

func (m *MockLandAcquisitionRepository) CreateBoundaryHistory(ctx context.Context, history *domain.LandAcquisitionBoundaryHistory) error {
	return m.Called(ctx, history).Error(0)
}

func (m *MockLandAcquisitionRepository) ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.LandAcquisitionBoundaryHistory), args.Error(1)
}

func (m *MockLandAcquisitionRepository) GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error) {
	args := m.Called(ctx, acquisitionID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Parcel), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandAcquisitionRepository) GetParcelByID(ctx context.Context, id uuid.UUID) (*domain.Parcel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Parcel), args.Error(1)
}

func (m *MockLandAcquisitionRepository) GetParcelByCode(ctx context.Context, parcelCode string) (*domain.Parcel, error) {
	args := m.Called(ctx, parcelCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Parcel), args.Error(1)
}

func (m *MockLandAcquisitionRepository) GetParcelDetailByID(ctx context.Context, id uuid.UUID) (*domain.ParcelFull, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelFull), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error {
	return m.Called(ctx, parcelID, zone, basePricePerHa, auctionCoeff, auctionPrice).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateParcelAcquisitionGeom(ctx context.Context, parcelID uuid.UUID, acqWKT string) error {
	return m.Called(ctx, parcelID, acqWKT).Error(0)
}

func (m *MockLandAcquisitionRepository) UpsertHolder(ctx context.Context, src *domain.SourceParcel) (uuid.UUID, error) {
	args := m.Called(ctx, src)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpsertParcelRight(ctx context.Context, parcelID, acquisitionID, holderID uuid.UUID, src *domain.SourceParcel) error {
	return m.Called(ctx, parcelID, acquisitionID, holderID, src).Error(0)
}

func (m *MockLandAcquisitionRepository) Update(ctx context.Context, acq *domain.LandAcquisition) error {
	return m.Called(ctx, acq).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteAUs(ctx context.Context, acquisitionID uuid.UUID) error {
	return m.Called(ctx, acquisitionID).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteParcelsAndAUs(ctx context.Context, acquisitionID uuid.UUID) error {
	return m.Called(ctx, acquisitionID).Error(0)
}

func (m *MockLandAcquisitionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status int) error {
	return m.Called(ctx, id, status).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateDecree(ctx context.Context, id uuid.UUID, number string, date *time.Time) error {
	return m.Called(ctx, id, number, date).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
	return m.Called(ctx, parcelID, dbChanged, changedParcelID, acquisitionAreaM2, acquisitionGeomWKT).Error(0)
}

func (m *MockLandAcquisitionRepository) ListProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AcquisitionProgress), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateProgress(ctx context.Context, p *domain.AcquisitionProgress) error {
	for _, call := range m.ExpectedCalls {
		if call.Method == "CreateProgress" {
			return m.Called(ctx, p).Error(0)
		}
	}
	return nil
}

func (m *MockLandAcquisitionRepository) SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error {
	return m.Called(ctx, parcelID, paid).Error(0)
}

func (m *MockLandAcquisitionRepository) ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.GlobalParcel), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandAcquisitionRepository) ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.ReportParcelRow), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandAcquisitionRepository) ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error) {
	args := m.Called(ctx, acquisitionID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Asset), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandAcquisitionRepository) CreateAsset(ctx context.Context, b *domain.Asset) error {
	return m.Called(ctx, b).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateAsset(ctx context.Context, b *domain.Asset) error {
	return m.Called(ctx, b).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID) error {
	return m.Called(ctx, acquisitionID, id).Error(0)
}

func (m *MockLandAcquisitionRepository) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) ([]*domain.Compensation, error) {
	args := m.Called(ctx, acquisitionID, parcelCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Compensation), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateCompensation(ctx context.Context, c *domain.Compensation) error {
	return m.Called(ctx, c).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateCompensation(ctx context.Context, c *domain.Compensation) error {
	return m.Called(ctx, c).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error {
	return m.Called(ctx, acquisitionID, id).Error(0)
}

func (m *MockLandAcquisitionRepository) GetCompensation(ctx context.Context, id uuid.UUID) (*domain.Compensation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Compensation), args.Error(1)
}

func (m *MockLandAcquisitionRepository) ApproveCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error {
	return m.Called(ctx, id, note, reviewedBy).Error(0)
}

func (m *MockLandAcquisitionRepository) RejectCompensation(ctx context.Context, id uuid.UUID, note, reviewedBy string) error {
	return m.Called(ctx, id, note, reviewedBy).Error(0)
}

func (m *MockLandAcquisitionRepository) ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error) {
	args := m.Called(ctx, acquisitionID, compensationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.CompensationHistory), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpdateCompensationReport(ctx context.Context, id uuid.UUID, reportURL, reportName string) error {
	return m.Called(ctx, id, reportURL, reportName).Error(0)
}

func (m *MockLandAcquisitionRepository) CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error {
	return m.Called(ctx, g).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) error {
	return m.Called(ctx, g).Error(0)
}

func (m *MockLandAcquisitionRepository) GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CompensationGrant), args.Error(1)
}

func (m *MockLandAcquisitionRepository) DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error {
	return m.Called(ctx, acquisitionID, compensationID).Error(0)
}

func (m *MockLandAcquisitionRepository) ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error) {
	args := m.Called(ctx, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ParcelPayment), args.Error(1)
}

func (m *MockLandAcquisitionRepository) AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Document), args.Error(1)
}

func (m *MockLandAcquisitionRepository) AddAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, doc *domain.Document) error {
	return m.Called(ctx, acquisitionID, doc).Error(0)
}

func (m *MockLandAcquisitionRepository) ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error) {
	args := m.Called(ctx, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Document), args.Error(1)
}

func (m *MockLandAcquisitionRepository) AddParcelDocument(ctx context.Context, parcelID uuid.UUID, doc *domain.Document) error {
	return m.Called(ctx, parcelID, doc).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error {
	return m.Called(ctx, acquisitionID, documentID).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error {
	return m.Called(ctx, parcelID, documentID).Error(0)
}

func (m *MockLandAcquisitionRepository) GetDocument(ctx context.Context, documentID uuid.UUID) (*domain.Document, error) {
	args := m.Called(ctx, documentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *MockLandAcquisitionRepository) GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DashboardSummary), args.Error(1)
}

func (m *MockLandAcquisitionRepository) Suggest(ctx context.Context, q string) ([]*domain.LandAcquisition, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.LandAcquisition), args.Error(1)
}

func (m *MockLandAcquisitionRepository) SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error {
	return m.Called(ctx, acquisitionID, assignees).Error(0)
}

func (m *MockLandAcquisitionRepository) GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.AcquisitionAssignee), args.Error(1)
}

func (m *MockLandAcquisitionRepository) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, acquisitionID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandAcquisitionRepository) IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, acquisitionID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandAcquisitionRepository) IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, parcelID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandAcquisitionRepository) ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ParcelWorkflow), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error) {
	args := m.Called(ctx, fromStatusID, toStatusID, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelWorkflow), args.Error(1)
}

func (m *MockLandAcquisitionRepository) DeleteParcelWorkflow(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error) {
	args := m.Called(ctx, currentStatusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ParcelStatus), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	return m.Called(ctx, parcelID, acquisitionID, statusID, createdBy).Error(0)
}

func (m *MockLandAcquisitionRepository) InsertParcelStatusHistory(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	return m.Called(ctx, parcelID, acquisitionID, statusID, createdBy).Error(0)
}

func (m *MockLandAcquisitionRepository) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	args := m.Called(ctx, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ParcelStatusHistory), args.Error(1)
}

func (m *MockLandAcquisitionRepository) ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.FundingSource), args.Error(1)
}

func (m *MockLandAcquisitionRepository) GetFundingSource(ctx context.Context, id uuid.UUID) (*domain.FundingSource, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FundingSource), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateFundingSource(ctx context.Context, fs *domain.FundingSource) error {
	return m.Called(ctx, fs).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) error {
	return m.Called(ctx, fs).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteFundingSource(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error) {
	args := m.Called(ctx, acquisitionID, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AuthorizedRepresentative), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) error {
	return m.Called(ctx, rep).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AcquisitionWorkflow), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error) {
	args := m.Called(ctx, fromStatusID, toStatusID, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionWorkflow), args.Error(1)
}

func (m *MockLandAcquisitionRepository) DeleteAcquisitionWorkflow(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.GlobalCompensation), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandAcquisitionRepository) GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GlobalCompensation), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error {
	return m.Called(ctx, acquisitionID, assetID, specs).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error) {
	args := m.Called(ctx, acquisitionID, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetSpec), args.Error(1)
}

func (m *MockLandAcquisitionRepository) UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error {
	return m.Called(ctx, acquisitionID, assetID, calcs).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error) {
	args := m.Called(ctx, acquisitionID, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetCalculation), args.Error(1)
}

func (m *MockLandAcquisitionRepository) SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *uuid.UUID) error {
	return m.Called(ctx, acquisitionID, orgUserID).Error(0)
}

func (m *MockLandAcquisitionRepository) SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *uuid.UUID) error {
	return m.Called(ctx, acqID, parcelID, orgUserID).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetSpecType), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	return m.Called(ctx, t).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	return m.Called(ctx, t).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteAssetSpecType(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetCalcType), args.Error(1)
}

func (m *MockLandAcquisitionRepository) CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	return m.Called(ctx, t).Error(0)
}

func (m *MockLandAcquisitionRepository) UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	return m.Called(ctx, t).Error(0)
}

func (m *MockLandAcquisitionRepository) DeleteAssetCalcType(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandAcquisitionRepository) UpsertLandValuation(ctx context.Context, v *domain.LandValuation) error {
	return m.Called(ctx, v).Error(0)
}

func (m *MockLandAcquisitionRepository) GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID string) (*domain.LandValuation, error) {
	args := m.Called(ctx, acquisitionID, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandValuation), args.Error(1)
}

func (m *MockLandAcquisitionRepository) DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID string) error {
	return m.Called(ctx, acquisitionID, parcelID).Error(0)
}

func (m *MockLandAcquisitionRepository) ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error) {
	args := m.Called(ctx, acquisitionID, imp)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ValuationImportResult), args.Error(1)
}
