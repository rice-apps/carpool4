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
	// gets the current ride details
	ride, err := s.GetRide(ctx, currentUser, rideID)
	// if there is an error with currentUser then return nil and the error
	if err != nil {
		return Ride{}, ErrNotImplemented
	}
	//check if the ride status is active and if not, return empty ride and error
	if ride.Status != RideStatusActive {
		return Ride{}, ErrNotImplemented
	}
	//check if number of riders is greater than or equal to ride capacity and if so, return empty ride and error
	if len(ride.Riders) >= int(ride.Capacity) {
		return Ride{}, ErrNotImplemented
	}
	//check if the original owner is the current user and if so, return empty ride and error
	if ride.Owner.ID == currentUser.ID {
		return Ride{}, ErrNotImplemented
	}
	//start a new transaction and returns it as tx
	tx, err := s.database.BeginTx(ctx)
	//err gets reported if starting it failed
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()
	//returns the ride repository for tx
	ridesRepo := tx.RidesRepo()
	// get the error from the AddRideOccupant function in ridesRepo
	err = ridesRepo.AddRideOccupant(ctx, rideID, currentUser.ID)
	//if there is an error, return empty ride and error
	if err != nil {
		return Ride{}, ErrNotImplemented
	}
	// gets the details of the updated ride
	ride, err = getUserViewableRide(ctx, tx, currentUser.ID, rideID)
	//if there is an error, return an empty ride and the lookup or privacy error
	if err != nil {
		return Ride{}, ErrNotImplemented
	}
	//return updated ride
	return ride, nil

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
	return Ride{}, ErrNotImplemented
}
