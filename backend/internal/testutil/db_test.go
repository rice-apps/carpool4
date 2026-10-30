package testutil_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rice-apps/carpool4/backend/internal/testutil"
	"github.com/google/uuid"
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
	// PostgreSQL includes inherited grants in its effective privilege checks.
	for _, role := range []string{"anon", "authenticated"} {
		for _, table := range []string{"users", "rides", "ride_occupants", "locations"} {
			t.Run(role+"/"+table, func(t *testing.T) {
				var rls bool
				if err := db.QueryRow(`SELECT relrowsecurity FROM pg_class WHERE oid = $1::regclass`, schema+"."+table).Scan(&rls); err != nil {
					t.Fatal(err)
				}
				if !rls {
					t.Error("row security is disabled")
				}
				for _, privilege := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE"} {
					var allowed bool
					if err := db.QueryRow(`SELECT has_table_privilege($1, $2, $3)`, role, schema+"."+table, privilege).Scan(&allowed); err != nil {
						t.Fatal(err)
					}
					if allowed {
						t.Errorf("%s still has %s on %s", role, privilege, table)
					}
				}
			})
		}
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
