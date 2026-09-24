package app

import "github.com/google/uuid"

// Service provides carpool operations using application-owned storage.
//
// Fields:
//   - database: starts a transaction for each operation that reads or writes data.
type Service struct {
	database Database
}

// NewService returns a Service that uses database for its storage access.
func NewService(database Database) *Service {
	return &Service{database: database}
}

// validateActor returns ErrUnauthenticated when actor has no verified ID.
func validateActor(actor Actor) error {
	if actor.ID == uuid.Nil {
		return ErrUnauthenticated
	}
	return nil
}
