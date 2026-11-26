package rpc

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

func TestListLocationsConvertsEveryLocation(t *testing.T) {
	t.Parallel()
	rice := app.LocationRecord{ID: uuid.New(), Title: "Rice", Address: "6100 Main St"}
	airport := app.LocationRecord{ID: uuid.New(), Title: "IAH", Address: "2800 N Terminal Rd"}
	fake := &fakeDatabase{locations: []app.LocationRecord{rice, airport}}

	got, err := newTestServer(fake).ListLocations(context.Background(), connect.NewRequest(&carpoolv1.ListLocationsRequest{}))
	if err != nil {
		t.Fatalf("ListLocations() error = %v", err)
	}

	want := []*carpoolv1.Location{
		{Id: rice.ID.String(), Title: "Rice", Address: "6100 Main St"},
		{Id: airport.ID.String(), Title: "IAH", Address: "2800 N Terminal Rd"},
	}
	if diff := cmp.Diff(want, got.Msg.Locations, protocmp.Transform()); diff != "" {
		t.Errorf("ListLocations() mismatch (-want +got):\n%s", diff)
	}
}

// TestListLocationsReportsStorageFailure checks a failed read is not reported as
// an empty list, and that its cause stays available for server logs.
func TestListLocationsReportsStorageFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("locations unavailable")
	fake := (&fakeDatabase{}).failing(opListLocations, cause)

	_, err := newTestServer(fake).ListLocations(context.Background(), connect.NewRequest(&carpoolv1.ListLocationsRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("ListLocations() code = %v, want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, cause) {
		t.Errorf("ListLocations() error = %v, want it to wrap %v", err, cause)
	}
}
