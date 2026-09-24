package app

import (
	"context"
)

// ListLocations returns the available pickup and destination locations.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies the caller allowed to request locations.
//
// It returns location records or an error if the caller is unknown or storage
// fails.
func (s *Service) ListLocations(ctx context.Context, actor Actor) ([]LocationRecord, error) {
	// TODO(student): Implement ListLocations application logic.
	return nil, ErrNotImplemented
}
