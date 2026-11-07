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
	err := validateRideWrite(input) // calls validateRideWrite, checks for error.
	if err != nil {
		return Ride{}, err
	}

	tx, err := s.database.BeginTx(ctx) // starts a transaction to ensure consistency
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()
	rideRepo := tx.RidesRepo()

	ride, err := s.GetRide(ctx, currentUser, id) // calls GetRide for current ride.
	if err != nil {
		return Ride{}, err
	}

	rideOwner := ride.Owner
	currentUserID := currentUser.ID
	rideOwnerID := rideOwner.ID
	if currentUserID != rideOwnerID { // checks to ensure the person updating the ride is the owner; permission check.
		return Ride{}, err
	}

	if ride.Status == RideStatusCancelled { // checks to ensure the Ride Status is not Cancelled (ensures ride is active)
		return Ride{}, err
	}

	if ride.OccupiedSeats > input.Capacity { // checks to ensure new capacity is at least the current occupied seats
		return Ride{}, err
	}

	err = rideRepo.UpdateRide(ctx, ride.ID, input) // calls DB layer and check for errors in updating
	if err != nil {
		return Ride{}, err
	}
	return ride, err
}

// CancelRide cancels a ride and returns it with contacts currentUser may see.
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
	return Ride{}, ErrNotImplemented
}
