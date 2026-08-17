package service_test

import (
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompensation_List_Success tests successful listing of compensations
func TestCompensation_List_Success(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	expected := []*domain.Compensation{
		{
			ID:               uuid.New(),
			AcquisitionID:    acqID,
			TargetType:       "asset",
			ParcelID:         "PC-001",
			CompensationType: "cash",
			CoveragePercent:  100,
			Amount:           5000000,
		},
	}

	landRepo.On("ListCompensations", ctx, acqID, "", "").Return(expected, nil)

	list, err := svc.ListCompensations(ctx, acqID, "", "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, expected[0].ParcelID, list[0].ParcelID)
	landRepo.AssertExpectations(t)
}

// TestCompensation_Create_Success tests successful creation of a compensation
func TestCompensation_Create_Success(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	comp := &domain.Compensation{
		AcquisitionID:    acqID,
		TargetType:       "asset",
		ParcelID:         "PC-002",
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
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	compID := uuid.New()
	pending := &domain.Compensation{ID: compID, AcquisitionID: acqID, Status: "pending"}
	landRepo.On("GetCompensation", ctx, compID).Return(pending, nil)
	landRepo.On("DeleteCompensation", ctx, acqID, compID).Return(nil)

	err := svc.DeleteCompensation(ctx, acqID, compID, nil)
	require.NoError(t, err)
	landRepo.AssertExpectations(t)
}

// Гадаад дуудагч ӨӨРИЙН урсгалынхаа мөрийг л устгана: МИКА (allowed=["mika"])
// мэргэжлийн байгууллагын ("asset") үнэлгээг устгаж чадахгүй.
func TestCompensation_Delete_ForeignStream_Forbidden(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	compID := uuid.New()
	other := &domain.Compensation{
		ID: compID, AcquisitionID: acqID, Status: "pending",
		ValuationType: domain.ValuationTypeAsset,
	}
	landRepo.On("GetCompensation", ctx, compID).Return(other, nil)

	err := svc.DeleteCompensation(ctx, acqID, compID, []string{domain.ValuationTypeMika})
	require.ErrorIs(t, err, domain.ErrForbidden)
	// Устгах дуудалт хүрэлгүй таслагдсан байх ёстой.
	landRepo.AssertNotCalled(t, "DeleteCompensation", ctx, acqID, compID)
	landRepo.AssertExpectations(t)
}

// Өөрийн урсгалын мөрийг устгана.
func TestCompensation_Delete_OwnStream_OK(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	compID := uuid.New()
	own := &domain.Compensation{
		ID: compID, AcquisitionID: acqID, Status: "pending",
		ValuationType: domain.ValuationTypeMika,
	}
	landRepo.On("GetCompensation", ctx, compID).Return(own, nil)
	landRepo.On("DeleteCompensation", ctx, acqID, compID).Return(nil)

	err := svc.DeleteCompensation(ctx, acqID, compID, []string{domain.ValuationTypeMika})
	require.NoError(t, err)
	landRepo.AssertExpectations(t)
}

// TestCompensationGrant_Create_Success tests successful creation of a compensation grant
func TestCompensationGrant_Create_Success(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	compID := uuid.New()
	grant := &domain.CompensationGrant{
		AcquisitionID:  acqID,
		CompensationID: compID,
		Amount:         10000000,
		Note:           "Land grant for parcel PC-001",
		LandAreaM2:     500,
		LandPrice:      20000,
		LandLocation:   "Баянзүрх дүүрэг",
		LandPurpose:    "Орон сууцны газар",
		LandUseType:    "Өмчийн",
		ParcelNumber:   "PC-GRANT-001",
		DecreeNumber:   "DECREE-2024-001",
		CreatedBy:      "test-user",
	}

	landRepo.On("GetCompensation", ctx, compID).Return(&domain.Compensation{ID: compID, AcquisitionID: acqID}, nil)
	landRepo.On("CreateCompensationGrant", ctx, grant).Return(nil)

	created, err := svc.CreateCompensationGrant(ctx, grant)
	require.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, grant.Amount, created.Amount)
	assert.Equal(t, grant.ParcelNumber, created.ParcelNumber)
	assert.Equal(t, grant.DecreeNumber, created.DecreeNumber)
	landRepo.AssertExpectations(t)
}
