package app

import (
	"time"

	"github.com/google/uuid"
)

// Actor identifies the verified caller of an application method.
//
// Fields:
//   - UserID: Supabase user ID used for authorization and profile ownership.
//   - Email: verified address stored when the caller saves their profile.
type Actor struct {
	UserID uuid.UUID
	Email  string
}

// UserInput contains the profile details a caller may change.
//
// Fields:
//   - FirstName: caller's given name; must not be blank.
//   - LastName: caller's family name; must not be blank.
//   - Phone: optional contact number, needed before posting a ride.
//
// The profile ID and email come from Actor instead.
type UserInput struct {
	FirstName string
	LastName  string
	Phone     string
}

// ListRidesInput narrows the rides shown in discovery.
//
// Fields:
//   - DepartureLocationID: starting place; nil accepts any location.
//   - ArrivalLocationID: destination; nil accepts any location.
//   - DepartureAfter: requested earliest departure; nil uses the default cutoff.
//   - DepartureBefore: latest departure; nil leaves the end of the range open.
type ListRidesInput struct {
	DepartureLocationID *uuid.UUID
	ArrivalLocationID   *uuid.UUID
	DepartureAfter      *time.Time
	DepartureBefore     *time.Time
}

// User is a profile returned to a particular viewer.
//
// Fields:
//   - ID: user's stable Supabase ID.
//   - FirstName: given name visible to every viewer.
//   - LastName: family name visible to every viewer.
//   - Email: empty when the viewer cannot see contact details.
//   - Phone: empty when the viewer cannot see contact details.
type User struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

// Ride is a trip returned with contact details allowed for one viewer.
//
// Fields:
//   - ID: stable ride ID.
//   - DepartureTime: scheduled departure time.
//   - DepartureLocation: selected starting place.
//   - ArrivalLocation: selected destination.
//   - Owner: driver profile with viewer-allowed contact details.
//   - Riders: occupant profiles, including the owner, with viewer-allowed contacts.
//   - Notes: trip details supplied by the driver.
//   - Capacity: total seats, including the driver's seat.
//   - Status: whether the ride is active or cancelled.
type Ride struct {
	ID                uuid.UUID
	DepartureTime     time.Time
	DepartureLocation LocationRecord
	ArrivalLocation   LocationRecord
	Owner             User
	Riders            []User
	Notes             string
	Capacity          int32
	Status            RideStatus
}
