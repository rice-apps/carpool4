package rpc

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
)

// storageOp names a storage call. Tests inject failures with these rather than
// with bare strings, so a typo fails to compile.
type storageOp string

const (
	opBegin           storageOp = "BeginTx"
	opCommit          storageOp = "Commit"
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

// newTestServer wires a database into an RPC server for handler tests.
func newTestServer(database app.Database) *Server {
	return NewServer(app.NewService(database))
}

// testContext puts a verified identity into a fresh context.
func testContext(id uuid.UUID, email string) context.Context {
	return auth.WithUser(context.Background(), auth.User{ID: id, Email: email})
}

// testUser builds a stored profile with representative contact fields.
func testUser(id uuid.UUID, email string) app.UserRecord {
	return app.UserRecord{ID: id, FirstName: "Test", LastName: "User", Email: email, Phone: "713-555-0100"}
}

// fakeDatabase serves fixture records to handler tests and can fail any single
// storage call. Handlers only parse requests, call one service method and map
// the result, so the fake stores records rather than enforcing any rule.
//
// The embedded repositories are nil: a method no test configures compiles and
// panics when called, rather than failing the build.
type fakeDatabase struct {
	app.UserRepository
	app.RideRepository

	users     map[uuid.UUID]app.UserRecord
	rides     map[uuid.UUID]app.LoadedRide
	discovery []app.LoadedRide
	history   []app.LoadedRide
	locations []app.LocationRecord

	// visible reports whether viewer may see target's contacts. The zero value
	// reveals every contact, which keeps read tests focused on wire conversion.
	visible func(viewer, target uuid.UUID) bool

	failOp  storageOp
	failErr error

	calls        []storageOp
	ridesQuery   *app.ListRidesQuery
	historyOwner uuid.UUID
}

// failing returns the fake configured to fail op with err.
func (f *fakeDatabase) failing(op storageOp, err error) *fakeDatabase {
	f.failOp, f.failErr = op, err
	return f
}

// called reports whether the service made this call.
func (f *fakeDatabase) called(op storageOp) bool { return slices.Contains(f.calls, op) }

// record logs op and returns the injected failure when op is the one to fail.
func (f *fakeDatabase) record(op storageOp) error {
	f.calls = append(f.calls, op)
	if op == f.failOp {
		return f.failErr
	}
	return nil
}

func (f *fakeDatabase) BeginTx(context.Context) (app.Transaction, error) {
	if err := f.record(opBegin); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *fakeDatabase) UsersRepo() app.UserRepository         { return f }
func (f *fakeDatabase) RidesRepo() app.RideRepository         { return f }
func (f *fakeDatabase) LocationsRepo() app.LocationRepository { return f }

// Commit succeeds so that mutation handlers can be tested on their happy path.
func (f *fakeDatabase) Commit() error { return f.record(opCommit) }

func (f *fakeDatabase) Rollback() error { return nil }

func (f *fakeDatabase) FindUser(_ context.Context, id uuid.UUID) (app.UserRecord, error) {
	if err := f.record(opFindUser); err != nil {
		return app.UserRecord{}, err
	}
	user, ok := f.users[id]
	if !ok {
		return app.UserRecord{}, app.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeDatabase) UpsertUser(_ context.Context, params app.UserWriteParams) (app.UserRecord, error) {
	if err := f.record(opUpsertUser); err != nil {
		return app.UserRecord{}, err
	}
	return app.UserRecord{
		ID: params.ID, FirstName: params.FirstName, LastName: params.LastName,
		Email: params.Email, Phone: params.Phone,
	}, nil
}

func (f *fakeDatabase) ListContactVisibleUserIDs(_ context.Context, viewerID uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.record(opVisibleContacts); err != nil {
		return nil, err
	}
	if f.visible == nil {
		return targets, nil
	}
	allowed := make([]uuid.UUID, 0, len(targets))
	for _, target := range targets {
		if f.visible(viewerID, target) {
			allowed = append(allowed, target)
		}
	}
	return allowed, nil
}

func (f *fakeDatabase) GetRide(_ context.Context, id uuid.UUID) (app.LoadedRide, error) {
	if err := f.record(opGetRide); err != nil {
		return app.LoadedRide{}, err
	}
	ride, ok := f.rides[id]
	if !ok {
		return app.LoadedRide{}, app.ErrRideNotFound
	}
	return ride, nil
}

func (f *fakeDatabase) ListRides(_ context.Context, query app.ListRidesQuery) ([]app.LoadedRide, error) {
	if err := f.record(opListRides); err != nil {
		return nil, err
	}
	f.ridesQuery = &query
	return f.discovery, nil
}

func (f *fakeDatabase) ListMyRides(_ context.Context, viewerID uuid.UUID) ([]app.LoadedRide, error) {
	if err := f.record(opListMyRides); err != nil {
		return nil, err
	}
	f.historyOwner = viewerID
	return f.history, nil
}

func (f *fakeDatabase) ListLocations(context.Context) ([]app.LocationRecord, error) {
	if err := f.record(opListLocations); err != nil {
		return nil, err
	}
	return f.locations, nil
}

func (f *fakeDatabase) CreateRide(_ context.Context, ownerID uuid.UUID, input app.RideWriteParams) (uuid.UUID, error) {
	if err := f.record(opCreateRide); err != nil {
		return uuid.Nil, err
	}
	id := uuid.New()
	if f.rides == nil {
		f.rides = map[uuid.UUID]app.LoadedRide{}
	}
	f.rides[id] = app.LoadedRide{
		ID: id, Owner: f.users[ownerID], DepartureTime: input.DepartureTime,
		Departure: app.LocationRecord{ID: input.DepartureLocationID},
		Arrival:   app.LocationRecord{ID: input.ArrivalLocationID},
		Notes:     input.Notes, Capacity: input.Capacity, Status: app.RideStatusActive,
	}
	return id, nil
}

func (f *fakeDatabase) UpdateRide(_ context.Context, id uuid.UUID, input app.RideWriteParams) error {
	if err := f.record(opUpdateRide); err != nil {
		return err
	}
	return f.mutate(id, func(ride *app.LoadedRide) {
		ride.DepartureTime, ride.Notes, ride.Capacity = input.DepartureTime, input.Notes, input.Capacity
	})
}

func (f *fakeDatabase) CancelRide(_ context.Context, id uuid.UUID) error {
	if err := f.record(opCancelRide); err != nil {
		return err
	}
	return f.mutate(id, func(ride *app.LoadedRide) { ride.Status = app.RideStatusCancelled })
}

func (f *fakeDatabase) AddRideOccupant(_ context.Context, rideID, userID uuid.UUID) error {
	if err := f.record(opAddOccupant); err != nil {
		return err
	}
	return f.mutate(rideID, func(ride *app.LoadedRide) {
		ride.Riders = append(slices.Clone(ride.Riders), f.users[userID])
	})
}

func (f *fakeDatabase) RemoveRideOccupant(_ context.Context, rideID, userID uuid.UUID) (int64, error) {
	if err := f.record(opRemoveOccupant); err != nil {
		return 0, err
	}
	var removed int64
	err := f.mutate(rideID, func(ride *app.LoadedRide) {
		for i, rider := range ride.Riders {
			if rider.ID == userID {
				ride.Riders = slices.Delete(slices.Clone(ride.Riders), i, i+1)
				removed = 1
				return
			}
		}
	})
	return removed, err
}

// mutate applies change to the stored ride.
func (f *fakeDatabase) mutate(id uuid.UUID, change func(*app.LoadedRide)) error {
	ride, ok := f.rides[id]
	if !ok {
		return app.ErrRideNotFound
	}
	change(&ride)
	f.rides[id] = ride
	return nil
}
