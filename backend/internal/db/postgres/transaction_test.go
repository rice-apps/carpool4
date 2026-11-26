package postgres

import (
	"context"
	"testing"

	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// TestBeginTxUsesSerializableIsolation checks the isolation level PostgreSQL
// actually applies. The application reads a ride, decides, then writes, and
// only serializable isolation makes that sequence safe under concurrency.
func TestBeginTxUsesSerializableIsolation(t *testing.T) {
	db := dbtest.DB(t)
	tx := begin(t, db)

	postgresTx, ok := tx.(*Transaction)
	if !ok {
		t.Fatalf("BeginTx() returned %T, want *Transaction", tx)
	}
	var isolation string
	if err := postgresTx.tx.QueryRowContext(context.Background(), "SHOW transaction_isolation").Scan(&isolation); err != nil {
		t.Fatalf("read isolation level: %v", err)
	}

	if isolation != "serializable" {
		t.Errorf("transaction isolation = %q, want %q", isolation, "serializable")
	}
}

// TestConflictingUpdateFailsAtTheStatement checks that two transactions which
// read the same ride and then both write it produce a classified conflict on
// the losing statement, so the application can rerun the whole use case.
func TestConflictingUpdateFailsAtTheStatement(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, _, params := insertRide(t, db)
	first, second := begin(t, db), begin(t, db)

	// Both read before either writes, which is what makes the writes conflict.
	if _, err := first.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatalf("first GetRide() error = %v", err)
	}
	if _, err := second.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatalf("second GetRide() error = %v", err)
	}

	params.Notes = "first update"
	if err := first.RidesRepo().UpdateRide(ctx, rideID, params); err != nil {
		t.Fatalf("first UpdateRide() error = %v", err)
	}
	if err := first.Commit(); err != nil {
		t.Fatalf("first Commit() error = %v", err)
	}

	params.Notes = "second update"
	requireSerializationConflict(t, second.RidesRepo().UpdateRide(ctx, rideID, params))
}

// TestConflictingMembershipFailsAtCommit covers the other place a conflict can
// surface. Commit is the path the application's retry contract depends on, and
// it classifies errors separately from the statement path above.
func TestConflictingMembershipFailsAtCommit(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, _, _ := insertRide(t, db)
	riders := insertUsers(t, db, 2)
	first, second := begin(t, db), begin(t, db)

	if _, err := first.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatalf("first GetRide() error = %v", err)
	}
	if _, err := second.RidesRepo().GetRide(ctx, rideID); err != nil {
		t.Fatalf("second GetRide() error = %v", err)
	}
	if err := first.RidesRepo().AddRideOccupant(ctx, rideID, riders[0]); err != nil {
		t.Fatalf("first AddRideOccupant() error = %v", err)
	}
	if err := second.RidesRepo().AddRideOccupant(ctx, rideID, riders[1]); err != nil {
		t.Fatalf("second AddRideOccupant() error = %v", err)
	}

	firstErr, secondErr := first.Commit(), second.Commit()

	if (firstErr == nil) == (secondErr == nil) {
		t.Fatalf("commits = (%v, %v), want exactly one to succeed", firstErr, secondErr)
	}
	if firstErr != nil {
		requireSerializationConflict(t, firstErr)
	} else {
		requireSerializationConflict(t, secondErr)
	}
}
