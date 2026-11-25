package architecture_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

const module = "github.com/rice-apps/carpool4/backend"

// forbiddenImport reports whether a direct production import from one package
// to another crosses the application, RPC, or Postgres dependency boundary.
func forbiddenImport(from, to string) bool {
	switch from {
	case module + "/internal/app":
		return inPackageTree(to, module+"/internal/db") ||
			inPackageTree(to, module+"/internal/auth") ||
			inPackageTree(to, module+"/internal/gen") ||
			inPackageTree(to, "connectrpc.com/connect") ||
			inPackageTree(to, "google.golang.org/protobuf") ||
			inPackageTree(to, "database/sql") ||
			inPackageTree(to, "github.com/jackc/pgx")
	case module + "/internal/rpc":
		return inPackageTree(to, module+"/internal/db") ||
			inPackageTree(to, "database/sql") ||
			inPackageTree(to, "github.com/jackc/pgx")
	case module + "/internal/db/postgres":
		return inPackageTree(to, "connectrpc.com/connect") ||
			inPackageTree(to, module+"/internal/gen")
	default:
		return false
	}
}

// inPackageTree matches root and descendants without matching a sibling whose
// name merely starts with the same text.
func inPackageTree(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

// TestForbiddenImportIgnoresSimilarlyNamedPackages checks the boundary check matches
// whole package paths, so internal/dbfixtures is not mistaken for internal/db.
func TestForbiddenImportIgnoresSimilarlyNamedPackages(t *testing.T) {
	if forbiddenImport(module+"/internal/app", module+"/internal/dbfixtures") {
		t.Error("internal/dbfixtures was treated as a database package")
	}
	if !forbiddenImport(module+"/internal/app", module+"/internal/db/sqlc") {
		t.Error("internal/db/sqlc was not treated as a database package")
	}
}

// TestProductionDependencies inspects go list's direct production imports and
// fails when an implementation layer crosses a forbidden boundary.
func TestProductionDependencies(t *testing.T) {
	// Ask Go for each package's direct imports.
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = "../.."
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -json ./...: %v\n%s", err, stderr.String())
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	// Check every package against the boundary rules above.
	for {
		var pkg struct {
			ImportPath string
			Imports    []string
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		// Imports excludes TestImports and XTestImports: integration tests may use Postgres.
		for _, imported := range pkg.Imports {
			if forbiddenImport(pkg.ImportPath, imported) {
				t.Errorf("forbidden production import: %s -> %s", pkg.ImportPath, imported)
			}
		}
	}
}
