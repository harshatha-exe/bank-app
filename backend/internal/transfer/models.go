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
	ReferenceNumber      string     `json:"referenceNumber"`
	SourceAccountID      uuid.UUID  `json:"sourceAccountId"`
	DestinationAccountID uuid.UUID  `json:"destinationAccountId"`
	BeneficiaryID        *uuid.UUID `json:"beneficiaryId,omitempty"`
	Amount               string     `json:"amount"`
	Currency             string     `json:"currency"`
	Purpose              string     `json:"purpose"`
	Status               string     `json:"status"`
	CreatedAt            string     `json:"createdAt"`
}
