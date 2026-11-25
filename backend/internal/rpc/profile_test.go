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

// TestGetUserRejectsMalformedID checks invalid target UUIDs become InvalidArgument.
func TestGetUserRejectsMalformedID(t *testing.T) {
	server := newTestServer(&fakeStore{})
	target := "not-a-uuid"
	_, err := server.GetUser(testContext(uuid.New(), "caller@rice.edu"), connect.NewRequest(&v1.GetUserRequest{UserId: &target}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetUser malformed ID error = %v", err)
	}
}

// TestGetUserMapsPublicErrorsAndRetainsPrivateCause checks safe not-found output and diagnostic unwrapping.
func TestGetUserMapsPublicErrorsAndRetainsPrivateCause(t *testing.T) {
	private := errors.New("sql: private detail")
	wrapped := fmt.Errorf("find user: %w: %w", app.ErrUserNotFound, private)
	server := newTestServer(&fakeStore{findErr: wrapped})

	_, err := server.GetUser(testContext(uuid.New(), "caller@rice.edu"), connect.NewRequest(&v1.GetUserRequest{}))
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeNotFound || connectErr.Message() != "user not found" {
		t.Fatalf("GetUser error = %v, want public not-found", err)
	}
	if strings.Contains(connectErr.Message(), private.Error()) || !errors.Is(err, private) || loggedCause(err) != wrapped {
		t.Fatalf("GetUser exposed or discarded private cause: %v", err)
	}
}
