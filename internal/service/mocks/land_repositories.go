package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/your-org/go-api-template/internal/domain"
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

func (m *MockLandAcquisitionRepository) ValidateWithin(ctx context.Context, innerWKT, outerWKT string) (bool, error) {
	args := m.Called(ctx, innerWKT, outerWKT)
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

func (m *MockLandAcquisitionRepository) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string) error {
	return m.Called(ctx, parcelID, dbChanged, changedParcelID).Error(0)
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

func (m *MockLandAcquisitionRepository) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
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

func (m *MockLandAcquisitionRepository) DeleteCompensation(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
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

func (m *MockLandAcquisitionRepository) DeleteCompensationGrantByCompensationID(ctx context.Context, compensationID uuid.UUID) error {
	return m.Called(ctx, compensationID).Error(0)
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

func (m *MockLandAcquisitionRepository) DeleteDocument(ctx context.Context, documentID uuid.UUID) error {
	return m.Called(ctx, documentID).Error(0)
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
