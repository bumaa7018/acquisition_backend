package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestRouter_AcquisitionCategoriesList_AllowsAuthenticatedUserWithoutPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(1),
		Username:    "employee",
		Permissions: []string{},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	categories := []*domain.AcquisitionCategory{{ID: 1, Name: "Test", SortOrder: 1}}
	landSvc.On("ListAcquisitionCategories", mock.Anything, (*int)(nil)).Return(categories, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/acquisition-categories", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []domain.AcquisitionCategory `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "Test", body.Data[0].Name)
	landSvc.AssertExpectations(t)
}

func TestRouter_ProfLandAcquisitions_AllowsProfessionalOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := domain.ID(501)
	orgID := domain.ID(9001)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       userID,
		OrgID:    &orgID,
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	acq := &domain.LandAcquisition{ID: uuid.New(), PlanCode: "LA-2024-001"}
	landSvc.On("List", mock.Anything, mock.MatchedBy(func(filter domain.LandAcquisitionFilter) bool {
		return filter.ProfessionalOrgID == strconv.Itoa(int(orgID)) &&
			filter.Page == 1 &&
			filter.PageSize == 20
	})).Return([]*domain.LandAcquisition{acq}, int64(1), nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prof/land-acquisitions?page=1&page_size=20", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

func TestRouter_Dashboard_ScopesProfessionalOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := domain.ID(502)
	orgID := domain.ID(9002)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       userID,
		OrgID:    &orgID,
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	landSvc.On("GetDashboard", mock.Anything, mock.MatchedBy(func(filter domain.DashboardFilter) bool {
		return filter.ProfessionalOrgID == strconv.Itoa(int(orgID)) &&
			filter.FilterByParcelStatus &&
			filter.ParcelStatusID == domain.ParcelStatusEvaluating &&
			filter.Status == 0
	})).Return(&domain.DashboardSummary{
		ValuationStatuses: []domain.ValuationStatusStat{{Status: "draft", Count: 2}},
		ValuationTypes:    []domain.ValuationTypeStat{{ValuationType: "asset", Count: 2, Amount: 5000000}},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body.Data, "valuation_statuses")
	assert.Contains(t, body.Data, "valuation_types")
	landSvc.AssertExpectations(t)
}

func TestRouter_Dashboard_ScopesFinanceSpecialist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(3),
		Username:    "finance",
		Roles:       []string{"finance_specialist"},
		Permissions: []string{"compensation:read"},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	landSvc.On("GetDashboard", mock.Anything, mock.MatchedBy(func(filter domain.DashboardFilter) bool {
		return filter.ProfessionalOrgID == "" &&
			filter.FilterByParcelStatus &&
			filter.ParcelStatusID == domain.ParcelStatusEvaluating &&
			filter.Status == domain.LandAcquisitionStatusField
	})).Return(&domain.DashboardSummary{
		ValuationStatuses: []domain.ValuationStatusStat{{Status: "submitted", Count: 3}},
		ValuationTypes:    []domain.ValuationTypeStat{{ValuationType: "asset", Count: 3, Amount: 7000000}},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body.Data, "valuation_statuses")
	assert.Contains(t, body.Data, "valuation_types")
	landSvc.AssertExpectations(t)
}

func TestRouter_Dashboard_AdminKeepsLegacyShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(4),
		Username:    "admin",
		Roles:       []string{"admin"},
		Permissions: []string{"land:read"},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	landSvc.On("GetDashboard", mock.Anything, mock.MatchedBy(func(filter domain.DashboardFilter) bool {
		return !filter.FilterByParcelStatus &&
			filter.ParcelStatusID == 0 &&
			filter.Status == 0 &&
			filter.ProfessionalOrgID == ""
	})).Return(&domain.DashboardSummary{
		ValuationStatuses: []domain.ValuationStatusStat{{Status: "approved", Count: 5}},
		ValuationTypes:    []domain.ValuationTypeStat{{ValuationType: "asset", Count: 5, Amount: 12000000}},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotContains(t, body.Data, "valuation_statuses")
	assert.NotContains(t, body.Data, "valuation_types")
	landSvc.AssertExpectations(t)
}

func TestRouter_Dashboard_ScopesEachUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type dashboardCase struct {
		name  string
		roles []string
		// valuationOrg — үнэлгээний байгууллагын ажилтан (роль/эрхгүй).
		valuationOrg          bool
		permissions           []string
		wantProfessionalScope bool
		wantReviewerScope     bool
		wantLegacyShape       bool
		wantForbidden         bool
	}

	cases := []dashboardCase{
		{
			name:            "admin",
			roles:           []string{"admin"},
			permissions:     []string{"land:read"},
			wantLegacyShape: true,
		},
		{
			name:            "senior specialist",
			roles:           []string{"senior_specialist"},
			permissions:     []string{"land:read"},
			wantLegacyShape: true,
		},
		{
			// Роль/эрхгүй — зөвхөн токены `valuation_org` тэмдэг.
			name:                  "valuation org employee",
			valuationOrg:          true,
			wantProfessionalScope: true,
		},
		{
			name:              "finance specialist",
			roles:             []string{"finance_specialist"},
			permissions:       []string{},
			wantReviewerScope: true,
		},
		{
			name:              "mika",
			roles:             []string{"mika"},
			permissions:       []string{},
			wantReviewerScope: true,
		},
		{
			name:          "employee without land read",
			roles:         []string{"employee"},
			permissions:   []string{},
			wantForbidden: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
			userID := domain.ID(503)
			// Байгууллагын харьяалал нь хэрэглэгчийн ID-аас ТУСДАА утга —
			// хоёрыг андуурсан тохиолдолд тест барих ёстой.
			orgID := domain.ID(9003)
			tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
				ID:           userID,
				OrgID:        &orgID,
				Username:     tc.name,
				ValuationOrg: tc.valuationOrg,
				Roles:        tc.roles,
				Permissions:  tc.permissions,
			})
			require.NoError(t, err)

			landSvc := &mocks.MockLandService{}
			acquisitionID := uuid.New()
			if !tc.wantForbidden {
				landSvc.On("GetDashboard", mock.Anything, mock.MatchedBy(func(filter domain.DashboardFilter) bool {
					if filter.PlanCode != "PLAN-2024" ||
						filter.Years == nil ||
						len(filter.Years) != 2 ||
						filter.Years[0] != 2024 ||
						filter.Years[1] != 2025 ||
						filter.GeneralCategoryID != 10 ||
						filter.SubCategoryID != 20 ||
						filter.AssignedUserID != "assignee-1" {
						return false
					}

					if tc.wantProfessionalScope {
						return filter.ProfessionalOrgID == strconv.Itoa(int(orgID)) &&
							filter.FilterByParcelStatus &&
							filter.ParcelStatusID == domain.ParcelStatusEvaluating &&
							filter.Status == 0
					}
					if tc.wantReviewerScope {
						return filter.ProfessionalOrgID == "" &&
							filter.FilterByParcelStatus &&
							filter.ParcelStatusID == domain.ParcelStatusEvaluating &&
							filter.Status == domain.LandAcquisitionStatusField
					}
					return filter.ProfessionalOrgID == "" &&
						!filter.FilterByParcelStatus &&
						filter.ParcelStatusID == 0 &&
						filter.Status == 0
				})).Return(&domain.DashboardSummary{
					Acquisitions: []*domain.LandAcquisition{
						{
							ID:              acquisitionID,
							AcquisitionName: "Dashboard acquisition",
							PlanCode:        "PLAN-2024",
							Status:          domain.LandAcquisitionStatusField,
							ParcelCount:     7,
						},
					},
					TotalParcels:      7,
					FreedParcels:      2,
					FreedAreaM2:       1234.5,
					PlanAreaM2:        9876.5,
					TotalOrders:       1,
					TotalCompensation: 1500000,
					StatusBreakdown: []domain.ParcelStatusStat{
						{StatusID: domain.ParcelStatusEvaluating, Name: "Evaluating", Count: 5, AreaM2: 321.5},
					},
					Timeline: []domain.TimelinePoint{
						{Date: "2026-07-01", Count: 4},
					},
					ValuationStatuses: []domain.ValuationStatusStat{{Status: "submitted", Count: 1}},
					ValuationTypes:    []domain.ValuationTypeStat{{ValuationType: "asset", Count: 1, Amount: 1500000}},
					FilteredAU1Codes:  []string{"11"},
					FilteredAU2Codes:  []string{"1101"},
					FilteredAU3Codes:  []string{"110101"},
				}, nil)
			}

			router := handler.NewRouter(
				nil,
				jwtManager,
				nil,
				nil,
				nil,
				nil,
				landSvc,
				nil,
				nil,
				nil,
				nil,
			)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard?plan_code=PLAN-2024&year=2024&year=bad&year=2025&general_category_id=10&sub_category_id=20&assigned_user_id=assignee-1", nil)
			req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.wantForbidden {
				assert.Equal(t, http.StatusForbidden, w.Code)
				landSvc.AssertNotCalled(t, "GetDashboard", mock.Anything, mock.Anything)
				return
			}

			assert.Equal(t, http.StatusOK, w.Code)
			var body struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Contains(t, body.Data, "acquisitions")
			assert.Equal(t, float64(7), body.Data["total_parcels"])
			assert.Equal(t, float64(2), body.Data["freed_parcels"])
			assert.Equal(t, 1234.5, body.Data["freed_area_m2"])
			assert.Equal(t, 9876.5, body.Data["plan_area_m2"])
			assert.Equal(t, float64(1), body.Data["total_orders"])
			assert.Equal(t, float64(1500000), body.Data["total_compensation"])

			acquisitions, ok := body.Data["acquisitions"].([]any)
			require.True(t, ok)
			require.Len(t, acquisitions, 1)
			acq, ok := acquisitions[0].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, acquisitionID.String(), acq["id"])
			assert.Equal(t, "Dashboard acquisition", acq["acquisition_name"])
			assert.Equal(t, "PLAN-2024", acq["plan_code"])
			assert.Equal(t, float64(domain.LandAcquisitionStatusField), acq["status"])
			assert.Equal(t, float64(7), acq["parcel_count"])

			statusBreakdown, ok := body.Data["status_breakdown"].([]any)
			require.True(t, ok)
			require.Len(t, statusBreakdown, 1)
			statusStat, ok := statusBreakdown[0].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, float64(domain.ParcelStatusEvaluating), statusStat["status_id"])
			assert.Equal(t, "Evaluating", statusStat["name"])
			assert.Equal(t, float64(5), statusStat["count"])
			assert.Equal(t, 321.5, statusStat["area_m2"])

			timeline, ok := body.Data["timeline"].([]any)
			require.True(t, ok)
			require.Len(t, timeline, 1)
			point, ok := timeline[0].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "2026-07-01", point["date"])
			assert.Equal(t, float64(4), point["count"])

			assert.Equal(t, []any{"11"}, body.Data["filtered_au1_codes"])
			assert.Equal(t, []any{"1101"}, body.Data["filtered_au2_codes"])
			assert.Equal(t, []any{"110101"}, body.Data["filtered_au3_codes"])
			if tc.wantLegacyShape {
				assert.NotContains(t, body.Data, "valuation_statuses")
				assert.NotContains(t, body.Data, "valuation_types")
			} else {
				assert.Contains(t, body.Data, "valuation_statuses")
				assert.Contains(t, body.Data, "valuation_types")
				valuationStatuses, ok := body.Data["valuation_statuses"].([]any)
				require.True(t, ok)
				require.Len(t, valuationStatuses, 1)
				valuationStatus, ok := valuationStatuses[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "submitted", valuationStatus["status"])
				assert.Equal(t, float64(1), valuationStatus["count"])

				valuationTypes, ok := body.Data["valuation_types"].([]any)
				require.True(t, ok)
				require.Len(t, valuationTypes, 1)
				valuationType, ok := valuationTypes[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "asset", valuationType["valuation_type"])
				assert.Equal(t, float64(1), valuationType["count"])
				assert.Equal(t, float64(1500000), valuationType["amount"])
			}
			landSvc.AssertExpectations(t)
		})
	}
}

