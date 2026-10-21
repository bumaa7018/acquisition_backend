package http_test

// Захирамжийн төслийн маршрутууд ЗӨВХӨН decision:* эрхээр хамгаалагдсан эсэх.
//
// Өмнө нь land:* эрхээр удирдагдаж байсан тул газар чөлөөлөлт засах эрхтэй хүн
// бүр захирамжийн төсөл үүсгэж, батлах боломжтой байв. Эрхийн шалгалт нь
// middleware дээр (handler-аас ӨМНӨ) хийгддэг тул stub service хүртэл
// дуудагдахгүйгээр 403 буцна.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	jwtpkg "acquisition_backend/internal/pkg/jwt"
	"acquisition_backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDecisionDraftService — эрхийн шалгалтыг тестлэхэд хангалттай хоосон
// хэрэгжүүлэлт. Хориглогдох тохиолдолд ЭНЭ КОД ХҮРТЭЛ ХҮРЭХГҮЙ.
type stubDecisionDraftService struct{ called bool }

func (s *stubDecisionDraftService) List(context.Context, domain.DecisionDraftFilter) ([]*domain.DecisionDraft, int64, error) {
	s.called = true
	return []*domain.DecisionDraft{}, 0, nil
}

func (s *stubDecisionDraftService) GetByID(context.Context, string) (*domain.DecisionDraft, error) {
	s.called = true
	return &domain.DecisionDraft{}, nil
}

func (s *stubDecisionDraftService) Create(context.Context, service.CreateDecisionDraftRequest, string) (*domain.DecisionDraft, error) {
	s.called = true
	return &domain.DecisionDraft{}, nil
}

func (s *stubDecisionDraftService) Update(context.Context, string, service.UpdateDecisionDraftRequest, string) (*domain.DecisionDraft, error) {
	s.called = true
	return &domain.DecisionDraft{}, nil
}

func (s *stubDecisionDraftService) Delete(context.Context, string) error {
	s.called = true
	return nil
}

func (s *stubDecisionDraftService) Confirm(context.Context, string, service.ConfirmDecisionDraftRequest, string) (*domain.DecisionDraft, error) {
	s.called = true
	return &domain.DecisionDraft{}, nil
}

func (s *stubDecisionDraftService) AddProgress(context.Context, string, service.AddDecisionDraftProgressRequest, string) (*domain.DecisionDraftProgressHistory, error) {
	s.called = true
	return &domain.DecisionDraftProgressHistory{}, nil
}

func (s *stubDecisionDraftService) ListProgressHistory(context.Context, string) ([]*domain.DecisionDraftProgressHistory, error) {
	s.called = true
	return nil, nil
}

func (s *stubDecisionDraftService) ListParcels(context.Context, string) ([]*domain.DecisionDraftParcel, error) {
	s.called = true
	return nil, nil
}

func (s *stubDecisionDraftService) ListByParcel(context.Context, string) ([]*domain.DecisionDraftParcel, error) {
	s.called = true
	return nil, nil
}

func (s *stubDecisionDraftService) LinkParcel(context.Context, string, string, string, string) error {
	s.called = true
	return nil
}

func (s *stubDecisionDraftService) SetParcelFundingSource(context.Context, string, string, string) error {
	s.called = true
	return nil
}

func (s *stubDecisionDraftService) UnlinkParcel(context.Context, string, string, string) error {
	s.called = true
	return nil
}

func (s *stubDecisionDraftService) ListFundingSourceOptions(context.Context) ([]*domain.FundingSourceOption, error) {
	s.called = true
	return nil, nil
}

func (s *stubDecisionDraftService) CreateFundingSourceOption(context.Context, service.CreateFundingSourceRequest, string) (*domain.FundingSourceOption, error) {
	s.called = true
	return &domain.FundingSourceOption{}, nil
}

func (s *stubDecisionDraftService) ListDraftFundingSources(context.Context, string) ([]*domain.DecisionDraftFundingLink, error) {
	s.called = true
	return nil, nil
}

func (s *stubDecisionDraftService) AddFundingSource(context.Context, string, string, float64, string) error {
	s.called = true
	return nil
}

func (s *stubDecisionDraftService) RemoveFundingSource(context.Context, string, string) error {
	s.called = true
	return nil
}

func decisionRouter(t *testing.T, stub *stubDecisionDraftService) (*gin.Engine, *jwtpkg.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtManager := jwtpkg.NewManager("access-secret", "refresh-secret", time.Hour, time.Hour)
	router := handler.NewRouter(
		nil, jwtManager, nil, nil, nil, nil, nil, nil, nil, nil,
		service.DecisionServices{Draft: stub},
	)
	return router, jwtManager
}

func decisionToken(t *testing.T, m *jwtpkg.Manager, perms ...string) string {
	t.Helper()
	pair, err := m.GenerateTokenPair(jwtpkg.UserInfo{
		ID:          domain.ID(1),
		Username:    "specialist",
		Roles:       []string{"senior_specialist"},
		Permissions: perms,
	})
	require.NoError(t, err)
	return pair.AccessToken
}

func TestRouter_DecisionDrafts_RequireDecisionPermissions(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		// Тухайн маршрутад ШААРДАГДАХ эрх
		required string
	}{
		{"жагсаалт", http.MethodGet, "/api/v1/decision-drafts", "decision:read"},
		{"дэлгэрэнгүй", http.MethodGet, "/api/v1/decision-drafts/" + uuid.NewString(), "decision:read"},
		{"нэгж талбарууд", http.MethodGet, "/api/v1/decision-drafts/" + uuid.NewString() + "/parcels", "decision:read"},
		{"эх үүсвэрийн сонголт", http.MethodGet, "/api/v1/funding-sources", "decision:read"},
		{"нэгж талбарын захирамж", http.MethodGet, "/api/v1/parcels/" + uuid.NewString() + "/decision-drafts", "decision:read"},
		{"үүсгэх", http.MethodPost, "/api/v1/decision-drafts", "decision:create"},
		{"засах", http.MethodPut, "/api/v1/decision-drafts/" + uuid.NewString(), "decision:update"},
		{"батлах", http.MethodPost, "/api/v1/decision-drafts/" + uuid.NewString() + "/confirm", "decision:update"},
		{"устгах", http.MethodDelete, "/api/v1/decision-drafts/" + uuid.NewString(), "decision:delete"},
	}

	for _, tc := range cases {
		t.Run(tc.name+" — land:* эрх хүрэхгүй", func(t *testing.T) {
			stub := &stubDecisionDraftService{}
			router, jwtManager := decisionRouter(t, stub)
			token := decisionToken(t, jwtManager,
				"land:read", "land:create", "land:update", "land:delete")

			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.False(t, stub.called, "эрхгүй хүсэлт handler хүртэл хүрсэн")
		})

		t.Run(tc.name+" — "+tc.required+" эрхтэй нэвтэрнэ", func(t *testing.T) {
			stub := &stubDecisionDraftService{}
			router, jwtManager := decisionRouter(t, stub)
			token := decisionToken(t, jwtManager, tc.required)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Эрх нь давсан — 403 БИШ байх нь хангалттай (body-гоос хамаарахгүй:
			// POST/PUT нь хоосон биетэй тул 400 буцаж болно).
			assert.NotEqual(t, http.StatusForbidden, w.Code)
		})
	}
}
