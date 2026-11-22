package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestListLocationsReturnsRepositoryValues checks that callers receive the
// available locations and the read transaction closes.
func TestListLocationsReturnsRepositoryValues(t *testing.T) {
	want := []LocationRecord{{ID: uuid.New(), Title: "Rice", Address: "6100 Main St"}}
	database := &locationStore{locations: want}
	service := NewService(database)

	got, err := service.ListLocations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("ListLocations = %#v, want %#v", got, want)
	}
	if !database.begun || !database.rolledBack {
		t.Fatal("ListLocations did not close its read transaction")
	}
}

// locationStore records transaction use while supplying location fixtures.
//
// Fields:
//   - orderTx: other repository methods that must not be reached by these tests.
//   - locations: values returned by the location query.
//   - begun: whether the service started a transaction.
//   - rolledBack: whether the service closed its read transaction.
type locationStore struct {
	orderTx
	locations  []LocationRecord
	begun      bool
	rolledBack bool
}

// BeginTx marks a location read as started and returns the fixture transaction.
func (s *locationStore) BeginTx(context.Context) (Transaction, error) {
	s.begun = true
	return s, nil
}

// LocationsRepo exposes the fixture location repository.
func (s *locationStore) LocationsRepo() LocationRepository { return s }

// Rollback records closure of the read transaction.
func (s *locationStore) Rollback() error {
	s.rolledBack = true
	return nil
}

// ListLocations returns the stored fixture locations without an error.
func (s *locationStore) ListLocations(context.Context) ([]LocationRecord, error) {
	return s.locations, nil
}
