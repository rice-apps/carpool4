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

// TestConvertToConnectErrorMapsApplicationVocabulary checks every sentinel the
// application can return reaches clients as the agreed code.
func TestConvertToConnectErrorMapsApplicationVocabulary(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err  error
		want connect.Code
	}{
		{app.ErrUnauthenticated, connect.CodeUnauthenticated},
		{app.ErrInvalidArgument, connect.CodeInvalidArgument},
		{app.ErrPermissionDenied, connect.CodePermissionDenied},
		{app.ErrUserNotFound, connect.CodeNotFound},
		{app.ErrRideNotFound, connect.CodeNotFound},
		{app.ErrPhoneRequired, connect.CodeFailedPrecondition},
		{app.ErrRideCancelled, connect.CodeFailedPrecondition},
		{app.ErrCapacityBelowOccupancy, connect.CodeFailedPrecondition},
		{app.ErrOwnerCannotChangeMembership, connect.CodeFailedPrecondition},
		{app.ErrAlreadyJoined, connect.CodeFailedPrecondition},
		{app.ErrFullCapacity, connect.CodeFailedPrecondition},
		{app.ErrNotJoined, connect.CodeFailedPrecondition},
		{app.ErrNotImplemented, connect.CodeUnimplemented},
		{context.Canceled, connect.CodeCanceled},
		{context.DeadlineExceeded, connect.CodeDeadlineExceeded},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			t.Parallel()
			if got := connect.CodeOf(convertToConnectError(tc.err)); got != tc.want {
				t.Errorf("convertToConnectError(%v) code = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestConvertToConnectErrorPassesNilThrough(t *testing.T) {
	t.Parallel()
	if got := convertToConnectError(nil); got != nil {
		t.Errorf("convertToConnectError(nil) = %v, want nil", got)
	}
}

// TestConvertToConnectErrorKeepsConflictDetailOffTheWire checks that a storage
// conflict is reported as a retryable Aborted without naming the database
// detail that caused it.
func TestConvertToConnectErrorKeepsConflictDetailOffTheWire(t *testing.T) {
	t.Parallel()
	cause := fmt.Errorf("%w: private database detail", app.ErrTransactionConflict)

	got := convertToConnectError(cause)

	if code := connect.CodeOf(got); code != connect.CodeAborted {
		t.Errorf("code = %v, want %v", code, connect.CodeAborted)
	}
	if got.Error() != "aborted: transaction conflict" {
		t.Errorf("wire message = %q, want %q", got.Error(), "aborted: transaction conflict")
	}
	if !errors.Is(got, cause) {
		t.Error("the storage cause is no longer reachable for logs")
	}
}

// TestConvertToConnectErrorPrefersContextErrors checks that a cancelled request
// is reported as cancelled even when the failure it caused looks retryable, so
// clients do not retry work the caller already abandoned.
func TestConvertToConnectErrorPrefersContextErrors(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("%w: %w", context.Canceled, app.ErrTransactionConflict)

	if got := connect.CodeOf(convertToConnectError(err)); got != connect.CodeCanceled {
		t.Errorf("code = %v, want %v", got, connect.CodeCanceled)
	}
}

// TestConvertToConnectErrorReplacesDatabaseConstraintDetail checks a constraint
// violation reaches clients as a generic invalid-argument message with the raw
// database text removed.
func TestConvertToConnectErrorReplacesDatabaseConstraintDetail(t *testing.T) {
	t.Parallel()
	cause := fmt.Errorf(`%w: duplicate key value violates unique constraint "rides_pkey"`, app.ErrInvalidRideWrite)

	err := convertToConnectError(cause)

	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want %v", got, connect.CodeInvalidArgument)
	}
	if !strings.Contains(err.Error(), "ride violates a database constraint") {
		t.Errorf("message = %q, want the substituted constraint message", err)
	}
	if strings.Contains(err.Error(), "rides_pkey") {
		t.Errorf("message = %q, want the raw constraint detail removed", err)
	}
	if !errors.Is(err, app.ErrInvalidRideWrite) {
		t.Error("convertToConnectError dropped the application cause")
	}
}

// TestConvertToConnectErrorSanitizesUnknownCauses checks an unrecognised failure
// becomes a generic internal error whose cause stays available to server logs.
func TestConvertToConnectErrorSanitizesUnknownCauses(t *testing.T) {
	t.Parallel()
	cause := errors.New("database password leaked")

	err := convertToConnectError(cause)

	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("code = %v, want %v", got, connect.CodeInternal)
	}
	if strings.Contains(err.Error(), cause.Error()) {
		t.Errorf("wire message %q leaked the cause", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is no longer reachable for logs")
	}
	if loggedCause(err) != cause {
		t.Errorf("loggedCause() = %v, want %v", loggedCause(err), cause)
	}
}
