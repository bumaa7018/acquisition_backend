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

func TestAcquisitionProgressStatusHandler_List(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	desc := "тест тайлбар"
	order := 1
	list := []*domain.AcquisitionProgressStatus{
		{ID: 1, Name: "Явцын статус 1", Description: &desc, SortOrder: &order},
	}
	mockSvc.On("List", mock.Anything).Return(list, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-progress-statuses", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	aps := &domain.AcquisitionProgressStatus{ID: 1, Name: "Явцын статус 1"}
	mockSvc.On("GetByID", mock.Anything, 1).Return(aps, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-progress-statuses/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	mockSvc.On("GetByID", mock.Anything, 99).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-progress-statuses/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_GetByID_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-progress-statuses/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionProgressStatusHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	aps := &domain.AcquisitionProgressStatus{ID: 1, Name: "Шинэ статус"}
	mockSvc.On("Create", mock.Anything, service.CreateAcquisitionProgressStatusRequest{
		Name: "Шинэ статус",
	}).Return(aps, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Шинэ статус"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-progress-statuses", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_Create_BadJSON(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-progress-statuses", bytes.NewBufferString("not-json"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionProgressStatusHandler_Update_Success(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	aps := &domain.AcquisitionProgressStatus{ID: 1, Name: "Шинэчлэгдсэн"}
	mockSvc.On("Update", mock.Anything, 1, service.UpdateAcquisitionProgressStatusRequest{
		Name: "Шинэчлэгдсэн",
	}).Return(aps, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Шинэчлэгдсэн"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/acquisition-progress-statuses/1", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_Update_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/acquisition-progress-statuses/abc", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionProgressStatusHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 1).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/acquisition-progress-statuses/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 99).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/acquisition-progress-statuses/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionProgressStatusHandler_Delete_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockAcquisitionProgressStatusService{}
	h := handler.NewAcquisitionProgressStatusHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/acquisition-progress-statuses/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
