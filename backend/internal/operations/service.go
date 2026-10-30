package operations

import (
	"context"
	"errors"
	"time"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/repository"

	"github.com/google/uuid"
)

type accountRepository interface {
	ListForOperations(context.Context, uuid.UUID) ([]repository.AccountRecord, error)
	FindForOperations(context.Context, uuid.UUID, uuid.UUID) (repository.AccountRecord, error)
	SetStatus(context.Context, uuid.UUID, uuid.UUID, string) (repository.AccountRecord, error)
	ListTransactionsForOperations(context.Context, uuid.UUID, uuid.UUID, repository.TransactionFilter) ([]repository.TransactionRecord, int, error)
}
type assignmentRepository interface {
	ListForUser(context.Context, uuid.UUID) ([]uuid.UUID, error)
	HasAccount(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}
type transferRepository interface {
	ListForOperations(context.Context, uuid.UUID, repository.OperationsTransferFilter) ([]repository.TransferRecord, int, error)
	Review(context.Context, uuid.UUID, string, string) (repository.TransferRecord, string, error)
}

type Service struct {
	accounts    accountRepository
	assignments assignmentRepository
	transfers   transferRepository
}

func NewService(accounts accountRepository, assignments assignmentRepository, transfers transferRepository) *Service {
	return &Service{accounts: accounts, assignments: assignments, transfers: transfers}
}

func mask(number string) string {
	if len(number) <= 4 {
		return number
	}
	return "******" + number[len(number)-4:]
}
func accountDTO(r repository.AccountRecord) Account {
	return Account{ID: r.ID, AccountNumberMasked: mask(r.AccountNumber), AccountType: r.AccountType, Balance: r.Balance, Currency: r.Currency, Status: r.Status}
}
func transferDTO(r repository.TransferRecord) Transfer {
	return Transfer{ReferenceNumber: r.ReferenceNumber, SourceAccountID: r.SourceAccountID, DestinationAccountID: r.DestinationAccountID, Amount: r.Amount, Currency: r.Currency, Purpose: r.Purpose, Status: r.Status, ReviewStatus: r.ReviewStatus, DestinationAccountMasked: mask(r.DestinationAccountNumber), CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339)}
}

func (s *Service) ListAccounts(ctx context.Context, userID uuid.UUID) ([]Account, error) {
	rows, err := s.accounts.ListForOperations(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, row := range rows {
		out = append(out, accountDTO(row))
	}
	return out, nil
}

func (s *Service) Transactions(ctx context.Context, userID, accountID uuid.UUID, filter repository.TransactionFilter) ([]Transaction, int, error) {
	rows, total, err := s.accounts.ListTransactionsForOperations(ctx, userID, accountID, filter)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Transaction, 0, len(rows))
	for _, row := range rows {
		out = append(out, Transaction{ID: row.ID, Type: row.Type, Amount: row.Amount, Currency: row.Currency, Description: row.Description, BalanceAfter: row.BalanceAfter, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339)})
	}
	return out, total, nil
}

func (s *Service) Assigned(ctx context.Context, userID, accountID uuid.UUID) (bool, error) {
	return s.assignments.HasAccount(ctx, userID, accountID)
}

func (s *Service) ListTransfers(ctx context.Context, userID uuid.UUID, filter repository.OperationsTransferFilter) ([]Transfer, int, error) {
	rows, total, err := s.transfers.ListForOperations(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Transfer, 0, len(rows))
	for _, row := range rows {
		out = append(out, transferDTO(row))
	}
	return out, total, nil
}

func (s *Service) ReviewTransfer(ctx context.Context, userID uuid.UUID, reference, status string) (Transfer, string, error) {
	if status != "UNDER_REVIEW" && status != "RESOLVED" {
		return Transfer{}, "", apperr.Validation(map[string]string{"reviewStatus": "must be UNDER_REVIEW or RESOLVED"})
	}
	row, old, err := s.transfers.Review(ctx, userID, reference, status)
	if err != nil {
		return Transfer{}, "", err
	}
	return transferDTO(row), old, nil
}

func (s *Service) SetAccountStatus(ctx context.Context, userID, accountID uuid.UUID, status string) (Account, bool, error) {
	assigned, err := s.assignments.HasAccount(ctx, userID, accountID)
	if err != nil {
		return Account{}, false, err
	}
	if !assigned {
		return Account{}, false, repository.ErrNotFound
	}
	current, err := s.accounts.FindForOperations(ctx, userID, accountID)
	if err != nil {
		return Account{}, false, err
	}
	if current.Status == status {
		// ASSUMPTION: repeated freeze/unfreeze requests are idempotent and do not emit a state-change audit event.
		return accountDTO(current), false, nil
	}
	updated, err := s.accounts.SetStatus(ctx, userID, accountID, status)
	if err != nil {
		return Account{}, false, err
	}
	return accountDTO(updated), true, nil
}

func IsNotFound(err error) bool { return errors.Is(err, repository.ErrNotFound) }
