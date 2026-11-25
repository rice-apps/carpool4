package rpc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
)

func TestConvertRideToProtoKeepsGuestOwnerAbsent(t *testing.T) {
	ride := app.Ride{ID: uuid.New(), DepartureTime: time.Now(), Capacity: 2, OccupiedSeats: 1}
	got := convertRideToProto(ride)
	if got.Owner != nil || len(got.Riders) != 0 || got.OccupiedSeats != 1 {
		t.Fatalf("guest ride = %#v", got)
	}
}
