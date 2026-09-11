package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// stageChecker — чөлөөлөлтийн төлөвийг тогтмол буцаах тестийн хувилбар.
type stageChecker struct{ status int }

func (s stageChecker) AcquisitionStatus(context.Context, uuid.UUID) (int, error) {
	return s.status, nil
}

// "Шинэ" төлөвт нэгж талбартай ажиллах БИЧИХ үйлдлийг (явц солих, үнэлгээ)
// хаана: тэр шат нь нэгж талбарын бүрдэл хийгддэг ба бүрдэл өөрчлөгдөхөд
// бичигдсэн өгөгдөл нэгж талбартайгаа хамт устах эрсдэлтэй.
func TestRequireAcquisitionFieldStage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	acqID := uuid.New()

	for _, tt := range []struct {
		name    string
		status  int
		method  string
		allowed bool
	}{
		{name: "шинэ төлөвт явц солихыг хаана", status: domain.LandAcquisitionStatusNew, method: http.MethodPatch, allowed: false},
		{name: "хээрийн судалгаанд зөвшөөрнө", status: domain.LandAcquisitionStatusField, method: http.MethodPatch, allowed: true},
		// УНШИЛТ хаагдахгүй — "Шинэ" төлөвт ч явцын түүх/жагсаалт харагдана.
		{name: "шинэ төлөвт ч уншина", status: domain.LandAcquisitionStatusNew, method: http.MethodGet, allowed: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, engine := gin.CreateTestContext(w)
			called := false
			engine.Handle(tt.method, "/land-acquisitions/:id/parcels/:parcel_id/status",
				middleware.RequireAcquisitionFieldStage(stageChecker{status: tt.status}),
				func(c *gin.Context) { called = true; c.Status(http.StatusOK) })
			c.Request = httptest.NewRequest(tt.method, "/land-acquisitions/"+acqID.String()+"/parcels/"+uuid.New().String()+"/status", nil)
			engine.ServeHTTP(w, c.Request)

			require.Equal(t, tt.allowed, called, "handler дуудагдсан эсэх")
			if !tt.allowed {
				// 423 Locked — "энэ шатанд боломжгүй" гэдгийг frontend ялгаж таних.
				require.Equal(t, http.StatusLocked, w.Code, w.Body.String())
			}
		})
	}
}
