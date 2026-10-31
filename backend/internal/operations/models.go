package operations

import "github.com/google/uuid"

type Account struct {
	ID                  uuid.UUID `json:"id" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	AccountNumberMasked string    `json:"accountNumberMasked" example:"******4821"`
	AccountType         string    `json:"accountType" example:"SAVINGS"`
	Balance             string    `json:"balance" example:"12500.00"`
	Currency            string    `json:"currency" example:"INR"`
	Status              string    `json:"status" example:"ACTIVE"`
}

type Transaction struct {
	ID           uuid.UUID `json:"id" example:"2a8b4c6d-1e3f-4a90-b7c2-5d6e8f901234"`
	Type         string    `json:"type" example:"DEBIT"`
	Amount       string    `json:"amount" example:"3000.00"`
	Currency     string    `json:"currency" example:"INR"`
	Description  string    `json:"description" example:"Monthly rent"`
	BalanceAfter string    `json:"balanceAfter" example:"9500.00"`
	CreatedAt    string    `json:"createdAt" example:"2026-10-31T10:15:00Z"`
}

type Transfer struct {
	ReferenceNumber          string    `json:"referenceNumber" example:"TRF-9F3A7C2E51B8D604"`
	SourceAccountID          uuid.UUID `json:"sourceAccountId" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	DestinationAccountID     uuid.UUID `json:"destinationAccountId" example:"5b2d7f41-9c63-4e80-a5b1-2d7c8f904613"`
	Amount                   string    `json:"amount" example:"3000.00"`
	Currency                 string    `json:"currency" example:"INR"`
	Purpose                  string    `json:"purpose" example:"Monthly rent"`
	Status                   string    `json:"status" example:"COMPLETED"`
	ReviewStatus             string    `json:"reviewStatus" example:"UNDER_REVIEW"`
	DestinationAccountMasked string    `json:"destinationAccountMasked" example:"******4821"`
	CreatedAt                string    `json:"createdAt" example:"2026-10-31T10:15:00Z"`
}

type ReviewRequest struct {
	ReviewStatus string `json:"reviewStatus" example:"UNDER_REVIEW"`
}
