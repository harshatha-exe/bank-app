package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"bank-app-ioc/backend/internal/admin"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/beneficiary"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/internal/middleware"
	"bank-app-ioc/backend/internal/operations"
	"bank-app-ioc/backend/internal/telemetry"
	"bank-app-ioc/backend/internal/transfer"
	"bank-app-ioc/backend/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const csrfSecret = "api-test-csrf-secret"

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
	pool                                 *pgxpool.Pool
	router                               *gin.Engine
	oneName, twoName, opsName, adminName string
	one, two, ops, admin                 uuid.UUID
	oneAccount, twoAccount, destination  uuid.UUID
}

func newRouter(t *testing.T, pool *pgxpool.Pool) (*gin.Engine, *auth.Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	sessions := repository.NewSessionRepository(pool)
	users := repository.NewUserRepository(pool)
	authService := auth.NewService(users, sessions, 30*time.Minute, 24*time.Hour, 12)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	configRepo := repository.NewConfigRepository(pool)
	r := gin.New()
	r.Use(middleware.CorrelationID(), middleware.Logging(), telemetry.HTTPMetrics(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.ErrorHandler())
	loginLimiter := middleware.NewRateLimiterWithClock(5, time.Minute, time.Now, auditService)
	t.Cleanup(loginLimiter.Stop)
	r.Use(loginLimiter.Handler())
	authHandler := auth.NewHandler(authService, auditService, csrfSecret)
	r.POST("/api/auth/signup", authHandler.Signup)
	r.POST("/api/auth/login", authHandler.Login)
	protected := r.Group("/api", middleware.Authenticate(authService), middleware.CSRF(csrfSecret))
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/me", authHandler.Me)
	accountHandler := account.NewHandler(account.NewService(repository.NewAccountRepository(pool)), auditService)
	beneficiaryHandler := beneficiary.NewHandler(beneficiary.NewService(repository.NewBeneficiaryRepository(pool)), auditService)
	transferHandler := transfer.NewHandler(transfer.NewServiceFromConfig(repository.NewTransferRepository(pool), config.Config{MaxTransferPaise: 10000000}).WithLimitProvider(configRepo.MaxTransferPaise), auditService)
	opsHandler := operations.NewHandler(operations.NewService(repository.NewAccountRepository(pool), repository.NewAssignmentRepository(pool), repository.NewTransferRepository(pool)), auditService)
	adminHandler := admin.NewHandler(admin.NewService(repository.NewAdminRepository(pool), repository.NewAssignmentRepository(pool), configRepo, repository.NewAuditQueryRepository(pool), 12))
	customer := protected.Group("", middleware.CustomerOnly(auditService))
	customer.GET("/accounts", accountHandler.List)
	customer.GET("/accounts/:accountId", accountHandler.Get)
	customer.GET("/accounts/:accountId/transactions", accountHandler.Transactions)
	customer.POST("/beneficiaries", beneficiaryHandler.Create)
	customer.GET("/beneficiaries", beneficiaryHandler.List)
	customer.PATCH("/beneficiaries/:beneficiaryId/status", beneficiaryHandler.UpdateStatus)
	customer.POST("/transfers", transferHandler.Create)
	customer.GET("/transfers", transferHandler.List)
	protected.GET("/transfers/:referenceNumber", transferHandler.Get)
	ops := protected.Group("/operations", middleware.OperationsOnly(auditService))
	ops.GET("/accounts", opsHandler.ListAccounts)
	ops.GET("/accounts/:accountId/transactions", opsHandler.Transactions)
	ops.GET("/transfers", opsHandler.ListTransfers)
	ops.PATCH("/transfers/:reference/review", opsHandler.ReviewTransfer)
	ops.POST("/accounts/:accountId/freeze", opsHandler.Freeze)
	ops.POST("/accounts/:accountId/unfreeze", opsHandler.Unfreeze)
	adminRoutes := protected.Group("/admin", middleware.AdminOnly(auditService))
	adminRoutes.POST("/users", adminHandler.CreateUser)
	adminRoutes.GET("/users", adminHandler.ListUsers)
	adminRoutes.PATCH("/users/:userId/role", adminHandler.ChangeRole)
	adminRoutes.POST("/users/:userId/reset-password", adminHandler.ResetPassword)
	adminRoutes.POST("/account-assignments", adminHandler.AssignAccount)
	adminRoutes.GET("/config/transaction-limit", adminHandler.GetLimit)
	return r, authService
}

func request(t *testing.T, r http.Handler, method, url string, body any, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, url, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func jsonBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, rec.Body.String())
	}
	return body
}

