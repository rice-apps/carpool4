package rpc_test

import (
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
)

// newRPCServer wires a supplied database into the public RPC server for integration tests.
func newRPCServer(database app.Database) *rpc.Server {
	return rpc.NewServer(app.NewService(database))
}
