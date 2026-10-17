package middleware

import (
	"errors"
	"log"

	"bank-app-ioc/backend/internal/apperr"
	"github.com/gin-gonic/gin"
)

// ErrorHandler runs AFTER the handler (code below c.Next()). It is the only
// place that turns errors into JSON, so the shape is identical everywhere.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		var appErr *apperr.Error
		err := c.Errors.Last().Err
		if !errors.As(err, &appErr) { // unknown error => generic 500
			appErr = apperr.Internal(err)
		}

		cid := c.GetString("correlation_id")
		if appErr.Cause != nil { // real reason stays in OUR logs only
			log.Printf("request error correlation_id=%s route=%q cause=%q", cid, c.FullPath(), appErr.Cause.Error())
		}

		body := gin.H{"code": appErr.Code, "message": appErr.Message, "correlationId": cid}
		if appErr.Fields != nil {
			body["fields"] = appErr.Fields
		}
		c.AbortWithStatusJSON(appErr.Status, gin.H{"error": body})
	}
}
