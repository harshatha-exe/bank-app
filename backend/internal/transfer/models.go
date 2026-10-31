package transfer

import "github.com/google/uuid"

type CreateRequest struct {
	SourceAccountID      uuid.UUID `json:"sourceAccountId" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	DestinationAccountID uuid.UUID `json:"destinationAccountId" example:"5b2d7f41-9c63-4e80-a5b1-2d7c8f904613"`
	BeneficiaryID        uuid.UUID `json:"beneficiaryId" example:"3c7d9e10-2a4b-4f86-b1d3-6e8a0c925174"`
	Amount               string    `json:"amount" example:"3000.00"`
	Purpose              string    `json:"purpose" example:"Monthly rent"`
}

type Transfer struct {
	ReferenceNumber          string     `json:"referenceNumber" example:"TRF-9F3A7C2E51B8D604"`
	SourceAccountID          uuid.UUID  `json:"sourceAccountId" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	DestinationAccountID     uuid.UUID  `json:"destinationAccountId" example:"5b2d7f41-9c63-4e80-a5b1-2d7c8f904613"`
	BeneficiaryID            *uuid.UUID `json:"beneficiaryId,omitempty" example:"3c7d9e10-2a4b-4f86-b1d3-6e8a0c925174"`
	Amount                   string     `json:"amount" example:"3000.00"`
	Currency                 string     `json:"currency" example:"INR"`
	Purpose                  string     `json:"purpose" example:"Monthly rent"`
	Status                   string     `json:"status" example:"COMPLETED"`
	DestinationAccountMasked string     `json:"destinationAccountMasked" example:"******4821"`
	ReviewStatus             string     `json:"reviewStatus" example:"NONE"`
	CreatedAt                string     `json:"createdAt" example:"2026-10-31T10:15:00Z"`
}
