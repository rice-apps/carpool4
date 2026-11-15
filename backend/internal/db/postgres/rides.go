package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"

	"github.com/rice-apps/carpool4/backend/internal/app"
)

// GetRide returns a ride with its owner, locations, and riders.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database query.
//   - id (uuid.UUID): the ride to load.
//
// A missing ride returns app.ErrRideNotFound; invalid rider data and other
// storage failures return errors.
func (r rideRepository) GetRide(ctx context.Context, id uuid.UUID) (app.LoadedRide, error) {
	// TODO(student): Implement GetRide Postgres repository method.
	return app.LoadedRide{}, app.ErrNotImplemented
}

// CreateRide stores a new ride.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database write.
//   - ownerID (uuid.UUID): the ride owner.
//   - params (app.RideWriteParams): writable ride fields.
//
// It returns the new ride UUID or a storage error. Invalid database constraints
// map to app.ErrInvalidRideWrite.
func (r rideRepository) CreateRide(ctx context.Context, ownerID uuid.UUID, params app.RideWriteParams) (uuid.UUID, error) {

	id, err := r.queries.CreateRide(ctx, sqlc.CreateRideParams{
		DepartureTime:       params.DepartureTime,
		DepartureLocationID: params.DepartureLocationID,
		ArrivalLocationID:   params.ArrivalLocationID,
		OwnerID:             ownerID,
		Notes:               params.Notes,
		Capacity:            params.Capacity,
		Status:              "active",
	})
	if err != nil {
		return uuid.Nil, translateRideWriteError(err)
	}
	return id, nil
}

// UpdateRide replaces a ride's writable fields.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database write.
//   - id (uuid.UUID): the ride to update.
//   - params (app.RideWriteParams): replacement ride fields.
//
// It returns an error on failure, including app.ErrRideNotFound for a missing
// ride or app.ErrInvalidRideWrite for invalid fields.
func (r rideRepository) UpdateRide(ctx context.Context, id uuid.UUID, params app.RideWriteParams) error {
	// TODO(student): Implement UpdateRide Postgres repository method.

	return app.ErrNotImplemented
}

// CancelRide marks a ride as canceled.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database write.
//   - id (uuid.UUID): the ride to cancel.
//
// It returns an error on failure, including app.ErrRideNotFound for a missing
// ride.
func (r rideRepository) CancelRide(ctx context.Context, id uuid.UUID) error {
	// TODO(student): Implement CancelRide Postgres repository method.
	// call the CancelRide method from sqlc
	// r= ride repository, encapsulates the functions of the storage layer
	numbOfRows, err := r.queries.CancelRide(ctx, id)
	if err != nil {
		return translateRideError(err)
	}
	if numbOfRows == 0 {
		return translateRideError(sql.ErrNoRows)
	}
	return app.ErrRideCancelled
}

// AddRideOccupant adds a user to a ride.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database write.
//   - rideID (uuid.UUID): the ride.
//   - userID (uuid.UUID): the user to add.
//
// It returns an error if the write fails.
func (r rideRepository) AddRideOccupant(ctx context.Context, rideID, userID uuid.UUID) error {
	// TODO(student): Implement AddRideOccupant Postgres repository method.
	// Create an 'arg' variable that stores the rideID and userID as a AddRideOccupantParams struct
	arg := sqlc.AddRideOccupantParams{RideID: rideID, UserID: userID}
	// Call the generated sql go function 'AddRideOccupant'
	err := r.queries.AddRideOccupant(ctx, arg)
	// Check if there is an error. If so, return the error that recognizes database transaction conflicts and returns an error identifiable as app.ErrTransactionConflict.
	if err != nil {
		return translateTransactionError(err)
	}
	// Return no error
	return err
}

// RemoveRideOccupant removes a user from a ride.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database write.
//   - rideID (uuid.UUID): the ride.
//   - userID (uuid.UUID): the user to remove.
//
// It returns the number of affected rows or an error. Zero rows means the
// membership was absent.
func (r rideRepository) RemoveRideOccupant(ctx context.Context, rideID, userID uuid.UUID) (int64, error) {
	// TODO(student): Implement RemoveRideOccupant Postgres repository method.
	return 0, app.ErrNotImplemented
}

// ListRides returns rides matching a discovery query, including related users
// and locations.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database query.
//   - query (app.ListRidesQuery): time bounds and optional location filters.
//
// It returns the matching rides or a query or mapping error.
func (r rideRepository) ListRides(ctx context.Context, query app.ListRidesQuery) ([]app.LoadedRide, error) {
	// TODO(student): Implement ListRides Postgres repository method.
	return nil, app.ErrNotImplemented
}

// ListMyRides returns rides owned or joined by viewerID, including history.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database query.
//   - viewerID (uuid.UUID): the user whose rides are requested.
//
// It returns loaded rides or an error. The application controls contact
// visibility in the returned records.
func (r rideRepository) ListMyRides(ctx context.Context, viewerID uuid.UUID) ([]app.LoadedRide, error) {
	// TODO(student): Implement ListMyRides Postgres repository method.
	myRidesQuery, err := r.queries.ListMyRides(ctx, viewerID) // queries all locations
	if err != nil {
		return nil, translateTransactionError(err)
	}

	//converts locations to LocationRecords
	myRides := make([]app.LoadedRide, len(myRidesQuery))
	for i, row := range myRidesQuery {
		myRides[i], err = convertRideRows(row.Ride, row.User, row.Location, row.Location_2, row.Riders)
		if err != nil {
			return nil, translateRideError(err)
		}
	}
	return myRides, nil
}
