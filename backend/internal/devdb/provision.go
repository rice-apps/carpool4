package devdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type ProvisionOptions struct {
	Root, AdminURL, SupabaseURL, PublicKey string
}

// Provision prepares the shared app and eight private SQL sandboxes, then writes pair .env files.
func Provision(ctx context.Context, options ProvisionOptions) error {
	config, err := pgx.ParseConfig(options.AdminURL)
	if err != nil || options.AdminURL == "" {
		return fmt.Errorf("ADMIN_DATABASE_URL must name the dedicated development database")
	}
	db, err := sql.Open("pgx", options.AdminURL)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(784219); CREATE SCHEMA IF NOT EXISTS carpool_admin; REVOKE ALL ON SCHEMA carpool_admin FROM PUBLIC; CREATE TABLE IF NOT EXISTS carpool_admin.migrations (name text PRIMARY KEY);`); err != nil {
		return err
	}
	migrations, err := filepath.Glob(filepath.Join(options.Root, "supabase", "migrations", "*.sql"))
	if err != nil || len(migrations) == 0 {
		return fmt.Errorf("application migrations not found")
	}
	for _, path := range migrations {
		name := filepath.Base(path)
		var applied bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT FROM carpool_admin.migrations WHERE name = $1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			return fmt.Errorf("apply %s (use a dedicated development project with no pre-existing application tables): %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO carpool_admin.migrations(name) VALUES ($1)`, name); err != nil {
			return err
		}
	}
	output := filepath.Join(options.Root, ".local", "pairs")
	if err := os.MkdirAll(output, 0700); err != nil {
		return err
	}
	firstEnv, _ := godotenv.Read(filepath.Join(output, featurePairs[0].configName()))
	developmentURL, err := ensureLogin(ctx, tx, config, "carpool_development", firstEnv["DATABASE_URL"])
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `GRANT USAGE ON SCHEMA public TO carpool_development`); err != nil {
		return err
	}
	for _, table := range appTables {
		if err := grantBackend(ctx, tx, "public", table, "carpool_development"); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `GRANT EXECUTE ON FUNCTION public.ride_riders(uuid) TO carpool_development`); err != nil {
		return err
	}
	seed, err := os.ReadFile(filepath.Join(options.Root, "supabase", "seed.sql"))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(seed)); err != nil {
		return err
	}
	for _, pair := range featurePairs {
		name := pair.configName()
		previous, _ := godotenv.Read(filepath.Join(output, name))
		role := pair.roleName()
		testURL, err := ensureLogin(ctx, tx, config, role, previous["TEST_DATABASE_URL"])
		if err != nil {
			return err
		}
		if err := provisionSandbox(ctx, tx, role); err != nil {
			return fmt.Errorf("provision %s: %w", pair.label, err)
		}
		contents, err := godotenv.Marshal(map[string]string{
			"DATABASE_URL": developmentURL, "TEST_DATABASE_URL": testURL,
			"SUPABASE_URL": options.SupabaseURL, "SUPABASE_ANON_KEY": options.PublicKey,
		})
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(output, name), []byte(contents+"\n"), 0600); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var appTables = []string{"users", "locations", "rides", "ride_occupants"}

