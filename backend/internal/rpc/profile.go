package rpc

import (
	"connectrpc.com/connect"
	"context"
	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// GetUser reads a profile for the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.GetUserRequest]): an optional target user ID string
//     in UUID format; omit it to select the caller's own profile.
//
// It returns a user with caller-visible contacts or a Connect error on failure.
func (s *Server) GetUser(ctx context.Context, req *connect.Request[v1.GetUserRequest]) (*connect.Response[v1.GetUserResponse], error) {
	// TODO(student): Implement GetUser RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}

// UpdateUser saves the verified caller's profile.
//
// Inputs:
//   - ctx (context.Context): supplies verified caller ID/email and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[v1.UpdateUserRequest]): the caller's names and phone.
//
// It returns the updated user on success or a Connect error on failure.
func (s *Server) UpdateUser(ctx context.Context, req *connect.Request[v1.UpdateUserRequest]) (*connect.Response[v1.UpdateUserResponse], error) {
	// TODO(student): Implement UpdateUser RPC adapter.
	return nil, toConnectError(app.ErrNotImplemented)
}
