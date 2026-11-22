package app

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestGetUserAppliesContactVisibility checks which contact fields a viewer
// receives and confirms each read transaction closes.
func TestGetUserAppliesContactVisibility(t *testing.T) {
	viewer, target := uuid.New(), uuid.New()
	record := testUserRecord(target, "target@rice.edu")

	for _, visible := range []bool{false, true} {
		t.Run(map[bool]string{false: "redacted", true: "visible"}[visible], func(t *testing.T) {
			store := &profileStore{users: map[uuid.UUID]UserRecord{target: record}, visible: map[uuid.UUID]bool{target: visible}}
			service := NewService(store)
			got, err := service.GetUser(context.Background(), CurrentUser{ID: viewer}, &target)
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != target || got.FirstName != record.FirstName || got.LastName != record.LastName {
				t.Fatalf("GetUser = %#v, want target identity", got)
			}
			if visible && (got.Email != record.Email || got.Phone != record.Phone) {
				t.Fatalf("GetUser redacted authorized contact: %#v", got)
			}
			if !visible && (got.Email != "" || got.Phone != "") {
				t.Fatalf("GetUser exposed unauthorized contact: %#v", got)
			}
			if got, want := store.tx.calls, []string{"begin", "rollback"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("read transaction calls = %v, want %v", got, want)
			}
		})
	}
}

// TestGetUserPreservesApplicationErrors checks that lookup errors reach the
// caller and the read transaction closes.
func TestGetUserPreservesApplicationErrors(t *testing.T) {
	cause := errors.New("private storage detail")
	store := &profileStore{findErr: cause}
	service := NewService(store)
	_, err := service.GetUser(context.Background(), CurrentUser{ID: uuid.New()}, nil)
	if !errors.Is(err, cause) {
		t.Fatalf("GetUser error = %v, want storage cause", err)
	}
	if got, want := store.tx.calls, []string{"begin", "rollback"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("failed read transaction calls = %v, want %v", got, want)
	}
}

// TestUpdateUserUsesCurrentUserIdentityAndNormalizesFields checks that the verified
// currentUser owns the saved profile and surrounding whitespace is removed.
func TestUpdateUserUsesCurrentUserIdentityAndNormalizesFields(t *testing.T) {
	currentUser := CurrentUser{ID: uuid.New(), Email: "  verified@rice.edu  "}
	store := &profileStore{}
	service := NewService(store)

	got, err := service.UpdateUser(context.Background(), currentUser, UserInput{
		FirstName: "  Ada  ", LastName: "  Lovelace  ", Phone: "  713-555-0100  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != currentUser.ID || got.Email != "verified@rice.edu" || got.FirstName != "Ada" || got.LastName != "Lovelace" || got.Phone != "713-555-0100" {
		t.Fatalf("UpdateUser = %#v, want normalized currentUser-owned profile", got)
	}
	if store.saved.ID != currentUser.ID || store.saved.Email != "verified@rice.edu" {
		t.Fatalf("UpsertUser params = %#v, want verified currentUser identity", store.saved)
	}
	if got, want := store.tx.calls, []string{"begin", "upsert", "commit", "rollback"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("transaction calls = %v, want %v", got, want)
	}
}

// TestUpdateUserRollsBackUpsertAndCommitFailures checks that failed saves
// return their cause without publishing a profile.
func TestUpdateUserRollsBackUpsertAndCommitFailures(t *testing.T) {
	for _, stage := range []string{"upsert", "commit"} {
		t.Run(stage, func(t *testing.T) {
			cause := errors.New(stage + " failed")
			store := &profileStore{fail: stage, cause: cause}
			service := NewService(store)

			got, err := service.UpdateUser(context.Background(), CurrentUser{ID: uuid.New(), Email: "user@rice.edu"}, UserInput{
				FirstName: "Test", LastName: "User",
			})
			if !errors.Is(err, cause) {
				t.Fatalf("UpdateUser error = %v, want %v", err, cause)
			}
			if got != (User{}) || store.saved != (UserWriteParams{}) || store.tx.committed || !store.tx.rolledBack {
				t.Fatalf("failed transaction published result or write: result=%#v store=%#v tx=%#v", got, store.saved, store.tx)
			}
		})
	}
}

// TestUpdateUserRejectsInvalidApplicationValues checks invalid profile fields.
func TestUpdateUserRejectsInvalidApplicationValues(t *testing.T) {
	service := NewService(&profileStore{})
	currentUser := CurrentUser{ID: uuid.New(), Email: "caller@rice.edu"}

	for _, input := range []UserInput{
		{FirstName: "", LastName: "User"},
		{FirstName: "Ada", LastName: strings.Repeat("x", 101)},
	} {
		if _, err := service.UpdateUser(context.Background(), currentUser, input); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("UpdateUser(%#v) error = %v, want invalid argument", input, err)
		}
	}
}

