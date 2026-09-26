package rpc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
)

func TestRideToProtoKeepsGuestOwnerAbsent(t *testing.T) {
	ride := app.Ride{ID: uuid.New(), DepartureTime: time.Now(), Capacity: 2, OccupiedSeats: 1}
	got := rideToProto(ride)
	if got.Owner != nil || len(got.Riders) != 0 || got.OccupiedSeats != 1 {
		t.Fatalf("guest ride = %#v", got)
	}
}

func TestRideToProtoKeepsSignedInOwner(t *testing.T) {
	ownerID := uuid.New()
	ride := app.Ride{
		ID: uuid.New(), DepartureTime: time.Now(),
		Owner: app.User{ID: ownerID, FirstName: "Alex"}, OccupiedSeats: 1,
	}
	got := rideToProto(ride)
	if got.Owner == nil || got.Owner.Id != ownerID.String() || got.OccupiedSeats != 1 {
		t.Fatalf("signed-in ride = %#v", got)
	}
}
