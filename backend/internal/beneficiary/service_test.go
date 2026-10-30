package beneficiary

import (
	"encoding/json"
	"strings"
	"testing"

	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

func TestMaskAndDTO(t *testing.T) {
	if got := mask("1234567890123456"); got != "******3456" {
		t.Fatalf("got %q", got)
	}
	if got := mask("1234"); got != "1234" {
		t.Fatalf("got %q", got)
	}
	record := repository.BeneficiaryRecord{ID: uuid.New(), DestinationAccountNumber: "1234567890123456", Nickname: "Rent", Status: "ACTIVE"}
	data, err := json.Marshal(dto(record))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "1234567890123456") || !strings.Contains(string(data), "******3456") {
		t.Fatalf("unexpected JSON: %s", data)
	}
}

func TestCreateAndUpdateValidation(t *testing.T) {
	service := NewService(nil)
	for _, req := range []CreateRequest{
		{Nickname: "valid"},
		{DestinationAccountID: uuid.New()},
		{DestinationAccountID: uuid.New(), Nickname: strings.Repeat("x", 101)},
	} {
		if _, err := service.Create(nil, uuid.New(), req); err == nil {
			t.Fatal("expected create validation error")
		}
	}
	for _, status := range []string{"", "PENDING", "ARCHIVED"} {
		if _, err := service.UpdateStatus(nil, uuid.New(), uuid.New(), status); err == nil {
			t.Fatalf("expected invalid status %q", status)
		}
	}
}
