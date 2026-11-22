package config_test

import (
	"reflect"
	"testing"

	"github.com/rice-apps/carpool4/backend/internal/config"
)

// TestLoad_ResourceAndOriginConfiguration checks parsed pool limits, trimmed
// origins, and the normalized Supabase URL.
func TestLoad_ResourceAndOriginConfiguration(t *testing.T) {
	t.Setenv("SUPABASE_URL", "https://project.supabase.co/")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("DATABASE_MAX_CONNECTIONS", "3")
	t.Setenv("CORS_ORIGINS", "https://carpool.rice.edu, http://localhost:3000")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SupabaseURL != "https://project.supabase.co" {
		t.Fatalf("URL = %q", cfg.SupabaseURL)
	}
	if cfg.DatabaseMaxConnections != 3 {
		t.Fatalf("pool size = %d", cfg.DatabaseMaxConnections)
	}
	if !reflect.DeepEqual(cfg.AllowedOrigins, []string{"https://carpool.rice.edu", "http://localhost:3000"}) {
		t.Fatalf("origins = %v", cfg.AllowedOrigins)
	}
}

// TestLoad_DefaultFrontendOrigins checks the local CORS defaults when no
// origin override is configured.
func TestLoad_DefaultFrontendOrigins(t *testing.T) {
	t.Setenv("SUPABASE_URL", "http://localhost:54321")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("CORS_ORIGINS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if !reflect.DeepEqual(cfg.AllowedOrigins, want) {
		t.Fatalf("origins = %v, want %v", cfg.AllowedOrigins, want)
	}
}

// TestLoad_RejectsInvalidResourceConfiguration checks that invalid pool,
// origin, and Supabase URL values each fail loading.
func TestLoad_RejectsInvalidResourceConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, key, value string }{
		{"zero pool", "DATABASE_MAX_CONNECTIONS", "0"},
		{"invalid pool", "DATABASE_MAX_CONNECTIONS", "many"},
		{"origin with path", "CORS_ORIGINS", "https://carpool.rice.edu/path"},
		{"empty origin", "CORS_ORIGINS", "https://carpool.rice.edu,"},
		{"invalid auth scheme", "SUPABASE_URL", "file:///auth"},
		{"auth query", "SUPABASE_URL", "https://project.supabase.co?other=issuer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SUPABASE_URL", "http://localhost:54321")
			t.Setenv("DATABASE_URL", "postgres://localhost/test")
			t.Setenv("DATABASE_MAX_CONNECTIONS", "")
			t.Setenv("CORS_ORIGINS", "")
			t.Setenv(tc.key, tc.value)
			if _, err := config.Load(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

// TestLoad_MissingSupabaseURL checks that auth configuration is required.
func TestLoad_MissingSupabaseURL(t *testing.T) {
	t.Setenv("SUPABASE_URL", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	if _, err := config.Load(); err == nil {
		t.Error("expected an error with SUPABASE_URL unset, got nil")
	}
}

// TestLoad_MissingDatabaseURL checks that a database target is required.
func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("SUPABASE_URL", "http://localhost:54321")
	t.Setenv("DATABASE_URL", "")

	if _, err := config.Load(); err == nil {
		t.Error("expected an error with DATABASE_URL unset, got nil")
	}
}

// TestLoad_DefaultPort checks the HTTP port used when PORT is unset.
func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("SUPABASE_URL", "http://localhost:54321")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("PORT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.Port, "8080"; got != want {
		t.Errorf("Port = %q, want %q", got, want)
	}
}
