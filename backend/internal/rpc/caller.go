package rpc

import (
	"context"
	"fmt"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	"github.com/google/uuid"
)

// actorFromContext returns the verified user's ID and email as an application actor.
// If ctx has no user or its ID is nil, it returns ErrUnauthenticated.
func actorFromContext(ctx context.Context) (app.Actor, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return app.Actor{}, app.ErrUnauthenticated
	}
	if user.ID == uuid.Nil {
		return app.Actor{}, fmt.Errorf("%w: invalid caller ID", app.ErrUnauthenticated)
	}
	// Only authentication middleware can supply these fields; request IDs name targets.
	return app.Actor{ID: user.ID, Email: user.Email}, nil
}
