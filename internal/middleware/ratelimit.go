package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter is a simple fixed-window per-client limiter.
// For multi-instance production, back this with Redis; the interface stays same.
type rateLimiter struct {
	mu       sync.Mutex
	counters map[string]*window
	limit    int
	window   time.Duration
}

type window struct {
	count int
	reset time.Time
}

// RateLimit limits requests per client IP within a fixed window.
func RateLimit(requests, windowSec int) gin.HandlerFunc {
	if requests <= 0 {
		requests = 100
	}
	if windowSec <= 0 {
		windowSec = 60
	}
	rl := &rateLimiter{
		counters: make(map[string]*window),
		limit:    requests,
		window:   time.Duration(windowSec) * time.Second,
	}

	// Periodic cleanup of stale windows.
	go func() {
		ticker := time.NewTicker(rl.window)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			rl.mu.Lock()
			for k, w := range rl.counters {
				if now.After(w.reset) {
					delete(rl.counters, k)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		w, ok := rl.counters[key]
		if !ok || now.After(w.reset) {
			w = &window{count: 0, reset: now.Add(rl.window)}
			rl.counters[key] = w
		}
		w.count++
		count := w.count
		reset := w.reset
		rl.mu.Unlock()

		if count > rl.limit {
			c.Header("Retry-After", reset.UTC().Format(http.TimeFormat))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
