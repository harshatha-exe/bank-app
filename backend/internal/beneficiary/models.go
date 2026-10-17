package beneficiary

import "github.com/google/uuid"

type Beneficiary struct {
	ID                       uuid.UUID `json:"id"`
	DestinationAccountID     uuid.UUID `json:"destinationAccountId,omitempty"`
	DestinationAccountMasked string    `json:"destinationAccountMasked"`
	Nickname                 string    `json:"nickname"`
	Status                   string    `json:"status"`
}

type CreateRequest struct {
	DestinationAccountID uuid.UUID `json:"destinationAccountId"`
	Nickname             string    `json:"nickname"`
}

type StatusRequest struct {
	Status string `json:"status"`
}
