// Package migrations_test checks the properties of supabase/migrations that the
// application relies on but cannot enforce: referential integrity, the API-role
// privilege posture, and the Rice admission hook.
package migrations_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/dbtest"
)

func TestMain(m *testing.M) { os.Exit(dbtest.Run(m)) }

func TestProfileRequiresAuthIdentity(t *testing.T) {
	db := dbtest.DB(t)

	_, err := db.Exec(`INSERT INTO users (id, email) VALUES ($1, 'missing@rice.edu')`, uuid.New())
	if err == nil {
		t.Error("INSERT INTO users without an auth identity succeeded, want a foreign key violation")
	}
}

// TestAPIRolesCannotReachApplicationData covers a production control rather than
// a convenience: Supabase publishes PostgREST at the project URL, so a missing
// REVOKE would let anyone holding the public anon key read application data
// directly, bypassing the contact-privacy rules in the Go service.
func TestAPIRolesCannotReachApplicationData(t *testing.T) {
	db := dbtest.DB(t)

	// PostgreSQL counts inherited grants in its effective privilege checks, so
	// one query covers the whole role x table x privilege matrix.
	rows, err := db.Query(`
		SELECT t.name, r.role, p.privilege,
		       has_table_privilege(r.role, c.oid, p.privilege),
		       c.relrowsecurity
		FROM unnest(ARRAY['users', 'rides', 'ride_occupants', 'locations']) AS t(name)
		JOIN pg_class c ON c.oid = ('public.' || quote_ident(t.name))::regclass
		CROSS JOIN unnest(ARRAY['anon', 'authenticated']) AS r(role)
		CROSS JOIN unnest(ARRAY['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE']) AS p(privilege)`)
	if err != nil {
		t.Fatalf("privilege matrix query: %v", err)
	}
	defer rows.Close()

	checked := 0
	for rows.Next() {
		var table, role, privilege string
		var granted, rowSecurity bool
		if err := rows.Scan(&table, &role, &privilege, &granted, &rowSecurity); err != nil {
			t.Fatalf("scan privilege row: %v", err)
		}
		if granted {
			t.Errorf("has_table_privilege(%q, %q, %q) = true, want false", role, table, privilege)
		}
		if !rowSecurity {
			t.Errorf("row security on %q = false, want true", table)
		}
		checked++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read privilege matrix: %v", err)
	}
	// Guards against a query that silently matches nothing and passes green.
	if want := 4 * 2 * 5; checked != want {
		t.Errorf("checked %d role/table/privilege combinations, want %d", checked, want)
	}
}

func TestRiceGoogleSignupHook(t *testing.T) {
	db := dbtest.DB(t)

	for _, tc := range []struct {
		name  string
		event string
		admit bool
	}{
		{
			name:  "Rice Google account",
			event: `{"user":{"app_metadata":{"provider":"google"},"user_metadata":{"custom_claims":{"hd":"rice.edu"}}}}`,
			admit: true,
		},
		{
			name:  "personal Google account",
			event: `{"user":{"app_metadata":{"provider":"google"},"user_metadata":{}}}`,
		},
		{
			name:  "different Google Workspace",
			event: `{"user":{"app_metadata":{"provider":"google"},"user_metadata":{"custom_claims":{"hd":"example.edu"}}}}`,
		},
		{
			// user_metadata is client-writable, so a password account must not be
			// able to claim a Rice hosted domain.
			name:  "password account forging a hosted domain",
			event: `{"user":{"app_metadata":{"provider":"email"},"user_metadata":{"custom_claims":{"hd":"rice.edu"}}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw []byte
			if err := db.QueryRow(`SELECT hook_restrict_rice_google_signup($1::jsonb)`, tc.event).Scan(&raw); err != nil {
				t.Fatalf("hook_restrict_rice_google_signup(%s) error = %v", tc.event, err)
			}
			var result map[string]any
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatalf("unmarshal hook result %s: %v", raw, err)
			}
			if _, rejected := result["error"]; rejected == tc.admit {
				t.Errorf("hook admitted = %v, want %v (result %s)", !rejected, tc.admit, raw)
			}
		})
	}
}
