package service_test

// Төлөвлөгөөний шинэ урсгал:
//   - Чөлөөлөлт үүсгэхэд төлөвлөгөө нь дундын сервисийн /plan/project-ээс авагдана
//   - Тэр мэдээлэл MAIN DB-ийн plan хүснэгтэд хадгалагдаж (Upsert),
//     land_acquisition.plan_id нь ТЭР мөрийг хэлнэ
//   - Төлөвлөгөө олдоогүй бол чөлөөлөлт үүсэхгүй (ErrNotFound)
//   - Төлөвлөгөөгөөр хайх/сонгох нь appdb-ийн plan хүснэгтээс уншина

import (
	"errors"
	"strings"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// storedPlanBoundary — plan хүснэгтэд хадгалагдсан мөрийн хил. Чөлөөлөлтийн
// хил нь ҮҮНЭЭС хуулагдана (дундын сервисийн түүхий хариунаас БИШ).
const storedPlanBoundary = "POLYGON((106.9274 47.9846,106.9259 47.9831,106.9280 47.9820,106.9274 47.9846))"

func middlewarePlan() *domain.Plan {
	return &domain.Plan{
		ProjectID:    "PRJ-9",
		Code:         testPlanCode,
		PlanCode:     testPlanCode,
		Name:         "Дундын сервисийн төлөвлөгөө",
		PlanTypeName: "Хэсэгчилсэн ерөнхий төлөвлөгөө",
		ParcelID:     "5029776",
		Gazner:       "Дамбадаржаа дэд төвийн бүтээн байгуулалтын ажил",
		// /plan/project нь хилийг буцаадаг болсон
		BoundaryWKT: "POLYGON((106.9274 47.9846,106.9259 47.9831,106.9280 47.9820,106.9274 47.9846))",
	}
}

func TestLandService_Create_StoresPlanFromMiddleware(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(landRepo, nil, nil, projects, planStore)

	storedID := uuid.New()
	projects.On("GetPlanProject", ctx, testPlanCode).Return(middlewarePlan(), nil)
	// Дундын сервисийн мэдээлэл plan хүснэгтэд хадгалагдаж, ID-тай мөр буцна
	planStore.On("Upsert", ctx, mock.MatchedBy(func(p *domain.Plan) bool {
		// Дундын сервисийн БҮХ талбар (хил, нэгж талбарын дугаар, ажлын нэр
		// зэрэг шинэ талбарууд орсон эсэх) хадгалахаар дамжина
		return p.PlanCode == testPlanCode && p.ProjectID == "PRJ-9" &&
			p.Name == "Дундын сервисийн төлөвлөгөө" &&
			p.ParcelID == "5029776" &&
			p.Gazner == "Дамбадаржаа дэд төвийн бүтээн байгуулалтын ажил" &&
			strings.HasPrefix(p.BoundaryWKT, "POLYGON((106.9274")
	})).Return(&domain.Plan{
		ID:          storedID,
		PlanCode:    testPlanCode,
		Code:        testPlanCode,
		Name:        "Дундын сервисийн төлөвлөгөө",
		BoundaryWKT: storedPlanBoundary,
	}, nil)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(41_250_000.0, nil)
	expectBoundaryOK(landRepo)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("CreateProgress", ctx, mock.Anything).Return(nil).Maybe()
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).Return([]domain.LandAcquisitionAU{}, nil)

	acq, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:  testPlanCode,
		StartDate: testStartDate,
		CreatedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, testPlanCode, acq.PlanCode)
	require.NotNil(t, acq.PlanID, "plan_id нь plan хүснэгтийн мөрийг хэлэх ёстой")
	assert.Equal(t, storedID, *acq.PlanID)
	// Чөлөөлөлтийн хил нь plan хүснэгтэд ХАДГАЛАГДСАН мөрийн хилээс хуулагдана.
	assert.Equal(t, storedPlanBoundary, acq.GeometryWKT)
	projects.AssertExpectations(t)
	planStore.AssertExpectations(t)
}

