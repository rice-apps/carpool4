package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GetRide returns the ride identified by id for an authenticated or anonymous viewer.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies the viewer, or is empty for a guest.
//   - id (uuid.UUID): identifies the ride to return.
//
// The returned Ride shows owner and rider contacts only when currentUser may see
// them. Guests see discoverable trips without people or notes. An empty ride
// ID, missing or undiscoverable guest ride, or storage failure returns an error.
func (s *Service) GetRide(ctx context.Context, currentUser CurrentUser, id uuid.UUID) (Ride, error) {
	if id == uuid.Nil {
		return Ride{}, fmt.Errorf("%w: ride ID", ErrInvalidArgument)
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()
	return getUserViewableRide(ctx, tx, currentUser.ID, id)

}

// getUserViewableRide returns rideID from tx with contacts visible to viewerID. A
// missing ride or failed lookup or privacy check returns an error.
func getUserViewableRide(ctx context.Context, tx Transaction, viewerID, rideID uuid.UUID) (Ride, error) {
	loadedRide, err := tx.RidesRepo().GetRide(ctx, rideID)
	if err != nil {
		return Ride{}, err
	}
	if viewerID == uuid.Nil && (loadedRide.DepartureTime.Before(time.Now().Add(-rideDiscoveryLookback)) || loadedRide.Status == RideStatusCancelled) {
		return Ride{}, ErrRideNotFound
	}

	rides := []LoadedRide{loadedRide}
	results, err := buildRidesForViewer(ctx, tx, viewerID, rides)
	if err != nil {
		return Ride{}, err
	}
	return results[0], err
}
