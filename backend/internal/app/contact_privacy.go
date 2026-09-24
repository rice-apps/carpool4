package app

import (
	"context"

	"github.com/google/uuid"
)

// projectUser returns a User from record. When includeContact is false, the
// returned profile omits email and phone.
func projectUser(record UserRecord, includeContact bool) User {
	user := User{ID: record.ID, FirstName: record.FirstName, LastName: record.LastName}
	// Names stay visible even when contact details do not.
	if !includeContact {
		return user
	}
	user.Email = record.Email
	user.Phone = record.Phone
	return user
}

// projectRides returns rides with owner and rider contacts visible to viewerID
// only when that viewer is allowed to see them. It uses tx for the visibility
// check and returns an error if that check fails.
func projectRides(ctx context.Context, tx Transaction, viewerID uuid.UUID, rides []LoadedRide) ([]Ride, error) {
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
	// A set makes each owner's and rider's permission easy to check.
	visibleSet := make(map[uuid.UUID]struct{}, len(visible))
	for _, id := range visible {
		visibleSet[id] = struct{}{}
	}

	result := make([]Ride, len(rides))
	// Rebuild each ride with only the contacts this viewer may see.
	for i, loaded := range rides {
		_, ownerVisible := visibleSet[loaded.Owner.ID]
		riders := make([]User, len(loaded.Riders))
		for j, rider := range loaded.Riders {
			_, contactVisible := visibleSet[rider.ID]
			riders[j] = projectUser(rider, contactVisible || rider.ID == viewerID)
		}
		result[i] = Ride{
			ID:                loaded.ID,
			DepartureDate:     loaded.DepartureDate,
			DepartureLocation: loaded.Departure,
			ArrivalLocation:   loaded.Arrival,
			Owner:             projectUser(loaded.Owner, ownerVisible || loaded.Owner.ID == viewerID),
			Riders:            riders,
			Notes:             loaded.Notes,
			Capacity:          loaded.Capacity,
			Status:            loaded.Status,
		}
	}
	return result, nil
}
