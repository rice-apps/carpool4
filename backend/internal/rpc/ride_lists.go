package rpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
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
	//create currentUser object and initialize with user values (or empty if guest)
	currentUser := app.CurrentUser{}
	if auth.IsSignedIn(ctx) {
		var err error
		currentUser, err = getCurrentUser(ctx)
		if err != nil {
			return nil, convertToConnectError(err)
		}
	}
	//create new ListRidesInput object, check if ListRidesRequest fields are nil first, if not, set that field in the new object
	requestInput := app.ListRidesInput{}

	if req.Msg.GetDepartureAfter() != nil {
		if !req.Msg.DepartureAfter.IsValid() {
			return nil, convertToConnectError(fmt.Errorf("%w: departure after", app.ErrInvalidArgument))
		}
		departureAfterTime := req.Msg.DepartureAfter.AsTime()
		requestInput.DepartureAfter = &departureAfterTime
	}

	if req.Msg.GetDepartureBefore() != nil {
		if !req.Msg.DepartureBefore.IsValid() {
			return nil, convertToConnectError(fmt.Errorf("%w: departure before", app.ErrInvalidArgument))
		}
		departureBeforeTime := req.Msg.DepartureBefore.AsTime()
		requestInput.DepartureBefore = &departureBeforeTime
	}

	if req.Msg.ArrivalLocationId != nil {
		parsedArrivalID, err := uuid.Parse(*req.Msg.ArrivalLocationId)
		if err != nil {
			return nil, convertToConnectError(fmt.Errorf("%w: arrival location ID", app.ErrInvalidArgument))
		}
		requestInput.ArrivalLocationID = &parsedArrivalID
	}

	if req.Msg.DepartureLocationId != nil {
		parsedDepartureID, err := uuid.Parse(*req.Msg.DepartureLocationId)
		if err != nil {
			return nil, convertToConnectError(fmt.Errorf("%w: departure location ID", app.ErrInvalidArgument))
		}
		requestInput.DepartureLocationID = &parsedDepartureID
	}

	//call app layer ListRides method with provided arguments
	rideList, err := s.service.ListRides(ctx, currentUser, requestInput)
	if err != nil {
		return nil, convertToConnectError(err)
	}
	ridePointerList := make([]*carpoolv1.Ride, len(rideList))
	for i, ride := range rideList {
		ridePointerList[i] = convertRideToProto(ride)
	}
	return connect.NewResponse(&carpoolv1.ListRidesResponse{Rides: ridePointerList}), nil
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
	// get user
	user, err := getCurrentUser(ctx)
	if err != nil {
		return nil, convertToConnectError(err)
	}

	// get rides
	rides, err := s.service.ListMyRides(ctx, user)
	if err != nil {
		return nil, convertToConnectError(err)
	}

	// process rides to proto
	protoRides := make([]*carpoolv1.Ride, len(rides))
	for i, ride := range rides {
		protoRides[i] = convertRideToProto(ride)
	}

	response := &carpoolv1.ListMyRidesResponse{Rides: protoRides}

	return connect.NewResponse(response), nil
}
