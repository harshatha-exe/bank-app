package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/normal", func(c *gin.Context) {})
	router.GET("/swagger/index.html", func(c *gin.Context) {})
	for _, path := range []string{"/normal", "/swagger/index.html"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("%s missing security headers", path)
		}
		wantCSP := "default-src 'none'; frame-ancestors 'none'"
		if path == "/swagger/index.html" {
			wantCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'"
		}
		if rec.Header().Get("Content-Security-Policy") != wantCSP {
			t.Fatalf("%s has unexpected CSP %q", path, rec.Header().Get("Content-Security-Policy"))
		}
	}
}
