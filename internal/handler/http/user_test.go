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
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testDepartment — тестийн дуудагч ба байны нийтлэг хэлтэс.
var testDepartment = domain.ID(7)

// sameDepartment — дуудагчийг АДМИН БИШ, байтай ИЖИЛ хэлтэст байлгана.
// Хэлтсийн хязгаарлалтаас бусад дүрмийг шалгах тестүүд үүгээр эхэлнэ.
func sameDepartment(c *gin.Context, svc *mocks.MockUserService) {
	c.Set(middleware.CtxDepartmentID, testDepartment)
	dept := testDepartment
	svc.On("DepartmentOf", mock.Anything, mock.Anything).Return(&dept, nil).Maybe()
}

func TestUserHandler_GetByID_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(1)
	user := &domain.User{ID: id, Email: "u@example.com"}
	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

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

	id := domain.ID(2)
	mockSvc.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.GetByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserHandler_GetByID_InvalidID(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/bad-id", nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "bad-id"}}

	h.GetByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetMe(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(3)
	user := &domain.User{ID: id, Email: "me@example.com"}
	mockSvc.On("GetByID", mock.Anything, id).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/me", nil)
	c.Set(middleware.CtxUserID, id)

	h.GetMe(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Delete_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(4)
	mockSvc.On("Delete", mock.Anything, id).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodDelete, "/users/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Delete(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_List(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	users := []*domain.User{{ID: domain.ID(5), Username: "user1"}}
	mockSvc.On("List", mock.Anything, domain.UserFilter{
		Search:       "",
		Role:         "professional_org",
		DepartmentID: &testDepartment,
		Page:         2,
		PageSize:     25,
	}).Return(users, int64(1), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodGet, "/users?role=professional_org&page=2&page_size=25", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Create_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	user := &domain.User{ID: domain.ID(6), Username: "newuser"}
	active := true
	mockSvc.On("Create", mock.Anything, service.CreateUserRequest{
		Username:  "newuser",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "User",
		IsActive:  &active,

		RequireDepartment: true,
		ScopeDepartmentID: &testDepartment,
	}).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
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
	sameDepartment(c, mockSvc)
	body := `{"password":"short"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Update_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(7)
	user := &domain.User{ID: id, Username: "updated"}
	mockSvc.On("Update", mock.Anything, id, service.UpdateUserRequest{FirstName: "Updated"}).Return(user, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	body := `{"first_name":"Updated"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(id)), bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Update_InvalidUUID(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
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

	id := domain.ID(8)
	mockSvc.On("GetPermissions", mock.Anything, id).Return([]string{"land:read"}, nil)
	mockSvc.On("ChangePassword", mock.Anything, id, "Password123!").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(9))
	c.Set(middleware.CtxPermissions, []string{"users:update", "land:read"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(id))+"/password",
		bytes.NewBufferString(`{"password":"Password123!"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.ChangePassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Backend min=8 — 7 тэмдэгт нууц үг батлагдахгүй (frontend-тэй ижил дүрэм).
func TestUserHandler_ChangePassword_TooShort(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(10)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(id))+"/password",
		bytes.NewBufferString(`{"password":"Short1!"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "ChangePassword", mock.Anything, mock.Anything, mock.Anything)
}

// ── Өөрийгөө хаахаас хамгаалах (self-lockout) ────────────────────────────────

func TestUserHandler_Delete_RejectsSelf(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(11)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, id)
	c.Request = httptest.NewRequest(http.MethodDelete, "/users/"+strconv.Itoa(int(id)), nil)
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Delete(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestUserHandler_Update_RejectsSelfDeactivation(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	id := domain.ID(12)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, id)
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(id)),
		bytes.NewBufferString(`{"is_active":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(id))}}

	h.Update(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

// Бусад хэрэглэгчийг идэвхгүй болгох нь зөвшөөрөгдөнө.
func TestUserHandler_Update_AllowsDeactivatingOthers(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	callerID := domain.ID(13)
	targetID := domain.ID(14)
	inactive := false
	mockSvc.On("Update", mock.Anything, targetID, service.UpdateUserRequest{IsActive: &inactive}).
		Return(&domain.User{ID: targetID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, callerID)
	c.Set(middleware.CtxPermissions, []string{"users:update", "land:read"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(targetID)),
		bytes.NewBufferString(`{"is_active":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(targetID))}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ── Роль олгох / хураах ──────────────────────────────────────────────────────

func TestUserHandler_AssignRole_Success(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	targetID := domain.ID(15)
	roleID := domain.ID(16)
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
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(17))
	c.Set(middleware.CtxPermissions, []string{"land:read", "users:update"})
	body, _ := json.Marshal(map[string]string{"role_id": strconv.Itoa(int(roleID))})
	c.Request = httptest.NewRequest(http.MethodPost, "/users/"+strconv.Itoa(int(targetID))+"/roles", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(targetID))}}

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

	targetID := domain.ID(18)
	roleID := domain.ID(19)
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
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(20))
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	body, _ := json.Marshal(map[string]string{"role_id": strconv.Itoa(int(roleID))})
	c.Request = httptest.NewRequest(http.MethodPost, "/users/"+strconv.Itoa(int(targetID))+"/roles", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(targetID))}}

	h.AssignRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRole.AssertNotCalled(t, "AssignToUser", mock.Anything, mock.Anything, mock.Anything)
}

