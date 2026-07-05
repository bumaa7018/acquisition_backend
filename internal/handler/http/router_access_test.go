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
		ID:          uuid.New(),
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
	userID := uuid.New()
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          userID,
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read"},
	})
	require.NoError(t, err)

	landSvc := &mocks.MockLandService{}
	acq := &domain.LandAcquisition{ID: uuid.New(), PlanCode: "LA-2024-001"}
	landSvc.On("List", mock.Anything, mock.MatchedBy(func(filter domain.LandAcquisitionFilter) bool {
		return filter.ProfessionalOrgUserID == userID.String() &&
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

func TestRouter_ProfLandAcquisitions_DeniesAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          uuid.New(),
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

func TestRouter_GeneralLandAcquisition_DeniesProfessionalOrgWhenNotAssigned(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := uuid.New()
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          userID,
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read"},
	})
	require.NoError(t, err)

	acqID := uuid.New()
	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAssigned", mock.Anything, acqID, userID).Return(false, nil)

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
	landSvc.AssertExpectations(t)
	landSvc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestRouter_GeneralLandAcquisitionList_DeniesProfessionalOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          uuid.New(),
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read"},
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
	userID := uuid.New()
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          userID,
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read"},
	})
	require.NoError(t, err)

	parcelID := uuid.New()
	landSvc := &mocks.MockLandService{}
	landSvc.On("IsParcelAcquisitionLocked", mock.Anything, parcelID).Return(false, nil)
	landSvc.On("IsProfMemberOfParcel", mock.Anything, parcelID, userID).Return(false, nil)

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

func TestRouter_ProfLandAcquisitionParcels_ReturnsOnlyEvaluationStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := uuid.New()
	tokenPair, err := jwtManager.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          userID,
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read"},
	})
	require.NoError(t, err)

	acqID := uuid.New()
	landSvc := &mocks.MockLandService{}
	landSvc.On("IsAcquisitionLocked", mock.Anything, acqID).Return(false, nil)
	landSvc.On("IsProfMemberOfAcquisition", mock.Anything, acqID, userID).Return(true, nil)
	landSvc.On("GetParcels", mock.Anything, acqID, mock.MatchedBy(func(filter domain.ParcelFilter) bool {
		return filter.StatusID == domain.ParcelStatusEvaluating &&
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
		ID:          uuid.New(),
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read"},
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
		ID:          uuid.New(),
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
		ID:          uuid.New(),
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
		ID:          uuid.New(),
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
		ID:          uuid.New(),
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
		ID:          uuid.New(),
		Username:    "professional_primary",
		Roles:       []string{"professional_org"},
		Permissions: []string{"compensation:read", "compensation:update"},
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
