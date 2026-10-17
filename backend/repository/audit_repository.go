package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct{ pool *pgxpool.Pool }

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository { return &AuditRepository{pool: pool} }

func (r *AuditRepository) Record(ctx context.Context, actor *uuid.UUID, eventType, action, result, ip string, correlation uuid.UUID, metadata map[string]string) error {
	var encoded []byte
	var err error
	if metadata != nil {
		encoded, err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO audit_events(actor_user_id,event_type,action,result,ip_address,correlation_id,metadata) VALUES($1,$2,$3,$4,NULLIF($5,'')::inet,$6,$7)`, actor, eventType, action, result, ip, correlation, encoded)
	return err
}
