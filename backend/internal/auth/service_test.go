package auth

import (
	"context"
	"testing"
	"time"
)

func TestSignupValidationRunsBeforeRepository(t *testing.T) {
	service := NewService(nil, nil, time.Hour, time.Hour, 12)
	tests := []struct {
		name string
		req  SignupRequest
	}{
		{"short username", SignupRequest{Username: "ab", Password: "long enough password", FullName: "User"}},
		{"username with spaces", SignupRequest{Username: "user name", Password: "long enough password", FullName: "User"}},
		{"short password", SignupRequest{Username: "user", Password: "short", FullName: "User"}},
		{"empty full name", SignupRequest{Username: "user", Password: "long enough password", FullName: ""}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Signup(context.Background(), test.req); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
