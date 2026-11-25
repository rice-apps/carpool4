package postgres

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/rice-apps/carpool4/backend/internal/app"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// TestListMyRidesFindsOwnedRideWithoutOccupantRow checks ownership is matched on the
// ride itself, so a missing occupant row cannot hide a ride from the person who owns it.
func TestListMyRidesFindsOwnedRideWithoutOccupantRow(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, _ := insertTransactionRide(t, db)

	var owner uuid.UUID
	if err := db.QueryRowContext(ctx, `SELECT owner_id FROM rides WHERE id = $1`, rideID).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	// Leave the ride without any occupant rows, as a backfill or support fix could.
	if _, err := db.ExecContext(ctx, `DELETE FROM ride_occupants WHERE ride_id = $1`, rideID); err != nil {
		t.Fatal(err)
	}

	tx, err := NewDatabase(db).BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	rides, err := tx.RidesRepo().ListMyRides(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(rides) != 1 || rides[0].ID != rideID {
		t.Fatalf("ListMyRides returned %d rides, want the owned ride %s", len(rides), rideID)
	}
}

// TestRideRepositoryReadAndMembershipEdges covers the storage answers the application
// branches on: a missing ride, a non-member leaving, and the discovery time window.
// One sandbox setup covers all three.
func TestRideRepositoryReadAndMembershipEdges(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, params := insertTransactionRide(t, db)

	tx, err := NewDatabase(db).BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	rides := tx.RidesRepo()

	t.Run("missing ride", func(t *testing.T) {
		if _, err := rides.GetRide(ctx, uuid.New()); !errors.Is(err, app.ErrRideNotFound) {
			t.Fatalf("GetRide of an absent ride = %v, want ErrRideNotFound", err)
		}
	})

	t.Run("non-member leave reports no rows", func(t *testing.T) {
		removed, err := rides.RemoveRideOccupant(ctx, rideID, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if removed != 0 {
			t.Fatalf("RemoveRideOccupant removed %d rows, want 0 for a non-member", removed)
		}
	})

	t.Run("locations list runs its real query", func(t *testing.T) {
		// The RPC tests all use a fake store, so this is the only place the
		// location query itself executes.
		found, err := tx.LocationsRepo().ListLocations(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// insertTransactionRide inserts exactly these two, so assert them by name
		// rather than comparing titles in Go: the query orders by the database
		// collation, which need not match Go's byte ordering.
		var titles []string
		for _, location := range found {
			if location.ID == uuid.Nil {
				t.Fatalf("location %q has no ID", location.Title)
			}
			titles = append(titles, location.Title)
		}
		if !reflect.DeepEqual(titles, []string{"Arrival", "Departure"}) {
			t.Fatalf("ListLocations = %q, want the fixture's two titles in order", titles)
		}
	})

	t.Run("occupant insert returns the raw constraint failure", func(t *testing.T) {
		// A rejected statement aborts its transaction, so this runs in its own.
		// JoinRide relies on this staying unclassified: an occupant foreign-key
		// failure is a server fault, not an invalid ride write.
		own, err := NewDatabase(db).BeginTx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer own.Rollback()

		err = own.RidesRepo().AddRideOccupant(ctx, rideID, uuid.New())
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != "23503" {
			t.Fatalf("occupant constraint error = %v, want a 23503 foreign-key failure", err)
		}
		if errors.Is(err, app.ErrInvalidRideWrite) {
			t.Fatalf("occupant insertion classified its raw failure: %v", err)
		}
	})

	t.Run("departure window excludes later rides", func(t *testing.T) {
		after := params.DepartureTime.Add(-time.Hour)
		tooEarly := params.DepartureTime.Add(-time.Minute)
		found, err := rides.ListRides(ctx, app.ListRidesQuery{DepartureAfter: after, DepartureBefore: &tooEarly})
		if err != nil {
			t.Fatal(err)
		}
		for _, ride := range found {
			if ride.ID == rideID {
				t.Fatal("ListRides returned a ride departing after DepartureBefore")
			}
		}

		late := params.DepartureTime.Add(time.Hour)
		found, err = rides.ListRides(ctx, app.ListRidesQuery{DepartureAfter: after, DepartureBefore: &late})
		if err != nil {
			t.Fatal(err)
		}
		var seen bool
		for _, ride := range found {
			seen = seen || ride.ID == rideID
		}
		if !seen {
			t.Fatal("ListRides omitted a ride inside the departure window")
		}
	})
}
