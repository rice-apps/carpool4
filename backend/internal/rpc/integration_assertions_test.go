package rpc_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
)

// newRPCServer wires a supplied database into the public RPC server for integration tests.
func newRPCServer(database app.Database) *rpc.Server {
	return rpc.NewServer(app.NewService(database))
}

// assertRPCContactRedacted fails t unless got keeps wantID but hides every protected contact field.
func assertRPCContactRedacted(t *testing.T, got *carpoolv1.User, wantID uuid.UUID) {
	t.Helper()
	if got == nil {
		t.Fatal("user is nil")
	}
	if got.Id != wantID.String() {
		t.Fatalf("user ID = %q, want %q", got.Id, wantID)
	}
	if got.Email != "" || got.Phone != "" {
		t.Fatalf("user %s exposed protected fields: email=%q phone=%q", got.Id, got.Email, got.Phone)
	}
}

// assertContactsHidden fails t unless got carries no protected contact fields.
// Use it where the expected ID is not known independently of got; deriving the
// want from the value under test would make the ID check a tautology.
func assertContactsHidden(t *testing.T, got *carpoolv1.User) {
	t.Helper()
	if got == nil {
		t.Fatal("user is nil")
	}
	if got.Email != "" || got.Phone != "" {
		t.Errorf("user %s exposed protected fields: email=%q phone=%q", got.Id, got.Email, got.Phone)
	}
}
