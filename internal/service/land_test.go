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

// buildSimpleShp нь нэг хэлбэр бүхий polygon .shp файлын байт буцаана
// (Улаанбаатар орчмын WGS84 координат — стандартад тохирсон).
func buildSimpleShp() []byte {
	return buildShpPts([][2]float64{
		{106.76, 47.66}, {106.82, 47.66},
		{106.82, 47.72}, {106.76, 47.72}, {106.76, 47.66},
	})
}

// buildShpPts — заасан координатаар .shp байт бүтээнэ (проекцийн шалгалтыг
// тестлэхэд метр координат ч дамжуулж болно).
func buildShpPts(pts [][2]float64) []byte {
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

// planMocks — төлөвлөгөөний ШИНЭ урсгалын mock-ууд: дундын сервисээс татаж,
// appdb-ийн plan хүснэгтэд хадгална. authdb-д plan байхгүй тул өөр эх сурвалж
// байхгүй.
type planMocks struct {
	projects *mocks.MockPlanProjectClient
	store    *mocks.MockPlanStoreRepository
}

// expectPlan — дундын сервисээс төлөвлөгөө олдож, plan хүснэгтэд хадгалагдана.
func (p planMocks) expectPlan(code string, plan *domain.Plan) {
	p.projects.On("GetPlanProject", ctx, code).Return(plan, nil)
	p.store.On("Upsert", ctx, mock.AnythingOfType("*domain.Plan")).Return(plan, nil)
}

// expectPlanMissing — дундын сервис төлөвлөгөө олоогүй.
func (p planMocks) expectPlanMissing(code string) {
	p.projects.On("GetPlanProject", ctx, code).Return(nil, domain.ErrNotFound)
}

// testAcqAreaM2 — тестийн shapefile-ийн (buildSimpleShp) талбай.
const testAcqAreaM2 = 41_250_000.0

// expectBoundaryOK — хилийн шалгалтыг ӨНГӨРҮҮЛЭХ mock: хил бүхэлдээ НЭГ
// сум/дүүрэгт багтаж, төлөвлөгөөний хилээс гараагүй.
func expectBoundaryOK(landRepo *mocks.MockLandAcquisitionRepository) {
	landRepo.On("FindAU2Coverage", ctx, mock.AnythingOfType("string")).Return(&domain.AU2Coverage{
		ReferenceRows: 340,
		Overlaps: []domain.AU2Overlap{
			{Code: "0101", Name: "Сүхбаатар", Covered: true, AreaM2: testAcqAreaM2},
		},
	}, nil).Maybe()
	landRepo.On("CheckContainment", ctx,
		mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&domain.GeomContainment{Covered: true, ContainerM2: 900_000_000}, nil).Maybe()
}

func newLandSvc(landRepo *mocks.MockLandAcquisitionRepository) (service.LandService, planMocks) {
	pm := planMocks{
		projects: &mocks.MockPlanProjectClient{},
		store:    &mocks.MockPlanStoreRepository{},
	}
	return service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, pm.projects, pm.store), pm
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
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)

	plan := testPlan()
	zip := buildSimpleShp()

	pm.expectPlan(testPlanCode, plan)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	expectBoundaryOK(landRepo)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{
		{AU1Code: "01", AU2Code: "0101", AU3Code: "010101"},
	}, nil)
	landRepo.On("CreateAU", ctx, mock.AnythingOfType("*domain.LandAcquisitionAU")).Return(nil)

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
	// Талбай нь гараас оруулалгүй, хилээс автоматаар тооцогдоно
	assert.Equal(t, 41_250_000.0, acq.AreaM2)
	pm.projects.AssertExpectations(t)
	landRepo.AssertExpectations(t)
}

// Чөлөөлөлт ҮҮСГЭХ үед нэгж талбар автоматаар ТАТАГДАХГҮЙ.
func TestLandService_Create_DoesNotFetchParcels(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	pm := planMocks{
		projects: &mocks.MockPlanProjectClient{},
		store:    &mocks.MockPlanStoreRepository{},
	}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo, pm.projects, pm.store)

	pm.expectPlan(testPlanCode, testPlan())
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	expectBoundaryOK(landRepo)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("CreateProgress", ctx, mock.Anything).Return(nil).Maybe()
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{}, nil)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})

	require.NoError(t, err)
	parcelInfo.AssertNotCalled(t, "ListParcelIDsByAcquisition", mock.Anything, mock.Anything)
	parcelInfo.AssertNotCalled(t, "GetParcelInfo", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "BulkInsertParcels", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLandService_Create_PlanNotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)

	pm.expectPlanMissing("UNKNOWN")

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      "UNKNOWN",
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: buildSimpleShp(),
	})

	assert.Equal(t, domain.ErrNotFound, err)
	pm.projects.AssertExpectations(t)
	landRepo.AssertNotCalled(t, "Create")
}

func TestLandService_Create_InvalidShapefile(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)

	pm.expectPlan(testPlanCode, testPlan())

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: []byte("not-a-zip"),
	})

	assert.Equal(t, domain.ErrShapefileFormat, err)
	pm.projects.AssertExpectations(t)
}

// Метр проекцтой (UTM) файл — координат градусын хүрээнд байхгүй тул
// бүртгэгдэхгүй, шалтгаан нь ТОДОРХОЙ алдаа болж буцна.
func TestLandService_Create_RejectsNonWGS84Shapefile(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)
	pm.expectPlan(testPlanCode, testPlan())

	// UTM 48N метр координат (X≈478000, Y≈5279000)
	utm := buildShpPts([][2]float64{
		{478000, 5279000}, {478500, 5279000},
		{478500, 5279500}, {478000, 5279500}, {478000, 5279000},
	})

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		ShapefileData: utm,
	})

	assert.Equal(t, domain.ErrShapefileProjection, err)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// WGS84 градус боловч Монголын гадна байрлах хил бүртгэгдэхгүй.
