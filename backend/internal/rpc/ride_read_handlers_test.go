package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// TestRideReadHandlersParseWireValues checks invalid ride and location UUIDs become InvalidArgument.
func TestRideReadHandlersParseWireValues(t *testing.T) {
	server := newTestServer(&fakeStore{})
	ctx := testContext(uuid.New(), "viewer@rice.edu")
	bad := "not-a-uuid"
	if _, err := server.GetRide(ctx, connect.NewRequest(&v1.GetRideRequest{Id: bad})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetRide malformed ID error = %v", err)
	}
	if _, err := server.ListRides(ctx, connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &bad})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ListRides malformed filter error = %v", err)
	}
}

// TestListRidesRejectsMalformedArrivalFilter checks the second location filter is parsed.
func TestListRidesRejectsMalformedArrivalFilter(t *testing.T) {
	server := newTestServer(&fakeStore{})
	bad := "not-a-uuid"

	_, err := server.ListRides(testContext(uuid.New(), "viewer@rice.edu"), connect.NewRequest(&v1.ListRidesRequest{ArrivalLocationId: &bad}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ListRides malformed arrival filter error = %v", err)
	}
}

// TestListRidesForwardsTimeBounds checks both departure bounds reach the service.
func TestListRidesForwardsTimeBounds(t *testing.T) {
	store := &fakeStore{}
	after, before := time.Now().Add(time.Hour), time.Now().Add(48*time.Hour)

	_, err := newTestServer(store).ListRides(testContext(uuid.New(), "viewer@rice.edu"), connect.NewRequest(&v1.ListRidesRequest{
		DepartureAfter: timestamppb.New(after), DepartureBefore: timestamppb.New(before),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if store.query == nil || store.query.DepartureBefore == nil || !store.query.DepartureBefore.Equal(before.UTC()) {
		t.Fatalf("DepartureBefore = %v, want %v", store.query, before.UTC())
	}
	if store.query.DepartureAfter.Before(after.UTC()) {
		t.Fatalf("DepartureAfter = %v, want at least %v", store.query.DepartureAfter, after.UTC())
	}
}

// TestListLocationsMapsServiceFailure checks a storage failure is not reported as success.
func TestListLocationsMapsServiceFailure(t *testing.T) {
	store := &fakeStore{locationsErr: errors.New("locations unavailable")}

	_, err := newTestServer(store).ListLocations(context.Background(), connect.NewRequest(&v1.ListLocationsRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("ListLocations error = %v, want internal", err)
	}
	if !errors.Is(err, store.locationsErr) {
		t.Error("ListLocations dropped the underlying cause")
	}
}
