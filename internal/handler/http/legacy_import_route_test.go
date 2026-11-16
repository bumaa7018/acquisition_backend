package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Хуучин мэдээллийн импортын замууд: ":id/commit" ба ":id/:step" зөрчилдөхгүй.
// Админ ба ахлах мэргэжилтэн нэвтэрнэ, бусад нь 403.
func TestLegacyImportRoutesAdminAndSenior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		roles []string
		want  int
	}{
		{[]string{"specialist"}, http.StatusForbidden},
		{[]string{"employee"}, http.StatusForbidden},
		{[]string{"department_head"}, http.StatusForbidden},
		{[]string{"finance_specialist"}, http.StatusForbidden},
		{[]string{"senior_specialist"}, http.StatusTeapot},
		{[]string{"Ахлах мэргэжилтэн"}, http.StatusTeapot},
		{[]string{"Админ"}, http.StatusTeapot},
		{[]string{"admin"}, http.StatusTeapot},
	} {
		r := gin.New()
		g := r.Group("/legacy-import", func(c *gin.Context) { c.Set(middleware.CtxRoles, tc.roles); c.Next() },
			middleware.RequireAnyRole(LegacyImportRoles...))
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

// Жагсаалтын хүрээ: админ бүгдийг (nil), бусад нь зөвхөн өөрийнхөө дугаараар.
func TestLegacyOwnerScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := func(roles []string, uid any) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(middleware.CtxRoles, roles)
		if uid != nil {
			c.Set(middleware.CtxUserID, uid)
		}
		return c
	}
	if got := legacyOwnerScope(ctx([]string{"admin"}, domain.ID(1))); got != nil {
		t.Errorf("admin: scope = %v, want nil (бүгд)", *got)
	}
	if got := legacyOwnerScope(ctx([]string{"Админ"}, domain.ID(1))); got != nil {
		t.Errorf("Админ: scope = %v, want nil (бүгд)", *got)
	}
	if got := legacyOwnerScope(ctx([]string{"senior_specialist"}, domain.ID(35))); got == nil || *got != 35 {
		t.Errorf("senior: scope = %v, want 35", got)
	}
	// Дугааргүй дуудагч юу ч харахгүй (бүгдийг харуулах руу унахгүй).
	if got := legacyOwnerScope(ctx([]string{"senior_specialist"}, nil)); got == nil || *got != -1 {
		t.Errorf("senior without id: scope = %v, want -1", got)
	}
}
