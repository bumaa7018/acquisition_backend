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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/bad-id", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-id"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetMe(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	user := &domain.User{ID: uuid.New(), Username: "newuser"}
	active := true
	mockSvc.On("Create", mock.Anything, service.CreateUserRequest{
		Username:  "newuser",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "User",
		IsActive:  &active,
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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

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
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-uuid"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/users/bad-uuid", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Нууц үг сэргээх ──────────────────────────────────────────────────────────

func TestUserHandler_ChangePassword_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := uuid.New()
	mockSvc.On("GetPermissions", mock.Anything, id).Return([]string{"land:read"}, nil)
	mockSvc.On("ChangePassword", mock.Anything, id, "Password123!").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update", "land:read"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+id.String()+"/password",
		bytes.NewBufferString(`{"password":"Password123!"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.ChangePassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Backend min=8 — 7 тэмдэгт нууц үг батлагдахгүй (frontend-тэй ижил дүрэм).
func TestUserHandler_ChangePassword_TooShort(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+id.String()+"/password",
		bytes.NewBufferString(`{"password":"Short1!"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "ChangePassword", mock.Anything, mock.Anything, mock.Anything)
}

// ── Өөрийгөө хаахаас хамгаалах (self-lockout) ────────────────────────────────

func TestUserHandler_Delete_RejectsSelf(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, id)
	c.Request = httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestUserHandler_Update_RejectsSelfDeactivation(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, id)
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+id.String(),
		bytes.NewBufferString(`{"is_active":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

// Бусад хэрэглэгчийг идэвхгүй болгох нь зөвшөөрөгдөнө.
func TestUserHandler_Update_AllowsDeactivatingOthers(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	callerID := uuid.New()
	targetID := uuid.New()
	inactive := false
	mockSvc.On("Update", mock.Anything, targetID, service.UpdateUserRequest{IsActive: &inactive}).
		Return(&domain.User{ID: targetID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, callerID)
	c.Set(middleware.CtxPermissions, []string{"users:update", "land:read"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+targetID.String(),
		bytes.NewBufferString(`{"is_active":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: targetID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ── Роль олгох / хураах ──────────────────────────────────────────────────────

func TestUserHandler_AssignRole_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	targetID := uuid.New()
	roleID := uuid.New()
	mockRole.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
		ID:          roleID,
		Name:        "employee",
		Permissions: []domain.Permission{{Name: "land:read"}},
	}, nil)
	mockRole.On("AssignToUser", mock.Anything, targetID, roleID).Return(nil)
	// Роль JWT-д агуулагддаг тул шинэ эрх шууд хүчинтэй болохын тулд сесс тасална.
	mockSvc.On("InvalidateSessions", mock.Anything, targetID).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"land:read", "users:update"})
	body, _ := json.Marshal(map[string]string{"role_id": roleID.String()})
	c.Request = httptest.NewRequest(http.MethodPost, "/users/"+targetID.String()+"/roles", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: targetID.String()}}

	h.AssignRole(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRole.AssertExpectations(t)
	mockSvc.AssertExpectations(t)
}

// Эрх нэмэгдүүлэлт: дуудагчид байхгүй эрх агуулсан роль олгогдохгүй.
func TestUserHandler_AssignRole_RejectsPrivilegeEscalation(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	targetID := uuid.New()
	roleID := uuid.New()
	mockRole.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
		ID:   roleID,
		Name: "admin",
		Permissions: []domain.Permission{
			{Name: "users:update"},
			{Name: "users:delete"}, // дуудагчид байхгүй
		},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	body, _ := json.Marshal(map[string]string{"role_id": roleID.String()})
	c.Request = httptest.NewRequest(http.MethodPost, "/users/"+targetID.String()+"/roles", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: targetID.String()}}

	h.AssignRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRole.AssertNotCalled(t, "AssignToUser", mock.Anything, mock.Anything, mock.Anything)
}

// Өөртөө роль олгохыг хориглоно (self-escalation).
func TestUserHandler_AssignRole_RejectsSelf(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	selfID := uuid.New()
	roleID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, selfID)
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	body, _ := json.Marshal(map[string]string{"role_id": roleID.String()})
	c.Request = httptest.NewRequest(http.MethodPost, "/users/"+selfID.String()+"/roles", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: selfID.String()}}

	h.AssignRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRole.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	mockRole.AssertNotCalled(t, "AssignToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_RemoveRole_RejectsHigherPrivilegeRole(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	targetID := uuid.New()
	roleID := uuid.New()
	mockRole.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []domain.Permission{{Name: "users:delete"}},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodDelete,
		"/users/"+targetID.String()+"/roles/"+roleID.String(), nil)
	c.Params = gin.Params{
		gin.Param{Key: "id", Value: targetID.String()},
		gin.Param{Key: "role_id", Value: roleID.String()},
	}

	h.RemoveRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRole.AssertNotCalled(t, "RemoveFromUser", mock.Anything, mock.Anything, mock.Anything)
}

