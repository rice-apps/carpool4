package rpc

import (
	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// userToProto returns a protobuf user with the identity and contact fields in
// user. Contact visibility must already have been applied to user.
func userToProto(user app.User) *v1.User {
	return &v1.User{
		Id: user.ID.String(), FirstName: user.FirstName, LastName: user.LastName,
		Email: user.Email, Phone: user.Phone,
	}
}

// rideToProto returns a protobuf ride with its people, route, departure, notes,
// capacity, and status. Contacts must already be projected for the viewer.
func rideToProto(ride app.Ride) *v1.Ride {
	// Riders and owner carry the service's viewer-specific contact projection.
	riders := make([]*v1.User, len(ride.Riders))
	for i, rider := range ride.Riders {
		riders[i] = userToProto(rider)
	}
	return &v1.Ride{
		Id: ride.ID.String(), DepartureDate: timestamppb.New(ride.DepartureDate),
		DepartureLocation: locationToProto(ride.DepartureLocation),
		ArrivalLocation:   locationToProto(ride.ArrivalLocation),
		Owner:             userToProto(ride.Owner), Riders: riders, Notes: ride.Notes,
		Capacity: ride.Capacity, Status: rideStatusToProto(ride.Status),
	}
}

// locationToProto returns a protobuf location with the stored ID, title, and address.
func locationToProto(location app.LocationRecord) *v1.Location {
	return &v1.Location{Id: location.ID.String(), Title: location.Title, Address: location.Address}
}

// rideStatusToProto returns the protobuf value for status, or UNSPECIFIED if unknown.
func rideStatusToProto(status app.RideStatus) v1.RideStatus {
	switch status {
	case app.RideStatusActive:
		return v1.RideStatus_RIDE_STATUS_ACTIVE
	case app.RideStatusCancelled:
		return v1.RideStatus_RIDE_STATUS_CANCELLED
	default:
		return v1.RideStatus_RIDE_STATUS_UNSPECIFIED
	}
}
