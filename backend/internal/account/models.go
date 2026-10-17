package account

import "github.com/google/uuid"

type Account struct {
	ID                  uuid.UUID `json:"id"`
	AccountNumberMasked string    `json:"accountNumberMasked"`
	AccountType         string    `json:"accountType"`
	Balance             string    `json:"balance"`
	Currency            string    `json:"currency"`
	Status              string    `json:"status"`
}

type Transaction struct {
	ID           uuid.UUID `json:"id"`
	Type         string    `json:"type"`
	Amount       string    `json:"amount"`
	Currency     string    `json:"currency"`
	Description  string    `json:"description"`
	BalanceAfter string    `json:"balanceAfter"`
	CreatedAt    string    `json:"createdAt"`
}
