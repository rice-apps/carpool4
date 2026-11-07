package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
)

// FindUser returns a profile without contact projection.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database query.
//   - id (uuid.UUID): the profile to load.
//
// It returns an error on failure, including app.ErrUserNotFound for a missing
// profile.
func (u userRepository) FindUser(ctx context.Context, id uuid.UUID) (app.UserRecord, error) {
	// TODO(student): Implement FindUser Postgres repository method.
	user, err := u.queries.FindUser(ctx, id)
	if err != nil {
		return app.UserRecord{}, err
	}
	record := convertUserRow(user)
	return record, nil
}

// UpsertUser creates or updates a profile from params.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database write.
//   - params (app.UserWriteParams): profile ID and fields to store.
//
// It returns the stored record or an error, including a classified transaction
// conflict.
func (u userRepository) UpsertUser(ctx context.Context, params app.UserWriteParams) (app.UserRecord, error) {
	// TODO(student): Implement UpsertUser Postgres repository method.

	userParams := sqlc.UpsertUserParams{
		ID:        params.ID,
		Email:     params.Email,
		FirstName: params.FirstName,
		LastName:  params.LastName,
		Phone:     params.Phone,
	}
	user, err := u.queries.UpsertUser(ctx, userParams)
	if err != nil {
		return app.UserRecord{}, err
	}
	record := convertUserRow(user)
	return record, nil
}
