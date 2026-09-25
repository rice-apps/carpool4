package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/db/sqlc"
)

// ListContactVisibleUserIDs returns target IDs whose contact details viewerID
// may see, including self and users who shared any ride.
//
// Inputs:
//   - ctx (context.Context): passes request timeout or cancellation to the database query.
//   - viewerID (uuid.UUID): the user requesting contacts.
//   - targetIDs ([]uuid.UUID): users to check in one batch.
//
// It returns visible IDs or an error, including a classified transaction
// conflict.
func (u userRepository) ListContactVisibleUserIDs(ctx context.Context, viewerID uuid.UUID, targetIDs []uuid.UUID) ([]uuid.UUID, error) {
	// Ask the query for self and shared-ride contacts in one batch.
	ids, err := u.queries.ListContactVisibleUserIDs(ctx, sqlc.ListContactVisibleUserIDsParams{ViewerID: viewerID, TargetIds: targetIDs})
	return ids, classifyTransactionError(err)
}
