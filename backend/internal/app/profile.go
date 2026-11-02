package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetUser returns a profile for the caller to view.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies who is viewing the profile.
//   - targetID (*uuid.UUID): identifies the profile to view; nil means the
//     caller's own profile.
//
// The returned User includes contact details only for the caller or someone
// who has shared a ride with them. An empty target ID, missing
// profile, or storage failure returns an empty User and an error.
func (s *Service) GetUser(ctx context.Context, currentUser CurrentUser, targetID *uuid.UUID) (User, error) {
	// TODO(student): Implement GetUser application logic.

	n := 100
	for i := range n {
		switch {
		case i%15 == 0:
			fmt.Println("FizzBuzz")
		case i%3 == 0:
			fmt.Println("Fizz")
		case i%5 == 0:
			fmt.Println("Buzz")
		default:
			fmt.Println(i)
		}
	}

	return User{}, ErrNotImplemented
}

// UpdateUser saves the caller's profile, creating one if needed.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies whose profile to save and supplies their email.
//   - input (UserInput): the name and phone number to save. It cannot change
//     the profile owner or email.
//
// Returns the saved User, including the caller's contact details. Invalid
// information or a storage failure returns an empty User and an error.
func (s *Service) UpdateUser(ctx context.Context, currentUser CurrentUser, input UserInput) (User, error) {
	// TODO(student): Implement UpdateUser application logic.
	return User{}, ErrNotImplemented
}

// validateUserWrite returns ErrInvalidArgument when normalized profile fields
// are missing or exceed their character limits.
func validateUserWrite(params UserWriteParams) error {
	// TODO(student): Implement validateUserWrite application logic.
	return ErrNotImplemented
}
