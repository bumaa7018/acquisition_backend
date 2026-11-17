package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	jwtpkg "acquisition_backend/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Аудитаар илэрсэн эрхийн цоорхойнуудын регресс тест.
//
// Тус бүр нь ЯГ нэг цоорхойг барина: засварыг буцаавал энд унана.

func authzRouter(t *testing.T, landSvc *mocks.MockLandService) (*gin.Engine, *jwtpkg.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil, landSvc, nil, nil, nil)
	return router, jwtManager
}

func authzToken(t *testing.T, m *jwtpkg.Manager, info jwtpkg.UserInfo) string {
	t.Helper()
	pair, err := m.GenerateTokenPair(info)
	require.NoError(t, err)
	return pair.AccessToken
}

func authzDo(router *gin.Engine, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// B3 — глобал /parcels/:id/* маршрут нь чөлөөлөлтийн замын хуваарилалтын
// хаалтыг тойрч болдог байв.
func TestRouter_GlobalParcelDocuments_RequiresAssignment(t *testing.T) {
	parcelID := uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("IsParcelAcquisitionLocked", mock.Anything, parcelID).Return(false, nil).Maybe()
	landSvc.On("IsAssignedToParcel", mock.Anything, parcelID, domain.ID(77)).Return(false, nil)

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(77),
		Username:    "employee",
		Roles:       []string{"employee"},
		Permissions: []string{"land:read"},
	})

	w := authzDo(router, http.MethodGet, "/api/v1/parcels/"+parcelID.String()+"/documents", token)
	assert.Equal(t, http.StatusForbidden, w.Code,
		"хуваарилагдаагүй мэргэжилтэн глобал замаар нэгж талбарын баримт уншиж болохгүй")
}

// Санхүү/МИКА хяналтын үүрэгтэй тул хуваарилалтгүйгээр УНШИНА (энэ чөлөөлөл
// байхгүй бол нөхөх олговрын таб дахь үнэлгээний тайлан тэдэнд хаагдана).
func TestRouter_GlobalParcelDocuments_AllowsSupervisoryRead(t *testing.T) {
	parcelID := uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("IsParcelAcquisitionLocked", mock.Anything, parcelID).Return(false, nil).Maybe()
	landSvc.On("ListParcelDocuments", mock.Anything, parcelID).
		Return([]*domain.Document{}, nil)
	dept := 5
	landSvc.On("ParcelDepartmentID", mock.Anything, parcelID).Return(&dept, nil).Maybe()

	router, jwtManager := authzRouter(t, landSvc)
	callerDept := domain.ID(5)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:           domain.ID(78),
		Username:     "finance",
		Roles:        []string{"finance_specialist"},
		Permissions:  []string{"compensation:read"},
		DepartmentID: &callerDept,
	})

	w := authzDo(router, http.MethodGet, "/api/v1/parcels/"+parcelID.String()+"/documents", token)
	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

// B4 — DELETE /land-acquisitions/:id нь POST/PUT-ийн адил ролийн хаалтгүй байв:
// `land:delete`-тэй custom роль аль ч чөлөөлөлтийг устгаж чаддаг.
func TestRouter_DeleteAcquisition_RequiresSeniorRole(t *testing.T) {
	acqID := uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAcquisitionLocked", mock.Anything, acqID).Return(false, nil).Maybe()

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(79),
		Username:    "custom",
		Roles:       []string{"custom_role"},
		Permissions: []string{"land:delete"},
	})

	w := authzDo(router, http.MethodDelete, "/api/v1/land-acquisitions/"+acqID.String(), token)
	assert.Equal(t, http.StatusForbidden, w.Code)
	landSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

