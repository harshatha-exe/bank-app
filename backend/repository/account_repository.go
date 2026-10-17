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

type AccountRepository struct{ pool *pgxpool.Pool }

type AccountRecord struct {
	ID                                                    uuid.UUID
	AccountNumber, AccountType, Balance, Currency, Status string
}

type TransactionRecord struct {
	ID                                                uuid.UUID
	Type, Amount, Currency, Description, BalanceAfter string
	CreatedAt                                         time.Time
}

type TransactionFilter struct {
	Page, PageSize        int
	From, To              *time.Time
	Type, Sort, Direction string
	MinAmount, MaxAmount  string
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]AccountRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT a.id,a.account_number,a.account_type,a.balance::text,a.currency,a.status
		FROM bank_accounts a JOIN customers c ON c.id=a.customer_id WHERE c.user_id=$1 ORDER BY a.created_at,a.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []AccountRecord
	for rows.Next() {
		var item AccountRecord
		if err := rows.Scan(&item.ID, &item.AccountNumber, &item.AccountType, &item.Balance, &item.Currency, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *AccountRepository) FindForUser(ctx context.Context, userID, accountID uuid.UUID) (AccountRecord, error) {
	var item AccountRecord
	err := r.pool.QueryRow(ctx, `SELECT a.id,a.account_number,a.account_type,a.balance::text,a.currency,a.status
		FROM bank_accounts a JOIN customers c ON c.id=a.customer_id
		WHERE c.user_id=$1 AND a.id=$2`, userID, accountID).
		Scan(&item.ID, &item.AccountNumber, &item.AccountType, &item.Balance, &item.Currency, &item.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountRecord{}, ErrNotFound
	}
	return item, err
}

func (r *AccountRepository) ListTransactions(ctx context.Context, userID, accountID uuid.UUID, f TransactionFilter) ([]TransactionRecord, int, error) {
	args := []any{userID, accountID}
	conditions := []string{"c.user_id=$1", "t.account_id=$2"}
	add := func(sql string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sql, len(args)))
	}
	if f.From != nil {
		add("t.created_at >= $%d", *f.From)
	}
	if f.To != nil {
		add("t.created_at <= $%d", *f.To)
	}
	if f.Type != "" {
		add("t.transaction_type = $%d", f.Type)
	}
	if f.MinAmount != "" {
		add("t.amount >= $%d::numeric", f.MinAmount)
	}
	if f.MaxAmount != "" {
		add("t.amount <= $%d::numeric", f.MaxAmount)
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM transactions t JOIN bank_accounts a ON a.id=t.account_id
		JOIN customers c ON c.id=a.customer_id WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sortColumn := map[string]string{"createdAt": "t.created_at", "amount": "t.amount", "type": "t.transaction_type"}[f.Sort]
	if sortColumn == "" {
		sortColumn = "t.created_at"
	}
	direction := map[string]string{"ASC": "ASC", "DESC": "DESC"}[f.Direction]
	if direction == "" {
		direction = "DESC"
	}
	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	query := `SELECT t.id,t.transaction_type,t.amount::text,t.currency,COALESCE(t.description,''),t.balance_after::text,t.created_at
		FROM transactions t JOIN bank_accounts a ON a.id=t.account_id JOIN customers c ON c.id=a.customer_id
		WHERE ` + where + ` ORDER BY ` + sortColumn + ` ` + direction + `, t.id LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var result []TransactionRecord
	for rows.Next() {
		var item TransactionRecord
		if err := rows.Scan(&item.ID, &item.Type, &item.Amount, &item.Currency, &item.Description, &item.BalanceAfter, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, item)
	}
	return result, total, rows.Err()
}
