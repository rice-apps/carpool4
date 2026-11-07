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
//   - currentUser (CurrentUser): identifies the caller joining the ride.
//   - rideID (uuid.UUID): identifies the ride to join.
//
// It returns the updated Ride with contacts currentUser may see. The ride must be
// active and have room; its owner and current riders cannot join again. A
// missing ride, disallowed membership, or storage failure returns
// an empty Ride and an error.
func (s *Service) JoinRide(ctx context.Context, currentUser CurrentUser, rideID uuid.UUID) (Ride, error) {
	// TODO(student): Implement JoinRide application logic.
	return Ride{}, ErrNotImplemented
}

// LeaveRide removes the caller from a ride.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies the caller leaving the ride.
//   - rideID (uuid.UUID): identifies the ride to leave.
//
// It returns the updated Ride with contacts currentUser may still see. A rider may
// leave a cancelled ride; its owner cannot leave. A missing ride,
// absent membership, or storage failure returns an empty Ride and an error.
func (s *Service) LeaveRide(ctx context.Context, currentUser CurrentUser, rideID uuid.UUID) (Ride, error) {
	// TODO(student): Implement LeaveRide application logic.
	// checking to see if ID is Nil or not
	if rideID == uuid.Nil {
		return Ride{}, ErrNotImplemented
	}
	// create a transaction:
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()
	// get the rides repository for tx
	ridesRepo := tx.RidesRepo()
	// call method on ridesRepo
	numberOfRemovedRider, err := ridesRepo.RemoveRideOccupant(ctx, rideID, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}
	// error handling
	if numberOfRemovedRider == 0 {
		return Ride{}, ErrNotImplemented
	}
	// using getUserViewableRide to take in the rideID and remove the rider using removeRider
	updateRide, err := getUserViewableRide(ctx, tx, rideID, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}
	// use tx.commit()
	if err := tx.Commit(); err != nil {
		return Ride{}, err
	}
	return updateRide, nil

}
