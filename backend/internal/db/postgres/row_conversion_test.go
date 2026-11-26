package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
)

// TestConvertRideRowsBuildsTheWholeRide checks every field the application
// reads, including the riders decoded from the query's JSON aggregate.
func TestConvertRideRowsBuildsTheWholeRide(t *testing.T) {
	t.Parallel()
	owner := sqlc.User{ID: uuid.New(), FirstName: "Owner"}
	rider := sqlc.User{ID: uuid.New(), FirstName: "Rider"}
	riders, err := json.Marshal([]sqlc.User{rider})
	if err != nil {
		t.Fatalf("marshal riders: %v", err)
	}
	departure := sqlc.Location{ID: uuid.New(), Title: "Rice", Address: "6100 Main"}
	arrival := sqlc.Location{ID: uuid.New(), Title: "IAH", Address: "Airport"}
	departureTime := time.Now().Truncate(time.Microsecond)
	ride := sqlc.Ride{
		ID: uuid.New(), OwnerID: owner.ID, DepartureTime: departureTime,
		DepartureLocationID: departure.ID, ArrivalLocationID: arrival.ID,
		Notes: "meet outside", Capacity: 3, Status: sqlc.RideStatusActive,
	}

	got, err := convertRideRows(ride, owner, departure, arrival, riders)
	if err != nil {
		t.Fatalf("convertRideRows() error = %v", err)
	}

	want := app.LoadedRide{
		ID: ride.ID, DepartureTime: departureTime,
		Owner:     app.UserRecord{ID: owner.ID, FirstName: "Owner"},
		Departure: app.LocationRecord{ID: departure.ID, Title: "Rice", Address: "6100 Main"},
		Arrival:   app.LocationRecord{ID: arrival.ID, Title: "IAH", Address: "Airport"},
		Riders:    []app.UserRecord{{ID: rider.ID, FirstName: "Rider"}},
		Notes:     "meet outside", Capacity: 3, Status: app.RideStatusActive,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("convertRideRows() mismatch (-want +got):\n%s", diff)
	}
}

// TestConvertRideRowsRejectsMalformedRiders checks a decode failure is reported
// rather than silently producing a ride with no riders.
func TestConvertRideRowsRejectsMalformedRiders(t *testing.T) {
	t.Parallel()

	_, err := convertRideRows(sqlc.Ride{}, sqlc.User{}, sqlc.Location{}, sqlc.Location{}, json.RawMessage("not json"))

	if err == nil {
		t.Error("convertRideRows() with malformed rider JSON = nil, want an error")
	}
}

// TestConvertRideRowsHandlesAnEmptyRiderList checks that a ride with no riders
// decodes to an empty slice rather than failing.
func TestConvertRideRowsHandlesAnEmptyRiderList(t *testing.T) {
	t.Parallel()

	got, err := convertRideRows(sqlc.Ride{}, sqlc.User{}, sqlc.Location{}, sqlc.Location{}, json.RawMessage("[]"))
	if err != nil {
		t.Fatalf("convertRideRows() error = %v", err)
	}

	if len(got.Riders) != 0 {
		t.Errorf("riders = %v, want none", got.Riders)
	}
}
