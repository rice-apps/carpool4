// Command server runs the Rice Carpool backend.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/cors"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	"github.com/rice-apps/carpool4/backend/internal/config"
	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
)

// main runs the server and logs a fatal error if startup or serving fails.
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run serves HTTP until interrupted. It returns startup or serving errors and
// closes the database pool before returning.
func run() error {
	// Use the process signal to stop active server work.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Build the database and RPC routes before listening.
	db, err := setupDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	mux, err := setupRouter(ctx, db, cfg)
	if err != nil {
		return err
	}

	handler := setupMiddleware(mux, cfg.AllowedOrigins)

	return startServer(ctx, ":"+cfg.Port, handler)
}

// setupDatabase opens and checks a PostgreSQL pool.
//
// Inputs:
//   - ctx (context.Context): limits the connection check.
//   - cfg (*config.Config): supplies the database URL and pool limits.
//
// It returns a ready *sql.DB for the caller to close, or an error.
func setupDatabase(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(cfg.DatabaseMaxConnections)
	db.SetMaxIdleConns(min(5, cfg.DatabaseMaxConnections))
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)
	// Check connectivity now so startup fails before serving requests.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// setupRouter builds the health and RPC routes.
//
// Inputs:
//   - ctx (context.Context): controls signing-key fetches and refreshes.
//   - db (*sql.DB): the pool backing application storage.
//   - cfg (*config.Config): supplies the Supabase URL.
//
// It returns an *http.ServeMux or a setup error, including an initial key-fetch
// failure.
func setupRouter(ctx context.Context, db *sql.DB, cfg *config.Config) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	// Health checks stay available without a caller token.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// RPC routes share token verification and application storage.
	interceptor, err := auth.NewInterceptor(ctx, cfg.SupabaseURL)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	service := app.NewService(postgres.NewDatabase(db))
	server := rpc.NewServer(service)
	if err := rpc.Register(mux, server, interceptor, logger); err != nil {
		return nil, err
	}

	return mux, nil
}

// setupMiddleware configures the HTTP middleware.
//
// Inputs:
//   - mux (*http.ServeMux): the routes to serve.
//   - origins ([]string): browser origins allowed by CORS.
//
// It returns a handler with CORS and an eight-second request deadline.
func setupMiddleware(mux *http.ServeMux, origins []string) http.Handler {
	// Allow configured browser origins to call the Connect routes.
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin", "X-Grpc-Web"},
	}).Handler(mux)

	// Give each request a deadline that also reaches database queries.
	return requestTimeoutHandler(corsHandler, 8*time.Second)
}

// requestTimeoutHandler adds a context deadline to each request.
//
// Inputs:
//   - next (http.Handler): receives the timed request.
//   - timeout (time.Duration): the allowed duration per request.
//
// It returns a handler that cancels downstream work when the deadline expires.
func requestTimeoutHandler(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// startServer listens and serves HTTP.
//
// Inputs:
//   - ctx (context.Context): stops serving when canceled.
//   - addr (string): the TCP address to listen on.
//   - handler (http.Handler): the HTTP handler to serve.
//
// It returns a listen, serving, or shutdown error.
func startServer(ctx context.Context, addr string, handler http.Handler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("listening on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return serve(ctx, srv, listener)
}

// serve runs an HTTP server and handles graceful shutdown.
//
// Inputs:
//   - ctx (context.Context): triggers shutdown when canceled.
//   - srv (*http.Server): the server to run.
//   - listener (net.Listener): the open listener used by srv.
//
// It gives active requests ten seconds to finish, then returns any serving or
// shutdown error.
func serve(ctx context.Context, srv *http.Server, listener net.Listener) error {
	// Keep serving while the main goroutine watches for shutdown.
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Let active requests finish within the shutdown window.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}
