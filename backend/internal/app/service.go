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

// requireSignIn checks that this operation received a signed-in user's ID.
// An empty ID returns ErrUnauthenticated; token verification happens before RPC.
func requireSignIn(currentUser CurrentUser) error {
	if currentUser.UserID == uuid.Nil {
		return ErrUnauthenticated
	}
	return nil
}
