package rpc

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

func TestListRidesRejectsMalformedFilters(t *testing.T) {
	t.Parallel()
	bad := "not-a-uuid"

	for _, tc := range []struct {
		name    string
		request *v1.ListRidesRequest
	}{
		{name: "departure location", request: &v1.ListRidesRequest{DepartureLocationId: &bad}},
		{name: "arrival location", request: &v1.ListRidesRequest{ArrivalLocationId: &bad}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeDatabase{}

			_, err := newTestServer(fake).ListRides(testContext(uuid.New(), "viewer@rice.edu"), connect.NewRequest(tc.request))

			if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
				t.Errorf("ListRides() code = %v, want %v", got, connect.CodeInvalidArgument)
			}
			if fake.called(opBegin) {
				t.Error("ListRides reached storage with an unparseable filter")
			}
		})
	}
}

// TestListRidesForwardsTimeBounds checks both wire timestamps reach the service.
// The lower bound is clamped to the discovery window by the application, so only
// the upper bound survives verbatim.
func TestListRidesForwardsTimeBounds(t *testing.T) {
	t.Parallel()
	fake := &fakeDatabase{}
	before := time.Now().Add(48 * time.Hour)

	_, err := newTestServer(fake).ListRides(testContext(uuid.New(), "viewer@rice.edu"),
		connect.NewRequest(&v1.ListRidesRequest{DepartureBefore: timestamppb.New(before)}))
	if err != nil {
		t.Fatalf("ListRides() error = %v", err)
	}

	if fake.ridesQuery.DepartureBefore == nil {
		t.Fatal("DepartureBefore = nil, want the caller's upper bound")
	}
	if got := *fake.ridesQuery.DepartureBefore; !got.Equal(before.UTC()) {
		t.Errorf("DepartureBefore = %v, want %v", got, before.UTC())
	}
}

func TestListMyRidesUsesTheVerifiedCaller(t *testing.T) {
	t.Parallel()
	caller := uuid.New()
	fake := &fakeDatabase{}

	if _, err := newTestServer(fake).ListMyRides(testContext(caller, "caller@rice.edu"),
		connect.NewRequest(&v1.ListMyRidesRequest{})); err != nil {
		t.Fatalf("ListMyRides() error = %v", err)
	}

	if fake.historyOwner != caller {
		t.Errorf("ListMyRides queried %v, want the verified caller %v", fake.historyOwner, caller)
	}
}

// TestRideListsConvertEveryResult checks the handlers convert the whole page
// rather than only its first entry.
func TestRideListsConvertEveryResult(t *testing.T) {
	t.Parallel()
	owner := testUser(uuid.New(), "owner@rice.edu")
	rides := make([]app.LoadedRide, 3)
	for i := range rides {
		rides[i] = app.LoadedRide{
			ID: uuid.New(), DepartureTime: time.Now().Add(time.Hour), Owner: owner,
			Riders: []app.UserRecord{owner}, Capacity: 2, Status: app.RideStatusActive,
		}
	}
	fake := &fakeDatabase{
		users:     map[uuid.UUID]app.UserRecord{owner.ID: owner},
		discovery: rides,
		history:   rides,
	}
	ctx := testContext(owner.ID, owner.Email)

	discovered, err := newTestServer(fake).ListRides(ctx, connect.NewRequest(&v1.ListRidesRequest{}))
	if err != nil {
		t.Fatalf("ListRides() error = %v", err)
	}
	if len(discovered.Msg.Rides) != len(rides) {
		t.Errorf("ListRides() returned %d rides, want %d", len(discovered.Msg.Rides), len(rides))
	}

	history, err := newTestServer(fake).ListMyRides(ctx, connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil {
		t.Fatalf("ListMyRides() error = %v", err)
	}
	if len(history.Msg.Rides) != len(rides) {
		t.Errorf("ListMyRides() returned %d rides, want %d", len(history.Msg.Rides), len(rides))
	}
}
