package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditQueryRepository struct{ pool *pgxpool.Pool }

type AuditFilter struct {
	EventType, EntityType, Result string
	ActorUserID, EntityID         *uuid.UUID
	From, To                      *time.Time
}

type AuditRecord struct {
	ID, CorrelationID     uuid.UUID
	ActorUserID, EntityID *uuid.UUID
	EventType, Action     string
	Result                string
	EntityType, IPAddress *string
	Metadata              []byte
	CreatedAt             time.Time
}

func NewAuditQueryRepository(pool *pgxpool.Pool) *AuditQueryRepository {
	return &AuditQueryRepository{pool: pool}
}

func (r *AuditQueryRepository) Search(ctx context.Context, f AuditFilter, page, size int) ([]AuditRecord, int, error) {
	conds := []string{"TRUE"}
	args := []any{}
	// add appends one condition. `format` is always a constant written right here;
	// user values travel ONLY in args ($1, $2, ...), never inside the SQL text.
	add := func(format string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(format, len(args)))
	}
	if f.EventType != "" {
		add("event_type = $%d", f.EventType)
	}
	if f.EntityType != "" {
		add("entity_type = $%d", f.EntityType)
	}
	if f.Result != "" {
		add("result = $%d", f.Result)
	}
	if f.ActorUserID != nil {
		add("actor_user_id = $%d", *f.ActorUserID)
	}
	if f.EntityID != nil {
		add("entity_id = $%d", *f.EntityID)
	}
	if f.From != nil {
		add("created_at >= $%d", *f.From)
	}
	if f.To != nil {
		add("created_at <= $%d", *f.To)
	}
	where := strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, size, (page-1)*size)
	query := fmt.Sprintf(`
		SELECT id, actor_user_id, event_type, entity_type, entity_id, action, result,
		       host(ip_address), correlation_id, metadata, created_at
		  FROM audit_events
		 WHERE %s
		 ORDER BY created_at DESC, id DESC
		 LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []AuditRecord{}
	for rows.Next() {
		var a AuditRecord
		if err := rows.Scan(&a.ID, &a.ActorUserID, &a.EventType, &a.EntityType, &a.EntityID, &a.Action,
			&a.Result, &a.IPAddress, &a.CorrelationID, &a.Metadata, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
