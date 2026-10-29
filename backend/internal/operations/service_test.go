package operations

import (
	"context"
	"testing"
	"time"

	"bank-app-ioc/backend/repository"

	"github.com/google/uuid"
)

type accountRepoStub struct {
	account  repository.AccountRecord
	setCalls int
}

func (s *accountRepoStub) ListForOperations(context.Context, uuid.UUID) ([]repository.AccountRecord, error) {
	return []repository.AccountRecord{s.account}, nil
}
func (s *accountRepoStub) FindForOperations(context.Context, uuid.UUID, uuid.UUID) (repository.AccountRecord, error) {
	return s.account, nil
}
func (s *accountRepoStub) SetStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID, status string) (repository.AccountRecord, error) {
	s.setCalls++
	s.account.Status = status
	return s.account, nil
}
func (s *accountRepoStub) ListTransactionsForOperations(context.Context, uuid.UUID, uuid.UUID, repository.TransactionFilter) ([]repository.TransactionRecord, int, error) {
	return nil, 0, nil
}

type assignmentRepoStub struct{ assigned bool }

func (s assignmentRepoStub) ListForUser(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (s assignmentRepoStub) HasAccount(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return s.assigned, nil
}

type transferRepoStub struct{ called bool }

func (s *transferRepoStub) ListForOperations(context.Context, uuid.UUID, repository.OperationsTransferFilter) ([]repository.TransferRecord, int, error) {
	return nil, 0, nil
}
func (s *transferRepoStub) Review(context.Context, uuid.UUID, string, string) (repository.TransferRecord, string, error) {
	s.called = true
	return repository.TransferRecord{ReferenceNumber: "REF-1", ReviewStatus: "RESOLVED", CreatedAt: time.Unix(0, 0)}, "NONE", nil
}

func TestAssignedAccountScoping(t *testing.T) {
	accountID, userID := uuid.New(), uuid.New()
	service := NewService(&accountRepoStub{}, assignmentRepoStub{assigned: false}, &transferRepoStub{})
	if assigned, err := service.Assigned(context.Background(), userID, accountID); err != nil || assigned {
		t.Fatalf("expected out-of-scope account, assigned=%v err=%v", assigned, err)
	}
	service = NewService(&accountRepoStub{}, assignmentRepoStub{assigned: true}, &transferRepoStub{})
	if assigned, err := service.Assigned(context.Background(), userID, accountID); err != nil || !assigned {
		t.Fatalf("expected assigned account, assigned=%v err=%v", assigned, err)
	}
}

func TestReviewStatusValidation(t *testing.T) {
	transfers := &transferRepoStub{}
	service := NewService(&accountRepoStub{}, assignmentRepoStub{assigned: true}, transfers)
	if _, _, err := service.ReviewTransfer(context.Background(), uuid.New(), "REF-1", "NONE"); err == nil {
		t.Fatal("expected invalid review status")
	}
	if transfers.called {
		t.Fatal("repository should not be called for invalid status")
	}
}

func TestFreezeUnfreezeIdempotency(t *testing.T) {
	accountID, userID := uuid.New(), uuid.New()
	accounts := &accountRepoStub{account: repository.AccountRecord{ID: accountID, AccountNumber: "1234567890", Status: "FROZEN"}}
	service := NewService(accounts, assignmentRepoStub{assigned: true}, &transferRepoStub{})
	if _, changed, err := service.SetAccountStatus(context.Background(), userID, accountID, "FROZEN"); err != nil || changed {
		t.Fatalf("freeze should be idempotent, changed=%v err=%v", changed, err)
	}
	if accounts.setCalls != 0 {
		t.Fatal("idempotent freeze should not update")
	}
	if _, changed, err := service.SetAccountStatus(context.Background(), userID, accountID, "ACTIVE"); err != nil || !changed {
		t.Fatalf("unfreeze should update, changed=%v err=%v", changed, err)
	}
	if accounts.setCalls != 1 {
		t.Fatal("expected one status update")
	}
}
