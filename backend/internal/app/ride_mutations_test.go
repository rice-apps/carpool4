package app

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mutationStore holds ride fixtures and publishes changes only at commit.
//
// Fields:
//   - profileStore: user fixtures and profile methods shared with these tests.
//   - state: last committed ride state.
//   - tx: most recent fixture transaction.
//   - fail: operation name where a test injects an error.
//   - cause: error returned at the chosen operation.
//   - skip: matching calls allowed through before the error is injected.
type mutationStore struct {
	*profileStore
	state LoadedRide
	tx    *mutationTx
	fail  string
	cause error
	skip  int
}

// mutationTx holds ride changes until Commit and can return configured errors.
//
// Fields:
//   - store: parent fixture that receives committed changes.
//   - state: ride state staged in this transaction.
//   - committed: whether the staged ride was published.
//   - rolledBack: whether rollback was called.
//   - viewer: caller ID passed to contact visibility lookup.
type mutationTx struct {
	store      *mutationStore
	state      LoadedRide
	committed  bool
	rolledBack bool
	viewer     uuid.UUID
}

// BeginTx returns a transaction over the fixture ride, or the configured
// begin or context error.
func (s *mutationStore) BeginTx(ctx context.Context) (Transaction, error) {
	if s.fail == "begin" {
		return nil, s.cause
	}
	s.tx = &mutationTx{store: s, state: s.state}
	// Copy the slice so an uncommitted membership change cannot mutate the store.
	s.tx.state.Riders = append([]UserRecord(nil), s.state.Riders...)
	return s.tx, ctx.Err()
}

// UsersRepo exposes profile and visibility methods on this transaction.
func (tx *mutationTx) UsersRepo() UserRepository { return tx }

// RidesRepo exposes the transaction's staged ride operations.
func (tx *mutationTx) RidesRepo() RideRepository { return tx }

// LocationsRepo panics because ride mutations do not need location-list storage.
func (*mutationTx) LocationsRepo() LocationRepository { panic("unexpected location call") }

// step returns an injected error for name or the context error for a normal call.
func (tx *mutationTx) step(ctx context.Context, name string) error {
	if tx.store.fail == name {
		if tx.store.skip > 0 {
			tx.store.skip--
		} else {
			return tx.store.cause
		}
	}
	return ctx.Err()
}

// FindUser returns id's fixture profile or the configured error.
func (tx *mutationTx) FindUser(ctx context.Context, id uuid.UUID) (UserRecord, error) {
	if err := tx.step(ctx, "find"); err != nil {
		return UserRecord{}, err
	}
	return tx.store.FindUser(ctx, id)
}

// CreateRide stages a new ride for owner from input and returns its ID, unless
// the fixture injects a failure.
func (tx *mutationTx) CreateRide(ctx context.Context, owner uuid.UUID, input RideWriteParams) (uuid.UUID, error) {
	if err := tx.step(ctx, "create"); err != nil {
		return uuid.Nil, err
	}
	tx.state = LoadedRide{
		ID: uuid.New(), Owner: tx.store.users[owner], DepartureTime: input.DepartureTime,
		Departure: LocationRecord{ID: input.DepartureLocationID}, Arrival: LocationRecord{ID: input.ArrivalLocationID},
		Notes: input.Notes, Capacity: input.Capacity, Status: RideStatusActive,
	}
	return tx.state.ID, nil
}

// UpdateRide stages input's editable fields on the fixture ride or fails at update.
func (tx *mutationTx) UpdateRide(ctx context.Context, id uuid.UUID, input RideWriteParams) error {
	if err := tx.step(ctx, "update"); err != nil {
		return err
	}
	tx.state.DepartureTime = input.DepartureTime
	tx.state.Departure = LocationRecord{ID: input.DepartureLocationID}
	tx.state.Arrival = LocationRecord{ID: input.ArrivalLocationID}
	tx.state.Notes = input.Notes
	tx.state.Capacity = input.Capacity
	return nil
}

// CancelRide stages a cancelled status on the fixture ride or fails at cancel.
func (tx *mutationTx) CancelRide(ctx context.Context, id uuid.UUID) error {
	if err := tx.step(ctx, "cancel"); err != nil {
		return err
	}
	tx.state.Status = RideStatusCancelled
	return nil
}

// AddRideOccupant stages userID as a rider on id, or returns the injected add error.
func (tx *mutationTx) AddRideOccupant(ctx context.Context, id, userID uuid.UUID) error {
	if err := tx.step(ctx, "add"); err != nil {
		return err
	}
	user, ok := tx.store.users[userID]
	if !ok {
		panic("wrong occupant ID")
	}
	tx.state.Riders = append(tx.state.Riders, user)
	return nil
}