func TestLandService_Create_RejectsForeignCoordinates(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)
	pm.expectPlan(testPlanCode, testPlan())

	// Берлин орчим
	foreign := buildShpPts([][2]float64{
		{13.40, 52.51}, {13.41, 52.51}, {13.41, 52.52}, {13.40, 52.52}, {13.40, 52.51},
	})

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		ShapefileData: foreign,
	})

	assert.Equal(t, domain.ErrShapefileOutsideCountry, err)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// ── Хилийн шалгалт: сум/дүүрэг дамнах ────────────────────────

// createBoundarySvc — хилийн шалгалтыг тест хийх бэлтгэл. Шалгалт өнгөрвөл
// бүртгэгдэх бүх mock тавигдана (Maybe: шалгалт унавал дуудагдахгүй).
func createBoundarySvc(landRepo *mocks.MockLandAcquisitionRepository) (service.LandService, planMocks) {
	svc, pm := newLandSvc(landRepo)
	pm.expectPlan(testPlanCode, testPlan())
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(testAcqAreaM2, nil)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil).Maybe()
	landRepo.On("CreateProgress", ctx, mock.Anything).Return(nil).Maybe()
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).
		Return([]domain.LandAcquisitionAU{}, nil).Maybe()
	return svc, pm
}

func createWithShapefile(svc service.LandService) (*domain.LandAcquisition, error) {
	return svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		EndDate:       &testEndDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})
}

// expectAU2 — сум/дүүргийн давхцлын mock.
func expectAU2(landRepo *mocks.MockLandAcquisitionRepository, refRows int, overlaps ...domain.AU2Overlap) {
	landRepo.On("FindAU2Coverage", ctx, mock.AnythingOfType("string")).
		Return(&domain.AU2Coverage{ReferenceRows: refRows, Overlaps: overlaps}, nil)
}

// expectPlanContainment — төлөвлөгөөний хилд багтах эсэхийн mock.
func expectPlanContainment(landRepo *mocks.MockLandAcquisitionRepository, covered bool, outsideM2, planM2 float64) {
	landRepo.On("CheckContainment", ctx,
		mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&domain.GeomContainment{
			Covered: covered, OutsideM2: outsideM2, ContainerM2: planM2,
		}, nil)
}

// Хоёр сум/дүүрэг дамнасан хил бүртгэгдэхгүй (нэг ч au2 бүрэн бүрхээгүй).
func TestLandService_Create_RejectsAU2Crossing(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	expectAU2(landRepo, 340,
		domain.AU2Overlap{Code: "0101", AreaM2: testAcqAreaM2 * 0.6},
		domain.AU2Overlap{Code: "0102", AreaM2: testAcqAreaM2 * 0.4},
	)

	_, err := createWithShapefile(svc)

	assert.Equal(t, domain.ErrAU2Crossing, err)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Сум/дүүргийн хилээс ГАДНА гарсан хэсэг байвал бүртгэгдэхгүй.
func TestLandService_Create_RejectsOutsideAU2(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	// Нэг сумтай давхцсан ч хилийн 30% нь ямар ч сумд хамаарахгүй тул
	// бүрэн бүрхэгдээгүй (Covered=false).
	expectAU2(landRepo, 340, domain.AU2Overlap{Code: "0101", AreaM2: testAcqAreaM2 * 0.7})

	_, err := createWithShapefile(svc)

	assert.Equal(t, domain.ErrAU2Crossing, err)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Хөрш сум руу ХЭДХЭН м² орсон ч дамнасанд тооцно — хувиар хүлцэл БАЙХГҮЙ.
func TestLandService_Create_RejectsSmallAU2Overlap(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	// 50 м² нь хилээс гарсан тул аль ч сум хилийг бүрэн бүрхэхгүй.
	expectAU2(landRepo, 340,
		domain.AU2Overlap{Code: "0101", AreaM2: testAcqAreaM2 - 50},
		domain.AU2Overlap{Code: "0102", AreaM2: 50},
	)

	_, err := createWithShapefile(svc)

	assert.Equal(t, domain.ErrAU2Crossing, err)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Хил нь сумын хилийн шугамтай давхцаж таарсан (хөрш сумтай зүгээр
// ХҮРЭЛЦСЭН) тохиолдол дамналт БИШ: ST_CoveredBy нь шугам дээрхийг багтсанд
// тооцно.
func TestLandService_Create_AllowsAU2BorderTouch(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	expectAU2(landRepo, 340,
		domain.AU2Overlap{Code: "0101", Covered: true, AreaM2: testAcqAreaM2},
		domain.AU2Overlap{Code: "0102", AreaM2: 0},
	)
	expectPlanContainment(landRepo, true, 0, 900_000_000)

	acq, err := createWithShapefile(svc)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, acq.ID)
}

// au2 лавлах геометр ачаалагдаагүй бол шалгалт алгасагдана (бүртгэл зогсохгүй).
func TestLandService_Create_SkipsAU2CheckWithoutReferenceData(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	expectAU2(landRepo, 0)
	expectPlanContainment(landRepo, true, 0, 900_000_000)

	_, err := createWithShapefile(svc)

	require.NoError(t, err)
}

// ── Хилийн шалгалт: төлөвлөгөөний нэгж талбар ────────────────

// Төлөвлөгөөний хилээс гарсан хэсэг 50%-иас БАГА бол бүртгэгдэнэ.
func TestLandService_Create_AllowsOutsidePlanUnderHalf(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	expectAU2(landRepo, 340, domain.AU2Overlap{Code: "0101", Covered: true, AreaM2: testAcqAreaM2})
	expectPlanContainment(landRepo, false, 40_000_000, 100_000_000) // 40%

	acq, err := createWithShapefile(svc)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, acq.ID)
}

// Төлөвлөгөөний хилээс гарсан хэсэг 50%-иас ИХ бол бүртгэгдэхгүй.
func TestLandService_Create_RejectsOutsidePlanOverHalf(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := createBoundarySvc(landRepo)
	expectAU2(landRepo, 340, domain.AU2Overlap{Code: "0101", Covered: true, AreaM2: testAcqAreaM2})
	expectPlanContainment(landRepo, false, 60_000_000, 100_000_000) // 60%

	_, err := createWithShapefile(svc)

	assert.Equal(t, domain.ErrOutsidePlanExceeds, err)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Төлөвлөгөөний хил байхгүй бол багтах шалгалт алгасагдана.
func TestLandService_Create_SkipsPlanCheckWithoutBoundary(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)
	noGeomPlan := testPlan()
	noGeomPlan.BoundaryWKT = ""
	pm.expectPlan(testPlanCode, noGeomPlan)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(testAcqAreaM2, nil)
	expectAU2(landRepo, 340, domain.AU2Overlap{Code: "0101", Covered: true, AreaM2: testAcqAreaM2})
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).
		Return([]domain.LandAcquisitionAU{}, nil)

	_, err := createWithShapefile(svc)

	require.NoError(t, err)
	landRepo.AssertNotCalled(t, "CheckContainment", mock.Anything, mock.Anything, mock.Anything)
}

// ХИЛ ШИНЭЧЛЭХ үед ч шалгалт хийгдэнэ: сум/дүүрэг дамнасан шинэ хил
// хадгалагдахгүй (Update-ээр тойрч гарах боломжгүй).
func TestLandService_Update_RejectsAU2Crossing(t *testing.T) {
	id := uuid.New()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, pm := newLandSvc(landRepo)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{
		ID: id, PlanCode: testPlanCode,
		GeometryWKT: "POLYGON((106.76 47.66,106.82 47.66,106.82 47.72,106.76 47.72,106.76 47.66))",
		Status:      domain.LandAcquisitionStatusField,
	}, nil)
	pm.expectPlan(testPlanCode, testPlan())
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(testAcqAreaM2, nil)
	expectAU2(landRepo, 340,
		domain.AU2Overlap{Code: "0101", AreaM2: testAcqAreaM2 * 0.5},
		domain.AU2Overlap{Code: "0102", AreaM2: testAcqAreaM2 * 0.5},
	)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		ShapefileData:   buildSimpleShp(),
		UpdatedBy:       "user-1",
	})

	assert.Equal(t, domain.ErrAU2Crossing, err)
	landRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "CreateBoundaryHistory", mock.Anything, mock.Anything)
}

