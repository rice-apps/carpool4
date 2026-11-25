package rpc_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
	"github.com/rice-apps/carpool4/backend/internal/testutil"
)

// TestMain runs this package's tests with the disposable database harness.
func TestMain(m *testing.M) { os.Exit(testutil.Run(m)) }

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
	request              *v1.CreateRideRequest
}

// newDBMutationFixture returns a disposable database, three users, two locations,
// an RPC server, and a valid ride request. Setup failures stop t.
func newDBMutationFixture(t *testing.T) dbMutationFixture {
	t.Helper()
	f := dbMutationFixture{db: testutil.SetupTestDB(t), owner: uuid.New(), first: uuid.New(), second: uuid.New()}
	for _, id := range []uuid.UUID{f.owner, f.first, f.second} {
		if _, err := f.db.Exec(`INSERT INTO auth_users (id) VALUES ($1)`, id); err != nil {
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
	f.request = &v1.CreateRideRequest{DepartureTime: timestamppb.New(time.Now().Add(24 * time.Hour)), DepartureLocationId: departure.String(), ArrivalLocationId: arrival.String(), Capacity: 2}
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
func (f dbMutationFixture) update(id string, capacity int32) *connect.Request[v1.UpdateRideRequest] {
	return connect.NewRequest(&v1.UpdateRideRequest{Id: id, DepartureTime: f.request.DepartureTime, DepartureLocationId: f.request.DepartureLocationId, ArrivalLocationId: f.request.ArrivalLocationId, Capacity: capacity})
}

// mutationReadBarrier pauses concurrent mutations after their ride reads.
//
// Fields:
//   - arrived: signals that a mutation reached its ride read.
//   - release: signals waiting mutations to continue.
//   - once: ensures release is signaled only once.
type mutationReadBarrier struct {
	arrived chan struct{}
	release chan struct{}
	once    sync.Once
}

// newMutationReadBarrier returns a two-reader barrier that t releases on cleanup.
func newMutationReadBarrier(t *testing.T) *mutationReadBarrier {
	t.Helper()
	b := &mutationReadBarrier{arrived: make(chan struct{}, 2), release: make(chan struct{})}
	t.Cleanup(func() { b.once.Do(func() { close(b.release) }) })
	return b
}

// wait holds a successful ride read until release or ctx cancellation.
// It returns nil after release or the context error after cancellation.
func (b *mutationReadBarrier) wait(ctx context.Context) error {
	select {
	case <-b.release:
		return nil
	default:
	}
	select {
	case b.arrived <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseWhenArrived frees the mutations after want ride reads arrive.
// It fails t if they do not arrive within five seconds.
func (b *mutationReadBarrier) releaseWhenArrived(t *testing.T, want int) {
	t.Helper()
	for range want {
		select {
		case <-b.arrived:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %d mutation reads", want)
		}
	}
	b.once.Do(func() { close(b.release) })
}

// barrierMutationDatabase wraps transactions so ride reads can be synchronized.
//
// Fields:
//   - Database: the underlying database used to begin real transactions.
//   - barrier: the shared pause point for ride reads.
type barrierMutationDatabase struct {
	app.Database
	barrier *mutationReadBarrier
}

// BeginTx returns a real transaction whose ride loads wait at the barrier,
// or returns the database error from starting the transaction with ctx.
func (s barrierMutationDatabase) BeginTx(ctx context.Context) (app.Transaction, error) {
	tx, err := s.Database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return barrierMutationTx{Transaction: tx, barrier: s.barrier}, nil
}

// barrierMutationTx delegates transaction work while intercepting ride repository access.
//
// Fields:
//   - Transaction: the real transaction used for delegated operations.
//   - barrier: the shared pause point passed to its ride repository.
type barrierMutationTx struct {
	app.Transaction
	barrier *mutationReadBarrier
}

// RidesRepo returns a repository that pauses successful ride loads.
func (tx barrierMutationTx) RidesRepo() app.RideRepository {
	return barrierRideRepository{RideRepository: tx.Transaction.RidesRepo(), barrier: tx.barrier}
}

// barrierRideRepository delegates storage and pauses after a successful GetRide.
//
// Fields:
//   - RideRepository: the underlying repository used for ride operations.
//   - barrier: the pause point reached after a successful ride load.
type barrierRideRepository struct {
	app.RideRepository
	barrier *mutationReadBarrier
}

// GetRide returns the ride named by id after a successful read reaches the
// barrier. Storage failures and ctx cancellation return errors.
func (r barrierRideRepository) GetRide(ctx context.Context, id uuid.UUID) (app.LoadedRide, error) {
	ride, err := r.RideRepository.GetRide(ctx, id)
	if err == nil {
		err = r.barrier.wait(ctx)
	}
	return ride, err
}

// mutationResult pairs a concurrent caller with its mutation error.
//
// Fields:
//   - who: the caller whose mutation completed.
//   - err: that mutation's error, or nil on success.
type mutationResult struct {
	who uuid.UUID
	err error
}

// TestMutationDBConcurrentLastSeat checks two joins for one seat yield one commit and one conflict.
// A fresh retry must then see the full ride rather than repeating the stale decision.
func TestMutationDBConcurrentLastSeat(t *testing.T) {
	f := newDBMutationFixture(t)
	id := f.create(t)
	barrier := newMutationReadBarrier(t)
	server := newRPCServer(barrierMutationDatabase{Database: postgres.NewDatabase(f.db), barrier: barrier})
	results := make(chan mutationResult, 2)
	for _, who := range []uuid.UUID{f.first, f.second} {
		go func() {
			ctx, cancel := context.WithTimeout(dbCaller(who), 10*time.Second)
			defer cancel()
			_, err := server.JoinRide(ctx, connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
			results <- mutationResult{who: who, err: err}
		}()
	}
	barrier.releaseWhenArrived(t, 2)
	successes, conflicts := 0, 0
	var conflicted uuid.UUID
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
		} else if connect.CodeOf(result.err) == connect.CodeAborted {
			conflicts++
			conflicted = result.who
		} else {
			t.Fatalf("join: %v", result.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", successes, conflicts)
	}
	var occupants int
	if err := f.db.QueryRow(`SELECT count(*) FROM ride_occupants WHERE ride_id=$1`, id).Scan(&occupants); err != nil {
		t.Fatal(err)
	}
	if occupants != 2 {
		t.Fatalf("capacity 2 has %d occupants", occupants)
	}
	if _, err := f.server.JoinRide(dbCaller(conflicted), connect.NewRequest(&v1.JoinRideRequest{RideId: id})); connect.CodeOf(err) != connect.CodeFailedPrecondition || err.Error() != "failed_precondition: this ride is full" {
		t.Fatalf("fresh retry after conflict = %v, want full ride", err)
	}
}

// TestMutationDBConcurrentJoinAndCapacityReduction checks serializable join versus capacity update.
// The persisted seat count and capacity must match one complete ordering.
func TestMutationDBConcurrentJoinAndCapacityReduction(t *testing.T) {
	f := newDBMutationFixture(t)
	id := f.create(t)
	barrier := newMutationReadBarrier(t)
	server := newRPCServer(barrierMutationDatabase{Database: postgres.NewDatabase(f.db), barrier: barrier})
	results := make(chan error, 2)
	go func() {
		ctx, cancel := context.WithTimeout(dbCaller(f.first), 10*time.Second)
		defer cancel()
		_, err := server.JoinRide(ctx, connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
		results <- err
	}()
	go func() {
		ctx, cancel := context.WithTimeout(dbCaller(f.owner), 10*time.Second)
		defer cancel()
		_, err := server.UpdateRide(ctx, f.update(id, 1))
		results <- err
	}()
	barrier.releaseWhenArrived(t, 2)
	assertOneSuccessOneConflict(t, <-results, <-results)

	var capacity, occupants int
	if err := f.db.QueryRow(`SELECT capacity,(SELECT count(*) FROM ride_occupants WHERE ride_id=rides.id) FROM rides WHERE id=$1`, id).Scan(&capacity, &occupants); err != nil {
		t.Fatal(err)
	}
	if !((capacity == 2 && occupants == 2) || (capacity == 1 && occupants == 1)) {
		t.Fatalf("non-serial result: capacity=%d occupants=%d", capacity, occupants)
	}
}

// assertOneSuccessOneConflict fails t unless two mutation errors contain one nil and one Aborted.
func assertOneSuccessOneConflict(t *testing.T, first, second error) {
	t.Helper()
	errors := []error{first, second}
	successes, conflicts := 0, 0
	for _, err := range errors {
		switch {
		case err == nil:
			successes++
		case connect.CodeOf(err) == connect.CodeAborted:
			conflicts++
		default:
			t.Fatalf("mutation error = %v, want success or transaction conflict", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("mutation errors = %v, want one success and one conflict", errors)
	}
}

var errInjectedLoad = errors.New("injected load failure")

// failingLoadDatabase injects a post-write ride-load failure into real transactions.
//
// Fields:
//   - Database: the underlying database used to begin real transactions.
type failingLoadDatabase struct{ app.Database }

// BeginTx returns a real transaction whose ride loads fail with errInjectedLoad,
// or returns the database error from starting the transaction with ctx.
func (s failingLoadDatabase) BeginTx(ctx context.Context) (app.Transaction, error) {
	tx, err := s.Database.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return failingLoadTx{Transaction: tx}, nil
}

// failingLoadTx delegates transaction work except ride repository access.
//
// Fields:
//   - Transaction: the real transaction used for delegated operations.
type failingLoadTx struct{ app.Transaction }

// RidesRepo returns a repository that fails every ride load.
func (tx failingLoadTx) RidesRepo() app.RideRepository {
	return failingGetRides{RideRepository: tx.Transaction.RidesRepo()}
}

// failingGetRides injects an error when the service assembles the response ride.
//
// Fields:
//   - RideRepository: the underlying repository for operations other than GetRide.
type failingGetRides struct{ app.RideRepository }

// GetRide returns the shared injected error instead of reading the requested UUID.
func (failingGetRides) GetRide(context.Context, uuid.UUID) (app.LoadedRide, error) {
	return app.LoadedRide{}, errInjectedLoad
}

// TestMutationDBPostWriteLoadFailureRollsBackCreate checks response assembly failure rolls back both rows.
func TestMutationDBPostWriteLoadFailureRollsBackCreate(t *testing.T) {
	f := newDBMutationFixture(t)
	server := newRPCServer(failingLoadDatabase{Database: postgres.NewDatabase(f.db)})
	_, err := server.CreateRide(dbCaller(f.owner), connect.NewRequest(f.request))
	if connect.CodeOf(err) != connect.CodeInternal || !errors.Is(err, errInjectedLoad) {
		t.Fatalf("post-write load failure = %v", err)
	}
	var rides, occupants int
	if err := f.db.QueryRow(`SELECT (SELECT count(*) FROM rides), (SELECT count(*) FROM ride_occupants)`).Scan(&rides, &occupants); err != nil {
		t.Fatal(err)
	}
	if rides != 0 || occupants != 0 {
		t.Fatalf("failed creation persisted: rides=%d occupants=%d", rides, occupants)
	}
}

// TestMutationDBMissingProfileInsertFailure checks raw foreign-key diagnostics remain private on the wire.
// A failed join must also leave occupancy unchanged.
func TestMutationDBMissingProfileInsertFailure(t *testing.T) {
	// Catches classifying occupant foreign-key failures as invalid ride writes:
	// this adapter operation historically returned the raw failure, which JoinRide
	// must expose as a generic Internal error with its diagnostic cause retained.
	f := newDBMutationFixture(t)
	id := f.create(t)
	if _, err := f.db.Exec(`DELETE FROM users WHERE id=$1`, f.first); err != nil {
		t.Fatal(err)
	}

	t.Run("RPC returns sanitized internal", func(t *testing.T) {
		_, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
		if connect.CodeOf(err) != connect.CodeInternal || err.Error() != "internal: internal server error" {
			t.Fatalf("JoinRide missing profile = %v, want sanitized Internal", err)
		}
		var wireError *connect.Error
		if !errors.As(err, &wireError) || wireError.Message() != "internal server error" {
			t.Fatalf("unsafe wire message: %v", wireError)
		}
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != "23503" || !errors.Is(err, pgError) {
			t.Fatalf("JoinRide discarded PostgreSQL cause: %v", err)
		}
		var occupants int
		if err := f.db.QueryRow(`SELECT count(*) FROM ride_occupants WHERE ride_id=$1`, id).Scan(&occupants); err != nil {
			t.Fatal(err)
		}
		if occupants != 1 {
			t.Fatalf("failed join persisted: %d occupants", occupants)
		}
	})
}
