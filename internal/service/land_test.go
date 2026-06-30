package service_test

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// buildSimpleShp нь нэг хэлбэр бүхий polygon .shp файлын байт буцаана.
func buildSimpleShp() []byte {
	pts := [][2]float64{
		{106.76, 47.66}, {106.82, 47.66},
		{106.82, 47.72}, {106.76, 47.72}, {106.76, 47.66},
	}
	numParts, numPoints := 1, len(pts)
	contentLen := 4 + 32 + 4 + 4 + numParts*4 + numPoints*16
	shpBuf := make([]byte, 100+8+contentLen)
	binary.BigEndian.PutUint32(shpBuf[0:], 9994)
	binary.BigEndian.PutUint32(shpBuf[24:], uint32((100+8+contentLen)/2))
	binary.LittleEndian.PutUint32(shpBuf[28:], 1000)
	binary.LittleEndian.PutUint32(shpBuf[32:], 5)
	binary.BigEndian.PutUint32(shpBuf[100:], 1)
	binary.BigEndian.PutUint32(shpBuf[104:], uint32(contentLen/2))
	off := 108
	binary.LittleEndian.PutUint32(shpBuf[off:], 5)
	off += 4 + 32
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(numParts))
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], uint32(numPoints))
	off += 4
	binary.LittleEndian.PutUint32(shpBuf[off:], 0)
	off += 4
	for _, p := range pts {
		binary.LittleEndian.PutUint64(shpBuf[off:], math.Float64bits(p[0]))
		off += 8
		binary.LittleEndian.PutUint64(shpBuf[off:], math.Float64bits(p[1]))
		off += 8
	}
	return shpBuf
}

func newLandSvc(planRepo *mocks.MockPlanRepository, landRepo *mocks.MockLandAcquisitionRepository) service.LandService {
	return service.NewLandService(planRepo, landRepo, filestorage.DisabledStorage{})
}

var (
	ctx           = context.Background()
	testPlanCode  = "PLAN-2024-001"
	testBoundary  = "POLYGON((106.5 47.5,107.5 47.5,107.5 48.0,106.5 48.0,106.5 47.5))"
	testStartDate = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	testEndDate   = time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
)

func testPlan() *domain.Plan {
	return &domain.Plan{
		ID:          uuid.New(),
		PlanCode:    testPlanCode,
		Name:        "Test Plan",
		BoundaryWKT: testBoundary,
		AreaM2:      8500000000,
		Status:      domain.PlanStatusActive,
	}
}

// ── Create ────────────────────────────────────────────────────

func TestLandService_Create_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	plan := testPlan()
	zip := buildSimpleShp()

	planRepo.On("GetByCode", ctx, testPlanCode).Return(plan, nil)
	landRepo.On("ValidateOverlap", ctx, mock.AnythingOfType("string"), testBoundary).Return(true, nil)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{
		{AU1Code: "01", AU2Code: "0101", AU3Code: "010101"},
	}, nil)
	landRepo.On("CreateAU", ctx, mock.AnythingOfType("*domain.LandAcquisitionAU")).Return(nil)
	planRepo.On("GetSourceParcels", ctx, testPlanCode).Return([]*domain.SourceParcel{
		{ID: uuid.New(), ParcelID: "010101000001", GeometryWKT: "POLYGON((106.76 47.66,106.82 47.66,106.82 47.72,106.76 47.72,106.76 47.66))"},
	}, nil)
	landRepo.On("BulkInsertParcels", ctx, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.AnythingOfType("string")).Return(nil)

	acq, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: zip,
		CreatedBy:     "user-1",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, acq.ID)
	assert.Equal(t, testPlanCode, acq.PlanCode)
	assert.Equal(t, domain.LandAcquisitionStatusNew, acq.Status)
	assert.Len(t, acq.AUs, 1)
	planRepo.AssertExpectations(t)
	landRepo.AssertExpectations(t)
}

func TestLandService_Create_PlanNotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	planRepo.On("GetByCode", ctx, "UNKNOWN").Return(nil, domain.ErrNotFound)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      "UNKNOWN",
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: buildSimpleShp(),
	})

	assert.Equal(t, domain.ErrNotFound, err)
	planRepo.AssertExpectations(t)
	landRepo.AssertNotCalled(t, "ValidateOverlap")
}

