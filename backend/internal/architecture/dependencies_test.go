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

// TestForbiddenImport checks representative allowed and forbidden edges,
// including descendants and similarly named sibling packages.
func TestForbiddenImport(t *testing.T) {
	for _, tt := range []struct {
		name      string
		from      string
		to        string
		forbidden bool
	}{
		{"app sqlc", module + "/internal/app", module + "/internal/db/sqlc", true},
		{"app unrelated db prefix", module + "/internal/app", module + "/internal/dbtest", false},
		{"app SQL driver", module + "/internal/app", "database/sql/driver", true},
		{"app auth", module + "/internal/app", module + "/internal/auth", true},
		{"app protobuf", module + "/internal/app", module + "/internal/gen/carpool/v1", true},
		{"app protobuf runtime child", module + "/internal/app", "google.golang.org/protobuf/types/known/timestamppb", true},
		{"app Connect", module + "/internal/app", "connectrpc.com/connect", true},
		{"app pgx child", module + "/internal/app", "github.com/jackc/pgx/v5/pgconn", true},
		{"rpc app", module + "/internal/rpc", module + "/internal/app", false},
		{"rpc database", module + "/internal/rpc", module + "/internal/db/postgres", true},
		{"rpc SQL", module + "/internal/rpc", "database/sql", true},
		{"rpc pgx", module + "/internal/rpc", "github.com/jackc/pgx/v5", true},
		{"postgres Connect child", module + "/internal/db/postgres", "connectrpc.com/connect/cmd", true},
		{"postgres generated Connect", module + "/internal/db/postgres", module + "/internal/gen/carpool/v1/carpoolv1connect", true},
		{"postgres sqlc", module + "/internal/db/postgres", module + "/internal/db/sqlc", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := forbiddenImport(tt.from, tt.to); got != tt.forbidden {
				t.Errorf("forbiddenImport(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.forbidden)
			}
		})
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
