package http_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"
	"acquisition_backend/internal/middleware"
	"acquisition_backend/internal/service"

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
	return transitionCtxAs(t, roles, false, body)
}

// transitionCtxAs — valuationOrg=true бол дуудагч нь ҮНЭЛГЭЭНИЙ БАЙГУУЛЛАГЫН
// ажилтан (токены `valuation_org`). Тэдгээр хэрэглэгч роль эзэмшихгүй тул
// roles нь хоосон байхыг зөвшөөрнө.
func transitionCtxAs(t *testing.T, roles []string, valuationOrg bool, body map[string]any) (*mocks.MockLandService, *httptest.ResponseRecorder, func(acqID, parcelID uuid.UUID)) {
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
		c.Set(middleware.CtxValuationOrg, valuationOrg)
		c.Set(middleware.CtxUserID, domain.ID(77))
		c.Set(middleware.CtxUserName, "Тест Ажилтан")
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
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "independent" && in.Action == "approve" && in.Note == "баталгаажуулав"
		}), mock.Anything).
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
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "asset" && in.Action == "return" && in.Note == "засварла"
		}), mock.Anything).
		Return(sampleSubmission(acqID, domain.ValuationStatusReturned, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Санхүү баталгаажсан үнэлгээг хүчингүй болгож, дахин үнэлгээ хийх төлөвт буцааж болно.
func TestTransitionValuation_FinanceCanCancelApproved(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		[]string{"finance_specialist"},
		map[string]string{"action": "cancel", "note": "шинээр үнэлгээ хийлгэнэ", "valuation_type": "asset"},
		"cancel-report.pdf", samplePDF)
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "asset" && in.Action == "cancel" &&
				in.Note == "шинээр үнэлгээ хийлгэнэ" && in.UserID == "Тест Ажилтан"
		}),
		mock.MatchedBy(func(a *service.ValuationAttachment) bool {
			return a != nil && a.Filename == "cancel-report.pdf"
		})).
		Return(sampleSubmission(acqID, domain.ValuationStatusDraft, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Эрхтэй хэрэглэгч ч цуцлахдаа PDF хавсралт заавал явуулна.
func TestTransitionValuation_CancelRequiresAttachment(t *testing.T) {
	mockSvc, w, run := transitionCtx(t,
		[]string{"finance_specialist"},
		map[string]any{"action": "cancel", "note": "шинээр үнэлгээ хийлгэнэ", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything)
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
	mockSvc, w, run := transitionCtxAs(t,
		nil, true,
		map[string]any{"action": "submit", "note": "илгээв", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "asset" && in.Action == "submit" && in.Note == "илгээв"
		}), mock.Anything).
		Return(sampleSubmission(acqID, domain.ValuationStatusSubmitted, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Мэргэжлийн байгууллага өөрийн илгээсэн/баталгаажсан үнэлгээг өөрөө хүчингүй болгохгүй.
func TestTransitionValuation_ProfCannotCancelApproved(t *testing.T) {
	mockSvc, w, run := transitionCtxAs(t,
		nil, true,
		map[string]any{"action": "cancel", "note": "цуцлах", "valuation_type": "asset"})

	run(uuid.New(), uuid.New())

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything)
}

// Мэргэжлийн байгууллага approve хийж болохгүй.
func TestTransitionValuation_ProfCannotApprove(t *testing.T) {
	mockSvc, w, run := transitionCtxAs(t,
		nil, true,
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
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "mika" && in.Action == "submit" && in.Note == "илгээв"
		}), mock.Anything).
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

// Ахлах мэргэжилтэн submit/approve/return хийж чадна.
func TestTransitionValuation_SeniorCanSubmitAndApprove(t *testing.T) {
	for _, action := range []string{"submit", "approve", "return"} {
		acqID, parcelID := uuid.New(), uuid.New()
		mockSvc, w, run := transitionCtx(t,
			[]string{"senior_specialist"},
			map[string]any{"action": action, "note": "т", "valuation_type": "asset"})
		mockSvc.On("TransitionValuationSubmission", mock.Anything,
			mock.MatchedBy(func(in domain.ValuationTransition) bool {
				return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
					in.ValuationType == "asset" && in.Action == action && in.Note == "т"
			}), mock.Anything).
			Return(sampleSubmission(acqID, domain.ValuationStatusSubmitted, "asset"), nil)

		run(acqID, parcelID)

		assert.Equal(t, http.StatusOK, w.Code, "action=%s", action)
		mockSvc.AssertExpectations(t)
	}
}

// Ахлах мэргэжилтэн баталгаажсан үнэлгээг цуцалж чадна.
func TestTransitionValuation_SeniorCanCancelApproved(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		[]string{"senior_specialist"},
		map[string]string{"action": "cancel", "note": "дахин үнэлгээ", "valuation_type": "asset"},
		"cancel.pdf", samplePDF)
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.Action == "cancel" && in.UserID == "Тест Ажилтан"
		}),
		mock.MatchedBy(func(a *service.ValuationAttachment) bool { return a != nil && a.Filename == "cancel.pdf" })).
		Return(sampleSubmission(acqID, domain.ValuationStatusDraft, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Тухайн чөлөөлөлт дээр ажиллаж буй дотоод ажилтан баталгаажсан үнэлгээг цуцалж чадна.
func TestTransitionValuation_AssignedEmployeeCanCancelApproved(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		nil,
		map[string]string{"action": "cancel", "note": "дахин үнэлгээ", "valuation_type": "asset"},
		"cancel.pdf", samplePDF)
	mockSvc.On("IsAssigned", mock.Anything, acqID, domain.ID(77)).Return(true, nil)
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.Action == "cancel" && in.UserID == "Тест Ажилтан"
		}),
		mock.MatchedBy(func(a *service.ValuationAttachment) bool { return a != nil && a.Filename == "cancel.pdf" })).
		Return(sampleSubmission(acqID, domain.ValuationStatusDraft, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Repo-гийн ErrInvalidTransition (ж: rejected урсгалыг дахин илгээх) — 409.
func TestTransitionValuation_InvalidTransitionConflict(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtxAs(t,
		nil, true,
		map[string]any{"action": "submit", "note": "т", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "asset" && in.Action == "submit" && in.Note == "т"
		}), mock.Anything).
		Return(nil, domain.ErrInvalidTransition)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockSvc.AssertExpectations(t)
}

// Зураггүй үл хөдлөх хөрөнгөтэй илгээлт — 422 (repo ErrAssetPhotoRequired).
func TestTransitionValuation_PhotoRequired(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := transitionCtxAs(t,
		nil, true,
		map[string]any{"action": "submit", "note": "т", "valuation_type": "asset"})
	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.AcquisitionID == acqID && in.ParcelID == parcelID &&
				in.ValuationType == "asset" && in.Action == "submit" && in.Note == "т"
		}), mock.Anything).
		Return(nil, domain.ErrAssetPhotoRequired)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	mockSvc.AssertExpectations(t)
}

