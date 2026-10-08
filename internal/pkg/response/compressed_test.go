package response

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOKCompressed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) { OKCompressed(c, map[string]int{"n": 42}) })

	// gzip дэмжвэл — шахсан JSON, ердийн бүтэцтэй.
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	zr, err := gzip.NewReader(w.Body)
	require.NoError(t, err)
	var body struct {
		Code int            `json:"code"`
		Data map[string]int `json:"data"`
	}
	require.NoError(t, json.NewDecoder(zr).Decode(&body))
	require.Equal(t, 200, body.Code)
	require.Equal(t, 42, body.Data["n"])

	// Дэмжихгүй бол — энгийн JSON.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Empty(t, w.Header().Get("Content-Encoding"))
	require.Contains(t, w.Body.String(), `"n":42`)
}
