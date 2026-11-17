package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// Санхүү/МИКА — хяналтын гадаад роль. Backend өөрөө тэдэнд зөвхөн үнэлгээтэй
// холбоотой өгөгдлийг өгөх ёстой (frontend-ийн нуултаас үл хамааран).

const (
	docTypeReport = 11
	docTypeIDCard = 12
)

func reviewerRouter(t *testing.T, roles []string, perms []string) (*gin.Engine, *mocks.MockLandService, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{
		ID: 55, Roles: roles, Permissions: perms, DepartmentID: &testDeptID,
	})
	require.NoError(t, err)
	svc := &mocks.MockLandService{}
	svc.On("IsAcquisitionLocked", mock.Anything, mock.Anything).Return(false, nil).Maybe()
	svc.On("IsParcelAcquisitionLocked", mock.Anything, mock.Anything).Return(false, nil).Maybe()
	svc.On("AcquisitionDepartmentID", mock.Anything, mock.Anything).Return(&testDeptInt, nil).Maybe()
	svc.On("ParcelDepartmentID", mock.Anything, mock.Anything).Return(&testDeptInt, nil).Maybe()
	svc.On("GetDocumentType", mock.Anything, docTypeReport).Return(&domain.DocumentType{ID: docTypeReport, Type: "valuation_report"}, nil).Maybe()
	svc.On("GetDocumentType", mock.Anything, docTypeIDCard).Return(&domain.DocumentType{ID: docTypeIDCard, Type: "id_card"}, nil).Maybe()
	return handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil), svc, pair.AccessToken
}

