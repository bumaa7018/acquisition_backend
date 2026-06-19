package mocks

import (
	"context"

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

// MockLandAcquisitionRepository — LandAcquisitionRepository-ийн mock
type MockLandAcquisitionRepository struct{ mock.Mock }

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
