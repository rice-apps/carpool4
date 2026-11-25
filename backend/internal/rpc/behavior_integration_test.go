package rpc_test

import (
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/testutil"
)

// TestDBPastSharedRideGrantsSelectiveContactVisibility checks historical co-rider access.
// A past cancelled ride reveals shared users' contacts, but unrelated passengers remain redacted.
func TestDBPastSharedRideGrantsSelectiveContactVisibility(t *testing.T) {
	// A past, cancelled ride must grant contact access across other ride responses,
	// but must not grant access to unrelated passengers on those rides.
	f := newDBMutationFixture(t)
	f.request.Capacity = 3
	id := f.create(t)
	if _, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id})); err != nil {
		t.Fatal(err)
	}
	owner := f.owner.String()
	before, err := f.server.GetUser(dbCaller(f.second), connect.NewRequest(&v1.GetUserRequest{UserId: &owner}))
	if err != nil {
		t.Fatal(err)
	}
	assertRPCContactRedacted(t, before.Msg.User, f.owner)
	past := uuid.New()
	if _, err := f.db.Exec(`INSERT INTO rides (id, departure_time, departure_location_id, arrival_location_id, owner_id, capacity, status) VALUES ($1, now() - interval '1 year', $2, $3, $4, 2, 'cancelled')`, past, f.request.DepartureLocationId, f.request.ArrivalLocationId, f.owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`INSERT INTO ride_occupants (ride_id, user_id) VALUES ($1, $2), ($1, $3)`, past, f.owner, f.second); err != nil {
		t.Fatal(err)
	}
	user, err := f.server.GetUser(dbCaller(f.second), connect.NewRequest(&v1.GetUserRequest{UserId: &owner}))
	if err != nil || user.Msg.User.Email != owner+"@rice.edu" || user.Msg.User.Phone != "7135550100" {
		t.Fatalf("past shared user = %v, %v", user, err)
	}
	ride, err := f.server.GetRide(dbCaller(f.second), connect.NewRequest(&v1.GetRideRequest{Id: id}))
	if err != nil {
		t.Fatal(err)
	}
	if ride.Msg.Ride.Owner.Email != owner+"@rice.edu" {
		t.Fatal("past ride did not grant owner contact")
	}
	for _, rider := range ride.Msg.Ride.Riders {
		if rider.Id == f.first.String() {
			assertRPCContactRedacted(t, rider, f.first)
		}
	}
	mine, err := f.server.ListMyRides(dbCaller(f.first), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil || len(mine.Msg.Rides) != 1 || mine.Msg.Rides[0].Owner.Email != owner+"@rice.edu" {
		t.Fatalf("participant list = %v, %v", mine, err)
	}
	participant, err := f.server.GetRide(dbCaller(f.first), connect.NewRequest(&v1.GetRideRequest{Id: id}))
	if err != nil || participant.Msg.Ride.Owner.Email != owner+"@rice.edu" {
		t.Fatalf("participant detail = %v, %v", participant, err)
	}
}

