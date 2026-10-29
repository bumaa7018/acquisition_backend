package service_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/service"
	"acquisition_backend/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Гараас (shapefile) хил ЗАСВАРЛАХ урсгалын тестүүд.
//
// Юуг батлах вэ:
//   - Файлаас уншсан хил нь чөлөөлөлтийн шинэ хил болно.
//   - Талбайн зөрүү 30%-иас бага үед л хүлээж авна (яг 30% ч татгалзана).
//   - Зөрүү нь ТӨЛӨВЛӨГӨӨНИЙ хилээс хэмжигдэнэ (жишиг нүүхгүй).
//   - Төлөвлөгөөний мэдээлэл хөндөгдөхгүй, дундын сервис ч дуудагдахгүй.
//   - Хуучин шалгалт (нэг сум/дүүрэг) энэ зам дээр мөн ажиллана.
//   - Шинэ хилээс гарсан нэгж талбар УСТГАГДАХГҮЙ, зөвхөн анхааруулна.

// planReferenceWKT — жишиг (төлөвлөгөөний) хил. Талбайн зөрүү үүнээс хэмжигдэнэ.
const planReferenceWKT = "POLYGON((106.70 47.60,106.90 47.60,106.90 47.80,106.70 47.80,106.70 47.60))"

// buildBoundaryShp — өгсөн цэгүүдээс хүчинтэй raw .shp файл (Polygon) угсарна.
// Хилийн ФАЙЛААР дамжих замыг бүрэн шалгахын тулд жинхэнэ байт бүтэц хэрэгтэй
// (service нь shapefile.Parse-аар уншина).
func buildBoundaryShp(pts [][2]float64) []byte {
	numParts, numPoints := 1, len(pts)
	contentLen := 4 + 32 + 4 + 4 + numParts*4 + numPoints*16
	buf := make([]byte, 100+8+contentLen)
	binary.BigEndian.PutUint32(buf[0:], 9994)
	binary.BigEndian.PutUint32(buf[24:], uint32((100+8+contentLen)/2))
	binary.LittleEndian.PutUint32(buf[28:], 1000)
	binary.LittleEndian.PutUint32(buf[32:], 5)
	binary.BigEndian.PutUint32(buf[100:], 1)
	binary.BigEndian.PutUint32(buf[104:], uint32(contentLen/2))
	off := 108
	binary.LittleEndian.PutUint32(buf[off:], 5)
	off += 4 + 32
	binary.LittleEndian.PutUint32(buf[off:], uint32(numParts))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], uint32(numPoints))
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], 0)
	off += 4
	for _, p := range pts {
		binary.LittleEndian.PutUint64(buf[off:], math.Float64bits(p[0]))
		off += 8
		binary.LittleEndian.PutUint64(buf[off:], math.Float64bits(p[1]))
		off += 8
	}
	return buf
}

// testBoundaryShp — хүчинтэй хилийн файл (Улаанбаатар орчим, нэг полигон).
func testBoundaryShp() []byte {
	return buildBoundaryShp([][2]float64{
		{106.72, 47.62}, {106.88, 47.62},
		{106.88, 47.78}, {106.72, 47.78}, {106.72, 47.62},
	})
}

// shapefileUpdateSvc — гараас хил засах урсгалын бэлтгэл.
//
// newAreaM2 — файлаас уншсан хилийн талбай, planAreaM2 — жишиг (төлөвлөгөөний)
// хилийн талбай. Хоёул ИЖИЛ функцээр (ComputeAreaM2 = public.calculate_area_utm)
// бодогддогийг дуурайхын тулд WKT-ээр нь ялгаж хариулна.
func shapefileUpdateSvc(t *testing.T, id uuid.UUID, planAreaM2, newAreaM2 float64) (
	service.LandService,
	*mocks.MockLandAcquisitionRepository,
	*mocks.MockParcelInfoClient,
	planMocks,
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
		ID:          id,
		PlanCode:    testPlanCode,
		PlanName:    "Test Plan",
		PlanGeomWKT: planReferenceWKT,
		GeometryWKT: planReferenceWKT,
		Status:      domain.LandAcquisitionStatusField,
	}
	landRepo.On("GetByID", ctx, id).Return(existing, nil)
	// Жишиг хил — төлөвлөгөөнийх.
	landRepo.On("ComputeAreaM2", ctx, planReferenceWKT).Return(planAreaM2, nil)
	// Файлаас уншсан бусад бүх хил.
	landRepo.On("ComputeAreaM2", ctx, mock.MatchedBy(func(wkt string) bool {
		return wkt != planReferenceWKT
	})).Return(newAreaM2, nil)
	expectBoundaryOK(landRepo)
	landRepo.On("Update", ctx, mock.AnythingOfType("*domain.LandAcquisition")).Return(nil).Maybe()
	landRepo.On("CreateBoundaryHistory", ctx, mock.Anything).Return(nil).Maybe()
	landRepo.On("DeleteAUs", ctx, id).Return(nil).Maybe()
	landRepo.On("FindIntersectingAU3", ctx, mock.AnythingOfType("string")).
		Return([]domain.LandAcquisitionAU{}, nil).Maybe()
	landRepo.On("RecomputeAcquisitionAreas", ctx, id, mock.AnythingOfType("string")).
		Return(0, 0, nil).Maybe()
	return svc, landRepo, parcelInfo, pm
}

