package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/your-org/go-api-template/internal/domain"
	jwtpkg "github.com/your-org/go-api-template/internal/pkg/jwt"
	"github.com/your-org/go-api-template/internal/service"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req service.RegisterRequest) (*domain.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req service.LoginRequest) (*jwtpkg.TokenPair, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.TokenPair), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwtpkg.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.TokenPair), args.Error(1)
}

func (m *MockAuthService) ForgotPassword(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

type MockLandService struct{ mock.Mock }

func (m *MockLandService) List(ctx context.Context, filter domain.LandAcquisitionFilter) ([]*domain.LandAcquisition, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.LandAcquisition), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) Create(ctx context.Context, req service.CreateLandAcquisitionRequest) (*domain.LandAcquisition, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandAcquisition), args.Error(1)
}

func (m *MockLandService) GetByID(ctx context.Context, id uuid.UUID) (*domain.LandAcquisition, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandAcquisition), args.Error(1)
}

func (m *MockLandService) Update(ctx context.Context, id uuid.UUID, req service.UpdateLandAcquisitionRequest) (*domain.LandAcquisition, bool, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*domain.LandAcquisition), args.Bool(1), args.Error(2)
}

func (m *MockLandService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) GetParcels(ctx context.Context, acquisitionID uuid.UUID, filter domain.ParcelFilter) ([]*domain.Parcel, int64, error) {
	args := m.Called(ctx, acquisitionID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Parcel), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) GetParcelDetail(ctx context.Context, acquisitionID, parcelID uuid.UUID) (*domain.ParcelFull, error) {
	args := m.Called(ctx, acquisitionID, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelFull), args.Error(1)
}

func (m *MockLandService) UpdateParcelValuation(ctx context.Context, parcelID uuid.UUID, zone string, basePricePerHa, auctionCoeff, auctionPrice *float64) error {
	return m.Called(ctx, parcelID, zone, basePricePerHa, auctionCoeff, auctionPrice).Error(0)
}

func (m *MockLandService) SyncParcel(ctx context.Context, acquisitionID, parcelID uuid.UUID) error {
	return m.Called(ctx, acquisitionID, parcelID).Error(0)
}

func (m *MockLandService) GetProgress(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.AcquisitionProgress, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AcquisitionProgress), args.Error(1)
}

func (m *MockLandService) GetAvailableStatuses(ctx context.Context, acquisitionID uuid.UUID) ([]domain.StatusOption, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.StatusOption), args.Error(1)
}

func (m *MockLandService) AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy string) (*domain.LandAcquisition, error) {
	args := m.Called(ctx, acquisitionID, toStatus, note, changedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandAcquisition), args.Error(1)
}

func (m *MockLandService) ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Document), args.Error(1)
}

func (m *MockLandService) UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error) {
	args := m.Called(ctx, acquisitionID, filename, contentType, data, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *MockLandService) DeleteAcquisitionDocument(ctx context.Context, documentID uuid.UUID) error {
	return m.Called(ctx, documentID).Error(0)
}

func (m *MockLandService) SetParcelCompensation(ctx context.Context, parcelID uuid.UUID, paid bool) error {
	return m.Called(ctx, parcelID, paid).Error(0)
}

func (m *MockLandService) ListGlobalParcels(ctx context.Context, filter domain.GlobalParcelFilter) ([]*domain.GlobalParcel, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.GlobalParcel), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) ListBuildings(ctx context.Context, acquisitionID uuid.UUID, filter domain.BuildingFilter) ([]*domain.Building, int64, error) {
	args := m.Called(ctx, acquisitionID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Building), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) CreateBuilding(ctx context.Context, b *domain.Building) (*domain.Building, error) {
	args := m.Called(ctx, b)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Building), args.Error(1)
}

func (m *MockLandService) UpdateBuilding(ctx context.Context, b *domain.Building) (*domain.Building, error) {
	args := m.Called(ctx, b)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Building), args.Error(1)
}

func (m *MockLandService) DeleteBuilding(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) ([]*domain.Compensation, error) {
	args := m.Called(ctx, acquisitionID, parcelCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Compensation), args.Error(1)
}

func (m *MockLandService) CreateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Compensation), args.Error(1)
}

func (m *MockLandService) UpdateCompensation(ctx context.Context, c *domain.Compensation) (*domain.Compensation, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Compensation), args.Error(1)
}

func (m *MockLandService) DeleteCompensation(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) CreateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	args := m.Called(ctx, g)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CompensationGrant), args.Error(1)
}

func (m *MockLandService) UpdateCompensationGrant(ctx context.Context, g *domain.CompensationGrant) (*domain.CompensationGrant, error) {
	args := m.Called(ctx, g)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CompensationGrant), args.Error(1)
}

func (m *MockLandService) GetCompensationGrant(ctx context.Context, id uuid.UUID) (*domain.CompensationGrant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CompensationGrant), args.Error(1)
}

func (m *MockLandService) DeleteCompensationGrant(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) DeleteCompensationGrantByCompensationID(ctx context.Context, compensationID uuid.UUID) error {
	return m.Called(ctx, compensationID).Error(0)
}

func (m *MockLandService) ListParcelPayments(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelPayment, error) {
	args := m.Called(ctx, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ParcelPayment), args.Error(1)
}

func (m *MockLandService) AddParcelPayment(ctx context.Context, p *domain.ParcelPayment) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockLandService) ListParcelDocuments(ctx context.Context, parcelID uuid.UUID) ([]*domain.Document, error) {
	args := m.Called(ctx, parcelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Document), args.Error(1)
}

func (m *MockLandService) UploadParcelDocument(ctx context.Context, parcelID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Document, error) {
	args := m.Called(ctx, parcelID, filename, contentType, data, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *MockLandService) DeleteParcelDocument(ctx context.Context, documentID uuid.UUID) error {
	return m.Called(ctx, documentID).Error(0)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) Create(ctx context.Context, req service.CreateUserRequest) (*domain.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, id uuid.UUID, req service.UpdateUserRequest) (*domain.User, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockRoleService struct{ mock.Mock }

func (m *MockRoleService) List(ctx context.Context) ([]*domain.Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Role), args.Error(1)
}

func (m *MockRoleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Role), args.Error(1)
}

func (m *MockRoleService) Create(ctx context.Context, req service.CreateRoleRequest) (*domain.Role, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Role), args.Error(1)
}

func (m *MockRoleService) Update(ctx context.Context, id uuid.UUID, req service.UpdateRoleRequest) (*domain.Role, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Role), args.Error(1)
}

func (m *MockRoleService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRoleService) AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return m.Called(ctx, roleID, permissionID).Error(0)
}

func (m *MockRoleService) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return m.Called(ctx, roleID, permissionID).Error(0)
}

func (m *MockRoleService) AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

func (m *MockRoleService) RemoveFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return m.Called(ctx, userID, roleID).Error(0)
}

type MockPermissionService struct{ mock.Mock }

func (m *MockPermissionService) List(ctx context.Context) ([]*domain.Permission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Permission), args.Error(1)
}

func (m *MockPermissionService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Permission), args.Error(1)
}

func (m *MockPermissionService) Create(ctx context.Context, req service.CreatePermissionRequest) (*domain.Permission, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Permission), args.Error(1)
}

func (m *MockPermissionService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type MockPlanService struct{ mock.Mock }

func (m *MockPlanService) Search(ctx context.Context, code string) (*domain.Plan, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Plan), args.Error(1)
}

func (m *MockPlanService) Suggest(ctx context.Context, q string) ([]*domain.Plan, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Plan), args.Error(1)
}