// Дундын сервисээс олдоогүй бол ХААНААС Ч хайхгүй — plan хүснэгт нь зөвхөн
// дундын сервисийн датаг агуулдаг тул түүнээс "нөхөж" уншихгүй.
func TestLandService_Create_NoFallbackWhenMiddlewareNotFound(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(landRepo, nil, nil, projects, planStore)

	projects.On("GetPlanProject", ctx, "UNKNOWN").Return(nil, domain.ErrNotFound)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:  "UNKNOWN",
		StartDate: testStartDate,
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrNotFound)
	planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	planStore.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Холболтын алдаа нь "олдсонгүй" болж хувирахгүй — шалтгаан хэвээр дамжина
func TestLandService_Create_MiddlewareErrorPropagates(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(landRepo, nil, nil, projects, planStore)

	projects.On("GetPlanProject", ctx, testPlanCode).Return(nil, domain.ErrInternalServer)

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:  testPlanCode,
		StartDate: testStartDate,
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrInternalServer)
	planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestLandService_Create_PlanCodeRequired(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	svc := service.NewLandService(landRepo, nil, nil, &mocks.MockPlanProjectClient{}, &mocks.MockPlanStoreRepository{})

	_, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:  "   ",
		StartDate: testStartDate,
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	landRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Чөлөөлөлтийн хил нь plan хүснэгтэд ХАДГАЛАГДСАН мөрийн хилээс хуулагдана —
// дундын сервисийн түүхий хариунаас БИШ. (Upsert нь хоосон утгаар хуучныг дарж
// бичдэггүй тул хоёр нь зөрж болно.)
func TestLandService_Create_CopiesStoredPlanBoundary(t *testing.T) {
	landRepo := &mocks.MockLandAcquisitionRepository{}
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewLandService(landRepo, nil, nil, projects, planStore)

	// Хадгалагдсан мөрийн хил нь дундын сервисийн буцаасан хилээс ӨӨР.
	otherBoundary := "POLYGON((106.70 47.60,106.75 47.60,106.75 47.65,106.70 47.65,106.70 47.60))"
	projects.On("GetPlanProject", ctx, testPlanCode).Return(middlewarePlan(), nil)
	planStore.On("Upsert", ctx, mock.Anything).Return(&domain.Plan{
		ID: uuid.New(), PlanCode: testPlanCode, BoundaryWKT: otherBoundary,
	}, nil)
	// Талбай, огтлолцол ХАДГАЛАГДСАН хилээр тооцогдоно (аргумент нь яг тэр WKT).
	landRepo.On("ComputeAreaM2", ctx, otherBoundary).Return(41_250_000.0, nil)
	expectBoundaryOK(landRepo)
	landRepo.On("Create", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil)
	landRepo.On("CreateProgress", ctx, mock.Anything).Return(nil).Maybe()
	landRepo.On("FindIntersectingAU3", ctx, otherBoundary).Return([]domain.LandAcquisitionAU{}, nil)

	acq, err := svc.Create(ctx, service.CreateLandAcquisitionRequest{
		PlanCode:  testPlanCode,
		StartDate: testStartDate,
		CreatedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, otherBoundary, acq.GeometryWKT)
	assert.Equal(t, otherBoundary, acq.PlanGeomWKT)
	assert.Equal(t, 41_250_000.0, acq.AreaM2)
	landRepo.AssertExpectations(t)
}

func TestPlanService_Search_StoresMiddlewareDataThenReadsAppDB(t *testing.T) {
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewPlanService(planStore, projects)

	// Код хоосон бол дуудалт ХИЙХГҮЙ
	_, err := svc.Search(ctx, "  ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	projects.AssertNotCalled(t, "GetPlanProject", mock.Anything, mock.Anything)

	// Олдмогц MAIN DB-д хадгалж, хариуг appdb-ийн plan хүснэгтээс уншина.
	found := middlewarePlan()
	stored := *found
	stored.ID = uuid.New()
	projects.On("GetPlanProject", ctx, testPlanCode).Return(found, nil)
	planStore.On("Upsert", ctx, found).Return(&stored, nil)
	planStore.On("GetByCode", ctx, testPlanCode).Return(&stored, nil)

	plan, err := svc.Search(ctx, testPlanCode)
	require.NoError(t, err)
	assert.Equal(t, "PRJ-9", plan.ProjectID)
	assert.Equal(t, stored.ID, plan.ID)
	planStore.AssertExpectations(t)
}

// Хадгалалт унавал raw middleware дата буцаахгүй. Уншилт зөвхөн appdb-ийн
// plan хүснэгтээс явах ёстой тул хайлт алдаагаар дуусна.
func TestPlanService_Search_StoreFailureFailsSearch(t *testing.T) {
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewPlanService(planStore, projects)

	found := middlewarePlan()
	projects.On("GetPlanProject", ctx, testPlanCode).Return(found, nil)
	planStore.On("Upsert", ctx, found).Return(nil, errors.New("db down"))

	plan, err := svc.Search(ctx, testPlanCode)

	require.ErrorIs(t, err, domain.ErrInternalServer)
	assert.Nil(t, plan)
	planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
}

// Дундын сервис нэгж талбарын дугаар буцаагаагүй бол ХАЙСАН дугаараар бүртгэнэ.
func TestPlanService_Search_FillsParcelIDFromQuery(t *testing.T) {
	planStore := &mocks.MockPlanStoreRepository{}
	projects := &mocks.MockPlanProjectClient{}
	svc := service.NewPlanService(planStore, projects)

	found := middlewarePlan()
	found.ParcelID = ""
	stored := *found
	stored.ParcelID = testPlanCode
	projects.On("GetPlanProject", ctx, testPlanCode).Return(found, nil)
	planStore.On("Upsert", ctx, mock.MatchedBy(func(p *domain.Plan) bool {
		return p.ParcelID == testPlanCode
	})).Return(&stored, nil)
	planStore.On("GetByCode", ctx, testPlanCode).Return(&stored, nil)

	_, err := svc.Search(ctx, testPlanCode)

	require.NoError(t, err)
	planStore.AssertExpectations(t)
}

// Хайлт нь ЗӨВХӨН дундын сервисээс — олдоогүй/алдаа гарсан үед хадгалагдсан
// plan хүснэгт ч, auth DB ч уншигдахгүй.
func TestPlanService_Search_NoFallback(t *testing.T) {
	for name, middlewareErr := range map[string]error{
		"олдсонгүй":       domain.ErrNotFound,
		"холболтын алдаа": domain.ErrInternalServer,
	} {
		t.Run(name, func(t *testing.T) {
			planStore := &mocks.MockPlanStoreRepository{}
			projects := &mocks.MockPlanProjectClient{}
			svc := service.NewPlanService(planStore, projects)

			projects.On("GetPlanProject", ctx, testPlanCode).Return(nil, middlewareErr)

			_, err := svc.Search(ctx, testPlanCode)
			assert.ErrorIs(t, err, middlewareErr, "алдаа хувирахгүй дамжина")
			planStore.AssertNotCalled(t, "GetByCode", mock.Anything, mock.Anything)
		})
	}
}

func TestPlanService_Suggest_ReadsFromPlanTable(t *testing.T) {
	planStore := &mocks.MockPlanStoreRepository{}
	svc := service.NewPlanService(planStore)

	planStore.On("Suggest", ctx, "PLAN").Return([]*domain.Plan{
		{ID: uuid.New(), PlanCode: testPlanCode, Name: "Хадгалагдсан"},
	}, nil)

	plans, err := svc.Suggest(ctx, "PLAN")
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, testPlanCode, plans[0].PlanCode)
	// plan сонголт зөвхөн appdb-ийн plan хүснэгтээс уншина
}
