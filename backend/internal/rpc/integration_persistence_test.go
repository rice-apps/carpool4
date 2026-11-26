package rpc_test

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
)

// newProfileServer returns a server and a caller who exists in auth.users but
// has no profile row yet, which is the state every new sign-in starts from.
func newProfileServer(t *testing.T) (*rpc.Server, *sql.DB, uuid.UUID) {
	t.Helper()
	db := dbtest.DB(t)
	id := uuid.New()
	if _, err := db.Exec(`INSERT INTO auth.users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("insert auth identity: %v", err)
	}
	return newRPCServer(postgres.NewDatabase(db)), db, id
}

// trimmedProfileRequest carries padded values so a test can check normalization.
func trimmedProfileRequest() *v1.UpdateUserRequest {
	return &v1.UpdateUserRequest{FirstName: "  Ada  ", LastName: "  Lovelace  ", Phone: "  713-555-0100  "}
}

// TestUpdateUserCreatesAProfileForANewCaller checks the first write inserts
// rather than updates. An update-only implementation passes every fake-backed
// test and fails here.
func TestUpdateUserCreatesAProfileForANewCaller(t *testing.T) {
	server, db, id := newProfileServer(t)

	updated, err := server.UpdateUser(dbCaller(id), connect.NewRequest(trimmedProfileRequest()))
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	want := &v1.User{
		Id: id.String(), Email: id.String() + "@rice.edu",
		FirstName: "Ada", LastName: "Lovelace", Phone: "713-555-0100",
	}
	if diff := cmp.Diff(want, updated.Msg.User, protocmp.Transform()); diff != "" {
		t.Errorf("UpdateUser() mismatch (-want +got):\n%s", diff)
	}

	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM users WHERE id=$1`, id).Scan(&rows); err != nil {
		t.Fatalf("count profiles: %v", err)
	}
	if rows != 1 {
		t.Errorf("stored profiles = %d, want exactly 1", rows)
	}
}

// TestUpdateUserStoresNormalizedFields checks the trimming reaches the database,
// not just the response.
func TestUpdateUserStoresNormalizedFields(t *testing.T) {
	server, db, id := newProfileServer(t)

	if _, err := server.UpdateUser(dbCaller(id), connect.NewRequest(trimmedProfileRequest())); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	var stored []string
	var email, firstName, lastName, phone string
	if err := db.QueryRow(`SELECT email, first_name, last_name, phone FROM users WHERE id=$1`, id).
		Scan(&email, &firstName, &lastName, &phone); err != nil {
		t.Fatalf("read profile: %v", err)
	}
	stored = []string{email, firstName, lastName, phone}
	want := []string{id.String() + "@rice.edu", "Ada", "Lovelace", "713-555-0100"}
	if diff := cmp.Diff(want, stored); diff != "" {
		t.Errorf("stored profile mismatch (-want +got):\n%s", diff)
	}
}

// TestUpdateUserClearsAnOptionalField checks a blank phone removes the stored
// value rather than being rejected or ignored.
func TestUpdateUserClearsAnOptionalField(t *testing.T) {
	server, db, id := newProfileServer(t)
	request := trimmedProfileRequest()
	if _, err := server.UpdateUser(dbCaller(id), connect.NewRequest(request)); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	request.Phone = "   "
	if _, err := server.UpdateUser(dbCaller(id), connect.NewRequest(request)); err != nil {
		t.Fatalf("UpdateUser() clearing the phone error = %v", err)
	}

	var phone string
	if err := db.QueryRow(`SELECT phone FROM users WHERE id=$1`, id).Scan(&phone); err != nil {
		t.Fatalf("read phone: %v", err)
	}
	if phone != "" {
		t.Errorf("stored phone = %q, want it cleared", phone)
	}
}

