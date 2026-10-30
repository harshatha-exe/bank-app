package account

import (
	"encoding/json"
	"strings"
	"testing"

	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

func TestMask(t *testing.T) {
	if got := mask("1234567890123456"); got != "******3456" {
		t.Fatalf("got %q", got)
	}
	for _, value := range []string{"", "1", "1234"} {
		if got := mask(value); got != value {
			t.Fatalf("mask(%q) = %q", value, got)
		}
	}
}

func TestAccountDTOMasksNumber(t *testing.T) {
	account := accountDTO(repository.AccountRecord{ID: uuid.New(), AccountNumber: "1234567890123456"})
	if account.AccountNumberMasked != "******3456" {
		t.Fatalf("unexpected account: %+v", account)
	}
	data, err := json.Marshal(account)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "1234567890123456") {
		t.Fatalf("full account number leaked: %s", data)
	}
}
