package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Correlation-ID")
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.NewString()
		}
		c.Set("correlation_id", id)
		c.Header("X-Correlation-ID", id)
		c.Next()
	}
}
