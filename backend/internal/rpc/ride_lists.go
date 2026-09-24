package rpc

import (
	"connectrpc.com/connect"
	"context"
	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// ListRides finds rides visible to the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.ListRidesRequest]): optional departure and arrival
//     location ID strings in UUID format, plus departure time bounds.
//
// It returns matching rides with caller-visible contacts or a Connect error.
func (s *Server) ListRides(ctx context.Context, req *connect.Request[v1.ListRidesRequest]) (*connect.Response[v1.ListRidesResponse], error) {
	// TODO(student): Implement ListRides RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}

// ListMyRides reads the verified caller's ride history.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.ListMyRidesRequest]): an empty request.
//
// It returns the caller's rides with visible contacts or a Connect error.
func (s *Server) ListMyRides(ctx context.Context, req *connect.Request[v1.ListMyRidesRequest]) (*connect.Response[v1.ListMyRidesResponse], error) {
	// TODO(student): Implement ListMyRides RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
