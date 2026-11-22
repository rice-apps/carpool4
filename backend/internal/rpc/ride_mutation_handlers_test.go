package rpc

import (
	"connectrpc.com/connect"
	"github.com/google/uuid"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
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
