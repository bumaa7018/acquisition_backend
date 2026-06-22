package service_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	"github.com/your-org/go-api-template/internal/service/mocks"
)

// TestCompensation_List_Success tests successful listing of compensations
func TestCompensation_List_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	expected := []*domain.Compensation{
		{
			ID:               uuid.New(),
			AcquisitionID:    acqID,
			AssetType:        "parcel",
			ParcelID:       "PC-001",
			CompensationType: "cash",
			CoveragePercent:  100,
			Amount:           5000000,
		},
	}

	landRepo.On("ListCompensations", ctx, acqID, "").Return(expected, nil)

	list, err := svc.ListCompensations(ctx, acqID, "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, expected[0].ParcelID, list[0].ParcelID)
	landRepo.AssertExpectations(t)
}

// TestCompensation_Create_Success tests successful creation of a compensation
func TestCompensation_Create_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	comp := &domain.Compensation{
		AcquisitionID:    acqID,
		AssetType:        "parcel",
		ParcelID:       "PC-002",
		CompensationType: "cash",
		CoveragePercent:  100,
		Amount:           3000000,
		CreatedBy:        "test-user",
	}

	landRepo.On("CreateCompensation", ctx, comp).Return(nil)

	created, err := svc.CreateCompensation(ctx, comp)
	require.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, comp.ParcelID, created.ParcelID)
	assert.Equal(t, comp.Amount, created.Amount)
	landRepo.AssertExpectations(t)
}

// TestCompensation_Delete_Success tests successful deletion of a compensation
func TestCompensation_Delete_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	compID := uuid.New()
	landRepo.On("DeleteCompensation", ctx, compID).Return(nil)

	err := svc.DeleteCompensation(ctx, compID)
	require.NoError(t, err)
	landRepo.AssertExpectations(t)
}

// TestCompensationGrant_Create_Success tests successful creation of a compensation grant
func TestCompensationGrant_Create_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	grant := &domain.CompensationGrant{
		AcquisitionID: acqID,
		Amount:        10000000,
		Note:          "Land grant for parcel PC-001",
		LandAreaM2:    500,
		LandPrice:     20000,
		LandLocation:  "Баянзүрх дүүрэг",
		LandPurpose:   "Орон сууцны газар",
		LandUseType:   "Өмчийн",
		ParcelNumber:  "PC-GRANT-001",
		DecreeNumber:  "DECREE-2024-001",
		CreatedBy:     "test-user",
	}

	landRepo.On("CreateCompensationGrant", ctx, grant).Return(nil)

	created, err := svc.CreateCompensationGrant(ctx, grant)
	require.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, grant.Amount, created.Amount)
	assert.Equal(t, grant.ParcelNumber, created.ParcelNumber)
	assert.Equal(t, grant.DecreeNumber, created.DecreeNumber)
	landRepo.AssertExpectations(t)
}
