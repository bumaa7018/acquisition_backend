package mocks

import (
	"context"
	"time"

	"acquisition_backend/internal/domain"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
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

func (m *MockLandService) ListConstructionTypes(ctx context.Context) ([]*domain.ConstructionType, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.ConstructionType), args.Error(1)
}

func (m *MockLandService) ListAcquisitionCategories(ctx context.Context, parentID *int) ([]*domain.AcquisitionCategory, error) {
	args := m.Called(ctx, parentID)
	return args.Get(0).([]*domain.AcquisitionCategory), args.Error(1)
}

func (m *MockLandService) CreateAcquisitionCategory(ctx context.Context, name string, parentID *int, sortOrder int) (*domain.AcquisitionCategory, error) {
	args := m.Called(ctx, name, parentID, sortOrder)
	return args.Get(0).(*domain.AcquisitionCategory), args.Error(1)
}

func (m *MockLandService) UpdateAcquisitionCategory(ctx context.Context, id int, name string, sortOrder int) (*domain.AcquisitionCategory, error) {
	args := m.Called(ctx, id, name, sortOrder)
	return args.Get(0).(*domain.AcquisitionCategory), args.Error(1)
}

func (m *MockLandService) DeleteAcquisitionCategory(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLandService) ListParcelStatuses(ctx context.Context) ([]*domain.ParcelStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.ParcelStatus), args.Error(1)
}

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

func (m *MockLandService) ListBoundaryHistory(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.LandAcquisitionBoundaryHistory, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.LandAcquisitionBoundaryHistory), args.Error(1)
}

func (m *MockLandService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) Suggest(ctx context.Context, q string) ([]*domain.LandAcquisition, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.LandAcquisition), args.Error(1)
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

func (m *MockLandService) SyncParcel(ctx context.Context, acquisitionID uuid.UUID, parcelCode string) error {
	return m.Called(ctx, acquisitionID, parcelCode).Error(0)
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

func (m *MockLandService) AdvanceStatus(ctx context.Context, acquisitionID uuid.UUID, toStatus int, note, changedBy, decreeNumber string, decreeDate *time.Time) (*domain.LandAcquisition, error) {
	args := m.Called(ctx, acquisitionID, toStatus, note, changedBy, decreeNumber, decreeDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandAcquisition), args.Error(1)
}

func (m *MockLandService) UpdateParcelMeta(ctx context.Context, parcelID uuid.UUID, dbChanged bool, changedParcelID string, acquisitionAreaM2 *float64, acquisitionGeomWKT *string) error {
	args := m.Called(ctx, parcelID, dbChanged, changedParcelID, acquisitionAreaM2, acquisitionGeomWKT)
	return args.Error(0)
}

func (m *MockLandService) ListAcquisitionDocuments(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.Document, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Document), args.Error(1)
}

func (m *MockLandService) UploadAcquisitionDocument(ctx context.Context, acquisitionID uuid.UUID, filename, contentType string, data []byte, uploadedBy string, documentTypeID int) (*domain.Document, error) {
	args := m.Called(ctx, acquisitionID, filename, contentType, data, uploadedBy, documentTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Document), args.Error(1)
}

func (m *MockLandService) DeleteAcquisitionDocument(ctx context.Context, acquisitionID, documentID uuid.UUID) error {
	return m.Called(ctx, acquisitionID, documentID).Error(0)
}

func (m *MockLandService) DeleteLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) error {
	return m.Called(ctx, acquisitionID, parcelID, valuationType).Error(0)
}

func (m *MockLandService) ImportValuation(ctx context.Context, acquisitionID uuid.UUID, imp *domain.ValuationImport) (*domain.ValuationImportResult, error) {
	args := m.Called(ctx, acquisitionID, imp)
	var res *domain.ValuationImportResult
	if v := args.Get(0); v != nil {
		res = v.(*domain.ValuationImportResult)
	}
	return res, args.Error(1)
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

func (m *MockLandService) ListReportParcels(ctx context.Context, filter domain.ReportParcelFilter) ([]*domain.ReportParcelRow, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.ReportParcelRow), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) ListAssets(ctx context.Context, acquisitionID uuid.UUID, filter domain.AssetFilter) ([]*domain.Asset, int64, error) {
	args := m.Called(ctx, acquisitionID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Asset), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) CreateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error) {
	args := m.Called(ctx, b)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockLandService) UpdateAsset(ctx context.Context, b *domain.Asset) (*domain.Asset, error) {
	args := m.Called(ctx, b)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockLandService) DeleteAsset(ctx context.Context, acquisitionID, id uuid.UUID) error {
	return m.Called(ctx, acquisitionID, id).Error(0)
}

