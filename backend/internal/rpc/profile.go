package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// GetUser reads a profile for the verified caller.
//
// Inputs:
//   - ctx (context.Context): supplies the verified caller and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.GetUserRequest]): an optional target user ID string
//     in UUID format; omit it to select the caller's own profile.
//
// It returns a user with caller-visible contacts or a Connect error on failure.
func (s *Server) GetUser(ctx context.Context, req *connect.Request[carpoolv1.GetUserRequest]) (*connect.Response[carpoolv1.GetUserResponse], error) {
	// TODO(student): Implement GetUser RPC adapter.
	var targetID *uuid.UUID
	if req.Msg.UserId != nil {
		id, err := uuid.Parse(*req.Msg.UserId)
		if err != nil {
			return nil, err
		}
		targetID = &id
	}

	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	user, err := s.service.GetUser(ctx, currentUser, targetID)
	if err != nil {
		return nil, err
	}

	response := &carpoolv1.GetUserResponse{User: convertUserToProto(user)}
	return connect.NewResponse(response), nil
}

// UpdateUser saves the verified caller's profile.
//
// Inputs:
//   - ctx (context.Context): supplies verified caller ID/email and passes
//     timeout/cancellation to the service call.
//   - req (*connect.Request[carpoolv1.UpdateUserRequest]): the caller's names and phone.
//
// It returns the updated user on success or a Connect error on failure.
func (s *Server) UpdateUser(ctx context.Context, req *connect.Request[carpoolv1.UpdateUserRequest]) (*connect.Response[carpoolv1.UpdateUserResponse], error) {
	// TODO(student): Implement UpdateUser RPC adapter.
	firstName := req.Msg.FirstName
	lastName := req.Msg.LastName
	phone := req.Msg.Phone
	input := app.UserInput{
		FirstName: firstName,
		LastName:  lastName,
		Phone:     phone,
	}
	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.service.UpdateUser(ctx, currentUser, input)
	if err != nil {
		return nil, err
	}

	response := &carpoolv1.UpdateUserResponse{User: convertUserToProto(user)}
	return connect.NewResponse(response), nil
}