// updateGeomSvc — ХИЛ ШИНЭЧЛЭХ (Update) урсгалын нэгж талбар татах хэсгийг
// шалгах бэлтгэл. Чөлөөлөлт үүсгэх үед нэгж талбар татагддаггүй тул нэгж
// талбарын логик Update дээр шалгагдана.
func updateGeomSvc(t *testing.T, id uuid.UUID) (
	service.LandService,
	*mocks.MockLandAcquisitionRepository,
	*mocks.MockParcelInfoClient,
) {
	t.Helper()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	pm := planMocks{
		projects: &mocks.MockPlanProjectClient{},
		store:    &mocks.MockPlanStoreRepository{},
	}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo, pm.projects, pm.store)

	existing := &domain.LandAcquisition{
		ID: id, PlanCode: testPlanCode,
		GeometryWKT: "POLYGON((1 1,2 1,2 2,1 2,1 1))",
		Status:      domain.LandAcquisitionStatusField,
	}
	landRepo.On("GetByID", ctx, id).Return(existing, nil)
	pm.expectPlan(testPlanCode, testPlan())
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	expectBoundaryOK(landRepo)
	landRepo.On("Update", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("CreateBoundaryHistory", ctx, mock.Anything).Return(nil)
	landRepo.On("DeleteAUs", ctx, id).Return(nil)
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{}, nil)
	return svc, landRepo, parcelInfo
}

// Нэгж талбарын мэдээлэл ЗӨВХӨН дугаараар үүснэ: чөлөөлөх хилээр дугаарууд
// авагдаж, дугаар тус бүрийн мэдээлэл (геометргүй байсан ч) бүртгэгдэнэ.
func TestLandService_Update_CreatesParcelsFromParcelIDsOnly(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := updateGeomSvc(t, id)

	// Хилээр 4 дугаар ирнэ (нэг нь хоосон, нэг нь давхардсан)
	parcelInfo.On("ListParcelIDsByAcquisition", ctx, mock.AnythingOfType("string")).
		Return([]string{"010101000001", " ", "010101000002", "010101000001"}, nil)
	// Мэдээлэл нь геометр БУЦААХГҮЙ — гэхдээ бүртгэгдэх ёстой
	parcelInfo.On("GetParcelInfo", ctx, "010101000001").
		Return(&domain.SourceParcel{ParcelID: "010101000001", AreaM2: 500}, nil)
	// Дугаарыг хариултад буцаагаагүй ч дуудсан дугаараар бүртгэгдэнэ
	parcelInfo.On("GetParcelInfo", ctx, "010101000002").
		Return(&domain.SourceParcel{AreaM2: 700}, nil)

	var saved []*domain.SourceParcel
	landRepo.On("BulkInsertParcels", ctx, id, mock.Anything, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			saved = args.Get(2).([]*domain.SourceParcel)
		}).Return(nil)

	_, geomChanged, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		ShapefileData: buildSimpleShp(),
	})

	require.NoError(t, err)
	assert.True(t, geomChanged)
	require.Len(t, saved, 2, "хоосон дугаар алгасагдаж, давхардсан нь нэг л удаа бүртгэгдэнэ")
	assert.Equal(t, "010101000001", saved[0].ParcelID)
	assert.Equal(t, "010101000002", saved[1].ParcelID)
	assert.Empty(t, saved[0].GeometryWKT, "геометр байхгүй ч нэгж талбар үүснэ")
	// auth DB-ийн хуучин урсгал ХЭРЭГЛЭГДЭХГҮЙ
	parcelInfo.AssertExpectations(t)
}

