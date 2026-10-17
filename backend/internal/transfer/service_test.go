package transfer

import (
	"context"
	"errors"
	"testing"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

type mockRepo struct {
	create func(context.Context, uuid.UUID, uuid.UUID, repository.TransferInput, string) (repository.TransferRecord, bool, error)
}

func (m mockRepo) Create(c context.Context, u, cid uuid.UUID, in repository.TransferInput, ip string) (repository.TransferRecord, bool, error) {
	return m.create(c, u, cid, in, ip)
}
func (m mockRepo) GetForCustomer(context.Context, uuid.UUID, string) (repository.TransferRecord, error) {
	return repository.TransferRecord{}, repository.ErrNotFound
}
func (m mockRepo) GetForOperations(context.Context, uuid.UUID, string) (repository.TransferRecord, error) {
	return repository.TransferRecord{}, repository.ErrNotFound
}
func (m mockRepo) ListForCustomer(context.Context, uuid.UUID, int, int) ([]repository.TransferRecord, int, error) {
	return nil, 0, nil
}

func validRequest() CreateRequest {
	return CreateRequest{SourceAccountID: uuid.New(), DestinationAccountID: uuid.New(), Amount: "10.00", Purpose: "test"}
}
func TestServiceValidationAndIdempotency(t *testing.T) {
	user := uuid.New()
	repeated := repository.TransferRecord{ID: uuid.New(), SourceAccountID: uuid.New(), DestinationAccountID: uuid.New(), ReferenceNumber: "TRF-1", Amount: "10.00", Currency: "INR", Status: "COMPLETED"}
	cases := []struct {
		name string
		req  CreateRequest
	}{
		{"same account", CreateRequest{SourceAccountID: uuid.Nil, DestinationAccountID: uuid.Nil, Amount: "10.00"}},
		{"invalid amount", func() CreateRequest { r := validRequest(); r.Amount = "1.001"; return r }()},
		{"limit exceeded", func() CreateRequest { r := validRequest(); r.Amount = "100.01"; return r }()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewService(mockRepo{}, 10000)
			_, _, err := s.Create(context.Background(), user, uuid.New(), tc.req, "idempotency-key-1234", "127.0.0.1")
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	s := NewService(mockRepo{create: func(context.Context, uuid.UUID, uuid.UUID, repository.TransferInput, string) (repository.TransferRecord, bool, error) {
		return repeated, true, nil
	}}, 10000)
	got, wasRepeated, err := s.Create(context.Background(), user, uuid.New(), validRequest(), "idempotency-key-1234", "127.0.0.1")
	if err != nil || !wasRepeated || got.ReferenceNumber != "TRF-1" {
		t.Fatalf("repeated idempotency result: got=%+v repeated=%v err=%v", got, wasRepeated, err)
	}
}

func TestServiceSuccess(t *testing.T) {
	source, destination := uuid.New(), uuid.New()
	s := NewService(mockRepo{create: func(_ context.Context, _, _ uuid.UUID, in repository.TransferInput, _ string) (repository.TransferRecord, bool, error) {
		if in.SourceAccountID != source || in.DestinationAccountID != destination || in.AmountPaise != 1250 {
			t.Fatalf("unexpected transfer input: %+v", in)
		}
		return repository.TransferRecord{ID: uuid.New(), SourceAccountID: source, DestinationAccountID: destination, ReferenceNumber: "TRF-OK", Amount: "12.50", Currency: "INR", Status: "COMPLETED"}, false, nil
	}}, 100000)
	got, repeated, err := s.Create(context.Background(), uuid.New(), uuid.New(), CreateRequest{SourceAccountID: source, DestinationAccountID: destination, Amount: "12.50"}, "idempotency-key-1234", "127.0.0.1")
	if err != nil || repeated || got.ReferenceNumber != "TRF-OK" {
		t.Fatalf("expected successful transfer, got=%+v repeated=%v err=%v", got, repeated, err)
	}
}

func TestServiceMapsTransferErrors(t *testing.T) {
	for _, want := range []error{repository.ErrInsufficientFunds, repository.ErrFrozenAccount, repository.ErrNotFound} {
		s := NewService(mockRepo{create: func(context.Context, uuid.UUID, uuid.UUID, repository.TransferInput, string) (repository.TransferRecord, bool, error) {
			return repository.TransferRecord{}, false, want
		}}, 100000)
		_, _, err := s.Create(context.Background(), uuid.New(), uuid.New(), validRequest(), "idempotency-key-1234", "127.0.0.1")
		var app *apperr.Error
		if !errors.As(err, &app) {
			t.Fatalf("expected app error for %v", want)
		}
	}
}