func (m *MockLandService) UpsertAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID, specs []domain.AssetSpec) error {
	return m.Called(ctx, acquisitionID, assetID, specs).Error(0)
}

func (m *MockLandService) ListAssetSpecs(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetSpec, error) {
	args := m.Called(ctx, acquisitionID, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetSpec), args.Error(1)
}

func (m *MockLandService) UpsertAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID, calcs []domain.AssetCalculation) error {
	return m.Called(ctx, acquisitionID, assetID, calcs).Error(0)
}

func (m *MockLandService) ListAssetCalculations(ctx context.Context, acquisitionID, assetID uuid.UUID) ([]*domain.AssetCalculation, error) {
	args := m.Called(ctx, acquisitionID, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetCalculation), args.Error(1)
}

func (m *MockLandService) ListAssetSpecTypes(ctx context.Context) ([]*domain.AssetSpecType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetSpecType), args.Error(1)
}

func (m *MockLandService) CreateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockLandService) UpdateAssetSpecType(ctx context.Context, t *domain.AssetSpecType) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockLandService) DeleteAssetSpecType(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLandService) ListAssetCalcTypes(ctx context.Context) ([]*domain.AssetCalcType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AssetCalcType), args.Error(1)
}

func (m *MockLandService) CreateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockLandService) UpdateAssetCalcType(ctx context.Context, t *domain.AssetCalcType) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockLandService) DeleteAssetCalcType(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLandService) UpsertLandValuation(ctx context.Context, v *domain.LandValuation) (*domain.LandValuation, error) {
	args := m.Called(ctx, v)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandValuation), args.Error(1)
}

func (m *MockLandService) GetLandValuation(ctx context.Context, acquisitionID uuid.UUID, parcelID, valuationType string) (*domain.LandValuation, error) {
	args := m.Called(ctx, acquisitionID, parcelID, valuationType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandValuation), args.Error(1)
}

func (m *MockLandService) ListCompensations(ctx context.Context, acquisitionID uuid.UUID, parcelCode, valuationType string) ([]*domain.Compensation, error) {
	args := m.Called(ctx, acquisitionID, parcelCode, valuationType)
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

func (m *MockLandService) DeleteCompensation(ctx context.Context, acquisitionID, id uuid.UUID) error {
	return m.Called(ctx, acquisitionID, id).Error(0)
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

func (m *MockLandService) DeleteCompensationGrantByCompensationID(ctx context.Context, acquisitionID, compensationID uuid.UUID) error {
	return m.Called(ctx, acquisitionID, compensationID).Error(0)
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

func (m *MockLandService) DeleteParcelDocument(ctx context.Context, parcelID, documentID uuid.UUID) error {
	return m.Called(ctx, parcelID, documentID).Error(0)
}

func (m *MockLandService) GetDashboard(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DashboardSummary), args.Error(1)
}

func (m *MockLandService) SetAssignees(ctx context.Context, acquisitionID uuid.UUID, assignees []domain.AcquisitionAssignee) error {
	return m.Called(ctx, acquisitionID, assignees).Error(0)
}

func (m *MockLandService) GetAssignees(ctx context.Context, acquisitionID uuid.UUID) ([]domain.AcquisitionAssignee, error) {
	args := m.Called(ctx, acquisitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.AcquisitionAssignee), args.Error(1)
}

func (m *MockLandService) IsAssigned(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, acquisitionID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandService) IsProfMemberOfAcquisition(ctx context.Context, acquisitionID uuid.UUID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, acquisitionID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandService) IsProfMemberOfParcel(ctx context.Context, parcelID uuid.UUID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, parcelID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandService) IsAcquisitionLocked(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandService) IsParcelAcquisitionLocked(ctx context.Context, parcelID uuid.UUID) (bool, error) {
	args := m.Called(ctx, parcelID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLandService) ListParcelWorkflows(ctx context.Context) ([]*domain.ParcelWorkflow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.ParcelWorkflow), args.Error(1)
}

func (m *MockLandService) CreateParcelWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.ParcelWorkflow, error) {
	args := m.Called(ctx, fromStatusID, toStatusID, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelWorkflow), args.Error(1)
}

func (m *MockLandService) DeleteParcelWorkflow(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) GetAvailableParcelStatuses(ctx context.Context, currentStatusID int) ([]*domain.ParcelStatus, error) {
	args := m.Called(ctx, currentStatusID)
	return args.Get(0).([]*domain.ParcelStatus), args.Error(1)
}

func (m *MockLandService) UpdateParcelStatus(ctx context.Context, parcelID uuid.UUID, acquisitionID uuid.UUID, statusID int, createdBy string) error {
	return m.Called(ctx, parcelID, acquisitionID, statusID, createdBy).Error(0)
}

func (m *MockLandService) ListParcelStatusHistory(ctx context.Context, parcelID uuid.UUID) ([]*domain.ParcelStatusHistory, error) {
	args := m.Called(ctx, parcelID)
	return args.Get(0).([]*domain.ParcelStatusHistory), args.Error(1)
}

func (m *MockLandService) UploadAssetPhoto(ctx context.Context, acqID, assetID uuid.UUID, filename, contentType string, data []byte) (string, string, error) {
	args := m.Called(ctx, acqID, assetID, filename, contentType, data)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockLandService) GetValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) (*domain.ValuationSubmission, error) {
	args := m.Called(ctx, acquisitionID, parcelID, valuationType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ValuationSubmission), args.Error(1)
}