// RemoveRideOccupant stages removal of userID from id and returns one affected
// row when found, zero when absent, or the injected remove error.
func (tx *mutationTx) RemoveRideOccupant(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	if err := tx.step(ctx, "remove"); err != nil {
		return 0, err
	}
	for i, user := range tx.state.Riders {
		if user.ID == userID {
			tx.state.Riders = append(tx.state.Riders[:i], tx.state.Riders[i+1:]...)
			return 1, nil
		}
	}
	return 0, nil
}

// GetRide returns the staged ride for id, ErrRideNotFound for another ID, or
// the injected load error.
func (tx *mutationTx) GetRide(ctx context.Context, id uuid.UUID) (LoadedRide, error) {
	if err := tx.step(ctx, "load"); err != nil {
		return LoadedRide{}, err
	}
	if id != tx.state.ID {
		return LoadedRide{}, ErrRideNotFound
	}
	return tx.state, nil
}

// ListRides panics because mutation tests must not run a discovery query.
func (*mutationTx) ListRides(context.Context, ListRidesQuery) ([]LoadedRide, error) {
	panic("unexpected list call")
}

// ListMyRides panics because mutation tests must not run a membership query.
func (*mutationTx) ListMyRides(context.Context, uuid.UUID) ([]LoadedRide, error) {
	panic("unexpected list call")
}

// UpsertUser panics because ride mutations must not write profiles.
func (*mutationTx) UpsertUser(context.Context, UserWriteParams) (UserRecord, error) {
	panic("unexpected call")
}

// ListContactVisibleUserIDs records viewer and makes targets visible when that
// viewer is a current rider in the staged ride, unless projection is failed.
func (tx *mutationTx) ListContactVisibleUserIDs(ctx context.Context, viewer uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	tx.viewer = viewer
	if err := tx.step(ctx, "visible"); err != nil {
		return nil, err
	}
	for _, rider := range tx.state.Riders {
		if rider.ID == viewer {
			return targets, nil
		}
	}
	return nil, nil
}

// Commit publishes the staged ride, or returns the injected commit error.
func (tx *mutationTx) Commit() error {
	if err := tx.step(context.Background(), "commit"); err != nil {
		return err
	}
	tx.committed = true
	tx.store.state = tx.state
	return nil
}

// Rollback marks the transaction closed without publishing its staged ride.
func (tx *mutationTx) Rollback() error {
	tx.rolledBack = true
	return tx.step(context.Background(), "rollback")
}

// mutationFixture returns a service, ride store, owner and passenger IDs, and
// valid ride input for mutation tests.
func mutationFixture(t *testing.T) (*Service, *mutationStore, uuid.UUID, uuid.UUID, RideWriteParams) {
	t.Helper()
	owner, passenger := uuid.New(), uuid.New()
	departure, arrival := uuid.New(), uuid.New()
	users := map[uuid.UUID]UserRecord{
		owner:     testUserRecord(owner, "owner@rice.edu"),
		passenger: testUserRecord(passenger, "passenger@rice.edu"),
	}
	input := RideWriteParams{
		DepartureTime: time.Now().Add(24 * time.Hour), DepartureLocationID: departure,
		ArrivalLocationID: arrival, Capacity: 2, Notes: "meet outside",
	}
	store := &mutationStore{
		profileStore: &profileStore{users: users},
		state: LoadedRide{
			ID: uuid.New(), DepartureTime: input.DepartureTime, Capacity: 2, Status: RideStatusActive,
			Owner: users[owner], Riders: []UserRecord{users[owner]},
			Departure: LocationRecord{ID: departure}, Arrival: LocationRecord{ID: arrival},
		},
	}
	service := NewService(store)
	return service, store, owner, passenger, input
}

// callMutation runs op for currentUser and id with input, returning its Ride or error.
func callMutation(ctx context.Context, service *Service, op string, currentUser CurrentUser, id uuid.UUID, input RideWriteParams) (Ride, error) {
	switch op {
	case "create":
		return service.CreateRide(ctx, currentUser, input)
	case "update":
		return service.UpdateRide(ctx, currentUser, id, input)
	case "cancel":
		return service.CancelRide(ctx, currentUser, id)
	case "join":
		return service.JoinRide(ctx, currentUser, id)
	case "leave":
		return service.LeaveRide(ctx, currentUser, id)
	default:
		panic("unknown mutation")
	}
}

