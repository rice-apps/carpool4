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
	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	departureID, err := uuid.Parse(req.Msg.DepartureLocationId)
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("%w: departure location ID", app.ErrInvalidArgument))
	}
	arrivalID, err := uuid.Parse(req.Msg.ArrivalLocationId)
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("%w: arrival location ID", app.ErrInvalidArgument))
	}
	if !req.Msg.DepartureTime.IsValid() {
		return nil, convertToConnectError(fmt.Errorf("%w: departure time", app.ErrInvalidArgument))
	}
	params := app.RideWriteParams{
		DepartureTime:       req.Msg.DepartureTime.AsTime(),
		DepartureLocationID: departureID,
		ArrivalLocationID:   arrivalID,
		Notes:               req.Msg.Notes,
		Capacity:            req.Msg.Capacity,
	}
	ride, err := s.service.CreateRide(ctx, currentUser, params)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	response := &carpoolv1.CreateRideResponse{Ride: convertRideToProto(ride)}
	return connect.NewResponse(response), nil
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
		return nil, convertToConnectError(fmt.Errorf("invalid argument: '%q' could not be parsed: %w", req.Msg.Id, app.ErrInvalidArgument))
	}

	err = req.Msg.DepartureTime.CheckValid() // validates the DepartureTime as a timestamp
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("invalid argument: departure time must be a valid timestamp: %w", app.ErrInvalidArgument))
	}
	departureTime := req.Msg.DepartureTime.AsTime() // gets the departureTime as a timestamp object.

	departureLocationID, err := uuid.Parse(req.Msg.DepartureLocationId) // gets departure location as uuid
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("invalid argument: '%q' could not be parsed: %w", req.Msg.DepartureLocationId, app.ErrInvalidArgument))
	}

	arrivalLocationID, err := uuid.Parse(req.Msg.ArrivalLocationId) // gets arrival location as uuid
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("invalid argument: '%q' could not be parsed: %w", req.Msg.ArrivalLocationId, app.ErrInvalidArgument))
	}

	notes := req.Msg.Notes       // gets notes as a string
	capacity := req.Msg.Capacity // gets capacity as an int32

	inputChanges := app.RideWriteParams{ // uses previous fields as a RideWriteParams struct object
		DepartureTime:       departureTime,
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
	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		return nil, convertToConnectError(err)
	}

	rideID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, convertToConnectError(fmt.Errorf("%w: ride ID", app.ErrInvalidArgument))
	}
	ride, err := s.service.CancelRide(ctx, currentUser, rideID)
	if err != nil {
		return nil, convertToConnectError(err)
	}

	response := &carpoolv1.CancelRideResponse{Ride: convertRideToProto(ride)}
	return connect.NewResponse(response), nil
}
