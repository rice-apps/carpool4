package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
)

// mapUser converts a database user row, including contact fields,
// into an application record without applying viewer privacy.
func mapUser(user sqlc.User) app.UserRecord {
	return app.UserRecord{
		ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, Phone: user.Phone,
	}
}

// mapLocation converts a database location row to the application shape.
func mapLocation(location sqlc.Location) app.LocationRecord {
	return app.LocationRecord{ID: location.ID, Title: location.Title, Address: location.Address}
}

// mapLoadedRide returns an application ride from SQL rows.
//
// Inputs:
//   - ride (sqlc.Ride): the ride row.
//   - owner (sqlc.User): the owner's user row.
//   - departure, arrival (sqlc.Location): route location rows.
//   - ridersJSON (json.RawMessage): array of rider user rows.
//
// Invalid rider JSON returns an error. Contact visibility is applied by the
// application service.
func mapLoadedRide(ride sqlc.Ride, owner sqlc.User, departure, arrival sqlc.Location, ridersJSON json.RawMessage) (app.LoadedRide, error) {
	// The ride query returns its rider rows as a JSON array.
	var riders []sqlc.User
	if err := json.Unmarshal(ridersJSON, &riders); err != nil {
		return app.LoadedRide{}, fmt.Errorf("decode riders: %w", err)
	}
	// Convert database rows into records the application can project safely.
	mappedRiders := make([]app.UserRecord, len(riders))
	for i, rider := range riders {
		mappedRiders[i] = mapUser(rider)
	}
	return app.LoadedRide{
		ID: ride.ID, DepartureDate: ride.DepartureDate,
		Owner: mapUser(owner), Departure: mapLocation(departure), Arrival: mapLocation(arrival), Riders: mappedRiders,
		Notes: ride.Notes, Capacity: ride.Capacity, Status: app.RideStatus(ride.Status),
	}, nil
}
