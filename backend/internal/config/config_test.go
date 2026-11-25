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

// TestLoad_RequiresConnectionTargetsAndDefaultsPort checks the required variables and
// the port fallback.
func TestLoad_RequiresConnectionTargetsAndDefaultsPort(t *testing.T) {
	for _, tc := range []struct {
		name, supabaseURL, databaseURL string
	}{
		{"missing Supabase URL", "", "postgres://localhost/test"},
		{"missing database URL", "http://localhost:54321", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SUPABASE_URL", tc.supabaseURL)
			t.Setenv("DATABASE_URL", tc.databaseURL)
			if _, err := config.Load(); err == nil {
				t.Errorf("Load() with %s = nil, want an error", tc.name)
			}
		})
	}

	t.Run("default port", func(t *testing.T) {
		t.Setenv("SUPABASE_URL", "http://localhost:54321")
		t.Setenv("DATABASE_URL", "postgres://localhost/test")
		t.Setenv("PORT", "")
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want %q", cfg.Port, "8080")
		}
	})
}
