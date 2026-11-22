package rpc_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/auth"
	"github.com/rice-apps/carpool4/backend/internal/db/postgres"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1/carpoolv1connect"
	"github.com/rice-apps/carpool4/backend/internal/rpc"
	"github.com/rice-apps/carpool4/backend/internal/testutil"
)

const testUserHeader = "X-Carpool-Test-User"

// TestRPCContactPrivacyAcrossUserRideAndMutationResponses checks contact projection over real HTTP RPCs.
// A shared ride reveals contacts to its participant while unrelated callers see redacted fields.
func TestRPCContactPrivacyAcrossUserRideAndMutationResponses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	server := newRPCServer(postgres.NewDatabase(db))

	ownerID := uuid.New()
	passengerID := uuid.New()
	strangerID := uuid.New()
	insertRPCUser(t, db, ownerID, "Owner", "owner@rice.edu", "713-555-0101")
	insertRPCUser(t, db, passengerID, "Passenger", "passenger@rice.edu", "713-555-0102")
	insertRPCUser(t, db, strangerID, "Stranger", "stranger@rice.edu", "713-555-0103")
	departureID, arrivalID := insertRPCLocations(t, db)
	created, err := server.CreateRide(dbCaller(ownerID), connect.NewRequest(&v1.CreateRideRequest{
		DepartureTime:       timestamppb.New(time.Now().Add(24 * time.Hour)),
		DepartureLocationId: departureID.String(), ArrivalLocationId: arrivalID.String(), Capacity: 3,
	}))
	if err != nil {
		t.Fatalf("create ride: %v", err)
	}

	mux := http.NewServeMux()
	if err := rpc.Register(mux, server, testIdentityInterceptor(), slog.New(slog.NewJSONHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	userClient := carpoolv1connect.NewUserServiceClient(httpServer.Client(), httpServer.URL)
	rideClient := carpoolv1connect.NewRideServiceClient(httpServer.Client(), httpServer.URL)

	joinReq := connect.NewRequest(&v1.JoinRideRequest{RideId: created.Msg.Ride.Id})
	setTestCaller(joinReq, passengerID)
	joined, err := rideClient.JoinRide(context.Background(), joinReq)
	if err != nil {
		t.Fatalf("JoinRide: %v", err)
	}
	assertRPCContact(t, joined.Msg.Ride.Owner, ownerID, "owner@rice.edu", "713-555-0101")
	assertRideRiderContact(t, joined.Msg.Ride, passengerID, "passenger@rice.edu", "713-555-0102")

	getUserReq := connect.NewRequest(&v1.GetUserRequest{UserId: stringPointer(ownerID.String())})
	setTestCaller(getUserReq, strangerID)
	gotUser, err := userClient.GetUser(context.Background(), getUserReq)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	assertRPCContactRedacted(t, gotUser.Msg.User, ownerID)

	getRideReq := connect.NewRequest(&v1.GetRideRequest{Id: created.Msg.Ride.Id})
	setTestCaller(getRideReq, strangerID)
	gotRide, err := rideClient.GetRide(context.Background(), getRideReq)
	if err != nil {
		t.Fatalf("GetRide: %v", err)
	}
	assertRideContactsRedacted(t, gotRide.Msg.Ride, ownerID, passengerID)

	listReq := connect.NewRequest(&v1.ListRidesRequest{})
	setTestCaller(listReq, strangerID)
	listed, err := rideClient.ListRides(context.Background(), listReq)
	if err != nil {
		t.Fatalf("ListRides: %v", err)
	}
	if len(listed.Msg.Rides) != 1 {
		t.Fatalf("ListRides returned %d rides, want 1", len(listed.Msg.Rides))
	}
	assertRideContactsRedacted(t, listed.Msg.Rides[0], ownerID, passengerID)
}

// testIdentityInterceptor maps the test-only user header to a verified context identity.
// Malformed UUIDs are rejected as Unauthenticated before reaching handlers.
func testIdentityInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			id, err := uuid.Parse(req.Header().Get(testUserHeader))
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}
			ctx = auth.WithUser(ctx, auth.User{ID: id, Email: id.String() + "@rice.edu"})
			return next(ctx, req)
		}
	}
}

