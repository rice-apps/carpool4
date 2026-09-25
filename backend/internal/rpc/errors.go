package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
)

var errInternal = errors.New("internal server error")

// safeRPCError carries a client-safe Connect error and its original cause for logging.
//
// Fields:
//   - public: the error and message exposed to clients.
//   - cause: the underlying failure retained for server diagnostics.
type safeRPCError struct {
	public *connect.Error
	cause  error
}

// Error returns the message safe to send to clients.
func (e *safeRPCError) Error() string { return e.public.Error() }

// Unwrap exposes the public Connect error and original cause to errors.Is and errors.As.
func (e *safeRPCError) Unwrap() []error { return []error{e.public, e.cause} }

// toConnectError turns err into a Connect error, or returns nil for nil input.
// It maps known application and context failures to public codes; unknown causes
// get a generic client message while remaining available to server logs.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}

	code := connect.CodeInternal
	switch {
	case errors.Is(err, context.Canceled):
		code = connect.CodeCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = connect.CodeDeadlineExceeded
	case errors.Is(err, app.ErrNotImplemented):
		code = connect.CodeUnimplemented
	case errors.Is(err, app.ErrTransactionConflict):
		// Tell clients the operation conflicted without exposing database details.
		return &safeRPCError{public: connect.NewError(connect.CodeAborted, app.ErrTransactionConflict), cause: err}
	case errors.Is(err, app.ErrUnauthenticated):
		code = connect.CodeUnauthenticated
	case errors.Is(err, app.ErrUserNotFound):
		return &safeRPCError{public: connect.NewError(connect.CodeNotFound, app.ErrUserNotFound), cause: err}
	case errors.Is(err, app.ErrRideNotFound):
		return &safeRPCError{public: connect.NewError(connect.CodeNotFound, app.ErrRideNotFound), cause: err}
	case errors.Is(err, app.ErrInvalidRideWrite):
		// Replace the database constraint detail with a useful public message.
		public := errors.New("invalid argument: ride violates a database constraint")
		return &safeRPCError{public: connect.NewError(connect.CodeInvalidArgument, public), cause: err}
	case errors.Is(err, app.ErrInvalidArgument):
		code = connect.CodeInvalidArgument
	case errors.Is(err, app.ErrPermissionDenied):
		code = connect.CodePermissionDenied
	case errors.Is(err, app.ErrPhoneRequired),
		errors.Is(err, app.ErrRideCancelled),
		errors.Is(err, app.ErrCapacityBelowOccupancy),
		errors.Is(err, app.ErrOwnerCannotChangeMembership),
		errors.Is(err, app.ErrAlreadyJoined),
		errors.Is(err, app.ErrFullCapacity),
		errors.Is(err, app.ErrNotJoined):
		code = connect.CodeFailedPrecondition
	}
	if code == connect.CodeInternal {
		// Keep the original cause for server logs without sending its details to clients.
		return &safeRPCError{public: connect.NewError(code, errInternal), cause: err}
	}
	return connect.NewError(code, err)
}
