package transfer

import (
	"context"
	"errors"
	"time"

	"bank-app-ioc/backend/config"
	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/validate"
	"bank-app-ioc/backend/repository"

	"github.com/google/uuid"
)

type transferRepository interface {
	Create(context.Context, uuid.UUID, uuid.UUID, repository.TransferInput, string) (repository.TransferRecord, bool, error)
	GetForCustomer(context.Context, uuid.UUID, string) (repository.TransferRecord, error)
	GetForOperations(context.Context, uuid.UUID, string) (repository.TransferRecord, error)
	ListForCustomer(context.Context, uuid.UUID, int, int) ([]repository.TransferRecord, int, error)
}
type Service struct {
	repo          transferRepository
	maxPaise      int64
	limitProvider func(context.Context) (int64, error)
}

func NewService(repo transferRepository, maxPaise int64) *Service {
	return &Service{repo: repo, maxPaise: maxPaise}
}
func NewServiceFromConfig(repo transferRepository, cfg config.Config) *Service {
	return NewService(repo, cfg.MaxTransferPaise)
}

// WithLimitProvider makes the limit come from the database (admin-configurable).
func (s *Service) WithLimitProvider(p func(context.Context) (int64, error)) *Service {
	s.limitProvider = p
	return s
}
func dto(r repository.TransferRecord) Transfer {
	var ben *uuid.UUID
	if r.BeneficiaryID != uuid.Nil {
		b := r.BeneficiaryID
		ben = &b
	}
	return Transfer{ReferenceNumber: r.ReferenceNumber, SourceAccountID: r.SourceAccountID, DestinationAccountID: r.DestinationAccountID,
		BeneficiaryID: ben, Amount: r.Amount, Currency: r.Currency, Purpose: r.Purpose, Status: r.Status,
		DestinationAccountMasked: mask(r.DestinationAccountNumber), ReviewStatus: r.ReviewStatus,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339)}
}
func mask(number string) string {
	if len(number) <= 4 {
		return number
	}
	return "******" + number[len(number)-4:]
}
func (s *Service) Create(ctx context.Context, userID, correlationID uuid.UUID, req CreateRequest, key, ip string) (Transfer, bool, error) {
	v := validate.New()
	v.Check(req.SourceAccountID != uuid.Nil, "sourceAccountId", "is required")
	v.Check(req.DestinationAccountID != uuid.Nil, "destinationAccountId", "is required")
	v.Check(req.SourceAccountID != req.DestinationAccountID, "destinationAccountId", "must differ from sourceAccountId")
	paise, ok := validate.Money(req.Amount)
	v.Check(ok && paise > 0, "amount", "must be positive with at most 2 decimal places")
	v.Check(validate.IdempotencyKey(key), "idempotencyKey", "must be 16-100 alphanumeric, underscore, or hyphen characters")
	if err := v.Err(); err != nil {
		return Transfer{}, false, err
	}
	limit := s.maxPaise
	if s.limitProvider != nil {
		l, err := s.limitProvider(ctx)
		if err != nil {
			return Transfer{}, false, apperr.Internal(err) // fail closed
		}
		limit = l
	}
	if paise > limit {
		return Transfer{}, false, apperr.Unprocessable("TRANSFER_LIMIT_EXCEEDED", "Transfer exceeds the transaction limit.")
	}
	req.Purpose = validate.Text(req.Purpose)
	if len([]rune(req.Purpose)) > 255 {
		return Transfer{}, false, apperr.Validation(map[string]string{"purpose": "must be at most 255 characters"})
	}
	ref, err := NewReferenceNumber()
	if err != nil {
		return Transfer{}, false, apperr.Internal(err)
	}
	r, repeated, err := s.repo.Create(ctx, userID, correlationID, repository.TransferInput{SourceAccountID: req.SourceAccountID, DestinationAccountID: req.DestinationAccountID, BeneficiaryID: req.BeneficiaryID, AmountPaise: paise, Purpose: req.Purpose, IdempotencyKey: key, ReferenceNumber: ref}, ip)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrIdempotencyConflict):
			return Transfer{}, false, apperr.Conflict("IDEMPOTENCY_CONFLICT", "Idempotency key was already used with a different request.")
		case errors.Is(err, repository.ErrNotFound):
			return Transfer{}, false, apperr.NotFound()
		case errors.Is(err, repository.ErrFrozenAccount):
			return Transfer{}, false, apperr.Unprocessable("ACCOUNT_FROZEN", "An account is frozen.")
		case errors.Is(err, repository.ErrInsufficientFunds):
			return Transfer{}, false, apperr.Unprocessable("INSUFFICIENT_FUNDS", "Insufficient funds.")
		case errors.Is(err, repository.ErrBeneficiaryInactive):
			return Transfer{}, false, apperr.Unprocessable("BENEFICIARY_INACTIVE", "Beneficiary is not active.")
		default:
			return Transfer{}, false, err
		}
	}
	return dto(r), repeated, nil
}
func (s *Service) GetCustomer(ctx context.Context, userID uuid.UUID, ref string) (Transfer, error) {
	r, e := s.repo.GetForCustomer(ctx, userID, ref)
	if e != nil {
		return Transfer{}, e
	}
	return dto(r), nil
}
func (s *Service) GetOperations(ctx context.Context, userID uuid.UUID, ref string) (Transfer, error) {
	r, e := s.repo.GetForOperations(ctx, userID, ref)
	if e != nil {
		return Transfer{}, e
	}
	return dto(r), nil
}
func (s *Service) List(ctx context.Context, userID uuid.UUID, page, size int) ([]Transfer, int, error) {
	rows, total, err := s.repo.ListForCustomer(ctx, userID, page, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Transfer, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto(row))
	}
	return out, total, nil
}