func TestLandService_Create_InvalidShapefile(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	planRepo.On("GetByCode", ctx, testPlanCode).Return(testPlan(), nil)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: []byte("not-a-zip"),
	})

	assert.Equal(t, domain.ErrInvalidInput, err)
	planRepo.AssertExpectations(t)
}

func TestLandService_Create_GeometryOutsideBoundary(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	planRepo.On("GetByCode", ctx, testPlanCode).Return(testPlan(), nil)
	landRepo.On("ValidateOverlap", ctx, mock.AnythingOfType("string"), testBoundary).Return(false, nil)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: buildSimpleShp(),
	})

	assert.Equal(t, domain.ErrInvalidInput, err)
}

func TestLandService_Create_ValidateOverlapError(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	planRepo.On("GetByCode", ctx, testPlanCode).Return(testPlan(), nil)
	landRepo.On("ValidateOverlap", ctx, mock.Anything, mock.Anything).Return(false, assert.AnError)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: buildSimpleShp(),
	})

	assert.Equal(t, domain.ErrInternalServer, err)
}

// ── GetByID ───────────────────────────────────────────────────

func TestLandService_GetByID_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	expected := &domain.LandAcquisition{ID: id, PlanCode: testPlanCode, Status: domain.LandAcquisitionStatusField}
	landRepo.On("GetByID", ctx, id).Return(expected, nil)

	acq, err := svc.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, acq.ID)
	landRepo.AssertExpectations(t)
}

func TestLandService_GetByID_NotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	_, err := svc.GetByID(ctx, id)
	assert.Equal(t, domain.ErrNotFound, err)
}

// ── Update ────────────────────────────────────────────────────

func TestLandService_Update_DatesOnly(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	existing := &domain.LandAcquisition{ID: id, PlanCode: testPlanCode, GeometryWKT: "POLYGON((1 1,2 1,2 2,1 2,1 1))", Status: domain.LandAcquisitionStatusField}
	newEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)

	landRepo.On("GetByID", ctx, id).Return(existing, nil)
	landRepo.On("Update", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)

	acq, geomChanged, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		EndDate: &newEnd,
	})
	require.NoError(t, err)
	assert.False(t, geomChanged)
	assert.Equal(t, newEnd, *acq.EndDate)
	landRepo.AssertNotCalled(t, "DeleteAUs")
	landRepo.AssertNotCalled(t, "DeleteParcelsAndAUs")
}

func TestLandService_Update_GeometryChange(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	oldGeometryWKT := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	existing := &domain.LandAcquisition{ID: id, PlanCode: testPlanCode, GeometryWKT: oldGeometryWKT, Status: domain.LandAcquisitionStatusField}
	plan := testPlan()

	landRepo.On("GetByID", ctx, id).Return(existing, nil)
	planRepo.On("GetByCode", ctx, testPlanCode).Return(plan, nil)
	landRepo.On("ValidateOverlap", ctx, mock.AnythingOfType("string"), testBoundary).Return(true, nil)
	landRepo.On("Update", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("CreateBoundaryHistory", ctx, mock.MatchedBy(func(h *domain.LandAcquisitionBoundaryHistory) bool {
		return h.LandAcquisitionID == id &&
			h.OldGeometryWKT == oldGeometryWKT &&
			h.NewGeometryWKT != "" &&
			h.ID != uuid.Nil
	})).Return(nil)
	landRepo.On("DeleteAUs", ctx, id).Return(nil)
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{}, nil)
	srcParcels := []*domain.SourceParcel{{ParcelID: "123456789012"}}
	planRepo.On("GetSourceParcels", ctx, testPlanCode).Return(srcParcels, nil)
	landRepo.On("BulkInsertParcels", ctx, id, srcParcels, mock.AnythingOfType("string")).Return(nil)

	acq, geomChanged, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		ShapefileData: buildSimpleShp(),
	})
	require.NoError(t, err)
	assert.True(t, geomChanged)
	assert.NotEmpty(t, acq.GeometryWKT)
	landRepo.AssertCalled(t, "CreateBoundaryHistory", ctx, mock.AnythingOfType("*domain.LandAcquisitionBoundaryHistory"))
	landRepo.AssertCalled(t, "DeleteAUs", ctx, id)
	landRepo.AssertCalled(t, "BulkInsertParcels", ctx, id, srcParcels, mock.AnythingOfType("string"))
	landRepo.AssertNotCalled(t, "DeleteParcelsAndAUs")
}

