package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// ListRides finds rides visible to the verified caller or a guest.
//
// Inputs:
//   - ctx (context.Context): may supply the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.ListRidesRequest]): optional departure and arrival
//     location ID strings in UUID format, plus departure time bounds.
//
// It returns matching viewer-authorized rides or a Connect error. Guest
// results omit people and notes.
func (s *Server) ListRides(ctx context.Context, req *connect.Request[carpoolv1.ListRidesRequest]) (*connect.Response[carpoolv1.ListRidesResponse], error) {
	// TODO(student): Implement ListRides RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}

// ListMyRides reads the verified caller's ride history.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.ListMyRidesRequest]): an empty request.
//
// It returns the caller's rides with visible contacts or a Connect error.
func (s *Server) ListMyRides(ctx context.Context, req *connect.Request[carpoolv1.ListMyRidesRequest]) (*connect.Response[carpoolv1.ListMyRidesResponse], error) {
	// TODO(student): Implement ListMyRides RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
