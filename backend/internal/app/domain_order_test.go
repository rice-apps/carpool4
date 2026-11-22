package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// TestRideWritesAuthorizeBeforeBusinessInputValidation checks that missing phone
// and ownership errors take precedence over invalid ride fields.
func TestRideWritesAuthorizeBeforeBusinessInputValidation(t *testing.T) {
	owner := uuid.New()

	t.Run("create checks profile first", func(t *testing.T) {
		tx := &orderTx{user: UserRecord{ID: owner}}
		service := NewService(orderStore{tx: tx})
		_, err := service.CreateRide(context.Background(), CurrentUser{ID: owner}, RideWriteParams{})
		if !errors.Is(err, ErrPhoneRequired) {
			t.Fatalf("CreateRide error = %v, want phone prerequisite", err)
		}
	})

	t.Run("update checks ownership first", func(t *testing.T) {
		tx := &orderTx{ride: LoadedRide{ID: uuid.New(), Owner: UserRecord{ID: uuid.New()}, Status: RideStatusActive}}
		service := NewService(orderStore{tx: tx})
		_, err := service.UpdateRide(context.Background(), CurrentUser{ID: owner}, tx.ride.ID, RideWriteParams{})
		if !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("UpdateRide error = %v, want permission denied", err)
		}
	})
}

// orderStore supplies the transaction used by validation-order tests.
//
// Fields:
//   - tx: fixture transaction returned to the service.
type orderStore struct{ tx Transaction }

// BeginTx returns the fixture transaction without opening real storage.
func (s orderStore) BeginTx(context.Context) (Transaction, error) { return s.tx, nil }

// orderTx holds values needed to test which rejection happens first.
//
// Fields:
//   - user: profile returned by user lookup.
//   - ride: ride returned by ride lookup.
type orderTx struct {
	user UserRecord
	ride LoadedRide
}

// UsersRepo exposes this fixture as the user repository.
func (tx *orderTx) UsersRepo() UserRepository { return tx }

// RidesRepo exposes this fixture as the ride repository.
func (tx *orderTx) RidesRepo() RideRepository { return tx }

// LocationsRepo exposes this fixture as the location repository.
func (tx *orderTx) LocationsRepo() LocationRepository { return tx }

// FindUser returns the fixture user, ignoring the requested ID.
func (tx *orderTx) FindUser(context.Context, uuid.UUID) (UserRecord, error) { return tx.user, nil }

// Rollback closes the fixture transaction without a storage side effect.
func (*orderTx) Rollback() error { return nil }

// GetRide returns the fixture ride, ignoring the requested ID.
func (tx *orderTx) GetRide(context.Context, uuid.UUID) (LoadedRide, error) {
	return tx.ride, nil
}

// ListContactVisibleUserIDs panics because these cases must reject before projection.
func (*orderTx) ListContactVisibleUserIDs(context.Context, uuid.UUID, []uuid.UUID) ([]uuid.UUID, error) {
	panic("unexpected call")
}

// ListRides panics if validation-order tests unexpectedly query ride discovery.
func (*orderTx) ListRides(context.Context, ListRidesQuery) ([]LoadedRide, error) {
	panic("unexpected call")
}

// ListMyRides panics if these tests unexpectedly query a user's rides.
func (*orderTx) ListMyRides(context.Context, uuid.UUID) ([]LoadedRide, error) {
	panic("unexpected call")
}

// ListLocations panics if these tests unexpectedly query locations.
func (*orderTx) ListLocations(context.Context) ([]LocationRecord, error) {
	panic("unexpected call")
}

// CreateRide panics if validation-order tests reach ride creation.
func (*orderTx) CreateRide(context.Context, uuid.UUID, RideWriteParams) (uuid.UUID, error) {
	panic("unexpected call")
}

// UpsertUser panics if validation-order tests unexpectedly write a profile.
func (*orderTx) UpsertUser(context.Context, UserWriteParams) (UserRecord, error) {
	panic("unexpected call")
}

// UpdateRide panics if the ownership check fails to stop the write.
func (*orderTx) UpdateRide(context.Context, uuid.UUID, RideWriteParams) error {
	panic("unexpected call")
}

// CancelRide panics if these tests unexpectedly cancel a ride.
func (*orderTx) CancelRide(context.Context, uuid.UUID) error { panic("unexpected call") }

// AddRideOccupant panics if these tests unexpectedly change membership.
func (*orderTx) AddRideOccupant(context.Context, uuid.UUID, uuid.UUID) error {
	panic("unexpected call")
}

// RemoveRideOccupant panics if these tests unexpectedly change membership.
func (*orderTx) RemoveRideOccupant(context.Context, uuid.UUID, uuid.UUID) (int64, error) {
	panic("unexpected call")
}

// Commit panics because rejected writes must not commit.
func (*orderTx) Commit() error { panic("unexpected call") }
