package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFindsRootFromPackageAndPreservesOverrides(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "backend", "internal", "example")
	if err := os.MkdirAll(packageDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "go.mod"), []byte("module example"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("CARPOOL_ENV_TEST_OVERRIDE=file\nCARPOOL_ENV_TEST_QUOTED='contains # and = characters'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CARPOOL_ENV_TEST_OVERRIDE", "process")
	t.Cleanup(func() { os.Unsetenv("CARPOOL_ENV_TEST_QUOTED") })
	t.Chdir(packageDir)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("CARPOOL_ENV_TEST_OVERRIDE") != "process" {
		t.Fatal("process override was replaced")
	}
	if os.Getenv("CARPOOL_ENV_TEST_QUOTED") != "contains # and = characters" {
		t.Fatal("quoted configuration was not parsed")
	}
}

func TestLoadAllowsConfiguredServerOutsideRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "provided-by-runtime")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DATABASE_URL") != "provided-by-runtime" {
		t.Fatal("runtime configuration was replaced")
	}
}