func TestRouter_ProfLandAcquisitions_DeniesAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(5),
		Username:    "admin",
		Roles:       []string{"admin"},
		Permissions: []string{"land:read"},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prof/land-acquisitions", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	landSvc.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
}

// Дотоод чөлөөлөлтийн маршрут нь үнэлгээний байгууллагад ХААЛТТАЙ: хүсэлт
// хуваарилалтын шалгалт хүрэхээсээ ӨМНӨ allowlist-ээр (ValuationOrgScope)
// таслагдана. Ингэснээр /prof-ээс гадуурх ямар ч зам тэдэнд нээгдэхгүй.
func TestRouter_GeneralLandAcquisition_DeniesValuationOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := domain.ID(504)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       userID,
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	acqID := uuid.New()
	landSvc := &mocks.MockLandService{}

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/"+acqID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	// Handler-ийн шалгалт ХҮРЭХГҮЙ — хүсэлт allowlist дээр таслагдсан.
	landSvc.AssertNotCalled(t, "IsAssigned", mock.Anything, mock.Anything, mock.Anything)
	landSvc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestRouter_GeneralLandAcquisitionList_DeniesProfessionalOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       domain.ID(6),
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions?status=1&page=1&page_size=15", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	landSvc.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
}

func TestRouter_ProfParcelDocuments_DeniesProfessionalOrgWhenParcelNotMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := domain.ID(505)
	orgID := domain.ID(9005)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       userID,
		OrgID:    &orgID,
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	parcelID := uuid.New()
	landSvc := &mocks.MockLandService{}
	landSvc.On("IsParcelAcquisitionLocked", mock.Anything, parcelID).Return(false, nil)
	landSvc.On("IsProfMemberOfParcel", mock.Anything, parcelID, orgID).Return(false, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prof/parcels/"+parcelID.String()+"/documents", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	landSvc.AssertExpectations(t)
	landSvc.AssertNotCalled(t, "ListParcelDocuments", mock.Anything, mock.Anything)
}

