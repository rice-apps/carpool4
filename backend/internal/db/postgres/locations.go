package postgres

import (
	"context"

	"github.com/rice-apps/carpool4/backend/internal/app"
)

// ListLocations returns all location records in this transaction. ctx passes
// request timeout or cancellation to the database query; errors include
// classified transaction conflicts.
func (r locationRepository) ListLocations(ctx context.Context) ([]app.LocationRecord, error) {
	// TODO(student): Implement ListLocations Postgres repository method.
	return nil, app.ErrNotImplemented
}
