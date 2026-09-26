package app

import (
	"context"

	"github.com/google/uuid"
)

// GetRide returns the ride identified by id for an authenticated or anonymous viewer.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies the viewer, or is empty for a guest.
//   - id (uuid.UUID): identifies the ride to return.
//
// The returned Ride shows owner and rider contacts only when actor may see
// them. Guests see discoverable trips without people or notes. An empty ride
// ID, missing or undiscoverable guest ride, or storage failure returns an error.
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
