package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// TestUpsertUserRefreshesUpdatedAt checks an edited profile records when it
// changed. The column is NOT NULL DEFAULT now(), so it looks maintained whether
// or not the update path sets it, and nothing else in the suite reads it.
func TestUpsertUserRefreshesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	id := insertUsers(t, db, 1)[0]

	var before time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM users WHERE id = $1`, id).Scan(&before); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}

	tx := begin(t, db)
	if _, err := tx.UsersRepo().UpsertUser(ctx, app.UserWriteParams{
		ID: id, Email: id.String() + "@rice.edu", FirstName: "Edited", LastName: "User", Phone: "713-555-0199",
	}); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	var after time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM users WHERE id = $1`, id).Scan(&after); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if !after.After(before) {
		t.Errorf("updated_at = %s, want a time after the insert at %s", after, before)
	}
}

func TestFindUserReportsMissingProfile(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	id := insertUsers(t, db, 1)[0]

	t.Run("existing profile", func(t *testing.T) {
		got, err := begin(t, db).UsersRepo().FindUser(ctx, id)
		if err != nil {
			t.Fatalf("FindUser(%v) error = %v", id, err)
		}
		if got.ID != id {
			t.Errorf("FindUser(%v) returned %v", id, got.ID)
		}
	})

	t.Run("absent profile", func(t *testing.T) {
		_, err := begin(t, db).UsersRepo().FindUser(ctx, uuid.New())
		if err == nil {
			t.Error("FindUser() of an absent profile succeeded, want ErrUserNotFound")
		}
	})
}
