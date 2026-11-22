package rpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// TestListLocationsConvertsResult checks public location conversion.
func TestListLocationsConvertsResult(t *testing.T) {
	location := app.LocationRecord{ID: uuid.New(), Title: "Rice", Address: "6100 Main St"}
	server := newTestServer(&fakeStore{locations: []app.LocationRecord{location}})

	response, err := server.ListLocations(context.Background(), connect.NewRequest(&v1.ListLocationsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Locations) != 1 || response.Msg.Locations[0].Id != location.ID.String() || response.Msg.Locations[0].Title != location.Title {
		t.Fatalf("ListLocations response = %#v", response.Msg.Locations)
	}
}
