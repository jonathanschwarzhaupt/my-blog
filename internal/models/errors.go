package models

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNoRecord       = errors.New("models: no matching record found")
	ErrEditConflict   = errors.New("models: edit conflict")
	ErrDuplicateSlug  = errors.New("models: a post with this slug already exists")
	ErrInvalidProject = errors.New("models: a referenced project no longer exists")
)

const (
	pgUniqueViolationCode     = "23505"
	pgForeignKeyViolationCode = "23503"
)

// WrapDBError translates Postgres/pgx-specific errors into models sentinel
// errors, so callers check against these instead of driver-specific types.
func WrapDBError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoRecord
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolationCode:
			return ErrDuplicateSlug
		case pgForeignKeyViolationCode:
			return ErrInvalidProject
		}
	}

	return err
}
