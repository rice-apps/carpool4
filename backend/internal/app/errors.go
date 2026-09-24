package app

import "errors"

var (
	// ErrNotImplemented marks an operation students have not completed yet.
	ErrNotImplemented = errors.New("operation not implemented")
	// ErrUnauthenticated means the caller has no verified user ID.
	ErrUnauthenticated = errors.New("missing caller identity")
	// ErrInvalidArgument means a required value is missing or invalid.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrPermissionDenied means the caller cannot change the requested resource.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrPhoneRequired means a ride owner needs a profile with a phone number.
	ErrPhoneRequired = errors.New("phone number required before creating a ride")
	// ErrRideCancelled means a cancelled ride cannot be updated or joined.
	ErrRideCancelled = errors.New("this ride has been cancelled")
	// ErrCapacityBelowOccupancy means the new capacity is too small for current riders.
	ErrCapacityBelowOccupancy = errors.New("capacity cannot be below current occupancy")
	// ErrOwnerCannotChangeMembership means an owner cannot join or leave their ride.
	ErrOwnerCannotChangeMembership = errors.New("owners cannot join or leave their own ride")
	// ErrAlreadyJoined means the caller is already a rider on the ride.
	ErrAlreadyJoined = errors.New("you're already part of this ride")
	// ErrFullCapacity means the ride has no room for another rider.
	ErrFullCapacity = errors.New("this ride is full")
	// ErrNotJoined means the caller is not a rider on the ride.
	ErrNotJoined = errors.New("you're not part of this ride")
)
