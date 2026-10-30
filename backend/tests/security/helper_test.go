package security

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
	"bank-app-ioc/backend/internal/account"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/beneficiary"
	"bank-app-ioc/backend/internal/middleware"
	"bank-app-ioc/backend/internal/telemetry"
	"bank-app-ioc/backend/internal/transfer"
	"bank-app-ioc/backend/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const csrfSecret = "security-test-csrf-secret"

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
	pool                 *pgxpool.Pool
	router               *gin.Engine
	user, destUser       uuid.UUID
	account, destination uuid.UUID
	name                 string
}

func newRouter(t *testing.T, pool *pgxpool.Pool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	authService := auth.NewService(repository.NewUserRepository(pool), repository.NewSessionRepository(pool), 30*time.Minute, 24*time.Hour, 12)
	r := gin.New()
	r.Use(middleware.CorrelationID(), middleware.Logging(), telemetry.HTTPMetrics(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.ErrorHandler())
	limiter := middleware.NewRateLimiterWithClock(5, time.Minute, time.Now, auditService)
	t.Cleanup(limiter.Stop)
	r.Use(limiter.Handler())
	authHandler := auth.NewHandler(authService, auditService, csrfSecret)
	r.POST("/api/auth/login", authHandler.Login)
	accountHandler := account.NewHandler(account.NewService(repository.NewAccountRepository(pool)), auditService)
	beneficiaryHandler := beneficiary.NewHandler(beneficiary.NewService(repository.NewBeneficiaryRepository(pool)), auditService)
	configRepo := repository.NewConfigRepository(pool)
	transferHandler := transfer.NewHandler(transfer.NewService(repository.NewTransferRepository(pool), 10000000).WithLimitProvider(configRepo.MaxTransferPaise), auditService)
	protected := r.Group("/api", middleware.Authenticate(authService), middleware.CSRF(csrfSecret))
	customer := protected.Group("", middleware.CustomerOnly(auditService))
	customer.GET("/accounts", accountHandler.List)
	customer.GET("/accounts/:accountId", accountHandler.Get)
	customer.GET("/accounts/:accountId/transactions", accountHandler.Transactions)
	customer.POST("/beneficiaries", beneficiaryHandler.Create)
	customer.GET("/beneficiaries", beneficiaryHandler.List)
	customer.POST("/transfers", transferHandler.Create)
	return r
}

func newFixture(t *testing.T) *fixture {
	pool := testPool(t)
	f := &fixture{pool: pool, name: "security-" + uuid.NewString(), user: uuid.New(), destUser: uuid.New()}
	hash, err := auth.HashPassword("fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `INSERT INTO users(id,username,password_hash,role_id,status) SELECT $1,$2,$3,id,'ACTIVE' FROM roles WHERE name='CUSTOMER'`, f.user, f.name, hash); err != nil {
		t.Fatal(err)
	}
	customer := uuid.New()
	if _, err = pool.Exec(context.Background(), `INSERT INTO customers(id,user_id,customer_number,full_name) VALUES($1,$2,$3,'Security Fixture')`, customer, f.user, "CUST-"+uuid.NewString()[:16]); err != nil {
		t.Fatal(err)
	}
	destName := f.name + "-dest"
	if _, err = pool.Exec(context.Background(), `INSERT INTO users(id,username,password_hash,role_id,status) SELECT $1,$2,$3,id,'ACTIVE' FROM roles WHERE name='CUSTOMER'`, f.destUser, destName, hash); err != nil {
		t.Fatal(err)
	}
	destCustomer := uuid.New()
	if _, err = pool.Exec(context.Background(), `INSERT INTO customers(id,user_id,customer_number,full_name) VALUES($1,$2,$3,'Security Fixture Destination')`, destCustomer, f.destUser, "CUST-"+uuid.NewString()[:16]); err != nil {
		t.Fatal(err)
	}
	f.account, f.destination = uuid.New(), uuid.New()
	if _, err = pool.Exec(context.Background(), `INSERT INTO bank_accounts(id,customer_id,account_number,account_type,balance,currency,status) VALUES($1,$2,$3,'SAVINGS',1000,'INR','ACTIVE')`, f.account, customer, "S"+uuid.NewString()[:20]); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `INSERT INTO bank_accounts(id,customer_id,account_number,account_type,balance,currency,status) VALUES($1,$2,$3,'SAVINGS',1000,'INR','ACTIVE')`, f.destination, destCustomer, "S"+uuid.NewString()[:20]); err != nil {
		t.Fatal(err)
	}
	f.router = newRouter(t, pool)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM audit_events WHERE actor_user_id IN ($1,$2)`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM transactions WHERE transfer_id IN (SELECT id FROM fund_transfers WHERE initiated_by IN ($1,$2))`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM ledger_entries WHERE transfer_id IN (SELECT id FROM fund_transfers WHERE initiated_by IN ($1,$2))`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM fund_transfers WHERE initiated_by IN ($1,$2)`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM beneficiaries WHERE customer_id IN (SELECT id FROM customers WHERE user_id IN ($1,$2))`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE user_id IN ($1,$2)`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM bank_accounts WHERE id IN ($1,$2)`, f.account, f.destination)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE user_id IN ($1,$2)`, f.user, f.destUser)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1,$2)`, f.user, f.destUser)
	})
	return f
}

func login(t *testing.T, f *fixture) (*http.Cookie, string) {
	data, _ := json.Marshal(map[string]string{"username": f.name, "password": "fixture-password-123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
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

func req(t *testing.T, f *fixture, method, target string, body any, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	request := httptest.NewRequest(method, target, bytes.NewReader(data))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, request)
	return rec
}

func transferReq(t *testing.T, f *fixture, cookie *http.Cookie, csrf, key, purpose string) *httptest.ResponseRecorder {
	t.Helper()
	return reqWithHeaders(t, f, http.MethodPost, "/api/transfers", map[string]any{
		"sourceAccountId": f.account, "destinationAccountId": f.destination, "amount": "1.00", "purpose": purpose,
	}, cookie, csrf, map[string]string{"Idempotency-Key": key})
}

func reqWithHeaders(t *testing.T, f *fixture, method, target string, body any, cookie *http.Cookie, csrf string, headers map[string]string) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	request := httptest.NewRequest(method, target, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, request)
	return rec
}

var _ = config.Config{}
