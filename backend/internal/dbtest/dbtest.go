// Package dbtest runs tests against a disposable PostgreSQL container.
//
// Tests need no configuration and no running database: Run starts a container,
// applies supabase/migrations verbatim, and tears it down afterwards. The only
// prerequisite is a running Docker daemon.
package dbtest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// fixtureTables are emptied between tests, ordered so foreign keys allow it.
var fixtureTables = []string{"ride_occupants", "rides", "users", "locations", "auth.users"}

// pool is the connection shared by every test in the package, set by Run.
var pool *sql.DB

// Run starts a PostgreSQL container, applies the schema, runs the package's
// tests against it and reports their exit status. Call it from TestMain.
func Run(m *testing.M) int {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("carpool"),
		postgres.WithUsername("carpool"),
		postgres.WithPassword("carpool"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, startupError(err))
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection string: %v\n", err)
		return 1
	}
	pool, err = sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		return 1
	}
	defer func() { _ = pool.Close() }()

	if err := applySchema(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "apply schema: %v\n", err)
		return 1
	}
	return m.Run()
}

// DB returns the package's database with the fixture tables emptied. Tests that
// use it cannot run in parallel with each other, since they share these tables.
func DB(t *testing.T) *sql.DB {
	t.Helper()
	if pool == nil {
		t.Fatal("dbtest.DB called without dbtest.Run in TestMain")
	}
	if _, err := pool.ExecContext(t.Context(), "TRUNCATE "+strings.Join(fixtureTables, ", ")+" CASCADE"); err != nil {
		t.Fatalf("reset fixtures: %v", err)
	}
	return pool
}

// applySchema creates the Supabase roles and auth schema this project's
// migrations expect, then runs every migration in filename order.
func applySchema(ctx context.Context, db *sql.DB) error {
	// Supabase supplies these; a bare PostgreSQL image does not. The migrations
	// grant to supabase_auth_admin unconditionally, so all three must exist.
	const supabaseObjects = `
		CREATE ROLE anon NOLOGIN;
		CREATE ROLE authenticated NOLOGIN;
		CREATE ROLE supabase_auth_admin NOLOGIN;
		CREATE SCHEMA auth;
		CREATE TABLE auth.users (id uuid PRIMARY KEY);`
	if _, err := db.ExecContext(ctx, supabaseObjects); err != nil {
		return fmt.Errorf("create Supabase objects: %w", err)
	}

	dir, err := migrationsDir()
	if err != nil {
		return err
	}
	names, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no migrations in %s", dir)
	}
	sort.Strings(names)
	for _, name := range names {
		statements, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(statements)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(name), err)
		}
	}
	return nil
}

// migrationsDir walks up from this file to the repository root.
func migrationsDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot locate the dbtest package")
	}
	for dir := filepath.Dir(file); dir != string(filepath.Separator); dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "supabase", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("supabase/migrations not found above " + file)
}

// startupError explains a stopped Docker daemon, which is the only setup this
// suite needs and the only failure a contributor is likely to hit.
func startupError(err error) string {
	if strings.Contains(strings.ToLower(err.Error()), "docker") {
		return "cannot reach Docker, which these tests need for a disposable PostgreSQL container.\n" +
			"Start Docker and re-run `go test ./...`.\n\n" +
			"underlying error: " + err.Error()
	}
	return "start PostgreSQL container: " + err.Error()
}
