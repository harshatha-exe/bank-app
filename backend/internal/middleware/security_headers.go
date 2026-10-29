package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		csp := "default-src 'none'; frame-ancestors 'none'"
		if len(c.Request.URL.Path) >= len("/swagger/") && c.Request.URL.Path[:len("/swagger/")] == "/swagger/" {
			csp = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'"
		}
		c.Header("Content-Security-Policy", csp)
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}
