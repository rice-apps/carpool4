package rpc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1/carpoolv1connect"
)

// registeredServer starts both HTTP services with database, test identity, and captured logs.
// The returned server is closed by t's cleanup; the buffer lets tests inspect middleware output.
func registeredServer(t *testing.T, database app.Database) (*httptest.Server, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	identity := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			id, err := uuid.Parse(req.Header().Get("X-Test-User"))
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
			}
			return next(auth.WithUser(ctx, auth.User{ID: id, Email: "caller@rice.edu"}), req)
		}
	}
	mux := http.NewServeMux()
	if err := Register(mux, newTestServer(database), identity, slog.New(slog.NewJSONHandler(&logs, nil))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, &logs
}

func TestRegisterPublicRideReads(t *testing.T) {
	owner := testUser(uuid.New(), "owner@rice.edu")
	passenger := testUser(uuid.New(), "passenger@rice.edu")
	rideID := uuid.New()
	ride := app.LoadedRide{
		ID:            rideID,
		DepartureTime: time.Now().Add(time.Hour),
		Owner:         owner,
		Riders:        []app.UserRecord{owner, passenger},
		Notes:         "Call owner at 713-555-0100",
		Capacity:      3,
		Status:        app.RideStatusActive,
		Departure:     app.LocationRecord{ID: uuid.New(), Title: "Rice"},
		Arrival:       app.LocationRecord{ID: uuid.New(), Title: "IAH"},
	}
	server, _ := registeredServer(t, &fakeStore{
		rides:     map[uuid.UUID]app.LoadedRide{rideID: ride},
		listed:    []app.LoadedRide{ride},
		locations: []app.LocationRecord{ride.Departure, ride.Arrival},
	})
	rides := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)
	users := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)

	listed, err := rides.ListRides(context.Background(), connect.NewRequest(&v1.ListRidesRequest{}))
	if err != nil || len(listed.Msg.Rides) != 1 {
		t.Fatalf("public list = %v, %v", listed, err)
	}
	assertPublicRide(t, listed.Msg.Rides[0], rideID)
	got, err := rides.GetRide(context.Background(), connect.NewRequest(&v1.GetRideRequest{Id: rideID.String()}))
	if err != nil {
		t.Fatalf("public ride = %v", err)
	}
	assertPublicRide(t, got.Msg.Ride, rideID)
	locations, err := rides.ListLocations(context.Background(), connect.NewRequest(&v1.ListLocationsRequest{}))
	if err != nil || len(locations.Msg.Locations) != 2 {
		t.Fatalf("public locations = %v, %v", locations, err)
	}

	if _, err := rides.ListMyRides(context.Background(), connect.NewRequest(&v1.ListMyRidesRequest{})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("ListMyRides without auth = %v", err)
	}
	if _, err := users.GetUser(context.Background(), connect.NewRequest(&v1.GetUserRequest{})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("GetUser without auth = %v", err)
	}

	for _, authorization := range []string{"Bearer invalid", "Basic anything", ""} {
		t.Run("credential_"+authorization, func(t *testing.T) {
			listReq := connect.NewRequest(&v1.ListRidesRequest{})
			listReq.Header().Set("Authorization", authorization)
			if _, err := rides.ListRides(context.Background(), listReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
				t.Fatalf("ListRides with %q = %v", authorization, err)
			}
			getReq := connect.NewRequest(&v1.GetRideRequest{Id: rideID.String()})
			getReq.Header().Set("Authorization", authorization)
			if _, err := rides.GetRide(context.Background(), getReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
				t.Fatalf("GetRide with %q = %v", authorization, err)
			}
			locationReq := connect.NewRequest(&v1.ListLocationsRequest{})
			locationReq.Header().Set("Authorization", authorization)
			if _, err := rides.ListLocations(context.Background(), locationReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
				t.Fatalf("ListLocations with %q = %v", authorization, err)
			}
		})
	}

	if _, err := rides.GetRide(context.Background(), connect.NewRequest(&v1.GetRideRequest{Id: "bad-id"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("guest GetRide bad ID = %v", err)
	}
	badID := "bad-id"
	if _, err := rides.ListRides(context.Background(), connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &badID})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("guest ListRides bad filter = %v", err)
	}
}

