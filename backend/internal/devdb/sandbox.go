// Package devdb owns development provisioning and restricted SQL test fixtures.
package devdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// OpenTestPool configures and verifies every physical connection, including pool replacements.
func OpenTestPool(ctx context.Context, dsn string) (*sql.DB, string, error) {
	if dsn == "" {
		return nil, "", fmt.Errorf("TEST_DATABASE_URL is missing; copy your pair's supplied .env to the repository root")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, "", fmt.Errorf("TEST_DATABASE_URL is invalid")
	}
	role, _, _ := strings.Cut(config.User, ".")
	if !isTestRole(role) {
		return nil, "", fmt.Errorf("TEST_DATABASE_URL must use a restricted pair test login")
	}
	db := stdlib.OpenDB(*config, stdlib.OptionAfterConnect(func(ctx context.Context, conn *pgx.Conn) error {
		var actual string
		var privileged bool
		if err := conn.QueryRow(ctx, `SELECT current_user, rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&actual, &privileged); err != nil {
			return err
		}
		if actual != role || privileged {
			return fmt.Errorf("test connection is not the assigned restricted login")
		}
		// Public supplies shared enum types; table grants still deny development-data access.
		_, err := conn.Exec(ctx, `SELECT pg_catalog.set_config('search_path', $1, false)`, pgx.Identifier{role}.Sanitize()+", public")
		return err
	}))
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	var assigned string
	err = db.QueryRowContext(ctx, `SELECT role_name FROM `+pgx.Identifier{role, "__carpool_test_environment"}.Sanitize()).Scan(&assigned)
	if err != nil {
		db.Close()
		return nil, "", fmt.Errorf("test environment is not provisioned for this pair: %w", err)
	}
	if assigned != role {
		db.Close()
		return nil, "", fmt.Errorf("test environment belongs to a different login")
	}
	return db, role, nil
}

// OpenSandbox keeps a cross-process lock until cleanup so tests cannot reset an active fixture.
func OpenSandbox(ctx context.Context, dsn string) (*sql.DB, func() error, error) {
	db, schema, err := OpenTestPool(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	lock, err := db.Conn(ctx)
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	if _, err = lock.ExecContext(ctx, `SELECT pg_advisory_lock(hashtextextended($1, 784218))`, schema); err != nil {
		lock.Close()
		db.Close()
		return nil, nil, err
	}
	reset := func(ctx context.Context) error {
		tables := []string{"ride_occupants", "rides", "users", "locations", "auth_users"}
		qualified := make([]string, 0, len(tables))
		for _, table := range tables {
			qualified = append(qualified, pgx.Identifier{schema, table}.Sanitize())
		}
		_, err := lock.ExecContext(ctx, "TRUNCATE "+strings.Join(qualified, ", "))
		return err
	}
	close := func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		resetErr := reset(cleanupCtx)
		_, unlockErr := lock.ExecContext(cleanupCtx, `SELECT pg_advisory_unlock(hashtextextended($1, 784218))`, schema)
		return errors.Join(resetErr, unlockErr, lock.Close(), db.Close())
	}
	if err := reset(ctx); err != nil {
		_ = close()
		return nil, nil, err
	}
	return db, close, nil
}
