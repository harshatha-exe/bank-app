package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct{ pool *pgxpool.Pool }
type SessionRecord struct {
	ID, UserID              uuid.UUID
	Username, Role          string
	LastActiveAt, ExpiresAt time.Time
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, created, expires time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `INSERT INTO sessions(user_id,session_token_hash,created_at,last_active_at,expires_at) VALUES($1,$2,$3,$3,$4) RETURNING id`, userID, tokenHash, created, expires).Scan(&id)
	return id, err
}

func (r *SessionRepository) Find(ctx context.Context, tokenHash string) (authSession SessionRecord, err error) {
	err = r.pool.QueryRow(ctx, `SELECT s.id,s.user_id,u.username,r.name,s.last_active_at,s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id JOIN roles r ON r.id=u.role_id WHERE s.session_token_hash=$1`, tokenHash).
		Scan(&authSession.ID, &authSession.UserID, &authSession.Username, &authSession.Role, &authSession.LastActiveAt, &authSession.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionRecord{}, ErrNotFound
	}
	return
}

func (r *SessionRepository) Touch(ctx context.Context, tokenHash string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET last_active_at=$1 WHERE session_token_hash=$2`, at, tokenHash)
	return err
}
func (r *SessionRepository) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE session_token_hash=$1`, tokenHash)
	return err
}
