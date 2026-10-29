package repository

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"postgres error", &pgconn.PgError{Code: "23505"}, true},
		{"wrapped postgres error", fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "23505"}), true},
		{"foreign key violation", &pgconn.PgError{Code: "23503"}, false},
		{"nil", nil, false},
		{"matching text only", fmt.Errorf("duplicate key"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsUniqueViolation(test.err); got != test.want {
				t.Fatalf("IsUniqueViolation(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
