package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestGetVisibleProfileAppliesContactPermissions(t *testing.T) {
	t.Parallel()
	record := testUser(uuid.New(), "omar@rice.edu")
	viewer := uuid.New()

	for _, tc := range []struct {
		name         string
		viewer       uuid.UUID
		shareARide   bool
		wantContacts bool
		wantLookup   bool
	}{
		{name: "own profile", viewer: record.ID, wantContacts: true},
		{name: "shares a ride", viewer: viewer, shareARide: true, wantContacts: true, wantLookup: true},
		{name: "unrelated profile", viewer: viewer, wantLookup: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := newFakeDatabase([]UserRecord{record})
			if tc.shareARide {
				fake.showingContacts()
			}
			tx, err := fake.BeginTx(t.Context())
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}

			got, err := getVisibleProfile(context.Background(), tx, CurrentUser{ID: tc.viewer}, record)
			if err != nil {
				t.Fatalf("getVisibleProfile(%v) error = %v", tc.viewer, err)
			}

			want := User{ID: record.ID, FirstName: record.FirstName, LastName: record.LastName}
			if tc.wantContacts {
				want.Email, want.Phone = record.Email, record.Phone
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("getVisibleProfile(%v) mismatch (-want +got):\n%s", tc.viewer, diff)
			}

			// A viewer reading their own profile needs no permission lookup.
			if got, want := fake.called(opVisibleContacts), tc.wantLookup; got != want {
				t.Errorf("consulted contact permissions = %v, want %v", got, want)
			}
			if !tc.wantLookup {
				return
			}
			if diff := cmp.Diff([][]uuid.UUID{{record.ID}}, fake.tx.contactTargets); diff != "" {
				t.Errorf("permission lookup targets mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]uuid.UUID{tc.viewer}, fake.tx.contactViewers); diff != "" {
				t.Errorf("permission lookup viewers mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetVisibleProfileReturnsPermissionLookupFailure(t *testing.T) {
	t.Parallel()
	record := testUser(uuid.New(), "omar@rice.edu")
	cause := errors.New("permission lookup failed")
	fake := newFakeDatabase([]UserRecord{record}).failing(opVisibleContacts, cause)
	tx, err := fake.BeginTx(t.Context())
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	got, err := getVisibleProfile(context.Background(), tx, CurrentUser{ID: uuid.New()}, record)
	if !errors.Is(err, cause) {
		t.Errorf("getVisibleProfile() error = %v, want %v", err, cause)
	}
	if diff := cmp.Diff(User{}, got); diff != "" {
		t.Errorf("getVisibleProfile() returned a profile despite the failure (-want +got):\n%s", diff)
	}
}
