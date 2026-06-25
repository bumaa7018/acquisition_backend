package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/handler/http/mocks"
	"github.com/your-org/go-api-template/internal/service"
)

func TestParcelStatusHandler_List(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	list := []*domain.ParcelStatus{
		{ID: 1, Code: "negotiating", Name: "Зөвшилцөх шатандаа", SortOrder: 1},
		{ID: 2, Code: "nosk", Name: "НОСК", SortOrder: 2},
	}
	mockSvc.On("List", mock.Anything).Return(list, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/parcel-statuses", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	ps := &domain.ParcelStatus{ID: 1, Code: "negotiating", Name: "Зөвшилцөх шатандаа", SortOrder: 1}
	mockSvc.On("GetByID", mock.Anything, 1).Return(ps, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/parcel-statuses/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	mockSvc.On("GetByID", mock.Anything, 99).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/parcel-statuses/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_GetByID_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/parcel-statuses/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParcelStatusHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	ps := &domain.ParcelStatus{ID: 6, Code: "pending", Name: "Хүлээгдэж байна", SortOrder: 6}
	mockSvc.On("Create", mock.Anything, service.CreateParcelStatusRequest{
		Code:      "pending",
		Name:      "Хүлээгдэж байна",
		SortOrder: 6,
	}).Return(ps, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"code":"pending","name":"Хүлээгдэж байна","sort_order":6}`
	c.Request = httptest.NewRequest(http.MethodPost, "/parcel-statuses", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_Create_BadJSON(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/parcel-statuses", bytes.NewBufferString("not-json"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParcelStatusHandler_Update_Success(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	ps := &domain.ParcelStatus{ID: 1, Code: "negotiating", Name: "Шинэ нэр", SortOrder: 1}
	mockSvc.On("Update", mock.Anything, 1, service.UpdateParcelStatusRequest{
		Name:      "Шинэ нэр",
		SortOrder: 1,
	}).Return(ps, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Шинэ нэр","sort_order":1}`
	c.Request = httptest.NewRequest(http.MethodPut, "/parcel-statuses/1", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_Update_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/parcel-statuses/abc", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParcelStatusHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 1).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/parcel-statuses/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 99).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/parcel-statuses/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestParcelStatusHandler_Delete_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockParcelStatusService{}
	h := handler.NewParcelStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/parcel-statuses/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
