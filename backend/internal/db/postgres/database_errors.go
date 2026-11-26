package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rice-apps/carpool4/backend/internal/app"
)

// PostgreSQL error codes this package reacts to. See
// https://www.postgresql.org/docs/current/errcodes-appendix.html.
const (
	foreignKeyViolation  = "23503"
	checkViolation       = "23514"
	serializationFailure = "40001"
)

// translateUserError reports a missing profile as app.ErrUserNotFound.
func translateUserError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", app.ErrUserNotFound, err)
	}
	return translateTransactionError(err)
}

// translateRideError reports a missing ride as app.ErrRideNotFound.
func translateRideError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", app.ErrRideNotFound, err)
	}
	return translateTransactionError(err)
}

// translateRideWriteError reports a rejected ride write as invalid input. A
// missing referenced row or a failed table constraint both mean the caller
// supplied something the schema will not accept, which is their problem to fix
// rather than a server fault.
func translateRideWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == foreignKeyViolation || pgErr.Code == checkViolation) {
		return fmt.Errorf("%w: %w", app.ErrInvalidRideWrite, err)
	}
	return translateTransactionError(err)
}

// translateTransactionError reports a lost serializable race as
// app.ErrTransactionConflict, which tells callers to rerun the whole use case
// in a fresh transaction. Every other error passes through unchanged.
//
// All four translators keep the original error wrapped, so the PostgreSQL
// detail stays available to errors.Is and errors.As for server logs even
// though it never reaches a client.
func translateTransactionError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == serializationFailure {
		return fmt.Errorf("%w: %w", app.ErrTransactionConflict, err)
	}
	return err
}
