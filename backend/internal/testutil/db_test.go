package testutil_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/testutil"
)

// TestMain delegates package teardown and exit status to the database harness.
func TestMain(m *testing.M) { os.Exit(testutil.Run(m)) }

// TestSchemaRetainsAuthForeignKey checks that application profiles still need
// a corresponding Supabase Auth identity.
func TestSchemaRetainsAuthForeignKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	if _, err := db.Exec(`INSERT INTO users (id, email) VALUES ($1, 'missing@rice.edu')`, uuid.New()); err == nil {
		t.Fatal("public profile without an auth identity was accepted")
	}
}

// TestDirectAPIRolesCannotAccessApplicationData checks RLS and effective privileges
// for both client roles without granting the pair permission to impersonate them.
func TestDirectAPIRolesCannotAccessApplicationData(t *testing.T) {
	db := testutil.SetupTestDB(t)
	var schema string
	if err := db.QueryRow(`SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	// One round trip for the whole role x table x privilege matrix. PostgreSQL
	// includes inherited grants in its effective privilege checks.
	rows, err := db.Query(`
		SELECT t.name, r.role, p.privilege,
		       has_table_privilege(r.role, c.oid, p.privilege),
		       c.relrowsecurity
		FROM unnest(ARRAY['users', 'rides', 'ride_occupants', 'locations']) AS t(name)
		JOIN pg_class c ON c.oid = (quote_ident($1) || '.' || quote_ident(t.name))::regclass
		CROSS JOIN unnest(ARRAY['anon', 'authenticated']) AS r(role)
		CROSS JOIN unnest(ARRAY['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE']) AS p(privilege)`, schema)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	checked := 0
	for rows.Next() {
		var table, role, privilege string
		var allowed, rowSecurity bool
		if err := rows.Scan(&table, &role, &privilege, &allowed, &rowSecurity); err != nil {
			t.Fatal(err)
		}
		if allowed {
			t.Errorf("%s still has %s on %s", role, privilege, table)
		}
		if !rowSecurity {
			t.Errorf("row security is disabled on %s", table)
		}
		checked++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if want := 4 * 2 * 5; checked != want {
		t.Fatalf("checked %d role/table/privilege combinations, want %d", checked, want)
	}
}

// TestRiceGoogleSignupHook checks the migration's admission hook accepts Rice
// Google claims and rejects missing, foreign, or forged hosted-domain claims.
func TestRiceGoogleSignupHook(t *testing.T) {
	db := testutil.SetupTestDB(t)
	// Vary the provider and hosted-domain claim sent to the signup hook.
	tests := []struct {
		name    string
		event   string
		allowed bool
	}{
		{
			name:    "Rice Google account",
			event:   `{"user":{"app_metadata":{"provider":"google"},"user_metadata":{"custom_claims":{"hd":"rice.edu"}}}}`,
			allowed: true,
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
			name:  "password account cannot forge hosted domain",
			event: `{"user":{"app_metadata":{"provider":"email"},"user_metadata":{"custom_claims":{"hd":"rice.edu"}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			if err := db.QueryRow(
				`SELECT hook_restrict_rice_google_signup($1::jsonb)`,
				tt.event,
			).Scan(&raw); err != nil {
				t.Fatal(err)
			}

			var result map[string]any
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			_, rejected := result["error"]
			if rejected == tt.allowed {
				t.Fatalf("result = %s, allowed = %v", raw, tt.allowed)
			}
		})
	}
}
