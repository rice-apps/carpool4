package rpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/rice-apps/carpool4/backend/internal/app"
)

// TestConvertToConnectErrorMapsTransactionConflictSafely checks the public Aborted code and retained private cause.
func TestConvertToConnectErrorMapsTransactionConflictSafely(t *testing.T) {
	cause := fmt.Errorf("%w: private database detail", app.ErrTransactionConflict)
	got := convertToConnectError(cause)
	if code := connect.CodeOf(got); code != connect.CodeAborted {
		t.Fatalf("convertToConnectError(transaction conflict) code = %v, want %v", code, connect.CodeAborted)
	}
	if got.Error() != "aborted: transaction conflict" {
		t.Fatalf("convertToConnectError(transaction conflict) = %q, want safe public message", got)
	}
	if !errors.Is(got, cause) {
		t.Fatalf("convertToConnectError(transaction conflict) lost private cause: %v", got)
	}
}

// TestConvertToConnectErrorPrioritizesContextErrors checks cancellation wins over a wrapped transaction conflict.
func TestConvertToConnectErrorPrioritizesContextErrors(t *testing.T) {
	err := fmt.Errorf("%w: %w", context.Canceled, app.ErrTransactionConflict)
	if code := connect.CodeOf(convertToConnectError(err)); code != connect.CodeCanceled {
		t.Fatalf("convertToConnectError(canceled transaction) code = %v, want %v", code, connect.CodeCanceled)
	}
}

// TestConnectErrorMapsApplicationVocabulary checks Connect codes for expected application failures.
func TestConnectErrorMapsApplicationVocabulary(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code connect.Code
	}{
		{app.ErrUnauthenticated, connect.CodeUnauthenticated},
		{app.ErrInvalidArgument, connect.CodeInvalidArgument},
		{app.ErrPermissionDenied, connect.CodePermissionDenied},
		{app.ErrRideNotFound, connect.CodeNotFound},
		{app.ErrUserNotFound, connect.CodeNotFound},
		{app.ErrOwnerCannotChangeMembership, connect.CodeFailedPrecondition},
		{app.ErrFullCapacity, connect.CodeFailedPrecondition},
		{app.ErrPhoneRequired, connect.CodeFailedPrecondition},
		{app.ErrRideCancelled, connect.CodeFailedPrecondition},
		{app.ErrCapacityBelowOccupancy, connect.CodeFailedPrecondition},
		{app.ErrAlreadyJoined, connect.CodeFailedPrecondition},
		{app.ErrNotJoined, connect.CodeFailedPrecondition},
		{context.Canceled, connect.CodeCanceled},
		{context.DeadlineExceeded, connect.CodeDeadlineExceeded},
	} {
		if got := connect.CodeOf(convertToConnectError(tc.err)); got != tc.code {
			t.Fatalf("convertToConnectError(%v) code = %v, want %v", tc.err, got, tc.code)
		}
	}
}

// TestConnectErrorSanitizesUnknownCause checks that unknown details stay off the wire but remain unwrapable.
func TestConnectErrorSanitizesUnknownCause(t *testing.T) {
	cause := errors.New("database password leaked")
	err := convertToConnectError(cause)
	if connect.CodeOf(err) != connect.CodeInternal || strings.Contains(err.Error(), "password") || !errors.Is(err, cause) {
		t.Fatalf("convertToConnectError unknown cause = %v", err)
	}
}

// TestConnectErrorReplacesDatabaseConstraintDetail checks a constraint violation reaches
// clients as a generic invalid-argument message with the raw database detail removed.
func TestConnectErrorReplacesDatabaseConstraintDetail(t *testing.T) {
	cause := fmt.Errorf("%w: duplicate key value violates unique constraint \"rides_pkey\"", app.ErrInvalidRideWrite)

	err := convertToConnectError(cause)
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid argument", got)
	}
	if msg := err.Error(); !strings.Contains(msg, "ride violates a database constraint") {
		t.Fatalf("message = %q, want the substituted constraint message", msg)
	}
	if msg := err.Error(); strings.Contains(msg, "rides_pkey") {
		t.Fatalf("message = %q, want the raw constraint detail removed", msg)
	}
	if !errors.Is(err, app.ErrInvalidRideWrite) {
		t.Error("convertToConnectError dropped the application cause")
	}
}
