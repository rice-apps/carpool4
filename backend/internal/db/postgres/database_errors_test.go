package postgres

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rice-apps/carpool4/backend/internal/app"
)

// TestTranslatorsMapDatabaseFailuresToSentinels checks that a missing row and a
// rejected ride write reach callers as application errors, and that an
// unrelated failure is left alone.
func TestTranslatorsMapDatabaseFailuresToSentinels(t *testing.T) {
	t.Parallel()
	unrelated := errors.New("database unavailable")

	for _, tc := range []struct {
		name string
		got  error
		want error
	}{
		{name: "missing user row", got: translateUserError(sql.ErrNoRows), want: app.ErrUserNotFound},
		{name: "missing ride row", got: translateRideError(sql.ErrNoRows), want: app.ErrRideNotFound},
		{name: "foreign key violation", got: translateRideWriteError(&pgconn.PgError{Code: foreignKeyViolation}), want: app.ErrInvalidRideWrite},
		{name: "check violation", got: translateRideWriteError(&pgconn.PgError{Code: checkViolation}), want: app.ErrInvalidRideWrite},
		{name: "unrelated ride write failure", got: translateRideWriteError(unrelated), want: unrelated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(tc.got, tc.want) {
				t.Errorf("translated error = %v, want %v", tc.got, tc.want)
			}
		})
	}
}

// TestTranslateTransactionError checks SQLSTATE 40001 mapping and retention of
// the original cause, with unrelated errors unchanged.
func TestTranslateTransactionError(t *testing.T) {
	t.Parallel()
	cause := &pgconn.PgError{Code: serializationFailure, Message: "could not serialize access"}
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
	cause := &pgconn.PgError{Code: serializationFailure, Message: "could not serialize access"}
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
