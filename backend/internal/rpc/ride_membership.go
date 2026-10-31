package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// JoinRide adds the verified caller to a ride.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.JoinRideRequest]): a ride ID string in UUID format.
//
// It returns the joined ride with caller-visible contacts or a Connect error.
func (s *Server) JoinRide(ctx context.Context, req *connect.Request[carpoolv1.JoinRideRequest]) (*connect.Response[carpoolv1.JoinRideResponse], error) {
	// TODO(student): Implement JoinRide RPC adapter.
	return nil, convertToConnectError(app.ErrNotImplemented)
}

// LeaveRide removes the verified caller from a ride.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.LeaveRideRequest]): a ride ID string in UUID format.
//
// It returns the ride with caller-visible contacts or a Connect error.
func (s *Server) LeaveRide(ctx context.Context, req *connect.Request[carpoolv1.LeaveRideRequest]) (*connect.Response[carpoolv1.LeaveRideResponse], error) {
	rideID, err := uuid.Parse(req.Msg.RideId)
	if err != nil {
		return nil, convertToConnectError(err)
	}

	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	ride, err := s.service.LeaveRide(ctx, currentUser, rideID)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	response := &carpoolv1.LeaveRideResponse{Ride: convertRideToProto(ride)}
	return connect.NewResponse(response), nil
}
