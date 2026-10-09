package app

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// getVisibleProfile returns a stored profile with contacts allowed for currentUser.
// Self can see their contacts; shared-ride permission is checked through tx.
// Guests receive no contacts. A failed permission check returns no profile and
// the lookup error. The stored record is unchanged.
func getVisibleProfile(ctx context.Context, tx Transaction, currentUser CurrentUser, record UserRecord) (User, error) {
	if currentUser.UserID == uuid.Nil {
		return buildUserResult(record, false), nil
	}
	if currentUser.UserID == record.ID {
		return buildUserResult(record, true), nil
	}
	visibleIDs, err := tx.Users().ListContactVisibleUserIDs(ctx, currentUser.UserID, []uuid.UUID{record.ID})
	if err != nil {
		return User{}, err
	}
	return buildUserResult(record, slices.Contains(visibleIDs, record.ID)), nil
}

// buildUserResult is the internal copying step used by the privacy helpers.
// They decide contact permission before calling it. It does not query storage.
func buildUserResult(record UserRecord, includeContact bool) User {
	user := User{ID: record.ID, FirstName: record.FirstName, LastName: record.LastName}
	// Names stay visible even when contact details do not.
	if !includeContact {
		return user
	}
	user.Email = record.Email
	user.Phone = record.Phone
	return user
}

// buildRidesForViewer builds ride results with the fields viewerID may receive.
// A guest (uuid.Nil) receives trip facts without people or notes. For a signed-in
// viewer, it uses tx to check contact permissions in one query. A failed lookup
// returns an error. It preserves ride order and leaves the stored data unchanged.
func buildRidesForViewer(ctx context.Context, tx Transaction, viewerID uuid.UUID, rides []LoadedRide) ([]Ride, error) {
	visibleSet := make(map[uuid.UUID]struct{})
	if viewerID != uuid.Nil {
		// Resolve every contact decision in the same snapshot as the loaded rides.
		targets := make([]uuid.UUID, 0)
		for _, ride := range rides {
			targets = append(targets, ride.Owner.ID)
			for _, rider := range ride.Riders {
				targets = append(targets, rider.ID)
			}
		}
		visible, err := tx.Users().ListContactVisibleUserIDs(ctx, viewerID, targets)
		if err != nil {
			return nil, err
		}
		for _, id := range visible {
			visibleSet[id] = struct{}{}
		}
	}

	result := make([]Ride, len(rides))
	// Build trip facts for everyone; add personal fields only for a verified viewer.
	for i, loaded := range rides {
		result[i] = Ride{
			ID:                loaded.ID,
			DepartureTime:     loaded.DepartureTime,
			DepartureLocation: loaded.Departure,
			ArrivalLocation:   loaded.Arrival,
			Capacity:          loaded.Capacity,
			OccupiedSeats:     int32(len(loaded.Riders)),
			Status:            loaded.Status,
		}
		if viewerID == uuid.Nil {
			continue
		}
		_, ownerVisible := visibleSet[loaded.Owner.ID]
		riders := make([]User, len(loaded.Riders))
		for j, rider := range loaded.Riders {
			_, contactVisible := visibleSet[rider.ID]
			riders[j] = buildUserResult(rider, contactVisible || rider.ID == viewerID)
		}
		result[i].Owner = buildUserResult(loaded.Owner, ownerVisible || loaded.Owner.ID == viewerID)
		result[i].Riders = riders
		result[i].Notes = loaded.Notes
	}
	return result, nil
}
