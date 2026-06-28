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
)

func TestAcquisitionWorkflowHandler_List_Empty(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	mockSvc.On("ListAcquisitionWorkflows", mock.Anything).Return([]*domain.AcquisitionWorkflow{}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-workflow", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionWorkflowHandler_List_WithData(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	from := 1
	workflows := []*domain.AcquisitionWorkflow{
		{ID: 1, FromStatusID: &from, ToStatusID: 2, FromStatusName: "Шинэ", ToStatusName: "Хээрийн судалгаа", SortOrder: 0},
	}
	mockSvc.On("ListAcquisitionWorkflows", mock.Anything).Return(workflows, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-workflow", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, 1)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionWorkflowHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	from := 1
	w2 := &domain.AcquisitionWorkflow{
		ID: 1, FromStatusID: &from, ToStatusID: 2,
		FromStatusName: "Шинэ", ToStatusName: "Хээрийн судалгаа", SortOrder: 0,
	}
	mockSvc.On("CreateAcquisitionWorkflow", mock.Anything, &from, 2, 0).Return(w2, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"from_status_id":1,"to_status_id":2,"sort_order":0}`
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-workflow", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionWorkflowHandler_Create_NoToStatus(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"from_status_id":1}`
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-workflow", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionWorkflowHandler_Create_InvalidToStatus(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"from_status_id":1,"to_status_id":99}`
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-workflow", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionWorkflowHandler_Create_InvalidFromStatus(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"from_status_id":99,"to_status_id":2}`
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-workflow", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionWorkflowHandler_Create_BadJSON(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-workflow", bytes.NewBufferString("not-json"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionWorkflowHandler_Create_NullFromStatus(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w2 := &domain.AcquisitionWorkflow{ID: 2, FromStatusID: nil, ToStatusID: 1, ToStatusName: "Шинэ", SortOrder: 0}
	mockSvc.On("CreateAcquisitionWorkflow", mock.Anything, (*int)(nil), 1, 0).Return(w2, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"to_status_id":1,"sort_order":0}`
	c.Request = httptest.NewRequest(http.MethodPost, "/acquisition-workflow", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionWorkflowHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	mockSvc.On("DeleteAcquisitionWorkflow", mock.Anything, 1).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/acquisition-workflow/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionWorkflowHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	mockSvc.On("DeleteAcquisitionWorkflow", mock.Anything, 99).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/acquisition-workflow/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAcquisitionWorkflowHandler_Delete_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/acquisition-workflow/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcquisitionWorkflowHandler_ListStatuses(t *testing.T) {
	mockSvc := &mocks.MockLandService{}
	h := handler.NewAcquisitionWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/acquisition-workflow/statuses", nil)

	h.ListStatuses(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].([]any)
	require.True(t, ok)
	assert.Len(t, data, len(domain.AcquisitionStatusOrder))
}
