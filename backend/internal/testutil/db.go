// Package testutil prepares private SQL fixtures using the pair's supplied configuration.
package testutil

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/rice-apps/carpool4/backend/internal/devdb"
	"github.com/rice-apps/carpool4/backend/internal/envfile"
)

func Run(m *testing.M) int { return m.Run() }

// SetupTestDB returns the assigned schema with fresh fixtures. Cleanup releases its cross-process lock.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("SQL fixtures are excluded in short mode")
	}
	if err := envfile.Load(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, close, err := devdb.OpenSandbox(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := close(); err != nil {
			t.Errorf("clean test fixtures: %v", err)
		}
	})
	return db
}