// TestDBUpdateUserCreatesAndNormalizesProfile checks the first write for a caller who
// exists only in auth_users inserts rather than updates, that stored fields are
// normalized, that an optional value can be cleared, and that a differently-cased
// self ID resolves to the same profile.
func TestDBUpdateUserCreatesAndNormalizesProfile(t *testing.T) {
	db := testutil.SetupTestDB(t)
	id := uuid.New()
	if _, err := db.Exec(`INSERT INTO auth_users (id) VALUES ($1)`, id); err != nil {
		t.Fatal(err)
	}
	server := newRPCServer(postgres.NewDatabase(db))

	request := &v1.UpdateUserRequest{FirstName: "  Ada  ", LastName: "  Lovelace  ", Phone: "  713-555-0100  "}
	updated, err := server.UpdateUser(dbCaller(id), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Msg.User; got.Id != id.String() || got.Email != id.String()+"@rice.edu" || got.FirstName != "Ada" || got.LastName != "Lovelace" || got.Phone != "713-555-0100" {
		t.Fatalf("created profile = %v", got)
	}

	var persistedID uuid.UUID
	var email, firstName, lastName, phone string
	if err := db.QueryRow(`SELECT id, email, first_name, last_name, phone FROM users WHERE id=$1`, id).Scan(&persistedID, &email, &firstName, &lastName, &phone); err != nil {
		t.Fatal(err)
	}
	if persistedID != id || email != id.String()+"@rice.edu" || firstName != "Ada" || lastName != "Lovelace" || phone != "713-555-0100" {
		t.Fatalf("persisted profile: id=%s email=%q name=%q %q phone=%q", persistedID, email, firstName, lastName, phone)
	}

	upper := strings.ToUpper(id.String())
	got, err := server.GetUser(dbCaller(id), connect.NewRequest(&v1.GetUserRequest{UserId: &upper}))
	if err != nil || got.Msg.User.Email != id.String()+"@rice.edu" {
		t.Fatalf("equivalent self ID = %v, %v", got, err)
	}

	request.Phone = "   "
	if _, err := server.UpdateUser(dbCaller(id), connect.NewRequest(request)); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT phone FROM users WHERE id=$1`, id).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	if phone != "" {
		t.Fatalf("cleared phone = %q", phone)
	}
}

// TestDBListFiltersMembershipAndOrdering checks real SQL discovery filters and personal ride history.
func TestDBListFiltersMembershipAndOrdering(t *testing.T) {
	// Actual SQL must preserve membership, discovery filters, cancelled history,
	// and descending personal-history ordering; fake query argument checks cannot prove that.
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
	if len(owned.Msg.Rides) != 4 || owned.Msg.Rides[0].Id != ids[3] || owned.Msg.Rides[1].Id != ids[2] || owned.Msg.Rides[2].Id != ids[1] || owned.Msg.Rides[3].Id != ids[0] || owned.Msg.Rides[0].Status != v1.RideStatus_RIDE_STATUS_CANCELLED {
		t.Fatalf("owner history = %v", owned.Msg)
	}
	mine, err := f.server.ListMyRides(dbCaller(f.first), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil || len(mine.Msg.Rides) != 1 || mine.Msg.Rides[0].Id != ids[0] {
		t.Fatalf("passenger list = %v, %v", mine, err)
	}
	other, err := f.server.ListMyRides(dbCaller(f.second), connect.NewRequest(&v1.ListMyRidesRequest{}))
	if err != nil || len(other.Msg.Rides) != 0 {
		t.Fatalf("unrelated list = %v, %v", other, err)
	}
	listed, err := f.server.ListRides(dbCaller(f.second), connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &f.request.DepartureLocationId, ArrivalLocationId: &f.request.ArrivalLocationId}))
	if err != nil || len(listed.Msg.Rides) != 3 {
		t.Fatalf("discovery = %v, %v", listed, err)
	}
	// Discovery runs the real contact-visibility SQL, so a stranger browsing rides
	// must not receive anyone's contact details.
	for _, ride := range listed.Msg.Rides {
		assertRPCContactRedacted(t, ride.Owner, uuid.MustParse(ride.Owner.Id))
		for _, rider := range ride.Riders {
			if rider.Id == f.second.String() {
				continue // the caller always sees their own contacts
			}
			assertRPCContactRedacted(t, rider, uuid.MustParse(rider.Id))
		}
	}
	wrong, err := f.server.ListRides(dbCaller(f.second), connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &f.request.ArrivalLocationId}))
	if err != nil || len(wrong.Msg.Rides) != 0 {
		t.Fatalf("wrong location = %v, %v", wrong, err)
	}
}

// TestDBInvalidLocationAndSameLocationConstraint checks adapter error mapping and schema rejection.
func TestDBInvalidLocationAndSameLocationConstraint(t *testing.T) {
	// The adapter must classify foreign-key errors, and the schema must independently reject equal locations.
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

// TestDBRideLifecycle exercises membership, capacity, authorization, and cancellation against Postgres.
func TestDBRideLifecycle(t *testing.T) {
	// Only what needs real persistence. The business rules themselves are covered by
	// the fake-backed app tests, and their error-to-code mapping by errors_test.go;
	// what cannot be faked is whether the database actually holds what we claim.
	f := newDBMutationFixture(t)
	id := f.create(t)

	var occupants int
	if err := f.db.QueryRow(`SELECT count(*) FROM ride_occupants WHERE ride_id=$1 AND user_id=$2`, id, f.owner).Scan(&occupants); err != nil {
		t.Fatal(err)
	}
	if occupants != 1 {
		t.Fatalf("owner occupies %d seats after create, want 1", occupants)
	}

	if _, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id})); err != nil {
		t.Fatal(err)
	}

	// A rejected capacity shrink must leave the stored ride untouched.
	if _, err := f.server.UpdateRide(dbCaller(f.owner), f.update(id, 1)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("capacity shrink error = %v, want failed precondition", err)
	}
	var capacity int
	if err := f.db.QueryRow(`SELECT capacity FROM rides WHERE id=$1`, id).Scan(&capacity); err != nil || capacity != 2 {
		t.Fatalf("rejected shrink persisted capacity=%d err=%v", capacity, err)
	}

	// Leaving must actually delete the occupant row, not just return a ride.
	left, err := f.server.LeaveRide(dbCaller(f.first), connect.NewRequest(&v1.LeaveRideRequest{RideId: id}))
	if err != nil || len(left.Msg.Ride.Riders) != 1 {
		t.Fatalf("leave = %v, %v", left, err)
	}
	if err := f.db.QueryRow(`SELECT count(*) FROM ride_occupants WHERE ride_id=$1`, id).Scan(&occupants); err != nil {
		t.Fatal(err)
	}
	if occupants != 1 {
		t.Fatalf("after leave the ride holds %d occupants, want only the owner", occupants)
	}

	cancelled, err := f.server.CancelRide(dbCaller(f.owner), connect.NewRequest(&v1.CancelRideRequest{Id: id}))
	if err != nil || cancelled.Msg.Ride.Status != v1.RideStatus_RIDE_STATUS_CANCELLED {
		t.Fatalf("cancel = %v, %v", cancelled, err)
	}
	var status string
	if err := f.db.QueryRow(`SELECT status FROM rides WHERE id=$1`, id).Scan(&status); err != nil || status != "cancelled" {
		t.Fatalf("stored status = %q, err = %v", status, err)
	}

	// Both profile prerequisites answer the same way, and neither can be faked.
	if _, err := f.db.Exec(`UPDATE users SET phone='' WHERE id=$1`, f.second); err != nil {
		t.Fatal(err)
	}
	if _, err := f.server.CreateRide(dbCaller(f.second), connect.NewRequest(f.request)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("create without a phone = %v, want failed precondition", err)
	}
	if _, err := f.db.Exec(`DELETE FROM users WHERE id=$1`, f.second); err != nil {
		t.Fatal(err)
	}
	if _, err := f.server.CreateRide(dbCaller(f.second), connect.NewRequest(f.request)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("create without a profile = %v, want failed precondition", err)
	}
}
