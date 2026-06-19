package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/handler/http/mocks"
	"github.com/your-org/go-api-template/internal/middleware"
)

func TestUserHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	id := uuid.New()
	user := &domain.User{ID: id, Email: "u@example.com"}
	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserHandler_GetByID_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/bad-id", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-id"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetMe(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	id := uuid.New()
	user := &domain.User{ID: id, Email: "me@example.com"}
	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/me", nil)
	c.Set(middleware.CtxUserID, id)

	h.GetMe(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}
