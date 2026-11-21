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
	//if ride ID is not valid, return empty ride and error
	if rideID == uuid.Nil {
		return Ride{}, fmt.Errorf("%w: ride ID", ErrInvalidArgument)
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
	// gets the current ride details
	ride, err := ridesRepo.GetRide(ctx, rideID)
	// if there is an error with currentUser then return nil and the error
	if err != nil {
		return Ride{}, err
	}
	//check if the ride status is active and if not, return empty ride and error
	if ride.Status != RideStatusActive {
		return Ride{}, ErrRideCancelled
	}
	//check if the original owner is the current user and if so, return empty ride and error
	if ride.Owner.ID == currentUser.ID {
		return Ride{}, ErrOwnerCannotChangeMembership
	}
	// if current user is already in the ride then return empty ride and error
	for _, rider := range ride.Riders {
		if rider.ID == currentUser.ID {
			return Ride{}, ErrAlreadyJoined
		}
	}
	//check if number of riders is greater than or equal to ride capacity and if so, return empty ride and error
	if len(ride.Riders) >= int(ride.Capacity) {
		return Ride{}, ErrFullCapacity
	}
	// get the error from the AddRideOccupant function in ridesRepo
	err = ridesRepo.AddRideOccupant(ctx, rideID, currentUser.ID)
	//if there is an error, return empty ride and error
	if err != nil {
		return Ride{}, err
	}
	// gets the details of the updated ride
	updatedRide, err := getUserViewableRide(ctx, tx, currentUser.ID, rideID)
	//if there is an error, return an empty ride and the lookup or privacy error
	if err != nil {
		return Ride{}, err
	}
	//return updated ride
	if err := tx.Commit(); err != nil {
		return Ride{}, err
	}
	return updatedRide, nil
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
	// HIGH-LEVEL of func: 1) Checking current size of ride and making sure owner will not be removed
	// 2) Calling database to try and remove user 3) Checking if someone was actually removed
	// 4) getting the current state of the ride using getUserViewableRide

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
		return Ride{}, ErrOwnerCannotChangeMembership
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
