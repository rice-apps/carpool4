package rpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
)

// requireSignedInUser returns the ID and email of the person making this request.
// It reads the identity already verified by authentication, not a saved profile.
// Missing identity or an empty user ID returns app.ErrUnauthenticated.
func requireSignedInUser(ctx context.Context) (app.CurrentUser, error) {
	user, ok := auth.GetSignedInUser(ctx)
	if !ok {
		return app.CurrentUser{}, app.ErrUnauthenticated
	}
	if user.ID == uuid.Nil {
		return app.CurrentUser{}, fmt.Errorf("%w: invalid caller ID", app.ErrUnauthenticated)
	}
	// Only authentication middleware can supply these fields; request IDs name targets.
	return app.CurrentUser{UserID: user.ID, Email: user.Email}, nil
}
