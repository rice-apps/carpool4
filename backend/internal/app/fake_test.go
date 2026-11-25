package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

// storageOp names a storage call. Tests inject failures and assert call order
// with these rather than with bare strings, so a typo fails to compile.
type storageOp string

const (
	opBegin           storageOp = "BeginTx"
	opCommit          storageOp = "Commit"
	opRollback        storageOp = "Rollback"
	opFindUser        storageOp = "FindUser"
	opUpsertUser      storageOp = "UpsertUser"
	opVisibleContacts storageOp = "ListContactVisibleUserIDs"
	opGetRide         storageOp = "GetRide"
	opListRides       storageOp = "ListRides"
	opListMyRides     storageOp = "ListMyRides"
	opCreateRide      storageOp = "CreateRide"
	opUpdateRide      storageOp = "UpdateRide"
	opCancelRide      storageOp = "CancelRide"
	opAddOccupant     storageOp = "AddRideOccupant"
	opRemoveOccupant  storageOp = "RemoveRideOccupant"
	opListLocations   storageOp = "ListLocations"
)

// fakeDatabase is the package's only storage double. It serves fixture users, rides
// and locations, records every call, and can fail any one of them.
//
// Writes are staged on the transaction and published to the fake only on
// Commit, so a test can tell "the service wrote" from "the service committed".
type fakeDatabase struct {
	users     map[uuid.UUID]UserRecord
	rides     []LoadedRide
	locations []LocationRecord

	// visible reports whether viewer may see target's contact details. The zero
	// value hides every contact except the viewer's own.
	visible func(viewer, target uuid.UUID) bool

	// failOp fails after failAfter successful calls; zero fails the first.
	failOp    storageOp
	failErr   error
	failAfter int

	calls []storageOp
	tx    *fakeTransaction
}

// fakeTransaction holds writes until Commit publishes them to its fakeDatabase.
type fakeTransaction struct {
	db        *fakeDatabase
	rides     []LoadedRide
	savedUser UserWriteParams
	committed bool

	// Query arguments recorded for assertions.
	ridesQuery     *ListRidesQuery
	myRidesViewer  uuid.UUID
	contactViewers []uuid.UUID
	contactTargets [][]uuid.UUID
}

// newFakeDatabase returns a fake with the given users and rides and no injected failure.
func newFakeDatabase(users []UserRecord, rides ...LoadedRide) *fakeDatabase {
	byID := make(map[uuid.UUID]UserRecord, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}
	return &fakeDatabase{users: byID, rides: rides}
}

// failing returns the fake configured to fail op with err.
func (db *fakeDatabase) failing(op storageOp, err error) *fakeDatabase {
	db.failOp, db.failErr = op, err
	return db
}

// failingAfter returns the fake configured to fail op once it has succeeded n times.
func (db *fakeDatabase) failingAfter(n int, op storageOp, err error) *fakeDatabase {
	db.failAfter = n
	return db.failing(op, err)
}

// showingContacts returns the fake configured to reveal every target's contacts.
func (db *fakeDatabase) showingContacts() *fakeDatabase {
	db.visible = func(uuid.UUID, uuid.UUID) bool { return true }
	return db
}

// called reports whether the service made this call.
func (db *fakeDatabase) called(op storageOp) bool { return slices.Contains(db.calls, op) }

// record logs op and returns the injected failure when op is the one to fail.
func (db *fakeDatabase) record(op storageOp) error {
	db.calls = append(db.calls, op)
	if op != db.failOp {
		return nil
	}
	if db.failAfter > 0 {
		db.failAfter--
		return nil
	}
	return db.failErr
}

func (db *fakeDatabase) BeginTx(context.Context) (Transaction, error) {
	if err := db.record(opBegin); err != nil {
		return nil, err
	}
	db.tx = &fakeTransaction{db: db, rides: slices.Clone(db.rides)}
	return db.tx, nil
}

func (tx *fakeTransaction) UsersRepo() UserRepository         { return tx }
func (tx *fakeTransaction) RidesRepo() RideRepository         { return tx }
func (tx *fakeTransaction) LocationsRepo() LocationRepository { return tx }

func (tx *fakeTransaction) Commit() error {
	if err := tx.db.record(opCommit); err != nil {
		return err
	}
	tx.committed = true
	tx.db.rides = tx.rides
	return nil
}

func (tx *fakeTransaction) Rollback() error {
	_ = tx.db.record(opRollback)
	return nil
}

func (tx *fakeTransaction) FindUser(_ context.Context, id uuid.UUID) (UserRecord, error) {
	if err := tx.db.record(opFindUser); err != nil {
		return UserRecord{}, err
	}
	user, ok := tx.db.users[id]
	if !ok {
		return UserRecord{}, ErrUserNotFound
	}
	return user, nil
}