// Нэг дугаарын мэдээлэл татагдахгүй бол ЗӨВХӨН тэр нь алгасагдана —
// бусад нэгж талбар жагсаалтад бүртгэгдсэн хэвээр байна.
func TestLandService_Update_SkipsOnlyFailedParcelInfo(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo := updateGeomSvc(t, id)

	parcelInfo.On("ListParcelIDsByAcquisition", ctx, mock.AnythingOfType("string")).
		Return([]string{"010101000001", "010101000002"}, nil)
	parcelInfo.On("GetParcelInfo", ctx, "010101000001").Return(nil, domain.ErrNotFound)
	parcelInfo.On("GetParcelInfo", ctx, "010101000002").
		Return(&domain.SourceParcel{ParcelID: "010101000002"}, nil)

	var saved []*domain.SourceParcel
	landRepo.On("BulkInsertParcels", ctx, id, mock.Anything, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			saved = args.Get(2).([]*domain.SourceParcel)
		}).Return(nil)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		ShapefileData: buildSimpleShp(),
	})

	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.Equal(t, "010101000002", saved[0].ParcelID)
}

func TestLandService_AdvanceStatus_ConfirmedRequiresFinalParcelStatuses(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)
	acqID := uuid.New()
	decreeDate := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{
		ID:     acqID,
		Status: domain.LandAcquisitionStatusField,
	}, nil)
	landRepo.On("ListAcquisitionWorkflows", ctx).Return([]*domain.AcquisitionWorkflow{}, nil)
	landRepo.On("HasNonFinalParcels", ctx, acqID).Return(true, nil)

	_, err := svc.AdvanceStatus(
		ctx,
		acqID,
		domain.LandAcquisitionStatusVerified,
		"",
		"user-1",
		"А/001",
		&decreeDate,
	)

	assert.Equal(t, domain.ErrNonFinalParcels, err)
	landRepo.AssertNotCalled(t, "CreateProgress", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "UpdateDecree", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	landRepo.AssertExpectations(t)
}

func TestLandService_AdvanceStatus_ConfirmedAllowsFinalParcelStatuses(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)
	acqID := uuid.New()
	decreeDate := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{
		ID:     acqID,
		Status: domain.LandAcquisitionStatusField,
	}, nil)
	landRepo.On("ListAcquisitionWorkflows", ctx).Return([]*domain.AcquisitionWorkflow{}, nil)
	landRepo.On("HasNonFinalParcels", ctx, acqID).Return(false, nil)
	landRepo.On("CreateProgress", ctx, mock.AnythingOfType("*domain.AcquisitionProgress")).Return(nil)
	landRepo.On("UpdateStatus", ctx, acqID, domain.LandAcquisitionStatusVerified).Return(nil)
	landRepo.On("UpdateDecree", ctx, acqID, "А/001", &decreeDate).Return(nil)

	acq, err := svc.AdvanceStatus(
		ctx,
		acqID,
		domain.LandAcquisitionStatusVerified,
		"",
		"user-1",
		"А/001",
		&decreeDate,
	)

	require.NoError(t, err)
	assert.Equal(t, domain.LandAcquisitionStatusVerified, acq.Status)
	landRepo.AssertExpectations(t)
}

// ── GetByID ───────────────────────────────────────────────────

func TestLandService_GetByID_Success(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	id := uuid.New()
	expected := &domain.LandAcquisition{ID: id, PlanCode: testPlanCode, Status: domain.LandAcquisitionStatusField}
	landRepo.On("GetByID", ctx, id).Return(expected, nil)

	acq, err := svc.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, acq.ID)
	landRepo.AssertExpectations(t)
}

func TestLandService_GetByID_NotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	_, err := svc.GetByID(ctx, id)
	assert.Equal(t, domain.ErrNotFound, err)
}

// ── Update ────────────────────────────────────────────────────

func TestLandService_Update_DatesOnly(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

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
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	pm := planMocks{
		projects: &mocks.MockPlanProjectClient{},
		store:    &mocks.MockPlanStoreRepository{},
	}
	// Нэгж талбар нь ЗӨВХӨН дундын сервисээс татагдана.
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, parcelInfo, pm.projects, pm.store)

	id := uuid.New()
	oldGeometryWKT := "POLYGON((1 1,2 1,2 2,1 2,1 1))"
	existing := &domain.LandAcquisition{ID: id, PlanCode: testPlanCode, GeometryWKT: oldGeometryWKT, Status: domain.LandAcquisitionStatusField}
	plan := testPlan()

	landRepo.On("GetByID", ctx, id).Return(existing, nil)
	pm.expectPlan(testPlanCode, plan)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	expectBoundaryOK(landRepo)
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
	parcelInfo.On("ListParcelIDsByAcquisition", ctx, mock.AnythingOfType("string")).
		Return([]string{"123456789012"}, nil)
	parcelInfo.On("GetParcelInfo", ctx, "123456789012").Return(srcParcels[0], nil)
	landRepo.On("BulkInsertParcels", ctx, id, srcParcels, mock.AnythingOfType("string")).Return(nil)

	acq, geomChanged, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		ShapefileData: buildSimpleShp(),
	})
	require.NoError(t, err)
	assert.True(t, geomChanged)
	assert.NotEmpty(t, acq.GeometryWKT)
	assert.Equal(t, 41_250_000.0, acq.AreaM2, "шинэ хилээр талбай автоматаар шинэчлэгдэх ёстой")
	landRepo.AssertCalled(t, "CreateBoundaryHistory", ctx, mock.AnythingOfType("*domain.LandAcquisitionBoundaryHistory"))
	landRepo.AssertCalled(t, "DeleteAUs", ctx, id)
	landRepo.AssertCalled(t, "BulkInsertParcels", ctx, id, srcParcels, mock.AnythingOfType("string"))
	landRepo.AssertNotCalled(t, "DeleteParcelsAndAUs")
}

