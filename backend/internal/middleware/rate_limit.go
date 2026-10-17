package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

type limiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func NewRateLimiter(max int, window time.Duration) gin.HandlerFunc {
	l := &limiter{attempts: make(map[string][]time.Time), max: max, window: window}
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost && (c.Request.URL.Path == "/api/auth/login" || c.Request.URL.Path == "/api/auth/signup") {
			key := c.ClientIP() + ":" + c.Request.URL.Path
			now := time.Now()
			l.mu.Lock()
			values := l.attempts[key]
			cutoff := now.Add(-l.window)
			kept := values[:0]
			for _, value := range values {
				if value.After(cutoff) {
					kept = append(kept, value)
				}
			}
			if len(kept) >= l.max {
				l.mu.Unlock()
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "RATE_LIMITED", "message": "Too many requests."}, "meta": gin.H{"correlationId": c.GetString("correlation_id")}})
				return
			}
			l.attempts[key] = append(kept, now)
			l.mu.Unlock()
		}
		c.Next()
	}
}
