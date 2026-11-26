package rpc

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// TestConvertRideToProtoOmitsPeopleForGuests checks that a ride the application
// already redacted converts to a message with no owner and no riders, rather
// than to an empty User that a client could mistake for a real one.
func TestConvertRideToProtoOmitsPeopleForGuests(t *testing.T) {
	t.Parallel()
	departure := time.Now().Truncate(time.Second)
	ride := app.Ride{ID: uuid.New(), DepartureTime: departure, Capacity: 2, OccupiedSeats: 1}

	got := convertRideToProto(ride)

	want := &carpoolv1.Ride{
		Id:            ride.ID.String(),
		DepartureTime: timestamppb.New(departure),
		// Both locations are zero values, so their IDs convert to the nil UUID.
		DepartureLocation: &carpoolv1.Location{Id: uuid.Nil.String()},
		ArrivalLocation:   &carpoolv1.Location{Id: uuid.Nil.String()},
		Capacity:          2,
		OccupiedSeats:     1,
		Riders:            []*carpoolv1.User{},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("convertRideToProto() mismatch (-want +got):\n%s", diff)
	}
}

// TestConvertRideToProtoKeepsVisibleContacts checks that contact fields the
// application chose to reveal survive conversion.
func TestConvertRideToProtoKeepsVisibleContacts(t *testing.T) {
	t.Parallel()
	owner := app.User{ID: uuid.New(), FirstName: "Ada", LastName: "Lovelace", Email: "ada@rice.edu", Phone: "713-555-0100"}
	ride := app.Ride{ID: uuid.New(), DepartureTime: time.Now(), Owner: owner, Riders: []app.User{owner}, Capacity: 2, OccupiedSeats: 1}

	got := convertRideToProto(ride)

	wantOwner := &carpoolv1.User{Id: owner.ID.String(), FirstName: "Ada", LastName: "Lovelace", Email: "ada@rice.edu", Phone: "713-555-0100"}
	if diff := cmp.Diff(wantOwner, got.Owner, protocmp.Transform()); diff != "" {
		t.Errorf("owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]*carpoolv1.User{wantOwner}, got.Riders, protocmp.Transform()); diff != "" {
		t.Errorf("riders mismatch (-want +got):\n%s", diff)
	}
}
