package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProtectedUser     = errors.New("protected user")
	ErrNoCustomerProfile = errors.New("no customer profile")
)

type AdminRepository struct{ pool *pgxpool.Pool }

type AdminUser struct {
	ID        uuid.UUID
	Username  string
	Role      string
	Status    string
	CreatedAt time.Time
}

type UserFilter struct {
	Search, Role, Status string
}

func NewAdminRepository(pool *pgxpool.Pool) *AdminRepository { return &AdminRepository{pool: pool} }

func (r *AdminRepository) ListUsers(ctx context.Context, f UserFilter, page, size int) ([]AdminUser, int, error) {
	conds := []string{"TRUE"}
	args := []any{}
	add := func(format string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(format, len(args)))
	}
	if f.Search != "" {
		add("POSITION(LOWER($%d) IN LOWER(username)) > 0", f.Search)
	}
	if f.Role != "" {
		add("r.name = $%d", f.Role)
	}
	if f.Status != "" {
		add("u.status = $%d", f.Status)
	}
	where := strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM users u JOIN roles r ON r.id = u.role_id WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, size, (page-1)*size)
	query := fmt.Sprintf(`
		SELECT u.id, u.username, r.name, u.status, u.created_at
		  FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE %s
		 ORDER BY u.username ASC, u.id ASC
		 LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []AdminUser{}
	for rows.Next() {
		var user AdminUser
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.Status, &user.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, total, rows.Err()
}

// lockUserRole locks the user's row (so two admins can't race) and returns their role.
func lockUserRole(ctx context.Context, tx pgx.Tx, id uuid.UUID) (string, error) {
	var role string
	err := tx.QueryRow(ctx, `
		SELECT r.name FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE u.id = $1 FOR UPDATE OF u`, id).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func (r *AdminRepository) CreateUser(ctx context.Context, username, passwordHash, role, fullName string, e AuditEntry) (AdminUser, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AdminUser{}, err
	}
	defer tx.Rollback(ctx)

	u := AdminUser{Role: role}
	err = tx.QueryRow(ctx, `
		INSERT INTO users(username, password_hash, role_id, status, force_password_change)
		SELECT $1, $2, id, 'ACTIVE', TRUE FROM roles WHERE name = $3
		RETURNING id, username, status`, username, passwordHash, role).
		Scan(&u.ID, &u.Username, &u.Status)
	switch {
	case pgUniqueViolation(err):
		return AdminUser{}, ErrAlreadyExists
	case errors.Is(err, pgx.ErrNoRows):
		return AdminUser{}, ErrNotFound
	case err != nil:
		return AdminUser{}, err
	}

	if role == "CUSTOMER" { // a customer login needs a customer profile too
		number := "CUST-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:16])
		if _, err = tx.Exec(ctx, `INSERT INTO customers(user_id, customer_number, full_name) VALUES($1,$2,$3)`, u.ID, number, fullName); err != nil {
			return AdminUser{}, err
		}
	}

	e.EntityType, e.EntityID = "USER", &u.ID
	e.Metadata = map[string]string{"role": role}
	if err = insertAudit(ctx, tx, e); err != nil {
		return AdminUser{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AdminUser{}, err
	}
	return u, nil
}

func (r *AdminRepository) SetStatus(ctx context.Context, target uuid.UUID, status string, e AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	role, err := lockUserRole(ctx, tx, target)
	if err != nil {
		return err
	}
	if role == "ADMIN" {
		return ErrProtectedUser
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`, target, status); err != nil {
		return err
	}
	if status == "DEACTIVATED" { // kill their sessions NOW
		if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, target); err != nil {
			return err
		}
	}
	e.EntityType, e.EntityID = "USER", &target
	e.Metadata = map[string]string{"status": status}
	if err = insertAudit(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AdminRepository) ChangeRole(ctx context.Context, target uuid.UUID, role string, e AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := lockUserRole(ctx, tx, target)
	if err != nil {
		return err
	}
	if current == "ADMIN" {
		return ErrProtectedUser
	}
	if role == "CUSTOMER" { // can't become a customer without a customer profile
		var hasProfile bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE user_id = $1)`, target).Scan(&hasProfile); err != nil {
			return err
		}
		if !hasProfile {
			return ErrNoCustomerProfile
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET role_id = (SELECT id FROM roles WHERE name = $2), updated_at = NOW() WHERE id = $1`, target, role); err != nil {
		return err
	}
	if role != "OPERATIONS" { // least privilege: no longer ops, no more assigned accounts
		if _, err = tx.Exec(ctx, `DELETE FROM account_assignments WHERE user_id = $1`, target); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, target); err != nil { // privilege change => re-login
		return err
	}
	e.EntityType, e.EntityID = "USER", &target
	e.Metadata = map[string]string{"from": current, "to": role}
	if err = insertAudit(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AdminRepository) ResetPassword(ctx context.Context, target uuid.UUID, passwordHash string, e AuditEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	role, err := lockUserRole(ctx, tx, target)
	if err != nil {
		return err
	}
	if role == "ADMIN" {
		return ErrProtectedUser
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET password_hash = $2, force_password_change = TRUE, updated_at = NOW() WHERE id = $1`, target, passwordHash); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, target); err != nil {
		return err
	}
	e.EntityType, e.EntityID = "USER", &target
	// e.Metadata stays nil on purpose: NEVER put a password (or its hash) in an audit event
	if err = insertAudit(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