// /prof-ийн нэгж талбарын жагсаалт: "Хөндлөнгийн үнэлгээ" төлөвөөр шүүгдэхээс
// гадна БАЙГУУЛЛАГЫН хүрээгээр хязгаарлагдана.
func TestRouter_ProfLandAcquisitionParcels_ReturnsOnlyEvaluationStatusScopedToOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := domain.ID(506)
	orgID := domain.ID(9006)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       userID,
		OrgID:    &orgID,
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	acqID := uuid.New()
	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAcquisitionLocked", mock.Anything, acqID).Return(false, nil)
	landSvc.On("IsProfMemberOfAcquisition", mock.Anything, acqID, orgID).Return(true, nil)
	landSvc.On("GetParcels", mock.Anything, acqID, mock.MatchedBy(func(filter domain.ParcelFilter) bool {
		// Байгууллагын хүрээ ЗААВАЛ тавигдана: хөндлөнгийн үнэлгээгээр нэг
		// нэгж талбарт холбогдсон байгууллага бусдын нэгж талбарыг харахгүй.
		return filter.StatusID == domain.ParcelStatusEvaluating &&
			filter.ProfessionalOrgID == strconv.Itoa(int(orgID)) &&
			filter.Page == 1 &&
			filter.PageSize == 20
	})).Return([]*domain.Parcel{
		{ID: uuid.New(), ParcelID: "010101000001", StatusID: domain.ParcelStatusEvaluating},
	}, int64(1), nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/prof/land-acquisitions/"+acqID.String()+"/parcels?status_id=5", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

func TestRouter_ParcelStatuses_AllowsProfessionalOrgRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       domain.ID(8),
		Username: "professional_primary",
		// Үнэлгээний байгууллагын ажилтан — РОЛЬ/ЭРХГҮЙ, зөвхөн
		// `valuation_org` тэмдэгтэй.
		ValuationOrg: true,
	})
	require.NoError(t, err)

	parcelStatusSvc := &mocks.MockParcelStatusService{}
	parcelStatusSvc.On("List", mock.Anything).Return([]*domain.ParcelStatus{
		{ID: 1, Code: "new", Name: "Шинэ"},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		&mocks.MockLandService{},
		nil,
		nil,
		parcelStatusSvc,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/parcel-statuses", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	parcelStatusSvc.AssertExpectations(t)
}

func TestRouter_ParcelStatuses_AllowsAuthenticatedUserWithoutPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(9),
		Username:    "admin",
		Roles:       []string{"admin"},
		Permissions: []string{},
	})
	require.NoError(t, err)

	parcelStatusSvc := &mocks.MockParcelStatusService{}
	parcelStatusSvc.On("List", mock.Anything).Return([]*domain.ParcelStatus{
		{ID: 1, Code: "new", Name: "Шинэ"},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		&mocks.MockLandService{},
		nil,
		nil,
		parcelStatusSvc,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/parcel-statuses", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	parcelStatusSvc.AssertExpectations(t)
}

func TestRouter_DocumentTypes_AllowsAuthenticatedUserWithoutPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(10),
		Username:    "admin",
		Roles:       []string{"admin"},
		Permissions: []string{},
	})
	require.NoError(t, err)

	documentTypeSvc := &mocks.MockDocumentTypeService{}
	documentTypeSvc.On("List", mock.Anything, "acquisition").Return([]*domain.DocumentType{
		{ID: 1, Name: "Хавсралт", Target: "acquisition"},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		&mocks.MockLandService{},
		nil,
		nil,
		nil,
		documentTypeSvc,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/document-types?target=acquisition", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	documentTypeSvc.AssertExpectations(t)
}

func TestRouter_AssetSpecTypes_AllowsAuthenticatedUserWithoutPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(11),
		Username:    "admin",
		Roles:       []string{"admin"},
		Permissions: []string{},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	landSvc.On("ListAssetSpecTypes", mock.Anything).Return([]*domain.AssetSpecType{
		{ID: 1, Code: "construction_type", Name: "Хийцийн төрөл", SortOrder: 1},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/asset-spec-types", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

func TestRouter_AssetCalcTypes_AllowsAuthenticatedUserWithoutPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(12),
		Username:    "admin",
		Roles:       []string{"admin"},
		Permissions: []string{},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	landSvc.On("ListAssetCalcTypes", mock.Anything).Return([]*domain.AssetCalcType{
		{ID: 1, Code: "unit_cost", Name: "Барилгын нэгжийн өртөг", DefaultUnit: "төгрөг/м²", SortOrder: 1},
	}, nil)

	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		landSvc,
		nil,
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/asset-calc-types", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

func TestRouter_ParcelStatuses_DeniesProfessionalOrgWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:           domain.ID(13),
		Username:     "professional_primary",
		ValuationOrg: true,
	})
	require.NoError(t, err)

	parcelStatusSvc := &mocks.MockParcelStatusService{}
	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		&mocks.MockLandService{},
		nil,
		nil,
		parcelStatusSvc,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/parcel-statuses", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	parcelStatusSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Нээлттэй бүртгэл БАЙХГҮЙ байх ёстой: түүнийг эргүүлж нээвэл хэн ч идэвхтэй
// (`IsActive: true`) аккаунт үүсгэн хүчинтэй JWT авч, permission шалгадаггүй
// маршрутууд руу орно. Аккаунтыг зөвхөн `POST /users` (`users:create`) үүсгэнэ.
func TestRouter_Register_NotExposed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		nil,
		nil,
		nil,
		&mocks.MockLandService{},
		nil,
		nil,
		nil,
		nil,
	)

	body := `{"username":"intruder","password":"Password123!","first_name":"In","last_name":"Truder"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Токенгүйгээр хэрэглэгч үүсгэх боломжгүй — /users нь JWT-ийн ард.
func TestRouter_UsersCreate_RequiresToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userSvc := &mocks.MockUserService{}
	router := handler.NewRouter(
		nil,
		jwtManager,
		nil,
		userSvc,
		nil,
		nil,
		&mocks.MockLandService{},
		nil,
		nil,
		nil,
		nil,
	)

	body := `{"username":"intruder","password":"Password123!","first_name":"In","last_name":"Truder"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	userSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Шүүлтүүрийн dropdown-ы endpoint нь НЭВТЭРСЭН БҮХ хэрэглэгчид нээлттэй:
// permission огт байхгүй, түүнчлэн үндсэн жагсаалтаас хаагдсан хөндлөнгийн
// байгууллагын роль ч жагсаалтын шүүлтүүрийг харна.
func TestRouter_FilterOptions_OpenToAnyAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name        string
		roles       []string
		permissions []string
	}{
		{"эрхгүй хэрэглэгч", nil, nil},
		{"хөндлөнгийн байгууллага", []string{"professional_org"}, []string{"compensation:read"}},
		{"санхүү", []string{"finance_specialist"}, []string{"compensation:read"}},
		{"ахлах мэргэжилтэн", []string{"senior_specialist"}, []string{"land:read"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
			tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
				ID:          domain.ID(3),
				Username:    "u",
				Roles:       tc.roles,
				Permissions: tc.permissions,
			})
			require.NoError(t, err)

			landSvc := &mocks.MockLandService{}
			// Роль тус бүрд ИЖИЛ дуудлага — шүүлт/хязгаарлалтгүй.
			landSvc.On("ListFilterOptions", mock.Anything).
				Return([]*domain.LandAcquisitionOption{{
					AcquisitionName: "Тест чөлөөлөлт",
					PlanCode:        "PLAN-1",
				}}, nil)

			router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil, landSvc, nil, nil, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/filter-options", nil)
			req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			landSvc.AssertExpectations(t)
		})
	}
}

