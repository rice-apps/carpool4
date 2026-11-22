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

// TestDBUpdateUserCreatesFirstProfile checks a verified auth user can create a normalized profile.
func TestDBUpdateUserCreatesFirstProfile(t *testing.T) {
	// UpdateUser must insert a normalized profile for a verified caller who
	// exists only in auth_users; an update-only implementation must fail here.
	db := testutil.SetupTestDB(t)
	id := uuid.New()
	if _, err := db.Exec(`INSERT INTO auth_users (id) VALUES ($1)`, id); err != nil {
		t.Fatal(err)
	}
	server := newRPCServer(postgres.NewDatabase(db))
	updated, err := server.UpdateUser(dbCaller(id), connect.NewRequest(&v1.UpdateUserRequest{
		FirstName: "  Ada  ", LastName: "  Lovelace  ", Phone: "  713-555-0100  ",
	}))
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
}

// TestDBUserNormalizationAndEquivalentSelfID checks normalized writes, cleared optional fields, and UUID equality.
func TestDBUserNormalizationAndEquivalentSelfID(t *testing.T) {
	// The adapter must persist normalized fields, including optional values being cleared.
	f := newDBMutationFixture(t)
	request := &v1.UpdateUserRequest{FirstName: "  Ada  ", LastName: "  Lovelace  ", Phone: "  713-555-0100  "}
	updated, err := f.server.UpdateUser(dbCaller(f.owner), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.User.FirstName != "Ada" || updated.Msg.User.LastName != "Lovelace" || updated.Msg.User.Phone != "713-555-0100" {
		t.Fatalf("normalization = %v", updated.Msg.User)
	}
	upper := strings.ToUpper(f.owner.String())
	got, err := f.server.GetUser(dbCaller(f.owner), connect.NewRequest(&v1.GetUserRequest{UserId: &upper}))
	if err != nil || got.Msg.User.Email != f.owner.String()+"@rice.edu" {
		t.Fatalf("equivalent self ID = %v, %v", got, err)
	}
	request.Phone = "   "
	if _, err := f.server.UpdateUser(dbCaller(f.owner), connect.NewRequest(request)); err != nil {
		t.Fatal(err)
	}
	var phone string
	if err := f.db.QueryRow(`SELECT phone FROM users WHERE id=$1`, f.owner).Scan(&phone); err != nil {
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
	// Retains the old core lifecycle checks against real persistence after moving ownership to RPCs.
	f := newDBMutationFixture(t)
	id := f.create(t)
	check := func(err error, code connect.Code) {
		t.Helper()
		if connect.CodeOf(err) != code {
			t.Fatalf("error = %v, want %v", err, code)
		}
	}
	_, err := f.server.JoinRide(dbCaller(f.owner), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
	check(err, connect.CodeFailedPrecondition)
	_, err = f.server.LeaveRide(dbCaller(f.owner), connect.NewRequest(&v1.LeaveRideRequest{RideId: id}))
	check(err, connect.CodeFailedPrecondition)
	if _, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id})); err != nil {
		t.Fatal(err)
	}
	_, err = f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
	check(err, connect.CodeFailedPrecondition)
	_, err = f.server.JoinRide(dbCaller(f.second), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
	check(err, connect.CodeFailedPrecondition)
	_, err = f.server.UpdateRide(dbCaller(f.owner), f.update(id, 1))
	check(err, connect.CodeFailedPrecondition)
	var capacity int
	if err := f.db.QueryRow(`SELECT capacity FROM rides WHERE id=$1`, id).Scan(&capacity); err != nil || capacity != 2 {
		t.Fatalf("failed shrink capacity=%d err=%v", capacity, err)
	}
	_, err = f.server.UpdateRide(dbCaller(f.second), f.update(id, 3))
	check(err, connect.CodePermissionDenied)
	updated, err := f.server.UpdateRide(dbCaller(f.owner), f.update(id, 3))
	if err != nil || updated.Msg.Ride.Capacity != 3 {
		t.Fatalf("update = %v, %v", updated, err)
	}
	_, err = f.server.LeaveRide(dbCaller(f.second), connect.NewRequest(&v1.LeaveRideRequest{RideId: id}))
	check(err, connect.CodeFailedPrecondition)
	left, err := f.server.LeaveRide(dbCaller(f.first), connect.NewRequest(&v1.LeaveRideRequest{RideId: id}))
	if err != nil || len(left.Msg.Ride.Riders) != 1 {
		t.Fatalf("leave = %v, %v", left, err)
	}
	_, err = f.server.GetRide(dbCaller(f.owner), connect.NewRequest(&v1.GetRideRequest{Id: uuid.NewString()}))
	check(err, connect.CodeNotFound)
	_, err = f.server.CancelRide(dbCaller(f.second), connect.NewRequest(&v1.CancelRideRequest{Id: id}))
	check(err, connect.CodePermissionDenied)
	cancelled, err := f.server.CancelRide(dbCaller(f.owner), connect.NewRequest(&v1.CancelRideRequest{Id: id}))
	if err != nil || cancelled.Msg.Ride.Status != v1.RideStatus_RIDE_STATUS_CANCELLED {
		t.Fatalf("cancel = %v, %v", cancelled, err)
	}
	_, err = f.server.JoinRide(dbCaller(f.second), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
	check(err, connect.CodeFailedPrecondition)
	_, err = f.server.UpdateRide(dbCaller(f.owner), f.update(id, 3))
	check(err, connect.CodeFailedPrecondition)
	if _, err := f.db.Exec(`UPDATE users SET phone='' WHERE id=$1`, f.second); err != nil {
		t.Fatal(err)
	}
	_, err = f.server.CreateRide(dbCaller(f.second), connect.NewRequest(f.request))
	check(err, connect.CodeFailedPrecondition)
	if _, err := f.db.Exec(`DELETE FROM users WHERE id=$1`, f.second); err != nil {
		t.Fatal(err)
	}
	_, err = f.server.CreateRide(dbCaller(f.second), connect.NewRequest(f.request))
	check(err, connect.CodeFailedPrecondition)
}
