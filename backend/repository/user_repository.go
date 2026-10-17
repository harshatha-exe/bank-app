package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct{ pool *pgxpool.Pool }
type UserRecord struct {
	ID                                   uuid.UUID
	Username, PasswordHash, Role, Status string
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository { return &UserRepository{pool: pool} }

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx, `SELECT u.id,u.username,u.password_hash,r.name,u.status FROM users u JOIN roles r ON r.id=u.role_id WHERE u.username=$1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserRecord{}, ErrNotFound
	}
	return u, err
}

func (r *UserRepository) CreateCustomer(ctx context.Context, username, passwordHash, fullName string) (UserRecord, error) {
	var u UserRecord
	customerNumber := "CUST-" + strings.ToUpper(uuid.NewString()[:18])
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return u, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO users(username,password_hash,role_id,status) SELECT $1,$2,id,'ACTIVE' FROM roles WHERE name='CUSTOMER' RETURNING id,username,password_hash,status`, username, passwordHash).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Status)
	if err != nil {
		return u, err
	}
	u.Role = "CUSTOMER"
	_, err = tx.Exec(ctx, `INSERT INTO customers(user_id,customer_number,full_name) VALUES($1,$2,$3)`, u.ID, customerNumber, fullName)
	if err != nil {
		return u, err
	}
	return u, tx.Commit(ctx)
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at=$1,updated_at=$1 WHERE id=$2`, at, id)
	return err
}
