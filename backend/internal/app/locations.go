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
	return nil, ErrNotImplemented
}