// TestRideMutationsReturnProjectedResultsAndCommit checks that each write commits
// a viewer-safe result, including contact redaction after leaving a ride.
func TestRideMutationsReturnProjectedResultsAndCommit(t *testing.T) {
	for _, op := range []string{"create", "update", "cancel", "join", "leave"} {
		t.Run(op, func(t *testing.T) {
			service, store, owner, passenger, input := mutationFixture(t)
			caller := owner
			if op == "join" || op == "leave" {
				caller = passenger
			}
			if op == "leave" {
				store.state.Riders = append(store.state.Riders, store.users[passenger])
				store.state.Status = RideStatusCancelled
			}
			input.Capacity = 3
			got, err := callMutation(context.Background(), service, op, CurrentUser{ID: caller}, store.state.ID, input)
			if err != nil {
				t.Fatal(err)
			}
			if !store.tx.committed || store.tx.viewer != caller || got.ID != store.state.ID || got.Owner.ID != owner {
				t.Fatalf("mutation result/transaction mismatch: result=%#v tx=%#v", got, store.tx)
			}
			if op == "leave" && (got.Owner.Email != "" || len(got.Riders) != 1) {
				t.Fatalf("leave leaked contact or kept passenger: %#v", got)
			}
			if op != "leave" && got.Owner.Email != "owner@rice.edu" {
				t.Fatalf("shared/self owner contact missing: %#v", got.Owner)
			}
			if op == "join" && (len(got.Riders) != 2 || got.Riders[1].ID != passenger) {
				t.Fatalf("join result missing passenger: %#v", got)
			}
			if op == "cancel" && got.Status != RideStatusCancelled {
				t.Fatalf("cancel result status = %q", got.Status)
			}
		})
	}
}

// TestRideMutationsReturnApplicationRejections checks domain errors and confirms
// rejected ride writes never commit.
func TestRideMutationsReturnApplicationRejections(t *testing.T) {
	for _, tc := range []struct {
		name, op string
		want     error
		change   func(*mutationStore, *uuid.UUID, *RideWriteParams)
	}{
		{"phone required", "create", ErrPhoneRequired, func(s *mutationStore, caller *uuid.UUID, _ *RideWriteParams) {
			user := s.users[*caller]
			user.Phone = "  "
			s.users[*caller] = user
		}},
		{"invalid create", "create", ErrInvalidArgument, func(_ *mutationStore, _ *uuid.UUID, input *RideWriteParams) {
			input.DepartureTime = time.Now().Add(-time.Hour)
		}},
		{"nonowner update", "update", ErrPermissionDenied, func(_ *mutationStore, caller *uuid.UUID, _ *RideWriteParams) { *caller = uuid.New() }},
		{"nonowner cancel", "cancel", ErrPermissionDenied, func(_ *mutationStore, caller *uuid.UUID, _ *RideWriteParams) { *caller = uuid.New() }},
		{"cancelled update", "update", ErrRideCancelled, func(s *mutationStore, _ *uuid.UUID, _ *RideWriteParams) { s.state.Status = RideStatusCancelled }},
		{"capacity below occupancy", "update", ErrCapacityBelowOccupancy, func(s *mutationStore, _ *uuid.UUID, input *RideWriteParams) {
			s.state.Riders = append(s.state.Riders, UserRecord{ID: uuid.New()})
			input.Capacity = 1
		}},
		{"owner join", "join", ErrOwnerCannotChangeMembership, func(_ *mutationStore, _ *uuid.UUID, _ *RideWriteParams) {}},
		{"owner leave", "leave", ErrOwnerCannotChangeMembership, func(_ *mutationStore, _ *uuid.UUID, _ *RideWriteParams) {}},
		{"cancelled join", "join", ErrRideCancelled, func(s *mutationStore, caller *uuid.UUID, _ *RideWriteParams) {
			*caller = uuid.New()
			s.state.Status = RideStatusCancelled
		}},
		{"already joined", "join", ErrAlreadyJoined, func(s *mutationStore, caller *uuid.UUID, _ *RideWriteParams) {
			*caller = uuid.New()
			s.state.Riders = append(s.state.Riders, UserRecord{ID: *caller})
		}},
		{"full", "join", ErrFullCapacity, func(s *mutationStore, caller *uuid.UUID, _ *RideWriteParams) {
			*caller = uuid.New()
			s.state.Capacity = 1
		}},
		{"not joined", "leave", ErrNotJoined, func(_ *mutationStore, caller *uuid.UUID, _ *RideWriteParams) { *caller = uuid.New() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, store, caller, _, input := mutationFixture(t)
			tc.change(store, &caller, &input)
			_, err := callMutation(context.Background(), service, tc.op, CurrentUser{ID: caller}, store.state.ID, input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if store.tx != nil && (!store.tx.rolledBack || store.tx.committed) {
				t.Fatal("rejected mutation committed or failed to roll back")
			}
		})
	}
}

// TestRideMutationFailuresRollbackAndPreserveCause checks that failed writes
// return their cause and leave the original ride unchanged.
func TestRideMutationFailuresRollbackAndPreserveCause(t *testing.T) {
	for _, stage := range []string{"visible", "commit"} {
		for _, op := range []string{"create", "update", "cancel", "join", "leave"} {
			t.Run(stage+"/"+op, func(t *testing.T) {
				service, store, owner, passenger, input := mutationFixture(t)
				caller := owner
				if op == "join" || op == "leave" {
					caller = passenger
				}
				if op == "leave" {
					store.state.Riders = append(store.state.Riders, store.users[passenger])
				}
				before := store.state
				before.Riders = append([]UserRecord(nil), store.state.Riders...)
				store.fail, store.cause = stage, context.Canceled
				_, err := callMutation(context.Background(), service, op, CurrentUser{ID: caller}, store.state.ID, input)
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want canceled cause", err)
				}
				if !reflect.DeepEqual(store.state, before) || store.tx.committed || !store.tx.rolledBack {
					t.Fatal("failed result projection persisted the mutation")
				}
			})
		}
	}
}

