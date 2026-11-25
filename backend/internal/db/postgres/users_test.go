package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// TestUpsertUserRefreshesUpdatedAt checks an edited profile records when it changed,
// so the column keeps tracking reality after the first insert.
func TestUpsertUserRefreshesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	id := insertTransactionUsers(t, db, 1)[0]

	var before time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM users WHERE id = $1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}

	tx, err := NewDatabase(db).BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.UsersRepo().UpsertUser(ctx, app.UserWriteParams{
		ID: id, Email: id.String() + "@rice.edu", FirstName: "Edited", LastName: "User", Phone: "713-555-0199",
	}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var after time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM users WHERE id = $1`, id).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !after.After(before) {
		t.Fatalf("updated_at = %s, want a time after the insert at %s", after, before)
	}
}
