package http_test

// Гадаад системийн нөхөх олговрын API — ГУРВАН хэлбэрийн шүүлтүүр зөв
// дамжиж, хариултад төлөвийн тоолол ба нийт дүн орж байгааг шалгана.
//
// Танилтын middleware-г ЭНД оролцуулахгүй (тэр нь middleware-ийн тестээр
// хамгаалагдсан) — зөвхөн handler-ийн шүүлтүүр/хариултын гэрээ.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type externalAPIResponse struct {
	Data struct {
		Mode   string `json:"mode"`
		Filter struct {
			ParcelID     string `json:"parcel_id"`
			PlanParcelID string `json:"plan_parcel_id"`
			PlanCode     string `json:"plan_code"`
		} `json:"filter"`
		Summary struct {
			AcquisitionCount  int     `json:"acquisition_count"`
			ParcelCount       int     `json:"parcel_count"`
			AcquisitionAreaM2 float64 `json:"acquisition_area_m2"`
			Compensation      struct {
				Land       float64 `json:"land"`
				RealEstate float64 `json:"real_estate"`
				Property   float64 `json:"property"`
				Total      float64 `json:"total"`
				Currency   string  `json:"currency"`
			} `json:"compensation"`
			StatusCounts []struct {
				Status     int    `json:"status"`
				StatusName string `json:"status_name"`
				Count      int    `json:"count"`
			} `json:"status_counts"`
		} `json:"summary"`
		Parcels []struct {
			ParcelID          string  `json:"parcel_id"`
			AcquisitionAreaM2 float64 `json:"acquisition_area_m2"`
			RightTypeName     string  `json:"right_type_name"`
			Compensation      struct {
				Land       float64 `json:"land"`
				RealEstate float64 `json:"real_estate"`
				Property   float64 `json:"property"`
				Total      float64 `json:"total"`
			} `json:"compensation"`
		} `json:"parcels"`
		Total    int64 `json:"total"`
		PageSize int   `json:"page_size"`
	} `json:"data"`
}

func externalAPIFixture() ([]*domain.ReportParcelRow, *domain.ReportSummary) {
	rows := []*domain.ReportParcelRow{{
		ParcelID:          "1600900137",
		AreaM2:            5000,
		AcquisitionAreaM2: 4200,
		RightType:         domain.RightTypeLease,
		Status:            3,
		StatusName:        "Үнэлгээ хийх",
		AcquisitionID:     uuid.New(),
		AcquisitionName:   "Тест чөлөөлөлт",
		PlanCode:          "2026-04-011-9878",
		LandComp:          84_000_000,
		RealStateComp:     136_628_143.92,
		PropertyComp:      77_211_810,
	}}
	summary := &domain.ReportSummary{
		AcquisitionCount:      1,
		ParcelCount:           2,
		TotalAreaM2:           9200,
		TotalParcelAreaM2:     11000,
		TotalCompensation:     297_839_953.92,
		LandCompensation:      84_000_000,
		RealStateCompensation: 136_628_143.92,
		PropertyCompensation:  77_211_810,
		StatusBreakdown: []domain.ReportStatusStat{
			{Status: 3, StatusName: "Үнэлгээ хийх", Count: 1},
			{Status: 5, StatusName: "Чөлөөлсөн", Count: 1},
		},
	}
	return rows, summary
}

// externalAPIRouter — mock service-тэй handler. Дамжсан шүүлтүүрийг барьж авна.
func externalAPIRouter(t *testing.T) (*gin.Engine, *domain.ReportParcelFilter) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	rows, summary := externalAPIFixture()
	seen := &domain.ReportParcelFilter{}
	svc := new(mocks.MockLandService)
	svc.On("ListReportParcels", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { *seen = args.Get(1).(domain.ReportParcelFilter) }).
		Return(rows, int64(2), nil)
	svc.On("GetReportSummary", mock.Anything, mock.Anything).Return(summary, nil)

	r := gin.New()
	h := handler.NewExternalCompensationHandler(svc)
	r.GET("/external/compensations", h.ListCompensations)
	return r, seen
}

func callExternalAPI(t *testing.T, r *gin.Engine, query string) (*httptest.ResponseRecorder, externalAPIResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/external/compensations"+query, nil))
	var body externalAPIResponse
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}
	return w, body
}

