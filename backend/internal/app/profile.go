package app

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()

	usersRepo := tx.UsersRepo()
	if targetID == nil {
		targetID = &currentUser.ID
	}

	userRecord, err := usersRepo.FindUser(ctx, *targetID)
	if err != nil {
		return User{}, err
	}

	profile, err := getVisibleProfile(ctx, tx, currentUser, userRecord)
	if err != nil {
		return User{}, err
	}
	return profile, nil
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
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()

	usersRepo := tx.UsersRepo()

	userWriteParams := UserWriteParams{
		FirstName: strings.TrimSpace(input.FirstName),
		LastName:  strings.TrimSpace(input.LastName),
		Phone:     strings.TrimSpace(input.Phone),
		ID:        currentUser.ID,
		Email:     strings.TrimSpace(currentUser.Email),
	}
	if validateUserWrite(userWriteParams) != nil {
		return User{}, validateUserWrite(userWriteParams)
	}
	userRecord, err := usersRepo.UpsertUser(ctx, userWriteParams)
	if err != nil {
		return User{}, err
	}

	profile, err := getVisibleProfile(ctx, tx, currentUser, userRecord)
	if err != nil {
		return User{}, err
	}

	if err := tx.Commit(); err != nil {
		return User{}, err
	}

	return profile, nil
}

// validateUserWrite returns ErrInvalidArgument when normalized profile fields
// are missing or exceed their character limits.
func validateUserWrite(params UserWriteParams) error {
	// TODO(student): Implement validateUserWrite application logic.
	firstNameLen := utf8.RuneCountInString(params.FirstName)
	lastNameLen := utf8.RuneCountInString(params.LastName)
	phoneLen := utf8.RuneCountInString(params.Phone)
	emailLen := utf8.RuneCountInString(params.Email)
	if !(1 <= firstNameLen && firstNameLen <= 100) {
		return fmt.Errorf("%w: first name must be less than 100 characters", ErrInvalidArgument)
	}
	if !(1 <= lastNameLen && lastNameLen <= 100) {
		return fmt.Errorf("%w: last name must be less than 100 characters", ErrInvalidArgument)
	}
	if phoneLen > 32 {
		return fmt.Errorf("%w: phone must be less than 32 characters", ErrInvalidArgument)
	}
	if !(1 <= emailLen && emailLen <= 254) {
		return fmt.Errorf("%w: email must be less than 254 characters", ErrInvalidArgument)
	}
	return nil
}