func TestLandService_Update_PlanChangedWithoutShapefile(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

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
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{})
	assert.Equal(t, domain.ErrNotFound, err)
}

// ── Delete ────────────────────────────────────────────────────

func TestLandService_Delete_Success(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, Status: domain.LandAcquisitionStatusNew}, nil)
	landRepo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	require.NoError(t, err)
	landRepo.AssertExpectations(t)
}

func TestLandService_Delete_NotNewStatus(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{ID: id, Status: domain.LandAcquisitionStatusField}, nil)

	err := svc.Delete(ctx, id)
	assert.Equal(t, domain.ErrForbidden, err)
	landRepo.AssertNotCalled(t, "Delete")
}

func TestLandService_Delete_NotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	id := uuid.New()
	landRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

	err := svc.Delete(ctx, id)
	assert.Equal(t, domain.ErrNotFound, err)
}

// ── SyncParcel ────────────────────────────────────────────────

func TestLandService_SyncParcel_Success(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	pm := planMocks{
		projects: &mocks.MockPlanProjectClient{},
		store:    &mocks.MockPlanStoreRepository{},
	}
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, parcelInfo, pm.projects, pm.store)

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
	parcelInfo.On("GetParcelInfo", ctx, parcelCode).Return(srcParcel, nil)
	landRepo.On("UpsertHolder", ctx, srcParcel).Return(holderID, nil)
	landRepo.On("ReplaceParcelDetail", ctx, parcelID, acqID, holderID, srcParcel).Return(nil)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	require.NoError(t, err)
	pm.projects.AssertExpectations(t)
	landRepo.AssertExpectations(t)
}

func TestLandService_SyncParcel_AcquisitionNotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)

	acqID := uuid.New()
	parcelCode := "010101000001"
	landRepo.On("GetByID", ctx, acqID).Return(nil, domain.ErrNotFound)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	assert.Equal(t, domain.ErrNotFound, err)
	landRepo.AssertNotCalled(t, "GetParcelByCode")
}

func TestLandService_SyncParcel_ParcelNotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, parcelInfo)

	acqID := uuid.New()
	parcelCode := "010101000001"
	acq := &domain.LandAcquisition{ID: acqID}

	landRepo.On("GetByID", ctx, acqID).Return(acq, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).Return(nil, domain.ErrNotFound)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	assert.Equal(t, domain.ErrNotFound, err)
	parcelInfo.AssertNotCalled(t, "GetParcelInfo")
}

func TestLandService_SyncParcel_SourceNotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, filestorage.DisabledStorage{}, nil, parcelInfo)

	acqID := uuid.New()
	parcelCode := "MISSING"
	parcelID := uuid.New()
	acq := &domain.LandAcquisition{ID: acqID}
	parcel := &domain.Parcel{ID: parcelID, ParcelID: parcelCode}

	landRepo.On("GetByID", ctx, acqID).Return(acq, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).Return(parcel, nil)
	landRepo.On("UpdateParcelAcquisitionGeom", ctx, parcelID, mock.AnythingOfType("string")).Return(nil)
	parcelInfo.On("GetParcelInfo", ctx, parcelCode).Return(nil, domain.ErrNotFound)

	err := svc.SyncParcel(ctx, acqID, parcelCode)
	assert.Equal(t, domain.ErrNotFound, err)
	landRepo.AssertNotCalled(t, "UpsertHolder")
}

// ── SyncParcel: эзэмшигчид (олон хүн) ──────────────────────────

// syncHolderSvc — дундын сервистэй sync-ийн бэлтгэл: /parcel/info дуудагдаж,
// дэлгэрэнгүй хадгалагдана. Эзэмшигчийн дуудалт нь тест тус бүрд тодорхойлогдоно.
func syncHolderSvc(t *testing.T, acqID, parcelID uuid.UUID, parcelCode, appNo string) (
	service.LandService,
	*mocks.MockLandAcquisitionRepository,
	*mocks.MockParcelInfoClient,
	*domain.SourceParcel,
) {
	t.Helper()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo)

	src := &domain.SourceParcel{ParcelID: parcelCode, AppNo: appNo}
	holderID := uuid.New()

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{ID: acqID}, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).
		Return(&domain.Parcel{ID: parcelID, ParcelID: parcelCode}, nil)
	landRepo.On("UpdateParcelAcquisitionGeom", ctx, parcelID, mock.AnythingOfType("string")).Return(nil)
	parcelInfo.On("GetParcelInfo", ctx, parcelCode).Return(src, nil)
	landRepo.On("UpsertHolder", ctx, src).Return(holderID, nil)
	landRepo.On("ReplaceParcelDetail", ctx, parcelID, acqID, holderID, src).Return(nil)
	return svc, landRepo, parcelInfo, src
}