// B3 — тайлан нь БҮХ чөлөөлөлтийг нэгтгэдэг тул `land:read` дангаараа
// хангалтгүй (хуваарилагдаагүй мэргэжилтэн байгууллага даяарх өгөгдлийг
// Excel-ээр татаж чаддаг байв).
func TestRouter_ReportDownload_RequiresSenior(t *testing.T) {
	landSvc := &mocks.MockLandService{}
	router, jwtManager := authzRouter(t, landSvc)

	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(80),
		Username:    "employee",
		Roles:       []string{"employee"},
		Permissions: []string{"land:read"},
	})
	w := authzDo(router, http.MethodGet, "/api/v1/report/download?page=1", token)
	assert.Equal(t, http.StatusForbidden, w.Code)
	landSvc.AssertNotCalled(t, "ListReportParcels", mock.Anything, mock.Anything)
}

func TestRouter_ReportDownload_AllowsSenior(t *testing.T) {
	landSvc := &mocks.MockLandService{}
	landSvc.On("ListReportParcels", mock.Anything, mock.Anything).
		Return([]*domain.ReportParcelRow{}, int64(0), nil)

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(81),
		Username:    "senior",
		Roles:       []string{"senior_specialist"},
		Permissions: []string{"land:read"},
	})
	w := authzDo(router, http.MethodGet, "/api/v1/report/download?page=1", token)
	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

// F3 — файлын түлхүүрийн эрх. Хуваарилагдаагүй мэргэжилтэн чөлөөлөлтийн ID
// таамаглан баримт татахаас сэргийлнэ.
func TestRouter_FileAuthorize_DeniesUnassigned(t *testing.T) {
	acqID := uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAssigned", mock.Anything, acqID, domain.ID(82)).Return(false, nil)

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(82),
		Username:    "employee",
		Roles:       []string{"employee"},
		Permissions: []string{"land:read"},
	})

	w := authzDo(router, http.MethodGet,
		"/api/v1/files/authorize?key=acquisition%2F"+acqID.String()+"%2F%D0%93%D1%8D%D1%80%D1%8D%D1%8D.pdf", token)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRouter_FileAuthorize_AllowsAssigned(t *testing.T) {
	acqID := uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAssigned", mock.Anything, acqID, domain.ID(83)).Return(true, nil)
	landSvc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&testDeptInt, nil)

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:           domain.ID(83),
		Username:     "employee",
		Roles:        []string{"employee"},
		Permissions:  []string{"land:read"},
		DepartmentID: &testDeptID,
	})

	w := authzDo(router, http.MethodGet,
		"/api/v1/files/authorize?key=acquisition%2F"+acqID.String()+"%2Ffile.pdf", token)
	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

// Түлхүүр гурван хэсэггүй бол (bucket/prefix жагсаах оролдлого) хаана.
func TestRouter_FileAuthorize_RejectsMalformedKey(t *testing.T) {
	landSvc := &mocks.MockLandService{}
	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(84),
		Username:    "senior",
		Roles:       []string{"senior_specialist"},
		Permissions: []string{"land:read"},
	})

	for _, key := range []string{"", "acquisition", "acquisition%2F" + uuid.New().String()} {
		w := authzDo(router, http.MethodGet, "/api/v1/files/authorize?key="+key, token)
		assert.Equal(t, http.StatusForbidden, w.Code, "key=%q", key)
	}
}

// Эрхгүй (land:read ч, compensation:read ч байхгүй) хэрэглэгч файлд хүрэхгүй.
func TestRouter_FileAuthorize_RequiresReadPermission(t *testing.T) {
	landSvc := &mocks.MockLandService{}
	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:          domain.ID(85),
		Username:    "decision",
		Roles:       []string{"decision_registration"},
		Permissions: []string{"decision:read"},
	})

	w := authzDo(router, http.MethodGet,
		"/api/v1/files/authorize?key=acquisition%2F"+uuid.New().String()+"%2Ffile.pdf", token)
	assert.Equal(t, http.StatusForbidden, w.Code)
	landSvc.AssertNotCalled(t, "IsAssigned", mock.Anything, mock.Anything, mock.Anything)
}

