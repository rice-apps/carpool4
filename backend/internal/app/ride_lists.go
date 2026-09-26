package app

import (
	"context"
	"time"
)

// rideDiscoveryLookback is how long departed rides remain in discovery results.
const rideDiscoveryLookback = time.Hour

// ListRides returns rides matching input for an authenticated or anonymous viewer.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies the viewer, or is empty for a guest.
//   - input (ListRidesInput): optionally filters departure and arrival
//     locations and departure times.
//
// Results may include rides departed within the past hour, but never rides
// older than input.DepartureAfter when that bound is later. Contacts are shown
// only when actor may see them. Guest results omit people and notes. Storage
// failures return an error.
func (s *Service) ListRides(ctx context.Context, actor Actor, input ListRidesInput) ([]Ride, error) {
	// TODO(student): Implement ListRides application logic.
	return nil, ErrNotImplemented
}

// ListMyRides returns rides owned or joined by actor.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies whose rides to return and which contacts
//     they may see.
//
// The returned rides show only allowed contacts. A missing caller or storage
// failure returns an error.
func (s *Service) ListMyRides(ctx context.Context, actor Actor) ([]Ride, error) {
	// TODO(student): Implement ListMyRides application logic.
	return nil, ErrNotImplemented
}
