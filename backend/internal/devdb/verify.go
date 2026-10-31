package devdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
)

// Verify exercises real restricted connections without changing shared development data.
func Verify(ctx context.Context, root string) error {
	for index, pair := range featurePairs {
		settings, err := godotenv.Read(filepath.Join(root, ".local", "pairs", pair.configName()))
		if err != nil {
			return err
		}
		other := featurePairs[(index+1)%len(featurePairs)]
		if err := verifyPair(ctx, settings, pair.roleName(), other.roleName()); err != nil {
			return fmt.Errorf("%s: %w", pair.label, err)
		}
		pool, _, err := OpenTestPool(ctx, settings["TEST_DATABASE_URL"])
		if err != nil {
			return err
		}
		var remaining int
		err = pool.QueryRowContext(ctx, `SELECT count(*) FROM locations`).Scan(&remaining)
		pool.Close()
		if err != nil || remaining != 0 {
			return fmt.Errorf("%s fixture cleanup failed", pair.label)
		}
	}
	fmt.Println("Verified all 8 feature pairs: real SQL, pooled schema selection, cross-pair isolation, protected development/Auth data, fixture cleanup, and concurrent-run locking.")
	return nil
}

func verifyPair(ctx context.Context, settings map[string]string, schema, other string) (result error) {
	db, close, err := OpenSandbox(ctx, settings["TEST_DATABASE_URL"])
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, close()) }()
	var count int
	var status string
	if err := db.QueryRowContext(ctx, `SELECT 'active'::ride_status`).Scan(&status); err != nil || status != "active" {
		return fmt.Errorf("application enum types are unavailable")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO locations(title, address) VALUES ('Private fixture', 'private-fixture')`); err != nil {
		return err
	}
	connections := make([]*sql.Conn, 0, 2)
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()
	for range 2 {
		conn, err := db.Conn(ctx)
		if err != nil {
			return err
		}
		connections = append(connections, conn)
		var actual string
		if err := conn.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&actual); err != nil || actual != schema {
			return fmt.Errorf("pooled connection selected the wrong schema")
		}
	}
	for _, conn := range connections {
		conn.Close()
	}
	connections = nil
	for _, query := range []string{
		`SELECT * FROM ` + pgx.Identifier{other, "locations"}.Sanitize(),
		`DELETE FROM ` + pgx.Identifier{other, "locations"}.Sanitize() + ` WHERE false`,
		`UPDATE public.locations SET title = title WHERE false`,
		`INSERT INTO auth.users(id) VALUES (gen_random_uuid())`,
		`CREATE SCHEMA carpool_unassigned`,
	} {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, query)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			return fmt.Errorf("restricted operation did not fail with permission denied: %s: %v", query, err)
		}
	}
	for _, role := range []string{"anon", "authenticated"} {
		var allowed bool
		if err := db.QueryRowContext(ctx, `SELECT has_table_privilege($1, $2, 'SELECT')`, role, schema+".users").Scan(&allowed); err != nil {
			return err
		}
		if allowed {
			return fmt.Errorf("client role can read private application data")
		}
	}
	blockedCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	_, unexpectedClose, err := OpenSandbox(blockedCtx, settings["TEST_DATABASE_URL"])
	if err == nil {
		_ = unexpectedClose()
		return fmt.Errorf("a concurrent fixture bypassed the schema lock")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("concurrent fixture failed before reaching its lock: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM locations`).Scan(&count); err != nil || count != 1 {
		return fmt.Errorf("concurrent fixture changed active test data")
	}
	development, err := sql.Open("pgx", settings["DATABASE_URL"])
	if err != nil {
		return err
	}
	defer development.Close()
	_, err = development.ExecContext(ctx, `SELECT * FROM `+pgx.Identifier{schema, "locations"}.Sanitize())
	var developmentAccess *pgconn.PgError
	if !errors.As(err, &developmentAccess) || developmentAccess.Code != "42501" {
		return fmt.Errorf("shared development login can access test fixtures")
	}
	if err := development.QueryRowContext(ctx, `SELECT count(*) FROM public.locations WHERE address = 'private-fixture'`).Scan(&count); err != nil || count != 0 {
		return fmt.Errorf("test fixture reached shared development data")
	}
	return nil
}
