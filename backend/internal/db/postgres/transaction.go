package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
)

// Database adapts a SQL connection pool to the application's transaction port.
//
// Fields:
//   - db: holds the pool used to begin application transactions.
type Database struct {
	db *sql.DB
}

var _ app.Database = (*Database)(nil)

// NewDatabase adapts db, an open *sql.DB pool, for application transactions.
// The caller closes the pool after those transactions finish.
func NewDatabase(db *sql.DB) *Database {
	return &Database{db: db}
}

// Transaction holds one SQL transaction and its bound sqlc query runner so all
// repository operations share the same snapshot and writes.
//
// Fields:
//   - tx: controls commit and rollback of the SQL transaction.
//   - queries: runs repository queries within that transaction.
type Transaction struct {
	tx      *sql.Tx
	queries *sqlc.Queries
}

// userRepository performs user queries within a Transaction.
//
// Fields:
//   - queries: runs user queries within the owning transaction.
type userRepository struct {
	queries *sqlc.Queries
}

// rideRepository performs ride queries within a Transaction.
//
// Fields:
//   - queries: runs ride queries within the owning transaction.
type rideRepository struct {
	queries *sqlc.Queries
}

// locationRepository performs location queries within a Transaction.
//
// Fields:
//   - queries: runs location queries within the owning transaction.
type locationRepository struct {
	queries *sqlc.Queries
}

var _ app.UserRepository = userRepository{}
var _ app.RideRepository = rideRepository{}
var _ app.LocationRepository = locationRepository{}

var _ app.Transaction = (*Transaction)(nil)

// BeginTx starts a Serializable transaction.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the SQL transaction.
//
// It returns an app.Transaction whose repositories share the transaction, or
// an error if creation fails.
func (d *Database) BeginTx(ctx context.Context) (app.Transaction, error) {
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	// Bind sqlc to this transaction so every repository sees the same snapshot.
	return &Transaction{tx: tx, queries: sqlc.New(tx)}, nil
}

// Users returns user storage scoped to this transaction.
func (t *Transaction) Users() app.UserRepository {
	return userRepository{queries: t.queries}
}

// Rides returns ride storage scoped to this transaction.
func (t *Transaction) Rides() app.RideRepository {
	return rideRepository{queries: t.queries}
}

// Locations returns location storage scoped to this transaction.
func (t *Transaction) Locations() app.LocationRepository {
	return locationRepository{queries: t.queries}
}

// Commit persists the transaction. A serialization abort returns
// app.ErrTransactionConflict while preserving the PostgreSQL cause.
func (t *Transaction) Commit() error {
	return classifyTransactionError(t.tx.Commit())
}

// Rollback abandons an open transaction. It also returns nil after a prior
// commit or rollback, so callers may safely defer it.
func (t *Transaction) Rollback() error {
	err := t.tx.Rollback()
	if err == sql.ErrTxDone {
		return nil
	}
	return err
}