func (m *MockLandService) TransitionValuationSubmission(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType, action, note, userID string) (*domain.ValuationSubmission, error) {
	args := m.Called(ctx, acquisitionID, parcelID, valuationType, action, note, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ValuationSubmission), args.Error(1)
}

func (m *MockLandService) ListValuationSubmissionHistory(ctx context.Context, acquisitionID, parcelID uuid.UUID, valuationType string) ([]*domain.ValuationSubmissionHistory, error) {
	args := m.Called(ctx, acquisitionID, parcelID, valuationType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ValuationSubmissionHistory), args.Error(1)
}

func (m *MockLandService) ApproveCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error {
	return m.Called(ctx, acquisitionID, id, note, reviewedBy).Error(0)
}

func (m *MockLandService) RejectCompensation(ctx context.Context, acquisitionID, id uuid.UUID, note, reviewedBy string) error {
	return m.Called(ctx, acquisitionID, id, note, reviewedBy).Error(0)
}

func (m *MockLandService) ListCompensationHistory(ctx context.Context, acquisitionID, compensationID uuid.UUID) ([]*domain.CompensationHistory, error) {
	args := m.Called(ctx, acquisitionID, compensationID)
	return args.Get(0).([]*domain.CompensationHistory), args.Error(1)
}

func (m *MockLandService) ListFundingSources(ctx context.Context, acquisitionID uuid.UUID) ([]*domain.FundingSource, error) {
	args := m.Called(ctx, acquisitionID)
	return args.Get(0).([]*domain.FundingSource), args.Error(1)
}

func (m *MockLandService) CreateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	args := m.Called(ctx, fs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FundingSource), args.Error(1)
}

func (m *MockLandService) UpdateFundingSource(ctx context.Context, fs *domain.FundingSource) (*domain.FundingSource, error) {
	args := m.Called(ctx, fs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FundingSource), args.Error(1)
}

func (m *MockLandService) DeleteFundingSource(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) ListAuthorizedRepresentatives(ctx context.Context, acquisitionID uuid.UUID, parcelID string) ([]*domain.AuthorizedRepresentative, error) {
	args := m.Called(ctx, acquisitionID, parcelID)
	return args.Get(0).([]*domain.AuthorizedRepresentative), args.Error(1)
}

func (m *MockLandService) CreateAuthorizedRepresentative(ctx context.Context, rep *domain.AuthorizedRepresentative) (*domain.AuthorizedRepresentative, error) {
	args := m.Called(ctx, rep)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AuthorizedRepresentative), args.Error(1)
}

