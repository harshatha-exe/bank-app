package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CSRF uses an HMAC of the session cookie; this is the documented token
// issuance assumption because no separate CSRF storage was specified.
func CSRF(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodDelete {
			if c.Request.URL.Path != "/api/auth/login" && c.Request.URL.Path != "/api/auth/signup" {
				cookie, err := c.Request.Cookie("session_id")
				if err != nil || !validCSRF(cookie.Value, c.GetHeader("X-CSRF-Token"), secret) {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "CSRF_INVALID", "message": "Invalid CSRF token."}, "meta": gin.H{"correlationId": c.GetString("correlation_id")}})
					return
				}
			}
		}
		c.Next()
	}
}
func validCSRF(sessionID, token, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(token))
}
