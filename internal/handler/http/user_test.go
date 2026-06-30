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
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestUserHandler_List(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	users := []*domain.User{{ID: uuid.New(), Username: "user1"}}
	mockSvc.On("List", mock.Anything, domain.UserFilter{
		Search:   "",
		Role:     "professional_org",
		Page:     2,
		PageSize: 25,
	}).Return(users, int64(1), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users?role=professional_org&page=2&page_size=25", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	user := &domain.User{ID: uuid.New(), Username: "newuser"}
	mockSvc.On("Create", mock.Anything, service.CreateUserRequest{
		Username:  "newuser",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "User",
		IsActive:  true,
	}).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"username":"newuser","password":"Password123!","first_name":"New","last_name":"User","is_active":true}`
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Create_ValidationError(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"password":"short"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Update_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	id := uuid.New()
	user := &domain.User{ID: id, Username: "updated"}
	mockSvc.On("Update", mock.Anything, id, service.UpdateUserRequest{FirstName: "Updated"}).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"first_name":"Updated"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+id.String(), bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Update_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-uuid"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/users/bad-uuid", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