// Цуцлалтын PDF (valuation-cancel/<нэгж талбар>/…) нь файлын зөвшөөрөгдсөн
// namespace-ийн жагсаалтад ороогүй тул шалгалт "танихгүй төрөл" гэж 403
// буцаадаг байв — хавсралт нь өөрөө нээгддэггүй болно.
func TestRouter_FileAuthorize_AllowsValuationCancelAttachment(t *testing.T) {
	parcelID := uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAssignedToParcel", mock.Anything, parcelID, domain.ID(86)).Return(true, nil)
	landSvc.On("ParcelDepartmentID", mock.Anything, parcelID).Return(&testDeptInt, nil)

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:           domain.ID(86),
		Username:     "employee",
		Roles:        []string{"employee"},
		Permissions:  []string{"land:read"},
		DepartmentID: &testDeptID,
	})

	w := authzDo(router, http.MethodGet,
		"/api/v1/files/authorize?key=valuation-cancel%2F"+parcelID.String()+"%2Fcancel.pdf", token)
	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

// Үнэлгээ цуцлагдахад олговрын мөр УСТДАГ. Түүхэн snapshot дахь үнэлгээний
// тайлан руу хандахад эзнийг нь архиваас олох ёстой — эс бөгөөс архивын
// тайлан бүр 403 болно.
func TestRouter_FileAuthorize_AllowsArchivedCompensationReport(t *testing.T) {
	compID, acqID := uuid.New(), uuid.New()

	landSvc := &mocks.MockLandService{}
	landSvc.On("GetCompensationAcquisition", mock.Anything, compID).Return(acqID, nil)
	landSvc.On("IsAssigned", mock.Anything, acqID, domain.ID(87)).Return(true, nil)
	landSvc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&testDeptInt, nil)

	router, jwtManager := authzRouter(t, landSvc)
	token := authzToken(t, jwtManager, jwtpkg.UserInfo{
		ID:           domain.ID(87),
		Username:     "employee",
		Roles:        []string{"employee"},
		Permissions:  []string{"land:read"},
		DepartmentID: &testDeptID,
	})

	w := authzDo(router, http.MethodGet,
		"/api/v1/files/authorize?key=compensation-report%2F"+compID.String()+"%2Freport.pdf", token)
	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

// Ахлах/санхүү/МИКА файлын түлхүүр мэдсэн ч ӨӨР хэлтсийн чөлөөлөлт, нэгж
// талбарын файлыг нээхгүй (маршрутын RequireAcquisitionDepartment-тэй ижил).
func TestRouter_FileAuthorize_DeniesOtherDepartment(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	other := 99
	// Санхүү бүх хэлтсийг хардаг тул энд ороогүй (TestFinance_SeesAllDepartments).
	for _, roles := range [][]string{{"senior_specialist"}, {"mika"}} {
		landSvc := &mocks.MockLandService{}
		landSvc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&other, nil)
		landSvc.On("ParcelDepartmentID", mock.Anything, parcelID).Return(&other, nil)
		router, jwtManager := authzRouter(t, landSvc)
		token := authzToken(t, jwtManager, jwtpkg.UserInfo{
			ID: domain.ID(88), Roles: roles,
			Permissions:  []string{"land:read", "compensation:read"},
			DepartmentID: &testDeptID,
		})
		for _, key := range []string{
			"compensation-report%2F" + uuid.New().String() + "%2Fr.pdf",
			"valuation-return%2F" + parcelID.String() + "%2Fr.pdf",
			"acquisition%2F" + acqID.String() + "%2Fa.pdf",
		} {
			landSvc.On("GetCompensationAcquisition", mock.Anything, mock.Anything).Return(acqID, nil).Maybe()
			w := authzDo(router, http.MethodGet, "/api/v1/files/authorize?key="+key, token)
			assert.Equal(t, http.StatusForbidden, w.Code, "%v %s", roles, key)
		}
	}
}
