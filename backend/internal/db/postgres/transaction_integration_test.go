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

// TestMain lets the shared test harness release its PostgreSQL container after
// the package tests complete and preserves their exit status.
func TestMain(m *testing.M) { os.Exit(dbtest.Run(m)) }

// TestBeginTxUsesSerializableIsolation checks the transaction isolation seen
// by PostgreSQL, which protects application read-check-write decisions.
func TestBeginTxUsesSerializableIsolation(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	tx, err := NewDatabase(db).BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	postgresTx, ok := tx.(*Transaction)
	if !ok {
		t.Fatalf("BeginTx returned %T, want *Transaction", tx)
	}
	var isolation string
	if err := postgresTx.tx.QueryRowContext(ctx, "SHOW transaction_isolation").Scan(&isolation); err != nil {
		t.Fatal(err)
	}
	if isolation != "serializable" {
		t.Fatalf("transaction isolation = %q, want serializable", isolation)
	}
}

// TestTransactionWritesAreVisibleThenRollbackTogether checks read-your-writes
// inside one transaction and the absence of those writes after rollback.
func TestTransactionWritesAreVisibleThenRollbackTogether(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	id := uuid.New()
	if _, err := db.ExecContext(ctx, `INSERT INTO auth.users (id) VALUES ($1)`, id); err != nil {
		t.Fatal(err)
	}

	database := NewDatabase(db)
	tx, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Write and read the profile within the same transaction.
	record, err := tx.UsersRepo().UpsertUser(ctx, app.UserWriteParams{
		ID: id, Email: "user@rice.edu", FirstName: "Test", LastName: "User",
	})
	if err != nil {
		t.Fatal(err)
	}
	if found, err := tx.UsersRepo().FindUser(ctx, id); err != nil || found.ID != record.ID {
		t.Fatalf("transaction did not see its own write: user=%#v error=%v", found, err)
	}
	// Start a fresh transaction to confirm the rollback removed the write.
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	readTx, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	if _, err := readTx.UsersRepo().FindUser(ctx, id); !errors.Is(err, app.ErrUserNotFound) {
		t.Fatalf("rolled-back user lookup error = %v, want ErrUserNotFound", err)
	}
}

// TestTransactionQueryClassifiesSerializationFailure creates overlapping ride
// updates and checks a statement-level serialization abort is classified.
func TestTransactionQueryClassifiesSerializationFailure(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, params := insertTransactionRide(t, db)
	database := NewDatabase(db)
	first, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback()
	second, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback()

	// Both transactions read the same ride before either writes.
	if _, err := first.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatal(err)
	}
	if _, err := second.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatal(err)
	}
	params.Notes = "first update"
	if err := first.RidesRepo().UpdateRide(ctx, rideID, params); err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	// The stale second transaction should fail on its update statement.
	params.Notes = "second update"
	err = second.RidesRepo().UpdateRide(ctx, rideID, params)
	assertSerializationConflict(t, err)
}

// TestTransactionCommitClassifiesSerializationFailure creates overlapping
// membership writes and checks the conflict surfaced at commit.
func TestTransactionCommitClassifiesSerializationFailure(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, _ := insertTransactionRide(t, db)
	users := insertTransactionUsers(t, db, 2)
	firstUser, secondUser := users[0], users[1]
	database := NewDatabase(db)
	first, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback()
	second, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback()

	// Both transactions read the same ride before changing membership.
	if _, err := first.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatal(err)
	}
	if _, err := second.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatal(err)
	}
	if err := first.RidesRepo().AddRideOccupant(ctx, rideID, firstUser); err != nil {
		t.Fatal(err)
	}
	if err := second.RidesRepo().AddRideOccupant(ctx, rideID, secondUser); err != nil {
		t.Fatal(err)
	}

	// Only one overlapping transaction can commit successfully.
	errFirst := first.Commit()
	errSecond := second.Commit()
	if (errFirst == nil) == (errSecond == nil) {
		t.Fatalf("commit errors = (%v, %v), want one success and one conflict", errFirst, errSecond)
	}
	if errFirst != nil {
		assertSerializationConflict(t, errFirst)
	} else {
		assertSerializationConflict(t, errSecond)
	}
}

// TestRideWritesPreserveMissingRideErrors checks update and cancel retain both
// the application not-found sentinel and the SQL no-rows cause.
func TestRideWritesPreserveMissingRideErrors(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	_, params := insertTransactionRide(t, db)
	tx, err := NewDatabase(db).BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	missing := uuid.New()
	for name, err := range map[string]error{
		"update": tx.RidesRepo().UpdateRide(ctx, missing, params),
		"cancel": tx.RidesRepo().CancelRide(ctx, missing),
	} {
		if !errors.Is(err, app.ErrRideNotFound) || !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("%s missing ride = %v, want ErrRideNotFound retaining ErrNoRows", name, err)
		}
	}
}

// insertTransactionRide creates an owner, locations, and a ride in a disposable
// test database, returning its ID and writable fields for update scenarios.
func insertTransactionRide(t *testing.T, db *sql.DB) (uuid.UUID, app.RideWriteParams) {
	t.Helper()
	ctx := context.Background()
	owner := insertTransactionUsers(t, db, 1)[0]
	departure, arrival := uuid.New(), uuid.New()
	if _, err := db.ExecContext(ctx, `INSERT INTO locations (id, title, address) VALUES ($1, 'Departure', $3), ($2, 'Arrival', $4)`, departure, arrival, departure.String(), arrival.String()); err != nil {
		t.Fatal(err)
	}
	rideID := uuid.New()
	params := app.RideWriteParams{DepartureTime: time.Now().Add(24 * time.Hour), DepartureLocationID: departure, ArrivalLocationID: arrival, Capacity: 3}
	if _, err := db.ExecContext(ctx, `INSERT INTO rides (id, departure_time, departure_location_id, arrival_location_id, owner_id, capacity) VALUES ($1, $2, $3, $4, $5, $6)`, rideID, params.DepartureTime, departure, arrival, owner, params.Capacity); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ride_occupants (ride_id, user_id) VALUES ($1, $2)`, rideID, owner); err != nil {
		t.Fatal(err)
	}
	return rideID, params
}

// insertTransactionUsers creates count matching Auth and profile rows and
// returns their IDs for membership scenarios; setup failures fail the test.
func insertTransactionUsers(t *testing.T, db *sql.DB, count int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, count)
	for i := range ids {
		ids[i] = uuid.New()
		if _, err := db.Exec(`INSERT INTO auth.users (id) VALUES ($1)`, ids[i]); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO users (id, first_name, last_name, email) VALUES ($1, 'Test', 'User', $2)`, ids[i], ids[i].String()+"@rice.edu"); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

// assertSerializationConflict fails t unless err exposes both the application
// conflict sentinel and the original PostgreSQL 40001 cause.
func assertSerializationConflict(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, app.ErrTransactionConflict) {
		t.Fatalf("error = %v, want ErrTransactionConflict", err)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "40001" {
		t.Fatalf("error lost PostgreSQL serialization cause: %v", err)
	}
}
