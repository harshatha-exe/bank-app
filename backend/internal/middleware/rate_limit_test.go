package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bank-app-ioc/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRateLimiterLoginBucketsAndClock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	limiter := NewRateLimiterWithClock(5, time.Minute, clock, nil)
	defer limiter.Stop()

	router := gin.New()
	router.Use(limiter.Handler())
	router.POST("/api/auth/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := func(username string) *httptest.ResponseRecorder {
		body := `{"username":"` + username + `","password":"not-a-password"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		req.Header.Set("X-Real-IP", "5.6.7.8")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	for i := 0; i < 5; i++ {
		if got := request("Alice"); got.Code != http.StatusNoContent {
			t.Fatalf("attempt %d status = %d", i+1, got.Code)
		}
	}
	if got := request("Alice"); got.Code != http.StatusTooManyRequests || got.Header().Get("Retry-After") == "" {
		t.Fatalf("sixth same-user attempt = %d, Retry-After=%q", got.Code, got.Header().Get("Retry-After"))
	}
	if got := request("Bob"); got.Code != http.StatusNoContent {
		t.Fatalf("different username status = %d", got.Code)
	}
	now = now.Add(time.Minute + time.Second)
	if got := request("Alice"); got.Code != http.StatusNoContent {
		t.Fatalf("reset window status = %d", got.Code)
	}
}

func TestRateLimiterTransferIsPerUserAndJanitorEvicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter := newRateLimiter(2, time.Minute, func() time.Time { return now }, nil, transferLimiter)
	defer limiter.Stop()

	userRequest := func(user string) *httptest.ResponseRecorder {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("session", testSession(user))
			c.Next()
		}, limiter.Handler())
		router.POST("/api/transfers", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/transfers", nil)
		router.ServeHTTP(recorder, req)
		return recorder
	}
	if userRequest("user-a").Code != http.StatusNoContent || userRequest("user-a").Code != http.StatusNoContent {
		t.Fatal("first user should be allowed twice")
	}
	if userRequest("user-a").Code != http.StatusTooManyRequests {
		t.Fatal("third request for first user should be blocked")
	}
	if userRequest("user-b").Code != http.StatusNoContent {
		t.Fatal("different user should be unaffected")
	}
	now = now.Add(time.Minute + time.Second)
	limiter.mu.Lock()
	limiter.janitorTick(now)
	remaining := len(limiter.attempts)
	limiter.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("attempts after janitor = %d", remaining)
	}
}

func testSession(user string) auth.Session {
	return auth.Session{UserID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(user)), Username: user, Role: "CUSTOMER"}
}
