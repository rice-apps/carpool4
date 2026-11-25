package rpc

import (
	"context"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
)

// newTestServer wires a supplied application database into an RPC server for unit tests.
func newTestServer(database app.Database) *Server {
	return NewServer(app.NewService(database))
}

// testContext puts the specified verified UUID and email into a fresh context.
func testContext(id uuid.UUID, email string) context.Context {
	return auth.WithUser(context.Background(), auth.User{ID: id, Email: email})
}

// testUser builds a stored user with the given identity and representative contact fields.
func testUser(id uuid.UUID, email string) app.UserRecord {
	return app.UserRecord{ID: id, FirstName: "Test", LastName: "User", Email: email, Phone: "713-555-0100"}
}

// fakeStore implements transaction repositories with in-memory records for read-path tests.
//
// Fields:
//   - UserRepository: fallback user repository methods not defined on the fake.
//   - RideRepository: fallback ride repository methods not defined on the fake.
//   - users: user records available for lookup.
//   - findErr: optional failure injected into user lookup.
//   - loadErr: optional failure injected into ride loading.
//   - rides: ride records available for lookup.
//   - listed: rides returned by list operations.
//   - locations: locations returned by list operations.
//   - locationsErr: optional failure injected into the location list.
//   - query: last ride-discovery filters received.
type fakeStore struct {
	app.UserRepository
	app.RideRepository
	users        map[uuid.UUID]app.UserRecord
	findErr      error
	loadErr      error
	rides        map[uuid.UUID]app.LoadedRide
	listed       []app.LoadedRide
	locations    []app.LocationRecord
	locationsErr error
	query        *app.ListRidesQuery
}

// GetRide returns the requested fixture ride, an injected failure, or ErrRideNotFound.
func (s *fakeStore) GetRide(_ context.Context, id uuid.UUID) (app.LoadedRide, error) {
	if s.loadErr != nil {
		return app.LoadedRide{}, s.loadErr
	}
	ride, ok := s.rides[id]
	if !ok {
		return app.LoadedRide{}, app.ErrRideNotFound
	}
	return ride, nil
}

// ListContactVisibleUserIDs treats all target UUIDs as visible in read-path unit tests.
func (*fakeStore) ListContactVisibleUserIDs(_ context.Context, _ uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	return targets, nil
}

// FindUser returns the requested fixture user, an injected failure, or ErrUserNotFound.
func (s *fakeStore) FindUser(_ context.Context, id uuid.UUID) (app.UserRecord, error) {
	if s.findErr != nil {
		return app.UserRecord{}, s.findErr
	}
	user, ok := s.users[id]
	if !ok {
		return app.UserRecord{}, app.ErrUserNotFound
	}
	return user, nil
}

// ListLocations returns the fixture locations without database access.
func (s *fakeStore) ListLocations(context.Context) ([]app.LocationRecord, error) {
	return s.locations, s.locationsErr
}

// ListRides returns fixture rides; SQL filtering is covered by integration tests.
func (s *fakeStore) ListRides(_ context.Context, query app.ListRidesQuery) ([]app.LoadedRide, error) {
	s.query = &query
	return s.listed, nil
}

// ListMyRides returns fixture rides for read-path adapter tests.
func (s *fakeStore) ListMyRides(context.Context, uuid.UUID) ([]app.LoadedRide, error) {
	return s.listed, nil
}

// BeginTx reuses the fake as a transaction without persistence.
func (s *fakeStore) BeginTx(context.Context) (app.Transaction, error) {
	return s, nil
}

// UsersRepo exposes the fake user repository for the transaction.
func (s *fakeStore) UsersRepo() app.UserRepository { return s }

// RidesRepo exposes the fake ride repository for the transaction.
func (s *fakeStore) RidesRepo() app.RideRepository { return s }

// LocationsRepo exposes the fake location repository for the transaction.
func (s *fakeStore) LocationsRepo() app.LocationRepository { return s }

// Commit panics because read-path tests should never commit a mutation.
func (*fakeStore) Commit() error { panic("unexpected commit") }

// Rollback completes a read-only fake transaction without side effects.
func (*fakeStore) Rollback() error { return nil }
