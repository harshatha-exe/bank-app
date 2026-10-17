package account

import (
	"context"
	"strings"
	"time"

	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

type Service struct{ repo *repository.AccountRepository }

func NewService(repo *repository.AccountRepository) *Service { return &Service{repo: repo} }

func mask(number string) string {
	if len(number) <= 4 {
		return number
	}
	return "******" + number[len(number)-4:]
}
func accountDTO(r repository.AccountRecord) Account {
	return Account{ID: r.ID, AccountNumberMasked: mask(r.AccountNumber), AccountType: r.AccountType, Balance: r.Balance, Currency: r.Currency, Status: r.Status}
}
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Account, error) {
	records, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Account, 0, len(records))
	for _, r := range records {
		result = append(result, accountDTO(r))
	}
	return result, nil
}
func (s *Service) Find(ctx context.Context, userID, accountID uuid.UUID) (Account, error) {
	r, err := s.repo.FindForUser(ctx, userID, accountID)
	if err != nil {
		return Account{}, err
	}
	return accountDTO(r), nil
}
func (s *Service) Transactions(ctx context.Context, userID, accountID uuid.UUID, filter repository.TransactionFilter) ([]Transaction, int, error) {
	records, total, err := s.repo.ListTransactions(ctx, userID, accountID, filter)
	if err != nil {
		return nil, 0, err
	}
	result := make([]Transaction, 0, len(records))
	for _, r := range records {
		result = append(result, Transaction{ID: r.ID, Type: r.Type, Amount: r.Amount, Currency: r.Currency, Description: r.Description, BalanceAfter: r.BalanceAfter, CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339)})
	}
	return result, total, nil
}
func parseUUID(value string) (uuid.UUID, error) { return uuid.Parse(strings.TrimSpace(value)) }
