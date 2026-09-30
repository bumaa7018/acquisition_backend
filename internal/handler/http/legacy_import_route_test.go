package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Хуучин мэдээллийн импортын замууд: ":id/commit" ба ":id/:step" зөрчилдөхгүй,
// admin биш хэрэглэгч 403 авна.
func TestLegacyImportRoutesAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		roles []string
		want  int
	}{
		{[]string{"specialist"}, http.StatusForbidden},
		{[]string{"senior_specialist"}, http.StatusForbidden},
		{[]string{"Админ"}, http.StatusTeapot},
		{[]string{"admin"}, http.StatusTeapot},
	} {
		r := gin.New()
		g := r.Group("/legacy-import", func(c *gin.Context) { c.Set(middleware.CtxRoles, tc.roles); c.Next() },
			middleware.RequireAnyRole("admin"))
		reached := func(c *gin.Context) { c.String(http.StatusTeapot, c.Param("step")) }
		g.POST("/sessions/:id/commit", reached)
		g.POST("/sessions/:id/rollback", reached)
		g.POST("/sessions/:id/:step", reached)
		for _, path := range []string{"/legacy-import/sessions/x/commit", "/legacy-import/sessions/x/rollback", "/legacy-import/sessions/x/parcels"} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
			if w.Code != tc.want {
				t.Errorf("roles=%v %s: status %d, want %d", tc.roles, path, w.Code, tc.want)
			}
		}
	}
}
