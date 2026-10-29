package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrAlreadyExists = errors.New("already exists")

// IsUniqueViolation reports Postgres error 23505 (unique constraint violated).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func pgUniqueViolation(err error) bool { return IsUniqueViolation(err) }
