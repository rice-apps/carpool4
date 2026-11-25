package rpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// TestProtectedRPCsRequireVerifiedIdentity checks authentication before request
// parsing or application calls. No application service is available in this test.
func TestProtectedRPCsRequireVerifiedIdentity(t *testing.T) {
	server := NewServer(nil)
	operations := map[string]func(context.Context) error{
		"GetUser": func(ctx context.Context) error {
			_, err := server.GetUser(ctx, connect.NewRequest(&v1.GetUserRequest{}))
			return err
		},
		"UpdateUser": func(ctx context.Context) error {
			_, err := server.UpdateUser(ctx, connect.NewRequest(&v1.UpdateUserRequest{}))
			return err
		},
		"ListMyRides": func(ctx context.Context) error {
			_, err := server.ListMyRides(ctx, connect.NewRequest(&v1.ListMyRidesRequest{}))
			return err
		},
		"CreateRide": func(ctx context.Context) error {
			_, err := server.CreateRide(ctx, connect.NewRequest(&v1.CreateRideRequest{}))
			return err
		},
		"UpdateRide": func(ctx context.Context) error {
			_, err := server.UpdateRide(ctx, connect.NewRequest(&v1.UpdateRideRequest{}))
			return err
		},
		"CancelRide": func(ctx context.Context) error {
			_, err := server.CancelRide(ctx, connect.NewRequest(&v1.CancelRideRequest{}))
			return err
		},
		"JoinRide": func(ctx context.Context) error {
			_, err := server.JoinRide(ctx, connect.NewRequest(&v1.JoinRideRequest{}))
			return err
		},
		"LeaveRide": func(ctx context.Context) error {
			_, err := server.LeaveRide(ctx, connect.NewRequest(&v1.LeaveRideRequest{}))
			return err
		},
	}
	for operation, call := range operations {
		t.Run(operation, func(t *testing.T) {
			for name, ctx := range map[string]context.Context{
				"missing": context.Background(),
				"nil ID":  auth.WithUser(context.Background(), auth.User{Email: "caller@rice.edu"}),
			} {
				t.Run(name, func(t *testing.T) {
					if err := call(ctx); connect.CodeOf(err) != connect.CodeUnauthenticated {
						t.Fatalf("%s() error = %v, want Unauthenticated", operation, err)
					}
				})
			}
		})
	}
}

// TestPublicReadRPCsRejectInvalidIdentity checks that a supplied empty user ID
// is rejected rather than treated as a guest.
func TestPublicReadRPCsRejectInvalidIdentity(t *testing.T) {
	server := NewServer(nil)
	ctx := auth.WithUser(context.Background(), auth.User{Email: "caller@rice.edu"})
	operations := map[string]func(context.Context) error{
		"GetRide": func(ctx context.Context) error {
			_, err := server.GetRide(ctx, connect.NewRequest(&v1.GetRideRequest{Id: uuid.NewString()}))
			return err
		},
		"ListRides": func(ctx context.Context) error {
			_, err := server.ListRides(ctx, connect.NewRequest(&v1.ListRidesRequest{}))
			return err
		},
	}
	for operation, call := range operations {
		t.Run(operation, func(t *testing.T) {
			if err := call(ctx); connect.CodeOf(err) != connect.CodeUnauthenticated {
				t.Fatalf("%s() error = %v, want Unauthenticated", operation, err)
			}
		})
	}
}
