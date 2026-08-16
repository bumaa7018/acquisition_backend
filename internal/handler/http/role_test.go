package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRoleHandler_List(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	roles := []*domain.Role{{ID: domain.ID(1), Name: "admin"}}
	mockSvc.On("List", mock.Anything).Return(roles, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/roles", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	id := domain.ID(2)
	role := &domain.Role{ID: id, Name: "admin"}
	mockSvc.On("GetByID", mock.Anything, id).Return(role, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/roles/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.GetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_GetByID_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-uuid"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/roles/bad-uuid", nil)

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_GetByID_NotFound(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	id := domain.ID(3)
	mockSvc.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/roles/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoleHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	role := &domain.Role{ID: domain.ID(4), Name: "editor"}
	mockSvc.On("Create", mock.Anything, service.CreateRoleRequest{Name: "editor", Description: "Editor role"}).Return(role, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"editor","description":"Editor role"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_Create_ValidationError(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"description":"no name"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Update_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	id := domain.ID(5)
	role := &domain.Role{ID: id, Name: "editor-updated"}
	mockSvc.On("Update", mock.Anything, id, service.UpdateRoleRequest{Name: "editor-updated", Description: ""}).Return(role, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"editor-updated"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/roles/"+strconv.Itoa(int(id)), bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_Update_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-uuid"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/roles/bad-uuid", bytes.NewBufferString(`{"name":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	id := domain.ID(6)
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	id := domain.ID(7)
	mockSvc.On("Delete", mock.Anything, id).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoleHandler_AssignPermission_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	mockPerm := &mocks.MockPermissionService{}
	h := handler.NewRoleHandler(mockSvc, mockPerm)

	roleID := domain.ID(8)
	permID := domain.ID(9)
	// Дуудагч тухайн эрхийг өөрөө эзэмшиж байж л олгож чадна (escalation хамгаалалт).
	mockSvc.On("GetByID", mock.Anything, roleID).Return(&domain.Role{ID: roleID}, nil)
	mockPerm.On("GetByID", mock.Anything, permID).Return(&domain.Permission{ID: permID, Name: "land:update", Resource: "land"}, nil)
	mockSvc.On("AssignPermission", mock.Anything, roleID, permID).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("permissions", []string{"land:update"})
	body, _ := json.Marshal(map[string]string{"permission_id": strconv.Itoa(int(permID))})
	c.Request = httptest.NewRequest(http.MethodPost, "/roles/"+strconv.Itoa(int(roleID))+"/permissions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(roleID))}}

	h.AssignPermission(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_AssignPermission_InvalidRoleUUID(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/roles/bad/permissions", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.AssignPermission(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_RemovePermission_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	mockPerm := &mocks.MockPermissionService{}
	h := handler.NewRoleHandler(mockSvc, mockPerm)

	roleID := domain.ID(10)
	permID := domain.ID(11)
	mockSvc.On("GetByID", mock.Anything, roleID).Return(&domain.Role{ID: roleID}, nil)
	mockPerm.On("GetByID", mock.Anything, permID).Return(&domain.Permission{ID: permID, Name: "land:update", Resource: "land"}, nil)
	mockSvc.On("RemovePermission", mock.Anything, roleID, permID).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("permissions", []string{"land:update"})
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/"+strconv.Itoa(int(roleID))+"/permissions/"+strconv.Itoa(int(permID)), nil)
	c.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(int(roleID))},
		{Key: "permission_id", Value: strconv.Itoa(int(permID))},
	}

	h.RemovePermission(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_RemovePermission_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "bad"},
		{Key: "permission_id", Value: "12"},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/bad/permissions/"+"13", nil)

	h.RemovePermission(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
