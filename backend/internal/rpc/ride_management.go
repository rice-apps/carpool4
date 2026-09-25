package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// CreateRide creates a ride owned by the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.CreateRideRequest]): departure timestamp, location
//     ID strings in UUID format, notes, and capacity for the new ride.
//
// It returns the new ride with caller-visible contacts or a Connect error.
func (s *Server) CreateRide(ctx context.Context, req *connect.Request[carpoolv1.CreateRideRequest]) (*connect.Response[carpoolv1.CreateRideResponse], error) {
	// TODO(student): Implement CreateRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}

// UpdateRide changes a ride for the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.UpdateRideRequest]): ride ID and location ID strings
//     in UUID format, plus replacement departure timestamp, notes, and capacity.
//
// It returns the updated ride with caller-visible contacts or a Connect error.
func (s *Server) UpdateRide(ctx context.Context, req *connect.Request[carpoolv1.UpdateRideRequest]) (*connect.Response[carpoolv1.UpdateRideResponse], error) {
	// TODO(student): Implement UpdateRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}

// CancelRide cancels a ride for the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.CancelRideRequest]): a ride ID string in UUID format.
//
// It returns the cancelled ride with caller-visible contacts or a Connect error.
func (s *Server) CancelRide(ctx context.Context, req *connect.Request[carpoolv1.CancelRideRequest]) (*connect.Response[carpoolv1.CancelRideResponse], error) {
	// TODO(student): Implement CancelRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
