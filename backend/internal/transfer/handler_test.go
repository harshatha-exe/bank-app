package transfer

import (
	"fmt"
	"testing"

	"bank-app-ioc/backend/internal/apperr"
)

func TestTransferFailureCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"insufficient funds", apperr.Unprocessable("INSUFFICIENT_FUNDS", "no funds"), "INSUFFICIENT_FUNDS"},
		{"validation", apperr.Validation(map[string]string{"amount": "invalid"}), "VALIDATION_ERROR"},
		{"not found", apperr.NotFound(), "NOT_FOUND"},
		{"unknown app error", apperr.BadRequest(), "INTERNAL_ERROR"},
		{"unknown error", fmt.Errorf("database unavailable"), "INTERNAL_ERROR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := transferFailureCode(test.err); got != test.want {
				t.Fatalf("transferFailureCode() = %q, want %q", got, test.want)
			}
		})
	}
}
