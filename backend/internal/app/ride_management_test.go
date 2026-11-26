package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestCreateRideSeatsTheOwnerAndCommits(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)

	got, err := NewService(f.fake).CreateRide(t.Context(), CurrentUser{ID: f.owner}, f.input)
	if err != nil {
		t.Fatalf("CreateRide() error = %v", err)
	}

	if got.Owner.ID != f.owner {
		t.Errorf("owner = %v, want the caller %v", got.Owner.ID, f.owner)
	}
	// The owner occupies one of their own seats, so capacity counts them.
	if len(got.Riders) != 1 || got.Riders[0].ID != f.owner {
		t.Errorf("riders = %+v, want only the owner", got.Riders)
	}
	if !f.fake.tx.committed {
		t.Error("CreateRide did not commit")
	}
}

func TestCreateRideRequiresAReachableOwner(t *testing.T) {
	t.Parallel()
	create := func(s *Service, f rideFixture) error {
		_, err := s.CreateRide(t.Context(), CurrentUser{ID: f.owner}, f.input)
		return err
	}

	// Riders contact the driver by phone, so both of these leave them unreachable
	// and answer the same way.
	t.Run("owner has no phone number", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrPhoneRequired, func(f *rideFixture) {
			f.editOwner(func(o *UserRecord) { o.Phone = "  " })
		}, create)
	})
	t.Run("owner has no profile", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrPhoneRequired, func(f *rideFixture) { delete(f.fake.users, f.owner) }, create)
	})
}

// TestCreateRideRollsBackWhenSeatingTheOwnerFails checks that a ride is never
// published without its owner aboard.
func TestCreateRideRollsBackWhenSeatingTheOwnerFails(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)
	cause := errors.New("occupant write failed")
	f.fake.failing(opAddOccupant, cause)

	if _, err := NewService(f.fake).CreateRide(t.Context(), CurrentUser{ID: f.owner}, f.input); !errors.Is(err, cause) {
		t.Errorf("CreateRide() error = %v, want %v", err, cause)
	}
	if f.fake.tx.committed {
		t.Error("CreateRide committed a ride whose owner could not be seated")
	}
	if len(f.fake.rides) != 1 {
		t.Errorf("published %d rides, want only the fixture ride", len(f.fake.rides))
	}
}

func TestUpdateRideStoresNewDetails(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)
	f.input.Notes = "new meeting point"

	got, err := NewService(f.fake).UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input)
	if err != nil {
		t.Fatalf("UpdateRide() error = %v", err)
	}

	if got.Notes != "new meeting point" || got.Capacity != 3 {
		t.Errorf("UpdateRide() = notes %q capacity %d, want %q and 3", got.Notes, got.Capacity, "new meeting point")
	}
	if !f.fake.tx.committed {
		t.Error("UpdateRide did not commit")
	}
}

func TestCancelRideStoresCancelledStatus(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)

	got, err := NewService(f.fake).CancelRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID)
	if err != nil {
		t.Fatalf("CancelRide() error = %v", err)
	}

	if got.Status != RideStatusCancelled {
		t.Errorf("status = %q, want %q", got.Status, RideStatusCancelled)
	}
	if !f.fake.tx.committed {
		t.Error("CancelRide did not commit")
	}
}

// TestCancelRideReportsProjectionFailure checks that a lookup failure after the
// cancel reaches the caller rather than being replaced by the commit result.
func TestCancelRideReportsProjectionFailure(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)
	cause := errors.New("projection unavailable")
	// The ownership check consults permissions first; fail the lookup after it.
	f.fake.failingAfter(1, opVisibleContacts, cause)

	got, err := NewService(f.fake).CancelRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID)
	if !errors.Is(err, cause) {
		t.Errorf("CancelRide() error = %v, want %v", err, cause)
	}
	if diff := cmp.Diff(Ride{}, got); diff != "" {
		t.Errorf("CancelRide() returned a ride despite the failure (-want +got):\n%s", diff)
	}
	if f.fake.tx.committed {
		t.Error("CancelRide committed after the projection failed")
	}
}

func TestRideWritesRequireTheOwner(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrPermissionDenied, nil, func(s *Service, f rideFixture) error {
			_, err := s.UpdateRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID, f.input)
			return err
		})
	})
	t.Run("cancel", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrPermissionDenied, nil, func(s *Service, f rideFixture) error {
			_, err := s.CancelRide(t.Context(), CurrentUser{ID: f.passenger}, f.ride.ID)
			return err
		})
	})
}

func TestRideWriteValidationRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(*RideWriteParams)
	}{
		{name: "departure in the past", change: func(in *RideWriteParams) { in.DepartureTime = time.Now().Add(-time.Minute) }},
		{name: "missing departure location", change: func(in *RideWriteParams) { in.DepartureLocationID = uuid.Nil }},
		{name: "missing arrival location", change: func(in *RideWriteParams) { in.ArrivalLocationID = uuid.Nil }},
		{name: "identical locations", change: func(in *RideWriteParams) { in.ArrivalLocationID = in.DepartureLocationID }},
		{name: "capacity below one", change: func(in *RideWriteParams) { in.Capacity = 0 }},
		{name: "notes over the length limit", change: func(in *RideWriteParams) { in.Notes = strings.Repeat("界", 501) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newRideFixture(t)
			tc.change(&f.input)

			if _, err := NewService(f.fake).UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input); !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("UpdateRide() error = %v, want ErrInvalidArgument", err)
			}
			if f.fake.called(opUpdateRide) {
				t.Error("UpdateRide wrote invalid details")
			}
		})
	}
}

func TestUpdateRideReportsACommitFailure(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)
	cause := errors.New("commit failed")
	f.fake.failing(opCommit, cause)

	if _, err := NewService(f.fake).UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input); !errors.Is(err, cause) {
		t.Errorf("UpdateRide() error = %v, want %v", err, cause)
	}
}

func TestUpdateRideRejectsCancelledRides(t *testing.T) {
	t.Parallel()
	rejects(t, ErrRideCancelled, func(f *rideFixture) {
		f.editRide(func(r *LoadedRide) { r.Status = RideStatusCancelled })
	}, func(s *Service, f rideFixture) error {
		_, err := s.UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input)
		return err
	})
}

// TestUpdateRideCapacityMustFitOccupancy checks both sides of the rule: the
// seats already taken set the floor, and matching that floor exactly is allowed.
func TestUpdateRideCapacityMustFitOccupancy(t *testing.T) {
	t.Parallel()

	t.Run("below the seats taken", func(t *testing.T) {
		t.Parallel()
		rejects(t, ErrCapacityBelowOccupancy, func(f *rideFixture) {
			f.seatPassenger()
			f.input.Capacity = 1
		}, func(s *Service, f rideFixture) error {
			_, err := s.UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input)
			return err
		})
	})

	t.Run("equal to the seats taken", func(t *testing.T) {
		t.Parallel()
		f := newRideFixture(t).seatPassenger()
		f.input.Capacity = 2

		if _, err := NewService(f.fake).UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input); err != nil {
			t.Errorf("UpdateRide() with capacity equal to occupancy error = %v, want success", err)
		}
	})
}

func TestUpdateRideAcceptsNotesAtTheLengthLimit(t *testing.T) {
	t.Parallel()
	f := newRideFixture(t)
	f.input.Notes = strings.Repeat("界", 500)

	if _, err := NewService(f.fake).UpdateRide(t.Context(), CurrentUser{ID: f.owner}, f.ride.ID, f.input); err != nil {
		t.Errorf("UpdateRide() with 500-rune notes error = %v, want success", err)
	}
}

func TestRideWritesRejectEmptyRideID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		call func(*Service, rideFixture) error
	}{
		{"UpdateRide", func(s *Service, f rideFixture) error {
			_, err := s.UpdateRide(t.Context(), CurrentUser{ID: f.owner}, uuid.Nil, f.input)
			return err
		}},
		{"CancelRide", func(s *Service, f rideFixture) error {
			_, err := s.CancelRide(t.Context(), CurrentUser{ID: f.owner}, uuid.Nil)
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

// TestRideWritesAuthorizeBeforeValidating checks that a caller who may not
// perform a write is refused on that ground, whatever is wrong with their
// input. Reversing the order would tell an unauthorized caller which of their
// fields the server dislikes.
func TestRideWritesAuthorizeBeforeValidating(t *testing.T) {
	t.Parallel()

	t.Run("create reports the missing phone, not the bad input", func(t *testing.T) {
		t.Parallel()
		f := newRideFixture(t).editOwner(func(o *UserRecord) { o.Phone = "" })

		if _, err := NewService(f.fake).CreateRide(t.Context(), CurrentUser{ID: f.owner}, RideWriteParams{}); !errors.Is(err, ErrPhoneRequired) {
			t.Errorf("CreateRide() with no phone and empty input error = %v, want ErrPhoneRequired", err)
		}
		if f.fake.called(opCreateRide) {
			t.Error("CreateRide wrote a ride for a caller without a phone number")
		}
	})

	t.Run("update reports the denied owner, not the bad input", func(t *testing.T) {
		t.Parallel()
		f := newRideFixture(t)

		if _, err := NewService(f.fake).UpdateRide(t.Context(), CurrentUser{ID: uuid.New()}, f.ride.ID, RideWriteParams{}); !errors.Is(err, ErrPermissionDenied) {
			t.Errorf("UpdateRide() by a stranger with empty input error = %v, want ErrPermissionDenied", err)
		}
		if f.fake.called(opUpdateRide) {
			t.Error("UpdateRide wrote details for a caller who does not own the ride")
		}
	})
}