// ── БУЦААХ/ЦУЦЛАХ ХАВСРАЛТ (multipart) ─────────────────────────────────────
//
// Буцаах хавсралт нь ЗААВАЛ БИШ, цуцлах хавсралт нь ЗААВАЛ. Файл ирвэл
// service-д ValuationAttachment дамжина.

// multipartTransition — multipart/form-data хүсэлтээр шилжилт дуудна.
// fileName хоосон бол файл ХАВСАРГАХГҮЙ.
func multipartTransition(
	t *testing.T, roles []string, fields map[string]string, fileName string, fileBody []byte,
) (*mocks.MockLandService, *httptest.ResponseRecorder, func(acqID, parcelID uuid.UUID)) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mockSvc := &mocks.MockLandService{}
	h := handler.NewParcelWorkflowHandler(mockSvc)

	w := httptest.NewRecorder()
	run := func(acqID, parcelID uuid.UUID) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		for k, v := range fields {
			require.NoError(t, mw.WriteField(k, v))
		}
		if fileName != "" {
			part, err := mw.CreateFormFile("file", fileName)
			require.NoError(t, err)
			_, err = part.Write(fileBody)
			require.NoError(t, err)
		}
		require.NoError(t, mw.Close())

		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost,
			"/land-acquisitions/"+acqID.String()+"/parcels/"+parcelID.String()+"/valuation-status",
			&buf)
		c.Request.Header.Set("Content-Type", mw.FormDataContentType())
		c.Params = gin.Params{
			{Key: "id", Value: acqID.String()},
			{Key: "parcel_id", Value: parcelID.String()},
		}
		c.Set("roles", roles)
		c.Set(middleware.CtxUserID, domain.ID(77))
		c.Set(middleware.CtxUserName, "Тест Ажилтан")
		h.TransitionValuationSubmission(c)
	}
	return mockSvc, w, run
}