// profileStore holds profile fixtures and saved results for service tests.
//
// Fields:
//   - users: stored profiles available to lookups.
//   - visible: whether each user's contacts are visible to the test viewer.
//   - findErr: optional error returned by profile lookup.
//   - saved: last profile published by a committed write.
//   - tx: most recent fixture transaction.
//   - fail: name of the transaction stage set to fail.
//   - cause: error returned at that stage.
type profileStore struct {
	users   map[uuid.UUID]UserRecord
	visible map[uuid.UUID]bool
	findErr error
	saved   UserWriteParams
	tx      *profileTx
	fail    string
	cause   error
}

// profileTx tracks profile reads and holds writes until commit.
//
// Fields:
//   - store: parent fixture that receives committed values.
//   - saved: profile staged for this transaction.
//   - calls: transaction operations in the order they occurred.
//   - committed: whether the write was published.
//   - rolledBack: whether rollback was called.
type profileTx struct {
	store      *profileStore
	saved      UserWriteParams
	calls      []string
	committed  bool
	rolledBack bool
}

// BeginTx returns a fresh profile transaction or the configured begin error.
func (s *profileStore) BeginTx(context.Context) (Transaction, error) {
	if s.fail == "begin" {
		return nil, s.cause
	}
	s.tx = &profileTx{store: s, calls: []string{"begin"}}
	return s.tx, nil
}

// UsersRepo exposes the profile methods of this transaction.
func (tx *profileTx) UsersRepo() UserRepository { return tx }

// RidesRepo panics because profile tests must not access ride storage.
func (*profileTx) RidesRepo() RideRepository { panic("unexpected ride call") }

// LocationsRepo panics because profile tests must not access location storage.
func (*profileTx) LocationsRepo() LocationRepository { panic("unexpected location call") }

// FindUser returns id's fixture profile or its configured error.
func (tx *profileTx) FindUser(ctx context.Context, id uuid.UUID) (UserRecord, error) {
	return tx.store.FindUser(ctx, id)
}

// ListContactVisibleUserIDs returns fixture contacts visible among targets.
func (tx *profileTx) ListContactVisibleUserIDs(ctx context.Context, viewer uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	return tx.store.ListContactVisibleUserIDs(ctx, viewer, targets)
}

// FindUser returns id's fixture profile, a configured failure, or ErrUserNotFound.
func (s *profileStore) FindUser(_ context.Context, id uuid.UUID) (UserRecord, error) {
	if s.findErr != nil {
		return UserRecord{}, s.findErr
	}
	user, ok := s.users[id]
	if !ok {
		return UserRecord{}, ErrUserNotFound
	}
	return user, nil
}

// UpsertUser returns a profile from params or the configured error. The
// fixture publishes saved fields only on Commit.
func (tx *profileTx) UpsertUser(_ context.Context, params UserWriteParams) (UserRecord, error) {
	tx.calls = append(tx.calls, "upsert")
	if tx.store.fail == "upsert" {
		return UserRecord{}, tx.store.cause
	}
	tx.saved = params
	return UserRecord{
		ID: params.ID, FirstName: params.FirstName, LastName: params.LastName,
		Email: params.Email, Phone: params.Phone,
	}, nil
}

// Commit publishes the staged profile or returns the configured commit failure.
func (tx *profileTx) Commit() error {
	tx.calls = append(tx.calls, "commit")
	if tx.store.fail == "commit" {
		return tx.store.cause
	}
	tx.committed = true
	tx.store.saved = tx.saved
	return nil
}

// Rollback records closure without publishing staged profile fields.
func (tx *profileTx) Rollback() error {
	tx.calls = append(tx.calls, "rollback")
	tx.rolledBack = true
	return nil
}

// ListContactVisibleUserIDs returns target IDs marked visible by the fixture.
func (s *profileStore) ListContactVisibleUserIDs(_ context.Context, _ uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(targets))
	for _, id := range targets {
		if s.visible[id] {
			result = append(result, id)
		}
	}
	return result, nil
}

// testUserRecord returns a fixture profile with id and email.
func testUserRecord(id uuid.UUID, email string) UserRecord {
	return UserRecord{ID: id, FirstName: "Target", LastName: "User", Email: email, Phone: "713-555-0100"}
}
