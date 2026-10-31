# Architecture

> **In one minute:** A React app talks to a Go API, and the API talks to PostgreSQL. Every request passes through the same middleware chain, then handler, service and repository. All SQL lives in repositories, and ownership checks happen inside that SQL.

Related: [design.md](design.md) (why), [database.md](database.md) (data), [transfer_lifecycle.md](transfer_lifecycle.md) (money flow), [observability.md](observability.md) (telemetry).

## 1. System context

```mermaid
flowchart LR
    U["User: customer, operations or admin"] --> FE["React app, Vite dev server on 5173"]
    FE -->|"/api proxied, same origin"| API["Go API, Gin on 8080"]
    API -->|"pgx pool, SQL with placeholders"| DB[("PostgreSQL 16")]
    API -->|"OTLP over HTTP: traces and metrics"| NR["New Relic"]
    API -->|"JSON logs on stdout"| LOGS["Log collector, optional"]
    DEV["Developer"] -->|"/swagger"| API
```

The browser only talks to the frontend origin. In development Vite proxies `/api` to the backend, so the browser makes no cross-origin calls and the API needs no CORS configuration. In production a reverse proxy does the same job.

## 2. Layers

```mermaid
flowchart TD
    MW["Middleware: tracing, correlation id, logging, metrics, recovery, security headers, error handler, rate limits, authenticate, CSRF, role gate"]
    H["Handler: reads path and body, takes identity from the session, calls the service, writes the response"]
    S["Service: validation, business rules, error mapping"]
    R["Repository: the only place SQL exists, ownership and scope enforced inside queries"]
    DB[("PostgreSQL: constraints, row locks, transactions")]
    MW --> H --> S --> R --> DB
    SH["Shared packages: apperr, validate, httpx, audit, telemetry"]
    SH -.-> H
    SH -.-> S
    SH -.-> MW
```

| Layer | Does | Never does |
|---|---|---|
| Middleware | Cross-cutting concerns: identity, CSRF, rate limits, headers, logging, errors | Business rules |
| Handler | Parse input, call service, shape output, record some audit events | SQL, business decisions |
| Service | Validate, apply rules, translate repository errors into safe API errors | Touch `gin.Context`, write HTTP |
| Repository | Parameterised SQL, transactions, row locks | Decide permissions in Go after the fact |

Rule of thumb: **ownership is checked inside the SQL** (`WHERE ... AND customer_id = $n`). A request for someone else's object finds no row and returns the same 404 as a missing object.

## 3. Middleware chain

Global middleware runs in this order (registered in `cmd/server/main.go`):

```mermaid
flowchart TD
    A["otelgin: starts the trace span"] --> B["CorrelationID: uuid per request"]
    B --> C["Logging: one structured line per request"]
    C --> D["HTTPMetrics: latency histogram"]
    D --> E["Recovery: panic becomes a generic 500"]
    E --> F["SecurityHeaders"]
    F --> G["ErrorHandler: writes every error response"]
    G --> H["Auth rate limiter: applies only to POST login and signup"]
    H --> I{"Route group"}
    I -->|"public"| P["signup, login, health, ready, swagger"]
    I -->|"protected: /api"| J["Authenticate: session cookie to session"]
    J --> K["CSRF: POST, PATCH, DELETE"]
    K --> L{"Role gate"}
    L -->|"CustomerOnly"| CU["accounts, beneficiaries, transfers"]
    L -->|"OperationsOnly"| OP["operations endpoints"]
    L -->|"AdminOnly"| AD["admin endpoints"]
    CU --> T["POST /transfers also passes the per-user transfer rate limiter"]
```

Notes:
- Tracing is first so every later layer runs inside the span. Metrics sit before Recovery so a panic still records as a 500.
- `GET /api/transfers/{referenceNumber}` is in the protected group but **not** inside a role group. Its handler allows `CUSTOMER` (own transfers) and `OPERATIONS` (assigned scope) and returns `403` for any other role.
- `/health`, `/ready` and `/swagger` are registered outside the protected group, so they need no session.

## 4. Request lifecycle