// Нээлттэй гэдэг нь "танилтгүй" гэсэн үг БИШ — токенгүй хүсэлт 401 хэвээр.
func TestRouter_FilterOptions_StillRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	landSvc := &mocks.MockLandService{}

	router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil, landSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/filter-options", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	landSvc.AssertNotCalled(t, "ListFilterOptions", mock.Anything)
}

// ── Үнэлгээний байгууллага: роль БИШ, аппын өөрийн хаалт ─────────────────────

// stubMenuService — /menus/me-д цэсний service ДУУДАГДАХ ЁСГҮЙ (үнэлгээний
// байгууллагын цэс нь аппын өөрийн жагсаалт). Дуудагдвал тест барина.
type stubMenuService struct{ t *testing.T }

func (s stubMenuService) List(context.Context) ([]*domain.Menu, error) { return nil, nil }
func (s stubMenuService) ListByRole(context.Context, domain.ID) ([]*domain.Menu, error) {
	return nil, nil
}
func (s stubMenuService) ListForUser(context.Context, domain.ID) ([]*domain.Menu, error) {
	s.t.Fatal("ListForUser дуудагдах ёсгүй: үнэлгээний байгууллагын цэс роль/DB-ээс уншигдахгүй")
	return nil, nil
}
func (s stubMenuService) ListPermissions(context.Context, domain.ID) ([]domain.Permission, error) {
	return nil, nil
}
func (s stubMenuService) Attach(context.Context, domain.ID, domain.ID) error { return nil }
func (s stubMenuService) Detach(context.Context, domain.ID, domain.ID) error { return nil }

