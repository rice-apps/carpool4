package app

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()

	err = validateRideWrite(input)
	if err != nil {
		return Ride{}, err
	}

	user, err := tx.UsersRepo().FindUser(ctx, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}

	if strings.TrimSpace(user.Phone) == "" {
		return Ride{}, ErrPhoneRequired
	}

	ride, err := tx.RidesRepo().CreateRide(ctx, currentUser.ID, input)
	if err != nil {
		return Ride{}, err
	}

	err = tx.RidesRepo().AddRideOccupant(ctx, ride, currentUser.ID)
	if err != nil {
		return Ride{}, err
	}

	ride_object, err := getUserViewableRide(ctx, tx, currentUser.ID, ride)
	if err != nil {
		return Ride{}, err
	}

	err = tx.Commit()
	if err != nil {
		return Ride{}, err
	}

	return ride_object, nil
}

// validateRideWrite returns ErrInvalidArgument unless input has a future
// departure, two distinct locations, capacity of at least one, and notes of
// at most 500 characters.
func validateRideWrite(input RideWriteParams) error {
	// TODO(student): Implement validateRideWrite application logic.\
	switch {
	case input.DepartureTime.Before(time.Now()):
		return fmt.Errorf("departure %w is in the past", ErrInvalidArgument)
	case input.ArrivalLocationID == input.DepartureLocationID:
		return fmt.Errorf("arrival and departure locations must differ: %w", ErrInvalidArgument)
	case input.ArrivalLocationID == uuid.Nil:
		return fmt.Errorf("arrival location is required: %w", ErrInvalidArgument)
	case input.DepartureLocationID == uuid.Nil:
		return fmt.Errorf("departure location is required: %w", ErrInvalidArgument)
	case input.Capacity < 1:
		return ErrCapacityBelowOccupancy
	case utf8.RuneCountInString(input.Notes) > 500:
		return fmt.Errorf("notes must be below 500 characters: %w", ErrInvalidArgument)
	default:
		return nil
	}
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
