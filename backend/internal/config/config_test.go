package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/rice-apps/carpool4/backend/internal/config"
)

// Load reads the process environment, so these tests set it with t.Setenv and
// cannot run in parallel.

func TestLoadParsesResourceAndOriginConfiguration(t *testing.T) {
	t.Setenv("SUPABASE_URL", "https://project.supabase.co/")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("DATABASE_MAX_CONNECTIONS", "3")
	t.Setenv("CORS_ORIGINS", "https://carpool.rice.edu, http://localhost:3000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if want := "https://project.supabase.co"; cfg.SupabaseURL != want {
		t.Errorf("SupabaseURL = %q, want %q", cfg.SupabaseURL, want)
	}
	if want := 3; cfg.DatabaseMaxConnections != want {
		t.Errorf("DatabaseMaxConnections = %d, want %d", cfg.DatabaseMaxConnections, want)
	}
	want := []string{"https://carpool.rice.edu", "http://localhost:3000"}
	if diff := cmp.Diff(want, cfg.AllowedOrigins); diff != "" {
		t.Errorf("AllowedOrigins mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadDefaultsOriginsAndPort(t *testing.T) {
	t.Setenv("SUPABASE_URL", "http://localhost:54321")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("PORT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if diff := cmp.Diff(want, cfg.AllowedOrigins); diff != "" {
		t.Errorf("AllowedOrigins mismatch (-want +got):\n%s", diff)
	}
	if want := "8080"; cfg.Port != want {
		t.Errorf("Port = %q, want %q", cfg.Port, want)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, key, value string }{
		{"zero pool size", "DATABASE_MAX_CONNECTIONS", "0"},
		{"non-numeric pool size", "DATABASE_MAX_CONNECTIONS", "many"},
		{"origin with a path", "CORS_ORIGINS", "https://carpool.rice.edu/path"},
		{"empty origin", "CORS_ORIGINS", "https://carpool.rice.edu,"},
		{"origin with a query", "CORS_ORIGINS", "https://carpool.rice.edu?a=b"},
		{"origin with credentials", "CORS_ORIGINS", "https://user@carpool.rice.edu"},
		{"Supabase URL with a non-HTTP scheme", "SUPABASE_URL", "file:///auth"},
		{"Supabase URL with a query", "SUPABASE_URL", "https://project.supabase.co?other=issuer"},
		{"missing Supabase URL", "SUPABASE_URL", ""},
		{"missing database URL", "DATABASE_URL", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SUPABASE_URL", "http://localhost:54321")
			t.Setenv("DATABASE_URL", "postgres://localhost/test")
			t.Setenv("DATABASE_MAX_CONNECTIONS", "")
			t.Setenv("CORS_ORIGINS", "")
			t.Setenv(tc.key, tc.value)

			if _, err := config.Load(); err == nil {
				t.Errorf("Load() with %s = nil, want an error", tc.name)
			}
		})
	}
}
