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

// validCreateRequest returns a request whose every wire value parses, so a test
// can corrupt exactly one field.
func validCreateRequest() *v1.CreateRideRequest {
	return &v1.CreateRideRequest{
		DepartureTime:       timestamppb.New(time.Now().Add(time.Hour)),
		DepartureLocationId: uuid.NewString(),
		ArrivalLocationId:   uuid.NewString(),
		Capacity:            3,
	}
}

// validUpdateRequest is the UpdateRide equivalent of validCreateRequest.
func validUpdateRequest() *v1.UpdateRideRequest {
	return &v1.UpdateRideRequest{
		Id:                  uuid.NewString(),
		DepartureTime:       timestamppb.New(time.Now().Add(time.Hour)),
		DepartureLocationId: uuid.NewString(),
		ArrivalLocationId:   uuid.NewString(),
		Capacity:            3,
	}
}

// TestCreateRideParsesEveryWireValue checks each field CreateRide reads is
// rejected on its own. A timestamp can be unusable while non-nil, so the
// out-of-range case proves the check is more than a nil guard.
func TestCreateRideParsesEveryWireValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		corrupt func(*v1.CreateRideRequest)
	}{
		{name: "missing departure time", corrupt: func(r *v1.CreateRideRequest) { r.DepartureTime = nil }},
		{name: "departure time out of range", corrupt: func(r *v1.CreateRideRequest) {
			r.DepartureTime = &timestamppb.Timestamp{Seconds: 253402300800}
		}},
		{name: "departure location", corrupt: func(r *v1.CreateRideRequest) { r.DepartureLocationId = "not-a-uuid" }},
		{name: "arrival location", corrupt: func(r *v1.CreateRideRequest) { r.ArrivalLocationId = "not-a-uuid" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := validCreateRequest()
			tc.corrupt(request)
			fake := &fakeDatabase{}

			_, err := newTestServer(fake).CreateRide(testContext(uuid.New(), "owner@rice.edu"), connect.NewRequest(request))

			if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
				t.Errorf("CreateRide() with a bad %s: code = %v, want %v", tc.name, got, connect.CodeInvalidArgument)
			}
			if fake.called(opBegin) {
				t.Errorf("CreateRide reached storage with a bad %s", tc.name)
			}
		})
	}
}

// TestUpdateRideParsesEveryWireValue checks UpdateRide's own parse branches.
// Its ride ID is read first, so a test that corrupts only the ID never reaches
// the rest.
func TestUpdateRideParsesEveryWireValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		corrupt func(*v1.UpdateRideRequest)
	}{
		{name: "ride ID", corrupt: func(r *v1.UpdateRideRequest) { r.Id = "not-a-uuid" }},
		{name: "departure time", corrupt: func(r *v1.UpdateRideRequest) {
			r.DepartureTime = &timestamppb.Timestamp{Seconds: 253402300800}
		}},
		{name: "departure location", corrupt: func(r *v1.UpdateRideRequest) { r.DepartureLocationId = "not-a-uuid" }},
		{name: "arrival location", corrupt: func(r *v1.UpdateRideRequest) { r.ArrivalLocationId = "not-a-uuid" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := validUpdateRequest()
			tc.corrupt(request)
			fake := &fakeDatabase{}

			_, err := newTestServer(fake).UpdateRide(testContext(uuid.New(), "owner@rice.edu"), connect.NewRequest(request))

			if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
				t.Errorf("UpdateRide() with a bad %s: code = %v, want %v", tc.name, got, connect.CodeInvalidArgument)
			}
		})
	}
}

func TestCancelRideRejectsMalformedRideID(t *testing.T) {
	t.Parallel()
	fake := &fakeDatabase{}

	_, err := newTestServer(fake).CancelRide(testContext(uuid.New(), "owner@rice.edu"),
		connect.NewRequest(&v1.CancelRideRequest{Id: "not-a-uuid"}))

	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("CancelRide() code = %v, want %v", got, connect.CodeInvalidArgument)
	}
	if fake.called(opBegin) {
		t.Error("CancelRide reached storage with an unparseable ride ID")
	}
}

func TestCreateRideReturnsTheStoredRide(t *testing.T) {
	t.Parallel()
	owner := testUser(uuid.New(), "owner@rice.edu")
	fake := &fakeDatabase{
		users: map[uuid.UUID]app.UserRecord{owner.ID: owner},
		rides: map[uuid.UUID]app.LoadedRide{},
	}

	got, err := newTestServer(fake).CreateRide(testContext(owner.ID, owner.Email),
		connect.NewRequest(validCreateRequest()))
	if err != nil {
		t.Fatalf("CreateRide() error = %v", err)
	}

	if got.Msg.Ride.Owner.GetId() != owner.ID.String() {
		t.Errorf("owner = %q, want the caller %q", got.Msg.Ride.Owner.GetId(), owner.ID)
	}
	if !fake.called(opCommit) {
		t.Error("CreateRide did not commit")
	}
}
