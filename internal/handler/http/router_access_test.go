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
