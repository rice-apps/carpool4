package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rice-apps/carpool4/backend/internal/app"
)

// classifyUserError maps a missing user row to the application sentinel and
// passes other errors through transaction conflict classification.
func classifyUserError(err error) error {
	// Translate a missing SQL row into the application-level user error.
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", app.ErrUserNotFound, err)
	}
	return classifyTransactionError(err)
}

// classifyRideError maps a missing ride row to the application sentinel and
// passes other errors through transaction conflict classification.
func classifyRideError(err error) error {
	// Translate a missing SQL row into the application-level ride error.
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", app.ErrRideNotFound, err)
	}
	return classifyTransactionError(err)
}

// classifyRideWriteErrorForApp maps foreign-key and check failures to invalid
// ride input, leaving the PostgreSQL cause available through errors.Is/As.
func classifyRideWriteErrorForApp(err error) error {
	if err == nil {
		return nil
	}
	// A missing related row or failed ride constraint means invalid input.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514") {
		return fmt.Errorf("%w: %w", app.ErrInvalidRideWrite, err)
	}
	return classifyTransactionError(err)
}

// classifyTransactionError maps SQLSTATE 40001 to a retryable application
// conflict while retaining the original error; all other errors pass through.
func classifyTransactionError(err error) error {
	// Preserve the database cause while giving callers a conflict they can retry.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "40001" {
		return fmt.Errorf("%w: %w", app.ErrTransactionConflict, err)
	}
	return err
}
