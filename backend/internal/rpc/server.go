package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1/carpoolv1connect"
)

// Server adapts Connect requests to the application service.
//
// Fields:
//   - service: the application use cases invoked by RPC handlers.
type Server struct {
	service *app.Service
}

var _ carpoolv1connect.UserServiceHandler = (*Server)(nil)
var _ carpoolv1connect.RideServiceHandler = (*Server)(nil)

// NewServer returns an RPC adapter for service.
func NewServer(service *app.Service) *Server {
	return &Server{service: service}
}

// Register adds the user and ride handlers with authentication, validation,
// logging, and a request body limit.
//
// Inputs:
//   - mux (*http.ServeMux): the HTTP router to register with.
//   - server (*Server): the handlers' application adapter.
//   - authInterceptor (connect.UnaryInterceptorFunc): supplies verified caller
//     details or rejects unauthenticated requests.
//   - logger (*slog.Logger): request and error logging.
//
// It returns an error if validation setup fails.
func Register(mux *http.ServeMux, server *Server, authInterceptor connect.UnaryInterceptorFunc, logger *slog.Logger) error {
	validateInterceptor, err := newValidationInterceptor()
	if err != nil {
		return fmt.Errorf("failed to create validation interceptor: %w", err)
	}
	// Missing credentials are allowed only for public ride reads. Supplied
	// credentials and all other procedures still use strict authentication.
	authenticate := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		strict := authInterceptor(next)
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if isPublicRideRead(req.Spec().Procedure) && len(req.Header().Values("Authorization")) == 0 {
				return next(ctx, req)
			}
			return strict(ctx, req)
		}
	})
	// Apply shared request checks and cap the size of incoming bodies.
	opts := []connect.HandlerOption{
		connect.WithInterceptors(newLoggingInterceptor(logger), authenticate, validateInterceptor),
		connect.WithReadMaxBytes(64 << 10),
	}
	mux.Handle(carpoolv1connect.NewUserServiceHandler(server, opts...))
	mux.Handle(carpoolv1connect.NewRideServiceHandler(server, opts...))
	return nil
}

func isPublicRideRead(procedure string) bool {
	switch procedure {
	case carpoolv1connect.RideServiceListRidesProcedure,
		carpoolv1connect.RideServiceGetRideProcedure,
		carpoolv1connect.RideServiceListLocationsProcedure:
		return true
	default:
		return false
	}
}