// SyncParcel нь эзэмшигчийг ТАТАХГҮЙ — тэр нь SyncParcelHolders-ийн ажил
// (дэлгэц дээр тусдаа алхам болсон).
func TestLandService_SyncParcel_DoesNotSyncHolders(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "2024-0001"
	svc, landRepo, parcelInfo, _ := syncHolderSvc(t, acqID, parcelID, parcelCode, appNo)

	require.NoError(t, svc.SyncParcel(ctx, acqID, parcelCode))

	parcelInfo.AssertNotCalled(t, "ListApplicationPersons", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "ReplaceParcelHolders",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	landRepo.AssertExpectations(t)
}

// SyncParcelHolders — хадгалагдсан app_no-гоор эзэмшигчдийг татаж, БҮГДИЙГ
// хадгалж, үндсэн хүсэлт гаргагчийг parcel_detail-д холбоно.
func TestLandService_SyncParcelHolders_SavesAllByAppNo(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "2024-0001"
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo)

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{ID: acqID}, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).
		Return(&domain.Parcel{ID: parcelID, ParcelID: parcelCode}, nil)
	landRepo.On("GetParcelDetailByID", ctx, parcelID).
		Return(&domain.ParcelFull{Detail: &domain.ParcelDetailInfo{AppNo: appNo}}, nil)

	persons := []*domain.ParcelHolder{
		{LastName: "Дорж", Name: "Бат", RegisterNo: "УА00112233", MainApplicant: true},
		{LastName: "Дорж", Name: "Болд", RegisterNo: "УА00445566"},
	}
	parcelInfo.On("ListApplicationPersons", ctx, appNo).Return(persons, nil)

	mainHolderID := uuid.New()
	landRepo.On("ReplaceParcelHolders", ctx, parcelID, acqID, appNo, persons).Return(mainHolderID, nil)
	landRepo.On("SetParcelDetailHolder", ctx, parcelID, mainHolderID).Return(nil)

	res, err := svc.SyncParcelHolders(ctx, acqID, parcelCode)

	require.NoError(t, err)
	assert.Equal(t, 2, res.Found)
	assert.Equal(t, 2, res.Saved)
	landRepo.AssertExpectations(t)
	parcelInfo.AssertExpectations(t)
	// Хадгалахаар илгээсэн эзэмшигч бүр app_no-гоор тэмдэглэгдэнэ
	assert.Equal(t, appNo, persons[0].AppNo)
	assert.Equal(t, appNo, persons[1].AppNo)
}

// app_no хадгалагдаагүй бол 422 — эхлээд /sync ажиллуулах шаардлагатай.
func TestLandService_SyncParcelHolders_AppNoMissing(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode := "010101000001"
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo)

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{ID: acqID}, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).Return(&domain.Parcel{ID: parcelID}, nil)
	landRepo.On("GetParcelDetailByID", ctx, parcelID).
		Return(&domain.ParcelFull{Detail: &domain.ParcelDetailInfo{AppNo: "  "}}, nil)

	res, err := svc.SyncParcelHolders(ctx, acqID, parcelCode)

	assert.Nil(t, res)
	assert.ErrorIs(t, err, domain.ErrParcelAppNoMissing)
	parcelInfo.AssertNotCalled(t, "ListApplicationPersons", mock.Anything, mock.Anything)
}

// Эзэмшигч татахад алдаа гарвал sync унахгүй — дэлгэрэнгүй нь хадгалагдсан.
func TestLandService_SyncParcel_HolderFetchFailureDoesNotFailSync(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "2024-0001"
	svc, landRepo, parcelInfo, _ := syncHolderSvc(t, acqID, parcelID, parcelCode, appNo)

	parcelInfo.On("ListApplicationPersons", ctx, appNo).
		Return(nil, domain.ErrInternalServer)

	require.NoError(t, svc.SyncParcel(ctx, acqID, parcelCode))
	landRepo.AssertNotCalled(t, "ReplaceParcelHolders",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "SetParcelDetailHolder", mock.Anything, mock.Anything, mock.Anything)
}

// app_no хоосон бол эзэмшигч ОГТ татагдахгүй: нэгж талбарын дугаараар шүүвэл
// тухайн талбарын бүх хүсэлтийн эзэмшигчид хольцолдоно. Хадгалагдсан
// эзэмшигчид ХӨНДӨГДӨХГҮЙ (устгах нь өгөгдөл гээхэд хүргэнэ), sync унахгүй.
func TestLandService_SyncParcel_SkipsHoldersWhenAppNoEmpty(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode := "010101000001"
	svc, landRepo, parcelInfo, _ := syncHolderSvc(t, acqID, parcelID, parcelCode, "")

	require.NoError(t, svc.SyncParcel(ctx, acqID, parcelCode))

	parcelInfo.AssertNotCalled(t, "ListApplicationPersons", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "ReplaceParcelHolders",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "SetParcelDetailHolder", mock.Anything, mock.Anything, mock.Anything)
}

// Эх системд эзэмшигч үлдээгүй бол хадгалагдсан эзэмшигчид УСТГАГДАНА
// (sync нь орлуулах үйлдэл).
func TestLandService_SyncParcelHolders_ClearsWhenSourceEmpty(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "2024-0001"
	svc, landRepo, parcelInfo, _ := syncHolderSvc(t, acqID, parcelID, parcelCode, appNo)
	landRepo.On("GetParcelDetailByID", ctx, parcelID).
		Return(&domain.ParcelFull{Detail: &domain.ParcelDetailInfo{AppNo: appNo}}, nil)

	parcelInfo.On("ListApplicationPersons", ctx, appNo).
		Return([]*domain.ParcelHolder{}, nil)
	landRepo.On("ReplaceParcelHolders", ctx, parcelID, acqID, appNo, []*domain.ParcelHolder{}).
		Return(uuid.Nil, nil)
	landRepo.On("SetParcelDetailHolder", ctx, parcelID, uuid.Nil).Return(nil)

	_, err := svc.SyncParcelHolders(ctx, acqID, parcelCode)
	require.NoError(t, err)
	landRepo.AssertCalled(t, "ReplaceParcelHolders", ctx, parcelID, acqID, appNo, []*domain.ParcelHolder{})
}

