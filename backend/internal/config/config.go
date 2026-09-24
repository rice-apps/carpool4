package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds the server port, Supabase URL, database settings, and allowed
// browser origins after validation by Load.
//
// Fields:
//   - Port: selects the HTTP listener port.
//   - SupabaseURL: locates the Supabase project for token verification.
//   - DatabaseURL: connects the server to PostgreSQL.
//   - DatabaseMaxConnections: limits the database connection pool.
//   - AllowedOrigins: lists browser origins allowed by CORS.
type Config struct {
	Port                   string
	SupabaseURL            string
	DatabaseURL            string
	DatabaseMaxConnections int
	AllowedOrigins         []string
}

// Load returns validated server settings from environment variables, with
// defaults for the port, pool size, and browser origins. Missing required URLs
// or invalid connection limits and origins return an error.
func Load() (*Config, error) {
	// Read optional settings with local development defaults.
	cfg := &Config{
		Port:                   os.Getenv("PORT"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		DatabaseMaxConnections: 10,
		AllowedOrigins:         []string{"http://localhost:3000", "http://127.0.0.1:3000"},
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	// Check the required connection settings before starting the server.
	if cfg.SupabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	// The issuer URL needs an HTTP(S) host and no credentials or query.
	parsedURL, err := url.Parse(cfg.SupabaseURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.RawQuery != "" || parsedURL.Fragment != "" || parsedURL.User != nil {
		return nil, fmt.Errorf("SUPABASE_URL must be an HTTP(S) URL without credentials, query, or fragment")
	}
	cfg.SupabaseURL = strings.TrimRight(cfg.SupabaseURL, "/")
	// Override pool size and browser origins only when supplied.
	if value := os.Getenv("DATABASE_MAX_CONNECTIONS"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 {
			return nil, fmt.Errorf("DATABASE_MAX_CONNECTIONS must be a positive integer")
		}
		cfg.DatabaseMaxConnections = limit
	}
	if value := os.Getenv("CORS_ORIGINS"); value != "" {
		cfg.AllowedOrigins = strings.Split(value, ",")
		for i, origin := range cfg.AllowedOrigins {
			origin = strings.TrimSpace(origin)
			u, err := url.Parse(origin)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
				return nil, fmt.Errorf("CORS_ORIGINS must contain comma-separated HTTP(S) origins")
			}
			cfg.AllowedOrigins[i] = origin
		}
	}

	return cfg, nil
}
