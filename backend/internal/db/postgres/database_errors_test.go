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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	cause := &pgconn.PgError{Code: "40001", Message: "could not serialize access"}
	for _, tc := range []struct {
		name     string
		classify func(error) error
	}{
		{"translateUserError", translateUserError},
		{"translateRideError", translateRideError},
		{"translateRideWriteError", translateRideWriteError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.classify(cause)

			if !errors.Is(got, app.ErrTransactionConflict) {
				t.Errorf("%s(40001) = %v, want ErrTransactionConflict", tc.name, got)
			}
			if !errors.Is(got, cause) {
				t.Errorf("%s(40001) dropped the PostgreSQL cause: %v", tc.name, got)
			}
		})
	}
}

// TestTranslatorsPassNilThrough checks every translator can be applied to a
// successful call without inventing an error, so callers need no nil guard.
func TestTranslatorsPassNilThrough(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		classify func(error) error
	}{
		{"translateUserError", translateUserError},
		{"translateRideError", translateRideError},
		{"translateRideWriteError", translateRideWriteError},
		{"translateTransactionError", translateTransactionError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.classify(nil); got != nil {
				t.Errorf("%s(nil) = %v, want nil", tc.name, got)
			}
		})
	}
}
