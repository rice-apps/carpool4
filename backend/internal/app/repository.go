package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RideStatus identifies whether a ride is active or cancelled.
type RideStatus string

const (
	RideStatusActive    RideStatus = "active"
	RideStatusCancelled RideStatus = "cancelled"
)

// UserRecord is a stored profile before contact privacy is applied.
//
// Fields:
//   - ID: user's Supabase ID and profile key.
//   - FirstName: stored given name.
//   - LastName: stored family name.
//   - Email: full email from the verified caller.
//   - Phone: full phone number, which may be blank.
type UserRecord struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

// LocationRecord is a curated starting place or destination.
//
// Fields:
//   - ID: stable location ID used by rides.
//   - Title: short name shown to users.
//   - Address: full place address.
type LocationRecord struct {
	ID      uuid.UUID
	Title   string
	Address string
}

// LoadedRide contains stored ride data before contact privacy is applied.
//
// Fields:
//   - ID: stable ride ID.
//   - DepartureDate: scheduled departure time.
//   - Owner: driver's complete stored profile.
//   - Departure: selected starting location.
//   - Arrival: selected destination.
//   - Riders: complete profiles of all occupants, including the owner.
//   - Notes: trip details supplied by the driver.
//   - Capacity: total seats, including the driver.
//   - Status: active or cancelled state.
type LoadedRide struct {
	ID            uuid.UUID
	DepartureDate time.Time
	Owner         UserRecord
	Departure     LocationRecord
	Arrival       LocationRecord
	Riders        []UserRecord
	Notes         string
	Capacity      int32
	Status        RideStatus
}

var (
	// ErrUserNotFound means no profile exists for the requested user ID.
	ErrUserNotFound = errors.New("user not found")
	// ErrRideNotFound means no ride exists for the requested ride ID.
	ErrRideNotFound = errors.New("ride not found")
	// ErrInvalidRideWrite means storage rejected invalid ride data.
	ErrInvalidRideWrite = errors.New("invalid ride write")
	// ErrTransactionConflict means concurrent work prevented a transaction from committing.
	ErrTransactionConflict = errors.New("transaction conflict")
)

// RideWriteParams contains the editable fields of a ride.
//
// Fields:
//   - DepartureDate: requested departure time.
//   - DepartureLocationID: selected starting location ID.
//   - ArrivalLocationID: selected destination ID.
//   - Notes: driver's trip details.
//   - Capacity: total seats, including the driver's seat.
type RideWriteParams struct {
	DepartureDate       time.Time
	DepartureLocationID uuid.UUID
	ArrivalLocationID   uuid.UUID
	Notes               string
	Capacity            int32
}

// UserWriteParams contains the profile values sent to storage.
//
// Fields:
//   - ID: profile key taken from the verified caller.
//   - Email: verified caller's email, rather than client-supplied text.
//   - FirstName: given name to save.
//   - LastName: family name to save.
//   - Phone: contact number to save; may be blank.
type UserWriteParams struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	LastName  string
	Phone     string
}

// ListRidesQuery contains the filters sent to ride storage.
//
// Fields:
//   - DepartureLocationID: starting place; nil accepts any location.
//   - ArrivalLocationID: destination; nil accepts any location.
//   - DepartureAfter: required earliest departure, including the default cutoff.
//   - DepartureBefore: latest departure; nil leaves the range open.
type ListRidesQuery struct {
	DepartureLocationID *uuid.UUID
	ArrivalLocationID   *uuid.UUID
	DepartureAfter      time.Time
	DepartureBefore     *time.Time
}

// Database begins the serializable transactions used for all storage access.
type Database interface {
	// BeginTx opens a transaction scoped to ctx; callers must commit or roll it back.
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction groups the repositories that share one consistent snapshot.
type Transaction interface {
	// Users returns the user repository bound to this transaction.
	Users() UserRepository
	// Rides returns the ride repository bound to this transaction.
	Rides() RideRepository
	// Locations returns the location repository bound to this transaction.
	Locations() LocationRepository
	// Commit makes all transaction writes durable or returns an error.
	Commit() error
	// Rollback closes the transaction without publishing uncommitted writes.
	Rollback() error
}

// UserRepository reads and saves profiles and checks which contacts a viewer
// may see.
type UserRepository interface {
	// FindUser loads id's profile or returns ErrUserNotFound.
	FindUser(ctx context.Context, id uuid.UUID) (UserRecord, error)
	// UpsertUser stores params and returns the complete stored profile.
	UpsertUser(ctx context.Context, params UserWriteParams) (UserRecord, error)
	// ListContactVisibleUserIDs returns targetIDs whose contacts viewerID may see.
	// Self and shared-ride peers remain visible for past or cancelled rides.
	ListContactVisibleUserIDs(ctx context.Context, viewerID uuid.UUID, targetIDs []uuid.UUID) ([]uuid.UUID, error)
}

// RideRepository reads and writes rides within a transaction.
type RideRepository interface {
	// GetRide loads rideID and its related records or returns ErrRideNotFound.
	GetRide(ctx context.Context, rideID uuid.UUID) (LoadedRide, error)
	// ListRides finds rides matching query's location and departure filters.
	ListRides(ctx context.Context, query ListRidesQuery) ([]LoadedRide, error)
	// ListMyRides finds rides owned or joined by userID.
	ListMyRides(ctx context.Context, userID uuid.UUID) ([]LoadedRide, error)
	// CreateRide stores input for ownerID and returns the new ride ID.
	CreateRide(ctx context.Context, ownerID uuid.UUID, input RideWriteParams) (uuid.UUID, error)
	// UpdateRide replaces rideID's editable fields with input.
	UpdateRide(ctx context.Context, rideID uuid.UUID, input RideWriteParams) error
	// CancelRide marks rideID as cancelled.
	CancelRide(ctx context.Context, rideID uuid.UUID) error
	// AddRideOccupant adds userID to rideID's occupants.
	AddRideOccupant(ctx context.Context, rideID, userID uuid.UUID) error
	// RemoveRideOccupant removes userID from rideID and returns the affected row count.
	RemoveRideOccupant(ctx context.Context, rideID, userID uuid.UUID) (int64, error)
}

// LocationRepository supplies the available pickup and destination locations.
type LocationRepository interface {
	// ListLocations returns all available locations.
	ListLocations(ctx context.Context) ([]LocationRecord, error)
}
