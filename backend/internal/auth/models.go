package auth

import "github.com/google/uuid"

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Role         string
	Status       string
}

type SignupRequest struct {
	Username string `json:"username" example:"aarav.demo"`
	Password string `json:"password" example:"ExamplePass123!"`
	FullName string `json:"fullName" example:"Aarav Demo"`
}

type LoginRequest struct {
	Username string `json:"username" example:"aarav.demo"`
	Password string `json:"password" example:"ExamplePass123!"`
}

type PublicUser struct {
	ID       uuid.UUID `json:"id" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	Username string    `json:"username" example:"aarav.demo"`
	Role     string    `json:"role" example:"CUSTOMER"`
}

// swagger documentation only, not used at runtime
type AuthUserResponse struct {
	User PublicUser `json:"user"`
}

// swagger documentation only, not used at runtime
type CurrentUserResponse struct {
	ID       uuid.UUID `json:"id" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	Username string    `json:"username" example:"aarav.demo"`
	Role     string    `json:"role" example:"CUSTOMER"`
}
