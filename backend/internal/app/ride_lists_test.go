package app

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestListRidesAppliesFiltersAndContactPermissions(t *testing.T) {
	t.Parallel()
	viewer, owner, rider := uuid.New(), uuid.New(), uuid.New()
	departureID, arrivalID := uuid.New(), uuid.New()
	departure := time.Now().Add(2 * time.Hour).Truncate(time.Microsecond)

	ride := testRide(uuid.New(), owner, departure)
	ride.Riders = append(ride.Riders, testUser(rider, "rider@rice.edu"))
	fake := newFakeDatabase([]UserRecord{ride.Owner, ride.Riders[1]}, ride)
	fake.visible = func(_, target uuid.UUID) bool { return target == rider }

	got, err := NewService(fake).ListRides(t.Context(), CurrentUser{ID: viewer}, ListRidesInput{
		DepartureLocationID: &departureID,
		ArrivalLocationID:   &arrivalID,
		DepartureAfter:      &departure,
	})
	if err != nil {
		t.Fatalf("ListRides() error = %v", err)
	}

	wantQuery := ListRidesQuery{
		DepartureLocationID: &departureID,
		ArrivalLocationID:   &arrivalID,
		DepartureAfter:      departure,
	}
	if diff := cmp.Diff(wantQuery, *fake.tx.ridesQuery); diff != "" {
		t.Errorf("ListRides query mismatch (-want +got):\n%s", diff)
	}
	if len(got) != 1 {
		t.Fatalf("ListRides() returned %d rides, want 1", len(got))
	}
	if got[0].Owner.Email != "" {
		t.Errorf("owner email = %q, want it hidden from an unrelated viewer", got[0].Owner.Email)
	}
	if got[0].Riders[1].Email == "" {
		t.Error("rider email is hidden, want it visible to a viewer who may see this rider")
	}
	// Contact permissions resolve in one batch for the whole page of rides.
	if len(fake.tx.contactTargets) != 1 {
		t.Errorf("contact permission lookups = %d, want 1", len(fake.tx.contactTargets))
	}
	requireReadOnly(t, fake)
}

func TestListRidesHidesPeopleAndNotesFromGuests(t *testing.T) {
	t.Parallel()
	owner, passenger := uuid.New(), uuid.New()
	ride := testRide(uuid.New(), owner, time.Now().Add(time.Hour))
	ride.Riders = append(ride.Riders, testUser(passenger, "passenger@rice.edu"))
	ride.Notes = "Call me at 713-555-0100"
	ride.Capacity = 2
	fake := newFakeDatabase([]UserRecord{ride.Owner, ride.Riders[1]}, ride).showingContacts()

	got, err := NewService(fake).ListRides(t.Context(), CurrentUser{}, ListRidesInput{})
	if err != nil {
		t.Fatalf("ListRides() error = %v", err)
	}

	want := []Ride{{
		ID:                ride.ID,
		DepartureTime:     ride.DepartureTime,
		DepartureLocation: ride.Departure,
		ArrivalLocation:   ride.Arrival,
		Capacity:          2,
		OccupiedSeats:     2,
		Status:            RideStatusActive,
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("guest ListRides mismatch (-want +got):\n%s", diff)
	}
	// A guest has no identity to check contacts against, so asking would leak
	// the question itself to the database.
	if fake.called(opVisibleContacts) {
		t.Error("ListRides consulted contact permissions for a guest, want no lookup")
	}
}

// TestListRidesClampsDepartureAfterToLookback checks that the discovery window
// cannot be widened past the lookback, however early a bound the caller asks for.
func TestListRidesClampsDepartureAfterToLookback(t *testing.T) {
	t.Parallel()
	stale := time.Now().Add(-30 * 24 * time.Hour)

	for _, tc := range []struct {
		name  string
		input ListRidesInput
	}{
		{name: "no bound supplied", input: ListRidesInput{}},
		{name: "bound earlier than the lookback", input: ListRidesInput{DepartureAfter: &stale}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := newFakeDatabase(nil)

			if _, err := NewService(fake).ListRides(t.Context(), CurrentUser{ID: uuid.New()}, tc.input); err != nil {
				t.Fatalf("ListRides() error = %v", err)
			}

			cutoff := time.Now().Add(-rideDiscoveryLookback)
			got := fake.tx.ridesQuery.DepartureAfter
			if got.Before(cutoff.Add(-time.Minute)) || got.After(cutoff.Add(time.Minute)) {
				t.Errorf("DepartureAfter = %v, want approximately the %v cutoff %v", got, rideDiscoveryLookback, cutoff)
			}
		})
	}
}

// TestListRidesKeepsLaterDepartureAfter checks the caller can narrow the window.
func TestListRidesKeepsLaterDepartureAfter(t *testing.T) {
	t.Parallel()
	soon := time.Now().Add(2 * time.Hour).Truncate(time.Microsecond)
	fake := newFakeDatabase(nil)

	if _, err := NewService(fake).ListRides(t.Context(), CurrentUser{ID: uuid.New()}, ListRidesInput{DepartureAfter: &soon}); err != nil {
		t.Fatalf("ListRides() error = %v", err)
	}

	if got := fake.tx.ridesQuery.DepartureAfter; !got.Equal(soon) {
		t.Errorf("DepartureAfter = %v, want the caller's later bound %v", got, soon)
	}
}

func TestListMyRidesUsesVerifiedCaller(t *testing.T) {
	t.Parallel()
	viewer := uuid.New()
	fake := newFakeDatabase(nil, testRide(uuid.New(), uuid.New(), time.Now()))

	if _, err := NewService(fake).ListMyRides(t.Context(), CurrentUser{ID: viewer}); err != nil {
		t.Fatalf("ListMyRides() error = %v", err)
	}

	if fake.tx.myRidesViewer != viewer {
		t.Errorf("ListMyRides queried viewer %v, want the verified caller %v", fake.tx.myRidesViewer, viewer)
	}
	requireReadOnly(t, fake)
}
