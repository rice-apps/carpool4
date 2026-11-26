package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// Every test in this package shares one PostgreSQL container and truncates the
// fixture tables between tests, so none of them calls t.Parallel.
func TestMain(m *testing.M) { os.Exit(dbtest.Run(m)) }

// begin opens a transaction that rolls back when the test ends. Each test and
// subtest takes its own, so one can be run in isolation and mean the same thing.
func begin(t *testing.T, db *sql.DB) app.Transaction {
	t.Helper()
	tx, err := NewDatabase(db).BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	return tx
}

// insertUsers creates count matching auth identities and profiles.
func insertUsers(t *testing.T, db *sql.DB, count int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, count)
	for i := range ids {
		ids[i] = uuid.New()
		if _, err := db.Exec(`INSERT INTO auth.users (id) VALUES ($1)`, ids[i]); err != nil {
			t.Fatalf("insert auth identity: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO users (id, first_name, last_name, email) VALUES ($1, 'Test', 'User', $2)`,
			ids[i], ids[i].String()+"@rice.edu"); err != nil {
			t.Fatalf("insert profile: %v", err)
		}
	}
	return ids
}

// insertLocations creates the two locations a ride needs and returns them in
// title order, which is the order ListLocations must produce.
func insertLocations(t *testing.T, db *sql.DB) (departure, arrival uuid.UUID) {
	t.Helper()
	departure, arrival = uuid.New(), uuid.New()
	if _, err := db.Exec(`INSERT INTO locations (id, title, address) VALUES ($1, 'Arrival', $3), ($2, 'Departure', $4)`,
		arrival, departure, arrival.String(), departure.String()); err != nil {
		t.Fatalf("insert locations: %v", err)
	}
	return departure, arrival
}

// insertRide creates an owner, two locations and an active ride the owner
// occupies, returning the ride and the parameters that produced it.
func insertRide(t *testing.T, db *sql.DB) (uuid.UUID, uuid.UUID, app.RideWriteParams) {
	t.Helper()
	owner := insertUsers(t, db, 1)[0]
	departure, arrival := insertLocations(t, db)
	rideID := uuid.New()
	params := app.RideWriteParams{
		DepartureTime:       time.Now().Add(24 * time.Hour),
		DepartureLocationID: departure,
		ArrivalLocationID:   arrival,
		Capacity:            3,
	}
	if _, err := db.Exec(
		`INSERT INTO rides (id, departure_time, departure_location_id, arrival_location_id, owner_id, capacity)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		rideID, params.DepartureTime, departure, arrival, owner, params.Capacity); err != nil {
		t.Fatalf("insert ride: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO ride_occupants (ride_id, user_id) VALUES ($1, $2)`, rideID, owner); err != nil {
		t.Fatalf("seat owner: %v", err)
	}
	return rideID, owner, params
}

// requireSerializationConflict fails t unless err carries both the application
// sentinel callers match on and the PostgreSQL cause for logs.
func requireSerializationConflict(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, app.ErrTransactionConflict) {
		t.Errorf("error = %v, want it to wrap ErrTransactionConflict", err)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want it to carry a *pgconn.PgError", err)
	}
	if pgErr.Code != "40001" {
		t.Errorf("SQLSTATE = %q, want a serialization failure (40001)", pgErr.Code)
	}
}
