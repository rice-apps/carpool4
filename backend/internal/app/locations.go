package app

import (
	"context"
)

// ListLocations returns the available pickup and destination locations.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//
// It returns location records or a storage error.
func (s *Service) ListLocations(ctx context.Context) ([]LocationRecord, error) {
	// TODO(student): Implement ListLocations application logic.
	// start transaction
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	// get locations repo
	locationsRepo, err := tx.LocationsRepo().ListLocations(ctx)
	if err != nil {
		return nil, err
	}

	return locationsRepo, nil
}
