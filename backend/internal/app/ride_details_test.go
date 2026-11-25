package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetRideAnonymousDiscoveryBoundary(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       RideStatus
		departure    time.Time
		wantNotFound bool
	}{
		{"future", RideStatusActive, time.Now().Add(time.Hour), false},
		{"cancelled", RideStatusCancelled, time.Now().Add(time.Hour), true},
		{"old", RideStatusActive, time.Now().Add(-2 * time.Hour), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ride := testLoadedRide(uuid.New(), uuid.New(), uuid.New(), tc.departure)
			ride.Status = tc.status
			ride.Riders = []UserRecord{ride.Owner}
			ride.Notes = "private note"
			got, err := NewService(&lookupStore{rides: []LoadedRide{ride}}).GetRide(context.Background(), CurrentUser{}, ride.ID)
			if tc.wantNotFound {
				if !errors.Is(err, ErrRideNotFound) {
					t.Fatalf("GetRide error = %v, want ride not found", err)
				}
				return
			}
			if err != nil || got.Owner != (User{}) || len(got.Riders) != 0 || got.Notes != "" || got.OccupiedSeats != 1 {
				t.Fatalf("anonymous GetRide = %#v, %v", got, err)
			}
		})
	}
}

// TestGetRideShowsViewerAllowedContacts checks a signed-in viewer receives trip
// details plus the contacts their shared ride entitles them to.
func TestGetRideShowsViewerAllowedContacts(t *testing.T) {
	ownerID, strangerID := uuid.New(), uuid.New()
	ride := testLoadedRide(uuid.New(), ownerID, uuid.New(), time.Now().Add(time.Hour))
	ride.Riders = []UserRecord{ride.Owner}
	ride.Notes = "meet at the north gate"

	t.Run("rider sees owner contact", func(t *testing.T) {
		store := &lookupStore{visible: map[uuid.UUID]bool{ownerID: true}}
		store.rides = []LoadedRide{ride}
		got, err := NewService(store).GetRide(context.Background(), CurrentUser{ID: ownerID}, ride.ID)
		if err != nil || got.Owner.ID != ownerID || got.Owner.Email == "" || got.Notes != ride.Notes {
			t.Fatalf("GetRide = %#v, %v; want the owner's contacts and notes", got, err)
		}
	})

	t.Run("stranger sees redacted owner", func(t *testing.T) {
		store := &lookupStore{visible: map[uuid.UUID]bool{}}
		store.rides = []LoadedRide{ride}
		got, err := NewService(store).GetRide(context.Background(), CurrentUser{ID: strangerID}, ride.ID)
		if err != nil || got.Owner.ID != ownerID || got.Owner.Email != "" || got.Owner.Phone != "" {
			t.Fatalf("GetRide = %#v, %v; want the owner without contact fields", got, err)
		}
	})
}
