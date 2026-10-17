package middleware

import (
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("request method=%s path=%s status=%d duration=%s correlation_id=%s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start), c.GetString("correlation_id"))
	}
}
