package rpc

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

func TestGetUserRejectsMalformedTargetID(t *testing.T) {
	t.Parallel()
	target := "not-a-uuid"
	fake := &fakeDatabase{}

	_, err := newTestServer(fake).GetUser(testContext(uuid.New(), "caller@rice.edu"),
		connect.NewRequest(&v1.GetUserRequest{UserId: &target}))

	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("GetUser(%q) code = %v, want %v", target, got, connect.CodeInvalidArgument)
	}
	if fake.called(opBegin) {
		t.Error("GetUser reached storage with an unparseable target ID")
	}
}

// TestGetUserKeepsStorageDetailOffTheWire checks the two halves of the error
// contract: clients see a fixed message, and the original cause stays reachable
// for server logs.
func TestGetUserKeepsStorageDetailOffTheWire(t *testing.T) {
	t.Parallel()
	private := errors.New("sql: private detail")
	wrapped := fmt.Errorf("find user: %w: %w", app.ErrUserNotFound, private)
	fake := (&fakeDatabase{}).failing(opFindUser, wrapped)

	_, err := newTestServer(fake).GetUser(testContext(uuid.New(), "caller@rice.edu"),
		connect.NewRequest(&v1.GetUserRequest{}))

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("GetUser() error = %v, want a *connect.Error", err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("code = %v, want %v", connectErr.Code(), connect.CodeNotFound)
	}
	if connectErr.Message() != "user not found" {
		t.Errorf("wire message = %q, want %q", connectErr.Message(), "user not found")
	}
	if strings.Contains(connectErr.Message(), private.Error()) {
		t.Errorf("wire message %q leaked the storage detail", connectErr.Message())
	}
	if !errors.Is(err, private) {
		t.Error("the storage cause is no longer reachable for logs")
	}
	if loggedCause(err) != wrapped {
		t.Errorf("loggedCause() = %v, want %v", loggedCause(err), wrapped)
	}
}

func TestUpdateUserSavesTheVerifiedCallersProfile(t *testing.T) {
	t.Parallel()
	caller := uuid.New()
	fake := &fakeDatabase{}

	got, err := newTestServer(fake).UpdateUser(testContext(caller, "caller@rice.edu"),
		connect.NewRequest(&v1.UpdateUserRequest{FirstName: "Ada", LastName: "Lovelace", Phone: "713-555-0100"}))
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	// The identity comes from the token, never from the request body.
	if got.Msg.User.Id != caller.String() {
		t.Errorf("saved profile ID = %q, want the verified caller %q", got.Msg.User.Id, caller)
	}
	if got.Msg.User.Email != "caller@rice.edu" {
		t.Errorf("saved email = %q, want the token's email", got.Msg.User.Email)
	}
}
