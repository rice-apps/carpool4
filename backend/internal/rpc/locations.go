package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// ListLocations lists public locations.
//
// Inputs:
//   - ctx (context.Context): passes timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.ListLocationsRequest]): an empty request.
//
// It returns protobuf locations on success or a Connect error on failure.
func (s *Server) ListLocations(ctx context.Context, req *connect.Request[carpoolv1.ListLocationsRequest]) (*connect.Response[carpoolv1.ListLocationsResponse], error) {
	// TODO(student): Implement ListLocations RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