func doGet(router *gin.Engine, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1"+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

var (
	financeRoles = []string{"finance_specialist"}
	financePerms = []string{"compensation:read", "compensation:update"}
)

func TestReviewer_ParcelDetailHidesHolderData(t *testing.T) {
	for _, tc := range []struct {
		name     string
		roles    []string
		perms    []string
		redacted bool
	}{
		{"санхүү", financeRoles, financePerms, true},
		{"МИКА", []string{"mika"}, []string{"compensation:read"}, true},
		{"ахлах мэргэжилтэн", []string{"senior_specialist"}, []string{"land:read"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, svc, token := reviewerRouter(t, tc.roles, tc.perms)
			acqID, parcelID := uuid.New(), uuid.New()
			svc.On("GetParcelDetail", mock.Anything, acqID, parcelID).Return(&domain.ParcelFull{
				Parcel:  domain.Parcel{ID: parcelID, AcquisitionID: acqID, ParcelID: "P1"},
				Detail:  &domain.ParcelDetailInfo{HolderName: "Бат", HolderRegisterNo: "УБ99999999", HolderPhone: "99119911"},
				Holders: []domain.ParcelHolder{{ID: uuid.New(), Name: "Бат", RegisterNo: "УБ99999999", Phone: "99119911"}},
			}, nil)

			w := doGet(router, token, "/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String())
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			body := w.Body.String()
			if tc.redacted {
				assert.NotContains(t, body, "УБ99999999")
				assert.NotContains(t, body, "99119911")
				var resp struct {
					Data struct {
						Holders  []json.RawMessage `json:"holders"`
						ParcelID string            `json:"parcel_id"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Empty(t, resp.Data.Holders)
				assert.Equal(t, "P1", resp.Data.ParcelID)
			} else {
				assert.Contains(t, body, "УБ99999999")
			}
		})
	}
}

func parcelDocs(parcelID uuid.UUID) (report, idCard *domain.Document) {
	report = &domain.Document{ID: uuid.New(), Name: "report.pdf", DocumentTypeID: docTypeReport,
		FileURL: "/api/files/parcel/" + parcelID.String() + "/report.pdf"}
	idCard = &domain.Document{ID: uuid.New(), Name: "id.pdf", DocumentTypeID: docTypeIDCard,
		FileURL: "/api/files/parcel/" + parcelID.String() + "/%D0%B8%D0%B4.pdf"}
	return
}

func TestReviewer_ParcelDocumentsOnlyValuationTypes(t *testing.T) {
	router, svc, token := reviewerRouter(t, financeRoles, financePerms)
	parcelID := uuid.New()
	report, idCard := parcelDocs(parcelID)
	svc.On("ListParcelDocuments", mock.Anything, parcelID).Return([]*domain.Document{report, idCard}, nil)

	w := doGet(router, token, "/parcels/"+parcelID.String()+"/documents")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, report.ID.String(), resp.Data[0].ID)

	// Үнэлгээний биш хавсралтын файлыг ID-аар ч татахгүй.
	w = doGet(router, token, "/parcels/"+parcelID.String()+"/documents/"+idCard.ID.String()+"/file")
	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "OpenParcelDocumentFile", mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewer_FileAuthorize(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	report, idCard := parcelDocs(parcelID)
	for _, tc := range []struct {
		name, key string
		want      int
	}{
		{"нийгмийн судалгаа хаалттай", "socio-survey/" + acqID.String() + "/survey.pdf", http.StatusForbidden},
		{"чөлөөлөлтийн баримт хаалттай", "acquisition/" + acqID.String() + "/x.pdf", http.StatusForbidden},
		{"мэдэгдэл хаалттай", "notice/" + parcelID.String() + "/n.pdf", http.StatusForbidden},
		{"дроны зураг хаалттай", "drone/" + acqID.String() + "/a.tif", http.StatusForbidden},
		{"иргэний үнэмлэх хаалттай", "parcel/" + parcelID.String() + "/ид.pdf", http.StatusForbidden},
		{"бүртгэлгүй файл хаалттай", "parcel/" + parcelID.String() + "/unknown.pdf", http.StatusForbidden},
		{"үнэлгээний тайлан нээлттэй", "parcel/" + parcelID.String() + "/report.pdf", http.StatusOK},
		{"архивын тайлан нээлттэй", "parcel/" + parcelID.String() + "/old-report.pdf", http.StatusOK},
		{"буцаалтын хавсралт нээлттэй", "valuation-return/" + parcelID.String() + "/r.pdf", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, svc, token := reviewerRouter(t, financeRoles, financePerms)
			svc.On("ListParcelDocuments", mock.Anything, parcelID).Return([]*domain.Document{report, idCard}, nil).Maybe()
			svc.On("ArchivedValuationFileURLs", mock.Anything, parcelID).
				Return([]string{"/api/files/parcel/" + parcelID.String() + "/old-report.pdf"}, nil).Maybe()

			w := doGet(router, token, "/files/authorize?key="+tc.key)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

// Дотоод ажилтан (ахлах) файлын эрх өөрчлөгдөөгүй — бүх төрлийн хавсралт.
func TestSenior_FileAuthorizeUnchanged(t *testing.T) {
	router, _, token := reviewerRouter(t, []string{"senior_specialist"}, []string{"land:read"})
	w := doGet(router, token, "/files/authorize?key=socio-survey/"+uuid.New().String()+"/survey.pdf")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// B7/B8 — `land`/`parcels` бүлгийн гадна байсан маршрутууд хэлтсээр хумигдана.
func TestRouter_OutOfGroupRoutesRequireDepartment(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	other := 99
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/parcels/" + parcelID.String() + "/survey"},
		{http.MethodPut, "/parcels/" + parcelID.String() + "/survey"},
		{http.MethodDelete, "/parcels/" + parcelID.String() + "/survey"},
		{http.MethodGet, "/land-acquisitions/" + acqID.String() + "/parcels/" + parcelID.String() + "/status-history"},
		{http.MethodGet, "/land-acquisitions/" + acqID.String() + "/parcels/" + parcelID.String() + "/available-statuses"},
		{http.MethodGet, "/land-acquisitions/" + acqID.String() + "/parcels/" + parcelID.String() + "/status-requirements"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
			pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{
				ID: 70, Roles: []string{"senior_specialist"},
				Permissions: []string{"land:read", "land:update"}, DepartmentID: &testDeptID,
			})
			require.NoError(t, err)
			svc := &mocks.MockLandService{}
			svc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&other, nil).Maybe()
			svc.On("ParcelDepartmentID", mock.Anything, parcelID).Return(&other, nil).Maybe()
			router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil,
				service.ParcelSurveyService(fakeSurveySvc{}))
			req := httptest.NewRequest(tc.method, "/api/v1"+tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		})
	}
}

type fakeSurveySvc struct{ service.ParcelSurveyService }

// B1/B2 — хяналтын самбар, санхүүгийн самбар, тайлан нь дуудагчийн хэлтсээр
// хумигдана; системийн админ бүх хэлтсийг (nil) харна; хэлтэсгүй бол -1 (хоосон).
func TestAggregates_ScopedByDepartment(t *testing.T) {
	five := testDeptInt
	for _, tc := range []struct {
		name  string
		roles []string
		dept  *domain.ID
		want  *int
	}{
		{"ахлах — өөрийн хэлтэс", []string{"senior_specialist"}, &testDeptID, &five},
		// Санхүү БҮХ хэлтсийг харна (хэрэглэгчийн шийдвэр) — хэлтэстэй ч nil.
		{"санхүү — бүх хэлтэс", []string{"finance_specialist"}, &testDeptID, nil},
		{"санхүү хэлтэсгүй ч — бүх хэлтэс", []string{"finance_specialist"}, nil, nil},
		{"хэлтэсгүй — юу ч биш", []string{"senior_specialist"}, nil, func() *int { v := -1; return &v }()},
		{"системийн админ — бүгд", []string{"admin"}, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			manager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
			pair, err := manager.GenerateTokenPair(jwtpkg.UserInfo{
				ID: 71, Roles: tc.roles, DepartmentID: tc.dept,
				Permissions: []string{"land:read", "compensation:read"},
			})
			require.NoError(t, err)
			same := func(got *int) bool {
				if tc.want == nil || got == nil {
					return tc.want == nil && got == nil
				}
				return *got == *tc.want
			}
			svc := &mocks.MockLandService{}
			svc.On("GetDashboard", mock.Anything, mock.MatchedBy(func(f domain.DashboardFilter) bool { return same(f.DepartmentID) })).
				Return(&domain.DashboardSummary{}, nil).Maybe()
			svc.On("GetFinanceDashboard", mock.Anything, mock.MatchedBy(func(f domain.FinanceDashboardFilter) bool { return same(f.DepartmentID) })).
				Return(&domain.FinanceDashboard{}, nil).Maybe()
			svc.On("GetReportSummary", mock.Anything, mock.MatchedBy(func(f domain.ReportParcelFilter) bool { return same(f.DepartmentID) })).
				Return(&domain.ReportSummary{}, nil).Maybe()
			router := handler.NewRouter(nil, manager, nil, nil, nil, nil, svc, nil, nil, nil)

			paths := []string{"/dashboard/finance"}
			if tc.roles[0] != "finance_specialist" {
				paths = append(paths, "/dashboard", "/report/summary")
			}
			for _, p := range paths {
				w := doGet(router, pair.AccessToken, p)
				assert.Equal(t, http.StatusOK, w.Code, p+" "+w.Body.String())
			}
		})
	}
}

// Санхүүгийн мэргэжилтэн БҮХ хэлтсийн чөлөөлөлт, нэгж талбарыг харна; ахлах
// роль ХАМТ эзэмшвэл энэ чөлөөлөл үйлчлэхгүй (ахлахын бичих эрх тархахгүй).
func TestFinance_SeesAllDepartments(t *testing.T) {
	other := 99
	acqID, parcelID := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name  string
		roles []string
		want  int
	}{
		{"санхүү — өөр хэлтсийн чөлөөлөлт", []string{"finance_specialist"}, http.StatusOK},
		{"МИКА — өөр хэлтэс хаалттай", []string{"mika"}, http.StatusForbidden},
		{"санхүү + ахлах — өөр хэлтэс хаалттай", []string{"finance_specialist", "senior_specialist"}, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, svc, token := reviewerRouter(t, tc.roles, []string{"land:read", "compensation:read"})
			// reviewerRouter-ийн хэлтсийн mock-ийг өөр хэлтсээр дарна.
			svc.ExpectedCalls = filterCalls(svc.ExpectedCalls, "AcquisitionDepartmentID", "ParcelDepartmentID")
			svc.On("AcquisitionDepartmentID", mock.Anything, acqID).Return(&other, nil).Maybe()
			svc.On("ParcelDepartmentID", mock.Anything, parcelID).Return(&other, nil).Maybe()
			svc.On("GetParcelDetail", mock.Anything, acqID, parcelID).Return(&domain.ParcelFull{
				Parcel: domain.Parcel{ID: parcelID, AcquisitionID: acqID, ParcelID: "P1"},
			}, nil).Maybe()
			w := doGet(router, token, "/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String())
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func filterCalls(calls []*mock.Call, drop ...string) []*mock.Call {
	out := calls[:0]
	for _, c := range calls {
		skip := false
		for _, d := range drop {
			if c.Method == d {
				skip = true
			}
		}
		if !skip {
			out = append(out, c)
		}
	}
	return out
}
