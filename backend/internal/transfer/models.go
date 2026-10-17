package transfer

import "github.com/google/uuid"

type CreateRequest struct {
	SourceAccountID      uuid.UUID `json:"sourceAccountId"`
	DestinationAccountID uuid.UUID `json:"destinationAccountId"`
	BeneficiaryID        uuid.UUID `json:"beneficiaryId"`
	Amount               string    `json:"amount"`
	Purpose              string    `json:"purpose"`
}

type Transfer struct {
	ID, SourceAccountID, DestinationAccountID, BeneficiaryID uuid.UUID
	ReferenceNumber, Amount, Currency, Purpose, Status       string
	CreatedAt                                                string
}
