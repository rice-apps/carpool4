package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
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
	ride, err := r.queries.GetRide(ctx, id)
	if err != nil {
		return app.LoadedRide{}, translateRideError(err)
	}
	return convertRideRows(ride.Ride, ride.User, ride.Location, ride.Location_2, ride.Riders)
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
	rideChanges := sqlc.UpdateRideParams{ // changes type by creating a new instance rideChanges of the UpdateRideParams struct
		DepartureTime:       params.DepartureTime,
		DepartureLocationID: params.DepartureLocationID,
		ArrivalLocationID:   params.ArrivalLocationID,
		Notes:               params.Notes,
		Capacity:            params.Capacity,
		ID:                  id,
	}
	rowsChanged, err := r.queries.UpdateRide(ctx, rideChanges) // checks for rows changed to be 0; returns error since that means ride was not found/something happened to stop rows from being updated
	if err != nil {
		return translateRideWriteError(err)
	}

	if rowsChanged == 0 {
		return app.ErrRideNotFound
	}

	return nil
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
	numbOfRows, err := r.queries.RemoveRideOccupant(ctx, sqlc.RemoveRideOccupantParams{RideID: rideID, UserID: userID})
	if err != nil {
		return numbOfRows, translateTransactionError(err)
	}
	return numbOfRows, nil
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
	params := sqlc.ListRidesParams{}
	if query.ArrivalLocationID != nil {
		params.ArrivalLocationID = uuid.NullUUID{UUID: *query.ArrivalLocationID, Valid: true}
	}
	if query.DepartureLocationID != nil {
		params.DepartureLocationID = uuid.NullUUID{UUID: *query.DepartureLocationID, Valid: true}
	}
	if query.DepartureBefore != nil {
		params.DepartureBefore = sql.NullTime{Time: *query.DepartureBefore, Valid: true}
	}
	params.DepartureAfter = query.DepartureAfter

	// fill out rest of params
	rides, err := r.queries.ListRides(ctx, params)
	if err != nil {
		// translateTransactionError
		return nil, translateTransactionError(err)
	}
	// loop over returned rides, use convertRideRows?
	returnedRides := make([]app.LoadedRide, len(rides))
	for i, ride := range rides {
		returnedRides[i], err = convertRideRows(
			ride.Ride, ride.User, ride.Location, ride.Location_2, ride.Riders)
		if err != nil {
			return nil, fmt.Errorf("error on ride conversion: %w", err)
		}
	}
	// return
	return returnedRides, nil
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