func (tx *fakeTransaction) UpsertUser(_ context.Context, params UserWriteParams) (UserRecord, error) {
	if err := tx.db.record(opUpsertUser); err != nil {
		return UserRecord{}, err
	}
	tx.savedUser = params
	return UserRecord{
		ID: params.ID, FirstName: params.FirstName, LastName: params.LastName,
		Email: params.Email, Phone: params.Phone,
	}, nil
}

func (tx *fakeTransaction) ListContactVisibleUserIDs(_ context.Context, viewerID uuid.UUID, targetIDs []uuid.UUID) ([]uuid.UUID, error) {
	if err := tx.db.record(opVisibleContacts); err != nil {
		return nil, err
	}
	tx.contactViewers = append(tx.contactViewers, viewerID)
	tx.contactTargets = append(tx.contactTargets, slices.Clone(targetIDs))
	if tx.db.visible == nil {
		return nil, nil
	}
	allowed := make([]uuid.UUID, 0, len(targetIDs))
	for _, target := range targetIDs {
		if tx.db.visible(viewerID, target) {
			allowed = append(allowed, target)
		}
	}
	return allowed, nil
}

func (tx *fakeTransaction) GetRide(_ context.Context, rideID uuid.UUID) (LoadedRide, error) {
	if err := tx.db.record(opGetRide); err != nil {
		return LoadedRide{}, err
	}
	for _, ride := range tx.rides {
		if ride.ID == rideID {
			return ride, nil
		}
	}
	return LoadedRide{}, ErrRideNotFound
}

func (tx *fakeTransaction) ListRides(_ context.Context, query ListRidesQuery) ([]LoadedRide, error) {
	if err := tx.db.record(opListRides); err != nil {
		return nil, err
	}
	tx.ridesQuery = &query
	return tx.rides, nil
}

func (tx *fakeTransaction) ListMyRides(_ context.Context, viewerID uuid.UUID) ([]LoadedRide, error) {
	if err := tx.db.record(opListMyRides); err != nil {
		return nil, err
	}
	tx.myRidesViewer = viewerID
	return tx.rides, nil
}

func (tx *fakeTransaction) CreateRide(_ context.Context, ownerID uuid.UUID, input RideWriteParams) (uuid.UUID, error) {
	if err := tx.db.record(opCreateRide); err != nil {
		return uuid.Nil, err
	}
	ride := LoadedRide{
		ID: uuid.New(), Owner: tx.db.users[ownerID], DepartureTime: input.DepartureTime,
		Departure: LocationRecord{ID: input.DepartureLocationID},
		Arrival:   LocationRecord{ID: input.ArrivalLocationID},
		Notes:     input.Notes, Capacity: input.Capacity, Status: RideStatusActive,
	}
	tx.rides = append(tx.rides, ride)
	return ride.ID, nil
}

func (tx *fakeTransaction) UpdateRide(_ context.Context, rideID uuid.UUID, input RideWriteParams) error {
	if err := tx.db.record(opUpdateRide); err != nil {
		return err
	}
	return tx.mutate(rideID, func(ride *LoadedRide) {
		ride.DepartureTime = input.DepartureTime
		ride.Departure = LocationRecord{ID: input.DepartureLocationID}
		ride.Arrival = LocationRecord{ID: input.ArrivalLocationID}
		ride.Notes, ride.Capacity = input.Notes, input.Capacity
	})
}

func (tx *fakeTransaction) CancelRide(_ context.Context, rideID uuid.UUID) error {
	if err := tx.db.record(opCancelRide); err != nil {
		return err
	}
	return tx.mutate(rideID, func(ride *LoadedRide) { ride.Status = RideStatusCancelled })
}

func (tx *fakeTransaction) AddRideOccupant(_ context.Context, rideID, userID uuid.UUID) error {
	if err := tx.db.record(opAddOccupant); err != nil {
		return err
	}
	return tx.mutate(rideID, func(ride *LoadedRide) {
		ride.Riders = append(slices.Clone(ride.Riders), tx.db.users[userID])
	})
}

func (tx *fakeTransaction) RemoveRideOccupant(_ context.Context, rideID, userID uuid.UUID) (int64, error) {
	if err := tx.db.record(opRemoveOccupant); err != nil {
		return 0, err
	}
	var removed int64
	err := tx.mutate(rideID, func(ride *LoadedRide) {
		riders := slices.Clone(ride.Riders)
		for i, rider := range riders {
			if rider.ID == userID {
				ride.Riders = slices.Delete(riders, i, i+1)
				removed = 1
				return
			}
		}
	})
	return removed, err
}

func (tx *fakeTransaction) ListLocations(context.Context) ([]LocationRecord, error) {
	if err := tx.db.record(opListLocations); err != nil {
		return nil, err
	}
	return tx.db.locations, nil
}

