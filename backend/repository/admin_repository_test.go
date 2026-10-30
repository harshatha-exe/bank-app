package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testAdminPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to an isolated migrated test database; see the setup steps in the test report")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL could not be reached: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("TEST_DATABASE_URL could not be reached: %v", err)
	}
	return pool
}

func TestAdminRepositoryListUsers(t *testing.T) {
	pool := testAdminPool(t)
	defer pool.Close()
	ctx := context.Background()
	repo := NewAdminRepository(pool)
	prefix := fmt.Sprintf("admin-list-%d-", time.Now().UnixNano())
	names := []struct {
		suffix, role, status string
	}{
		{"100%", "CUSTOMER", "ACTIVE"},
		{"100_", "OPERATIONS", "DEACTIVATED"},
		{"alpha", "CUSTOMER", "ACTIVE"},
		{"beta", "ADMIN", "ACTIVE"},
	}
	for _, user := range names {
		if err := pool.QueryRow(ctx, `
			INSERT INTO users(username, password_hash, role_id, status)
			SELECT $1, 'test', id, $2 FROM roles WHERE name = $3 RETURNING id`, prefix+user.suffix, user.status, user.role).Scan(new(any)); err != nil {
			t.Fatal(err)
		}
	}

	literal, total, err := repo.ListUsers(ctx, UserFilter{Search: prefix + "100%"}, 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(literal) != 1 || literal[0].Username != prefix+"100%" {
		t.Fatalf("literal percent search = %#v total %d", literal, total)
	}
	literal, total, err = repo.ListUsers(ctx, UserFilter{Search: prefix + "100_"}, 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(literal) != 1 || literal[0].Username != prefix+"100_" {
		t.Fatalf("literal underscore search = %#v total %d", literal, total)
	}

	pageOne, total, err := repo.ListUsers(ctx, UserFilter{Search: prefix}, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	pageTwo, totalTwo, err := repo.ListUsers(ctx, UserFilter{Search: prefix}, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || totalTwo != 4 || len(pageOne) != 2 || len(pageTwo) != 2 {
		t.Fatalf("pagination lengths = %d, %d totals = %d, %d", len(pageOne), len(pageTwo), total, totalTwo)
	}
	if pageOne[0].Username >= pageOne[1].Username || pageTwo[0].Username >= pageTwo[1].Username || pageOne[1].Username >= pageTwo[0].Username {
		t.Fatalf("pages are not in deterministic username order: %#v %#v", pageOne, pageTwo)
	}
	for _, first := range pageOne {
		for _, second := range pageTwo {
			if first.ID == second.ID {
				t.Fatalf("duplicate user across pages: %s", first.ID)
			}
		}
	}

	filtered, total, err := repo.ListUsers(ctx, UserFilter{Search: prefix, Role: "CUSTOMER", Status: "ACTIVE"}, 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(filtered) != 2 {
		t.Fatalf("combined role/status filter = %#v total %d", filtered, total)
	}
	for _, user := range filtered {
		if user.Role != "CUSTOMER" || user.Status != "ACTIVE" {
			t.Fatalf("unexpected combined-filter user = %#v", user)
		}
	}
}
