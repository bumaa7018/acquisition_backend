package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ЗАСВАРЛАХ үед гараас оруулсан хилийн файл service рүү дамжина.
func TestLandHandler_Update_PassesShapefileToService(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acq := sampleAcquisition()
	shp := buildTestShp()
	mockSvc.On("Update", mock.Anything, acq.ID, mock.MatchedBy(func(r service.UpdateLandAcquisitionRequest) bool {
		// Файл дамжсан, төлөвлөгөөний дугаар хоосон (хоёр зам зэрэг явахгүй).
		return len(r.ShapefileData) == len(shp) && r.PlanCode == ""
	})).Return(acq, &domain.BoundaryChange{Added: 1}, nil)

	body, ct := buildMultipartForm(t, map[string]string{
		"acquisition_name": "Test Project",
	}, "shapefile", shp)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+acq.ID.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: acq.ID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Талбайн зөрүү хэтэрсэн бол 422 — хил хадгалагдахгүй.
func TestLandHandler_Update_BoundaryAreaDeviation(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Update", mock.Anything, id, mock.Anything).
		Return(nil, nil, domain.ErrBoundaryAreaDeviation)

	body, ct := buildMultipartForm(t, map[string]string{
		"acquisition_name": "Test Project",
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+id.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// Хоёр замыг зэрэг өгвөл 400 (service шийднэ, handler статусыг зөв буулгана).
func TestLandHandler_Update_BoundarySourceConflict(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Update", mock.Anything, id, mock.MatchedBy(func(r service.UpdateLandAcquisitionRequest) bool {
		return r.PlanCode == "PLAN-2025-002" && len(r.ShapefileData) > 0
	})).Return(nil, nil, domain.ErrBoundarySourceConflict)

	body, ct := buildMultipartForm(t, map[string]string{
		"acquisition_name": "Test Project",
		"plan_parcel_id":   "PLAN-2025-002",
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+id.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

// Урьдчилан харах: талбайн зөрүүг хувиар буцаана (хадгалахгүй).
func TestLandHandler_PreviewBoundary(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("PreviewBoundaryShapefile", mock.Anything, id, mock.Anything).
		Return(&service.BoundaryAreaCheck{
			GeometryWKT:       "POLYGON((106.76 47.66,106.82 47.66,106.82 47.72,106.76 47.72,106.76 47.66))",
			AreaM2:            1_100_000,
			ReferenceAreaM2:   1_000_000,
			DeviationRatio:    0.1,
			MaxDeviationRatio: 0.3,
			Accepted:          true,
		}, nil)

	body, ct := buildMultipartForm(t, map[string]string{}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions/"+id.String()+"/boundary-preview", body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.PreviewBoundary(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, 1_100_000.0, data["area_m2"])
	assert.Equal(t, 1_000_000.0, data["reference_area_m2"])
	assert.InDelta(t, 10.0, data["deviation_percent"], 1e-9)
	assert.InDelta(t, 30.0, data["max_deviation_percent"], 1e-9)
	assert.Equal(t, true, data["accepted"])
	mockSvc.AssertExpectations(t)
}

// Файлгүй урьдчилан харах хүсэлт — 400.
func TestLandHandler_PreviewBoundary_RequiresFile(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	body, ct := buildMultipartForm(t, map[string]string{}, "", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions/"+id.String()+"/boundary-preview", body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.PreviewBoundary(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "PreviewBoundaryShapefile", mock.Anything, mock.Anything, mock.Anything)
}