```mermaid
sequenceDiagram
    participant C as Client
    participant M as Middleware
    participant H as Handler
    participant S as Service
    participant R as Repository
    participant D as PostgreSQL
    participant E as ErrorHandler

    C->>M: HTTP request with session cookie
    M->>M: authenticate, CSRF, role gate
    M->>H: request with session in context
    H->>H: strict JSON bind
    H->>S: call with typed input and user id
    S->>S: clean and validate input
    S->>R: query
    R->>D: SQL with placeholders
    D-->>R: rows or error
    R-->>S: record or sentinel error
    S-->>H: DTO or apperr
    alt success
        H-->>C: 2xx with data and meta.correlationId
    else failure
        H->>E: httpx.Fail(err)
        E-->>C: generic error with correlationId, real cause is logged only
    end
```

## 5. Authentication and sessions

```mermaid
sequenceDiagram
    participant B as Browser
    participant A as API
    participant D as PostgreSQL

    B->>A: POST /api/auth/login
    A->>D: find user, verify Argon2id password
    A->>D: store SHA-256 hash of a new random session id
    A-->>B: Set-Cookie session_id and meta.csrfToken
    B->>A: next request with cookie and X-CSRF-Token
    A->>D: find session hash joined to an ACTIVE user
    A->>D: check idle and absolute expiry, update last active time
    A-->>B: response
```

Deactivation, role change and password reset delete the user's sessions, so access is revoked immediately.

## 6. Startup and shutdown

```mermaid
sequenceDiagram
    participant OS as OS or Docker
    participant App as main
    participant DB as PostgreSQL
    participant NR as OTLP backend

    App->>App: init JSON logger, load config
    App->>DB: open pool
    App->>App: setup telemetry, build repositories, services, handlers, routes
    App->>App: ListenAndServe with timeouts
    OS->>App: SIGINT or SIGTERM
    App->>App: mark draining, ready returns 503
    App->>App: Shutdown, finish in-flight requests, 20 second limit
    App->>NR: flush spans and metrics
    App->>DB: close pool via defer
```

Order matters: stop new traffic, finish in-flight work while the database is still open, flush telemetry, then close the pool. Closing the pool first would break transfers midway.

## 7. Package map

| Package | Responsibility |
|---|---|
| `cmd/server` | Wiring, routes, HTTP server, graceful shutdown |
| `cmd/hashpw` | Small tool that prints an Argon2id hash for a password |
| `config` | Environment parsing and validation (rejects unsafe `TRUSTED_PROXIES`) |
| `database` | pgx pool with OpenTelemetry tracer |
| `internal/auth` | Signup, login, logout, sessions, Argon2id, CSRF token generation |
| `internal/account` | Customer accounts and statement filters |
| `internal/beneficiary` | Beneficiary create, list, activate and deactivate |
| `internal/transfer` | Transfer creation, lookup, listing, reference numbers |
| `internal/operations` | Assignment-scoped accounts, transactions, transfers, review, freeze |
| `internal/admin` | Users, roles, status, password reset, assignments, limit, audit search |
| `internal/audit` | Best-effort audit writer plus metric counter |
| `internal/apperr` | Typed errors: code, status, safe message, hidden cause |
| `internal/httpx` | Strict JSON binding, session accessor, response helpers |
| `internal/validate` | Sanitising and allow-list validators, money parsing |
| `internal/telemetry` | Logger, OTLP setup, metrics |
| `internal/middleware` | All middleware listed above |
| `repository` | All SQL, one file per aggregate |
| `internal/transaction`, `internal/user` | Empty placeholder packages, reserved for later |

## 8. Frontend structure

```mermaid
flowchart TD
    M["main.jsx"] --> APP["App.jsx: routes guarded by role"]
    APP --> AUTH["auth.jsx: session context, csrf token"]
    APP --> CUS["pages/customer: Dashboard, AccountDetail, Beneficiaries, SendMoney, Transfers, TransferDetail"]
    APP --> OPS["pages/ops: OpsAccounts, OpsAccountDetail, OpsTransfers"]
    APP --> ADM["pages/admin: AdminUsers, AdminAssignments, AdminLimit, AdminAudit"]
    CUS --> API["api.js: fetch wrapper"]
    OPS --> API
    ADM --> API
    API -->|"cookie + X-CSRF-Token"| BE["Go API"]
```

The frontend is JavaScript (`.jsx`), React 18 with Tailwind CSS. `components/` holds Layout, the Transactions table and shared UI. `format.js` formats money and dates.

The UI never decides access. It reflects what the backend allows, and every screen shows real backend errors with their correlation id instead of hiding them.
