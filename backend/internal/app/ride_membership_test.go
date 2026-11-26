package app

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestJoinRideSeatsTheCaller(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)

	got, err := NewService(f.fake).JoinRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID)
	if err != nil {
		t.Fatalf("JoinRide() error = %v", err)
	}

	if len(got.Riders) != 2 || got.Riders[1].ID != f.passenger {
		t.Errorf("riders = %+v, want the owner and the joining passenger", got.Riders)
	}
	if !f.fake.tx.committed {
		t.Error("JoinRide did not commit")
	}
}

// TestLeaveRideRemovesTheCallerAndTheirContactAccess checks that leaving both
// frees the seat and withdraws the contact visibility the shared ride granted.
func TestLeaveRideRemovesTheCallerAndTheirContactAccess(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t).seatPassenger()

	got, err := NewService(f.fake).LeaveRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID)
	if err != nil {
		t.Fatalf("LeaveRide() error = %v", err)
	}

	if len(got.Riders) != 1 || got.Riders[0].ID != f.owner {
		t.Errorf("riders = %+v, want only the owner", got.Riders)
	}
	if got.Owner.Email != "" {
		t.Errorf("owner email = %q, want it withdrawn once the caller left", got.Owner.Email)
	}
	if !f.fake.tx.committed {
		t.Error("LeaveRide did not commit")
	}
}

// TestOwnersCannotChangeTheirOwnMembership checks that the driver's seat is not
// theirs to give up or take twice; cancelling is how an owner withdraws a ride.
func TestOwnersCannotChangeTheirOwnMembership(t *testing.T) {
	t.Parallel()

	t.Run("join", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrOwnerCannotChangeMembership, nil, func(s *Service, f rideFixture) error {
			_, err := s.JoinRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID)
			return err
		})
	})
	t.Run("leave", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrOwnerCannotChangeMembership, nil, func(s *Service, f rideFixture) error {
			_, err := s.LeaveRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID)
			return err
		})
	})
}

func TestJoinRideFillsTheLastSeat(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)

	got, err := NewService(f.fake).JoinRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID)
	if err != nil {
		t.Fatalf("JoinRide() for the last seat error = %v, want success", err)
	}
	if got.Capacity != f.ride.Capacity {
		t.Errorf("capacity = %d, want the fixture's %d", got.Capacity, f.ride.Capacity)
	}
	if got.OccupiedSeats != f.ride.Capacity {
		t.Errorf("occupied seats = %d, want the ride full at %d", got.OccupiedSeats, f.ride.Capacity)
	}
}

func TestJoinRideRejectsImpossibleJoins(t *testing.T) {
	t.Parallel()
	join := func(s *Service, f rideFixture) error {
		_, err := s.JoinRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID)
		return err
	}

	t.Run("already aboard", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrAlreadyJoined, func(f *rideFixture) { f.seatPassenger() }, join)
	})
	t.Run("no seats left", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrFullCapacity, func(f *rideFixture) {
			f.editRide(func(r *LoadedRide) { r.Capacity = 1 })
		}, join)
	})
	t.Run("ride cancelled", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrRideCancelled, func(f *rideFixture) {
			f.editRide(func(r *LoadedRide) { r.Status = RideStatusCancelled })
		}, join)
	})
}

func TestLeaveRideRejectsNonMembers(t *testing.T) {
	t.Parallel()
	rejects(t, ErrNotJoined, nil, func(s *Service, f rideFixture) error {
		_, err := s.LeaveRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID)
		return err
	})
}

func TestMembershipChangesRejectEmptyRideID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		call func(*Service, rideFixture) error
	}{
		{"JoinRide", func(s *Service, f rideFixture) error {
			_, err := s.JoinRide(t.Context(), CurrentUser{ID: f.passenger}, uuid.Nil)
			return err
		}},
		{"LeaveRide", func(s *Service, f rideFixture) error {
			_, err := s.LeaveRide(t.Context(), CurrentUser{ID: f.passenger}, uuid.Nil)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newRideFixture(t)

			if err := tc.call(NewService(f.fake), f); !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("%s(uuid.Nil) error = %v, want ErrInvalidArgument", tc.name, err)
			}
			if f.fake.called(opBegin) {
				t.Errorf("%s opened a transaction for an empty ride ID", tc.name)
			}
		})
	}
}
