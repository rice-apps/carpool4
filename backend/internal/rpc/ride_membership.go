package rpc

import (
	"context"
	"fmt"

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
	// gets the current use ctx and stores it in currentUser variable
	currentUser, err := getCurrentUser(ctx)
	// if there is an error with currentUser then return nil and the internal error
	if err != nil {
		return nil, convertToConnectError(err)
	}
	// gets the ride ID in UUID format and stores it in parsedRideID variable
	parsedRideID, err := uuid.Parse(req.Msg.RideId)
	// if there is an error with parsedRideID then return nil and the invalid argument error
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("%w: ride ID", app.ErrInvalidArgument))
	}
	// calls the app layer JoinRide and return the updated ride with contacts currentUser may see
	ride, err := s.service.JoinRide(ctx, currentUser, parsedRideID)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	// copies an app.Ride result into carpoolv1.Ride for the API response and converts the people, locations, time, and status inside the ride
	response := &carpoolv1.JoinRideResponse{Ride: convertRideToProto(ride)}
	// It returns the joined ride with caller-visible contacts if there is no error
	return connect.NewResponse(response), nil
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
	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		return nil, convertToConnectError(err)
	}

	rideID, err := uuid.Parse(req.Msg.RideId)
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("%w: ride ID", app.ErrInvalidArgument))
	}

	ride, err := s.service.LeaveRide(ctx, currentUser, rideID)
	if err != nil {

		return nil, convertToConnectError(err)
	}

	response := &carpoolv1.LeaveRideResponse{Ride: convertRideToProto(ride)}
	return connect.NewResponse(response), nil
}
