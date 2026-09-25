package app

import (
	"context"

	"github.com/google/uuid"
)

// GetRide returns the ride identified by id for actor to view.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies who is viewing the ride.
//   - id (uuid.UUID): identifies the ride to return.
//
// The returned Ride shows owner and rider contacts only when actor may see
// them. A missing caller, empty ride ID, missing ride, or storage failure
// returns an empty Ride and an error.
func (s *Service) GetRide(ctx context.Context, actor Actor, id uuid.UUID) (Ride, error) {
	// TODO(student): Implement GetRide application logic.
	return Ride{}, ErrNotImplemented
}

// getRideInTx returns rideID from tx with contacts visible to viewerID. A
// missing ride or failed lookup or privacy check returns an error.
func getRideInTx(ctx context.Context, tx Transaction, viewerID, rideID uuid.UUID) (Ride, error) {
	// TODO(student): Implement getRideInTx application logic.
	return Ride{}, ErrNotImplemented
}
