package repository

import (
	"context"
	"encoding/json"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuditEntry struct {
	ActorUserID   *uuid.UUID
	EventType     string
	EntityType    string
	EntityID      *uuid.UUID
	Action        string
	Result        string
	IP            string
	CorrelationID uuid.UUID
	Metadata      map[string]string
}

// execer is satisfied by BOTH *pgxpool.Pool and pgx.Tx, so insertAudit works
// inside a transaction (what we want) or outside one.
type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func insertAudit(ctx context.Context, q execer, e AuditEntry) error {
	var meta []byte
	if e.Metadata != nil {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		meta = b
	}
	if net.ParseIP(e.IP) == nil { // an odd IP must never fail the whole transaction
		e.IP = ""
	}
	_, err := q.Exec(ctx, `
		INSERT INTO audit_events(actor_user_id, event_type, entity_type, entity_id, action, result, ip_address, correlation_id, metadata)
		VALUES($1, $2, NULLIF($3,''), $4, $5, $6, NULLIF($7,'')::inet, $8, $9)`,
		e.ActorUserID, e.EventType, e.EntityType, e.EntityID, e.Action, e.Result, e.IP, e.CorrelationID, meta)
	return err
}
