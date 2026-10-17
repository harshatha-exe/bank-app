package middleware

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("panic recovered: %v", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Internal server error."}, "meta": gin.H{"correlationId": c.GetString("correlation_id")}})
	})
}
