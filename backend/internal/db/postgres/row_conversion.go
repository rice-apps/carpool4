package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
)

// convertUserRow converts a database user row, including contact fields,
// into an application record without applying viewer privacy.
func convertUserRow(user sqlc.User) app.UserRecord {
	return app.UserRecord{
		ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, Phone: user.Phone,
	}
}

// convertLocationRow converts a database location row to the application shape.
func convertLocationRow(location sqlc.Location) app.LocationRecord {
	return app.LocationRecord{ID: location.ID, Title: location.Title, Address: location.Address}
}

// convertRideRows returns an application ride from SQL rows.
//
// Inputs:
//   - ride (sqlc.Ride): the ride row.
//   - owner (sqlc.User): the owner's user row.
//   - departure, arrival (sqlc.Location): route location rows.
//   - ridersJSON (json.RawMessage): array of rider user rows.
//
// Invalid rider JSON returns an error. Contact visibility is applied by the
// application service.
func convertRideRows(ride sqlc.Ride, owner sqlc.User, departure, arrival sqlc.Location, ridersJSON json.RawMessage) (app.LoadedRide, error) {
	// The ride query returns its rider rows as a JSON array.
	var riders []sqlc.User
	if err := json.Unmarshal(ridersJSON, &riders); err != nil {
		return app.LoadedRide{}, fmt.Errorf("decode riders: %w", err)
	}
	// Build stored records; the app later selects fields allowed for the viewer.
	mappedRiders := make([]app.UserRecord, len(riders))
	for i, rider := range riders {
		mappedRiders[i] = convertUserRow(rider)
	}
	return app.LoadedRide{
		ID: ride.ID, DepartureTime: ride.DepartureTime,
		Owner: convertUserRow(owner), Departure: convertLocationRow(departure), Arrival: convertLocationRow(arrival), Riders: mappedRiders,
		Notes: ride.Notes, Capacity: ride.Capacity, Status: app.RideStatus(ride.Status),
	}, nil
}
