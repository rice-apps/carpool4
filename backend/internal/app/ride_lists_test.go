package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestListRidesAppliesFiltersAndPrivacy checks location and time filters,
// contact visibility, and closure of the read transaction.
func TestListRidesAppliesFiltersAndPrivacy(t *testing.T) {
	viewer, owner, rider := uuid.New(), uuid.New(), uuid.New()
	departureID, arrivalID := uuid.New(), uuid.New()
	departure := time.Now().Add(2 * time.Hour).Truncate(time.Microsecond)
	rides := []LoadedRide{
		testLoadedRide(uuid.New(), owner, rider, departure),
		testLoadedRide(uuid.New(), owner, rider, departure.Add(time.Minute)),
	}
	store := &lookupStore{rides: rides, visible: map[uuid.UUID]bool{rider: true}}
	service := NewService(store)

	listed, err := service.ListRides(context.Background(), CurrentUser{ID: viewer}, ListRidesInput{
		DepartureLocationID: &departureID,
		ArrivalLocationID:   &arrivalID,
		DepartureAfter:      &departure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.query == nil || store.query.DepartureLocationID != &departureID || store.query.ArrivalLocationID != &arrivalID || !store.query.DepartureAfter.Equal(departure) {
		t.Fatalf("ListRides query = %#v, want filters", store.query)
	}
	if len(listed) != len(rides) {
		t.Fatalf("ListRides returned %d rides, want all %d", len(listed), len(rides))
	}
	if listed[0].Owner.Email != "" || listed[0].Riders[0].Email == "" {
		t.Fatalf("ListRides privacy projection = %#v", listed[0])
	}
	if len(store.visibleCalls) != 1 {
		t.Fatalf("ListContactVisibleUserIDs calls = %#v, want one batch", store.visibleCalls)
	}
	if !store.begun || !store.rolledBack {
		t.Fatal("ListRides did not close its read transaction")
	}
}

func TestListRidesAnonymousOmitsPeopleAndNotes(t *testing.T) {
	ownerID, passengerID := uuid.New(), uuid.New()
	ride := testLoadedRide(uuid.New(), ownerID, passengerID, time.Now().Add(time.Hour))
	ride.Riders = []UserRecord{ride.Owner, testUserRecord(passengerID, "passenger@rice.edu")}
	ride.Notes = "Call me at 713-555-0100"
	ride.Capacity = 2
	store := &lookupStore{rides: []LoadedRide{ride}}

	got, err := NewService(store).ListRides(context.Background(), CurrentUser{}, ListRidesInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Owner != (User{}) || len(got[0].Riders) != 0 || got[0].Notes != "" || got[0].OccupiedSeats != 2 {
		t.Fatalf("anonymous ride = %#v", got)
	}
	if len(store.visibleCalls) != 0 || !store.rolledBack {
		t.Fatalf("guest contact lookup or rollback = %#v", store)
	}
}

// TestListMyRidesUsesCurrentUser checks that the caller's verified ID selects their
// rides and that a successful read closes its transaction.
func TestListMyRidesUsesCurrentUser(t *testing.T) {
	viewer := uuid.New()
	store := &lookupStore{rides: []LoadedRide{testLoadedRide(uuid.New(), uuid.New(), viewer, time.Now())}}
	service := NewService(store)

	if _, err := service.ListMyRides(context.Background(), CurrentUser{ID: viewer}); err != nil {
		t.Fatal(err)
	}
	if store.myViewerID != viewer {
		t.Fatalf("ListMyRides viewer = %v, want currentUser %v", store.myViewerID, viewer)
	}
	if !store.begun || !store.rolledBack {
		t.Fatal("ListMyRides did not close its read transaction")
	}
}

// TestListRidesClampsEarlyDepartureAfterToLookback checks a caller cannot widen
// discovery past the lookback window by asking for an earlier departure bound.
func TestListRidesClampsEarlyDepartureAfterToLookback(t *testing.T) {
	store := &lookupStore{}
	stale := time.Now().Add(-30 * 24 * time.Hour)

	if _, err := NewService(store).ListRides(context.Background(), CurrentUser{ID: uuid.New()}, ListRidesInput{DepartureAfter: &stale}); err != nil {
		t.Fatal(err)
	}
	cutoff := time.Now().Add(-rideDiscoveryLookback)
	if store.query == nil || store.query.DepartureAfter.Before(cutoff.Add(-time.Minute)) {
		t.Fatalf("DepartureAfter = %v, want it clamped up to the %v lookback", store.query, rideDiscoveryLookback)
	}
}

// TestListRidesUsesLookbackWhenNoDepartureAfter checks the default discovery cutoff.
func TestListRidesUsesLookbackWhenNoDepartureAfter(t *testing.T) {
	store := &lookupStore{}

	if _, err := NewService(store).ListRides(context.Background(), CurrentUser{ID: uuid.New()}, ListRidesInput{}); err != nil {
		t.Fatal(err)
	}
	cutoff := time.Now().Add(-rideDiscoveryLookback)
	if store.query == nil || store.query.DepartureAfter.Before(cutoff.Add(-time.Minute)) || store.query.DepartureAfter.After(cutoff.Add(time.Minute)) {
		t.Fatalf("DepartureAfter = %v, want approximately the %v cutoff", store.query, rideDiscoveryLookback)
	}
}
