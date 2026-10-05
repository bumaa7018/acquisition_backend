//go:build e2e

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"acquisition_backend/internal/legacyimport"
	"acquisition_backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// HTTP давхарга: multipart файл, JSON хэлбэр (frontend-ийн api.ts), admin эрх.
func TestE2ELegacyImportHTTP(t *testing.T) {
	dsn := os.Getenv("E2E_DATABASE_URL")
	samples := filepath.Join("..", "..", "..", "..", "government-legacy-import", "input", "чөлөөлөлт")
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL тохируулаагүй")
	}
	if _, err := os.Stat(samples); err != nil {
		t.Skip("жишээ файл алга")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close) // t.Cleanup LIFO — цэвэрлэгээний дараа хаагдана

	gus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"code": "P-1", "parcel_id": "4932803",
			"gazner": "HTTP тест", "geometry_wkt": "POLYGON((106.7 47.85,106.95 47.85,106.95 48,106.7 48,106.7 47.85))"}})
	}))
	defer gus.Close()

	var catID int
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO acquisition_category(name,parent_id,sort_order)
		VALUES('HTTP e2e', NULL, 98) RETURNING id`).Scan(&catID))
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(c, `DELETE FROM legacy_import_stage.rollback_log WHERE session_id IN (SELECT id FROM legacy_import_stage.session WHERE created_by = 'HTTP тестер')`)
		_, _ = pool.Exec(c, `DELETE FROM legacy_import_stage.session WHERE created_by = 'HTTP тестер'`)
		_, _ = pool.Exec(c, `DELETE FROM acquisition_category WHERE id = $1`, catID)
	})

	svc := legacyimport.NewService(pool, legacyimport.GusConfig{BaseURL: gus.URL, Username: "u", Password: "p", Delay: time.Millisecond})
	h := NewLegacyImportHandler(svc)
	gin.SetMode(gin.TestMode)
	newRouter := func(roles ...string) *gin.Engine {
		r := gin.New()
		g := r.Group("/legacy-import", func(c *gin.Context) {
			c.Set(middleware.CtxRoles, roles)
			c.Set(middleware.CtxUserName, "HTTP тестер")
			c.Next()
		}, middleware.RequireAnyRole("admin"))
		g.GET("/categories", h.Categories)
		g.POST("/sessions", h.Create)
		g.GET("/sessions/:id", h.Get)
		g.POST("/sessions/:id/commit", h.Commit)
		g.GET("/sessions/:id/progress", h.Progress)
		g.GET("/sessions/:id/rollback", h.RollbackPreview)
		g.POST("/sessions/:id/rollback", h.Rollback)
		g.POST("/sessions/:id/:step", h.Upload)
		return r
	}
	admin, specialist := newRouter("Админ"), newRouter("specialist")
	do := func(r *gin.Engine, method, path string, body *bytes.Buffer, contentType string) (int, map[string]any) {
		if body == nil {
			body = &bytes.Buffer{}
		}
		req := httptest.NewRequest(method, path, body)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	code, _ := do(specialist, http.MethodPost, "/legacy-import/sessions", nil, "")
	require.Equal(t, http.StatusForbidden, code, "админ биш хэрэглэгч импорт үүсгэхгүй")

	code, body := do(admin, http.MethodPost, "/legacy-import/sessions", nil, "")
	require.Equal(t, http.StatusCreated, code)
	id := body["data"].(map[string]any)["id"].(string)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("category", strconv.Itoa(catID)))
	entries, _ := os.ReadDir(samples)
	for _, e := range entries {
		data, _ := os.ReadFile(filepath.Join(samples, e.Name()))
		fw, _ := mw.CreateFormFile("files", e.Name())
		_, _ = fw.Write(data)
	}
	require.NoError(t, mw.Close())
	code, body = do(admin, http.MethodPost, "/legacy-import/sessions/"+id+"/boundary", &buf, mw.FormDataContentType())
	require.Equal(t, http.StatusOK, code, "%v", body)
	boundary := body["data"].(map[string]any)["boundary"].(map[string]any)
	require.Equal(t, true, boundary["canProceed"], "%v", boundary)
	row := boundary["rows"].([]any)[0].(map[string]any)
	require.Equal(t, "4932803", row["planParcelID"])
	require.Equal(t, 100.0, row["insidePlanPct"])
	require.NotEmpty(t, boundary["map"])

	code, body = do(admin, http.MethodPost, "/legacy-import/sessions/"+id+"/reports", nil, "multipart/form-data; boundary=x")
	require.Equal(t, http.StatusBadRequest, code, "%v", body)

	code, body = do(admin, http.MethodPost, "/legacy-import/sessions/"+id+"/commit", nil, "")
	require.Equal(t, http.StatusConflict, code, "нэгж талбаргүй бол «Оруулах» хаалттай: %v", body)
	require.Contains(t, body["error"], "Нэгж талбарын файл оруулаагүй")

	code, body = do(admin, http.MethodGet, "/legacy-import/sessions/"+id+"/progress", nil, "")
	require.Equal(t, http.StatusOK, code, "%v", body)
	require.Equal(t, "draft", body["data"].(map[string]any)["status"])
	code, _ = do(specialist, http.MethodGet, "/legacy-import/sessions/"+id+"/progress", nil, "")
	require.Equal(t, http.StatusForbidden, code)

	// «Зөвхөн үнэлгээ» горим: JSON body-оор үүсгэнэ, хил/нэгж талбар хүлээн авахгүй, буруу горим 400.
	code, body = do(admin, http.MethodPost, "/legacy-import/sessions", bytes.NewBufferString(`{"mode":"valuation"}`), "application/json")
	require.Equal(t, http.StatusCreated, code, "%v", body)
	valID := body["data"].(map[string]any)["id"].(string)
	code, body = do(admin, http.MethodGet, "/legacy-import/sessions/"+valID, nil, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "valuation", body["data"].(map[string]any)["session"].(map[string]any)["mode"])
	var vbuf bytes.Buffer
	vmw := multipart.NewWriter(&vbuf)
	fw, _ := vmw.CreateFormFile("files", "x.shp")
	_, _ = fw.Write([]byte("x"))
	require.NoError(t, vmw.Close())
	code, body = do(admin, http.MethodPost, "/legacy-import/sessions/"+valID+"/boundary", &vbuf, vmw.FormDataContentType())
	require.Equal(t, http.StatusConflict, code, "%v", body)
	require.Contains(t, body["error"], "Зөвхөн үнэлгээ")
	code, _ = do(admin, http.MethodPost, "/legacy-import/sessions", bytes.NewBufferString(`{"mode":"bogus"}`), "application/json")
	require.Equal(t, http.StatusBadRequest, code, "буруу горим")

	// «Хасах»: баталгаажуулалтгүй бол 400, оруулаагүй импортыг хасахгүй (409), админ биш 403.
	code, body = do(admin, http.MethodGet, "/legacy-import/sessions/"+id+"/rollback", nil, "")
	require.Equal(t, http.StatusOK, code, "%v", body)
	require.Equal(t, false, body["data"].(map[string]any)["canRollback"])
	code, _ = do(admin, http.MethodPost, "/legacy-import/sessions/"+id+"/rollback", bytes.NewBufferString(`{}`), "application/json")
	require.Equal(t, http.StatusBadRequest, code)
	code, body = do(admin, http.MethodPost, "/legacy-import/sessions/"+id+"/rollback", bytes.NewBufferString(`{"confirm":"`+id+`"}`), "application/json")
	require.Equal(t, http.StatusConflict, code, "%v", body)
	code, _ = do(specialist, http.MethodPost, "/legacy-import/sessions/"+id+"/rollback", bytes.NewBufferString(`{"confirm":"`+id+`"}`), "application/json")
	require.Equal(t, http.StatusForbidden, code)
}
