package rpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

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
	return nil, convertToConnectError(app.ErrNotImplemented)
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
	currentUser, err := getCurrentUser(ctx) // gets current user
	if err != nil {                         // checks error for currentUser
		return nil, convertToConnectError(err)
	}

	rideID, err := uuid.Parse(req.Msg.Id) // gets UUID in UUID object
	if err != nil {                       // checks error for UUID
		return nil, convertToConnectError(fmt.Errorf("invalid argument: '%q' could not be parsed: %q", req.Msg.Id, app.ErrInvalidArgument))
	}

	err = req.Msg.DepartureDate.CheckValid() // validates the DepartureDate as a timestamp
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("invalid argument: departure date must be a valid timestamp: %w", app.ErrInvalidArgument))
	}
	departureDate := req.Msg.DepartureDate.AsTime() // gets the departureDat as a timestamp object.

	departureLocationID, err := uuid.Parse(req.Msg.DepartureLocationId) // gets departure location as uuid
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("invalid argument: '%q' could not be parsed: %q", req.Msg.DepartureLocationId, app.ErrInvalidArgument))
	}

	arrivalLocationID, err := uuid.Parse(req.Msg.ArrivalLocationId) // gets arrival location as uuid
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("invalid argument: '%q' could not be parsed: %q", req.Msg.ArrivalLocationId, app.ErrInvalidArgument))
	}

	notes := req.Msg.Notes       // gets notes as a string
	capacity := req.Msg.Capacity // gets capacity as an int32

	inputChanges := app.RideWriteParams{ // uses previous fields as a RideWriteParams struct object
		DepartureDate:       departureDate,
		DepartureLocationID: departureLocationID,
		ArrivalLocationID:   arrivalLocationID,
		Notes:               notes,
		Capacity:            capacity,
	}

	ride, err := s.service.UpdateRide(ctx, currentUser, rideID, inputChanges) // convert ride response helper to proto
	if err != nil {
		return nil, convertToConnectError(err)
	}
	response := &carpoolv1.UpdateRideResponse{Ride: convertRideToProto(ride)} // response as an UpdateRideResponse by calling convertRideToProto of our ride based on app UpdateRide function
	return connect.NewResponse(response), nil                                 // returns the correct type (NewResponse) as an UpdateRideResponse; ready for app layer
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
	return nil, convertToConnectError(app.ErrNotImplemented)
}