func valuationOrgToken(t *testing.T, jwtManager *jwtpkg.Manager, orgID domain.ID) string {
	t.Helper()
	pair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:       domain.ID(777),
		OrgID:    &orgID,
		Username: "valuation_org_1",
		// РОЛЬ/ЭРХ ХООСОН — хандалт зөвхөн энэ тэмдгээр шийдэгдэнэ.
		ValuationOrg: true,
	})
	require.NoError(t, err)
	return pair.AccessToken
}

// Цэс: ЗӨВХӨН Дашбоард ба Чөлөөлөлт.
func TestRouter_ValuationOrg_MenusMe_ReturnsAppOwnedMenus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	token := valuationOrgToken(t, jwtManager, domain.ID(9001))

	router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil,
		&mocks.MockLandService{}, nil, nil, nil, nil, stubMenuService{t: t})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/menus/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []domain.Menu `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	assert.Equal(t, domain.ValuationOrgMenuDashboardCode, body.Data[0].Code)
	assert.Equal(t, "/dashboard", body.Data[0].URL)
	assert.Equal(t, domain.ValuationOrgMenuLandCode, body.Data[1].Code)
	assert.Equal(t, "/land-acquisitions", body.Data[1].URL)
}

// Allowlist-ээс гадуурх БҮХ зам 403 — эрх/роль шалгах давхарга хүрэхгүй.
func TestRouter_ValuationOrg_DeniesRoutesOutsideAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	token := valuationOrgToken(t, jwtManager, domain.ID(9001))

	landSvc := &mocks.MockLandService{}
	router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil,
		landSvc, nil, nil, nil, nil, stubMenuService{t: t})

	// Дотоод хэрэглэгчийн бүх хэсэг, тайлан, лавлахын БИЧИХ үйлдэл, мөн
	// байгууллага даяарх шүүлтүүрийн жагсаалт.
	blocked := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/roles"},
		{http.MethodGet, "/api/v1/menus"},
		{http.MethodGet, "/api/v1/land-acquisitions"},
		{http.MethodGet, "/api/v1/report/download"},
		{http.MethodPost, "/api/v1/parcel-statuses"},
		// ЖИЧ: /audit-logs, /valuation-orgs, /decision-drafts зэрэг нь
		// харгалзах service өгөгдсөн үед л бүртгэгддэг тул энд 404 буцаана —
		// хаалтыг ValuationOrgScope-ийн unit тест хамарна.
	}
	for _, tc := range blocked {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
	}
	landSvc.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
}

// Шүүлтүүрийн dropdown нь НЭВТЭРСЭН бүх хэрэглэгчид (үнэлгээний байгууллага
// ч) нээлттэй — эрх/роль шаардахгүй.
func TestRouter_ValuationOrg_AllowsFilterOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	token := valuationOrgToken(t, jwtManager, domain.ID(9001))

	landSvc := &mocks.MockLandService{}
	landSvc.On("ListFilterOptions", mock.Anything).
		Return([]*domain.LandAcquisitionOption{{PlanCode: "LA-2024-001"}}, nil)

	router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil,
		landSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/land-acquisitions/filter-options", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	landSvc.AssertExpectations(t)
}

// Лавлах/шүүлтүүрийн маршрутууд — үнэлгээний байгууллага ЭРХГҮЙ ч токеноороо
// дуудна (dropdown-ууд ажиллах ёстой).
func TestRouter_ValuationOrg_AllowsLookupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	token := valuationOrgToken(t, jwtManager, domain.ID(9001))

	parentID := 2
	landSvc := &mocks.MockLandService{}
	landSvc.On("ListAcquisitionCategories", mock.Anything, (*int)(nil)).
		Return([]*domain.AcquisitionCategory{{ID: 1, Name: "Ерөнхий"}}, nil)
	landSvc.On("ListAcquisitionCategories", mock.Anything, &parentID).
		Return([]*domain.AcquisitionCategory{{ID: 3, Name: "Дэд", ParentID: &parentID}}, nil)
	landSvc.On("ListAssetSpecTypes", mock.Anything, mock.Anything).
		Return([]*domain.AssetSpecType{}, nil)

	planSvc := &mocks.MockPlanService{}
	planSvc.On("Suggest", mock.Anything, "PLAN").
		Return([]*domain.Plan{{PlanCode: "PLAN-2024-001"}}, nil)

	router := handler.NewRouter(nil, jwtManager, nil, nil, nil, nil,
		landSvc, planSvc, nil, nil, nil)

	for _, path := range []string{
		"/api/v1/acquisition-categories",
		"/api/v1/acquisition-categories?parent_id=2",
		"/api/v1/asset-spec-types",
		"/api/v1/plans/suggest?q=PLAN",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, path)
	}
}
