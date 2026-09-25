package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// JoinRide adds the verified caller to a ride.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.JoinRideRequest]): a ride ID string in UUID format.
//
// It returns the joined ride with caller-visible contacts or a Connect error.
func (s *Server) JoinRide(ctx context.Context, req *connect.Request[v1.JoinRideRequest]) (*connect.Response[v1.JoinRideResponse], error) {
	// TODO(student): Implement JoinRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}

// LeaveRide removes the verified caller from a ride.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.LeaveRideRequest]): a ride ID string in UUID format.
//
// It returns the ride with caller-visible contacts or a Connect error.
func (s *Server) LeaveRide(ctx context.Context, req *connect.Request[v1.LeaveRideRequest]) (*connect.Response[v1.LeaveRideResponse], error) {
	// TODO(student): Implement LeaveRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
