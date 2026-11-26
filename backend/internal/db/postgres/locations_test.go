package postgres

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

// TestListLocationsOrdersByTitle checks the real query. Every other layer uses a
// fake for locations, so this is the only place the SQL itself runs.
func TestListLocationsOrdersByTitle(t *testing.T) {
	ctx := context.Background()
	db := dbtest.DB(t)
	departure, arrival := insertLocations(t, db)

	found, err := begin(t, db).LocationsRepo().ListLocations(ctx)
	if err != nil {
		t.Fatalf("ListLocations() error = %v", err)
	}

	gotTitles := make([]string, len(found))
	gotIDs := make([]uuid.UUID, len(found))
	for i, location := range found {
		gotTitles[i], gotIDs[i] = location.Title, location.ID
	}
	if diff := cmp.Diff([]string{"Arrival", "Departure"}, gotTitles); diff != "" {
		t.Errorf("ListLocations() titles mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]uuid.UUID{arrival, departure}, gotIDs); diff != "" {
		t.Errorf("ListLocations() IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestListLocationsReturnsNothingWhenEmpty(t *testing.T) {
	ctx := context.Background()

	found, err := begin(t, dbtest.DB(t)).LocationsRepo().ListLocations(ctx)
	if err != nil {
		t.Fatalf("ListLocations() error = %v", err)
	}

	if len(found) != 0 {
		t.Errorf("ListLocations() returned %d locations, want none", len(found))
	}
}