// expectNoParcelChange — хил солиход нэгж талбарт юу ч хийгдэхгүй тул
// хүлээлт ТАВИХГҮЙ: ГУС эсвэл нэгж талбарын repo дуудагдвал mock panic хийнэ.
func expectNoParcelChange(*mocks.MockLandAcquisitionRepository, *mocks.MockParcelInfoClient, uuid.UUID) {
}

// Талбайн зөрүү хязгаарт багтсан үед хил ФАЙЛААС солигдоно.
func TestLandService_Update_AcceptsShapefileWithinDeviation(t *testing.T) {
	id := uuid.New()
	// Жишиг 1,000,000 м², шинэ хил 1,100,000 м² → 10% зөрүү.
	svc, landRepo, parcelInfo, pm := shapefileUpdateSvc(t, id, 1_000_000, 1_100_000)
	expectNoParcelChange(landRepo, parcelInfo, id)

	acq, change, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		ShapefileData:   testBoundaryShp(),
		UpdatedBy:       "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, change, "хил солигдсоныг хариултад тэмдэглэнэ")
	assert.Contains(t, acq.GeometryWKT, "POLYGON((106.720000 47.620000",
		"хил файлаас уншигдах ёстой")
	assert.Equal(t, 1_100_000.0, acq.AreaM2, "талбай шинэ хилээс автоматаар бодогдоно")
	// Хилийн өөрчлөлт ТҮҮХЭД бичигдэнэ (байршил таб дээр харагдана).
	landRepo.AssertCalled(t, "CreateBoundaryHistory", ctx, mock.MatchedBy(
		func(h *domain.LandAcquisitionBoundaryHistory) bool {
			return h.LandAcquisitionID == id &&
				h.OldGeometryWKT == planReferenceWKT &&
				h.NewGeometryWKT != "" &&
				h.NewGeometryWKT != planReferenceWKT
		}))
	// Гараас хил зассан үед дундын сервис рүү төлөвлөгөө хайхгүй.
	pm.projects.AssertNotCalled(t, "GetPlanProject", mock.Anything, mock.Anything)
}

// Гараас хил зассан ч ТӨЛӨВЛӨГӨӨНИЙ мэдээлэл хөндөгдөхгүй — дараагийн засварын
// жишиг (талбайн зөрүү) төлөвлөгөөний хил хэвээр байх ёстой.
func TestLandService_Update_ShapefileKeepsPlanReference(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo, _ := shapefileUpdateSvc(t, id, 1_000_000, 900_000)
	expectNoParcelChange(landRepo, parcelInfo, id)

	acq, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		ShapefileData:   testBoundaryShp(),
	})

	require.NoError(t, err)
	assert.Equal(t, testPlanCode, acq.PlanCode)
	assert.Equal(t, planReferenceWKT, acq.PlanGeomWKT)
	assert.NotEqual(t, planReferenceWKT, acq.GeometryWKT)
}

