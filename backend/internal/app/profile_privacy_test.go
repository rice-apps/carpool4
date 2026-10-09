package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestGetVisibleProfileAppliesContactPermissions(t *testing.T) {
	record := UserRecord{ID: uuid.New(), FirstName: "Alex", LastName: "Chen", Email: "alex@rice.edu", Phone: "713-555-0100"}
	otherUser := CurrentUser{UserID: uuid.New()}
	for _, test := range []struct {
		name         string
		currentUser  CurrentUser
		visibleIDs   []uuid.UUID
		wantContacts bool
		wantLookup   bool
	}{
		{name: "own profile", currentUser: CurrentUser{UserID: record.ID}, wantContacts: true},
		{name: "shared ride", currentUser: otherUser, visibleIDs: []uuid.UUID{record.ID}, wantContacts: true, wantLookup: true},
		{name: "unrelated profile", currentUser: otherUser, wantLookup: true},
		{name: "guest"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := &profilePrivacyTx{visibleIDs: test.visibleIDs}
			got, err := getVisibleProfile(context.Background(), tx, test.currentUser, record)
			if err != nil {
				t.Fatal(err)
			}
			want := User{ID: record.ID, FirstName: record.FirstName, LastName: record.LastName}
			if test.wantContacts {
				want.Email, want.Phone = record.Email, record.Phone
			}
			if got != want {
				t.Fatalf("profile = %#v, want %#v", got, want)
			}
			if test.wantLookup {
				if tx.lookups != 1 || tx.viewerID != test.currentUser.UserID || !reflect.DeepEqual(tx.targetIDs, []uuid.UUID{record.ID}) {
					t.Fatalf("permission lookup = %#v, want the current user and requested profile", tx)
				}
			} else if tx.lookups != 0 {
				t.Fatalf("permission lookups = %d, want none", tx.lookups)
			}
		})
	}
}

func TestGetVisibleProfileReturnsPermissionLookupFailure(t *testing.T) {
	cause := errors.New("permission lookup failed")
	tx := &profilePrivacyTx{lookupErr: cause}
	record := UserRecord{ID: uuid.New(), Email: "alex@rice.edu", Phone: "713-555-0100"}
	got, err := getVisibleProfile(context.Background(), tx, CurrentUser{UserID: uuid.New()}, record)
	if !errors.Is(err, cause) || got != (User{}) {
		t.Fatalf("profile = %#v, error = %v; want no result and the lookup failure", got, err)
	}
}

// profilePrivacyTx supplies contact permissions without opening a database.
type profilePrivacyTx struct {
	Transaction
	UserRepository
	visibleIDs []uuid.UUID
	lookupErr  error
	viewerID   uuid.UUID
	targetIDs  []uuid.UUID
	lookups    int
}

func (tx *profilePrivacyTx) Users() UserRepository { return tx }

func (tx *profilePrivacyTx) ListContactVisibleUserIDs(_ context.Context, viewerID uuid.UUID, targetIDs []uuid.UUID) ([]uuid.UUID, error) {
	tx.lookups++
	tx.viewerID, tx.targetIDs = viewerID, targetIDs
	return tx.visibleIDs, tx.lookupErr
}
