package rpc_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// TestPastSharedRideGrantsSelectiveContactVisibility checks historical co-rider access.
// A past cancelled ride reveals shared users' contacts, but unrelated passengers remain redacted.
func TestPastSharedRideGrantsSelectiveContactVisibility(t *testing.T) {
	f := newDBMutationFixture(t)
	f.request.Capacity = 3
	id := f.create(t)
	if _, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id})); err != nil {
		t.Fatal(err)
	}
	owner := f.owner.String()
	before, err := f.server.GetUser(dbCaller(f.second), connect.NewRequest(&v1.GetUserRequest{UserId: &owner}))
	if err != nil {
		t.Fatal(err)
	}
	assertRPCContactRedacted(t, before.Msg.User, f.owner)
	past := uuid.New()
	if _, err := f.db.Exec(`INSERT INTO rides (id, departure_time, departure_location_id, arrival_location_id, owner_id, capacity, status) VALUES ($1, now() - interval '1 year', $2, $3, $4, 2, 'cancelled')`, past, f.request.DepartureLocationId, f.request.ArrivalLocationId, f.owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`INSERT INTO ride_occupants (ride_id, user_id) VALUES ($1, $2), ($1, $3)`, past, f.owner, f.second); err != nil {
		t.Fatal(err)
	}
	user, err := f.server.GetUser(dbCaller(f.second), connect.NewRequest(&v1.GetUserRequest{UserId: &owner}))
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user.Msg.User.Email == "" || user.Msg.User.Phone == "" {
		t.Errorf("contacts = email %q phone %q, want both visible after a past shared ride",
			user.Msg.User.Email, user.Msg.User.Phone)
	}
	ride, err := f.server.GetRide(dbCaller(f.second), connect.NewRequest(&v1.GetRideRequest{Id: id}))
	if err != nil {
		t.Fatal(err)
	}
	if ride.Msg.Ride.Owner.Email != owner+"@rice.edu" {
		t.Fatal("past ride did not grant owner contact")
	}
	var checkedRider bool
	for _, rider := range ride.Msg.Ride.Riders {
		if rider.Id == f.first.String() {
			assertRPCContactRedacted(t, rider, f.first)
			checkedRider = true
		}
	}
	if !checkedRider {
		t.Fatalf("ride %s has no rider %s, so its redaction was never checked", id, f.first)
	}
	mine, err := f.server.ListMyRides(dbCaller(f.first), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil {
		t.Fatalf("ListMyRides() error = %v", err)
	}
	if len(mine.Msg.Rides) != 1 {
		t.Fatalf("participant history = %d rides, want 1", len(mine.Msg.Rides))
	}
	if mine.Msg.Rides[0].Owner.Email == "" {
		t.Error("owner email is hidden in the participant's own history, want it visible")
	}
	participant, err := f.server.GetRide(dbCaller(f.first), connect.NewRequest(&v1.GetRideRequest{Id: id}))
	if err != nil {
		t.Fatalf("GetRide() error = %v", err)
	}
	if participant.Msg.Ride.Owner.Email == "" {
		t.Error("owner email is hidden from a participant, want it visible")
	}
}