// Зөрүү 30%-аас БАГА байх ёстой: яг 30% ч татгалзана.
func TestLandService_Update_RejectsShapefileAtDeviationLimit(t *testing.T) {
	for name, newArea := range map[string]float64{
		"яг 30%":   1_300_000,
		"40% их":   1_400_000,
		"50% бага": 500_000,
	} {
		t.Run(name, func(t *testing.T) {
			id := uuid.New()
			svc, landRepo, parcelInfo, _ := shapefileUpdateSvc(t, id, 1_000_000, newArea)
			expectNoParcelChange(landRepo, parcelInfo, id)

			_, change, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
				AcquisitionName: "Тест",
				ShapefileData:   testBoundaryShp(),
			})

			assert.ErrorIs(t, err, domain.ErrBoundaryAreaDeviation)
			assert.Nil(t, change)
			// Хил, түүх, нэгж талбар — юу ч хөндөгдөхгүй.
			landRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			landRepo.AssertNotCalled(t, "CreateBoundaryHistory", mock.Anything, mock.Anything)
			landRepo.AssertNotCalled(t, "InsertParcelCodes", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			landRepo.AssertNotCalled(t, "PurgeParcels", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// Зөрүү хязгаарын ЯГ доор (29.9%) байвал хүлээж авна.
func TestLandService_Update_AcceptsShapefileJustUnderLimit(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo, _ := shapefileUpdateSvc(t, id, 1_000_000, 1_299_000)
	expectNoParcelChange(landRepo, parcelInfo, id)

	_, change, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		ShapefileData:   testBoundaryShp(),
	})

	require.NoError(t, err)
	assert.NotNil(t, change)
}

// Хоёр замыг (төлөвлөгөөний дугаар + файл) зэрэг өгвөл аль хил хүчинтэй нь
// тодорхойгүй тул татгалзана.
func TestLandService_Update_RejectsPlanAndShapefileTogether(t *testing.T) {
	id := uuid.New()
	svc, landRepo, _, pm := shapefileUpdateSvc(t, id, 1_000_000, 1_000_000)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		PlanCode:        testPlanCode,
		ShapefileData:   testBoundaryShp(),
	})

	assert.ErrorIs(t, err, domain.ErrBoundarySourceConflict)
	landRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	pm.projects.AssertNotCalled(t, "GetPlanProject", mock.Anything, mock.Anything)
}

// Стандартад тохирохгүй файл — ЯМАР шалтгаанаар татгалзсаныг ялгаж буцаана.
func TestLandService_Update_RejectsInvalidShapefile(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"shapefile биш": {
			data: []byte("not-a-shapefile-at-all-just-some-bytes-here-x"),
			want: domain.ErrShapefileFormat,
		},
		"Монголоос гадна": {
			data: buildBoundaryShp([][2]float64{
				{2.30, 48.85}, {2.40, 48.85}, {2.40, 48.90}, {2.30, 48.90}, {2.30, 48.85},
			}),
			want: domain.ErrShapefileOutsideCountry,
		},
		"метр проекц": {
			data: buildBoundaryShp([][2]float64{
				{478000, 5279000}, {479000, 5279000},
				{479000, 5280000}, {478000, 5280000}, {478000, 5279000},
			}),
			want: domain.ErrShapefileProjection,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			id := uuid.New()
			svc, landRepo, _, _ := shapefileUpdateSvc(t, id, 1_000_000, 1_000_000)

			_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
				AcquisitionName: "Тест",
				ShapefileData:   tc.data,
			})

			assert.ErrorIs(t, err, tc.want)
			landRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			landRepo.AssertNotCalled(t, "CreateBoundaryHistory", mock.Anything, mock.Anything)
		})
	}
}

// ХУУЧИН шалгалт: сум/дүүрэг дамнасан хилийг файлаар ч оруулж болохгүй.
func TestLandService_Update_ShapefileRejectsAU2Crossing(t *testing.T) {
	id := uuid.New()
	landRepo := &mocks.MockLandAcquisitionRepository{}
	pm := planMocks{
		projects: &mocks.MockPlanProjectClient{},
		store:    &mocks.MockPlanStoreRepository{},
	}
	svc := service.NewLandService(landRepo, nil, nil, pm.projects, pm.store)

	landRepo.On("GetByID", ctx, id).Return(&domain.LandAcquisition{
		ID: id, PlanCode: testPlanCode,
		PlanGeomWKT: planReferenceWKT,
		GeometryWKT: planReferenceWKT,
		Status:      domain.LandAcquisitionStatusField,
	}, nil)
	landRepo.On("ComputeAreaM2", ctx, mock.AnythingOfType("string")).Return(1_000_000.0, nil)
	expectAU2(landRepo, 340,
		domain.AU2Overlap{Code: "0101", AreaM2: 500_000},
		domain.AU2Overlap{Code: "0102", AreaM2: 500_000},
	)

	_, _, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		ShapefileData:   testBoundaryShp(),
	})

	assert.ErrorIs(t, err, domain.ErrAU2Crossing)
	landRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	landRepo.AssertNotCalled(t, "CreateBoundaryHistory", mock.Anything, mock.Anything)
}

