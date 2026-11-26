package rpc

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

func TestMembershipHandlersRejectMalformedRideID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		call func(*Server) error
	}{
		{"JoinRide", func(s *Server) error {
			_, err := s.JoinRide(testContext(uuid.New(), "rider@rice.edu"),
				connect.NewRequest(&carpoolv1.JoinRideRequest{RideId: "not-a-uuid"}))
			return err
		}},
		{"LeaveRide", func(s *Server) error {
			_, err := s.LeaveRide(testContext(uuid.New(), "rider@rice.edu"),
				connect.NewRequest(&carpoolv1.LeaveRideRequest{RideId: "not-a-uuid"}))
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeDatabase{}

			err := tc.call(newTestServer(fake))

			if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
				t.Errorf("%s() code = %v, want %v", tc.name, got, connect.CodeInvalidArgument)
			}
			if fake.called(opBegin) {
				t.Errorf("%s reached storage with an unparseable ride ID", tc.name)
			}
		})
	}
}

func TestJoinRideReturnsTheUpdatedRide(t *testing.T) {
	t.Parallel()
	owner := testUser(uuid.New(), "owner@rice.edu")
	rider := testUser(uuid.New(), "rider@rice.edu")
	ride := app.LoadedRide{
		ID: uuid.New(), DepartureTime: time.Now().Add(time.Hour), Owner: owner,
		Riders: []app.UserRecord{owner}, Capacity: 2, Status: app.RideStatusActive,
	}
	fake := &fakeDatabase{
		users: map[uuid.UUID]app.UserRecord{owner.ID: owner, rider.ID: rider},
		rides: map[uuid.UUID]app.LoadedRide{ride.ID: ride},
	}

	got, err := newTestServer(fake).JoinRide(testContext(rider.ID, rider.Email),
		connect.NewRequest(&carpoolv1.JoinRideRequest{RideId: ride.ID.String()}))
	if err != nil {
		t.Fatalf("JoinRide() error = %v", err)
	}

	if len(got.Msg.Ride.Riders) != 2 {
		t.Errorf("riders = %d, want the owner and the new rider", len(got.Msg.Ride.Riders))
	}
	if !fake.called(opCommit) {
		t.Error("JoinRide did not commit")
	}
}

// TestLeaveRideMapsAbsentMembershipToFailedPrecondition checks the handler maps
// the application's refusal rather than reporting a missing ride.
func TestLeaveRideMapsAbsentMembershipToFailedPrecondition(t *testing.T) {
	t.Parallel()
	owner := testUser(uuid.New(), "owner@rice.edu")
	stranger := testUser(uuid.New(), "stranger@rice.edu")
	ride := app.LoadedRide{
		ID: uuid.New(), DepartureTime: time.Now().Add(time.Hour), Owner: owner,
		Riders: []app.UserRecord{owner}, Capacity: 3, Status: app.RideStatusActive,
	}
	fake := &fakeDatabase{
		users: map[uuid.UUID]app.UserRecord{owner.ID: owner, stranger.ID: stranger},
		rides: map[uuid.UUID]app.LoadedRide{ride.ID: ride},
	}

	_, err := newTestServer(fake).LeaveRide(testContext(stranger.ID, stranger.Email),
		connect.NewRequest(&carpoolv1.LeaveRideRequest{RideId: ride.ID.String()}))

	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("LeaveRide() code = %v, want %v", got, connect.CodeFailedPrecondition)
	}
}