// %PDF- толгой — allowedPDFContentType нь агуулгаар шалгадаг.
var samplePDF = append([]byte("%PDF-1.7\n"), make([]byte, 64)...)

func TestTransitionValuation_ReturnWithAttachment(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		[]string{"finance_specialist"},
		map[string]string{"action": "return", "note": "3-р хуудсыг залруулна", "valuation_type": "asset"},
		"zalruulga.pdf", samplePDF)

	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool {
			return in.Action == "return" && in.Note == "3-р хуудсыг залруулна"
		}),
		mock.MatchedBy(func(a *service.ValuationAttachment) bool {
			return a != nil && a.Filename == "zalruulga.pdf" && len(a.Data) == len(samplePDF)
		})).
		Return(sampleSubmission(acqID, domain.ValuationStatusReturned, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Хавсралт нь ЗААВАЛ БИШ — multipart-аар файлгүй ирвэл nil дамжина.
func TestTransitionValuation_ReturnWithoutAttachment(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		[]string{"finance_specialist"},
		map[string]string{"action": "return", "note": "тайлбар", "valuation_type": "asset"},
		"", nil)

	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool { return in.Action == "return" }),
		(*service.ValuationAttachment)(nil)).
		Return(sampleSubmission(acqID, domain.ValuationStatusReturned, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// Хавсралт нь ЗӨВХӨН буцаалт/цуцлалтад хамаарна: "илгээх" дээр файл ирвэл
// хаягдана (эс бөгөөс тэр файл буцаалт/цуцлалтын хавсралт болж харагдана).
func TestTransitionValuation_SubmitDropsAttachment(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		[]string{"senior_specialist"},
		map[string]string{"action": "submit", "note": "илгээв", "valuation_type": "asset"},
		"file.pdf", samplePDF)

	mockSvc.On("TransitionValuationSubmission", mock.Anything,
		mock.MatchedBy(func(in domain.ValuationTransition) bool { return in.Action == "submit" }),
		(*service.ValuationAttachment)(nil)).
		Return(sampleSubmission(acqID, domain.ValuationStatusSubmitted, "asset"), nil)

	run(acqID, parcelID)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// PDF биш файлыг хүлээж авахгүй (allowedPDFContentType).
func TestTransitionValuation_ReturnRejectsNonPDF(t *testing.T) {
	acqID, parcelID := uuid.New(), uuid.New()
	mockSvc, w, run := multipartTransition(t,
		[]string{"finance_specialist"},
		map[string]string{"action": "return", "note": "тайлбар", "valuation_type": "asset"},
		"virus.exe", []byte("MZ\x90\x00 not a pdf"))

	run(acqID, parcelID)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "TransitionValuationSubmission",
		mock.Anything, mock.Anything, mock.Anything)
}
