package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// GetRide reads a ride for the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.GetRideRequest]): a ride ID string in UUID format.
//
// It returns the ride with caller-visible contacts or a Connect error on failure.
func (s *Server) GetRide(ctx context.Context, req *connect.Request[v1.GetRideRequest]) (*connect.Response[v1.GetRideResponse], error) {
	// TODO(student): Implement GetRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
