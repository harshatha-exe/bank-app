# Workflow

> **In one minute:** Branch, change, open a pull request, let CI check it, merge to `main`. To add an endpoint, work from the database up (repository, service, handler, route), then tests and docs.

## 1. Git flow

```mermaid
gitGraph
    commit id: "main"
    branch feature-endpoint
    commit id: "code + tests"
    commit id: "docs"
    checkout main
    merge feature-endpoint id: "PR merged, CI green"
```

- `main` is the integration branch. CI runs on every push to `main` and on every pull request.
- Use short-lived branches (`feat/...`, `fix/...`, `docs/...`).
- A new push to the same branch cancels the older CI run.

## 2. CI pipeline (`.github/workflows/ci.yml`)

```mermaid
flowchart LR
    A["Checkout"] --> B["Setup Go from go.mod"]
    B --> C["go mod verify"]
    C --> D["go vet"]
    D --> E["go build"]
    E --> F["migrate up (v4.17.1)"]
    F --> G["go test -race -shuffle=on -count=1"]
    G --> H["migrate down -all, then up"]
    H --> I["Upload coverage.out"]
```

One job (`backend`) with a Postgres 16 service, 15 minute timeout and read-only token permissions.

Prepared but commented out: `gofmt` check, `golangci-lint`, `govulncheck`, production Docker image build. Not in CI at all: frontend install, build and tests.

If you add a required variable to `config.go`, add a throw-away value to the workflow's `env:` block.

## 3. Local development

```sh
cp -n .env.example .env
make db-up && make migrate-up && make run     # API on :8080
cd frontend && npm install && npm run dev      # UI on :5173, proxies /api to :8080
```

| Make target | Does |
|---|---|
| `make db-up` / `db-down` / `logs` | Start, stop, tail the Postgres container |
| `make migrate-up` / `migrate-down` | Apply all, or roll back one migration (needs the `migrate` CLI) |
| `make run` | Load `.env`, run `go run ./cmd/server` |
| `make test` | `go test ./...` in `backend` |
| `cd backend && make swagger` | Regenerate `backend/docs` from annotations |

A `.devcontainer` is provided (Go image, Docker-in-Docker, Node LTS; installs `migrate`, `swag`, `air`). Its ports and extensions still mention Angular and port 4200; remove them.

## 4. Pull request checklist

- [ ] `go vet ./...` and `go test -race ./...` pass
- [ ] New or changed endpoint has swagger annotations and `make swagger` was re-run
- [ ] Authorisation is enforced by the route group **and** by SQL scoping
- [ ] Sensitive actions write an audit event
- [ ] Migration added (and reversible) if the schema changed
- [ ] Tests added, including a negative authorisation case
- [ ] `docs/` updated (`api.md`, `database.md`, and any lifecycle or security page)
- [ ] No secrets, real data or new demo credentials committed

## 5. How to add an endpoint

```mermaid
flowchart TD
    A["1. Models: request and response types"] --> B["2. Repository: parameterised SQL, scoped to the caller"]
    B --> C["3. Service: validate, rules, map errors to apperr"]
    C --> D["4. Handler: BindJSON, call service, httpx.OK or Fail, audit, swagger"]
    D --> E["5. Route in cmd/server/main.go inside the right group"]
    E --> F["6. Tests: unit, API, cross-user security"]
    F --> G["7. make swagger, update api.md"]
    G --> H["8. Frontend: api.js call and page"]
```

Route groups in `main.go`: `protected` (auth + CSRF), then `customer`, `ops` or `adminRoutes`.

## 6. Migration rules

```mermaid
flowchart LR
    N["Need a schema change"] --> W["Add NNN_name.up.sql and NNN_name.down.sql"]
    W --> T["Run up, down, up locally"]
    T --> P["Open PR, CI repeats the round trip"]
    P --> M["Merged: never edit this file again"]
```

- Files live in `backend/migrations/` (golang-migrate). Use the next unused number.
- Every `up` has a working `down`.
- Money columns are `NUMERIC(19,2)`; keep `CHECK` constraints for invariants.
- No real credentials or personal data in seed data. `001_init` still holds demo users and must be cleaned before production ([DEPLOYMENT.md](deployment.md)).

## 7. Errors

Responses use `{ "error": { "code", "message", "correlationId" } }` ([API.md](api.md)). Create new codes through `apperr` (`BadRequest`, `Validation`, `Conflict`, `Unprocessable`, `NotFound`, `Forbidden`) and list them in the endpoint's swagger `@Failure` lines.