package http_test

// POST /land-acquisitions/:id/parcels/by-acquisition — чөлөөлөх хилээр нэгж
// талбарыг тодорхойлж бүртгэх.
//
// Энэ маршрут нь /:id/parcels/:parcel_id/sync-тэй ижил түвшинд статик
// сегментээр сууна. Тиймээс "by-acquisition" нь sync маршрутыг далдлаагүй
// эсэхийг маршрутын түвшинд БАС шалгана.

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

func TestLandHandler_DiscoverParcels_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	mockSvc.On("DiscoverParcels", mock.Anything, acqID).Return(&domain.ParcelDiscoveryResult{
		Total: 3, Created: 2, Existing: 1,
		ParcelIDs: []string{"010101000001", "010101000002", "010101000003"},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/land-acquisitions/"+acqID.String()+"/parcels/by-acquisition", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.DiscoverParcels(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data domain.ParcelDiscoveryResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Data.Total)
	assert.Equal(t, 2, resp.Data.Created)
	assert.Equal(t, 1, resp.Data.Existing)
	assert.Len(t, resp.Data.ParcelIDs, 3)
	mockSvc.AssertExpectations(t)
}

func TestLandHandler_DiscoverParcels_InvalidAcqUUID(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/land-acquisitions/bad/parcels/by-acquisition", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	h.DiscoverParcels(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "DiscoverParcels", mock.Anything, mock.Anything)
}

func TestLandHandler_DiscoverParcels_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	acqID := uuid.New()
	mockSvc.On("DiscoverParcels", mock.Anything, acqID).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/land-acquisitions/"+acqID.String()+"/parcels/by-acquisition", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.DiscoverParcels(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Статик "by-acquisition" сегмент нь :parcel_id параметрийн маршрутыг
// далдлахгүй — хоёул зөв handler руу очно.
func TestRouting_DiscoverParcelsDoesNotShadowSync(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	var hit string
	var gotParcelID string
	land := r.Group("/land-acquisitions")
	land.POST("/:id/parcels/by-acquisition", func(c *gin.Context) {
		hit = "discover"
		c.Status(http.StatusOK)
	})
	land.POST("/:id/parcels/:parcel_id/sync", func(c *gin.Context) {
		hit = "sync"
		gotParcelID = c.Param("parcel_id")
		c.Status(http.StatusOK)
	})

	acqID := uuid.New().String()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
		"/land-acquisitions/"+acqID+"/parcels/by-acquisition", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "discover", hit)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
		"/land-acquisitions/"+acqID+"/parcels/010101000001/sync", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sync", hit)
	assert.Equal(t, "010101000001", gotParcelID)
}
