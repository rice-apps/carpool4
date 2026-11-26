package rpc_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/auth"
	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
)

// Every test in this package shares one PostgreSQL container and truncates the
// fixture tables between tests, so none of them calls t.Parallel.
func TestMain(m *testing.M) { os.Exit(dbtest.Run(m)) }

// dbMutationFixture holds a database, verified callers, and a reusable ride request.
//
// Fields:
//   - db: the disposable database used for assertions and setup.
//   - server: the RPC server backed by that database.
//   - owner: the user who creates and manages the ride.
//   - first: one competing rider in concurrency tests.
//   - second: the other competing rider in concurrency tests.
//   - request: a valid ride creation request reused by tests.
type dbMutationFixture struct {
	db                   *sql.DB
	server               *rpc.Server
	owner, first, second uuid.UUID
	request              *carpoolv1.CreateRideRequest
}

// occupants returns how many people the stored ride seats.
func (f dbMutationFixture) occupants(t *testing.T, rideID string) int {
	t.Helper()
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM ride_occupants WHERE ride_id=$1`, rideID).Scan(&count); err != nil {
		t.Fatalf("count occupants of %s: %v", rideID, err)
	}
	return count
}

// newDBMutationFixture returns a disposable database, three users, two locations,
// an RPC server, and a valid ride request. Setup failures stop t.
func newDBMutationFixture(t *testing.T) dbMutationFixture {
	t.Helper()
	f := dbMutationFixture{db: dbtest.DB(t), owner: uuid.New(), first: uuid.New(), second: uuid.New()}
	for _, id := range []uuid.UUID{f.owner, f.first, f.second} {
		if _, err := f.db.Exec(`INSERT INTO auth.users (id) VALUES ($1)`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := f.db.Exec(`INSERT INTO users (id, first_name, last_name, email, phone) VALUES ($1, 'Test', 'User', $2, '7135550100')`, id, id.String()+"@rice.edu"); err != nil {
			t.Fatal(err)
		}
	}
	departure, arrival := uuid.New(), uuid.New()
	if _, err := f.db.Exec(`INSERT INTO locations (id, title, address) VALUES ($1, 'Departure', $3), ($2, 'Arrival', $4)`, departure, arrival, departure.String(), arrival.String()); err != nil {
		t.Fatal(err)
	}
	f.request = &carpoolv1.CreateRideRequest{DepartureTime: timestamppb.New(time.Now().Add(24 * time.Hour)), DepartureLocationId: departure.String(), ArrivalLocationId: arrival.String(), Capacity: 2}
	f.server = newRPCServer(postgres.NewDatabase(f.db))
	return f
}

// dbCaller returns a verified context for id using the fixture email convention.
func dbCaller(id uuid.UUID) context.Context {
	return auth.WithUser(context.Background(), auth.User{ID: id, Email: id.String() + "@rice.edu"})
}

// create returns the new ride's ID string after f's owner creates it.
// It fails t if creation fails or the owner is missing from the riders.
func (f dbMutationFixture) create(t *testing.T) string {
	t.Helper()
	r, err := f.server.CreateRide(dbCaller(f.owner), connect.NewRequest(f.request))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Msg.Ride.Riders) != 1 || r.Msg.Ride.Riders[0].Id != f.owner.String() {
		t.Fatalf("missing owner occupant: %v", r.Msg.Ride)
	}
	return r.Msg.Ride.Id
}

// update returns a request for ride id with f's route and the given capacity.
func (f dbMutationFixture) update(id string, capacity int32) *connect.Request[carpoolv1.UpdateRideRequest] {
	return connect.NewRequest(&carpoolv1.UpdateRideRequest{Id: id, DepartureTime: f.request.DepartureTime, DepartureLocationId: f.request.DepartureLocationId, ArrivalLocationId: f.request.ArrivalLocationId, Capacity: capacity})
}
