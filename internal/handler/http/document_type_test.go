package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/handler/http/mocks"
	"github.com/your-org/go-api-template/internal/service"
)

func sampleDocumentType() *domain.DocumentType {
	desc := "тест тайлбар"
	return &domain.DocumentType{
		ID:          1,
		Type:        "contract",
		Name:        "Гэрээ",
		Description: &desc,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestDocumentTypeHandler_List(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	list := []*domain.DocumentType{sampleDocumentType()}
	mockSvc.On("List", mock.Anything, "").Return(list, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/document-types", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	dt := sampleDocumentType()
	mockSvc.On("GetByID", mock.Anything, 1).Return(dt, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/document-types/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	mockSvc.On("GetByID", mock.Anything, 99).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/document-types/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_GetByID_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/document-types/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentTypeHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	dt := sampleDocumentType()
	mockSvc.On("Create", mock.Anything, mock.MatchedBy(func(req service.CreateDocumentTypeRequest) bool {
		return req.Type == "contract" && req.Name == "Гэрээ"
	})).Return(dt, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"type":"contract","name":"Гэрээ"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/document-types", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_Create_BadJSON(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/document-types", bytes.NewBufferString("not-json"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentTypeHandler_Update_Success(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	dt := sampleDocumentType()
	mockSvc.On("Update", mock.Anything, 1, mock.MatchedBy(func(req service.UpdateDocumentTypeRequest) bool {
		return req.Name == "Шинэ нэр"
	})).Return(dt, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Шинэ нэр"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/document-types/1", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_Update_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/document-types/abc", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDocumentTypeHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 1).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/document-types/1", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	mockSvc.On("Delete", mock.Anything, 99).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/document-types/99", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestDocumentTypeHandler_Delete_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockDocumentTypeService{}
	h := handler.NewDocumentTypeHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/document-types/abc", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "abc"}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
