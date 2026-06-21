package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/your-org/go-api-template/internal/domain"
	handler "github.com/your-org/go-api-template/internal/handler/http"
	"github.com/your-org/go-api-template/internal/handler/http/mocks"
)

func TestPlanHandler_Search_Success(t *testing.T) {
	mockSvc := &mocks.MockPlanService{}
	h := handler.NewPlanHandler(mockSvc)

	plan := &domain.Plan{
		ID:       uuid.New(),
		PlanCode: "PLAN-2024-001",
		Name:     "Зуунмод төлөвлөгөө",
		AreaM2:   1500000,
		Status:   2,
	}
	mockSvc.On("Search", mock.Anything, "PLAN-2024-001").Return(plan, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/plans/search?code=PLAN-2024-001", nil)

	h.Search(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "PLAN-2024-001", data["plan_code"])
	mockSvc.AssertExpectations(t)
}

func TestPlanHandler_Search_MissingCode(t *testing.T) {
	mockSvc := &mocks.MockPlanService{}
	h := handler.NewPlanHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/plans/search", nil)

	h.Search(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlanHandler_Search_NotFound(t *testing.T) {
	mockSvc := &mocks.MockPlanService{}
	h := handler.NewPlanHandler(mockSvc)

	mockSvc.On("Search", mock.Anything, "INVALID-CODE").Return(nil, domain.ErrNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/plans/search?code=INVALID-CODE", nil)

	h.Search(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}
