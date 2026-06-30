package http_test

import (
	"bytes"
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

func TestPermissionHandler_List(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	perms := []*domain.Permission{{ID: uuid.New(), Name: "users:read"}}
	mockSvc.On("List", mock.Anything).Return(perms, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/permissions", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestPermissionHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	perm := &domain.Permission{ID: uuid.New(), Name: "land:read"}
	mockSvc.On("Create", mock.Anything, service.CreatePermissionRequest{
		Name:     "land:read",
		Resource: "land",
		Action:   "read",
	}).Return(perm, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"land:read","resource":"land","action":"read"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPermissionHandler_Create_ValidationError(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"land:read"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Create_Duplicate(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	mockSvc.On("Create", mock.Anything, mock.Anything).Return(nil, domain.ErrAlreadyExists)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"land:read","resource":"land","action":"read"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPermissionHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/permissions/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestPermissionHandler_Delete_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-uuid"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/permissions/bad-uuid", nil)

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockPermissionService{}
	h := handler.NewPermissionHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/permissions/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
