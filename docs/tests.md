# Tests

> **In one minute:** Fast unit tests run anywhere. Tests that touch the database only run when `TEST_DATABASE_URL` points at a throw-away `*_test` database. CI currently does **not** set that, so those suites are skipped there.

## 1. The test layers

```mermaid
flowchart TD
    FE["Frontend: Vitest + Testing Library"]
    SEC["Security tests (DB): sessions, CSRF, rate limit, injection, leakage"]
    API["API tests (DB): full router, real HTTP flow"]
    INT["Integration tests (DB): SQL, locks, concurrency, rollback"]
    UNIT["Unit tests: services, config, validation, middleware, DTOs"]
    UNIT --> INT --> API --> SEC
    UNIT --> FE
```

| Layer | Location | Needs DB |
|---|---|---|
| Unit | next to code, `*_test.go` | No |
| Integration | `backend/tests/integration` | Yes |
| API | `backend/tests/api` | Yes |
| Security | `backend/tests/security` | Yes |
| OpenAPI contract | `backend/docs/swagger_test.go` | No |
| Migration round trip | CI job: `down -all`, then `up` | Yes |
| Frontend | `frontend/src/**/*.test.jsx` (jsdom) | No |

## 2. How the DB gate works

```mermaid
flowchart TD
    S["DB-backed test starts"] --> Q1{"TEST_DATABASE_URL set?"}
    Q1 -->|no| SK["Skip test"]
    Q1 -->|yes| Q2{"Same as DATABASE_URL?"}
    Q2 -->|yes| F1["Fail: refuses to touch the real DB"]
    Q2 -->|no| Q3{"Database name ends with _test?"}
    Q3 -->|no| F2["Fail"]
    Q3 -->|yes| Q4{"Reachable?"}
    Q4 -->|no| SK
    Q4 -->|yes| RUN["Run test"]
```

## 3. Running them

```sh
# Quick: unit tests (DB suites skip themselves)
make test                     # cd backend && go test ./...

# Same flags as CI
cd backend && go test -race -shuffle=on -count=1 -coverprofile=coverage.out ./...

# Frontend
cd frontend && npm install && npm test        # vitest run
```

Enable the DB-backed suites:

```sh
make db-up
docker compose exec db psql -U bankuser -d bankdb -c "CREATE DATABASE bankdb_test;"
migrate -path=backend/migrations \
  -database "postgres://bankuser:bankpass@localhost:5432/bankdb_test?sslmode=disable" up

export TEST_DATABASE_URL="postgres://bankuser:bankpass@localhost:5432/bankdb_test?sslmode=disable"
cd backend && go test -race -count=1 ./...
```

Never point `TEST_DATABASE_URL` at a real database: tests insert and delete rows.

## 4. What exists

| Area | Tests |
|---|---|
| Config | `config_test.go`: defaults, required vars, trusted proxies, money parsing |
| Auth | password hash and verify, handler, service |
| Account, beneficiary | service tests |
| Admin | handler, models, repository |
| Operations | `TestAssignedAccountScoping`, `TestReviewStatusValidation`, `TestFreezeUnfreezeIdempotency`, DTO masking |
| Transfer | `TestServiceValidationAndIdempotency`, `TestServiceSuccess`, `TestServiceMapsTransferErrors`, `TestTransferFailureCode`, DTO masking |
| Middleware | CSRF, rate limiter (login buckets, per-user transfers, janitor), security headers, trusted-proxy client IP, admin route authorization through the real chain |
| Validation | `validate_test.go` |
| Repository | `pgerr_test.go`, `admin_repository_test.go` |
| API (DB) | `TestAuthAndSignup`, `TestOperationsAndAdmin`, `TestAccountsBOLAAndTransactions`, `TestBeneficiariesTransfersAndValidation` |
| Integration (DB) | `TestSuccessfulTransferIntegrity`, `TestTransferFailureIdempotencyAndConcurrency`, `TestFrozenAccountsRejected` |
| Security (DB) | `TestInvalidExpiredAndForgedSessions`, `TestRateLimitCSRFInjectionAndLeakage` |
| OpenAPI | `swagger_test.go` |
| Frontend | `AdminUsers.test.jsx`, `Beneficiaries.test.jsx`, `SendMoney.test.jsx` |

## 5. Critical scenarios covered

- A completed transfer moves both balances and writes matching ledger rows.
- A repeated idempotency key never debits twice; a changed payload is rejected.
- Concurrent transfers cannot overdraw an account.
- Frozen accounts are rejected.
- One customer cannot read another's accounts or statements.
- Operations sees only assigned accounts.
- Forged, expired and invalid sessions are rejected.
- CSRF, rate limiting, SQL injection and data leakage checks.

## 6. Gaps

- **Empty placeholder test files** (only a `package` line): `tests/api/{accounts,admin,auth,beneficiaries,operations,signup,transactions,transfers}_api_test.go`, `tests/integration/{concurrency,migrations,repository,rollback}_test.go`, `tests/security/{bola,data_leakage,privilege_escalation,rate_limit,session,sql_injection,xss}_test.go`, `internal/transaction/service_test.go`. The real scenarios live in the combined files above. Split them out or delete the stubs.
- CI does not set `TEST_DATABASE_URL`, so DB-backed suites are skipped there.
- CI runs no frontend install, build or test.
- `make test` does not use `-race`.
- No browser end-to-end tests, no load tests, no coverage threshold (`coverage.out` is only uploaded).