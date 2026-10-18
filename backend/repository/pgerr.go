package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrAlreadyExists = errors.New("already exists")

// pgUniqueViolation reports Postgres error 23505 (unique constraint violated).
func pgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