func assertPublicRide(t *testing.T, ride *v1.Ride, id uuid.UUID) {
	t.Helper()
	if ride == nil || ride.Id != id.String() || ride.Owner != nil || len(ride.Riders) != 0 || ride.Notes != "" || ride.OccupiedSeats != 2 || ride.Capacity != 3 {
		t.Fatalf("public ride = %#v", ride)
	}
}

// TestRegisterServesBothServices checks both generated services are reachable through Register.
func TestRegisterServesBothServices(t *testing.T) {
	// Removing either generated registration must break an authenticated successful RPC.
	id := uuid.New()
	server, logs := registeredServer(t, &fakeStore{users: map[uuid.UUID]app.UserRecord{id: testUser(id, "caller@rice.edu")}})
	userClient := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)
	userReq := connect.NewRequest(&v1.GetUserRequest{})
	userReq.Header().Set("X-Test-User", id.String())
	user, err := userClient.GetUser(context.Background(), userReq)
	if err != nil || user.Msg.User.Id != id.String() {
		t.Fatalf("GetUser = %v, %v", user, err)
	}
	rideClient := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)
	rideReq := connect.NewRequest(&v1.ListLocationsRequest{})
	rideReq.Header().Set("X-Test-User", id.String())
	if _, err := rideClient.ListLocations(context.Background(), rideReq); err != nil {
		t.Fatal(err)
	}
	if strings.Count(logs.String(), `"msg":"request handled"`) != 2 {
		t.Fatalf("success logs = %s", logs)
	}
}

// TestRegisterOrdersLoggingAuthenticationAndValidation checks rejection precedence and single logging.
func TestRegisterOrdersLoggingAuthenticationAndValidation(t *testing.T) {
	// Removing auth/validation, swapping them, or putting logging inside either loses the expected rejection/log.
	for _, authenticated := range []bool{false, true} {
		t.Run(map[bool]string{false: "unauthenticated", true: "invalid authenticated request"}[authenticated], func(t *testing.T) {
			server, logs := registeredServer(t, &fakeStore{})
			client := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)
			// Protovalidate rejects the raw 101-character name; the application
			// would trim it to "A" and successfully save without this interceptor.
			req := connect.NewRequest(&v1.UpdateUserRequest{FirstName: strings.Repeat(" ", 100) + "A", LastName: "User"})
			want := connect.CodeUnauthenticated
			if authenticated {
				req.Header().Set("X-Test-User", uuid.NewString())
				want = connect.CodeInvalidArgument
			}
			_, err := client.UpdateUser(context.Background(), req)
			if connect.CodeOf(err) != want {
				t.Fatalf("error = %v, want %v", err, want)
			}
			if strings.Count(strings.TrimSpace(logs.String()), "\n") != 0 || !strings.Contains(logs.String(), `"code":"`+want.String()+`"`) {
				t.Fatalf("rejection log = %s", logs)
			}
		})
	}
}

// TestRegisterLimitsRequestBody checks oversized payloads fail at the HTTP read limit.
func TestRegisterLimitsRequestBody(t *testing.T) {
	// Omitting the configured read limit changes this rejection to validation or auth failure.
	server, _ := registeredServer(t, &fakeStore{})
	client := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)
	_, err := client.CreateRide(context.Background(), connect.NewRequest(&v1.CreateRideRequest{Notes: strings.Repeat("x", 70<<10)}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("error = %v, want resource_exhausted", err)
	}
}

// TestRegisterSanitizesUnknownErrorsAndLogsCauseOnce checks private storage details stay out of client errors.
func TestRegisterSanitizesUnknownErrorsAndLogsCauseOnce(t *testing.T) {
	// Returning the storage cause exposes a secret; logging the public error loses diagnostics.
	const secret = "postgres://service-role:secret@db.internal/carpool"
	server, logs := registeredServer(t, &fakeStore{findErr: errors.New(secret)})
	client := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)
	req := connect.NewRequest(&v1.GetUserRequest{})
	req.Header().Set("X-Test-User", uuid.NewString())
	_, err := client.GetUser(context.Background(), req)
	if connect.CodeOf(err) != connect.CodeInternal || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), "internal server error") {
		t.Fatalf("client error = %v", err)
	}
	if strings.Count(logs.String(), secret) != 1 || !strings.Contains(logs.String(), `"code":"internal"`) || !strings.Contains(logs.String(), carpoolv1connect.UserServiceGetUserProcedure) {
		t.Fatalf("cause log = %s", logs)
	}
}
