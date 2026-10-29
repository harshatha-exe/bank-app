package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BeneficiaryRepository struct{ pool *pgxpool.Pool }
type BeneficiaryRecord struct {
	ID, DestinationAccountID                   uuid.UUID
	DestinationAccountNumber, Nickname, Status string
}

func NewBeneficiaryRepository(pool *pgxpool.Pool) *BeneficiaryRepository {
	return &BeneficiaryRepository{pool: pool}
}

func (r *BeneficiaryRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]BeneficiaryRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT b.id,b.destination_account_id,a.account_number,b.nickname,b.status
		FROM beneficiaries b JOIN customers c ON c.id=b.customer_id JOIN bank_accounts a ON a.id=b.destination_account_id
		WHERE c.user_id=$1 ORDER BY b.created_at,b.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []BeneficiaryRecord
	for rows.Next() {
		var item BeneficiaryRecord
		if err := rows.Scan(&item.ID, &item.DestinationAccountID, &item.DestinationAccountNumber, &item.Nickname, &item.Status); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *BeneficiaryRepository) CreateForUser(ctx context.Context, userID, destinationID uuid.UUID, nickname string) (BeneficiaryRecord, error) {
	var item BeneficiaryRecord
	err := r.pool.QueryRow(ctx, `INSERT INTO beneficiaries(customer_id,destination_account_id,nickname,status)
		SELECT c.id,$2,$3,'ACTIVE' FROM customers c JOIN bank_accounts a ON a.id=$2
		WHERE c.user_id=$1 AND a.customer_id <> c.id
		RETURNING id,destination_account_id,nickname,status`, userID, destinationID, nickname).
		Scan(&item.ID, &item.DestinationAccountID, &item.Nickname, &item.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return BeneficiaryRecord{}, ErrNotFound
	}
	if err != nil {
		if pgUniqueViolation(err) {
			return BeneficiaryRecord{}, ErrAlreadyExists
		}
		return BeneficiaryRecord{}, err
	}
	if err := r.pool.QueryRow(ctx, `SELECT account_number FROM bank_accounts WHERE id=$1`, destinationID).Scan(&item.DestinationAccountNumber); err != nil {
		return BeneficiaryRecord{}, err
	}
	return item, nil
}

func (r *BeneficiaryRepository) UpdateStatusForUser(ctx context.Context, userID, beneficiaryID uuid.UUID, status string) (BeneficiaryRecord, error) {
	var item BeneficiaryRecord
	err := r.pool.QueryRow(ctx, `UPDATE beneficiaries AS b SET status = $3,
    updated_at = $4 FROM customers AS c, bank_accounts AS a WHERE b.id = $2
    AND b.customer_id = c.id   AND c.user_id = $1  AND a.id = b.destination_account_id
	RETURNING b.id, b.destination_account_id, a.account_number, b.nickname,
    b.status`, userID, beneficiaryID, status, time.Now().UTC()).
		Scan(&item.ID, &item.DestinationAccountID, &item.DestinationAccountNumber, &item.Nickname, &item.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return BeneficiaryRecord{}, ErrNotFound
	}
	return item, err
}
