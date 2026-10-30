package admin

import (
	"context"
	"errors"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/validate"
	"bank-app-ioc/backend/repository"
	"github.com/google/uuid"
	"time"
)

type RequestMeta struct {
	IP            string
	CorrelationID uuid.UUID
}

type AuditPage struct {
	Events      []AuditEvent
	Total, Page int
	Size        int
}

type UserPage struct {
	Users      []UserDTO
	Total, Page int
	Size       int
}

type Service struct {
	users       *repository.AdminRepository
	assignments *repository.AssignmentRepository
	limits      *repository.ConfigRepository
	audits      *repository.AuditQueryRepository
	minPassword int
}

func NewService(users *repository.AdminRepository, assignments *repository.AssignmentRepository,
	limits *repository.ConfigRepository, audits *repository.AuditQueryRepository, minPassword int) *Service {
	return &Service{users: users, assignments: assignments, limits: limits, audits: audits, minPassword: minPassword}
}

// entry builds the audit record skeleton. The repository adds entity type/id and metadata.
func entry(actor uuid.UUID, eventType, action string, m RequestMeta) repository.AuditEntry {
	return repository.AuditEntry{ActorUserID: &actor, EventType: eventType, Action: action,
		Result: "SUCCESS", IP: m.IP, CorrelationID: m.CorrelationID}
}

// mapUserErr turns repository sentinel errors into safe, generic API errors.
func mapUserErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return apperr.NotFound()
	case errors.Is(err, repository.ErrProtectedUser):
		return apperr.Unprocessable("PROTECTED_USER", "Operation not allowed for this user.")
	case errors.Is(err, repository.ErrNoCustomerProfile):
		return apperr.Unprocessable("INVALID_ROLE_CHANGE", "Role change is not allowed for this user.")
	}
	return err // unknown => ErrorHandler turns it into a generic 500
}

func (s *Service) CreateUser(ctx context.Context, actor uuid.UUID, req CreateUserRequest, m RequestMeta) (UserDTO, error) {
	in, err := req.clean(s.minPassword)
	if err != nil {
		return UserDTO{}, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return UserDTO{}, apperr.Internal(err)
	}
	u, err := s.users.CreateUser(ctx, in.Username, hash, in.Role, in.FullName, entry(actor, "USER_CREATED", "CREATE", m))
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return UserDTO{}, apperr.Conflict("USERNAME_TAKEN", "Username is already taken.")
		}
		return UserDTO{}, mapUserErr(err)
	}
	return UserDTO{ID: u.ID, Username: u.Username, Role: u.Role, Status: u.Status}, nil
}

func (s *Service) ListUsers(ctx context.Context, q UserQuery) (UserPage, error) {
	filter, page, size, err := q.clean()
	if err != nil {
		return UserPage{}, err
	}
	rows, total, err := s.users.ListUsers(ctx, filter, page, size)
	if err != nil {
		return UserPage{}, err
	}
	users := make([]UserDTO, 0, len(rows))
	for _, user := range rows {
		users = append(users, UserDTO{ID: user.ID, Username: user.Username, Role: user.Role, Status: user.Status, CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339)})
	}
	return UserPage{Users: users, Total: total, Page: page, Size: size}, nil
}

func (s *Service) SetStatus(ctx context.Context, actor, target uuid.UUID, req StatusRequest, m RequestMeta) (string, error) {
	status := validate.Upper(req.Status)
	if !validate.OneOf(status, "ACTIVE", "DEACTIVATED") {
		return "", apperr.Validation(map[string]string{"status": "Status is invalid."})
	}
	if err := s.users.SetStatus(ctx, target, status, entry(actor, "USER_STATUS_CHANGED", "UPDATE", m)); err != nil {
		return "", mapUserErr(err)
	}
	return status, nil
}

func (s *Service) ChangeRole(ctx context.Context, actor, target uuid.UUID, req RoleRequest, m RequestMeta) (string, error) {
	role := validate.Upper(req.Role)
	if !validate.OneOf(role, "CUSTOMER", "OPERATIONS") { // ADMIN can never be granted
		return "", apperr.Validation(map[string]string{"role": "Role is invalid."})
	}
	if err := s.users.ChangeRole(ctx, target, role, entry(actor, "USER_ROLE_CHANGED", "UPDATE", m)); err != nil {
		return "", mapUserErr(err)
	}
	return role, nil
}

func (s *Service) ResetPassword(ctx context.Context, actor, target uuid.UUID, req ResetPasswordRequest, m RequestMeta) error {
	if !passwordOK(req.NewPassword, s.minPassword) {
		return apperr.Validation(map[string]string{"newPassword": "Password does not meet the policy."})
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return apperr.Internal(err)
	}
	return mapUserErr(s.users.ResetPassword(ctx, target, hash, entry(actor, "USER_PASSWORD_RESET", "UPDATE", m)))
}

func (s *Service) AssignAccount(ctx context.Context, actor uuid.UUID, req AssignRequest, m RequestMeta) (AssignmentDTO, error) {
	ops, acc, err := req.clean()
	if err != nil {
		return AssignmentDTO{}, err
	}
	a, err := s.assignments.Create(ctx, actor, ops, acc, entry(actor, "ACCOUNT_ASSIGNED", "CREATE", m))
	switch {
	case errors.Is(err, repository.ErrAlreadyExists):
		return AssignmentDTO{}, apperr.Conflict("ASSIGNMENT_EXISTS", "Account is already assigned to this user.")
	case errors.Is(err, repository.ErrNotFound):
		return AssignmentDTO{}, apperr.Unprocessable("INVALID_ASSIGNMENT", "Assignment is not valid.")
	case err != nil:
		return AssignmentDTO{}, err
	}
	return AssignmentDTO{ID: a.ID, OperationsUserID: a.OperationsUserID, AccountID: a.AccountID,
		AssignedAt: a.AssignedAt.UTC().Format(time.RFC3339)}, nil
}

func limitDTO(l repository.LimitRecord) LimitDTO {
	return LimitDTO{Currency: l.Currency, MaxTransferAmount: validate.FormatMoney(l.MaxPaise)}
}

func (s *Service) GetLimit(ctx context.Context) (LimitDTO, error) {
	l, err := s.limits.GetLimit(ctx)
	if err != nil {
		return LimitDTO{}, err
	}
	return limitDTO(l), nil
}

func (s *Service) SetLimit(ctx context.Context, actor uuid.UUID, req LimitRequest, m RequestMeta) (LimitDTO, error) {
	paise, err := req.clean()
	if err != nil {
		return LimitDTO{}, err
	}
	l, err := s.limits.SetLimit(ctx, paise, actor, entry(actor, "TRANSACTION_LIMIT_CHANGED", "UPDATE", m))
	if err != nil {
		return LimitDTO{}, err
	}
	return limitDTO(l), nil
}

func (s *Service) SearchAudit(ctx context.Context, q AuditQuery) (AuditPage, error) {
	filter, page, size, err := q.clean()
	if err != nil {
		return AuditPage{}, err
	}
	rows, total, err := s.audits.Search(ctx, filter, page, size)
	if err != nil {
		return AuditPage{}, err
	}
	events := make([]AuditEvent, 0, len(rows))
	for _, r := range rows {
		events = append(events, AuditEvent{
			ID: r.ID, ActorUserID: r.ActorUserID, EventType: r.EventType, EntityType: r.EntityType,
			EntityID: r.EntityID, Action: r.Action, Result: r.Result, IPAddress: r.IPAddress,
			CorrelationID: r.CorrelationID, Metadata: r.Metadata, CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return AuditPage{Events: events, Total: total, Page: page, Size: size}, nil
}
