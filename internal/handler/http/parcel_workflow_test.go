package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// transitionCtx — TransitionValuationSubmission handler-ийг шууд дуудах туслах.
// roles/userID-г auth middleware-ийн адилаар context-д тавьж өгнө.
func transitionCtx(t *testing.T, roles []string, body map[string]any) (*mocks.MockLandService, *httptest.ResponseRecorder, func(acqID, parcelID uuid.UUID)) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mockSvc := &mocks.MockLandService{}
	h := handler.NewParcelWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	run := func(acqID, parcelID uuid.UUID) {
		c, _ := gin.CreateTestContext(w)
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		c.Request = httptest.NewRequest(http.MethodPost,
			"/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String()+"/valuation-status",
			bytes.NewReader(payload))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{
			{Key: "id", Value: acqID.String()},
			{Key: "parcel_id", Value: parcelID.String()},
		}
		c.Set("roles", roles)
		c.Set("userID", uuid.New())
		h.TransitionValuationSubmission(c)
	}
	return mockSvc, w, run
}

func sampleSubmission(acqID uuid.UUID, status, vType string) *domain.ValuationSubmission {
	return &domain.ValuationSubmission{
		ID:            uuid.New(),
		AcquisitionID: acqID,
		ParcelID:      "11-01-04-004",
		ValuationType: vType,
		Status:        status,
	}
}

// Санхүү approve хийж болно — бусад урсгалыг repo талд rejected болгодог тул
// энд зөвхөн эрх нэвтэрч, service зөв параметрээр дуудагдахыг шалгана.
func TestTransitionValuation_FinanceCanApprove(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtx(t,
		[]string{"finance_specialist"},
		map[string]any{"action": "approve", "note": "баталгаажуулав", "valuation_type": "independent"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
		"independent", "approve", "баталгаажуулав", mock.Anything).
		Return(sampleSubmission(acqID, domain.ValuationStatusApproved, "independent"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Санхүү return (буцаах) хийж болно.
func TestTransitionValuation_FinanceCanReturn(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtx(t,
		[]string{"Санхүүгийн мэргэжилтэн"},
		map[string]any{"action": "return", "note": "засварла", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
		"asset", "return", "засварла", mock.Anything).
		Return(sampleSubmission(acqID, domain.ValuationStatusReturned, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Санхүү submit хийж болохгүй (илгээх нь мэрг. байгууллага/ахлах/МИКА-гийн үйлдэл).
func TestTransitionValuation_FinanceCannotSubmit(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"finance_specialist"},
		map[string]any{"action": "submit", "note": "т", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Мэргэжлийн байгууллага submit хийж болно.
func TestTransitionValuation_ProfCanSubmit(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtx(t,
		[]string{"professional_org"},
		map[string]any{"action": "submit", "note": "илгээв", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
		"asset", "submit", "илгээв", mock.Anything).
		Return(sampleSubmission(acqID, domain.ValuationStatusSubmitted, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Мэргэжлийн байгууллага approve хийж болохгүй.
func TestTransitionValuation_ProfCannotApprove(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"professional_org"},
		map[string]any{"action": "approve", "note": "т", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// МИКА зөвхөн өөрийн "mika" урсгалыг илгээнэ.
func TestTransitionValuation_MikaCanSubmitOwnStream(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtx(t,
		[]string{"mika"},
		map[string]any{"action": "submit", "note": "илгээв", "valuation_type": "mika"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
		"mika", "submit", "илгээв", mock.Anything).
		Return(sampleSubmission(acqID, domain.ValuationStatusSubmitted, "mika"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// МИКА өөр урсгал (asset) илгээж болохгүй.
func TestTransitionValuation_MikaCannotSubmitOtherStream(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"mika"},
		map[string]any{"action": "submit", "note": "т", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// МИКА approve хийж болохгүй (зөвхөн санхүү/ахлах).
func TestTransitionValuation_MikaCannotApprove(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"mika"},
		map[string]any{"action": "approve", "note": "т", "valuation_type": "mika"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Тайлбар (note) заавал — хоосон бол 400.
func TestTransitionValuation_NoteRequired(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"finance_specialist"},
		map[string]any{"action": "approve", "note": "   ", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Буруу action — 400.
func TestTransitionValuation_InvalidAction(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"senior_specialist"},
		map[string]any{"action": "noop", "note": "т", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Ахлах мэргэжилтэн бүх үйлдлийг хийж чадна (submit + approve).
func TestTransitionValuation_SeniorCanSubmitAndApprove(t *testing.T) {
	for _, action := range []string{"submit", "approve", "return"} {
		acqID, parcelID := uuid.New(), uuid.New()
		mockSvc, w, run := transitionCtx(t,
			[]string{"senior_specialist"},
			map[string]any{"action": action, "note": "т", "valuation_type": "asset"})
		mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
			"asset", action, "т", mock.Anything).
			Return(sampleSubmission(acqID, domain.ValuationStatusSubmitted, "asset"), nil)

		run(acqID, parcelID)

		assert.Equal(t, http.StatusOK, w.Code, "action=%s", action)
		mockSvc.AssertExpectations(t)
	}
}

// Repo-гийн ErrInvalidTransition (ж: rejected урсгалыг дахин илгээх) — 409.
func TestTransitionValuation_InvalidTransitionConflict(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtx(t,
		[]string{"professional_org"},
		map[string]any{"action": "submit", "note": "т", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
		"asset", "submit", "т", mock.Anything).
		Return(nil, domain.ErrInvalidTransition)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockSvc.AssertExpectations(t)
}

// Зураггүй үл хөдлөх хөрөнгөтэй илгээлт — 422 (repo ErrAssetPhotoRequired).
func TestTransitionValuation_PhotoRequired(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtx(t,
		[]string{"professional_org"},
		map[string]any{"action": "submit", "note": "т", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything, acqID, parcelID,
		"asset", "submit", "т", mock.Anything).
		Return(nil, domain.ErrAssetPhotoRequired)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	mockSvc.AssertExpectations(t)
}
