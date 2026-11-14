package app

import (
	"context"

	"github.com/google/uuid"
)

// CreateRide creates a ride owned by currentUser and returns it with contacts the
// caller may see.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies the ride's owner.
//   - input (RideWriteParams): supplies the departure time, locations,
//     capacity, and optional notes.
//
// The owner must have a profile with a phone number and counts as an occupant.
// The departure must be in the future, locations must differ, and capacity
// must be at least one. Invalid input, a missing phone, or a storage failure
// returns an empty Ride and an error.
func (s *Service) CreateRide(ctx context.Context, currentUser CurrentUser, input RideWriteParams) (Ride, error) {
	// TODO(student): Implement CreateRide application logic.
	return Ride{}, ErrNotImplemented
}

// validateRideWrite returns ErrInvalidArgument unless input has a future
// departure, two distinct locations, capacity of at least one, and notes of
// at most 500 characters.
func validateRideWrite(input RideWriteParams) error {
	// TODO(student): Implement validateRideWrite application logic.
	return ErrNotImplemented
}

// UpdateRide replaces the editable fields of id and returns the updated Ride
// with contacts currentUser may see.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies the caller who must own the ride.
//   - id (uuid.UUID): identifies the ride to update.
//   - input (RideWriteParams): supplies replacement departure, locations,
//     capacity, and notes.
//
// The ride must be active, input must be valid, and capacity must fit current
// occupants. A missing ride, denied owner, cancelled ride, invalid
// input, or storage failure returns an empty Ride and an error.
func (s *Service) UpdateRide(ctx context.Context, currentUser CurrentUser, id uuid.UUID, input RideWriteParams) (Ride, error) {
	// TODO(student): Implement UpdateRide application logic.
	return Ride{}, ErrNotImplemented
}

// CancelRide cancels a ride and returns it with contacts currentUser may see.
// HELPFUL NOTES:
// change the status of the ride in the database to cancel, returns canceled ride but w/ update the privacy
// update privacy with Get User Viewable Rides helper method
// Use the loadedride function somewhere
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies the caller who must own the ride.
//   - id (uuid.UUID): identifies the ride to cancel.
//
// A missing ride, denied owner, or storage failure returns an empty
// Ride and an error.
func (s *Service) CancelRide(ctx context.Context, currentUser CurrentUser, id uuid.UUID) (Ride, error) {
	// TODO(student): Implement CancelRide application logic.
	// Note: this code comes from LeaveRide method in app layer -- may need to change
	// checking to see if ID is Nil or not
	if id == uuid.Nil {
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

	numberOfRemovedRider, err := ridesRepo.RemoveRideOccupant(ctx, id, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}
	// error handling
	if numberOfRemovedRider == 0 {
		return Ride{}, ErrNotImplemented
	}
	// using getUserViewableRide to take in the rideID and remove the rider using removeRider
	updateRide, err := getUserViewableRide(ctx, tx, id, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}
	// use tx.commit()
	if err := tx.Commit(); err != nil {
		return Ride{}, err
	}
	return updateRide, nil
}
