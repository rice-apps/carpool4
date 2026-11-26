package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// TestListContactVisibleUserIDsRequiresASharedRide checks the query behind every
// contact-privacy decision in the product. The application trusts this list
// completely, so the rule has to be proven against real SQL rather than a fake.
func TestListContactVisibleUserIDsRequiresASharedRide(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, owner, _ := insertRide(t, db)
	rider, stranger := insertUsers(t, db, 2)[0], insertUsers(t, db, 1)[0]
	if _, err := db.ExecContext(ctx, `INSERT INTO ride_occupants (ride_id, user_id) VALUES ($1, $2)`, rideID, rider); err != nil {
		t.Fatalf("seat rider: %v", err)
	}

	for _, tc := range []struct {
		name   string
		viewer uuid.UUID
		target uuid.UUID
		want   bool
	}{
		{name: "rider may see the owner", viewer: rider, target: owner, want: true},
		{name: "owner may see the rider", viewer: owner, target: rider, want: true},
		{name: "stranger may not see the owner", viewer: stranger, target: owner},
		{name: "stranger may not see the rider", viewer: stranger, target: rider},
	} {
		t.Run(tc.name, func(t *testing.T) {
			visible, err := begin(t, db).UsersRepo().ListContactVisibleUserIDs(ctx, tc.viewer, []uuid.UUID{tc.target})
			if err != nil {
				t.Fatalf("ListContactVisibleUserIDs() error = %v", err)
			}

			got := len(visible) == 1 && visible[0] == tc.target
			if got != tc.want {
				t.Errorf("target visible = %v, want %v (returned %v)", got, tc.want, visible)
			}
		})
	}
}

// TestListContactVisibleUserIDsSurvivesACancelledRide checks that sharing a ride
// grants contact access even after it is cancelled, so riders can still reach
// each other about a trip that fell through.
func TestListContactVisibleUserIDsSurvivesACancelledRide(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	rideID, owner, _ := insertRide(t, db)
	rider := insertUsers(t, db, 1)[0]
	if _, err := db.ExecContext(ctx, `INSERT INTO ride_occupants (ride_id, user_id) VALUES ($1, $2)`, rideID, rider); err != nil {
		t.Fatalf("seat rider: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE rides SET status = 'cancelled' WHERE id = $1`, rideID); err != nil {
		t.Fatalf("cancel ride: %v", err)
	}

	visible, err := begin(t, db).UsersRepo().ListContactVisibleUserIDs(ctx, rider, []uuid.UUID{owner})
	if err != nil {
		t.Fatalf("ListContactVisibleUserIDs() error = %v", err)
	}

	if len(visible) != 1 || visible[0] != owner {
		t.Errorf("visible = %v, want the owner still reachable after cancellation", visible)
	}
}

// TestListContactVisibleUserIDsHandlesAnEmptyBatch checks the query copes with
// the no-targets case the application can produce for an empty ride list.
func TestListContactVisibleUserIDsHandlesAnEmptyBatch(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)

	visible, err := begin(t, db).UsersRepo().ListContactVisibleUserIDs(ctx, uuid.New(), nil)
	if err != nil {
		t.Fatalf("ListContactVisibleUserIDs() with no targets error = %v", err)
	}

	if len(visible) != 0 {
		t.Errorf("visible = %v, want none", visible)
	}
}
