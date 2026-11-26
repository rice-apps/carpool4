package app

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

// TestGetRideGuestDiscoveryBoundary checks which rides a signed-out visitor may
// open. Cancelled and long-departed rides are reported missing rather than
// refused, so browsing cannot be used to enumerate them.
func TestGetRideGuestDiscoveryBoundary(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		status    RideStatus
		departure time.Time
		wantFound bool
	}{
		{name: "upcoming", status: RideStatusActive, departure: time.Now().Add(time.Hour), wantFound: true},
		{name: "departed within the lookback", status: RideStatusActive, departure: time.Now().Add(-rideDiscoveryLookback / 2), wantFound: true},
		{name: "cancelled", status: RideStatusCancelled, departure: time.Now().Add(time.Hour)},
		{name: "departed before the lookback", status: RideStatusActive, departure: time.Now().Add(-2 * rideDiscoveryLookback)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ride := testRide(uuid.New(), uuid.New(), tc.departure)
			ride.Status = tc.status
			ride.Notes = "private note"
			fake := newFakeDatabase([]UserRecord{ride.Owner}, ride).showingContacts()

			got, err := NewService(fake).GetRide(t.Context(), CurrentUser{}, ride.ID)
			if !tc.wantFound {
				if !errors.Is(err, ErrRideNotFound) {
					t.Fatalf("GetRide(%v) error = %v, want ErrRideNotFound", ride.ID, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetRide(%v) error = %v", ride.ID, err)
			}

			want := Ride{
				ID:                ride.ID,
				DepartureTime:     ride.DepartureTime,
				DepartureLocation: ride.Departure,
				ArrivalLocation:   ride.Arrival,
				Capacity:          ride.Capacity,
				OccupiedSeats:     1,
				Status:            tc.status,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("guest GetRide(%v) mismatch (-want +got):\n%s", ride.ID, diff)
			}
		})
	}
}

func TestGetRideAppliesContactPermissions(t *testing.T) {
	t.Parallel()
	owner, stranger := uuid.New(), uuid.New()
	ride := testRide(uuid.New(), owner, time.Now().Add(time.Hour))
	ride.Notes = "meet at the north gate"

	for _, tc := range []struct {
		name         string
		viewer       uuid.UUID
		wantContacts bool
	}{
		{name: "owner sees their own contacts", viewer: owner, wantContacts: true},
		{name: "stranger sees a redacted owner", viewer: stranger},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := newFakeDatabase([]UserRecord{ride.Owner}, ride)

			got, err := NewService(fake).GetRide(t.Context(), CurrentUser{ID: tc.viewer}, ride.ID)
			if err != nil {
				t.Fatalf("GetRide(%v) error = %v", ride.ID, err)
			}

			wantOwner := User{ID: owner, FirstName: ride.Owner.FirstName, LastName: ride.Owner.LastName}
			if tc.wantContacts {
				wantOwner.Email, wantOwner.Phone = ride.Owner.Email, ride.Owner.Phone
			}
			if diff := cmp.Diff(wantOwner, got.Owner); diff != "" {
				t.Errorf("GetRide(%v) owner mismatch (-want +got):\n%s", ride.ID, diff)
			}
			// Notes reach every signed-in viewer, unlike contact details.
			if got.Notes != ride.Notes {
				t.Errorf("notes = %q, want %q", got.Notes, ride.Notes)
			}
		})
	}
}

func TestGetRideRejectsEmptyRideID(t *testing.T) {
	t.Parallel()
	fake := newFakeDatabase(nil)

	if _, err := NewService(fake).GetRide(t.Context(), CurrentUser{ID: uuid.New()}, uuid.Nil); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("GetRide(uuid.Nil) error = %v, want ErrInvalidArgument", err)
	}
	if fake.called(opBegin) {
		t.Error("GetRide opened a transaction for an empty ride ID")
	}
}
