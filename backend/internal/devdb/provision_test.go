package devdb

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestPairConnectionURLReplacesAdministratorCredentials checks that a pair's
// connection string carries only that pair's role and password. It is derived
// from the administrator connection, which holds credentials in both the
// userinfo and the query string, and students receive the result.
func TestPairConnectionURLReplacesAdministratorCredentials(t *testing.T) {
	t.Parallel()
	const adminDSN = "postgresql://postgres:admin-password@localhost/postgres" +
		"?user=postgres&password=query-admin-password&sslmode=disable"
	admin, err := pgx.ParseConfig(adminDSN)
	if err != nil {
		t.Fatalf("ParseConfig(admin) error = %v", err)
	}

	dsn, err := pairConnectionURL(admin, "carpool_profile_test", "pair-password")
	if err != nil {
		t.Fatalf("pairConnectionURL() error = %v", err)
	}

	parsed, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("ParseConfig(pair) error = %v", err)
	}
	if want := "carpool_profile_test"; parsed.User != want {
		t.Errorf("User = %q, want %q", parsed.User, want)
	}
	if want := "pair-password"; parsed.Password != want {
		t.Errorf("Password = %q, want %q", parsed.Password, want)
	}
	for _, secret := range []string{"admin-password", "query-admin-password"} {
		if strings.Contains(dsn, secret) {
			t.Errorf("connection string still carries the administrator %s", secret)
		}
	}
}
