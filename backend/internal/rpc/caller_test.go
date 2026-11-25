package rpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/auth"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// rpcCall names one handler and invokes it with a context the test controls.
type rpcCall struct {
	name string
	call func(*Server, context.Context) error
}

// protectedCalls are the handlers that refuse an unverified caller outright.
var protectedCalls = []rpcCall{
	{"GetUser", func(s *Server, ctx context.Context) error {
		_, err := s.GetUser(ctx, connect.NewRequest(&v1.GetUserRequest{}))
		return err
	}},
	{"UpdateUser", func(s *Server, ctx context.Context) error {
		_, err := s.UpdateUser(ctx, connect.NewRequest(&v1.UpdateUserRequest{}))
		return err
	}},
	{"ListMyRides", func(s *Server, ctx context.Context) error {
		_, err := s.ListMyRides(ctx, connect.NewRequest(&v1.ListMyRidesRequest{}))
		return err
	}},
	{"CreateRide", func(s *Server, ctx context.Context) error {
		_, err := s.CreateRide(ctx, connect.NewRequest(&v1.CreateRideRequest{}))
		return err
	}},
	{"UpdateRide", func(s *Server, ctx context.Context) error {
		_, err := s.UpdateRide(ctx, connect.NewRequest(&v1.UpdateRideRequest{}))
		return err
	}},
	{"CancelRide", func(s *Server, ctx context.Context) error {
		_, err := s.CancelRide(ctx, connect.NewRequest(&v1.CancelRideRequest{}))
		return err
	}},
	{"JoinRide", func(s *Server, ctx context.Context) error {
		_, err := s.JoinRide(ctx, connect.NewRequest(&v1.JoinRideRequest{}))
		return err
	}},
	{"LeaveRide", func(s *Server, ctx context.Context) error {
		_, err := s.LeaveRide(ctx, connect.NewRequest(&v1.LeaveRideRequest{}))
		return err
	}},
}

// publicReadCalls are the handlers a signed-out visitor may reach.
var publicReadCalls = []rpcCall{
	{"GetRide", func(s *Server, ctx context.Context) error {
		_, err := s.GetRide(ctx, connect.NewRequest(&v1.GetRideRequest{Id: uuid.NewString()}))
		return err
	}},
	{"ListRides", func(s *Server, ctx context.Context) error {
		_, err := s.ListRides(ctx, connect.NewRequest(&v1.ListRidesRequest{}))
		return err
	}},
}

// TestProtectedRPCsRequireVerifiedIdentity checks that authentication happens
// before request parsing or any application call. The server holds a nil
// service, so reaching one would panic rather than quietly pass.
func TestProtectedRPCsRequireVerifiedIdentity(t *testing.T) {
	t.Parallel()
	identities := []struct {
		name string
		ctx  context.Context
	}{
		{"no identity", context.Background()},
		{"identity with an empty user ID", auth.WithUser(context.Background(), auth.User{Email: "caller@rice.edu"})},
	}

	for _, op := range protectedCalls {
		t.Run(op.name, func(t *testing.T) {
			t.Parallel()
			for _, identity := range identities {
				t.Run(identity.name, func(t *testing.T) {
					t.Parallel()
					err := op.call(NewServer(nil), identity.ctx)
					if got := connect.CodeOf(err); got != connect.CodeUnauthenticated {
						t.Errorf("%s() code = %v, want %v", op.name, got, connect.CodeUnauthenticated)
					}
				})
			}
		})
	}
}

// TestPublicReadRPCsRejectInvalidIdentity checks that a supplied but unusable
// identity is refused rather than quietly downgraded to a guest, which would
// give a caller with a malformed token more than they are entitled to.
func TestPublicReadRPCsRejectInvalidIdentity(t *testing.T) {
	t.Parallel()
	ctx := auth.WithUser(context.Background(), auth.User{Email: "caller@rice.edu"})

	for _, op := range publicReadCalls {
		t.Run(op.name, func(t *testing.T) {
			t.Parallel()
			err := op.call(NewServer(nil), ctx)
			if got := connect.CodeOf(err); got != connect.CodeUnauthenticated {
				t.Errorf("%s() code = %v, want %v", op.name, got, connect.CodeUnauthenticated)
			}
		})
	}
}
