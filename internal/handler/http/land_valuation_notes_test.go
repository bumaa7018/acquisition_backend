package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Үнэлгээний хүснэгтийн ТАЙЛБАР — жагсаах/хадгалах handler-ууд.

func sampleSectionNote(acqID uuid.UUID, key, note string) *domain.ValuationSectionNote {
	return &domain.ValuationSectionNote{
		ID:            uuid.New(),
		AcquisitionID: acqID,
		ParcelID:      "010101000001",
		ValuationType: domain.ValuationTypeAsset,
		SectionKey:    key,
		Note:          note,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestLandHandler_ListValuationNotes_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)
	acqID := uuid.New()

	notes := []*domain.ValuationSectionNote{
		sampleSectionNote(acqID, "land_valuation", "Суурь үнийг бүсчлэлээр тодотгов"),
		sampleSectionNote(acqID, "building_cost", "Элэгдлийг ашиглалтын хугацаагаар тогтоов"),
	}
	mockSvc.On("ListValuationNotes", mock.Anything, acqID, "010101000001", "asset").Return(notes, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/land-acquisitions/x/valuation-notes?parcel_id=010101000001&valuation_type=asset", nil)
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}

	h.ListValuationNotes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp["data"].([]any), 2)
	mockSvc.AssertExpectations(t)
}

// parcel_id байхгүй бол бүх нэгж талбарын тайлбарыг буцаах эрсдэлтэй тул 400.
func TestLandHandler_ListValuationNotes_RequiresParcel(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/land-acquisitions/x/valuation-notes", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ListValuationNotes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "ListValuationNotes", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLandHandler_SaveValuationNotes_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)
	acqID := uuid.New()

	notes := map[string]string{"land_valuation": "Шинэчилсэн тайлбар"}
	saved := []*domain.ValuationSectionNote{sampleSectionNote(acqID, "land_valuation", "Шинэчилсэн тайлбар")}
	mockSvc.On("SaveValuationNotes", mock.Anything, acqID, "010101000001", "asset", notes, "7").Return(saved, nil)

	body, err := json.Marshal(map[string]any{
		"parcel_id":      "010101000001",
		"valuation_type": "asset",
		"notes":          notes,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/x/valuation-notes", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: acqID.String()}}
	c.Set("userID", domain.ID(7))

	h.SaveValuationNotes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp["data"].([]any), 1)
	mockSvc.AssertExpectations(t)
}

// parcel_id заавал — эс бөгөөс аль нэгж талбарт бичихийг тодорхойлох боломжгүй.
func TestLandHandler_SaveValuationNotes_RequiresParcel(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewLandHandler(mockSvc)

	body := []byte(`{"notes":{"summary":"x"}}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/land-acquisitions/x/valuation-notes", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.SaveValuationNotes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "SaveValuationNotes", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
