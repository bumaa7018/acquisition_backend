package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"acquisition_backend/internal/domain"
	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Хүсэлтийн ХЭЛБЭРийг шалгана: `value` талбар ирсэн бол шууд бичих/арилгах,
// ирээгүй бол сервер тооцох. Итгэлцүүр ЗААВАЛ эсэхийг эрхийн төрлөөр service
// шийддэг (land_estimated_value_test.go) — handler нь тэр дүрмийг зөвхөн
// дамжуулж, алдааг статус болгоно.
func TestEstimatedValueRequest(t *testing.T) {
	percent, baseFee := 2.0, 10.0
	for _, tt := range []struct {
		name, body string
		status     int
		// Тооцоолох салаа руу ямар аргументаар орох ёстой (nil = орохгүй).
		wantPercent, wantBaseFee *float64
		calculate                bool
		svcErr                   error
	}{
		{name: "calculate", body: `{"confidence_percent":2}`, status: http.StatusOK, calculate: true, wantPercent: &percent},
		{name: "manual", body: `{"confidence_percent":2,"base_fee_per_m2":10}`, status: http.StatusOK, calculate: true, wantPercent: &percent, wantBaseFee: &baseFee},
		// ӨМЧЛӨХ эрхийн газар — итгэлцүүргүйгээр зөвхөн жишиг үнэ илгээнэ.
		{name: "market price without percentage", body: `{"base_fee_per_m2":10}`, status: http.StatusOK, calculate: true, wantBaseFee: &baseFee},
		// Хадгалсан жишиг үнээр дахин тооцох (шинэ утга оруулаагүй).
		{name: "recalculate from saved values", body: `{}`, status: http.StatusOK, calculate: true},
		{name: "null percentage is not a clear request", body: `{"confidence_percent":null}`, status: http.StatusBadRequest, calculate: true, svcErr: domain.ErrInvalidInput},
		{name: "clear", body: `{"value":null}`, status: http.StatusOK},
		{name: "ambiguous input", body: `{"confidence_percent":2,"value":100}`, status: http.StatusBadRequest},
		{name: "base fee with value", body: `{"base_fee_per_m2":10,"value":100}`, status: http.StatusBadRequest},
		{name: "invalid percentage type", body: `{"confidence_percent":"bad"}`, status: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockLandService{}
			acqID, parcelID := uuid.New(), uuid.New()
			switch {
			case tt.calculate:
				svc.On("CalculateParcelEstimatedValue", mock.Anything, acqID, parcelID, tt.wantPercent, tt.wantBaseFee, mock.Anything).
					Return(125000.0, tt.svcErr)
			case tt.name == "clear":
				svc.On("SetParcelEstimatedValue", mock.Anything, acqID, parcelID, (*float64)(nil), mock.Anything).Return(nil)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: acqID.String()}, {Key: "parcel_id", Value: parcelID.String()}}
			c.Request = httptest.NewRequest(http.MethodPatch, "/estimated-value", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler.NewLandHandler(svc).SetParcelEstimatedValue(c)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			if tt.name == "calculate" {
				require.Contains(t, w.Body.String(), `"estimated_value":125000`)
				require.Contains(t, w.Body.String(), `"estimated_confidence_percent":2`)
			}
			if tt.name == "market price without percentage" {
				// Итгэлцүүр хэрэглээгүй нь хариунд ч харагдана.
				require.Contains(t, w.Body.String(), `"estimated_confidence_percent":null`)
			}
			svc.AssertExpectations(t)
		})
	}
}
