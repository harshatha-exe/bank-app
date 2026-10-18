package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssignmentRepository struct{ pool *pgxpool.Pool }

type AssignmentRecord struct {
	ID, OperationsUserID, AccountID uuid.UUID
	AssignedAt                      time.Time
}

func NewAssignmentRepository(pool *pgxpool.Pool) *AssignmentRepository {
	return &AssignmentRepository{pool: pool}
}

// Create inserts only if the target is an ACTIVE OPERATIONS user AND the account exists.
// One statement does the checks and the insert, so nothing can change in between.
func (r *AssignmentRepository) Create(ctx context.Context, adminID, opsUserID, accountID uuid.UUID, e AuditEntry) (AssignmentRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssignmentRecord{}, err
	}
	defer tx.Rollback(ctx)

	var a AssignmentRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO account_assignments(user_id, account_id, assigned_by)
		SELECT u.id, acc.id, $3
		  FROM users u
		  JOIN roles r ON r.id = u.role_id
		 CROSS JOIN bank_accounts acc
		 WHERE u.id = $1 AND r.name = 'OPERATIONS' AND u.status = 'ACTIVE' AND acc.id = $2
		RETURNING id, user_id, account_id, assigned_at`, opsUserID, accountID, adminID).
		Scan(&a.ID, &a.OperationsUserID, &a.AccountID, &a.AssignedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return AssignmentRecord{}, ErrNotFound
	case pgUniqueViolation(err):
		return AssignmentRecord{}, ErrAlreadyExists
	case err != nil:
		return AssignmentRecord{}, err
	}

	e.EntityType, e.EntityID = "ACCOUNT_ASSIGNMENT", &a.ID
	e.Metadata = map[string]string{"operationsUserId": opsUserID.String(), "accountId": accountID.String()}
	if err = insertAudit(ctx, tx, e); err != nil {
		return AssignmentRecord{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AssignmentRecord{}, err
	}
	return a, nil
}
