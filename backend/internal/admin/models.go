package admin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bank-app-ioc/backend/internal/validate"
	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
)

// ---------- requests: raw and untrusted ----------

type CreateUserRequest struct {
	Username          string `json:"username" example:"nisha.ops"`
	Role              string `json:"role" example:"OPERATIONS"`
	TemporaryPassword string `json:"temporaryPassword" example:"ExamplePass123!"`
	FullName          string `json:"fullName" example:"Nisha Demo"` // required only when role is CUSTOMER
}
type StatusRequest struct {
	Status string `json:"status" example:"DEACTIVATED"`
}
type RoleRequest struct {
	Role string `json:"role" example:"OPERATIONS"`
}
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" example:"ExamplePass123!"`
}
type AssignRequest struct {
	OperationsUserID string `json:"operationsUserId" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	AccountID        string `json:"accountId" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
}
type LimitRequest struct {
	MaxTransferAmount string `json:"maxTransferAmount" example:"100000.00"`
}

// AuditQuery holds the raw query-string values.
type AuditQuery struct {
	EventType, ActorUserID, EntityType, EntityID, Result, From, To, Page, PageSize string
}

type UserQuery struct {
	Search, Role, Status, Page, PageSize string
}

// ---------- responses ----------

type UserDTO struct {
	ID        uuid.UUID `json:"id" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	Username  string    `json:"username" example:"nisha.ops"`
	Role      string    `json:"role" example:"OPERATIONS"`
	Status    string    `json:"status" example:"ACTIVE"`
	CreatedAt string    `json:"createdAt" example:"2026-10-31T10:15:00Z"`
}
type AssignmentDTO struct {
	ID               uuid.UUID `json:"id" example:"6d9f2b41-8c73-4e05-a1b6-7d2f904c5381"`
	OperationsUserID uuid.UUID `json:"operationsUserId" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	AccountID        uuid.UUID `json:"accountId" example:"7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"`
	AssignedAt       string    `json:"assignedAt" example:"2026-10-31T10:15:00Z"`
}
type LimitDTO struct {
	Currency          string `json:"currency" example:"INR"`
	MaxTransferAmount string `json:"maxTransferAmount" example:"100000.00"`
}
type AuditEvent struct {
	ID            uuid.UUID       `json:"id" example:"6d9f2b41-8c73-4e05-a1b6-7d2f904c5381"`
	ActorUserID   *uuid.UUID      `json:"actorUserId" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	EventType     string          `json:"eventType" example:"USER_CREATED"`
	EntityType    *string         `json:"entityType" example:"USER"`
	EntityID      *uuid.UUID      `json:"entityId" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	Action        string          `json:"action" example:"CREATE"`
	Result        string          `json:"result" example:"SUCCESS"`
	IPAddress     *string         `json:"ipAddress" example:"192.0.2.10"`
	CorrelationID uuid.UUID       `json:"correlationId" example:"8a1d3f72-5c94-4e06-b7a2-9f1c63840527"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     string          `json:"createdAt" example:"2026-10-31T10:15:00Z"`
}

// swagger documentation only, not used at runtime
type UserStatusResponse struct {
	ID     uuid.UUID `json:"id" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	Status string    `json:"status" example:"DEACTIVATED"`
}

// swagger documentation only, not used at runtime
type UserRoleResponse struct {
	ID   uuid.UUID `json:"id" example:"4e8a1c53-7b20-4d96-a5f1-9c2e6b804317"`
	Role string    `json:"role" example:"OPERATIONS"`
}

// ---------- clean(): sanitize + validate + convert to typed values ----------

func parseID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	return id, err == nil && id != uuid.Nil
}

// Passwords are NEVER passed through validate.Text: changing a password silently locks people out.
func passwordOK(pw string, minLen int) bool { return validate.RuneLen(pw, minLen, 128) }

type newUser struct{ Username, Role, Password, FullName string }

func (r CreateUserRequest) clean(minPassword int) (newUser, error) {
	v := validate.New()
	username := strings.TrimSpace(r.Username)
	role := validate.Upper(r.Role)
	fullName := validate.Text(r.FullName)

	v.Check(validate.Username(username), "username", "Username is invalid.")
	v.Check(validate.OneOf(role, "CUSTOMER", "OPERATIONS"), "role", "Role is invalid.") // ADMIN is impossible by design
	v.Check(passwordOK(r.TemporaryPassword, minPassword), "temporaryPassword",
		fmt.Sprintf("Password must be %d-128 characters.", minPassword))
	if role == "CUSTOMER" {
		v.Check(validate.RuneLen(fullName, 1, 150) && validate.Name(fullName), "fullName", "Full name is invalid.")
	}
	if err := v.Err(); err != nil {
		return newUser{}, err
	}
	return newUser{Username: username, Role: role, Password: r.TemporaryPassword, FullName: fullName}, nil
}

