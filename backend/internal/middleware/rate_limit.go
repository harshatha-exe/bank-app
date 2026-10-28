package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	loginIPLimit = 20
	maxKeys      = 100000
)

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
	clock    func() time.Time
	audit    *audit.Service
	stop     chan struct{}
	done     chan struct{}
	kind     rateLimitKind
}

type rateLimitKind uint8

const (
	authLimiter rateLimitKind = iota
	transferLimiter
)

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return newRateLimiter(max, window, time.Now, nil, authLimiter)
}

func NewRateLimiterWithClock(max int, window time.Duration, clock func() time.Time, auditService *audit.Service) *RateLimiter {
	return newRateLimiter(max, window, clock, auditService, authLimiter)
}

func NewTransferRateLimiter(max int, window time.Duration, auditService *audit.Service) *RateLimiter {
	return newRateLimiter(max, window, time.Now, auditService, transferLimiter)
}

func newRateLimiter(max int, window time.Duration, clock func() time.Time, auditService *audit.Service, kind rateLimitKind) *RateLimiter {
	l := &RateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
		clock:    clock,
		audit:    auditService,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		kind:     kind,
	}
	go l.janitor()
	return l
}

func (l *RateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost || !l.applies(c.Request.URL.Path) {
			c.Next()
			return
		}

		keys := l.keys(c)
		now := l.clock()
		if !l.allow(keys, now) {
			retryAfter := l.retryAfter(keys, now)
			c.Header("Retry-After", retryAfter)
			l.record(c)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "Too many requests."},
				"meta":  gin.H{"correlationId": c.GetString("correlation_id")},
			})
			return
		}
		c.Next()
	}
}

func (l *RateLimiter) applies(path string) bool {
	if l.kind == transferLimiter {
		return path == "/api/transfers"
	}
	return path == "/api/auth/login" || path == "/api/auth/signup"
}

func (l *RateLimiter) keys(c *gin.Context) []string {
	if l.kind == transferLimiter {
		session, ok := httpx.Session(c)
		if !ok {
			return []string{"transfer:anonymous"}
		}
		return []string{"transfer:user:" + session.UserID.String()}
	}
	ip := c.ClientIP()
	if c.Request.URL.Path == "/api/auth/signup" {
		return []string{"signup:ip:" + ip}
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	username := ""
	if err == nil {
		var request struct {
			Username string `json:"username"`
		}
		if json.Unmarshal(body, &request) == nil {
			username = strings.ToLower(strings.TrimSpace(request.Username))
		}
	}
	if username == "" {
		return []string{"login:ip:" + ip}
	}
	return []string{"login:ip:" + ip + ":username:" + username, "login:ip:" + ip}
}

func (l *RateLimiter) allow(keys []string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	allowed := true
	for _, key := range keys {
		values := l.pruneLocked(key, now)
		if len(values) >= l.limitFor(key) {
			allowed = false
		}
	}
	if !allowed {
		return false
	}
	for _, key := range keys {
		l.attempts[key] = append(l.attempts[key], now)
	}
	l.enforceCapLocked()
	return true
}

func (l *RateLimiter) limitFor(key string) int {
	if l.kind == authLimiter && strings.HasPrefix(key, "login:ip:") && !strings.Contains(key, ":username:") {
		return loginIPLimit
	}
	return l.max
}

func (l *RateLimiter) pruneLocked(key string, now time.Time) []time.Time {
	values := l.attempts[key]
	cutoff := now.Add(-l.window)
	kept := values[:0]
	for _, value := range values {
		if value.After(cutoff) {
			kept = append(kept, value)
		}
	}
	if len(kept) == 0 {
		delete(l.attempts, key)
		return nil
	}
	l.attempts[key] = kept
	return kept
}

func (l *RateLimiter) retryAfter(keys []string, now time.Time) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var latest time.Time
	for _, key := range keys {
		values := l.attempts[key]
		if len(values) > 0 && (latest.IsZero() || values[0].After(latest)) {
			latest = values[0]
		}
	}
	seconds := int(math.Ceil((l.window - now.Sub(latest)).Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

func (l *RateLimiter) record(c *gin.Context) {
	if l.audit == nil {
		return
	}
	var actor *uuid.UUID
	if session, ok := httpx.Session(c); ok {
		actor = &session.UserID
	}
	l.audit.Record(c, actor, "RATE_LIMITED", c.Request.URL.Path, "FAILURE", c.ClientIP(), httpx.CorrelationID(c), nil)
}

func (l *RateLimiter) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	defer close(l.done)
	for {
		select {
		case now := <-ticker.C:
			l.mu.Lock()
			l.janitorTick(now)
			l.mu.Unlock()
		case <-l.stop:
			return
		}
	}
}

func (l *RateLimiter) janitorTick(now time.Time) {
	for key := range l.attempts {
		l.pruneLocked(key, now)
	}
}

func (l *RateLimiter) enforceCapLocked() {
	if len(l.attempts) <= maxKeys {
		return
	}
	type oldestKey struct {
		key string
		at  time.Time
	}
	keys := make([]oldestKey, 0, len(l.attempts))
	for key, values := range l.attempts {
		if len(values) > 0 {
			keys = append(keys, oldestKey{key, values[0]})
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].at.Before(keys[j].at) })
	for len(l.attempts) > maxKeys && len(keys) > 0 {
		delete(l.attempts, keys[0].key)
		keys = keys[1:]
	}
}

func (l *RateLimiter) Stop() {
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
	<-l.done
}
