package app

// Service provides carpool operations using application-owned storage.
// Protected operations require a verified, nonempty CurrentUser from the caller.
// Public ride reads also accept guests. Authentication belongs to the RPC layer.
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
