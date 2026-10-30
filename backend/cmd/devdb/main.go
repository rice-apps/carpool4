// Command devdb provisions the instructor-managed development database and checks pair configuration.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rice-apps/carpool4/backend/internal/devdb"
	"github.com/rice-apps/carpool4/backend/internal/envfile"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: devdb <provision|check|verify> [--env-file path]")
	}
	flags := flag.NewFlagSet("devdb", flag.ContinueOnError)
	configFile := flags.String("env-file", "", "private administrator configuration file")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if *configFile != "" {
		if err := godotenv.Load(*configFile); err != nil {
			return err
		}
	}
	if err := envfile.Load(); err != nil {
		return err
	}
	root, err := envfile.Root()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "inspect":
		db, err := sql.Open("pgx", os.Getenv("ADMIN_DATABASE_URL"))
		if err != nil {
			return fmt.Errorf("administrator connection is invalid")
		}
		defer db.Close()
		rows, err := db.QueryContext(ctx, `SELECT schemaname, tablename FROM pg_tables WHERE schemaname IN ('public', 'carpool_admin') ORDER BY 1, 2`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var schema, table string
			if err := rows.Scan(&schema, &table); err != nil {
				return err
			}
			fmt.Println(schema + "." + table)
		}
		return rows.Err()
	case "provision":
		for _, key := range []string{"ADMIN_DATABASE_URL", "SUPABASE_URL", "SUPABASE_ANON_KEY"} {
			if os.Getenv(key) == "" {
				return fmt.Errorf("%s is required in private administrator configuration", key)
			}
		}
		err := devdb.Provision(ctx, devdb.ProvisionOptions{Root: root, AdminURL: os.Getenv("ADMIN_DATABASE_URL"), SupabaseURL: os.Getenv("SUPABASE_URL"), PublicKey: os.Getenv("SUPABASE_ANON_KEY")})
		if err != nil {
			return err
		}
		fmt.Println("Provisioned shared development data and 8 restricted test schemas. Pair configuration:", filepath.Join(root, ".local", "pairs"))
		return nil
	case "check":
		pool, close, err := devdb.OpenSandbox(ctx, os.Getenv("TEST_DATABASE_URL"))
		if err != nil {
			return err
		}
		_ = pool
		if err := close(); err != nil {
			return err
		}
		db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
		if err != nil {
			return fmt.Errorf("DATABASE_URL is invalid")
		}
		defer db.Close()
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM public.locations`).Scan(&count); err != nil {
			return fmt.Errorf("shared development connection failed: %w", err)
		}
		fmt.Println("Shared development connection and restricted test schema are ready.")
		return nil
	case "verify":
		return devdb.Verify(ctx, root)
	default:
		return fmt.Errorf("unknown devdb command")
	}
}
