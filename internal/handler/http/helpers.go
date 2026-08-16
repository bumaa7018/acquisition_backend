package http

import (
	"net/http"
	"strconv"
	"strings"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

const maxPageSize = 200

func queryInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 {
		return def
	}
	if key == "page_size" && n > maxPageSize {
		return maxPageSize
	}
	return n
}

// parseID — sdplatform схемийн таних тэмдэг нь serial4 тул зам/query-гийн
// утгыг бүхэл тоо болгож хөрвүүлнэ.
func parseID(raw string) (domain.ID, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || v <= 0 {
		return 0, false
	}
	return domain.ID(v), true
}

// parseIDParam — :id зам параметрийг уншиж, буруу бол 400 буцаана.
func parseIDParam(c *gin.Context) (domain.ID, bool) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		response.Abort(c, http.StatusBadRequest, i18n.ErrInvalidUUID)
		return 0, false
	}
	return id, true
}