// TestGetUserAcceptsADifferentlyCasedSelfID checks UUID comparison is not
// string comparison, so a client that upper-cases the ID still reads itself and
// keeps its own contact details.
func TestGetUserAcceptsADifferentlyCasedSelfID(t *testing.T) {
	server, _, id := newProfileServer(t)
	if _, err := server.UpdateUser(dbCaller(id), connect.NewRequest(trimmedProfileRequest())); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	upper := strings.ToUpper(id.String())
	got, err := server.GetUser(dbCaller(id), connect.NewRequest(&v1.GetUserRequest{UserId: &upper}))
	if err != nil {
		t.Fatalf("GetUser(%q) error = %v", upper, err)
	}

	if got.Msg.User.Email != id.String()+"@rice.edu" {
		t.Errorf("email = %q, want the caller's own, which only self-recognition reveals", got.Msg.User.Email)
	}
}

// TestRideListsFilterAndOrderInSQL checks real SQL discovery filters and personal ride history.
func TestRideListsFilterAndOrderInSQL(t *testing.T) {
	f := newDBMutationFixture(t)
	base := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
	ids := make([]string, 4)
	for i := range ids {
		f.request.DepartureTime = timestamppb.New(base.Add(time.Duration(i) * time.Hour))
		ids[i] = f.create(t)
	}
	if _, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: ids[0]})); err != nil {
		t.Fatal(err)
	}
	if _, err := f.server.CancelRide(dbCaller(f.owner), connect.NewRequest(&v1.CancelRideRequest{Id: ids[3]})); err != nil {
		t.Fatal(err)
	}
	owned, err := f.server.ListMyRides(dbCaller(f.owner), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]string, len(owned.Msg.Rides))
	for i, ride := range owned.Msg.Rides {
		gotIDs[i] = ride.Id
	}
	if diff := cmp.Diff([]string{ids[3], ids[2], ids[1], ids[0]}, gotIDs); diff != "" {
		t.Errorf("owner history order mismatch (-want +got):\n%s", diff)
	}
	if got := owned.Msg.Rides[0].Status; got != v1.RideStatus_RIDE_STATUS_CANCELLED {
		t.Errorf("cancelled ride status = %v, want it retained in history", got)
	}
	mine, err := f.server.ListMyRides(dbCaller(f.first), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil {
		t.Fatalf("ListMyRides() for the passenger error = %v", err)
	}
	if len(mine.Msg.Rides) != 1 || mine.Msg.Rides[0].Id != ids[0] {
		t.Errorf("passenger history = %d rides, want only the one they joined", len(mine.Msg.Rides))
	}
	other, err := f.server.ListMyRides(dbCaller(f.second), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil {
		t.Fatalf("ListMyRides() for an unrelated user error = %v", err)
	}
	if len(other.Msg.Rides) != 0 {
		t.Errorf("unrelated history = %d rides, want none", len(other.Msg.Rides))
	}
	listed, err := f.server.ListRides(dbCaller(f.second), connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &f.request.DepartureLocationId, ArrivalLocationId: &f.request.ArrivalLocationId}))
	if err != nil {
		t.Fatalf("ListRides() error = %v", err)
	}
	if len(listed.Msg.Rides) != 3 {
		t.Fatalf("discovery returned %d rides, want the 3 active ones", len(listed.Msg.Rides))
	}
	// Discovery runs the real contact-visibility SQL, so a stranger browsing rides
	// must not receive anyone's contact details.
	for _, ride := range listed.Msg.Rides {
		assertRPCContactRedacted(t, ride.Owner, f.owner)
		for _, rider := range ride.Riders {
			if rider.Id == f.second.String() {
				continue // the caller always sees their own contacts
			}
			assertContactsHidden(t, rider)
		}
	}
	wrong, err := f.server.ListRides(dbCaller(f.second), connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &f.request.ArrivalLocationId}))
	if err != nil {
		t.Fatalf("ListRides() with a non-matching filter error = %v", err)
	}
	if len(wrong.Msg.Rides) != 0 {
		t.Errorf("non-matching filter returned %d rides, want none", len(wrong.Msg.Rides))
	}
}