// Регистрийн дугаар давхцсан эзэмшигчийг хадгалахгүй алгасна (эхний нь үлдэнэ).
func TestLandService_SyncParcelHolders_SkipsDuplicateRegisterNo(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode, appNo := "010101000001", "2024-0001"
	svc, landRepo, parcelInfo, _ := syncHolderSvc(t, acqID, parcelID, parcelCode, appNo)
	landRepo.On("GetParcelDetailByID", ctx, parcelID).
		Return(&domain.ParcelFull{Detail: &domain.ParcelDetailInfo{AppNo: appNo}}, nil)

	parcelInfo.On("ListApplicationPersons", ctx, appNo).Return([]*domain.ParcelHolder{
		{LastName: "Дорж", Name: "Бат", RegisterNo: "УА00112233", MainApplicant: true},
		// Ижил регистр (том/жижиг үсэг, зай хамаарахгүй) → алгасагдана
		{LastName: "Дорж", Name: "Бат", RegisterNo: " уа00112233 "},
		{LastName: "Дорж", Name: "Болд", RegisterNo: "УА00445566"},
		// Регистргүй хүн давхцал гэж тооцогдохгүй
		{LastName: "Цэрэн", Name: "Сүх"},
		{LastName: "Цэрэн", Name: "Мөнх"},
	}, nil)

	var saved []*domain.ParcelHolder
	mainHolderID := uuid.New()
	landRepo.On("ReplaceParcelHolders", ctx, parcelID, acqID, appNo, mock.Anything).
		Run(func(args mock.Arguments) {
			saved = args.Get(4).([]*domain.ParcelHolder)
		}).Return(mainHolderID, nil)
	landRepo.On("SetParcelDetailHolder", ctx, parcelID, mainHolderID).Return(nil)

	_, err := svc.SyncParcelHolders(ctx, acqID, parcelCode)
	require.NoError(t, err)

	require.Len(t, saved, 4, "давхцсан регистртэй нэг мөр л хаягдана")
	assert.Equal(t, "УА00112233", saved[0].RegisterNo)
	assert.True(t, saved[0].MainApplicant, "давхцалд ҮНДСЭН хүсэлт гаргагч үлдэнэ")
	assert.Equal(t, "УА00445566", saved[1].RegisterNo)
	assert.Equal(t, "Сүх", saved[2].Name)
	assert.Equal(t, "Мөнх", saved[3].Name)
}

// Sync үед ИРСЭН геометрээр чөлөөлөгдөх талбайн хил/хэмжээ, үлдэх талбай,
// АУ код тооцогдож хадгалагдана.
func TestLandService_SyncParcel_ComputesAcquisitionAreaFromSourceGeometry(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	parcelInfo := &mocks.MockParcelInfoClient{}
	svc := service.NewLandService(landRepo, nil, nil, parcelInfo)

	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode := "1800805431"
	parcelWKT := "POLYGON((106.815 47.9354,106.8163 47.9354,106.8163 47.9352,106.815 47.9354))"
	acqWKT := "POLYGON((106.80 47.93,106.83 47.93,106.83 47.94,106.80 47.94,106.80 47.93))"

	src := &domain.SourceParcel{ParcelID: parcelCode, GeometryWKT: parcelWKT, AreaM2: 49235, AppNo: "01116-54-00006-26"}
	holderID := uuid.New()

	landRepo.On("GetByID", ctx, acqID).Return(&domain.LandAcquisition{ID: acqID, GeometryWKT: acqWKT}, nil)
	landRepo.On("GetParcelByCode", ctx, parcelCode).
		Return(&domain.Parcel{ID: parcelID, ParcelID: parcelCode}, nil)
	parcelInfo.On("GetParcelInfo", ctx, parcelCode).Return(src, nil)
	landRepo.On("UpsertHolder", ctx, src).Return(holderID, nil)
	landRepo.On("ReplaceParcelDetail", ctx, parcelID, acqID, holderID, src).Return(nil)

	// Геометр ирсэн тул ИРСЭН геометрээр тооцно (хадгалагдсанаар биш)
	landRepo.On("ApplyParcelGeometry", ctx, parcelID, parcelWKT, acqWKT).
		Return(&domain.ParcelGeometryResult{
			ParcelAreaM2:      49235,
			AcquisitionAreaM2: 773,
			RemainingAreaM2:   48462,
			HasOverlap:        true,
		}, nil)
	landRepo.On("FillParcelAUCodesFromGeometry", ctx, parcelID).
		Return(domain.LandAcquisitionAU{AU1Code: "011", AU2Code: "01116", AU3Code: "0111659"}, nil)

	require.NoError(t, svc.SyncParcel(ctx, acqID, parcelCode))
	landRepo.AssertExpectations(t)
	// Хадгалагдсан геометрээр тооцох нөөц арга хэрэглэгдэхгүй
	landRepo.AssertNotCalled(t, "UpdateParcelAcquisitionGeom", mock.Anything, mock.Anything, mock.Anything)
}

