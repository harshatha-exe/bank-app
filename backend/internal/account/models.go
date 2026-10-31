package account

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
	Type         string    `json:"type" example:"CREDIT"`
	Amount       string    `json:"amount" example:"3000.00"`
	Currency     string    `json:"currency" example:"INR"`
	Description  string    `json:"description" example:"Salary credit"`
	BalanceAfter string    `json:"balanceAfter" example:"12500.00"`
	CreatedAt    string    `json:"createdAt" example:"2026-10-31T10:15:00Z"`
}