// TestSchemaRejectsInvalidRideLocations checks adapter error mapping and schema rejection.
func TestSchemaRejectsInvalidRideLocations(t *testing.T) {
	f := newDBMutationFixture(t)
	f.request.DepartureLocationId = uuid.NewString()
	_, err := f.server.CreateRide(dbCaller(f.owner), connect.NewRequest(f.request))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid location = %v", err)
	}
	_, err = f.db.Exec(`INSERT INTO rides (departure_time, departure_location_id, arrival_location_id, owner_id, capacity) VALUES (now() + interval '1 day', $1, $1, $2, 2)`, f.request.ArrivalLocationId, f.owner)
	if err == nil {
		t.Fatal("same-location insert succeeded")
	}
}

// TestRideLifecyclePersistsEachStep exercises membership, capacity, authorization, and cancellation against Postgres.
func TestRideLifecyclePersistsEachStep(t *testing.T) {
	f := newDBMutationFixture(t)
	id := f.create(t)

	// Each step builds on the one before, so these subtests share a fixture and
	// run in order. They assert what the database holds, not what the response
	// claims; the responses themselves are covered by the handler tests.
	t.Run("create seats the owner", func(t *testing.T) {
		if got := f.occupants(t, id); got != 1 {
			t.Errorf("occupants = %d, want the owner alone", got)
		}
	})

	t.Run("a rejected capacity shrink changes nothing", func(t *testing.T) {
		if _, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id})); err != nil {
			t.Fatalf("JoinRide() error = %v", err)
		}

		_, err := f.server.UpdateRide(dbCaller(f.owner), f.update(id, 1))
		if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
			t.Fatalf("UpdateRide() code = %v, want %v", got, connect.CodeFailedPrecondition)
		}
		var capacity int
		if err := f.db.QueryRow(`SELECT capacity FROM rides WHERE id=$1`, id).Scan(&capacity); err != nil {
			t.Fatalf("read capacity: %v", err)
		}
		if capacity != 2 {
			t.Errorf("stored capacity = %d, want the rejected shrink to leave 2", capacity)
		}
	})

	t.Run("leaving deletes the occupant row", func(t *testing.T) {
		left, err := f.server.LeaveRide(dbCaller(f.first), connect.NewRequest(&v1.LeaveRideRequest{RideId: id}))
		if err != nil {
			t.Fatalf("LeaveRide() error = %v", err)
		}
		if len(left.Msg.Ride.Riders) != 1 {
			t.Errorf("riders = %d, want only the owner", len(left.Msg.Ride.Riders))
		}
		if got := f.occupants(t, id); got != 1 {
			t.Errorf("occupants = %d, want only the owner", got)
		}
	})

	t.Run("cancelling stores the new status", func(t *testing.T) {
		cancelled, err := f.server.CancelRide(dbCaller(f.owner), connect.NewRequest(&v1.CancelRideRequest{Id: id}))
		if err != nil {
			t.Fatalf("CancelRide() error = %v", err)
		}
		if cancelled.Msg.Ride.Status != v1.RideStatus_RIDE_STATUS_CANCELLED {
			t.Errorf("response status = %v, want cancelled", cancelled.Msg.Ride.Status)
		}
		var status string
		if err := f.db.QueryRow(`SELECT status FROM rides WHERE id=$1`, id).Scan(&status); err != nil {
			t.Fatalf("read status: %v", err)
		}
		if status != "cancelled" {
			t.Errorf("stored status = %q, want %q", status, "cancelled")
		}
	})

	// Both prerequisites are read from real rows, so neither can be set up with
	// a fake.
	t.Run("posting needs a reachable owner", func(t *testing.T) {
		if _, err := f.db.Exec(`UPDATE users SET phone='' WHERE id=$1`, f.second); err != nil {
			t.Fatalf("clear phone: %v", err)
		}
		_, err := f.server.CreateRide(dbCaller(f.second), connect.NewRequest(f.request))
		if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
			t.Errorf("CreateRide() without a phone: code = %v, want %v", got, connect.CodeFailedPrecondition)
		}

		if _, err := f.db.Exec(`DELETE FROM users WHERE id=$1`, f.second); err != nil {
			t.Fatalf("delete profile: %v", err)
		}
		_, err = f.server.CreateRide(dbCaller(f.second), connect.NewRequest(f.request))
		if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
			t.Errorf("CreateRide() without a profile: code = %v, want %v", got, connect.CodeFailedPrecondition)
		}
	})
}
