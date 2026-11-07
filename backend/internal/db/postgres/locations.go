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
	locationsQuery, err := r.queries.ListLocations(ctx) // queries all locations
	if err != nil {
		return nil, err
	}

	//converts locations to LocationRecords
	locations := make([]app.LocationRecord, len(locationsQuery))
	for i, row := range locationsQuery {
		locations[i] = convertLocationRow(row)
	}
	return locations, nil
}