// Геометр ирээгүй бол хадгалагдсан геометрээр давхцлыг шинэчилнэ (нөөц арга).
func TestLandService_SyncParcel_FallsBackWhenSourceGeometryMissing(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	parcelCode := "010101000001"
	svc, landRepo, parcelInfo, _ := syncHolderSvc(t, acqID, parcelID, parcelCode, "2024-0001")

	parcelInfo.On("ListApplicationPersons", ctx, "2024-0001").
		Return([]*domain.ParcelHolder{}, nil)
	landRepo.On("ReplaceParcelHolders", ctx, parcelID, acqID, "2024-0001", mock.Anything).Return(uuid.Nil, nil)
	landRepo.On("SetParcelDetailHolder", ctx, parcelID, uuid.Nil).Return(nil)

	require.NoError(t, svc.SyncParcel(ctx, acqID, parcelCode))
	landRepo.AssertCalled(t, "UpdateParcelAcquisitionGeom", ctx, parcelID, mock.AnythingOfType("string"))
	landRepo.AssertNotCalled(t, "ApplyParcelGeometry",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ── Нэгж талбарын чөлөөлөх хил/талбай шинэчлэх шалгалт ───────

// parcelMetaSvc — UpdateParcelMeta-ийн шалгалтыг тестлэх бэлтгэл. Нэгж
// талбарын үндсэн хил нь 100 x 100 нэгж (тестийн WKT).
func parcelMetaSvc(parcelID uuid.UUID) (service.LandService, *mocks.MockLandAcquisitionRepository) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc, _ := newLandSvc(landRepo)
	landRepo.On("GetParcelByID", ctx, parcelID).Return(&domain.Parcel{
		ID:          parcelID,
		ParcelID:    "010101000001",
		GeometryWKT: "POLYGON((106.76 47.66,106.77 47.66,106.77 47.67,106.76 47.67,106.76 47.66))",
		AreaM2:      1_000_000,
	}, nil)
	landRepo.On("UpdateParcelMeta", ctx, parcelID,
		mock.AnythingOfType("bool"), mock.AnythingOfType("string"),
		mock.Anything, mock.Anything).Return(nil).Maybe()
	return svc, landRepo
}

func strPtr(s string) *string   { return &s }
func f64Ptr(f float64) *float64 { return &f }

// Үндсэн хил дотор БҮРЭН байгаа хил хадгалагдана.
func TestLandService_UpdateParcelMeta_AcceptsGeomInsideParcel(t *testing.T) {
	parcelID := uuid.New()
	svc, landRepo := parcelMetaSvc(parcelID)
	newWKT := "POLYGON((106.762 47.662,106.766 47.662,106.766 47.666,106.762 47.666,106.762 47.662))"

	landRepo.On("CheckContainment", ctx, newWKT, mock.AnythingOfType("string")).
		Return(&domain.GeomContainment{Covered: true, ContainerM2: 1_000_000}, nil)

	err := svc.UpdateParcelMeta(ctx, parcelID, false, "", f64Ptr(150_000), strPtr(newWKT))

	require.NoError(t, err)
	landRepo.AssertCalled(t, "UpdateParcelMeta", ctx, parcelID, false, "",
		mock.Anything, mock.Anything)
}

// Үндсэн хилээс ГАРСАН хил хадгалагдахгүй.
func TestLandService_UpdateParcelMeta_RejectsGeomOutsideParcel(t *testing.T) {
	parcelID := uuid.New()
	svc, landRepo := parcelMetaSvc(parcelID)
	newWKT := "POLYGON((106.76 47.66,106.80 47.66,106.80 47.70,106.76 47.70,106.76 47.66))"

	landRepo.On("CheckContainment", ctx, newWKT, mock.AnythingOfType("string")).
		Return(&domain.GeomContainment{OutsideM2: 50_000, ContainerM2: 1_000_000}, nil)

	err := svc.UpdateParcelMeta(ctx, parcelID, false, "", nil, strPtr(newWKT))

	assert.Equal(t, domain.ErrOutsideParcelBoundary, err)
	landRepo.AssertNotCalled(t, "UpdateParcelMeta",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Үндсэн хилээс ХЭДХЭН м² гарсан ч хадгалагдахгүй — 100% дотор байх ёстой.
func TestLandService_UpdateParcelMeta_RejectsTinyOverrun(t *testing.T) {
	parcelID := uuid.New()
	svc, landRepo := parcelMetaSvc(parcelID)
	newWKT := "POLYGON((106.76 47.66,106.7701 47.66,106.7701 47.67,106.76 47.67,106.76 47.66))"

	landRepo.On("CheckContainment", ctx, newWKT, mock.AnythingOfType("string")).
		Return(&domain.GeomContainment{OutsideM2: 3, ContainerM2: 1_000_000}, nil)

	err := svc.UpdateParcelMeta(ctx, parcelID, false, "", nil, strPtr(newWKT))

	assert.Equal(t, domain.ErrOutsideParcelBoundary, err)
	landRepo.AssertNotCalled(t, "UpdateParcelMeta",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Үндсэн хилтэй ЯГ давхцсан (шугам дээр таарсан) хил хадгалагдана —
// ST_CoveredBy нь хилийн шугамыг дотор гэж тооцно.
func TestLandService_UpdateParcelMeta_AcceptsCoincidentBoundary(t *testing.T) {
	parcelID := uuid.New()
	svc, landRepo := parcelMetaSvc(parcelID)
	newWKT := "POLYGON((106.76 47.66,106.77 47.66,106.77 47.67,106.76 47.67,106.76 47.66))"

	landRepo.On("CheckContainment", ctx, newWKT, mock.AnythingOfType("string")).
		Return(&domain.GeomContainment{Covered: true, ContainerM2: 1_000_000}, nil)

	require.NoError(t, svc.UpdateParcelMeta(ctx, parcelID, false, "", nil, strPtr(newWKT)))
}

// Чөлөөлөгдөх талбай нь нэгж талбарын талбайгаас их байвал хадгалагдахгүй.
func TestLandService_UpdateParcelMeta_RejectsAreaOverParcelArea(t *testing.T) {
	parcelID := uuid.New()
	svc, landRepo := parcelMetaSvc(parcelID)

	err := svc.UpdateParcelMeta(ctx, parcelID, false, "", f64Ptr(1_500_000), nil)

	assert.Equal(t, domain.ErrOutsideParcelBoundary, err)
	landRepo.AssertNotCalled(t, "UpdateParcelMeta",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Хил/талбай хөндөөгүй үед шалгалт хийгдэхгүй (зөвхөн тэмдэглэгээ шинэчилнэ).
func TestLandService_UpdateParcelMeta_SkipsCheckWithoutGeomOrArea(t *testing.T) {
	parcelID := uuid.New()
	svc, landRepo := parcelMetaSvc(parcelID)

	require.NoError(t, svc.UpdateParcelMeta(ctx, parcelID, true, "010101000002", nil, nil))
	landRepo.AssertNotCalled(t, "CheckContainment",
		mock.Anything, mock.Anything, mock.Anything)
}
