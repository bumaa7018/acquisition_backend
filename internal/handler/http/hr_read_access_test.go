package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Зөвхөн дуудагдах аргуудыг хэрэгжүүлсэн fake-ууд (бусад нь nil интерфэйс —
// дуудагдвал panic, өөрөөр хэлбэл тест буруу маршрутаар орсныг илтгэнэ).
type fakeOrgSvc struct{ service.OrganizationService }
type fakeDeptSvc struct{ service.DepartmentService }
type fakePosSvc struct{ service.PositionService }
type fakePersonSvc struct{ service.PersonService }
type fakeEmployeeSvc struct{ service.EmployeeService }

func (fakePersonSvc) List(context.Context, domain.HRFilter) ([]*domain.Person, int64, error) {
	return []*domain.Person{}, 0, nil
}

func (fakeEmployeeSvc) List(context.Context, domain.HRFilter) ([]*domain.Employee, int64, error) {
	return []*domain.Employee{}, 0, nil
}

// Иргэн/хуулийн этгээдийн бүртгэлийг (регистр, утас, хаяг) зөвхөн админ уншина;
// hr:read-тэй хэлтсийн дарга өөрийн хэлтсийн ажилтныг л.
func TestHR_PersonRegistryAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	router := handler.NewRouter(nil, manager, nil, nil, nil, nil, nil, nil, nil, nil,
		service.OrganizationService(fakeOrgSvc{}), service.DepartmentService(fakeDeptSvc{}),
		service.PositionService(fakePosSvc{}), service.PersonService(fakePersonSvc{}),
		service.EmployeeService(fakeEmployeeSvc{}))

	token := func(roles ...string) string {
		pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{
			ID: 61, Roles: roles, Permissions: []string{"hr:read"}, DepartmentID: &testDeptID,
		})
		require.NoError(t, err)
		return pair.AccessToken
	}
	head, admin := token("department_head"), token("admin")

	for _, path := range []string{"/persons", "/organizations", "/positions"} {
		assert.Equal(t, http.StatusForbidden, doGet(router, head, path).Code, "дарга "+path)
	}
	assert.Equal(t, http.StatusOK, doGet(router, admin, "/persons").Code)
	// Өөрийн хэлтсийн ажилтан (хэрэглэгч үүсгэхэд сонгох) хэвээр.
	assert.Equal(t, http.StatusOK, doGet(router, head, "/employees").Code)
}
