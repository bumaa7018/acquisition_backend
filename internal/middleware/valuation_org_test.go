package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Allowlist нь ЗАМ ЗҮЙН хилээр таарах ёстой: "/prof" нь "/prof/..."-той таарч,
// "/professional-orgs" мэт өөр маршруттай ХЭЗЭЭ Ч таарахгүй.
func TestValuationOrgRouteAllowed(t *testing.T) {
	cases := []struct {
		route  string
		method string
		want   bool
	}{
		{"/prof/land-acquisitions", http.MethodGet, true},
		{"/prof/land-acquisitions/:id/compensations", http.MethodPost, true},
		{"/prof", http.MethodGet, true},
		{"/dashboard", http.MethodGet, true},
		{"/users/me", http.MethodGet, true},
		{"/menus/me", http.MethodGet, true},
		{"/notifications/:id/read", http.MethodPost, true},
		// Лавлах — ЗӨВХӨН унших.
		{"/parcel-statuses", http.MethodGet, true},
		{"/parcel-statuses", http.MethodPost, false},
		{"/parcel-statuses/:id", http.MethodDelete, false},
		{"/document-types", http.MethodGet, true},
		{"/plans/search", http.MethodGet, true},
		// Шүүлтүүр/лавлах — токентой бол нээлттэй, БИЧИХ нь хаалттай.
		{"/land-acquisitions/filter-options", http.MethodGet, true},
		{"/land-acquisitions/districts", http.MethodGet, true},
		{"/acquisition-categories", http.MethodGet, true},
		{"/acquisition-categories", http.MethodPost, false},
		{"/acquisition-categories/:id", http.MethodPut, false},
		{"/parcel-workflow", http.MethodGet, true},
		{"/parcel-workflow", http.MethodPost, false},
		{"/acquisition-workflow", http.MethodGet, true},
		{"/acquisition-workflow/statuses", http.MethodGet, true},
		{"/acquisition-workflow/:id", http.MethodDelete, false},
		{"/asset-spec-types", http.MethodGet, true},
		{"/asset-spec-types/:id", http.MethodPut, false},
		{"/asset-calc-types", http.MethodGet, true},
		{"/decision-work-types", http.MethodGet, true},
		{"/decision-work-types/:id", http.MethodGet, true},
		{"/decision-work-types", http.MethodDelete, false},
		{"/decision-budgets", http.MethodGet, true},
		{"/plans/suggest", http.MethodGet, true},
		// Хаалттай — дотоод болон бүх нэрийн хувилбар.
		{"/professional-orgs", http.MethodGet, false},
		{"/profile", http.MethodGet, false},
		{"/users", http.MethodGet, false},
		{"/users/:id", http.MethodGet, false},
		{"/menus", http.MethodGet, false},
		{"/land-acquisitions", http.MethodGet, false},
		{"/land-acquisitions/filter-options", http.MethodPost, false},
		{"/land-acquisitions/districts", http.MethodPost, false},
		// Геометрийн тооцоолуур — гадаад дүрд ХЭРЭГГҮЙ (товч нь !isExternal-ын
		// дор), тиймээс allowlist-д ОРУУЛААГҮЙ.
		{"/geometry/area", http.MethodPost, false},
		{"/employees", http.MethodGet, false},
		{"/valuation-orgs", http.MethodGet, false},
		{"/report/download", http.MethodGet, false},
		{"/decision-drafts", http.MethodGet, false},
		{"/funding-sources", http.MethodGet, false},
		{"/compensations", http.MethodGet, false},
		{"/construction-types", http.MethodGet, false},
		{"/files/authorize", http.MethodGet, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, valuationOrgRouteAllowed(tc.route, tc.method),
			"%s %s", tc.method, tc.route)
	}
}

// Дотоод хэрэглэгчийг (valuation_org тэмдэггүй) allowlist ХӨНДӨХГҮЙ.
func TestValuationOrgScope_IgnoresInternalCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxValuationOrg, false); c.Next() })
	r.Use(ValuationOrgScope("/api/v1"))
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

// Үнэлгээний байгууллагын ажилтан allowlist-ээс гадуур 403 авна.
func TestValuationOrgScope_BlocksNonAllowlisted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxValuationOrg, true); c.Next() })
	r.Use(ValuationOrgScope("/api/v1"))
	r.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/v1/prof/land-acquisitions", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/prof/land-acquisitions", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

// RequireValuationOrgCaller — тэмдэггүй (дотоод, эсвэл роль л эзэмшсэн хуучин
// professional_org) хэрэглэгчийг нэвтрүүлэхгүй.
func TestRequireValuationOrgCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	run := func(flag any) int {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if flag != nil {
				c.Set(CtxValuationOrg, flag)
			}
			c.Next()
		})
		r.GET("/x", RequireValuationOrgCaller(), func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		return w.Code
	}
	assert.Equal(t, http.StatusOK, run(true))
	assert.Equal(t, http.StatusForbidden, run(false))
	assert.Equal(t, http.StatusForbidden, run(nil))
}
