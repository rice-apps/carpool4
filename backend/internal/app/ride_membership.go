package app

import (
	"context"

	"github.com/google/uuid"
)

// JoinRide adds the caller to a ride.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies the caller joining the ride.
//   - rideID (uuid.UUID): identifies the ride to join.
//
// It returns the updated Ride with contacts actor may see. The ride must be
// active and have room; its owner and current riders cannot join again. A
// missing caller or ride, disallowed membership, or storage failure returns
// an empty Ride and an error.
func (s *Service) JoinRide(ctx context.Context, actor Actor, rideID uuid.UUID) (Ride, error) {
	// TODO(student): Implement JoinRide application logic.
	return Ride{}, ErrNotImplemented
}

// LeaveRide removes the caller from a ride.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies the caller leaving the ride.
//   - rideID (uuid.UUID): identifies the ride to leave.
//
// It returns the updated Ride with contacts actor may still see. A rider may
// leave a cancelled ride; its owner cannot leave. A missing caller or ride,
// absent membership, or storage failure returns an empty Ride and an error.
func (s *Service) LeaveRide(ctx context.Context, actor Actor, rideID uuid.UUID) (Ride, error) {
	// TODO(student): Implement LeaveRide application logic.
	return Ride{}, ErrNotImplemented
}
