package rpc

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

func TestGetRideRejectsMalformedRideID(t *testing.T) {
	t.Parallel()
	fake := &fakeDatabase{}

	_, err := newTestServer(fake).GetRide(testContext(uuid.New(), "viewer@rice.edu"),
		connect.NewRequest(&carpoolv1.GetRideRequest{Id: "not-a-uuid"}))

	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("GetRide(\"not-a-uuid\") code = %v, want %v", got, connect.CodeInvalidArgument)
	}
	if fake.called(opBegin) {
		t.Error("GetRide reached storage with an unparseable ride ID")
	}
}

func TestGetRideReturnsTheRequestedRide(t *testing.T) {
	t.Parallel()
	owner := testUser(uuid.New(), "owner@rice.edu")
	ride := app.LoadedRide{
		ID: uuid.New(), DepartureTime: time.Now().Add(time.Hour), Owner: owner,
		Riders: []app.UserRecord{owner}, Capacity: 2, Status: app.RideStatusActive,
	}
	fake := &fakeDatabase{
		users: map[uuid.UUID]app.UserRecord{owner.ID: owner},
		rides: map[uuid.UUID]app.LoadedRide{ride.ID: ride},
	}

	got, err := newTestServer(fake).GetRide(testContext(owner.ID, owner.Email),
		connect.NewRequest(&carpoolv1.GetRideRequest{Id: ride.ID.String()}))
	if err != nil {
		t.Fatalf("GetRide(%v) error = %v", ride.ID, err)
	}

	if got.Msg.Ride.Id != ride.ID.String() {
		t.Errorf("ride ID = %q, want %q", got.Msg.Ride.Id, ride.ID)
	}
	if got.Msg.Ride.Owner.GetId() != owner.ID.String() {
		t.Errorf("owner ID = %q, want %q", got.Msg.Ride.Owner.GetId(), owner.ID)
	}
}

func TestGetRideMapsMissingRideToNotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeDatabase{}

	_, err := newTestServer(fake).GetRide(testContext(uuid.New(), "viewer@rice.edu"),
		connect.NewRequest(&carpoolv1.GetRideRequest{Id: uuid.NewString()}))

	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Errorf("GetRide() code = %v, want %v", got, connect.CodeNotFound)
	}
}
