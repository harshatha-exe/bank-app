package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ---------- logging: JSON, level from LOG_LEVEL, secrets redacted ----------

var sensitive = map[string]bool{"password": true, "token": true, "secret": true,
	"authorization": true, "cookie": true, "session_id": true, "csrf_token": true}

// InitLogger also routes the standard `log` package through slog, so the
// existing log.Printf calls become JSON lines with no code changes.
func InitLogger() {
	level := slog.LevelInfo
	_ = level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))) // invalid/empty keeps info
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if sensitive[strings.ToLower(a.Key)] { // defense in depth: a secret can't leak by accident
				a.Value = slog.StringValue("[REDACTED]")
			}
			return a
		},
	})))
}

// ---------- traces + metrics: no-op unless an OTLP endpoint is configured ----------

// Setup configures everything from standard OTEL_* env vars (see .env.example).
// The returned func flushes buffered data; call it during shutdown.
func Setup(ctx context.Context, service, version string) (func(context.Context) error, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		slog.Info("telemetry export disabled: OTEL_EXPORTER_OTLP_ENDPOINT not set")
		return func(context.Context) error { return nil }, nil
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(attribute.String("service.name", service), attribute.String("service.version", version)),
		resource.WithFromEnv(), resource.WithTelemetrySDK())
	if err != nil {
		return nil, err
	}
	traceExp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	metricExp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExp), sdktrace.WithResource(res))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(30*time.Second))),
		sdkmetric.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx)) // flush spans + metrics
	}, nil
}

// ---------- metrics ----------

var (
	meter           = otel.Meter("bank-api")
	httpDuration, _ = meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	auditEvents, _  = meter.Int64Counter("bank.audit.events")
	transfers, _    = meter.Int64Counter("bank.transfers")
)

// HTTPMetrics records latency per method, ROUTE TEMPLATE and status.
// Never use the raw URL as an attribute: unbounded values blow up metric cost.
func HTTPMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		httpDuration.Record(c.Request.Context(), time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", c.Writer.Status())))
	}
}

// CountAudit: one counter covers logins, denials, rate limits, transfer failures.
func CountAudit(ctx context.Context, eventType, result string) {
	auditEvents.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType), attribute.String("result", result)))
}

func CountTransfer(ctx context.Context, result string) {
	transfers.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}
