package rpc

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestMutationHandlersRejectMalformedWireValues checks date and UUID parsing across mutation adapters.
func TestMutationHandlersRejectMalformedWireValues(t *testing.T) {
	server := newTestServer(&fakeStore{})
	ctx := testContext(uuid.New(), "caller@rice.edu")
	future := timestamppb.Now()
	for _, call := range []func() error{
		func() error {
			_, err := server.CreateRide(ctx, connect.NewRequest(&v1.CreateRideRequest{DepartureTime: nil}))
			return err
		},
		func() error {
			_, err := server.UpdateRide(ctx, connect.NewRequest(&v1.UpdateRideRequest{DepartureTime: future, Id: "bad"}))
			return err
		},
		func() error {
			_, err := server.CancelRide(ctx, connect.NewRequest(&v1.CancelRideRequest{Id: "bad"}))
			return err
		},
		func() error {
			_, err := server.JoinRide(ctx, connect.NewRequest(&v1.JoinRideRequest{RideId: "bad"}))
			return err
		},
		func() error {
			_, err := server.LeaveRide(ctx, connect.NewRequest(&v1.LeaveRideRequest{RideId: "bad"}))
			return err
		},
	} {
		if err := call(); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("malformed mutation error = %v", err)
		}
	}
}

// TestUpdateRideParsesEveryWireValue checks each field UpdateRide parses is rejected on
// its own. Every other case supplies a bad ride ID, which returns before these run.
func TestUpdateRideParsesEveryWireValue(t *testing.T) {
	server := newTestServer(&fakeStore{})
	ctx := testContext(uuid.New(), "owner@rice.edu")
	valid := func() *v1.UpdateRideRequest {
		return &v1.UpdateRideRequest{
			Id: uuid.NewString(), DepartureTime: timestamppb.New(time.Now().Add(time.Hour)),
			DepartureLocationId: uuid.NewString(), ArrivalLocationId: uuid.NewString(), Capacity: 3,
		}
	}
	for _, tc := range []struct {
		name    string
		corrupt func(*v1.UpdateRideRequest)
	}{
		{"departure time", func(r *v1.UpdateRideRequest) { r.DepartureTime = &timestamppb.Timestamp{Seconds: 253402300800} }},
		{"departure location", func(r *v1.UpdateRideRequest) { r.DepartureLocationId = "not-a-uuid" }},
		{"arrival location", func(r *v1.UpdateRideRequest) { r.ArrivalLocationId = "not-a-uuid" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := valid()
			tc.corrupt(req)
			if _, err := server.UpdateRide(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("UpdateRide with a bad %s error = %v, want invalid argument", tc.name, err)
			}
		})
	}
}
