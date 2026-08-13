package service_test

// Төлөвлөгөөний шинэ урсгал:
//   - Чөлөөлөлт үүсгэхэд төлөвлөгөө нь дундын сервисийн /plan/project-ээс авагдана
//   - Тэр мэдээлэл MAIN DB-ийн plan хүснэгтэд хадгалагдаж (Upsert),
//     land_acquisition.plan_id нь ТЭР мөрийг хэлнэ
//   - Төлөвлөгөө олдоогүй бол чөлөөлөлт үүсэхгүй (ErrNotFound)
//   - Төлөвлөгөөгөөр хайх/сонгох (Suggest) нь plan хүснэгтээс уншина

import (
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func middlewarePlan() *domain.Plan {
	// /plan/project нь хил (boundary) буцаадаггүй
	return &domain.Plan{
		ProjectID:    "PRJ-9",
		Code:         testPlanCode,
		PlanCode:     testPlanCode,
		Name:         "Дундын сервисийн төлөвлөгөө",
		PlanTypeName: "Хэсэгчилсэн ерөнхий төлөвлөгөө",
	}
}

func TestLandService_Create_StoresPlanFromMiddleware(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(planRepo, landRepo, nil, nil, projects, planStore)

	storedID := uuid.New()
	projects.On("GetPlanProject", ctx, testPlanCode).Return(middlewarePlan(), nil)
	// Дундын сервисийн мэдээлэл plan хүснэгтэд хадгалагдаж, ID-тай мөр буцна
	planStore.On("Upsert", ctx, mock.MatchedBy(func(p *domain.Plan) bool {
		return p.PlanCode == testPlanCode && p.ProjectID == "PRJ-9" && p.Name == "Дундын сервисийн төлөвлөгөө"
	})).Return(&domain.Plan{
		ID:       storedID,
		PlanCode: testPlanCode,
		Code:     testPlanCode,
		Name:     "Дундын сервисийн төлөвлөгөө",
	}, nil)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("CreateProgress", ctx, mock.Anything).Return(nil).Maybe()
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{}, nil)

	acq, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, testPlanCode, acq.PlanCode)
	require.NotNil(t, acq.PlanID, "plan_id нь plan хүснэгтийн мөрийг хэлэх ёстой")
	assert.Equal(t, storedID, *acq.PlanID)
	// Хил байхгүй тул давхцлын шалгалт дуудагдахгүй
	landRepo.AssertNotCalled(t, "ValidateOverlap", mock.Anything, mock.Anything, mock.Anything)
	// Дундын сервисээс олдсон тул auth DB-ийн plan уншигдахгүй
	planRepo.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	projects.AssertExpectations(t)
	planStore.AssertExpectations(t)
}

// Дундын сервисээс олдоогүй бол ХААНААС Ч хайхгүй (plan хүснэгт ч, auth DB ч)
func TestLandService_Create_NoFallbackWhenMiddlewareNotFound(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(planRepo, landRepo, nil, nil, projects, planStore)

	projects.On("GetPlanProject", ctx, "UNKNOWN").Return(nil, domain.ErrNotFound)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      "UNKNOWN",
		StartDate:     testStartDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrNotFound)
	planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	planRepo.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	planStore.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Холболтын алдаа нь "олдсонгүй" болж хувирахгүй — шалтгаан хэвээр дамжина
func TestLandService_Create_MiddlewareErrorPropagates(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(planRepo, landRepo, nil, nil, projects, planStore)

	projects.On("GetPlanProject", ctx, testPlanCode).Return(nil, domain.ErrInternalServer)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrInternalServer)
	planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	planRepo.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestLandService_Create_PlanCodeRequired(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := service.NewLandService(planRepo, landRepo, nil, nil, &mocks.MockPlanProjectClient{}, &mocks.MockPlanStoreRepository{})

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      "   ",
		StartDate:     testStartDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Хадгалагдсан төлөвлөгөөнд ХИЛ байсан ч давхцлын шалгалт хийхгүй (хасагдсан)
func TestLandService_Create_IgnoresPlanBoundary(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(planRepo, landRepo, nil, nil, projects, planStore)

	projects.On("GetPlanProject", ctx, testPlanCode).Return(middlewarePlan(), nil)
	// Хадгалагдсан мөрөнд хол байрлах хил бий — гэхдээ шалгалтад хэрэглэгдэхгүй
	planStore.On("Upsert", ctx, mock.Anything).Return(&domain.Plan{
		ID: uuid.New(), PlanCode: testPlanCode, BoundaryWKT: "POLYGON((10 10,11 10,11 11,10 11,10 10))",
	}, nil)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{}, nil)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:      testPlanCode,
		StartDate:     testStartDate,
		ShapefileData: buildSimpleShp(),
		CreatedBy:     "user-1",
	})

	require.NoError(t, err)
	landRepo.AssertNotCalled(t, "ValidateOverlap", mock.Anything, mock.Anything, mock.Anything)
}

func TestPlanService_Search_UsesMiddlewareAndRequiresCode(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewPlanService(planRepo, projects, planStore)

	// Код хоосон бол дуудалт ХИЙХГҮЙ
	_, err := svc.Search(ctx, "  ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	projects.AssertNotCalled(t, "GetPlanProject", mock.Anything, mock.Anything)

	projects.On("GetPlanProject", ctx, testPlanCode).Return(middlewarePlan(), nil)
	plan, err := svc.Search(ctx, testPlanCode)
	require.NoError(t, err)
	assert.Equal(t, "PRJ-9", plan.ProjectID)
	planRepo.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
}

// Хайлт нь ЗӨВХӨН дундын сервисээс — олдоогүй/алдаа гарсан үед хадгалагдсан
// plan хүснэгт ч, auth DB ч уншигдахгүй.
func TestPlanService_Search_NoFallback(t *testing.T) {
	for name, middlewareErr := range map[string]error{
		"олдсонгүй":       domain.ErrNotFound,
		"холболтын алдаа": domain.ErrInternalServer,
	} {
		t.Run(name, func(t *testing.T) {
			planRepo := &mocks.MockPlanRepository{}
			planStore := &mocks.MockPlanStoreRepository{}
			projects := &mocks.MockPlanProjectClient{}
			svc := service.NewPlanService(planRepo, projects, planStore)

			projects.On("GetPlanProject", ctx, testPlanCode).Return(nil, middlewareErr)

			_, err := svc.Search(ctx, testPlanCode)
			assert.ErrorIs(t, err, middlewareErr, "алдаа хувирахгүй дамжина")
			planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
			planRepo.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
		})
	}
}

func TestPlanService_Suggest_ReadsFromPlanTable(t *testing.T) {
	planRepo := &mocks.MockPlanRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	svc := service.NewPlanService(planRepo, planStore)

	planStore.On("Suggest", ctx, "PLAN").Return([]*domain.Plan{
		{ID: uuid.New(), PlanCode: testPlanCode, Name: "Хадгалагдсан"},
	}, nil)

	plans, err := svc.Suggest(ctx, "PLAN")
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, testPlanCode, plans[0].PlanCode)
	// auth DB-ийн төсөөллийн plan хүснэгт ашиглагдахгүй
	planRepo.AssertNotCalled(t, "Suggest", mock.Anything, mock.Anything)
}
