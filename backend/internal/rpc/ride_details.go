package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// GetRide reads a ride for the verified caller or a guest.
//
// Inputs:
//   - ctx (context.Context): may supply the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.GetRideRequest]): a ride ID string in UUID format.
//
// It returns the viewer-authorized ride or a Connect error on failure. Guest
// results omit people and notes.
func (s *Server) GetRide(ctx context.Context, req *connect.Request[carpoolv1.GetRideRequest]) (*connect.Response[carpoolv1.GetRideResponse], error) {
	// TODO(student): Implement GetRide RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
