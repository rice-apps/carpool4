package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/auth"

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
	currentUser := app.CurrentUser{}
	if auth.IsSignedIn(ctx) {
		var err error
		currentUser, err = getCurrentUser(ctx)
		if err != nil {
			return nil, convertToConnectError(err)
		}
	}
	rideId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	ride, err := s.service.GetRide(ctx, currentUser, rideId)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	response := &carpoolv1.GetRideResponse{Ride: convertRideToProto(ride)}
	return connect.NewResponse(response), nil
}
