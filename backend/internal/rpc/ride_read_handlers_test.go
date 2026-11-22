package rpc

import (
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"testing"
	"time"
)

// TestRideReadHandlersParseWireValues checks invalid ride and location UUIDs become InvalidArgument.
func TestRideReadHandlersParseWireValues(t *testing.T) {
	server := newTestServer(&fakeStore{})
	ctx := testContext(uuid.New(), "viewer@rice.edu")
	bad := "not-a-uuid"
	if _, err := server.GetRide(ctx, connect.NewRequest(&v1.GetRideRequest{Id: bad})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetRide malformed ID error = %v", err)
	}
	if _, err := server.ListRides(ctx, connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &bad})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ListRides malformed filter error = %v", err)
	}
}

// TestListMyRidesReturnsCompleteHistory checks that adapter conversion keeps all fixture rides.
func TestListMyRidesReturnsCompleteHistory(t *testing.T) {
	viewer := uuid.New()
	owner := testUser(viewer, "viewer@rice.edu")
	rides := listedRides(51, owner)
	server := newTestServer(&fakeStore{listed: rides})

	response, err := server.ListMyRides(testContext(viewer, owner.Email), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Rides) != len(rides) {
		t.Fatalf("ListMyRides returned %d rides, want all %d", len(response.Msg.Rides), len(rides))
	}
}

// TestListRidesReturnsEveryMatch checks that discovery conversion keeps all fixture matches.
func TestListRidesReturnsEveryMatch(t *testing.T) {
	viewer := uuid.New()
	owner := testUser(viewer, "viewer@rice.edu")
	rides := listedRides(51, owner)
	server := newTestServer(&fakeStore{listed: rides})

	response, err := server.ListRides(testContext(viewer, owner.Email), connect.NewRequest(&v1.ListRidesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Rides) != len(rides) {
		t.Fatalf("ListRides returned %d rides, want all %d", len(response.Msg.Rides), len(rides))
	}
}

// listedRides builds count distinct active ride records for owner to exercise list conversion.
func listedRides(count int, owner app.UserRecord) []app.LoadedRide {
	rides := make([]app.LoadedRide, count)
	for i := range rides {
		rides[i] = app.LoadedRide{
			ID:            uuid.New(),
			DepartureTime: time.Now().Add(time.Duration(i) * time.Hour),
			Capacity:      2,
			Status:        app.RideStatusActive,
			Owner:         owner,
			Departure:     app.LocationRecord{ID: uuid.New(), Title: "Rice"},
			Arrival:       app.LocationRecord{ID: uuid.New(), Title: "IAH"},
		}
	}
	return rides
}
