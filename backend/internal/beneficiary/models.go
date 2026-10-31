package beneficiary

import "github.com/google/uuid"

type Beneficiary struct {
	ID                       uuid.UUID `json:"id" example:"3c7d9e10-2a4b-4f86-b1d3-6e8a0c925174"`
	DestinationAccountID     uuid.UUID `json:"destinationAccountId,omitempty" example:"5b2d7f41-9c63-4e80-a5b1-2d7c8f904613"`
	DestinationAccountMasked string    `json:"destinationAccountMasked" example:"******4821"`
	Nickname                 string    `json:"nickname" example:"Monthly rent"`
	Status                   string    `json:"status" example:"ACTIVE"`
}

type CreateRequest struct {
	DestinationAccountID uuid.UUID `json:"destinationAccountId" example:"5b2d7f41-9c63-4e80-a5b1-2d7c8f904613"`
	Nickname             string    `json:"nickname" example:"Monthly rent"`
}

type StatusRequest struct {
	Status string `json:"status" example:"DEACTIVATED"`
}
