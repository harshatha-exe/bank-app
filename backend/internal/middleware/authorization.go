package middleware

import (
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CustomerOnly(auditService *audit.Service) gin.HandlerFunc {
	return roleOnly(auditService, "CUSTOMER")
}

func OperationsOnly(auditService *audit.Service) gin.HandlerFunc {
	return roleOnly(auditService, "OPERATIONS")
}

func roleOnly(auditService *audit.Service, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := c.Get("session")
		s, valid := session.(auth.Session)
		if !ok || !valid || s.Role != role {
			if valid {
				auditService.Record(c, &s.UserID, "ACCESS_DENIED", c.Request.Method, "FAILURE", c.ClientIP(), correlationUUID(c), nil)
			} else {
				auditService.Record(c, nil, "ACCESS_DENIED", c.Request.Method, "FAILURE", c.ClientIP(), correlationUUID(c), nil)
			}
			c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied."}, "meta": gin.H{"correlationId": c.GetString("correlation_id")}})
			return
		}
		c.Next()
	}
}

func correlationUUID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("correlation_id"))
	return id
}
