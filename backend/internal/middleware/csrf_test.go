package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func csrfToken(secret, sessionID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRF("test-secret"))
	router.POST("/api/submit", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/api/read", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/auth/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/auth/signup", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	tests := []struct {
		name, token string
		want        int
	}{
		{"missing", "", http.StatusForbidden},
		{"wrong", "wrong", http.StatusForbidden},
		{"correct", csrfToken("test-secret", "session-1"), http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/submit", nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-1"})
			if test.token != "" {
				req.Header.Set("X-CSRF-Token", test.token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != test.want {
				t.Fatalf("got status %d, want %d", rec.Code, test.want)
			}
		})
	}
	for _, path := range []string{"/api/read", "/api/auth/login", "/api/auth/signup"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if path != "/api/read" {
			req = httptest.NewRequest(http.MethodPost, path, nil)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("%s got status %d", path, rec.Code)
		}
	}
}
