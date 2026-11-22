package postgres

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
	"testing"
	"time"
)

// TestConvertRideRowsDecodesRiders checks related row and rider conversion, and
// that malformed rider JSON reports an error.
func TestConvertRideRowsDecodesRiders(t *testing.T) {
	rider := sqlc.User{ID: uuid.New(), FirstName: "Rider"}
	riders, err := json.Marshal([]sqlc.User{rider})
	if err != nil {
		t.Fatal(err)
	}
	owner := sqlc.User{ID: uuid.New(), FirstName: "Owner"}
	departure := sqlc.Location{ID: uuid.New(), Title: "Rice", Address: "6100 Main"}
	arrival := sqlc.Location{ID: uuid.New(), Title: "IAH", Address: "Airport"}
	ride := sqlc.Ride{
		ID: uuid.New(), OwnerID: owner.ID, DepartureTime: time.Now(),
		DepartureLocationID: departure.ID, ArrivalLocationID: arrival.ID,
		Notes: "meet outside", Capacity: 3, Status: sqlc.RideStatusActive,
	}
	got, err := convertRideRows(ride, owner, departure, arrival, riders)
	if err != nil {
		t.Fatalf("convertRideRows() error = %v", err)
	}
	if got.ID != ride.ID || !got.DepartureTime.Equal(ride.DepartureTime) || got.Notes != ride.Notes || got.Capacity != ride.Capacity || got.Status != app.RideStatusActive {
		t.Fatalf("convertRideRows fields = %#v, want fields from %#v", got, ride)
	}
	if got.Owner.ID != owner.ID || got.Owner.FirstName != owner.FirstName ||
		got.Departure != (app.LocationRecord{ID: departure.ID, Title: departure.Title, Address: departure.Address}) ||
		got.Arrival != (app.LocationRecord{ID: arrival.ID, Title: arrival.Title, Address: arrival.Address}) {
		t.Fatalf("convertRideRows related records = %#v, want owner and both locations", got)
	}
	if len(got.Riders) != 1 || got.Riders[0].ID != rider.ID {
		t.Fatalf("convertRideRows riders = %#v, want decoded rider", got.Riders)
	}

	if _, err := convertRideRows(sqlc.Ride{}, sqlc.User{}, sqlc.Location{}, sqlc.Location{}, json.RawMessage("not json")); err == nil {
		t.Fatal("convertRideRows malformed riders JSON error = nil, want error")
	}
}
