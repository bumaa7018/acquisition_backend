package http

import "github.com/gin-gonic/gin"

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
