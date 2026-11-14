package app

import (
	"context"
	"fmt"

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
// HIGH-LEVEL of func: 1) Checking current size of ride and making sure owner will not be removed
// 2) Calling database to try and remove user 3) Checking if someone was actually removed
// 4) getting the current state of the ride using getUserViewableRide
func (s *Service) LeaveRide(ctx context.Context, currentUser CurrentUser, rideID uuid.UUID) (Ride, error) {
	// TODO(student): Implement LeaveRide application logic.
	// checking to see if ID is Nil or not
	if rideID == uuid.Nil {
		return Ride{}, fmt.Errorf("%w: ride ID", ErrInvalidArgument)
		// ErrInvalidArgument means is called bc rideID is not a valid ride ID
	}
	// create a transaction:
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()
	// get the rides repository for tx
	ridesRepo := tx.RidesRepo()
	// checking to make sure that currentUser != Owner
	ride, err := getUserViewableRide(ctx, tx, currentUser.ID, rideID)
	if err != nil {
		return Ride{}, err
	}
	if ride.Owner.ID == currentUser.ID {
		return Ride{}, ErrNotImplemented
	}
	// call method on ridesRepo
	numberOfRemovedRider, err := ridesRepo.RemoveRideOccupant(ctx, rideID, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}
	// error handling
	if numberOfRemovedRider == 0 {
		return Ride{}, ErrNotJoined
	}
	// using getUserViewableRide to take in the rideID and remove the rider using removeRider
	updateRide, err := getUserViewableRide(ctx, tx, currentUser.ID, rideID)
	if err != nil {
		return Ride{}, err
	}
	// use tx.commit()
	if err := tx.Commit(); err != nil {
		return Ride{}, err
	}
	return updateRide, nil

}
