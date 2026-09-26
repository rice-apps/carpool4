package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectRidesHidesPeopleAndNotesFromGuests(t *testing.T) {
	owner := UserRecord{ID: uuid.New(), FirstName: "Alex", Email: "alex@rice.edu", Phone: "7135550100"}
	loaded := LoadedRide{
		ID: uuid.New(), DepartureTime: time.Now().Add(time.Hour), Owner: owner,
		Riders: []UserRecord{owner}, Notes: "Private pickup details", Capacity: 2,
	}
	db := &starterDB{}
	got, err := projectRides(context.Background(), db, uuid.Nil, []LoadedRide{loaded})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Owner != (User{}) || len(got[0].Riders) != 0 || got[0].Notes != "" || got[0].OccupiedSeats != 1 {
		t.Fatalf("guest ride = %#v", got)
	}
	if db.contactLookups != 0 {
		t.Fatalf("guest checked contact visibility %d times", db.contactLookups)
	}
}

func TestProjectRidesKeepsSignedInPeople(t *testing.T) {
	owner := UserRecord{ID: uuid.New(), FirstName: "Alex", Email: "alex@rice.edu"}
	loaded := LoadedRide{ID: uuid.New(), Owner: owner, Riders: []UserRecord{owner}, Notes: "Pickup at gate", Capacity: 2}
	got, err := projectRides(context.Background(), &starterDB{}, owner.ID, []LoadedRide{loaded})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Owner.ID != owner.ID || got[0].Owner.Email != owner.Email || len(got[0].Riders) != 1 || got[0].Notes != loaded.Notes || got[0].OccupiedSeats != 1 {
		t.Fatalf("signed-in ride = %#v", got)
	}
}
