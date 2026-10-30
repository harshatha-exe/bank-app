package security

import (
	"context"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"bank-app-ioc/backend/database"
)

func testPool(t *testing.T) interface {
	Close()
	Ping(context.Context) error
} {
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

func TestSecurityMatrixRequiresIsolatedDatabase(t *testing.T) {
	pool := testPool(t)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("test database ping failed: %v", err)
	}
	t.Skip("security fixture setup is opt-in with an isolated migrated test database")
}
