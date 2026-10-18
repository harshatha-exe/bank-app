package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConfigRepository struct{ pool *pgxpool.Pool }

type LimitRecord struct {
	Currency string
	MaxPaise int64
}

func NewConfigRepository(pool *pgxpool.Pool) *ConfigRepository { return &ConfigRepository{pool: pool} }

func (r *ConfigRepository) GetLimit(ctx context.Context) (LimitRecord, error) {
	var l LimitRecord
	err := r.pool.QueryRow(ctx,
		`SELECT currency, (max_transfer_amount * 100)::bigint FROM transaction_limits WHERE id = TRUE`).
		Scan(&l.Currency, &l.MaxPaise)
	return l, err
}

// MaxTransferPaise has the exact shape the transfer service's limit provider needs.
func (r *ConfigRepository) MaxTransferPaise(ctx context.Context) (int64, error) {
	l, err := r.GetLimit(ctx)
	return l.MaxPaise, err
}

func paiseText(p int64) string { return fmt.Sprintf("%d.%02d", p/100, p%100) }

func (r *ConfigRepository) SetLimit(ctx context.Context, paise int64, actor uuid.UUID, e AuditEntry) (LimitRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LimitRecord{}, err
	}
	defer tx.Rollback(ctx)

	var old int64
	if err = tx.QueryRow(ctx,
		`SELECT (max_transfer_amount * 100)::bigint FROM transaction_limits WHERE id = TRUE FOR UPDATE`).Scan(&old); err != nil {
		return LimitRecord{}, err
	}
	var l LimitRecord
	if err = tx.QueryRow(ctx, `
		UPDATE transaction_limits
		   SET max_transfer_amount = $1::numeric / 100, updated_by = $2, updated_at = NOW()
		 WHERE id = TRUE
		RETURNING currency, (max_transfer_amount * 100)::bigint`, paise, actor).Scan(&l.Currency, &l.MaxPaise); err != nil {
		return LimitRecord{}, err
	}

	e.EntityType = "TRANSACTION_LIMIT"
	e.Metadata = map[string]string{"from": paiseText(old), "to": paiseText(paise)}
	if err = insertAudit(ctx, tx, e); err != nil {
		return LimitRecord{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return LimitRecord{}, err
	}
	return l, nil
}
