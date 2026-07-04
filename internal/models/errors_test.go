package models

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

func TestWrapDBError_NoRows(t *testing.T) {
	assert.Equal(t, WrapDBError(pgx.ErrNoRows), ErrNoRecord)
}

func TestWrapDBError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	assert.Equal(t, WrapDBError(pgErr), ErrDuplicateSlug)
}

func TestWrapDBError_ForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503"}
	assert.Equal(t, WrapDBError(pgErr), ErrInvalidProject)
}

func TestWrapDBError_OtherError(t *testing.T) {
	original := errors.New("some other db error")
	assert.Equal(t, WrapDBError(original), original)
}

func TestWrapDBError_Nil(t *testing.T) {
	assert.Nil(t, WrapDBError(nil))
}
