package rpc_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

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

// TestConcurrentJoinsClaimTheLastSeatOnce checks two joins for one seat yield one commit and one conflict.
// A fresh retry must then see the full ride rather than repeating the stale decision.
func TestConcurrentJoinsClaimTheLastSeatOnce(t *testing.T) {
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
	conflicted := assertOneSuccessOneConflict(t, <-results, <-results)
	if occupants := f.occupants(t, id); occupants != 2 {
		t.Errorf("occupants = %d, want the ride full at 2", occupants)
	}
	// Retrying in a fresh transaction must now see the ride as full, not as a
	// conflict: the race is over and the seat is gone.
	_, retryErr := f.server.JoinRide(dbCaller(conflicted), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
	if got := connect.CodeOf(retryErr); got != connect.CodeFailedPrecondition {
		t.Errorf("retry after conflict: code = %v, want %v", got, connect.CodeFailedPrecondition)
	}
	if want := "failed_precondition: this ride is full"; retryErr.Error() != want {
		t.Errorf("retry after conflict: error = %q, want %q", retryErr, want)
	}
}

// TestConcurrentJoinAndCapacityReductionSerialize checks serializable join versus capacity update.
// The persisted seat count and capacity must match one complete ordering.
func TestConcurrentJoinAndCapacityReductionSerialize(t *testing.T) {
	f := newDBMutationFixture(t)
	id := f.create(t)
	barrier := newMutationReadBarrier(t)
	server := newRPCServer(barrierMutationDatabase{Database: postgres.NewDatabase(f.db), barrier: barrier})
	results := make(chan mutationResult, 2)
	go func() {
		ctx, cancel := context.WithTimeout(dbCaller(f.first), 10*time.Second)
		defer cancel()
		_, err := server.JoinRide(ctx, connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
		results <- mutationResult{who: f.first, err: err}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(dbCaller(f.owner), 10*time.Second)
		defer cancel()
		_, err := server.UpdateRide(ctx, f.update(id, 1))
		results <- mutationResult{who: f.owner, err: err}
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
// assertOneSuccessOneConflict checks that exactly one of two racing mutations
// committed and the other was told to retry, and returns the caller that lost.
func assertOneSuccessOneConflict(t *testing.T, results ...mutationResult) uuid.UUID {
	t.Helper()
	var conflicted uuid.UUID
	successes, conflicts := 0, 0
	for _, result := range results {
		switch {
		case result.err == nil:
			successes++
		case connect.CodeOf(result.err) == connect.CodeAborted:
			conflicts++
			conflicted = result.who
		default:
			t.Fatalf("mutation error = %v, want success or transaction conflict", result.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("racing mutations gave %d successes and %d conflicts, want one of each", successes, conflicts)
	}
	return conflicted
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

// TestFailedPostWriteLoadRollsBackCreate checks response assembly failure rolls back both rows.
func TestFailedPostWriteLoadRollsBackCreate(t *testing.T) {
	f := newDBMutationFixture(t)
	server := newRPCServer(failingLoadDatabase{Database: postgres.NewDatabase(f.db)})
	_, err := server.CreateRide(dbCaller(f.owner), connect.NewRequest(f.request))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("code = %v, want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, errInjectedLoad) {
		t.Errorf("error = %v, want it to wrap the injected load failure", err)
	}
	var rides, occupants int
	if err := f.db.QueryRow(`SELECT (SELECT count(*) FROM rides), (SELECT count(*) FROM ride_occupants)`).Scan(&rides, &occupants); err != nil {
		t.Fatal(err)
	}
	if rides != 0 || occupants != 0 {
		t.Fatalf("failed creation persisted: rides=%d occupants=%d", rides, occupants)
	}
}

// TestJoinWithMissingProfileReportsSanitizedInternal checks raw foreign-key diagnostics remain private on the wire.
// A failed join must also leave occupancy unchanged.
func TestJoinWithMissingProfileReportsSanitizedInternal(t *testing.T) {
	// An occupant foreign-key failure is a server fault, not an invalid ride
	// write, so JoinRide must report Internal while keeping the cause for logs.
	f := newDBMutationFixture(t)
	id := f.create(t)
	if _, err := f.db.Exec(`DELETE FROM users WHERE id=$1`, f.first); err != nil {
		t.Fatal(err)
	}

	t.Run("RPC returns sanitized internal", func(t *testing.T) {
		_, err := f.server.JoinRide(dbCaller(f.first), connect.NewRequest(&v1.JoinRideRequest{RideId: id}))
		if got := connect.CodeOf(err); got != connect.CodeInternal {
			t.Errorf("code = %v, want %v", got, connect.CodeInternal)
		}
		if want := "internal: internal server error"; err.Error() != want {
			t.Errorf("error = %q, want the sanitized %q", err, want)
		}
		var wireError *connect.Error
		if !errors.As(err, &wireError) {
			t.Fatalf("error = %v, want a *connect.Error", err)
		}
		if want := "internal server error"; wireError.Message() != want {
			t.Errorf("wire message = %q, want %q", wireError.Message(), want)
		}
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) {
			t.Fatalf("error = %v, want it to carry a *pgconn.PgError for logs", err)
		}
		if pgError.Code != "23503" {
			t.Errorf("SQLSTATE = %q, want a foreign-key violation (23503)", pgError.Code)
		}
		if occupants := f.occupants(t, id); occupants != 1 {
			t.Fatalf("failed join persisted: %d occupants", occupants)
		}
	})
}
