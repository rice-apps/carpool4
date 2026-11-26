package devdb

import (
	"strings"
	"testing"
)

// TestOpenTestPoolRejectsUnrestrictedCredentials checks that the classroom pool
// refuses any connection string that is not a provisioned pair role, and that
// the refusal does not echo a password it was given.
func TestOpenTestPoolRejectsUnrestrictedCredentials(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, dsn, secret string }{
		{name: "empty", dsn: ""},
		{name: "not a URL", dsn: "not-a-url"},
		{name: "superuser", dsn: "postgresql://postgres:private-password@localhost/postgres", secret: "private-password"},
		{name: "development role", dsn: "postgresql://carpool_development:other-password@localhost/postgres", secret: "other-password"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := OpenTestPool(t.Context(), tc.dsn)
			if err == nil {
				t.Fatalf("OpenTestPool(%s) error = nil, want an error", tc.name)
			}
			if tc.secret != "" && strings.Contains(err.Error(), tc.secret) {
				t.Errorf("OpenTestPool(%s) error exposed the password: %v", tc.name, err)
			}
		})
	}
}
