package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestGetUserAppliesContactPermissions(t *testing.T) {
	t.Parallel()
	viewer, target := uuid.New(), uuid.New()
	record := testUser(target, "target@rice.edu")

	for _, tc := range []struct {
		name         string
		wantContacts bool
	}{
		{name: "permitted viewer sees contacts", wantContacts: true},
		{name: "unrelated viewer sees a redacted profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := newFakeDatabase([]UserRecord{record})
			if tc.wantContacts {
				fake.showingContacts()
			}

			got, err := NewService(fake).GetUser(t.Context(), CurrentUser{ID: viewer}, &target)
			if err != nil {
				t.Fatalf("GetUser(%v) error = %v", target, err)
			}

			want := User{ID: target, FirstName: record.FirstName, LastName: record.LastName}
			if tc.wantContacts {
				want.Email, want.Phone = record.Email, record.Phone
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("GetUser(%v) mismatch (-want +got):\n%s", target, diff)
			}
			requireReadOnly(t, fake)
		})
	}
}

func TestGetUserReturnsLookupFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("private storage detail")
	fake := newFakeDatabase(nil).failing(opFindUser, cause)

	if _, err := NewService(fake).GetUser(t.Context(), CurrentUser{ID: uuid.New()}, nil); !errors.Is(err, cause) {
		t.Errorf("GetUser() error = %v, want %v", err, cause)
	}
	requireReadOnly(t, fake)
}

func TestGetUserRejectsEmptyTarget(t *testing.T) {
	t.Parallel()
	empty := uuid.Nil

	for _, tc := range []struct {
		name   string
		caller CurrentUser
		target *uuid.UUID
	}{
		{name: "unidentified caller reading their own profile", caller: CurrentUser{}},
		{name: "explicitly empty target", caller: CurrentUser{ID: uuid.New()}, target: &empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := newFakeDatabase(nil)

			if _, err := NewService(fake).GetUser(t.Context(), tc.caller, tc.target); !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("GetUser() error = %v, want ErrInvalidArgument", err)
			}
			if fake.called(opBegin) {
				t.Error("GetUser opened a transaction for an empty user ID")
			}
		})
	}
}

func TestUpdateUserRejectsAnUnidentifiedCaller(t *testing.T) {
	t.Parallel()
	fake := newFakeDatabase(nil)

	_, err := NewService(fake).UpdateUser(t.Context(), CurrentUser{Email: "caller@rice.edu"}, UserInput{
		FirstName: "Ada", LastName: "Lovelace",
	})

	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("UpdateUser() error = %v, want ErrInvalidArgument", err)
	}
	if fake.called(opBegin) {
		t.Error("UpdateUser opened a transaction for an empty user ID")
	}
}

// TestUpdateUserTakesIdentityFromTheVerifiedCaller checks that the saved profile
// is keyed on the token's user and email rather than anything in the request,
// and that surrounding whitespace is removed.
func TestUpdateUserTakesIdentityFromTheVerifiedCaller(t *testing.T) {
	t.Parallel()
	caller := CurrentUser{ID: uuid.New(), Email: "  verified@rice.edu  "}
	fake := newFakeDatabase(nil).showingContacts()

	got, err := NewService(fake).UpdateUser(t.Context(), caller, UserInput{
		FirstName: "  Ada  ", LastName: "  Lovelace  ", Phone: "  713-555-0100  ",
	})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	want := User{ID: caller.ID, FirstName: "Ada", LastName: "Lovelace", Email: "verified@rice.edu", Phone: "713-555-0100"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("UpdateUser() mismatch (-want +got):\n%s", diff)
	}
	wantSaved := UserWriteParams{ID: caller.ID, FirstName: "Ada", LastName: "Lovelace", Email: "verified@rice.edu", Phone: "713-555-0100"}
	if diff := cmp.Diff(wantSaved, fake.tx.savedUser); diff != "" {
		t.Errorf("UpsertUser params mismatch (-want +got):\n%s", diff)
	}
	// Callers always see their own contacts, so saving a profile needs no
	// permission lookup. The result is built before the commit, so a projection
	// failure cannot leave a published write behind.
	requireCalls(t, fake, opBegin, opUpsertUser, opCommit, opRollback)
}

func TestUpdateUserRollsBackFailedWrites(t *testing.T) {
	t.Parallel()
	for _, op := range []storageOp{opUpsertUser, opCommit} {
		t.Run(string(op), func(t *testing.T) {
			t.Parallel()
			cause := errors.New(string(op) + " failed")
			fake := newFakeDatabase(nil).showingContacts().failing(op, cause)

			got, err := NewService(fake).UpdateUser(t.Context(), CurrentUser{ID: uuid.New(), Email: "user@rice.edu"},
				UserInput{FirstName: "Test", LastName: "User"})
			if !errors.Is(err, cause) {
				t.Errorf("UpdateUser() error = %v, want %v", err, cause)
			}
			if diff := cmp.Diff(User{}, got); diff != "" {
				t.Errorf("UpdateUser() returned a profile despite the failure (-want +got):\n%s", diff)
			}
			if fake.tx.committed {
				t.Error("UpdateUser committed after the write failed")
			}
		})
	}
}

func TestUpdateUserRejectsInvalidProfiles(t *testing.T) {
	t.Parallel()
	caller := CurrentUser{ID: uuid.New(), Email: "caller@rice.edu"}

	for _, tc := range []struct {
		name  string
		input UserInput
	}{
		{name: "missing first name", input: UserInput{LastName: "User"}},
		{name: "first name over 100 characters", input: UserInput{FirstName: strings.Repeat("x", 101), LastName: "User"}},
		{name: "last name over 100 characters", input: UserInput{FirstName: "Ada", LastName: strings.Repeat("x", 101)}},
		{name: "whitespace-only first name", input: UserInput{FirstName: "   ", LastName: "User"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := newFakeDatabase(nil)

			if _, err := NewService(fake).UpdateUser(t.Context(), caller, tc.input); !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("UpdateUser(%+v) error = %v, want ErrInvalidArgument", tc.input, err)
			}
			if fake.called(opUpsertUser) {
				t.Error("UpdateUser wrote an invalid profile")
			}
		})
	}
}

// TestUpdateUserAcceptsFieldsAtTheirLimit checks the inclusive end of each bound,
// which an off-by-one in validateUserWrite would reject.
func TestUpdateUserAcceptsFieldsAtTheirLimit(t *testing.T) {
	t.Parallel()
	fake := newFakeDatabase(nil).showingContacts()
	limit := strings.Repeat("x", 100)

	got, err := NewService(fake).UpdateUser(t.Context(), CurrentUser{ID: uuid.New(), Email: "caller@rice.edu"},
		UserInput{FirstName: limit, LastName: limit, Phone: strings.Repeat("9", 32)})
	if err != nil {
		t.Fatalf("UpdateUser() with fields at their limit error = %v", err)
	}
	if got.FirstName != limit {
		t.Errorf("first name = %q, want the 100-character value preserved", got.FirstName)
	}
}
