package app

import (
	"context"
	"github.com/google/uuid"
	"time"
)

// lookupStore holds ride-list fixtures and records query and privacy inputs.
//
// Fields:
//   - orderTx: fallback repository methods that these tests must not reach.
//   - rides: fixture rides returned by list queries.
//   - query: last ride-discovery filters received.
//   - myViewerID: user ID passed to the personal ride list.
//   - visible: which users' contacts the viewer may see.
//   - visibleCalls: target ID batches passed to the visibility lookup.
//   - begun: whether the service started a transaction.
//   - rolledBack: whether the service closed the read transaction.
type lookupStore struct {
	orderTx
	rides        []LoadedRide
	query        *ListRidesQuery
	myViewerID   uuid.UUID
	visible      map[uuid.UUID]bool
	visibleCalls [][]uuid.UUID
	begun        bool
	rolledBack   bool
}

// BeginTx marks the fixture read as started and returns this transaction.
func (s *lookupStore) BeginTx(context.Context) (Transaction, error) {
	s.begun = true
	return s, nil
}

// UsersRepo exposes contact visibility on this fixture.
func (s *lookupStore) UsersRepo() UserRepository { return s }

// RidesRepo exposes ride-list methods on this fixture.
func (s *lookupStore) RidesRepo() RideRepository { return s }

// LocationsRepo exposes inherited location behavior on this fixture.
func (s *lookupStore) LocationsRepo() LocationRepository { return s }

// Rollback records closure of the read transaction.
func (s *lookupStore) Rollback() error {
	s.rolledBack = true
	return nil
}

// GetRide panics because these list fixtures must not perform a detail lookup.
func (*lookupStore) GetRide(context.Context, uuid.UUID) (LoadedRide, error) {
	panic("unexpected ride lookup")
}

// ListRides records query filters and returns the fixture rides.
func (s *lookupStore) ListRides(_ context.Context, query ListRidesQuery) ([]LoadedRide, error) {
	s.query = &query
	return s.rides, nil
}

// ListMyRides records the requested viewer ID and returns the fixture rides.
func (s *lookupStore) ListMyRides(_ context.Context, viewerID uuid.UUID) ([]LoadedRide, error) {
	s.myViewerID = viewerID
	return s.rides, nil
}

// ListContactVisibleUserIDs records the batch of targets and returns only IDs
// enabled by the fixture's visibility map.
func (s *lookupStore) ListContactVisibleUserIDs(_ context.Context, _ uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	s.visibleCalls = append(s.visibleCalls, append([]uuid.UUID(nil), targets...))
	result := make([]uuid.UUID, 0, len(targets))
	for _, id := range targets {
		if s.visible[id] {
			result = append(result, id)
		}
	}
	return result, nil
}

// testLoadedRide returns a fixture ride with the given IDs and departure time.
func testLoadedRide(id, ownerID, riderID uuid.UUID, departure time.Time) LoadedRide {
	return LoadedRide{
		ID: id, DepartureTime: departure, Capacity: 3, Status: RideStatusActive,
		Owner:     testUserRecord(ownerID, "owner@rice.edu"),
		Riders:    []UserRecord{testUserRecord(riderID, "rider@rice.edu")},
		Departure: LocationRecord{ID: uuid.New(), Title: "Rice"},
		Arrival:   LocationRecord{ID: uuid.New(), Title: "IAH"},
	}
}
