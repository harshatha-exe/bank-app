# Observability

> **In one minute:** The API writes JSON logs, sends traces and metrics to New Relic (or any OTLP backend), and tags every request with a **correlation ID**. Quote that ID from any error message to find the matching log line, trace and audit row.

Code: `internal/telemetry`, `internal/middleware/{correlation_id,logging}.go`, `cmd/server/main.go`.

## 1. How the signals flow

```mermaid
flowchart LR
    REQ["HTTP request"] --> API["Go API"]
    API -->|"JSON lines, stdout"| LOG["Log collector (optional)"]
    API -->|"spans via otelgin + otelpgx"| OTLP["OTLP over HTTP"]
    API -->|"metrics every 30 s"| OTLP
    OTLP --> NR["New Relic"]
    API -->|"correlation id"| AUD[("audit_events")]
    API -->|"correlation id in error body"| USER["User / frontend"]
```

Export is a **no-op unless `OTEL_EXPORTER_OTLP_ENDPOINT` is set**, so local development needs nothing.

## 2. One ID to find everything

```mermaid
flowchart TD
    A["CorrelationID middleware creates a UUID"] --> B["Log line field: correlation_id"]
    A --> C["Span attribute: correlation_id"]
    A --> D["Error response body"]
    A --> E["X-Correlation-ID response header"]
    A --> F["audit_events.correlation_id"]
```

When a user reports an error, ask for the correlation ID. The frontend shows it next to backend errors.

## 3. Logging

- Level from `LOG_LEVEL` (`debug`, `info`, `warn`, `error`); empty or invalid means `info`.
- Standard `log` calls are routed through slog, so they also come out as JSON.
- Secrets are redacted by key: `password`, `token`, `secret`, `authorization`, `cookie`, `session_id`, `csrf_token`.
- One `request` line per call (probes `/health` and `/ready` are skipped). 5xx logs at `ERROR`, 4xx at `WARN`, the rest at `INFO`.

| Field | Meaning |
|---|---|
| `method`, `route`, `status`, `duration_ms` | `route` is the template (`/api/accounts/:accountId`), never the raw URL |
| `correlation_id` | Per-request id |
| `trace_id` | Present when a trace is active, links logs to traces |
| `client_ip` | Honors `TRUSTED_PROXIES` |
| `user_id`, `role` | Present when authenticated |

## 4. Traces

- `otelgin` creates one span per request (health and ready probes are filtered out).
- `otelpgx` adds one span per SQL statement under it, so slow queries show up inside their request.
- W3C trace context and baggage are propagated.
- Sampling uses the standard env vars `OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG`.

## 5. Metrics

| Metric | Type | Attributes |
|---|---|---|
| `http.server.request.duration` | histogram (seconds) | `http.request.method`, `http.route`, `http.response.status_code` |
| `bank.audit.events` | counter | `event_type`, `result` (logins, denials, rate limits) |
| `bank.transfers` | counter | `result` (`completed` or the failure code) |

Only route templates are used as attributes, which keeps the number of series bounded.

## 6. New Relic setup

Add to `.env` (see `.env.example`):

```sh
OTEL_SERVICE_NAME=bank-api
OTEL_EXPORTER_OTLP_ENDPOINT=<your New Relic OTLP endpoint>
OTEL_EXPORTER_OTLP_HEADERS=api-key=<NEW_RELIC_INGEST_LICENSE_KEY>
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=delta
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

Use the US or EU endpoint that matches your New Relic account. Never commit the key.

## 7. Health checks

```mermaid
flowchart TD
    P["Probe"] --> H["GET /health"]
    P --> RDY["GET /ready"]
    H --> H1["200 status ok, while the process runs"]
    RDY --> Q{"DB ping within 2 s and not draining?"}
    Q -->|yes| R1["200 status ready"]
    Q -->|no| R2["503 status unavailable"]
```

```sh
curl -s localhost:8080/health
curl -s localhost:8080/ready
```

## 8. Graceful shutdown

```mermaid
sequenceDiagram
    participant OS as OS or Docker
    participant App as API
    participant LB as Load balancer
    participant DB as PostgreSQL

    OS->>App: SIGINT or SIGTERM
    App->>App: draining = true
    LB->>App: GET /ready
    App-->>LB: 503, stop sending traffic
    App->>App: Shutdown, finish in-flight requests (max 20 s)
    App->>App: Flush spans and metrics
    App->>DB: Close pool after requests finish
```

Server timeouts: read header 5 s, read 10 s, write 15 s, idle 60 s.

## 9. Suggested alerts

- `/ready` failing for more than 1 minute
- 5xx rate above 2 percent for 5 minutes
- p95 request duration above 1 s
- Spike in `bank.audit.events` with `LOGIN_FAILURE` or `RATE_LIMITED`
- Rising `bank.transfers` failures