// setTestCaller puts id in the test identity header of any Connect request.
func setTestCaller[T any](req *connect.Request[T], id uuid.UUID) {
	req.Header().Set("Authorization", "Bearer test-fixture")
	req.Header().Set(testUserHeader, id.String())
}

// insertRPCUser creates matching auth and profile rows with the supplied identity and contact fields.
// It fails t if either insert fails.
func insertRPCUser(t *testing.T, db *sql.DB, id uuid.UUID, firstName, email, phone string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO auth_users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("insert auth user: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, first_name, last_name, email, phone) VALUES ($1, $2, 'User', $3, $4)`, id, firstName, email, phone); err != nil {
		t.Fatalf("insert public user: %v", err)
	}
}

// insertRPCLocations persists two distinct locations and returns their departure and arrival UUIDs.
func insertRPCLocations(t *testing.T, db *sql.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	departureID, arrivalID := uuid.New(), uuid.New()
	if _, err := db.Exec(`INSERT INTO locations (id, title, address) VALUES ($1, 'Departure', $3), ($2, 'Arrival', $4)`, departureID, arrivalID, departureID.String(), arrivalID.String()); err != nil {
		t.Fatalf("insert locations: %v", err)
	}
	return departureID, arrivalID
}

// assertRPCContactRedacted fails t unless got keeps wantID but hides every protected contact field.
func assertRPCContactRedacted(t *testing.T, got *v1.User, wantID uuid.UUID) {
	t.Helper()
	if got == nil {
		t.Fatal("user is nil")
	}
	if got.Id != wantID.String() {
		t.Fatalf("user ID = %q, want %q", got.Id, wantID)
	}
	if got.Email != "" || got.Phone != "" {
		t.Fatalf("user %s exposed protected fields: email=%q phone=%q", got.Id, got.Email, got.Phone)
	}
}

// assertRPCContact fails t unless got contains the expected ID and visible contact details.
func assertRPCContact(t *testing.T, got *v1.User, wantID uuid.UUID, email, phone string) {
	t.Helper()
	if got == nil {
		t.Fatal("user is nil")
	}
	if got.Id != wantID.String() || got.Email != email || got.Phone != phone {
		t.Fatalf("user contact = %+v, want id=%s email=%q phone=%q", got, wantID, email, phone)
	}
}

// assertRideRiderContact finds riderID in got and checks that rider's contact fields are visible.
func assertRideRiderContact(t *testing.T, got *v1.Ride, riderID uuid.UUID, email, phone string) {
	t.Helper()
	for _, rider := range got.Riders {
		if rider.Id == riderID.String() {
			assertRPCContact(t, rider, riderID, email, phone)
			return
		}
	}
	t.Fatalf("ride omitted rider %s", riderID)
}

// assertRideContactsRedacted checks both owner and passenger contacts are hidden in got.
func assertRideContactsRedacted(t *testing.T, got *v1.Ride, ownerID, passengerID uuid.UUID) {
	t.Helper()
	if got == nil {
		t.Fatal("ride is nil")
	}
	assertRPCContactRedacted(t, got.Owner, ownerID)
	wanted := map[uuid.UUID]bool{ownerID: false, passengerID: false}
	for _, rider := range got.Riders {
		id, err := uuid.Parse(rider.Id)
		if err != nil {
			t.Fatalf("invalid rider ID %q: %v", rider.Id, err)
		}
		if _, ok := wanted[id]; ok {
			assertRPCContactRedacted(t, rider, id)
			wanted[id] = true
		}
	}
	for id, found := range wanted {
		if !found {
			t.Fatalf("ride omitted rider %s", id)
		}
	}
}

// stringPointer returns a pointer to value for optional protobuf string fields in tests.
func stringPointer(value string) *string { return &value }
