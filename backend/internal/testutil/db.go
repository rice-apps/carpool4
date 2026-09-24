// Package testutil provisions disposable PostgreSQL databases with the real schema.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var server struct {
	once      sync.Once
	dsn       string
	err       error
	container *postgres.PostgresContainer
}

// Run executes m (*testing.M) and releases the package's PostgreSQL container.
// Call it from TestMain. It returns the test exit code, or 1 if container
// cleanup fails. A package shares one server; SetupTestDB gives each test a
// separate database.
func Run(m *testing.M) int {
	code := m.Run()
	// Stop the shared container after every test in this package finishes.
	if server.container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.container.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminate test postgres: %v\n", err)
			return 1
		}
	}
	return code
}

// startServer selects the package's PostgreSQL server and creates the roles
// required by migrations. TEST_DATABASE_URL must name a disposable server
// whose user can create databases and roles; otherwise a container is used.
// DATABASE_URL is ignored. The supplied database itself is never migrated or
// dropped. Setup failures are recorded in server.err.
func startServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// Use an explicit disposable server, or start one for this test package.
	server.dsn = os.Getenv("TEST_DATABASE_URL")
	if server.dsn == "" {
		server.container, server.err = postgres.Run(ctx, "postgres:16-alpine",
			postgres.WithDatabase("postgres"), postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"), postgres.BasicWaitStrategies())
		if server.err != nil {
			return
		}
		server.dsn, server.err = server.container.ConnectionString(ctx, "sslmode=disable")
		if server.err != nil {
			return
		}
	}
	// Create the Supabase roles expected by the migration files.
	db, err := sql.Open("pgx", server.dsn)
	if err != nil {
		server.err = err
		return
	}
	defer db.Close()
	// Serialize role creation across concurrently running Go test packages.
	_, server.err = db.ExecContext(ctx, `
		BEGIN;
		SELECT pg_advisory_xact_lock(784218);
		DO $$ BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN CREATE ROLE anon NOLOGIN; END IF;
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN CREATE ROLE authenticated NOLOGIN; END IF;
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'supabase_auth_admin') THEN CREATE ROLE supabase_auth_admin NOLOGIN; END IF;
		END $$;
		COMMIT;`)
}

// SetupTestDB gives t a separate database with the Supabase Auth stub and
// application migrations installed.
//
// Inputs:
//   - t (*testing.T): receives setup failures and owns database cleanup.
//
// Setup failures fail t. It returns an open *sql.DB valid until test cleanup,
// when that test's database is dropped.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Each package starts its test server only once.
	server.once.Do(startServer)
	if server.err != nil {
		t.Fatalf("start test Postgres (Docker or TEST_DATABASE_URL required): %v", server.err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Create a fresh database so tests cannot share application data.
	admin, err := sql.Open("pgx", server.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	name := "carpool_test_" + uuid.New().String()[:8] + uuid.New().String()[:8]
	// name is generated here, never supplied by an environment variable or caller.
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE "`+name+`"`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := admin.ExecContext(cleanupCtx, `DROP DATABASE "`+name+`" WITH (FORCE)`); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
	// Connect to the new database and add the minimal Supabase Auth schema.
	pgConfig, err := pgx.ParseConfig(server.dsn)
	if err != nil {
		t.Fatal(err)
	}
	pgConfig.Database = name
	db := stdlib.OpenDB(*pgConfig)
	db.SetMaxOpenConns(12)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(ctx, `
		CREATE SCHEMA auth;
		CREATE TABLE auth.users (id uuid PRIMARY KEY);
		GRANT USAGE ON SCHEMA public TO anon, authenticated;
		ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO anon, authenticated;`)
	if err != nil {
		t.Fatal(err)
	}
	// Walk upward because each Go package runs tests from its own directory.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var migrations []string
	for {
		migrations, err = filepath.Glob(filepath.Join(cwd, "supabase", "migrations", "*.sql"))
		if err != nil {
			t.Fatal(err)
		}
		if len(migrations) != 0 {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatal("could not find supabase/migrations")
		}
		cwd = parent
	}
	// Apply the real schema files in filename order.
	for _, migration := range migrations {
		contents, err := os.ReadFile(migration)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(contents)); err != nil {
			t.Fatalf("migration %s: %v", migration, err)
		}
	}
	return db
}