// TestCreateRideRollsBackWhenDependentOccupantWriteFails checks that ride
// creation fails without publishing a ride when its owner cannot be added.
func TestCreateRideRollsBackWhenDependentOccupantWriteFails(t *testing.T) {
	service, store, owner, _, input := mutationFixture(t)
	before := store.state
	store.fail, store.cause = "add", errors.New("occupant write failed")

	_, err := service.CreateRide(context.Background(), CurrentUser{ID: owner}, input)
	if !errors.Is(err, store.cause) {
		t.Fatalf("CreateRide error = %v, want dependent write cause", err)
	}
	if !reflect.DeepEqual(store.state, before) || store.tx.committed || !store.tx.rolledBack {
		t.Fatal("failed dependent write persisted ride creation")
	}
}

// TestRideWriteValidationUsesApplicationTypes checks domain validation of
// dates, locations, capacity, and Unicode note length.
func TestRideWriteValidationUsesApplicationTypes(t *testing.T) {
	service, store, caller, _, valid := mutationFixture(t)
	for _, tc := range []struct {
		name   string
		change func(*RideWriteParams)
	}{
		{"past departure", func(input *RideWriteParams) { input.DepartureTime = time.Now().Add(-time.Minute) }},
		{"missing departure", func(input *RideWriteParams) { input.DepartureLocationID = uuid.Nil }},
		{"missing arrival", func(input *RideWriteParams) { input.ArrivalLocationID = uuid.Nil }},
		{"same locations", func(input *RideWriteParams) { input.ArrivalLocationID = input.DepartureLocationID }},
		{"zero capacity", func(input *RideWriteParams) { input.Capacity = 0 }},
		{"long notes", func(input *RideWriteParams) { input.Notes = strings.Repeat("ç•Œ", 501) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.change(&input)
			_, err := service.UpdateRide(context.Background(), CurrentUser{ID: caller}, store.state.ID, input)
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("error = %v, want invalid argument", err)
			}
		})
	}
}

// TestCancelRideReportsProjectionFailure checks that a lookup failure after the
// cancel reaches the caller instead of being overwritten by the commit result.
func TestCancelRideReportsProjectionFailure(t *testing.T) {
	service, store, owner, _, _ := mutationFixture(t)
	// Let the ownership check load the ride, then fail the lookup that follows the cancel.
	store.fail, store.cause, store.skip = "visible", errors.New("projection unavailable"), 1

	got, err := service.CancelRide(context.Background(), CurrentUser{ID: owner}, store.state.ID)
	if !errors.Is(err, store.cause) {
		t.Fatalf("CancelRide error = %v, want the injected projection failure", err)
	}
	if got.ID != uuid.Nil {
		t.Errorf("CancelRide ride = %#v, want the zero ride", got)
	}
	if store.tx.committed {
		t.Error("CancelRide committed after the projection failed")
	}
}

// TestRideOperationsRejectNilRideID checks every ride operation refuses an empty ID
// before opening a transaction.
func TestRideOperationsRejectNilRideID(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		call func(*Service, CurrentUser) error
	}{
		{"GetRide", func(s *Service, c CurrentUser) error { _, err := s.GetRide(ctx, c, uuid.Nil); return err }},
		{"UpdateRide", func(s *Service, c CurrentUser) error {
			_, err := s.UpdateRide(ctx, c, uuid.Nil, RideWriteParams{})
			return err
		}},
		{"CancelRide", func(s *Service, c CurrentUser) error { _, err := s.CancelRide(ctx, c, uuid.Nil); return err }},
		{"JoinRide", func(s *Service, c CurrentUser) error { _, err := s.JoinRide(ctx, c, uuid.Nil); return err }},
		{"LeaveRide", func(s *Service, c CurrentUser) error { _, err := s.LeaveRide(ctx, c, uuid.Nil); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, store, owner, _, _ := mutationFixture(t)
			if err := tc.call(service, CurrentUser{ID: owner}); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("%s error = %v, want invalid argument", tc.name, err)
			}
			if store.tx != nil {
				t.Errorf("%s opened a transaction for an empty ride ID", tc.name)
			}
		})
	}
}
