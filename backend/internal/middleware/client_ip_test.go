package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClientIPTrustedProxyBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clientIP := func(trusted []string) string {
		router := gin.New()
		if err := router.SetTrustedProxies(trusted); err != nil {
			t.Fatal(err)
		}
		router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:4321"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		req.Header.Set("X-Real-IP", "5.6.7.8")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder.Body.String()
	}
	if got := clientIP(nil); got != "192.0.2.10" {
		t.Fatalf("untrusted proxy client IP = %q", got)
	}
	if got := clientIP([]string{"192.0.2.0/24"}); got != "1.2.3.4" {
		t.Fatalf("trusted proxy client IP = %q", got)
	}
}