// mutate applies change to the staged copy of rideID.
func (tx *fakeTransaction) mutate(rideID uuid.UUID, change func(*LoadedRide)) error {
	for i := range tx.rides {
		if tx.rides[i].ID == rideID {
			change(&tx.rides[i])
			return nil
		}
	}
	return ErrRideNotFound
}

// testUser returns a fixture profile with contact details filled in.
func testUser(id uuid.UUID, email string) UserRecord {
	return UserRecord{ID: id, FirstName: "Target", LastName: "User", Email: email, Phone: "713-555-0100"}
}

// testRide returns an active fixture ride departing at the given time, with its
// owner as the only rider.
func testRide(id, ownerID uuid.UUID, departure time.Time) LoadedRide {
	owner := testUser(ownerID, "owner@rice.edu")
	return LoadedRide{
		ID: id, DepartureTime: departure, Capacity: 3, Status: RideStatusActive,
		Owner: owner, Riders: []UserRecord{owner},
		Departure: LocationRecord{ID: uuid.New(), Title: "Rice"},
		Arrival:   LocationRecord{ID: uuid.New(), Title: "IAH"},
	}
}

// requireCalls fails t unless the service made exactly these storage calls.
// Reserve it for behaviour that *is* the ordering; prefer db.called for a
// single fact, so an unrelated new call does not fail an unrelated test.
func requireCalls(t *testing.T, db *fakeDatabase, want ...storageOp) {
	t.Helper()
	if !slices.Equal(db.calls, want) {
		t.Errorf("storage calls = %v, want %v", db.calls, want)
	}
}

// requireReadOnly fails t unless the service closed its transaction without
// publishing anything.
func requireReadOnly(t *testing.T, db *fakeDatabase) {
	t.Helper()
	if db.called(opCommit) {
		t.Error("a read committed its transaction")
	}
	if !db.called(opRollback) {
		t.Error("a read left its transaction open")
	}
}

// rideFixture is an active ride owned by owner, with one free seat and a
// passenger available to take it.
type rideFixture struct {
	fake      *fakeDatabase
	owner     uuid.UUID
	passenger uuid.UUID
	ride      LoadedRide
	input     RideWriteParams
}

func newRideFixture(t *testing.T) rideFixture {
	t.Helper()
	owner, passenger := uuid.New(), uuid.New()
	ride := testRide(uuid.New(), owner, time.Now().Add(24*time.Hour))
	ride.Capacity = 2

	return rideFixture{
		fake:      newFakeDatabase([]UserRecord{ride.Owner, testUser(passenger, "passenger@rice.edu")}, ride),
		owner:     owner,
		passenger: passenger,
		ride:      ride,
		input: RideWriteParams{
			DepartureTime:       ride.DepartureTime,
			DepartureLocationID: ride.Departure.ID,
			ArrivalLocationID:   ride.Arrival.ID,
			Capacity:            3,
			Notes:               "meet outside",
		},
	}
}

// editRide changes the stored ride before the service runs. Tests go through
// this rather than indexing db.rides, so adding a second fixture ride cannot
// silently change what a test means.
func (f rideFixture) editRide(change func(*LoadedRide)) rideFixture {
	for i := range f.fake.rides {
		if f.fake.rides[i].ID == f.ride.ID {
			change(&f.fake.rides[i])
			return f
		}
	}
	panic("fixture ride is missing")
}

// storedRide returns the ride as the fake currently holds it.
func (f rideFixture) storedRide() LoadedRide {
	for _, ride := range f.fake.rides {
		if ride.ID == f.ride.ID {
			return ride
		}
	}
	panic("fixture ride is missing")
}

// seatPassenger adds the fixture's passenger to the ride, filling its last seat.
func (f rideFixture) seatPassenger() rideFixture {
	return f.editRide(func(ride *LoadedRide) {
		ride.Riders = append(slices.Clone(ride.Riders), f.fake.users[f.passenger])
	})
}

// editOwner changes the owner's stored profile.
func (f rideFixture) editOwner(change func(*UserRecord)) rideFixture {
	owner := f.fake.users[f.owner]
	change(&owner)
	f.fake.users[f.owner] = owner
	return f
}

// rejects runs call against a fresh fixture and checks it is refused with want
// and writes nothing.
func rejects(t *testing.T, want error, setUp func(*rideFixture), call func(*Service, rideFixture) error) {
	t.Helper()
	f := newRideFixture(t)
	if setUp != nil {
		setUp(&f)
	}

	if err := call(NewService(f.fake), f); !errors.Is(err, want) {
		t.Errorf("error = %v, want %v", err, want)
	}
	if f.fake.tx != nil && f.fake.tx.committed {
		t.Error("a rejected mutation committed")
	}
}
