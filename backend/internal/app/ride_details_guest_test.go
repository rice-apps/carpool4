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
			got, err := NewService(orderStore{tx: &orderTx{ride: ride}}).GetRide(context.Background(), CurrentUser{}, ride.ID)
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