// Өөртөө роль олгохыг хориглоно (self-escalation).
func TestUserHandler_AssignRole_RejectsSelf(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	selfID := domain.ID(21)
	roleID := domain.ID(22)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, selfID)
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	body, _ := json.Marshal(map[string]string{"role_id": strconv.Itoa(int(roleID))})
	c.Request = httptest.NewRequest(http.MethodPost, "/users/"+strconv.Itoa(int(selfID))+"/roles", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(selfID))}}

	h.AssignRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRole.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	mockRole.AssertNotCalled(t, "AssignToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_RemoveRole_RejectsHigherPrivilegeRole(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	mockRole := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, mockRole)

	targetID := domain.ID(23)
	roleID := domain.ID(24)
	mockRole.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []domain.Permission{{Name: "users:delete"}},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(25))
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodDelete,
		"/users/"+strconv.Itoa(int(targetID))+"/roles/"+strconv.Itoa(int(roleID)), nil)
	c.Params = gin.Params{
		gin.Param{Key: "id", Value: strconv.Itoa(int(targetID))},
		gin.Param{Key: "role_id", Value: strconv.Itoa(int(roleID))},
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

	roleID := domain.ID(26)
	mockRole.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []domain.Permission{{Name: "users:delete"}},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxPermissions, []string{"users:create"})
	body, _ := json.Marshal(map[string]any{
		"username":   "newuser",
		"email":      "new@example.mn",
		"password":   "Password123!",
		"first_name": "New",
		"last_name":  "User",
		"role_ids":   []string{strconv.Itoa(int(roleID))},
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

	adminID := domain.ID(27)
	mockSvc.On("GetPermissions", mock.Anything, adminID).
		Return([]string{"users:update", "users:delete", "roles:update"}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(28))
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(adminID))+"/password",
		bytes.NewBufferString(`{"password":"Password123!"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(adminID))}}

	h.ChangePassword(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "ChangePassword", mock.Anything, mock.Anything, mock.Anything)
}

// Имэйл сольж /auth/forgot-password-оор бүртгэлийг гартаа авах зам хаагдсан.
func TestUserHandler_Update_RejectsHigherPrivilegedTarget(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	adminID := domain.ID(29)
	mockSvc.On("GetByID", mock.Anything, adminID).
		Return(&domain.User{ID: adminID, Email: "admin@example.com"}, nil)
	mockSvc.On("GetPermissions", mock.Anything, adminID).
		Return([]string{"users:update", "users:delete"}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(30))
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(adminID)),
		bytes.NewBufferString(`{"email":"attacker@evil.mn"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(adminID))}}

	h.Update(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

// Өөрийн бүртгэлээ засахад "хамрах" шалгалт хийгдэхгүй (өөрөө өөртэйгөө тэнцүү).
func TestUserHandler_Update_AllowsSelfEdit(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	selfID := domain.ID(31)
	mockSvc.On("Update", mock.Anything, selfID, service.UpdateUserRequest{FirstName: "Мөнх"}).
		Return(&domain.User{ID: selfID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, selfID)
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(selfID)),
		bytes.NewBufferString(`{"first_name":"Мөнх"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(selfID))}}

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

	adminID := domain.ID(32)
	mockSvc.On("Update", mock.Anything, adminID, service.UpdateUserRequest{Position: "Дарга"}).
		Return(&domain.User{ID: adminID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(33))
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(adminID)),
		bytes.NewBufferString(`{"position":"Дарга"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(adminID))}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertNotCalled(t, "GetPermissions", mock.Anything, mock.Anything)
}

// Имэйлийг ижил хэвээр (өөрчлөлтгүй) илгээвэл шалгалт хийгдэхгүй — UI нь бүх
// талбарыг илгээдэг тул энэ нь хэвийн засвар.
func TestUserHandler_Update_UnchangedEmailSkipsRankCheck(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	targetID := domain.ID(34)
	mockSvc.On("GetByID", mock.Anything, targetID).
		Return(&domain.User{ID: targetID, Email: "boss@example.mn"}, nil)
	mockSvc.On("Update", mock.Anything, targetID, service.UpdateUserRequest{
		Email:     "Boss@example.mn", // том/жижиг үсгийн зөрүү — өөрчлөлт биш
		FirstName: "Бат",
	}).Return(&domain.User{ID: targetID}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sameDepartment(c, mockSvc)
	c.Set(middleware.CtxUserID, domain.ID(35))
	c.Set(middleware.CtxPermissions, []string{"users:update"})
	c.Request = httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(targetID)),
		bytes.NewBufferString(`{"email":"Boss@example.mn","first_name":"Бат"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{gin.Param{Key: "id", Value: strconv.Itoa(int(targetID))}}

	h.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertNotCalled(t, "GetPermissions", mock.Anything, mock.Anything)
}

// ── Хэлтсээр хязгаарлах ─────────────────────────────────────────────────

func TestUserHandler_List_SystemAdminSeesAllDepartments(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})
	mockSvc.On("List", mock.Anything, domain.UserFilter{Page: 1, PageSize: 20}).
		Return([]*domain.User{}, int64(0), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxRoles, []string{"admin"})
	c.Set(middleware.CtxDepartmentID, testDepartment)
	c.Request = httptest.NewRequest(http.MethodGet, "/users", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_List_NoDepartmentSeesNobody(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxRoles, []string{"department_head"})
	c.Request = httptest.NewRequest(http.MethodGet, "/users", nil)

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
}

func TestUserHandler_OtherDepartmentTargetIsForbidden(t *testing.T) {
	other := domain.ID(99)
	target := domain.ID(3)
	cases := map[string]func(h *handler.UserHandler, c *gin.Context){
		"get": func(h *handler.UserHandler, c *gin.Context) {
			c.Request = httptest.NewRequest(http.MethodGet, "/users/3", nil)
			h.GetByID(c)
		},
		"update": func(h *handler.UserHandler, c *gin.Context) {
			c.Request = httptest.NewRequest(http.MethodPut, "/users/3", bytes.NewBufferString(`{"first_name":"X"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.Update(c)
		},
		"delete": func(h *handler.UserHandler, c *gin.Context) {
			c.Request = httptest.NewRequest(http.MethodDelete, "/users/3", nil)
			h.Delete(c)
		},
		"password": func(h *handler.UserHandler, c *gin.Context) {
			c.Request = httptest.NewRequest(http.MethodPut, "/users/3/password", bytes.NewBufferString(`{"password":"Password123"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.ChangePassword(c)
		},
		"assign role": func(h *handler.UserHandler, c *gin.Context) {
			c.Request = httptest.NewRequest(http.MethodPost, "/users/3/roles", bytes.NewBufferString(`{"role_id":"5"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.AssignRole(c)
		},
		"remove role": func(h *handler.UserHandler, c *gin.Context) {
			c.Params = append(c.Params, gin.Param{Key: "role_id", Value: "5"})
			c.Request = httptest.NewRequest(http.MethodDelete, "/users/3/roles/5", nil)
			h.RemoveRole(c)
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			mockSvc := &mocks.MockUserService{}
			h := handler.NewUserHandler(mockSvc, &mocks.MockRoleService{})
			mockSvc.On("DepartmentOf", mock.Anything, target).Return(&other, nil)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set(middleware.CtxUserID, domain.ID(1))
			c.Set(middleware.CtxRoles, []string{"department_head"})
			c.Set(middleware.CtxDepartmentID, testDepartment)
			c.Params = gin.Params{gin.Param{Key: "id", Value: "3"}}

			run(h, c)

			assert.Equal(t, http.StatusForbidden, w.Code)
			mockSvc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
			mockSvc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
			mockSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			mockSvc.AssertNotCalled(t, "ChangePassword", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUserHandler_Create_AdminRoleSkipsDepartmentRequirement(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	roleSvc := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, roleSvc)

	adminRole := &domain.Role{ID: domain.ID(1), Name: "admin",
		Permissions: []domain.Permission{{Name: "land:read", Resource: "land"}}}
	roleSvc.On("GetByID", mock.Anything, domain.ID(1)).Return(adminRole, nil)
	roleSvc.On("AssignToUser", mock.Anything, domain.ID(6), domain.ID(1)).Return(nil)
	mockSvc.On("EnsureSystemAdminSlot", mock.Anything, (*domain.ID)(nil)).Return(nil)
	mockSvc.On("Create", mock.Anything, mock.MatchedBy(func(req service.CreateUserRequest) bool {
		return !req.RequireDepartment && req.ScopeDepartmentID == nil
	})).Return(&domain.User{ID: domain.ID(6)}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxRoles, []string{"admin"})
	body := `{"username":"root2","password":"Password123!","role_ids":["1"]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestUserHandler_Create_SystemAdminLimit(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	roleSvc := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, roleSvc)

	adminRole := &domain.Role{ID: domain.ID(1), Name: "admin",
		Permissions: []domain.Permission{{Name: "land:read", Resource: "land"}}}
	roleSvc.On("GetByID", mock.Anything, domain.ID(1)).Return(adminRole, nil)
	mockSvc.On("EnsureSystemAdminSlot", mock.Anything, (*domain.ID)(nil)).Return(domain.ErrSystemAdminLimit)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxRoles, []string{"admin"})
	body := `{"username":"root4","password":"Password123!","role_ids":["1"]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestUserHandler_AssignRole_SystemAdminLimit(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	roleSvc := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, roleSvc)

	target := domain.ID(3)
	adminRole := &domain.Role{ID: domain.ID(1), Name: "admin",
		Permissions: []domain.Permission{{Name: "land:read", Resource: "land"}}}
	roleSvc.On("GetByID", mock.Anything, domain.ID(1)).Return(adminRole, nil)
	mockSvc.On("EnsureSystemAdminSlot", mock.Anything, &target).Return(domain.ErrSystemAdminLimit)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, domain.ID(1))
	c.Set(middleware.CtxRoles, []string{"admin"})
	c.Params = gin.Params{gin.Param{Key: "id", Value: "3"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/users/3/roles", bytes.NewBufferString(`{"role_id":"1"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.AssignRole(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	roleSvc.AssertNotCalled(t, "AssignToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_RemoveRole_AdminFromDepartmentlessUser(t *testing.T) {
	mockSvc := &mocks.MockUserService{}
	roleSvc := &mocks.MockRoleService{}
	h := handler.NewUserHandler(mockSvc, roleSvc)

	target := domain.ID(3)
	adminRole := &domain.Role{ID: domain.ID(1), Name: "admin",
		Permissions: []domain.Permission{{Name: "land:read", Resource: "land"}}}
	roleSvc.On("GetByID", mock.Anything, domain.ID(1)).Return(adminRole, nil)
	mockSvc.On("DepartmentOf", mock.Anything, target).Return(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, domain.ID(1))
	c.Set(middleware.CtxRoles, []string{"admin"})
	c.Params = gin.Params{gin.Param{Key: "id", Value: "3"}, gin.Param{Key: "role_id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/users/3/roles/1", nil)

	h.RemoveRole(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	roleSvc.AssertNotCalled(t, "RemoveFromUser", mock.Anything, mock.Anything, mock.Anything)
}
