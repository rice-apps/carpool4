package rpc

import (
	"context"

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
		departureAfterTime := req.Msg.DepartureAfter.AsTime()
		requestInput.DepartureAfter = &departureAfterTime
	}

	if req.Msg.GetDepartureBefore() != nil {
		departureBeforeTime := req.Msg.DepartureBefore.AsTime()
		requestInput.DepartureBefore = &departureBeforeTime
	}

	if req.Msg.ArrivalLocationId != nil {
		parsedArrivalID, err1 := uuid.Parse(*req.Msg.ArrivalLocationId)
		if err1 != nil {
			return nil, convertToConnectError(err1)
		}
		requestInput.ArrivalLocationID = &parsedArrivalID
	}

	if req.Msg.DepartureLocationId != nil {
		parsedDepartureID, err2 := uuid.Parse(*req.Msg.DepartureLocationId)
		if err2 != nil {
			return nil, convertToConnectError(err2)
		}
		requestInput.DepartureLocationID = &parsedDepartureID
	}

	//call app layer ListRides method with provided arguments
	rideList, rideListError := s.service.ListRides(ctx, currentUser, requestInput)
	if rideListError != nil {
		return nil, convertToConnectError(rideListError)
	}
	newResponse := connect.Response[carpoolv1.ListRidesResponse]{}
	ridePointerList := make([]*carpoolv1.Ride, len(rideList))
	for i, ride := range rideList {
		ridePointerList[i] = convertRideToProto(ride)
	}
	newResponse.Msg.Rides = ridePointerList
	newResponsePointer := &newResponse
	return newResponsePointer, nil
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
