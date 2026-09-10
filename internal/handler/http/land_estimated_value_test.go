package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "acquisition_backend/internal/handler/http"
	"acquisition_backend/internal/handler/http/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEstimatedValueRequest(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
	}{
		{"calculate", `{"confidence_percent":2}`, http.StatusOK},
		{"manual", `{"confidence_percent":2,"base_fee_per_m2":10}`, http.StatusOK},
		{"base fee without percentage", `{"base_fee_per_m2":10}`, http.StatusBadRequest},
		{"clear", `{"value":null}`, http.StatusOK},
		{"empty request must not clear", `{}`, http.StatusBadRequest},
		{"null percentage must not clear", `{"confidence_percent":null}`, http.StatusBadRequest},
		{"ambiguous input", `{"confidence_percent":2,"value":100}`, http.StatusBadRequest},
		{"invalid percentage type", `{"confidence_percent":"bad"}`, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mocks.MockLandService{}
			acqID, parcelID := uuid.New(), uuid.New()
			if tt.name == "calculate" {
				svc.On("CalculateParcelEstimatedValue", mock.Anything, acqID, parcelID, 2.0, (*float64)(nil), mock.Anything).Return(125000.0, nil)
			} else if tt.name == "manual" {
				baseFee := 10.0
				svc.On("CalculateParcelEstimatedValue", mock.Anything, acqID, parcelID, 2.0, &baseFee, mock.Anything).Return(100000.0, nil)
			} else if tt.name == "clear" {
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
			svc.AssertExpectations(t)
		})
	}
}
