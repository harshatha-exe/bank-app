package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"bank-app-ioc/backend/config"
	"bank-app-ioc/backend/database"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/internal/middleware"
	"bank-app-ioc/backend/internal/telemetry"
	"bank-app-ioc/backend/internal/transfer"
	"bank-app-ioc/backend/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const csrfSecret = "integration-test-csrf-secret"

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if raw == os.Getenv("DATABASE_URL") {
		t.Fatal("TEST_DATABASE_URL must differ from DATABASE_URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid TEST_DATABASE_URL: %v", err)
	}
	if !strings.HasSuffix(path.Base(parsed.Path), "_test") {
		t.Fatalf("TEST_DATABASE_URL database name must end in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx, raw)
	if err != nil {
		t.Fatalf("test database is unreachable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type fixture struct {
	pool                      *pgxpool.Pool
	router                    *gin.Engine
	user, source, destination uuid.UUID
	name                      string
}

func newRouter(t *testing.T, pool *pgxpool.Pool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	authService := auth.NewService(repository.NewUserRepository(pool), repository.NewSessionRepository(pool), time.Hour, 24*time.Hour, 12)
	r := gin.New()
	r.Use(middleware.CorrelationID(), middleware.Logging(), telemetry.HTTPMetrics(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.ErrorHandler())
	limiter := middleware.NewRateLimiterWithClock(50, time.Minute, time.Now, auditService)
	t.Cleanup(limiter.Stop)
	r.Use(limiter.Handler())
	authHandler := auth.NewHandler(authService, auditService, csrfSecret)
	r.POST("/api/auth/login", authHandler.Login)
	transferHandler := transfer.NewHandler(transfer.NewService(repository.NewTransferRepository(pool), 10000000).WithLimitProvider(repository.NewConfigRepository(pool).MaxTransferPaise), auditService)
	protected := r.Group("/api", middleware.Authenticate(authService), middleware.CSRF(csrfSecret))
	customer := protected.Group("", middleware.CustomerOnly(auditService))
	customer.POST("/transfers", transferHandler.Create)
	return r
}

func createFixture(t *testing.T) *fixture {
	pool := testPool(t)
	f := &fixture{pool: pool, name: "integrity-" + uuid.NewString()}
	f.user = uuid.New()
	hash, err := auth.HashPassword("fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `INSERT INTO users(id,username,password_hash,role_id,status) SELECT $1,$2,$3,id,'ACTIVE' FROM roles WHERE name='CUSTOMER'`, f.user, f.name, hash); err != nil {
		t.Fatal(err)
	}
	customer := uuid.New()
	if _, err = pool.Exec(context.Background(), `INSERT INTO customers(id,user_id,customer_number,full_name) VALUES($1,$2,$3,$4)`, customer, f.user, "CUST-"+uuid.NewString()[:16], "Integrity Fixture"); err != nil {
		t.Fatal(err)
	}
	f.source, f.destination = uuid.New(), uuid.New()
	for _, row := range [][3]string{{f.source.String(), "I" + uuid.NewString()[:20], "10000.00"}, {f.destination.String(), "I" + uuid.NewString()[:20], "5000.00"}} {
		uid, _ := uuid.Parse(row[0])
		if _, err = pool.Exec(context.Background(), `INSERT INTO bank_accounts(id,customer_id,account_number,account_type,balance,currency,status) VALUES($1,$2,$3,'SAVINGS',$4,'INR','ACTIVE')`, uid, customer, row[1], row[2]); err != nil {
			t.Fatal(err)
		}
	}
	f.router = newRouter(t, pool)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM audit_events WHERE actor_user_id=$1`, f.user)
		_, _ = pool.Exec(ctx, `DELETE FROM transactions WHERE transfer_id IN (SELECT id FROM fund_transfers WHERE initiated_by=$1)`, f.user)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger_entries WHERE transfer_id IN (SELECT id FROM fund_transfers WHERE initiated_by=$1)`, f.user)
		_, _ = pool.Exec(ctx, `DELETE FROM fund_transfers WHERE initiated_by=$1`, f.user)
		_, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, f.user)
		_, _ = pool.Exec(ctx, `DELETE FROM bank_accounts WHERE id IN ($1,$2)`, f.source, f.destination)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE user_id=$1`, f.user)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, f.user)
	})
	return f
}

func login(t *testing.T, f *fixture) (*http.Cookie, string) {
	data, _ := json.Marshal(map[string]string{"username": f.name, "password": "fixture-password-123"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Meta struct {
			CSRF string `json:"csrfToken"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" {
			return c, body.Meta.CSRF
		}
	}
	t.Fatal("missing session cookie")
	return nil, ""
}

func transferRequest(t *testing.T, f *fixture, cookie *http.Cookie, csrf, key string, amount string) *httptest.ResponseRecorder {
	data, _ := json.Marshal(map[string]any{"sourceAccountId": f.source, "destinationAccountId": f.destination, "amount": amount, "purpose": "fixture"})
	req := httptest.NewRequest(http.MethodPost, "/api/transfers", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func transferData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data
}

var _ = httpx.Response{}

func _config() config.Config { return config.Config{} }
