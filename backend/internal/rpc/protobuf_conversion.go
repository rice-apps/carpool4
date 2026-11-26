package rpc

import (
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// convertUserToProto returns a protobuf user with the identity and contact fields in
// user. Contact visibility must already have been applied to user.
func convertUserToProto(user app.User) *carpoolv1.User {
	return &carpoolv1.User{
		Id: user.ID.String(), FirstName: user.FirstName, LastName: user.LastName,
		Email: user.Email, Phone: user.Phone,
	}
}

// convertRideToProto copies a prepared ride result into an API message.
// The app must already have chosen the fields allowed for this viewer.
func convertRideToProto(ride app.Ride) *carpoolv1.Ride {
	// Owner and riders already contain only the contacts allowed for this viewer.
	riders := make([]*carpoolv1.User, len(ride.Riders))
	for i, rider := range ride.Riders {
		riders[i] = convertUserToProto(rider)
	}
	var owner *carpoolv1.User
	if ride.Owner.ID != uuid.Nil {
		owner = convertUserToProto(ride.Owner)
	}
	return &carpoolv1.Ride{
		Id: ride.ID.String(), DepartureTime: timestamppb.New(ride.DepartureTime),
		DepartureLocation: convertLocationToProto(ride.DepartureLocation),
		ArrivalLocation:   convertLocationToProto(ride.ArrivalLocation),
		Owner:             owner, Riders: riders, Notes: ride.Notes,
		Capacity: ride.Capacity, OccupiedSeats: ride.OccupiedSeats,
		Status: convertRideStatusToProto(ride.Status),
	}
}

// convertLocationToProto returns a protobuf location with the stored ID, title, and address.
func convertLocationToProto(location app.LocationRecord) *carpoolv1.Location {
	return &carpoolv1.Location{Id: location.ID.String(), Title: location.Title, Address: location.Address}
}

// convertRideStatusToProto returns the protobuf value for status, or UNSPECIFIED if unknown.
func convertRideStatusToProto(status app.RideStatus) carpoolv1.RideStatus {
	switch status {
	case app.RideStatusActive:
		return carpoolv1.RideStatus_RIDE_STATUS_ACTIVE
	case app.RideStatusCancelled:
		return carpoolv1.RideStatus_RIDE_STATUS_CANCELLED
	default:
		return carpoolv1.RideStatus_RIDE_STATUS_UNSPECIFIED
	}
}
