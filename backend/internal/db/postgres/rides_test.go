package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

func TestGetRideReportsMissingRide(t *testing.T) {
	ctx := context.Background()
	tx := begin(t, dbtest.DB(t))

	_, err := tx.RidesRepo().GetRide(ctx, uuid.New())

	if !errors.Is(err, app.ErrRideNotFound) {
		t.Errorf("GetRide() of an absent ride = %v, want ErrRideNotFound", err)
	}
}

// TestRideWritesPreserveMissingRideErrors checks both write paths report a
// missing ride with the application sentinel while keeping the SQL cause, which
// is what distinguishes "no such ride" from "the statement failed".
func TestRideWritesPreserveMissingRideErrors(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	_, _, params := insertRide(t, db)
	missing := uuid.New()

	for _, tc := range []struct {
		name string
		call func(app.RideRepository) error
	}{
		{"UpdateRide", func(r app.RideRepository) error { return r.UpdateRide(ctx, missing, params) }},
		{"CancelRide", func(r app.RideRepository) error { return r.CancelRide(ctx, missing) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(begin(t, db).RidesRepo())

			if !errors.Is(err, app.ErrRideNotFound) {
				t.Errorf("%s() = %v, want ErrRideNotFound", tc.name, err)
			}
			if !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("%s() dropped the sql.ErrNoRows cause: %v", tc.name, err)
			}
		})
	}
}

// TestRemoveRideOccupantReportsWhetherAnyoneLeft checks the affected-row count
// the application reads to tell "you left" from "you were never aboard".
func TestRemoveRideOccupantReportsWhetherAnyoneLeft(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, owner, _ := insertRide(t, db)

	t.Run("a member leaves", func(t *testing.T) {
		removed, err := begin(t, db).RidesRepo().RemoveRideOccupant(ctx, rideID, owner)
		if err != nil {
			t.Fatalf("RemoveRideOccupant() error = %v", err)
		}
		if removed != 1 {
			t.Errorf("removed = %d, want 1 for a seated rider", removed)
		}
	})

	t.Run("a non-member leaves", func(t *testing.T) {
		removed, err := begin(t, db).RidesRepo().RemoveRideOccupant(ctx, rideID, uuid.New())
		if err != nil {
			t.Fatalf("RemoveRideOccupant() error = %v", err)
		}
		if removed != 0 {
			t.Errorf("removed = %d, want 0 for someone never aboard", removed)
		}
	})
}

// TestAddRideOccupantReturnsTheRawConstraintFailure checks this adapter leaves a
// foreign-key violation unclassified, so a caller can tell a missing row from a
// ride write the schema rejected.
func TestAddRideOccupantReturnsTheRawConstraintFailure(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, _, _ := insertRide(t, db)

	err := begin(t, db).RidesRepo().AddRideOccupant(ctx, rideID, uuid.New())

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("AddRideOccupant() = %v, want a *pgconn.PgError", err)
	}
	if pgErr.Code != "23503" {
		t.Errorf("SQLSTATE = %q, want a foreign-key violation (23503)", pgErr.Code)
	}
	if errors.Is(err, app.ErrInvalidRideWrite) {
		t.Errorf("AddRideOccupant() classified its raw failure as %v", app.ErrInvalidRideWrite)
	}
}

// TestListRidesAppliesTheDepartureWindow checks the SQL time bounds, including
// the inclusive edge an off-by-one would move.
func TestListRidesAppliesTheDepartureWindow(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, _, params := insertRide(t, db)
	after := params.DepartureTime.Add(-time.Hour)

	for _, tc := range []struct {
		name  string
		bound time.Time
		want  bool
	}{
		{name: "bound before departure", bound: params.DepartureTime.Add(-time.Minute)},
		{name: "bound exactly at departure", bound: params.DepartureTime, want: true},
		{name: "bound after departure", bound: params.DepartureTime.Add(time.Hour), want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bound := tc.bound
			found, err := begin(t, db).RidesRepo().ListRides(ctx, app.ListRidesQuery{
				DepartureAfter: after, DepartureBefore: &bound,
			})
			if err != nil {
				t.Fatalf("ListRides() error = %v", err)
			}

			var seen bool
			for _, ride := range found {
				seen = seen || ride.ID == rideID
			}
			if seen != tc.want {
				t.Errorf("ride within the window = %v, want %v", seen, tc.want)
			}
		})
	}
}

// TestListMyRidesFindsOwnedRideWithoutOccupantRow checks ownership is matched on
// the ride itself. Every ride created through the application also has an
// occupant row for its owner, so only a ride without one proves the difference.
func TestListMyRidesFindsOwnedRideWithoutOccupantRow(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, owner, _ := insertRide(t, db)
	if _, err := db.ExecContext(ctx, `DELETE FROM ride_occupants WHERE ride_id = $1`, rideID); err != nil {
		t.Fatalf("remove occupants: %v", err)
	}

	rides, err := begin(t, db).RidesRepo().ListMyRides(ctx, owner)
	if err != nil {
		t.Fatalf("ListMyRides() error = %v", err)
	}

	if len(rides) != 1 || rides[0].ID != rideID {
		t.Errorf("ListMyRides() returned %d rides, want the owned ride %s", len(rides), rideID)
	}
}
