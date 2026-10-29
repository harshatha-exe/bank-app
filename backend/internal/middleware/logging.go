package middleware

import (
	"log/slog"
	"time"

	"bank-app-ioc/backend/internal/httpx"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		span := trace.SpanFromContext(c.Request.Context())
		span.SetAttributes(attribute.String("correlation_id", c.GetString("correlation_id")))
		c.Next()

		route := c.FullPath() // template, never the raw path: no log injection, bounded cardinality
		if route == "" {
			route = "unmatched"
		}
		if route == "/health" || route == "/ready" {
			return
		}
		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method), slog.String("route", route), slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("correlation_id", c.GetString("correlation_id")), slog.String("client_ip", c.ClientIP()),
		}
		if sc := span.SpanContext(); sc.IsValid() {
			attrs = append(attrs, slog.String("trace_id", sc.TraceID().String()))
		}
		if s, ok := httpx.Session(c); ok {
			attrs = append(attrs, slog.String("user_id", s.UserID.String()), slog.String("role", s.Role))
		}
		slog.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}
