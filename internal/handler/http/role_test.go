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

func TestRoleHandler_List(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	roles := []*domain.Role{{ID: uuid.New(), Name: "admin"}}
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

	id := uuid.New()
	role := &domain.Role{ID: id, Name: "admin"}
	mockSvc.On("GetByID", mock.Anything, id).Return(role, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/roles/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

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

	id := uuid.New()
	mockSvc.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/roles/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoleHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	role := &domain.Role{ID: uuid.New(), Name: "editor"}
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

	id := uuid.New()
	role := &domain.Role{ID: id, Name: "editor-updated"}
	mockSvc.On("Update", mock.Anything, id, service.UpdateRoleRequest{Name: "editor-updated", Description: ""}).Return(role, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"editor-updated"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/roles/"+id.String(), bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

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

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestRoleHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	id := uuid.New()
	mockSvc.On("Delete", mock.Anything, id).Return(domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoleHandler_AssignPermission_Success(t *testing.T) {
	mockSvc := &mocks.MockRoleService{}
	mockPerm := &mocks.MockPermissionService{}
	h := handler.NewRoleHandler(mockSvc, mockPerm)

	roleID := uuid.New()
	permID := uuid.New()
	// Дуудагч тухайн эрхийг өөрөө эзэмшиж байж л олгож чадна (escalation хамгаалалт).
	mockPerm.On("GetByID", mock.Anything, permID).Return(&domain.Permission{ID: permID, Name: "land:update"}, nil)
	mockSvc.On("AssignPermission", mock.Anything, roleID, permID).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("permissions", []string{"land:update"})
	body, _ := json.Marshal(map[string]string{"permission_id": permID.String()})
	c.Request = httptest.NewRequest(http.MethodPost, "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: roleID.String()}}

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
	h := handler.NewRoleHandler(mockSvc, &mocks.MockPermissionService{})

	roleID := uuid.New()
	permID := uuid.New()
	mockSvc.On("RemovePermission", mock.Anything, roleID, permID).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/"+roleID.String()+"/permissions/"+permID.String(), nil)
	c.Params = gin.Params{
		{Key: "id", Value: roleID.String()},
		{Key: "permission_id", Value: permID.String()},
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
		{Key: "permission_id", Value: uuid.New().String()},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/roles/bad/permissions/"+uuid.New().String(), nil)

	h.RemovePermission(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
