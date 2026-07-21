package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency", time.Since(start),
			"ip", c.ClientIP(),
		}

		switch {
		case status >= http.StatusInternalServerError:
			slog.Error("request", append(attrs, "result", "fail")...)
		case status >= http.StatusBadRequest:
			slog.Warn("request", append(attrs, "result", "fail")...)
		default:
			slog.Info("request", append(attrs, "result", "success")...)
		}
	}
}
