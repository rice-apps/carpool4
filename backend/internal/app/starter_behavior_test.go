package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// These small examples skip automatically while operations return ErrNotImplemented.
// As you implement each operation, its cases will run automatically. Make
// your operation's cases pass, then add cases for the remaining rules.
func TestStarterBehavior(t *testing.T) {
	ctx := context.Background()

	t.Run("GetUser/self contact", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).GetUser(ctx, Actor{UserID: f.owner.ID}, nil)
		skipIfUnimplemented(t, err)
		if err != nil || got.ID != f.owner.ID || got.Email != f.owner.Email {
			t.Fatalf("GetUser = %#v, %v; want caller's full profile", got, err)
		}
	})
	t.Run("GetUser/unrelated contact redacted", func(t *testing.T) {
		f := newStarterFixture()
		target := f.rider.ID
		got, err := NewService(f.db).GetUser(ctx, Actor{UserID: f.stranger.ID}, &target)
		skipIfUnimplemented(t, err)
		if err != nil || got.ID != target || got.Email != "" || got.Phone != "" {
			t.Fatalf("GetUser = %#v, %v; want target without contact fields", got, err)
		}
	})

	t.Run("UpdateUser/saves verified actor", func(t *testing.T) {
		f := newStarterFixture()
		actor := Actor{UserID: f.owner.ID, Email: "  owner@rice.edu  "}
		got, err := NewService(f.db).UpdateUser(ctx, actor, UserInput{FirstName: "  Ada  ", LastName: "  Rice  ", Phone: "  713-555-0100  "})
		skipIfUnimplemented(t, err)
		if err != nil || got.ID != actor.UserID || got.FirstName != "Ada" || got.Email != "owner@rice.edu" || f.db.saved.ID != actor.UserID || f.db.commits != 1 {
			t.Fatalf("UpdateUser = %#v, %v; saved = %#v, commits = %d", got, err, f.db.saved, f.db.commits)
		}
	})
	t.Run("UpdateUser/rejects blank name", func(t *testing.T) {
		f := newStarterFixture()
		_, err := NewService(f.db).UpdateUser(ctx, Actor{UserID: f.owner.ID, Email: f.owner.Email}, UserInput{LastName: "Rice"})
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrInvalidArgument) || f.db.commits != 0 {
			t.Fatalf("UpdateUser error = %v, commits = %d; want invalid argument without commit", err, f.db.commits)
		}
	})

	t.Run("ListLocations/returns ordered locations", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).ListLocations(ctx, Actor{UserID: f.owner.ID})
		skipIfUnimplemented(t, err)
		if err != nil || len(got) != 2 || got[0].Title != "Airport" || got[1].Title != "Rice" {
			t.Fatalf("ListLocations = %#v, %v; want two ordered locations", got, err)
		}
	})
	t.Run("ListLocations/requires caller", func(t *testing.T) {
		f := newStarterFixture()
		_, err := NewService(f.db).ListLocations(ctx, Actor{})
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("ListLocations error = %v, want unauthenticated", err)
		}
	})

	t.Run("GetRide/loads ride", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).GetRide(ctx, Actor{UserID: f.owner.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if err != nil || got.ID != f.db.ride.ID || got.Owner.Email != f.owner.Email || len(got.Riders) != 1 {
			t.Fatalf("GetRide = %#v, %v; want owner-visible ride", got, err)
		}
	})
	t.Run("GetRide/redacts stranger contacts", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).GetRide(ctx, Actor{UserID: f.stranger.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if err != nil || got.ID != f.db.ride.ID || got.Owner.Email != "" || got.Owner.Phone != "" {
			t.Fatalf("GetRide = %#v, %v; want ride without owner contact", got, err)
		}
	})

	t.Run("CreateRide/owner takes first seat", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).CreateRide(ctx, Actor{UserID: f.owner.ID}, f.input)
		skipIfUnimplemented(t, err)
		if err != nil || got.Owner.ID != f.owner.ID || len(got.Riders) != 1 || got.Riders[0].ID != f.owner.ID || f.db.commits != 1 {
			t.Fatalf("CreateRide = %#v, %v; commits = %d", got, err, f.db.commits)
		}
	})
	t.Run("CreateRide/requires owner phone", func(t *testing.T) {
		f := newStarterFixture()
		owner := f.owner
		owner.Phone = ""
		f.db.users[owner.ID] = owner
		_, err := NewService(f.db).CreateRide(ctx, Actor{UserID: owner.ID}, f.input)
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrPhoneRequired) || f.db.commits != 0 {
			t.Fatalf("CreateRide error = %v, commits = %d; want phone prerequisite", err, f.db.commits)
		}
	})

	t.Run("UpdateRide/owner edits ride", func(t *testing.T) {
		f := newStarterFixture()
		f.input.Notes = "New pickup point"
		f.input.Capacity = 4
		got, err := NewService(f.db).UpdateRide(ctx, Actor{UserID: f.owner.ID}, f.db.ride.ID, f.input)
		skipIfUnimplemented(t, err)
		if err != nil || got.Notes != f.input.Notes || got.Capacity != 4 || f.db.commits != 1 {
			t.Fatalf("UpdateRide = %#v, %v; commits = %d", got, err, f.db.commits)
		}
	})
	t.Run("UpdateRide/capacity fits occupants", func(t *testing.T) {
		f := newStarterFixture()
		f.db.ride.Riders = append(f.db.ride.Riders, f.rider)
		f.input.Capacity = 1
		_, err := NewService(f.db).UpdateRide(ctx, Actor{UserID: f.owner.ID}, f.db.ride.ID, f.input)
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrCapacityBelowOccupancy) || f.db.commits != 0 {
			t.Fatalf("UpdateRide error = %v, commits = %d; want capacity rejection", err, f.db.commits)
		}
	})

	t.Run("CancelRide/owner cancels", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).CancelRide(ctx, Actor{UserID: f.owner.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if err != nil || got.Status != RideStatusCancelled || f.db.commits != 1 {
			t.Fatalf("CancelRide = %#v, %v; commits = %d", got, err, f.db.commits)
		}
	})
	t.Run("CancelRide/nonowner denied", func(t *testing.T) {
		f := newStarterFixture()
		_, err := NewService(f.db).CancelRide(ctx, Actor{UserID: f.rider.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrPermissionDenied) || f.db.commits != 0 {
			t.Fatalf("CancelRide error = %v, commits = %d; want permission denied", err, f.db.commits)
		}
	})

	t.Run("JoinRide/adds rider", func(t *testing.T) {
		f := newStarterFixture()
		got, err := NewService(f.db).JoinRide(ctx, Actor{UserID: f.rider.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if err != nil || len(got.Riders) != 2 || got.Riders[1].ID != f.rider.ID || f.db.commits != 1 {
			t.Fatalf("JoinRide = %#v, %v; commits = %d", got, err, f.db.commits)
		}
	})
	t.Run("JoinRide/full ride rejected", func(t *testing.T) {
		f := newStarterFixture()
		f.db.ride.Capacity = 1
		_, err := NewService(f.db).JoinRide(ctx, Actor{UserID: f.rider.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrFullCapacity) || f.db.commits != 0 {
			t.Fatalf("JoinRide error = %v, commits = %d; want full capacity", err, f.db.commits)
		}
	})

	t.Run("LeaveRide/removes rider", func(t *testing.T) {
		f := newStarterFixture()
		f.db.ride.Riders = append(f.db.ride.Riders, f.rider)
		got, err := NewService(f.db).LeaveRide(ctx, Actor{UserID: f.rider.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if err != nil || len(got.Riders) != 1 || got.Riders[0].ID != f.owner.ID || f.db.commits != 1 {
			t.Fatalf("LeaveRide = %#v, %v; commits = %d", got, err, f.db.commits)
		}
	})
	t.Run("LeaveRide/owner stays on ride", func(t *testing.T) {
		f := newStarterFixture()
		_, err := NewService(f.db).LeaveRide(ctx, Actor{UserID: f.owner.ID}, f.db.ride.ID)
		skipIfUnimplemented(t, err)
		if !errors.Is(err, ErrOwnerCannotChangeMembership) || f.db.commits != 0 {
			t.Fatalf("LeaveRide error = %v, commits = %d; want owner rejection", err, f.db.commits)
		}
	})

	t.Run("ListRides/passes filters and returns matches", func(t *testing.T) {
		f := newStarterFixture()
		f.db.listed = []LoadedRide{f.db.ride}
		departureID := f.input.DepartureLocationID
		got, err := NewService(f.db).ListRides(ctx, Actor{UserID: f.stranger.ID}, ListRidesInput{DepartureLocationID: &departureID})
		skipIfUnimplemented(t, err)
		if err != nil || len(got) != 1 || f.db.query == nil || f.db.query.DepartureLocationID == nil || *f.db.query.DepartureLocationID != departureID {
			t.Fatalf("ListRides = %#v, %v; query = %#v", got, err, f.db.query)
		}
	})
	t.Run("ListRides/default lookback", func(t *testing.T) {
		f := newStarterFixture()
		before := time.Now().Add(-time.Hour - 5*time.Second)
		_, err := NewService(f.db).ListRides(ctx, Actor{UserID: f.owner.ID}, ListRidesInput{})
		skipIfUnimplemented(t, err)
		after := time.Now().Add(-time.Hour + 5*time.Second)
		if err != nil || f.db.query == nil || f.db.query.DepartureAfter.Before(before) || f.db.query.DepartureAfter.After(after) {
			t.Fatalf("ListRides error = %v, query = %#v; want one-hour lookback", err, f.db.query)
		}
	})

	t.Run("ListMyRides/includes owned and joined history", func(t *testing.T) {
		f := newStarterFixture()
		joined := f.db.ride
		joined.ID = uuid.New()
		joined.Owner = f.rider
		joined.Riders = []UserRecord{f.rider, f.owner}
		joined.Status = RideStatusCancelled
		f.db.listed = []LoadedRide{f.db.ride, joined}
		got, err := NewService(f.db).ListMyRides(ctx, Actor{UserID: f.owner.ID})
		skipIfUnimplemented(t, err)
		if err != nil || len(got) != 2 {
			t.Fatalf("ListMyRides = %#v, %v; want owned and joined rides", got, err)
		}
	})
	t.Run("ListMyRides/scopes to verified caller", func(t *testing.T) {
		f := newStarterFixture()
		unrelated := f.db.ride
		unrelated.ID = uuid.New()
		unrelated.Owner = f.stranger
		unrelated.Riders = []UserRecord{f.stranger}
		f.db.listed = []LoadedRide{f.db.ride, unrelated}
		got, err := NewService(f.db).ListMyRides(ctx, Actor{UserID: f.owner.ID})
		skipIfUnimplemented(t, err)
		if err != nil || f.db.myViewer != f.owner.ID || len(got) != 1 || got[0].ID != f.db.ride.ID {
			t.Fatalf("ListMyRides = %#v, %v; viewer = %s", got, err, f.db.myViewer)
		}
	})
}

func skipIfUnimplemented(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, ErrNotImplemented) {
		t.Skip("operation not implemented")
	}
}

type starterFixture struct {
	db       *starterDB
	owner    UserRecord
	rider    UserRecord
	stranger UserRecord
	input    RideWriteParams
}

func newStarterFixture() starterFixture {
	owner := UserRecord{ID: uuid.New(), FirstName: "Owner", LastName: "Rice", Email: "owner@rice.edu", Phone: "713-555-0100"}
	rider := UserRecord{ID: uuid.New(), FirstName: "Rider", LastName: "Rice", Email: "rider@rice.edu", Phone: "713-555-0101"}
	stranger := UserRecord{ID: uuid.New(), FirstName: "Other", LastName: "Rice", Email: "other@rice.edu", Phone: "713-555-0102"}
	airport := LocationRecord{ID: uuid.New(), Title: "Airport"}
	rice := LocationRecord{ID: uuid.New(), Title: "Rice"}
	input := RideWriteParams{DepartureTime: time.Now().Add(24 * time.Hour), DepartureLocationID: rice.ID, ArrivalLocationID: airport.ID, Capacity: 3}
	db := &starterDB{
		users:     map[uuid.UUID]UserRecord{owner.ID: owner, rider.ID: rider, stranger.ID: stranger},
		locations: []LocationRecord{airport, rice},
		ride: LoadedRide{
			ID: uuid.New(), DepartureTime: input.DepartureTime, Owner: owner,
			Departure: rice, Arrival: airport, Riders: []UserRecord{owner}, Capacity: 3, Status: RideStatusActive,
		},
	}
	return starterFixture{db: db, owner: owner, rider: rider, stranger: stranger, input: input}
}

// starterDB supplies storage results without enforcing application rules.
// Its methods are deliberately small so the tests focus on service behavior.
type starterDB struct {
	users     map[uuid.UUID]UserRecord
	locations []LocationRecord
	ride      LoadedRide
	listed    []LoadedRide
	saved     UserWriteParams
	query     *ListRidesQuery
	myViewer  uuid.UUID
	commits   int
}

func (db *starterDB) BeginTx(context.Context) (Transaction, error) { return db, nil }
func (db *starterDB) Users() UserRepository                        { return db }
func (db *starterDB) Rides() RideRepository                        { return db }
func (db *starterDB) Locations() LocationRepository                { return db }
func (db *starterDB) Commit() error                                { db.commits++; return nil }
func (db *starterDB) Rollback() error                              { return nil }

func (db *starterDB) FindUser(_ context.Context, id uuid.UUID) (UserRecord, error) {
	user, ok := db.users[id]
	if !ok {
		return UserRecord{}, ErrUserNotFound
	}
	return user, nil
}
func (db *starterDB) UpsertUser(_ context.Context, params UserWriteParams) (UserRecord, error) {
	db.saved = params
	record := UserRecord{ID: params.ID, FirstName: params.FirstName, LastName: params.LastName, Email: params.Email, Phone: params.Phone}
	db.users[params.ID] = record
	return record, nil
}
func (db *starterDB) ListContactVisibleUserIDs(_ context.Context, viewer uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
	var visible []uuid.UUID
	for _, id := range targets {
		if id == viewer {
			visible = append(visible, id)
		}
	}
	return visible, nil
}
func (db *starterDB) GetRide(_ context.Context, id uuid.UUID) (LoadedRide, error) {
	if id != db.ride.ID {
		return LoadedRide{}, ErrRideNotFound
	}
	return db.ride, nil
}
func (db *starterDB) CreateRide(_ context.Context, ownerID uuid.UUID, params RideWriteParams) (uuid.UUID, error) {
	db.ride = LoadedRide{
		ID: uuid.New(), Owner: db.users[ownerID], DepartureTime: params.DepartureTime,
		Departure: LocationRecord{ID: params.DepartureLocationID}, Arrival: LocationRecord{ID: params.ArrivalLocationID},
		Notes: params.Notes, Capacity: params.Capacity, Status: RideStatusActive,
	}
	return db.ride.ID, nil
}
func (db *starterDB) UpdateRide(_ context.Context, _ uuid.UUID, params RideWriteParams) error {
	db.ride.DepartureTime = params.DepartureTime
	db.ride.Departure = LocationRecord{ID: params.DepartureLocationID}
	db.ride.Arrival = LocationRecord{ID: params.ArrivalLocationID}
	db.ride.Notes = params.Notes
	db.ride.Capacity = params.Capacity
	return nil
}
func (db *starterDB) CancelRide(context.Context, uuid.UUID) error {
	db.ride.Status = RideStatusCancelled
	return nil
}
func (db *starterDB) AddRideOccupant(_ context.Context, _, userID uuid.UUID) error {
	db.ride.Riders = append(db.ride.Riders, db.users[userID])
	return nil
}
func (db *starterDB) RemoveRideOccupant(_ context.Context, _, userID uuid.UUID) (int64, error) {
	for i, rider := range db.ride.Riders {
		if rider.ID == userID {
			db.ride.Riders = append(db.ride.Riders[:i], db.ride.Riders[i+1:]...)
			return 1, nil
		}
	}
	return 0, nil
}
func (db *starterDB) ListRides(_ context.Context, query ListRidesQuery) ([]LoadedRide, error) {
	db.query = &query
	return db.listed, nil
}
func (db *starterDB) ListMyRides(_ context.Context, viewerID uuid.UUID) ([]LoadedRide, error) {
	db.myViewer = viewerID
	var rides []LoadedRide
	for _, ride := range db.listed {
		if ride.Owner.ID == viewerID {
			rides = append(rides, ride)
			continue
		}
		for _, rider := range ride.Riders {
			if rider.ID == viewerID {
				rides = append(rides, ride)
				break
			}
		}
	}
	return rides, nil
}
func (db *starterDB) ListLocations(context.Context) ([]LocationRecord, error) {
	return db.locations, nil
}