func TestLandService_Update_PlanChangedWithoutShapefile(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	existing := &domain.LandAcquisition{ID: id, PlanCode: testPlanCode}
	landRepo.On("GetByID", ctx, id).Return(existing, nil)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		PlanCode: "PLAN-2025-002",
		// ShapefileData nil — алдаа гарах ёстой
	})
	assert.Equal(t, domain.ErrInvalidInput, err)
}

func TestLandService_Update_NotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{})
	assert.Equal(t, domain.ErrNotFound, err)
}

// ── Delete ────────────────────────────────────────────────────

func TestLandService_Delete_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, Status: domain.LandAcquisitionStatusNew}, nil)
	landRepo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	require.NoError(t, err)
	landRepo.AssertExpectations(t)
}

func TestLandService_Delete_NotNewStatus(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, Status: domain.LandAcquisitionStatusField}, nil)

	err := svc.Delete(ctx, id)
	assert.Equal(t, domain.ErrForbidden, err)
	landRepo.AssertNotCalled(t, "Delete")
}

func TestLandService_Delete_NotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	err := svc.Delete(ctx, id)
	assert.Equal(t, domain.ErrNotFound, err)
}

// ── SyncParcel ────────────────────────────────────────────────

func TestLandService_SyncParcel_Success(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	parcelCode := "010101000001"
	parcelID := uuid.New()
	holderID := uuid.New()

	acq := &domain.LandAcquisition{ID: acqID, PlanCode: testPlanCode}
	parcel := &domain.Parcel{ID: parcelID, ParcelID: parcelCode}
	srcParcel := &domain.SourceParcel{ID: uuid.New(), ParcelID: parcelCode}

	landRepo.On("GetByID", ctx, acqID).Return(acq, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).Return(parcel, nil)
	landRepo.On("UpdateParcelAcquisitionGeom", ctx, parcelID, mock.AnythingOfType("string")).Return(nil)
	planRepo.On("GetSourceParcelByID", ctx, parcelCode).Return(srcParcel, nil)
	landRepo.On("UpsertHolder", ctx, srcParcel).Return(holderID, nil)
	landRepo.On("UpsertParcelRight", ctx, parcelID, acqID, holderID, srcParcel).Return(nil)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	require.NoError(t, err)
	planRepo.AssertExpectations(t)
	landRepo.AssertExpectations(t)
}

func TestLandService_SyncParcel_AcquisitionNotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	parcelCode := "010101000001"
	landRepo.On("GetByID", ctx, acqID).Return(nil, domain.ErrNotFound)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	assert.Equal(t, domain.ErrNotFound, err)
	landRepo.AssertNotCalled(t, "GetParcelByCode")
}

func TestLandService_SyncParcel_ParcelNotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	parcelCode := "010101000001"
	acq := &domain.LandAcquisition{ID: acqID}

	landRepo.On("GetByID", ctx, acqID).Return(acq, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).Return(nil, domain.ErrNotFound)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	assert.Equal(t, domain.ErrNotFound, err)
	planRepo.AssertNotCalled(t, "GetSourceParcelByID")
}

func TestLandService_SyncParcel_SourceNotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := newLandSvc(planRepo, landRepo)

	acqID := uuid.New()
	parcelCode := "MISSING"
	parcelID := uuid.New()
	acq := &domain.LandAcquisition{ID: acqID}
	parcel := &domain.Parcel{ID: parcelID, ParcelID: parcelCode}

	landRepo.On("GetByID", ctx, acqID).Return(acq, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).Return(parcel, nil)
	landRepo.On("UpdateParcelAcquisitionGeom", ctx, parcelID, mock.AnythingOfType("string")).Return(nil)
	planRepo.On("GetSourceParcelByID", ctx, parcelCode).Return(nil, domain.ErrNotFound)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	assert.Equal(t, domain.ErrNotFound, err)
	landRepo.AssertNotCalled(t, "UpsertHolder")
}
