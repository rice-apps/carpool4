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
	//first check if user is signed in --> if they are signed in, get user info and create an identity object, else create empty identity object
	//pass down empty identity object for app layer method
	//pass ctx down so it can be used in the storage layer method
	//pass req down so database request may be filtered later in app layer method
	//receive returned outputs from app and storage layer methods, present results to user, or return error

	//create currentUser object and initialize with user values (or empty if guest)
	currentUser := app.CurrentUser{}
	if auth.IsSignedIn(ctx) {
		var err error
		currentUser, err = getCurrentUser(ctx)
		if err != nil {
			return nil, convertToConnectError(err)
		}
	}

	//process raw values from ListRidesRequest and put values into new ListRidesInput object
	requestInput := app.ListRidesInput{}

	departureAfterTime := req.Msg.DepartureAfter.AsTime()
	departureAfterPointer := &departureAfterTime

	departureBeforeTime := req.Msg.DepartureBefore.AsTime()
	departureBeforePointer := &departureBeforeTime

	parsedArrivalID, err1 := uuid.Parse(*req.Msg.ArrivalLocationId)
	arrivalIDPointer := &parsedArrivalID
	if err1 != nil {
		return nil, convertToConnectError(err1)
	}

	parsedDepartureID, err2 := uuid.Parse(*req.Msg.DepartureLocationId)
	departureIDPointer := &parsedDepartureID
	if err2 != nil {
		return nil, convertToConnectError(err2)
	}

	requestInput.ArrivalLocationID = arrivalIDPointer
	requestInput.DepartureLocationID = departureIDPointer
	requestInput.DepartureAfter = departureAfterPointer
	requestInput.DepartureBefore = departureBeforePointer

	//call appy layer ListRides method with provided arguments
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
	return nil, convertToConnectError(app.ErrNotImplemented)
}