func (m *MockLandService) DeleteAuthorizedRepresentative(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) ListAcquisitionWorkflows(ctx context.Context) ([]*domain.AcquisitionWorkflow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.AcquisitionWorkflow), args.Error(1)
}

func (m *MockLandService) CreateAcquisitionWorkflow(ctx context.Context, fromStatusID *int, toStatusID int, sortOrder int) (*domain.AcquisitionWorkflow, error) {
	args := m.Called(ctx, fromStatusID, toStatusID, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionWorkflow), args.Error(1)
}

func (m *MockLandService) DeleteAcquisitionWorkflow(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockLandService) ListAllCompensations(ctx context.Context, filter domain.GlobalCompensationFilter) ([]*domain.GlobalCompensation, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.GlobalCompensation), args.Get(1).(int64), args.Error(2)
}

func (m *MockLandService) GetGlobalCompensation(ctx context.Context, id uuid.UUID) (*domain.GlobalCompensation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GlobalCompensation), args.Error(1)
}

func (m *MockLandService) UploadCompensationReport(ctx context.Context, acqID, compID uuid.UUID, filename, contentType string, data []byte, uploadedBy string) (*domain.Compensation, error) {
	args := m.Called(ctx, acqID, compID, filename, contentType, data, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Compensation), args.Error(1)
}

func (m *MockLandService) SetAcquisitionProfessionalOrg(ctx context.Context, acquisitionID uuid.UUID, orgUserID *uuid.UUID) error {
	return m.Called(ctx, acquisitionID, orgUserID).Error(0)
}

func (m *MockLandService) SetParcelIndependentOrg(ctx context.Context, acqID, parcelID uuid.UUID, orgUserID *uuid.UUID) error {
	return m.Called(ctx, acqID, parcelID, orgUserID).Error(0)
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

type MockAcquisitionProgressStatusService struct{ mock.Mock }

func (m *MockAcquisitionProgressStatusService) List(ctx context.Context) ([]*domain.AcquisitionProgressStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AcquisitionProgressStatus), args.Error(1)
}

func (m *MockAcquisitionProgressStatusService) GetByID(ctx context.Context, id int) (*domain.AcquisitionProgressStatus, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionProgressStatus), args.Error(1)
}

func (m *MockAcquisitionProgressStatusService) Create(ctx context.Context, req service.CreateAcquisitionProgressStatusRequest) (*domain.AcquisitionProgressStatus, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionProgressStatus), args.Error(1)
}

func (m *MockAcquisitionProgressStatusService) Update(ctx context.Context, id int, req service.UpdateAcquisitionProgressStatusRequest) (*domain.AcquisitionProgressStatus, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AcquisitionProgressStatus), args.Error(1)
}

func (m *MockAcquisitionProgressStatusService) Delete(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

type MockParcelStatusService struct{ mock.Mock }

func (m *MockParcelStatusService) List(ctx context.Context) ([]*domain.ParcelStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ParcelStatus), args.Error(1)
}

func (m *MockParcelStatusService) GetByID(ctx context.Context, id int) (*domain.ParcelStatus, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelStatus), args.Error(1)
}

func (m *MockParcelStatusService) Create(ctx context.Context, req service.CreateParcelStatusRequest) (*domain.ParcelStatus, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelStatus), args.Error(1)
}

func (m *MockParcelStatusService) Update(ctx context.Context, id int, req service.UpdateParcelStatusRequest) (*domain.ParcelStatus, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ParcelStatus), args.Error(1)
}

func (m *MockParcelStatusService) Delete(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

type MockDocumentTypeService struct{ mock.Mock }

func (m *MockDocumentTypeService) List(ctx context.Context, target string) ([]*domain.DocumentType, error) {
	args := m.Called(ctx, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.DocumentType), args.Error(1)
}

func (m *MockDocumentTypeService) GetByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DocumentType), args.Error(1)
}

func (m *MockDocumentTypeService) Create(ctx context.Context, req service.CreateDocumentTypeRequest) (*domain.DocumentType, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DocumentType), args.Error(1)
}

func (m *MockDocumentTypeService) Update(ctx context.Context, id int, req service.UpdateDocumentTypeRequest) (*domain.DocumentType, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DocumentType), args.Error(1)
}

func (m *MockDocumentTypeService) Delete(ctx context.Context, id int) error {
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
