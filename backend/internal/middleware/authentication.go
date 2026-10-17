package middleware

import (
	"net/http"

	"bank-app-ioc/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func Authenticate(service *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie("session_id")
		if err != nil {
			unauthorized(c)
			return
		}
		session, err := service.Authenticate(c, cookie.Value)
		if err != nil {
			unauthorized(c)
			return
		}
		c.Set("session", session)
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "Authentication required."}, "meta": gin.H{"correlationId": c.GetString("correlation_id")}})
}