// Файлаар хил солиход ГУС дуудагдахгүй (төлөвлөгөө ч хайгдахгүй); нэгж
// талбарын нөлөөлөлд өртсөн талбай ФАЙЛЫН хилээр дахин бодогдоно, өөр юу ч
// хийгдэхгүй.
func TestLandService_Update_ShapefileRecomputesAffectedAreasOnly(t *testing.T) {
	id := uuid.New()
	svc, landRepo, parcelInfo, pm := shapefileUpdateSvc(t, id, 1_000_000, 800_000)

	acq, change, err := svc.Update(ctx, id, service.UpdateLandAcquisitionRequest{
		AcquisitionName: "Тест",
		ShapefileData:   testBoundaryShp(),
	})

	require.NoError(t, err)
	require.NotNil(t, change)
	assert.NotEqual(t, planReferenceWKT, acq.GeometryWKT, "хил шинэчлэгдээгүй")
	landRepo.AssertCalled(t, "CreateBoundaryHistory", ctx, mock.Anything)
	landRepo.AssertCalled(t, "RecomputeAcquisitionAreas", ctx, id, acq.GeometryWKT)
	pm.projects.AssertNotCalled(t, "GetPlanProject", mock.Anything, mock.Anything)
	assertNoParcelSideEffects(t, landRepo, parcelInfo)
}

// Урьдчилан харах нь ХАДГАЛАХГҮЙ: зөрүүг тоогоор буцаана, хязгаараас хэтэрсэн
// ч алдаа биш (accepted=false) — хэрэглэгч хэр зөрснийг харах ёстой.
func TestLandService_PreviewBoundaryShapefile(t *testing.T) {
	t.Run("зөрүү хязгаарт багтсан", func(t *testing.T) {
		id := uuid.New()
		svc, landRepo, _, _ := shapefileUpdateSvc(t, id, 1_000_000, 1_100_000)
		outside := []domain.BoundaryOutsideParcel{{
			ParcelID:    "010101000009",
			StatusName:  "Чөлөөлсөн",
			Released:    true,
			GeometryWKT: "POLYGON((106.95 47.60,106.96 47.60,106.96 47.61,106.95 47.60))",
		}}
		// Хилээс гарах талбарыг ФАЙЛЫН (шинэ) хилээр тодорхойлно — хуучин
		// хилээр биш.
		landRepo.On("ListParcelsOutsideBoundary", ctx, id, mock.MatchedBy(func(wkt string) bool {
			return wkt != planReferenceWKT && wkt != ""
		})).Return(outside, nil)

		check, err := svc.PreviewBoundaryShapefile(ctx, id, testBoundaryShp())

		require.NoError(t, err)
		assert.True(t, check.Accepted)
		assert.Equal(t, 1_100_000.0, check.AreaM2)
		assert.Equal(t, 1_000_000.0, check.ReferenceAreaM2)
		assert.InDelta(t, 0.10, check.DeviationRatio, 1e-9)
		assert.InDelta(t, 0.30, check.MaxDeviationRatio, 1e-9)
		assert.NotEmpty(t, check.GeometryWKT)
		assert.Equal(t, outside, check.OutsideParcels)
		landRepo.AssertCalled(t, "ListParcelsOutsideBoundary", ctx, id, check.GeometryWKT)
		// Урьдчилан харах нь юу ч хадгалахгүй, устгахгүй.
		landRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		landRepo.AssertNotCalled(t, "CreateBoundaryHistory", mock.Anything, mock.Anything)
		landRepo.AssertNotCalled(t, "PurgeParcels", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("хилээс гарах талбар тодорхойлж чадахгүй бол алдаа", func(t *testing.T) {
		id := uuid.New()
		svc, landRepo, _, _ := shapefileUpdateSvc(t, id, 1_000_000, 1_100_000)
		landRepo.On("ListParcelsOutsideBoundary", ctx, id, mock.Anything).
			Return(nil, errors.New("db down"))

		_, err := svc.PreviewBoundaryShapefile(ctx, id, testBoundaryShp())

		assert.ErrorIs(t, err, domain.ErrInternalServer)
	})

	t.Run("зөрүү хэтэрсэн ч алдаа биш", func(t *testing.T) {
		id := uuid.New()
		svc, landRepo, _, _ := shapefileUpdateSvc(t, id, 1_000_000, 1_500_000)
		landRepo.On("ListParcelsOutsideBoundary", ctx, id, mock.Anything).
			Return([]domain.BoundaryOutsideParcel{}, nil)

		check, err := svc.PreviewBoundaryShapefile(ctx, id, testBoundaryShp())

		require.NoError(t, err)
		assert.False(t, check.Accepted)
		assert.InDelta(t, 0.50, check.DeviationRatio, 1e-9)
	})

	t.Run("буруу файл", func(t *testing.T) {
		id := uuid.New()
		svc, _, _, _ := shapefileUpdateSvc(t, id, 1_000_000, 1_000_000)

		_, err := svc.PreviewBoundaryShapefile(ctx, id, []byte("garbage-bytes-not-a-shapefile-at-all"))

		assert.ErrorIs(t, err, domain.ErrShapefileFormat)
	})
}
