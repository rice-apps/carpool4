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
//   - currentUser (CurrentUser): identifies the viewer, or is empty for a guest.
//   - input (ListRidesInput): optionally filters departure and arrival
//     locations and departure times.
//
// Results may include rides departed within the past hour, but never rides
// older than input.DepartureAfter when that bound is later. Contacts are shown
// only when currentUser may see them. Guest results omit people and notes. Storage
// failures return an error.
func (s *Service) ListRides(ctx context.Context, currentUser CurrentUser, input ListRidesInput) ([]Ride, error) {

	//initialize realDepartureTime to one hour before, change to preferred time if preferred is sooner
	realDepartureTime := time.Now().Add(-rideDiscoveryLookback)
	if input.DepartureAfter != nil {
		preferredTime := *input.DepartureAfter
		if preferredTime.Before(realDepartureTime) {
			realDepartureTime = preferredTime
		}
	}

	//create new ListRidesQuery object and populate with fields from input
	ridesQuery := ListRidesQuery{
		DepartureLocationID: input.DepartureLocationID,
		ArrivalLocationID:   input.ArrivalLocationID,
		DepartureBefore:     input.DepartureBefore,
		DepartureAfter:      realDepartureTime,
	}

	//start new transaction and call database layer ListRides
	newTransaction, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	loadedRides, err := newTransaction.RidesRepo().ListRides(ctx, ridesQuery)
	if err != nil {
		return nil, err
	}
	//convert database layer ListRides output to []Rides
	//helper prepares info based on user's allowed context, takes care of guest case
	rideList, err := buildRidesForViewer(ctx, newTransaction, currentUser.ID, loadedRides)
	if err != nil {
		return nil, err
	}

	return rideList, nil
}

// ListMyRides returns rides owned or joined by currentUser.
//
// Inputs:
//   - ctx (context.Context): passes the request's timeout or cancellation
//     to the database calls.
//   - currentUser (CurrentUser): identifies whose rides to return and which contacts
//     they may see.
//
// The returned rides show only allowed contacts. A storage failure returns an error.
func (s *Service) ListMyRides(ctx context.Context, currentUser CurrentUser) ([]Ride, error) {
	// TODO(student): Implement ListMyRides application logic.
	return nil, ErrNotImplemented
}