// Хэрэглэгч үүсгэхэд эрхгүй роль дурдвал хэрэглэгч БҮРТГЭГДЭХГҮЙ (хагас
// биелэхээс сэргийлнэ).
func TestUserHandler_Create_RejectsEscalatingRoleBeforeCreating(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	roleID := uuid.New()
	mockRole.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []domain.Permission{{Name: "users:delete"}},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxPermissions, []string{"users:create"})
	body, _ := json.Marshal(map[string]any{
		"username":   "newuser",
		"email":      "new@example.mn",
		"password":   "Password123!",
		"first_name": "New",
		"last_name":  "User",
		"role_ids":   []string{roleID.String()},
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// ── Өөрөөсөө өндөр эрхтэй бүртгэлийг гартаа авахаас хамгаалах ────────────────

// users:update эрхтэй хэрэглэгч админы нууц үгийг сэргээж, түүгээр нэвтрэх
// боломжгүй байх ёстой.
func TestUserHandler_ChangePassword_RejectsHigherPrivilegedTarget(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	adminID := uuid.New()
	mockSvc.On("GetPermissions", mock.Anything, adminID).
		Return([]string{"users:update", "users:delete", "roles:update"}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+adminID.String()+"/password",
		bytes.NewBufferString(`{"password":"Password123!"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: adminID.String()}}

	h.ChangePassword(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "ChangePassword", mock.Anything, mock.Anything, mock.Anything)
}

// Имэйл сольж /auth/forgot-password-оор бүртгэлийг гартаа авах зам хаагдсан.
func TestUserHandler_Update_RejectsHigherPrivilegedTarget(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	adminID := uuid.New()
	mockSvc.On("GetByID", mock.Anything, adminID).
		Return(&domain.User{ID: adminID, Email: "admin@example.com"}, nil)
	mockSvc.On("GetPermissions", mock.Anything, adminID).
		Return([]string{"users:update", "users:delete"}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+adminID.String(),
		bytes.NewBufferString(`{"email":"attacker@evil.mn"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: adminID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

// Өөрийн бүртгэлээ засахад "хамрах" шалгалт хийгдэхгүй (өөрөө өөртэйгөө тэнцүү).
func TestUserHandler_Update_AllowsSelfEdit(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	selfID := uuid.New()
	mockSvc.On("Update", mock.Anything, selfID, service.UpdateUserRequest{FirstName: "Мөнх"}).
		Return(&domain.User{ID: selfID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, selfID)
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+selfID.String(),
		bytes.NewBufferString(`{"first_name":"Мөнх"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: selfID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// Өөрөө дээрээ GetPermissions дуудагдахгүй.
	mockSvc.AssertNotCalled(t, "GetPermissions", mock.Anything, mock.Anything)
}

// Имэйл ХӨНДӨӨГҮЙ засвар (нэр/албан тушаал) нь өндөр эрхтэй хэрэглэгч дээр ч
// зөвшөөрөгдөнө — эрх нэмэгдүүлэх зам биш тул.
func TestUserHandler_Update_AllowsNonEmailEditOfHigherPrivilegedTarget(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	adminID := uuid.New()
	mockSvc.On("Update", mock.Anything, adminID, service.UpdateUserRequest{Position: "Дарга"}).
		Return(&domain.User{ID: adminID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+adminID.String(),
		bytes.NewBufferString(`{"position":"Дарга"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: adminID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertNotCalled(t, "GetPermissions", mock.Anything, mock.Anything)
}

// Имэйлийг ижил хэвээр (өөрчлөлтгүй) илгээвэл шалгалт хийгдэхгүй — UI нь бүх
// талбарыг илгээдэг тул энэ нь хэвийн засвар.
func TestUserHandler_Update_UnchangedEmailSkipsRankCheck(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	targetID := uuid.New()
	mockSvc.On("GetByID", mock.Anything, targetID).
		Return(&domain.User{ID: targetID, Email: "boss@example.mn"}, nil)
	mockSvc.On("Update", mock.Anything, targetID, service.UpdateUserRequest{
		Email:     "Boss@example.mn", // том/жижиг үсгийн зөрүү — өөрчлөлт биш
		FirstName: "Бат",
	}).Return(&domain.User{ID: targetID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, uuid.New())
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+targetID.String(),
		bytes.NewBufferString(`{"email":"Boss@example.mn","first_name":"Бат"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: targetID.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertNotCalled(t, "GetPermissions", mock.Anything, mock.Anything)
}
