package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fakeMenuSvc — зөвхөн хэрэглэгдэх аргууд.
type fakeMenuSvc struct {
	service.MenuService
	perms    []domain.Permission
	detached bool
}

func (f *fakeMenuSvc) ListPermissions(context.Context, domain.ID) ([]domain.Permission, error) {
	return f.perms, nil
}

func (f *fakeMenuSvc) Detach(context.Context, domain.ID, domain.ID) error {
	f.detached = true
	return nil
}

func roleToken(t *testing.T, m *jwtpkg.Manager, roles, perms []string) string {
	t.Helper()
	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{ID: 501, Roles: roles, Permissions: perms, DepartmentID: &testDeptID})
	require.NoError(t, err)
	return pair.AccessToken
}

// C5 — цэс салгах (тухайн цэсийн ролийн эрх хамт устдаг) нь хасагдах эрх бүр
// дуудагчид байхыг шаардана; амжилттай бол ролийн эзэмшигчдийн сесс тасарна.
func TestMenuDetach_RequiresHoldingRevokedPermissions(t *testing.T) {
	const roleID, menuID = domain.ID(40), domain.ID(7)
	menuPerms := []domain.Permission{{ID: 1, Name: "users:read"}, {ID: 2, Name: "users:delete"}}
	for _, tc := range []struct {
		name       string
		roles      []string
		perms      []string
		roleName   string
		want       int
		invalidate bool
	}{
		{"дуудагчид байхгүй эрх хасагдана", []string{"department_head"}, []string{"roles:update", "users:read"}, "hr_manager", http.StatusForbidden, false},
		{"админы ролийн эрх", []string{"department_head"}, []string{"roles:update", "users:read", "users:delete"}, "admin", http.StatusForbidden, false},
		{"бүх эрхийг эзэмшдэг", []string{"department_head"}, []string{"roles:update", "users:read", "users:delete"}, "hr_manager", http.StatusNoContent, true},
		{"системийн админ", []string{"admin"}, nil, "hr_manager", http.StatusNoContent, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
			roleSvc := &mocks.MockRoleService{}
			roleSvc.On("GetByID", mock.Anything, roleID).Return(&domain.Role{
				ID: roleID, Name: tc.roleName, Permissions: menuPerms,
			}, nil).Maybe()
			userSvc := &mocks.MockUserService{}
			userSvc.On("InvalidateRoleSessions", mock.Anything, roleID).Return(nil).Maybe()
			menu := &fakeMenuSvc{perms: menuPerms}
			router := handler.NewRouter(nil, manager, nil, userSvc, roleSvc, nil, nil, nil, nil, nil,
				service.MenuService(menu))

			req := httptest.NewRequest(http.MethodDelete,
				"/api/v1/roles/"+strconv.Itoa(int(roleID))+"/menus/"+strconv.Itoa(int(menuID)), nil)
			req.Header.Set("Authorization", "Bearer "+roleToken(t, manager, tc.roles, tc.perms))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.want, w.Code, w.Body.String())
			assert.Equal(t, tc.want == http.StatusNoContent, menu.detached)
			if tc.invalidate {
				userSvc.AssertCalled(t, "InvalidateRoleSessions", mock.Anything, roleID)
			} else {
				userSvc.AssertNotCalled(t, "InvalidateRoleSessions", mock.Anything, mock.Anything)
			}
		})
	}
}

// C5 — ролоос эрх хасахад эзэмшигчдийн хуучин токен хүчингүй болно.
func TestRoleRemovePermission_InvalidatesHolderSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	const roleID, permID = domain.ID(41), domain.ID(3)
	roleSvc := &mocks.MockRoleService{}
	roleSvc.On("GetByID", mock.Anything, roleID).Return(&domain.Role{ID: roleID, Name: "x"}, nil)
	roleSvc.On("RemovePermission", mock.Anything, roleID, permID).Return(nil)
	permSvc := &mocks.MockPermissionService{}
	permSvc.On("GetByID", mock.Anything, permID).Return(&domain.Permission{ID: permID, Name: "land:read", Resource: "land"}, nil)
	userSvc := &mocks.MockUserService{}
	userSvc.On("InvalidateRoleSessions", mock.Anything, roleID).Return(nil)
	router := handler.NewRouter(nil, manager, nil, userSvc, roleSvc, permSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/41/permissions/3", nil)
	req.Header.Set("Authorization", "Bearer "+roleToken(t, manager, []string{"admin"}, nil))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	userSvc.AssertExpectations(t)
}

// C3 — олговор батлах/татгалзахыг ЗӨВХӨН санхүү; админ автоматаар нэвтрэхгүй.
func TestCompensationApprove_FinanceOnly(t *testing.T) {
	acqID, compID := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name  string
		roles []string
		want  int
	}{
		{"админ", []string{"admin"}, http.StatusForbidden},
		{"ахлах", []string{"senior_specialist"}, http.StatusForbidden},
		{"санхүү", []string{"finance_specialist"}, http.StatusOK},
	} {
		for _, action := range []string{"approve", "reject"} {
			t.Run(tc.name+"/"+action, func(t *testing.T) {
				router, svc, token := reviewerRouter(t, tc.roles, []string{"land:update", "compensation:update"})
				svc.On("ApproveCompensation", mock.Anything, acqID, compID, mock.Anything, mock.Anything).Return(nil).Maybe()
				svc.On("RejectCompensation", mock.Anything, acqID, compID, mock.Anything, mock.Anything).Return(nil).Maybe()
				req := httptest.NewRequest(http.MethodPost,
					"/api/v1/land-acquisitions/"+acqID.String()+"/compensations/"+compID.String()+"/"+action, nil)
				req.Header.Set("Authorization", "Bearer "+token)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				assert.Equal(t, tc.want, w.Code, w.Body.String())
			})
		}
	}
}
