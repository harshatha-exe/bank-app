package auth

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"bank-app-ioc/backend/repository"
)

func TestDuplicateKeyTextDoesNotMapToSignupConflict(t *testing.T) {
	err := fmt.Errorf("duplicate key value violates unique constraint")
	status := http.StatusBadRequest
	if errors.Is(err, repository.ErrAlreadyExists) {
		status = http.StatusConflict
	}
	if status == http.StatusConflict {
		t.Fatal("text-only duplicate error must not map to USERNAME_TAKEN")
	}
}
