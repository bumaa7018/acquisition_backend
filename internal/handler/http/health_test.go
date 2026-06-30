package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	handler "acquisition_backend/internal/handler/http"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Live(t *testing.T) {
	h := handler.NewHealthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Live(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