func ensureLogin(ctx context.Context, tx *sql.Tx, config *pgx.ConnConfig, role, previousURL string) (string, error) {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT FROM pg_roles WHERE rolname = $1)`, role).Scan(&exists); err != nil {
		return "", err
	}
	password := ""
	if previousURL != "" {
		previous, err := pgx.ParseConfig(previousURL)
		if err != nil {
			return "", fmt.Errorf("saved configuration for %s is invalid", role)
		}
		password = previous.Password
	}
	if exists && password == "" {
		return "", fmt.Errorf("%s already exists but its saved pair configuration is missing; refusing to replace its password", role)
	}
	if password == "" {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		password = base64.RawURLEncoding.EncodeToString(bytes)
	}
	if !exists {
		var statement string
		if err := tx.QueryRowContext(ctx, `SELECT format('CREATE ROLE %I LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD %L', $1::text, $2::text)`, role, password).Scan(&statement); err != nil {
			return "", err
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return "", err
		}
	}
	return pairConnectionURL(config, role, password)
}

func pairConnectionURL(config *pgx.ConnConfig, role, password string) (string, error) {
	user := role
	if _, suffix, pooled := strings.Cut(config.User, "."); pooled {
		user += "." + suffix
	}
	parsed, err := url.Parse(config.ConnString())
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return "", fmt.Errorf("ADMIN_DATABASE_URL must be a PostgreSQL URL")
	}
	sslmode := parsed.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "require"
	}
	student := url.URL{
		Scheme: "postgresql", Host: net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Path: "/" + config.Database, User: url.UserPassword(user, password),
		RawQuery: url.Values{"sslmode": []string{sslmode}}.Encode(),
	}
	return student.String(), nil
}

func grantBackend(ctx context.Context, tx *sql.Tx, schema, table, role string) error {
	qualified := pgx.Identifier{schema, table}.Sanitize()
	login := pgx.Identifier{role}.Sanitize()
	_, err := tx.ExecContext(ctx, `ALTER TABLE `+qualified+` ENABLE ROW LEVEL SECURITY; REVOKE ALL ON `+qualified+` FROM PUBLIC, anon, authenticated; GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE ON `+qualified+` TO `+login+`; DROP POLICY IF EXISTS carpool_backend ON `+qualified+`; CREATE POLICY carpool_backend ON `+qualified+` TO `+login+` USING (true) WITH CHECK (true);`)
	return err
}

func provisionSandbox(ctx context.Context, tx *sql.Tx, role string) error {
	schema := pgx.Identifier{role}.Sanitize()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 784218))`, role); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT FROM pg_namespace WHERE nspname = $1)`, role).Scan(&exists); err != nil {
		return err
	}
	if exists {
		var assigned string
		if err := tx.QueryRowContext(ctx, `SELECT role_name FROM `+schema+`.__carpool_test_environment`).Scan(&assigned); err != nil || assigned != role {
			return fmt.Errorf("existing schema is not a managed test sandbox")
		}
		if _, err := tx.ExecContext(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA `+schema+`; REVOKE ALL ON SCHEMA `+schema+` FROM PUBLIC; GRANT USAGE ON SCHEMA `+schema+` TO `+schema+`; CREATE TABLE `+schema+`.__carpool_test_environment(role_name text PRIMARY KEY); INSERT INTO `+schema+`.__carpool_test_environment VALUES ('`+role+`'); GRANT SELECT ON `+schema+`.__carpool_test_environment TO `+schema+`; CREATE TABLE `+schema+`.auth_users(id uuid PRIMARY KEY); GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE ON `+schema+`.auth_users TO `+schema+`;`); err != nil {
		return err
	}
	for _, table := range appTables {
		if _, err := tx.ExecContext(ctx, `CREATE TABLE `+pgx.Identifier{role, table}.Sanitize()+` (LIKE `+pgx.Identifier{"public", table}.Sanitize()+` INCLUDING ALL)`); err != nil {
			return err
		}
	}
	for _, table := range appTables {
		rows, err := tx.QueryContext(ctx, `SELECT conname, pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = $1::regclass AND contype = 'f'`, "public."+table)
		if err != nil {
			return err
		}
		type constraint struct{ name, definition string }
		var constraints []constraint
		for rows.Next() {
			var c constraint
			if err := rows.Scan(&c.name, &c.definition); err != nil {
				rows.Close()
				return err
			}
			constraints = append(constraints, c)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, c := range constraints {
			definition := c.definition
			for _, target := range appTables {
				definition = strings.ReplaceAll(definition, "REFERENCES public."+target+"(", "REFERENCES "+pgx.Identifier{role, target}.Sanitize()+"(")
				definition = strings.ReplaceAll(definition, "REFERENCES "+target+"(", "REFERENCES "+pgx.Identifier{role, target}.Sanitize()+"(")
			}
			definition = strings.ReplaceAll(definition, "REFERENCES auth.users(", "REFERENCES "+schema+".auth_users(")
			if _, err := tx.ExecContext(ctx, `ALTER TABLE `+pgx.Identifier{role, table}.Sanitize()+` ADD CONSTRAINT `+pgx.Identifier{c.name}.Sanitize()+` `+definition); err != nil {
				return err
			}
		}
		if err := grantBackend(ctx, tx, role, table, role); err != nil {
			return err
		}
	}
	for _, signature := range []string{"public.ride_riders(uuid)", "public.hook_restrict_rice_google_signup(jsonb)"} {
		var definition string
		if err := tx.QueryRowContext(ctx, `SELECT pg_get_functiondef($1::regprocedure)`, signature).Scan(&definition); err != nil {
			return err
		}
		definition = strings.ReplaceAll(definition, "public.", schema+".")
		definition = strings.ReplaceAll(definition, "'public'", "'"+role+"'")
		if _, err := tx.ExecContext(ctx, definition); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `REVOKE ALL ON ALL FUNCTIONS IN SCHEMA `+schema+` FROM PUBLIC, anon, authenticated; GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA `+schema+` TO `+schema+`; GRANT USAGE ON SCHEMA public TO `+schema+`; ALTER ROLE `+schema+` SET search_path = `+schema+`, public;`)
	return err
}