func (r AssignRequest) clean() (uuid.UUID, uuid.UUID, error) {
	v := validate.New()
	ops, okOps := parseID(r.OperationsUserID)
	acc, okAcc := parseID(r.AccountID)
	v.Check(okOps, "operationsUserId", "Operations user is invalid.")
	v.Check(okAcc, "accountId", "Account is invalid.")
	if err := v.Err(); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return ops, acc, nil
}

const maxConfigurablePaise int64 = 10_000_000_000 // sanity ceiling (Rs 10 crore): guards against typos

func (r LimitRequest) clean() (int64, error) {
	paise, ok := validate.Money(strings.TrimSpace(r.MaxTransferAmount))
	v := validate.New()
	v.Check(ok && paise > 0 && paise <= maxConfigurablePaise, "maxTransferAmount",
		"Amount must be positive, have at most 2 decimals, and be within the allowed range.")
	if err := v.Err(); err != nil {
		return 0, err
	}
	return paise, nil
}

func intParam(raw string, def, lo, hi int) (int, bool) {
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return 0, false
	}
	return n, true
}

func (q AuditQuery) clean() (repository.AuditFilter, int, int, error) {
	v := validate.New()
	var f repository.AuditFilter

	code := func(raw, field string) string {
		s := validate.Upper(raw)
		if s != "" {
			v.Check(validate.Code(s), field, "Value is invalid.")
		}
		return s
	}
	f.EventType = code(q.EventType, "eventType")
	f.EntityType = code(q.EntityType, "entityType")
	f.Result = code(q.Result, "result")

	optID := func(raw, field string) *uuid.UUID {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		parsed, ok := parseID(raw)
		v.Check(ok, field, "Value is invalid.")
		if !ok {
			return nil
		}
		return &parsed
	}
	f.ActorUserID = optID(q.ActorUserID, "actorUserId")
	f.EntityID = optID(q.EntityID, "entityId")

	when := func(raw, field string) *time.Time {
		if raw == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, raw)
		v.Check(err == nil, field, "Use RFC 3339, e.g. 2026-10-04T10:30:00Z.")
		if err != nil {
			return nil
		}
		return &t
	}
	f.From = when(q.From, "from")
	f.To = when(q.To, "to")
	if f.From != nil && f.To != nil {
		v.Check(!f.From.After(*f.To), "from", "Must not be after 'to'.")
	}

	page, okPage := intParam(q.Page, 1, 1, 100000)
	size, okSize := intParam(q.PageSize, 25, 1, 100)
	v.Check(okPage, "page", "Must be between 1 and 100000.")
	v.Check(okSize, "pageSize", "Must be between 1 and 100.")

	if err := v.Err(); err != nil {
		return repository.AuditFilter{}, 0, 0, err
	}
	return f, page, size, nil
}

func (q UserQuery) clean() (repository.UserFilter, int, int, error) {
	v := validate.New()
	search := strings.TrimSpace(q.Search)
	v.Check(validate.RuneLen(search, 0, 100), "search", "Search must be at most 100 characters.")
	role := validate.Upper(q.Role)
	if role != "" {
		v.Check(validate.OneOf(role, "CUSTOMER", "OPERATIONS", "ADMIN"), "role", "Role is invalid.")
	}
	status := validate.Upper(q.Status)
	if status != "" {
		v.Check(validate.OneOf(status, "ACTIVE", "DEACTIVATED"), "status", "Status is invalid.")
	}
	page, okPage := intParam(q.Page, 1, 1, 100000)
	size, okSize := intParam(q.PageSize, 25, 1, 100)
	v.Check(okPage, "page", "Must be between 1 and 100000.")
	v.Check(okSize, "pageSize", "Must be between 1 and 100.")
	if err := v.Err(); err != nil {
		return repository.UserFilter{}, 0, 0, err
	}
	return repository.UserFilter{Search: search, Role: role, Status: status}, page, size, nil
}