func login(t *testing.T, f *fixture, username, password string) (*http.Cookie, string) {
	rec := request(t, f.router, http.MethodPost, "/api/auth/login", map[string]string{"username": username, "password": password}, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", username, rec.Code, rec.Body.String())
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
	t.Fatal("login did not set session cookie")
	return nil, ""
}

func createCustomer(t *testing.T, f *fixture, username string) uuid.UUID {
	rec := request(t, f.router, http.MethodPost, "/api/auth/signup", map[string]any{"username": username, "password": "fixture-password-123", "fullName": username}, nil, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", rec.Code, rec.Body.String())
	}
	var id uuid.UUID
	if err := f.pool.QueryRow(context.Background(), "SELECT id FROM users WHERE username=$1", username).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func createStaff(t *testing.T, f *fixture, username, role string) uuid.UUID {
	id := uuid.New()
	hash, err := auth.HashPassword("fixture-password-123")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(context.Background(), `INSERT INTO users(id,username,password_hash,role_id,status) SELECT $1,$2,$3,id,'ACTIVE' FROM roles WHERE name=$4`, id, username, hash, role)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func createAccount(t *testing.T, f *fixture, userID uuid.UUID, number string, balance string) uuid.UUID {
	id := uuid.New()
	customer := uuid.New()
	_, err := f.pool.Exec(context.Background(), `SELECT 1`)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(context.Background(), "SELECT id FROM customers WHERE user_id=$1", userID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(context.Background(), `INSERT INTO bank_accounts(id,customer_id,account_number,account_type,balance,currency,status) VALUES($1,$2,$3,'SAVINGS',$4,'INR','ACTIVE')`, id, customer, number, balance)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func newFixture(t *testing.T) *fixture {
	pool := testPool(t)
	f := &fixture{pool: pool}
	f.router, _ = newRouter(t, pool)
	prefix := fmt.Sprintf("apitest-%d-", time.Now().UnixNano())
	f.oneName, f.twoName = prefix+"one", prefix+"two"
	f.one = createCustomer(t, f, f.oneName)
	f.two = createCustomer(t, f, f.twoName)
	f.opsName, f.adminName = prefix+"ops", prefix+"admin"
	f.ops = createStaff(t, f, f.opsName, "OPERATIONS")
	f.admin = createStaff(t, f, f.adminName, "ADMIN")
	f.oneAccount = createAccount(t, f, f.one, "A"+uuid.NewString()[:20], "10000.00")
	f.destination = createAccount(t, f, f.one, "A"+uuid.NewString()[:20], "1000.00")
	f.twoAccount = createAccount(t, f, f.two, "A"+uuid.NewString()[:20], "5000.00")
	if _, err := pool.Exec(context.Background(), `INSERT INTO account_assignments(user_id,account_id,assigned_by) VALUES($1,$2,$3)`, f.ops, f.oneAccount, f.admin); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM audit_events WHERE actor_user_id IN (SELECT id FROM users WHERE username LIKE $1)`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM transactions WHERE transfer_id IN (SELECT id FROM fund_transfers WHERE initiated_by IN (SELECT id FROM users WHERE username LIKE $1))`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM ledger_entries WHERE transfer_id IN (SELECT id FROM fund_transfers WHERE initiated_by IN (SELECT id FROM users WHERE username LIKE $1))`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM fund_transfers WHERE initiated_by IN (SELECT id FROM users WHERE username LIKE $1)`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM beneficiaries WHERE customer_id IN (SELECT id FROM customers WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1))`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM account_assignments WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM bank_accounts WHERE id IN ($1,$2,$3)`, f.oneAccount, f.destination, f.twoAccount)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)`, prefix+"%")
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE username LIKE $1`, prefix+"%")
	})
	return f
}

func responseData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	body := jsonBody(t, rec)
	data, _ := body["data"].(map[string]any)
	return data
}

var _ = httpx.Response{}
