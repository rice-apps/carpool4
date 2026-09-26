package rpc

import (
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// userToProto returns a protobuf user with the identity and contact fields in
// user. Contact visibility must already have been applied to user.
func userToProto(user app.User) *carpoolv1.User {
	return &carpoolv1.User{
		Id: user.ID.String(), FirstName: user.FirstName, LastName: user.LastName,
		Email: user.Email, Phone: user.Phone,
	}
}

// rideToProto returns a protobuf ride with the trip fields projected for the
// viewer. Personal fields must already be omitted for anonymous viewers.
func rideToProto(ride app.Ride) *carpoolv1.Ride {
	// Riders and owner carry the service's viewer-specific contact projection.
	riders := make([]*carpoolv1.User, len(ride.Riders))
	for i, rider := range ride.Riders {
		riders[i] = userToProto(rider)
	}
	var owner *carpoolv1.User
	if ride.Owner.ID != uuid.Nil {
		owner = userToProto(ride.Owner)
	}
	return &carpoolv1.Ride{
		Id: ride.ID.String(), DepartureTime: timestamppb.New(ride.DepartureTime),
		DepartureLocation: locationToProto(ride.DepartureLocation),
		ArrivalLocation:   locationToProto(ride.ArrivalLocation),
		Owner:             owner, Riders: riders, Notes: ride.Notes,
		Capacity: ride.Capacity, OccupiedSeats: ride.OccupiedSeats,
		Status: rideStatusToProto(ride.Status),
	}
}

// locationToProto returns a protobuf location with the stored ID, title, and address.
func locationToProto(location app.LocationRecord) *carpoolv1.Location {
	return &carpoolv1.Location{Id: location.ID.String(), Title: location.Title, Address: location.Address}
}

// rideStatusToProto returns the protobuf value for status, or UNSPECIFIED if unknown.
func rideStatusToProto(status app.RideStatus) carpoolv1.RideStatus {
	switch status {
	case app.RideStatusActive:
		return carpoolv1.RideStatus_RIDE_STATUS_ACTIVE
	case app.RideStatusCancelled:
		return carpoolv1.RideStatus_RIDE_STATUS_CANCELLED
	default:
		return carpoolv1.RideStatus_RIDE_STATUS_UNSPECIFIED
	}
}
