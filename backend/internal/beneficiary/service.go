package beneficiary

import (
	"context"
	"strings"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/validate"
	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

type Service struct {
	repo *repository.BeneficiaryRepository
}

func NewService(repo *repository.BeneficiaryRepository) *Service { return &Service{repo: repo} }
func mask(number string) string {
	if len(number) <= 4 {
		return number
	}
	return "******" + number[len(number)-4:]
}
func dto(r repository.BeneficiaryRecord) Beneficiary {
	return Beneficiary{ID: r.ID, DestinationAccountID: r.DestinationAccountID, DestinationAccountMasked: mask(r.DestinationAccountNumber), Nickname: r.Nickname, Status: r.Status}
}
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Beneficiary, error) {
	rows, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Beneficiary, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto(r))
	}
	return out, nil
}
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (Beneficiary, error) {
	req.Nickname = validate.Text(req.Nickname)
	v := validate.New()
	v.Check(req.DestinationAccountID != uuid.Nil, "destinationAccountId", "is required")
	v.Check(req.Nickname != "" && len([]rune(req.Nickname)) <= 100, "nickname", "is required and must be at most 100 characters")
	if err := v.Err(); err != nil {
		return Beneficiary{}, err
	}
	r, err := s.repo.CreateForUser(ctx, userID, req.DestinationAccountID, req.Nickname)
	if err != nil {
		return Beneficiary{}, err
	}
	return dto(r), nil
}
func (s *Service) UpdateStatus(ctx context.Context, userID, beneficiaryID uuid.UUID, status string) (Beneficiary, error) {
	status = strings.TrimSpace(status)
	if status != "ACTIVE" && status != "DEACTIVATED" {
		return Beneficiary{}, apperr.Validation(map[string]string{"status": "must be ACTIVE or DEACTIVATED"})
	}
	r, err := s.repo.UpdateStatusForUser(ctx, userID, beneficiaryID, status)
	if err != nil {
		return Beneficiary{}, err
	}
	return dto(r), nil
}
