package middleware

// Гадаад системийн API-ийн танилт. Гол шалгах зүйлс: түлхүүр тохируулаагүй
// үед гарц ХААЛТТАЙ (санамсаргүй нээлттэй үлдэхгүй), буруу түлхүүрээр
// хандахгүй, зөв түлхүүрээр танигдсан системийн НЭР context-д тавигдана.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func externalTestRouter(keys []ExternalAPIKey) (*gin.Engine, *string) {
	gin.SetMode(gin.TestMode)
	seenClient := new(string)
	r := gin.New()
	r.Use(ExternalAPIKeyAuth(keys))
	r.GET("/external/data", func(c *gin.Context) {
		*seenClient = ExternalClient(c)
		c.Status(http.StatusOK)
	})
	return r, seenClient
}

func externalRequest(r *gin.Engine, header, value string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/external/data", nil)
	if header != "" {
		req.Header.Set(header, value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestExternalAPIKeyAuth_ClosedWhenNoKeysConfigured(t *testing.T) {
	r, _ := externalTestRouter(nil)
	// Тохируулаагүй бол ХЭН Ч хандахгүй — 503 (нээлттэй үлдэхгүй).
	assert.Equal(t, http.StatusServiceUnavailable, externalRequest(r, "", "").Code)
	assert.Equal(t, http.StatusServiceUnavailable, externalRequest(r, "X-API-Key", "anything").Code)
}

func TestExternalAPIKeyAuth_RejectsMissingOrWrongKey(t *testing.T) {
	r, _ := externalTestRouter([]ExternalAPIKey{{Label: "teg", Key: "correct-key"}})

	assert.Equal(t, http.StatusUnauthorized, externalRequest(r, "", "").Code, "толгойгүй")
	assert.Equal(t, http.StatusUnauthorized, externalRequest(r, "X-API-Key", "wrong-key").Code)
	// Урт нь ижил ч агуулга зөрсөн түлхүүр
	assert.Equal(t, http.StatusUnauthorized, externalRequest(r, "X-API-Key", "correct-kex").Code)
}

func TestExternalAPIKeyAuth_AcceptsKeyAndRecordsClient(t *testing.T) {
	r, seen := externalTestRouter([]ExternalAPIKey{
		{Label: "teg", Key: "teg-key"},
		{Key: "unnamed-key"}, // нэргүй → "external"
	})

	assert.Equal(t, http.StatusOK, externalRequest(r, "X-API-Key", "teg-key").Code)
	assert.Equal(t, "teg", *seen)

	// Authorization: Bearer <key> хэлбэрийг ч хүлээн авна
	assert.Equal(t, http.StatusOK, externalRequest(r, "Authorization", "Bearer teg-key").Code)
	assert.Equal(t, "teg", *seen)

	assert.Equal(t, http.StatusOK, externalRequest(r, "X-API-Key", "unnamed-key").Code)
	assert.Equal(t, "external", *seen)
}
