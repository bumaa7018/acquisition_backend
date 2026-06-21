package http_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/handler/http/mocks"
	"github.com/your-org/go-api-template/internal/service"
)

// buildTestShp — тест зориулсан raw .shp файлын байт
func buildTestShp() []byte {
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

// buildMultipartForm нь multipart/form-data хүсэлт үүсгэнэ.
func buildMultipartForm(t *testing.T, fields map[string]string, fileKey string, fileData []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	if fileData != nil {
		fw, err := mw.CreateFormFile(fileKey, "boundary.zip")
		require.NoError(t, err)
		_, err = fw.Write(fileData)
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())
	return &body, mw.FormDataContentType()
}

func sampleAcquisition() *domain.LandAcquisition {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	return &domain.LandAcquisition{
		ID:          uuid.New(),
		PlanCode:    "PLAN-2024-001",
		GeometryWKT: "POLYGON((106.76 47.66,106.82 47.66,106.82 47.72,106.76 47.72,106.76 47.66))",
		AreaM2:      85000,
		Status:      domain.LandAcquisitionStatusField,
		StartDate:   &start,
		EndDate:     &end,
		CreatedAt:   time.Now(),
		AUs: []domain.LandAcquisitionAU{
			{AU1Code: "01", AU2Code: "0101", AU3Code: "010101"},
		},
		Parcels: []*domain.Parcel{
			{ID: uuid.New(), ParcelID: "010101000001"},
		},
	}
}

// ── List ──────────────────────────────────────────────────────

func TestLandHandler_List_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	filter := domain.LandAcquisitionFilter{Page: 1, PageSize: 20}
	list := []*domain.LandAcquisition{sampleAcquisition(), sampleAcquisition()}
	mockSvc.On("List", mock.Anything, filter).Return(list, int64(2), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["total"])
	assert.Len(t, resp["data"].([]any), 2)
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_List_WithFilter(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	filter := domain.LandAcquisitionFilter{
		PlanCode: "PLAN-2024",
		Status:   domain.LandAcquisitionStatusField,
		Page:     2,
		PageSize: 10,
	}
	mockSvc.On("List", mock.Anything, filter).Return([]*domain.LandAcquisition{sampleAcquisition()}, int64(11), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/land-acquisitions?plan_code=PLAN-2024&status=2&page=2&page_size=10", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(11), resp["total"])
	assert.Equal(t, float64(2), resp["page"])
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_List_Empty(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	filter := domain.LandAcquisitionFilter{Page: 1, PageSize: 20}
	mockSvc.On("List", mock.Anything, filter).Return([]*domain.LandAcquisition{}, int64(0), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total"])
}

// ── Create ────────────────────────────────────────────────────

func TestLandHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acq := sampleAcquisition()
	mockSvc.On("Create", mock.Anything, mock.MatchedBy(func(r service.CreateLandAcquisitionRequest) bool {
		return r.PlanCode == "PLAN-2024-001"
	})).Return(acq, nil)

	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code":  "PLAN-2024-001",
		"start_date": "2024-01-01",
		"end_date":   "2024-12-31",
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions", body)
	c.Request.Header.Set("Content-Type", ct)

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_Create_MissingFields(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code": "PLAN-2024-001",
		// start_date, end_date missing
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions", body)
	c.Request.Header.Set("Content-Type", ct)

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

func TestLandHandler_Create_BadDateFormat(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code":  "PLAN-2024-001",
		"start_date": "01-01-2024", // wrong format
		"end_date":   "2024-12-31",
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions", body)
	c.Request.Header.Set("Content-Type", ct)

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLandHandler_Create_NoShapefile(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code":  "PLAN-2024-001",
		"start_date": "2024-01-01",
		"end_date":   "2024-12-31",
	}, "", nil) // no file

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions", body)
	c.Request.Header.Set("Content-Type", ct)

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLandHandler_Create_PlanNotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)

	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code":  "UNKNOWN",
		"start_date": "2024-01-01",
		"end_date":   "2024-12-31",
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions", body)
	c.Request.Header.Set("Content-Type", ct)

	h.Create(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLandHandler_Create_GeometryOutside(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidInput)

	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code":  "PLAN-2024-001",
		"start_date": "2024-01-01",
		"end_date":   "2024-12-31",
	}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions", body)
	c.Request.Header.Set("Content-Type", ct)

	h.Create(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── GetByID ───────────────────────────────────────────────────

func TestLandHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acq := sampleAcquisition()
	mockSvc.On("GetByID", mock.Anything, acq.ID).Return(acq, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/"+acq.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: acq.ID.String()}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, acq.ID.String(), data["id"])
	assert.Equal(t, "PLAN-2024-001", data["plan_code"])
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLandHandler_GetByID_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/bad", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-uuid"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── ListParcels ───────────────────────────────────────────────

func TestLandHandler_ListParcels_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	parcels := []*domain.Parcel{
		{ID: uuid.New(), ParcelID: "010101000001", AU1Code: "01", AU2Code: "0101", AU3Code: "010101", RightType: domain.RightTypeOwnership, Landuse: "AGRICULTURAL", AreaM2: 85000},
		{ID: uuid.New(), ParcelID: "010101000002", AU1Code: "01", AU2Code: "0101", AU3Code: "010101", RightType: domain.RightTypeLease, Landuse: "RESIDENTIAL", AreaM2: 1200},
	}
	filter := domain.ParcelFilter{Page: 1, PageSize: 20}
	mockSvc.On("GetParcels", mock.Anything, acqID, filter).Return(parcels, int64(2), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/"+acqID.String()+"/parcels", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.ListParcels(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 2)
	assert.Equal(t, float64(2), resp["total"])
	assert.Equal(t, float64(1), resp["page"])
	first := data[0].(map[string]any)
	assert.Equal(t, "010101000001", first["parcel_id"])
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_ListParcels_WithFilter(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	filter := domain.ParcelFilter{AU3Code: "010101", RightType: domain.RightTypeOwnership, Page: 1, PageSize: 10}
	mockSvc.On("GetParcels", mock.Anything, acqID, filter).Return([]*domain.Parcel{
		{ID: uuid.New(), ParcelID: "010101000001", RightType: domain.RightTypeOwnership},
	}, int64(1), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/land-acquisitions/"+acqID.String()+"/parcels?au3_code=010101&right_type=3&page_size=10", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.ListParcels(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_ListParcels_Empty(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	filter := domain.ParcelFilter{Page: 1, PageSize: 20}
	mockSvc.On("GetParcels", mock.Anything, acqID, filter).Return([]*domain.Parcel{}, int64(0), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/"+acqID.String()+"/parcels", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.ListParcels(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total"])
}

func TestLandHandler_ListParcels_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	filter := domain.ParcelFilter{Page: 1, PageSize: 20}
	mockSvc.On("GetParcels", mock.Anything, acqID, filter).Return(nil, int64(0), domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/"+acqID.String()+"/parcels", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.ListParcels(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLandHandler_ListParcels_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/bad/parcels", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-uuid"}}

	h.ListParcels(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Update ────────────────────────────────────────────────────

func TestLandHandler_Update_DatesOnly(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acq := sampleAcquisition()
	mockSvc.On("Update", mock.Anything, acq.ID, mock.MatchedBy(func(r service.UpdateLandAcquisitionRequest) bool {
		return r.PlanCode == "" && len(r.ShapefileData) == 0
	})).Return(acq, false, nil)

	body, ct := buildMultipartForm(t, map[string]string{
		"end_date": "2025-06-30",
	}, "", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+acq.ID.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: acq.ID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_Update_WithShapefile(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acq := sampleAcquisition()
	mockSvc.On("Update", mock.Anything, acq.ID, mock.MatchedBy(func(r service.UpdateLandAcquisitionRequest) bool {
		return len(r.ShapefileData) > 0
	})).Return(acq, true, nil)

	body, ct := buildMultipartForm(t, map[string]string{}, "shapefile", buildTestShp())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+acq.ID.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: acq.ID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["warning"]) // геометр өөрчлөгдсөн тул анхааруулга гарна
}

func TestLandHandler_Update_PlanChangedWithoutShapefile(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	body, ct := buildMultipartForm(t, map[string]string{
		"plan_code": "PLAN-2025-002",
	}, "", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+id.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "Update")
}

func TestLandHandler_Update_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Update", mock.Anything, id, mock.Anything).Return(nil, false, domain.ErrNotFound)

	body, ct := buildMultipartForm(t, map[string]string{"end_date": "2025-12-31"}, "", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/"+id.String(), body)
	c.Request.Header.Set("Content-Type", ct)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Delete ────────────────────────────────────────────────────

func TestLandHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/land-acquisitions/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_Delete_NotNewStatus(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(domain.ErrForbidden)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/land-acquisitions/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestLandHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/land-acquisitions/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── SyncParcel ────────────────────────────────────────────────

func TestLandHandler_SyncParcel_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	parcelID := uuid.New()
	mockSvc.On("SyncParcel", mock.Anything, acqID, parcelID).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String()+"/sync", nil)
	c.Params = gin.Params{
		{Key: "id", Value: acqID.String()},
		{Key: "parcel_id", Value: parcelID.String()},
	}

	h.SyncParcel(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(http.StatusOK), resp["code"])
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_SyncParcel_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	parcelID := uuid.New()
	mockSvc.On("SyncParcel", mock.Anything, acqID, parcelID).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String()+"/sync", nil)
	c.Params = gin.Params{
		{Key: "id", Value: acqID.String()},
		{Key: "parcel_id", Value: parcelID.String()},
	}

	h.SyncParcel(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLandHandler_SyncParcel_InvalidAcqUUID(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions/bad/parcels/any/sync", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "not-a-uuid"},
		{Key: "parcel_id", Value: uuid.New().String()},
	}

	h.SyncParcel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLandHandler_SyncParcel_InvalidParcelUUID(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions/"+uuid.New().String()+"/parcels/bad/sync", nil)
	c.Params = gin.Params{
		{Key: "id", Value: uuid.New().String()},
		{Key: "parcel_id", Value: "not-a-uuid"},
	}

	h.SyncParcel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
