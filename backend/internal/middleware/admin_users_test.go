package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"bank-app-ioc/backend/database"
	"bank-app-ioc/backend/internal/admin"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/repository"

	"github.com/gin-gonic/gin"
)

func TestAdminUserListAuthorizationRealChain(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to an isolated migrated test database; see the setup steps in the test report")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL could not be reached: %v", err)
	}
	defer pool.Close()

	prefix := fmt.Sprintf("admin-auth-%d-", time.Now().UnixNano())
	password := "AdminTestPassword123!"
	users := []struct{ role, username string }{
		{"ADMIN", prefix + "admin"},
		{"CUSTOMER", prefix + "customer"},
		{"OPERATIONS", prefix + "operations"},
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users(username, password_hash, role_id, status)
			SELECT $1, $2, id, 'ACTIVE' FROM roles WHERE name = $3`, user.username, passwordHash, user.role); err != nil {
			t.Fatal(err)
		}
	}

	authService := auth.NewService(repository.NewUserRepository(pool), repository.NewSessionRepository(pool), time.Hour, 24*time.Hour, 12)
	auditService := audit.NewService(repository.NewAuditRepository(pool))
	adminService := admin.NewService(repository.NewAdminRepository(pool), nil, nil, nil, 12)
	handler := admin.NewHandler(adminService)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler())
	protected := router.Group("/api", Authenticate(authService), CSRF("test-secret"))
	protected.Group("/admin", AdminOnly(auditService)).GET("/users", handler.ListUsers)

	request := func(cookie string) int {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "session_id", Value: cookie})
		}
		router.ServeHTTP(recorder, req)
		return recorder.Code
	}

	for _, user := range users {
		_, sessionID, err := authService.Login(ctx, auth.LoginRequest{Username: user.username, Password: password})
		if err != nil {
			t.Fatal(err)
		}
		want := http.StatusForbidden
		if user.role == "ADMIN" {
			want = http.StatusOK
		}
		if got := request(sessionID); got != want {
			t.Errorf("%s session status = %d, want %d", user.role, got, want)
		}
	}
	if got := request(""); got != http.StatusUnauthorized {
		t.Errorf("missing session status = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := request("bogus-session"); got != http.StatusUnauthorized {
		t.Errorf("bogus session status = %d, want %d", got, http.StatusUnauthorized)
	}
}