func TestExternalAPI_RequiresAFilter(t *testing.T) {
	r, _ := externalAPIRouter(t)
	w, _ := callExternalAPI(t, r, "")
	// Шүүлтүүргүйгээр бүх сангаа гадагш дамжуулахгүй
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 1-р хэлбэр: тухайлсан нэгж талбар.
func TestExternalAPI_ByParcelID(t *testing.T) {
	r, seen := externalAPIRouter(t)
	w, body := callExternalAPI(t, r, "?parcel_id=1600900137")
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "1600900137", seen.ParcelID, "шүүлтүүр service рүү дамжсан")
	assert.Equal(t, "parcel", body.Data.Mode)
	assert.Equal(t, "1600900137", body.Data.Filter.ParcelID)

	require.Len(t, body.Data.Parcels, 1)
	p := body.Data.Parcels[0]
	// Өртсөн талбай ба нөхөх олговрын задаргаа
	assert.Equal(t, 4200.0, p.AcquisitionAreaM2)
	assert.Equal(t, "Эзэмших", p.RightTypeName)
	assert.Equal(t, 84_000_000.0, p.Compensation.Land)
	assert.Equal(t, 136_628_143.92, p.Compensation.RealEstate)
	assert.Equal(t, 77_211_810.0, p.Compensation.Property)
	assert.InDelta(t, 297_839_953.92, p.Compensation.Total, 0.01)
}

// 2-р хэлбэр: чөлөөлөлтийн ТӨЛӨВЛӨГӨӨНИЙ нэгж талбарын дугаар.
func TestExternalAPI_ByPlanParcelID(t *testing.T) {
	r, seen := externalAPIRouter(t)
	w, body := callExternalAPI(t, r, "?plan_parcel_id=5029776")
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "5029776", seen.PlanParcelID)
	assert.Empty(t, seen.ParcelID)
	assert.Equal(t, "plan_parcel", body.Data.Mode)

	// Чөлөөлөлтийн НИЙТ дүн ба нэгж талбаруудын ТӨЛӨВИЙН ТООЛОЛ
	assert.Equal(t, 2, body.Data.Summary.ParcelCount)
	assert.Equal(t, 9200.0, body.Data.Summary.AcquisitionAreaM2)
	assert.InDelta(t, 297_839_953.92, body.Data.Summary.Compensation.Total, 0.01)
	assert.Equal(t, "MNT", body.Data.Summary.Compensation.Currency)
	require.Len(t, body.Data.Summary.StatusCounts, 2)
	assert.Equal(t, "Үнэлгээ хийх", body.Data.Summary.StatusCounts[0].StatusName)
	assert.Equal(t, 1, body.Data.Summary.StatusCounts[0].Count)
	assert.Equal(t, "Чөлөөлсөн", body.Data.Summary.StatusCounts[1].StatusName)
}

// 3-р хэлбэр: төлөвлөгөөний дугаар. ЯГ таарлаар шүүнэ (хэсэгчилсэн биш).
func TestExternalAPI_ByPlanCodeUsesExactMatch(t *testing.T) {
	r, seen := externalAPIRouter(t)
	w, body := callExternalAPI(t, r, "?plan_code=2026-04-011-9878")
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "2026-04-011-9878", seen.PlanCodeExact, "гадаад API яг таарлаар шүүнэ")
	assert.Empty(t, seen.PlanCode, "хэсэгчилсэн (ILIKE) шүүлтүүр хэрэглэгдэхгүй")
	assert.Equal(t, "plan", body.Data.Mode)
	assert.Equal(t, 1, body.Data.Summary.AcquisitionCount)
	assert.Len(t, body.Data.Summary.StatusCounts, 2)
}

func TestExternalAPI_RejectsBadAcquisitionUUIDAndCapsPageSize(t *testing.T) {
	r, seen := externalAPIRouter(t)

	w, _ := callExternalAPI(t, r, "?acquisition_id=zzz")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w, body := callExternalAPI(t, r, "?plan_code=2026-04-011-9878&page_size=5000")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 200, body.Data.PageSize, "дээд хязгаар 200")
	assert.Equal(t, 200, seen.PageSize)
}
