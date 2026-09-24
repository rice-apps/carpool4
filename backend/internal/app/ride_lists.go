package app

import (
	"context"
	"time"
)

// rideDiscoveryLookback is how long departed rides remain in discovery results.
const rideDiscoveryLookback = time.Hour

// ListRides returns rides matching input for actor to view.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - actor (Actor): identifies who is viewing the rides.
//   - input (ListRidesInput): optionally filters departure and arrival
//     locations and departure times.
//
// Results may include rides departed within the past hour, but never rides
// older than input.DepartureAfter when that bound is later. Contacts are shown
// only when actor may see them. A missing caller or storage failure returns
// an error.
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
