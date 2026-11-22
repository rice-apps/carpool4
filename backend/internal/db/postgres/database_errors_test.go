package postgres

import (
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"testing"
)

// TestClassifyErrors checks missing rows and invalid ride writes map to their
// application sentinels while unrelated failures pass through.
func TestClassifyErrors(t *testing.T) {
	if got := translateUserError(sql.ErrNoRows); !errors.Is(got, app.ErrUserNotFound) {
		t.Fatalf("translateUserError(sql.ErrNoRows) = %v, want ErrUserNotFound", got)
	}
	if got := translateRideError(sql.ErrNoRows); !errors.Is(got, app.ErrRideNotFound) {
		t.Fatalf("translateRideError(sql.ErrNoRows) = %v, want ErrRideNotFound", got)
	}
	for _, code := range []string{"23503", "23514"} {
		got := translateRideWriteError(&pgconn.PgError{Code: code})
		if !errors.Is(got, app.ErrInvalidRideWrite) {
			t.Fatalf("translateRideWriteError(%s) = %v, want ErrInvalidRideWrite", code, got)
		}
	}
	other := errors.New("database unavailable")
	if got := translateRideWriteError(other); !errors.Is(got, other) {
		t.Fatalf("translateRideWriteError(other) = %v, want original error", got)
	}
}

// TestTranslateTransactionError checks SQLSTATE 40001 mapping and retention of
// the original cause, with unrelated errors unchanged.
func TestTranslateTransactionError(t *testing.T) {
	cause := &pgconn.PgError{Code: "40001", Message: "could not serialize access"}
	got := translateTransactionError(cause)
	if !errors.Is(got, app.ErrTransactionConflict) {
		t.Fatalf("translateTransactionError(40001) = %v, want ErrTransactionConflict", got)
	}
	if !errors.Is(got, cause) {
		t.Fatalf("translateTransactionError(40001) lost original cause: %v", got)
	}

	other := errors.New("database unavailable")
	if got := translateTransactionError(other); got != other {
		t.Fatalf("translateTransactionError(other) = %v, want original error", got)
	}
}

// TestSpecializedClassifiersPreserveTransactionConflicts checks each area
// classifier preserves the conflict sentinel and PostgreSQL cause.
func TestSpecializedClassifiersPreserveTransactionConflicts(t *testing.T) {
	cause := &pgconn.PgError{Code: "40001", Message: "could not serialize access"}
	for name, classify := range map[string]func(error) error{
		"user":       translateUserError,
		"ride":       translateRideError,
		"ride write": translateRideWriteError,
	} {
		t.Run(name, func(t *testing.T) {
			got := classify(cause)
			if !errors.Is(got, app.ErrTransactionConflict) || !errors.Is(got, cause) {
				t.Fatalf("classifier(40001) = %v, want conflict retaining cause", got)
			}
		})
	}
}
