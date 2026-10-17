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
	Username string `json:"username"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type PublicUser struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
}
