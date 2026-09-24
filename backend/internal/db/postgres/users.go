package postgres

import (
	"context"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/google/uuid"
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
	return app.UserRecord{}, app.ErrNotImplemented
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
	return app.UserRecord{}, app.ErrNotImplemented
}
