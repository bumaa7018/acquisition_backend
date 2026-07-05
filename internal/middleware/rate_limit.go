package middleware

import (
	"net/http"
	"sync"
	"time"

	"acquisition_backend/internal/pkg/i18n"
	"acquisition_backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	windowStart time.Time
	count       int
}

func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	if maxRequests < 1 {
		maxRequests = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	var mu sync.Mutex
	entries := map[string]rateLimitEntry{}
	lastSweep := time.Now()

	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP() + "|" + c.FullPath()
		if c.FullPath() == "" {
			key = c.ClientIP() + "|" + c.Request.URL.Path
		}

		mu.Lock()
		if now.Sub(lastSweep) > window {
			for k, entry := range entries {
				if now.Sub(entry.windowStart) > window {
					delete(entries, k)
				}
			}
			lastSweep = now
		}

		entry := entries[key]
		if entry.windowStart.IsZero() || now.Sub(entry.windowStart) > window {
			entry = rateLimitEntry{windowStart: now}
		}
		entry.count++
		entries[key] = entry
		limited := entry.count > maxRequests
		mu.Unlock()

		if limited {
			response.Abort(c, http.StatusTooManyRequests, i18n.ErrTooManyRequests)
			return
		}
		c.Next()
	}
}
