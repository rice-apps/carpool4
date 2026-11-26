package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

// Load walks up from the working directory and writes to the process
// environment, so these tests use t.Chdir and t.Setenv and cannot run in
// parallel.

// TestLoadReadsRepositoryRootWithoutOverridingTheProcess checks that Load finds
// the .env above the module, parses a quoted value containing the delimiters it
// splits on, and leaves a variable the process already set alone.
func TestLoadReadsRepositoryRootWithoutOverridingTheProcess(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "backend", "internal", "example")
	if err := os.MkdirAll(packageDir, 0700); err != nil {
		t.Fatalf("create package directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "go.mod"), []byte("module example"), 0600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	env := "CARPOOL_ENV_TEST_OVERRIDE=file\nCARPOOL_ENV_TEST_QUOTED='contains # and = characters'\n"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(env), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Setenv("CARPOOL_ENV_TEST_OVERRIDE", "process")
	t.Cleanup(func() { os.Unsetenv("CARPOOL_ENV_TEST_QUOTED") })
	t.Chdir(packageDir)

	if err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := os.Getenv("CARPOOL_ENV_TEST_OVERRIDE"), "process"; got != want {
		t.Errorf("CARPOOL_ENV_TEST_OVERRIDE = %q, want the process value %q", got, want)
	}
	if got, want := os.Getenv("CARPOOL_ENV_TEST_QUOTED"), "contains # and = characters"; got != want {
		t.Errorf("CARPOOL_ENV_TEST_QUOTED = %q, want %q", got, want)
	}
}

// TestLoadSucceedsOutsideTheRepository checks that a deployed binary, which has
// no .env to find, starts on its runtime environment alone.
func TestLoadSucceedsOutsideTheRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "provided-by-runtime")

	if err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := os.Getenv("DATABASE_URL"), "provided-by-runtime"; got != want {
		t.Errorf("DATABASE_URL = %q, want %q", got, want)
	}
}
