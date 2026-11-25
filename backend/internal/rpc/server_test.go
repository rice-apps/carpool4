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
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	v1 "github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1/carpoolv1connect"
)

// registeredServer starts both services over HTTP with a test identity
// interceptor and captured logs. The returned buffer lets a test inspect what
// the middleware recorded.
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
		t.Fatalf("Register() error = %v", err)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, &logs
}

// publicRideFixture is a ride whose notes and people a guest must not see.
func publicRideFixture() app.LoadedRide {
	owner := testUser(uuid.New(), "owner@rice.edu")
	passenger := testUser(uuid.New(), "passenger@rice.edu")
	return app.LoadedRide{
		ID: uuid.New(), DepartureTime: time.Now().Add(time.Hour), Owner: owner,
		Riders: []app.UserRecord{owner, passenger}, Notes: "Call owner at 713-555-0100",
		Capacity: 3, Status: app.RideStatusActive,
		Departure: app.LocationRecord{ID: uuid.New(), Title: "Rice"},
		Arrival:   app.LocationRecord{ID: uuid.New(), Title: "IAH"},
	}
}

// assertGuestRide fails t unless ride carries trip facts and nothing personal.
// It diffs the whole message so a newly added field cannot leak unnoticed.
func assertGuestRide(t *testing.T, got *v1.Ride, want app.LoadedRide) {
	t.Helper()
	expected := &v1.Ride{
		Id:                want.ID.String(),
		DepartureTime:     got.GetDepartureTime(),
		DepartureLocation: &v1.Location{Id: want.Departure.ID.String(), Title: want.Departure.Title},
		ArrivalLocation:   &v1.Location{Id: want.Arrival.ID.String(), Title: want.Arrival.Title},
		Capacity:          want.Capacity,
		OccupiedSeats:     int32(len(want.Riders)),
		Status:            v1.RideStatus_RIDE_STATUS_ACTIVE,
		Riders:            []*v1.User{},
	}
	if diff := cmp.Diff(expected, got, protocmp.Transform()); diff != "" {
		t.Errorf("guest ride mismatch (-want +got):\n%s", diff)
	}
}

func TestRegisterServesGuestReads(t *testing.T) {
	t.Parallel()
	ride := publicRideFixture()
	server, _ := registeredServer(t, &fakeDatabase{
		rides:     map[uuid.UUID]app.LoadedRide{ride.ID: ride},
		discovery: []app.LoadedRide{ride},
		locations: []app.LocationRecord{ride.Departure, ride.Arrival},
	})
	rides := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)

	t.Run("ListRides", func(t *testing.T) {
		listed, err := rides.ListRides(context.Background(), connect.NewRequest(&v1.ListRidesRequest{}))
		if err != nil {
			t.Fatalf("ListRides() error = %v", err)
		}
		if len(listed.Msg.Rides) != 1 {
			t.Fatalf("ListRides() returned %d rides, want 1", len(listed.Msg.Rides))
		}
		assertGuestRide(t, listed.Msg.Rides[0], ride)
	})

	t.Run("GetRide", func(t *testing.T) {
		got, err := rides.GetRide(context.Background(), connect.NewRequest(&v1.GetRideRequest{Id: ride.ID.String()}))
		if err != nil {
			t.Fatalf("GetRide() error = %v", err)
		}
		assertGuestRide(t, got.Msg.Ride, ride)
	})

	t.Run("ListLocations", func(t *testing.T) {
		got, err := rides.ListLocations(context.Background(), connect.NewRequest(&v1.ListLocationsRequest{}))
		if err != nil {
			t.Fatalf("ListLocations() error = %v", err)
		}
		if len(got.Msg.Locations) != 2 {
			t.Errorf("ListLocations() returned %d locations, want 2", len(got.Msg.Locations))
		}
	})
}

func TestRegisterRefusesGuestsOnProtectedReads(t *testing.T) {
	t.Parallel()
	server, _ := registeredServer(t, &fakeDatabase{})
	rides := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)
	users := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)

	t.Run("ListMyRides", func(t *testing.T) {
		_, err := rides.ListMyRides(context.Background(), connect.NewRequest(&v1.ListMyRidesRequest{}))
		if got := connect.CodeOf(err); got != connect.CodeUnauthenticated {
			t.Errorf("ListMyRides() code = %v, want %v", got, connect.CodeUnauthenticated)
		}
	})
	t.Run("GetUser", func(t *testing.T) {
		_, err := users.GetUser(context.Background(), connect.NewRequest(&v1.GetUserRequest{}))
		if got := connect.CodeOf(err); got != connect.CodeUnauthenticated {
			t.Errorf("GetUser() code = %v, want %v", got, connect.CodeUnauthenticated)
		}
	})
}

// TestRegisterRefusesAnySuppliedCredential checks that the guest exemption
// applies only when no Authorization header is sent at all. A caller who offers
// a credential is held to it, so a malformed token cannot buy guest access.
func TestRegisterRefusesAnySuppliedCredential(t *testing.T) {
	t.Parallel()
	ride := publicRideFixture()
	server, _ := registeredServer(t, &fakeDatabase{
		rides:     map[uuid.UUID]app.LoadedRide{ride.ID: ride},
		discovery: []app.LoadedRide{ride},
		locations: []app.LocationRecord{ride.Departure},
	})
	rides := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)

	for _, tc := range []struct {
		name, authorization string
	}{
		{"invalid bearer token", "Bearer invalid"},
		{"wrong scheme", "Basic anything"},
		{"empty header", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := []struct {
				name string
				send func() error
			}{
				{"ListRides", func() error {
					req := connect.NewRequest(&v1.ListRidesRequest{})
					req.Header().Set("Authorization", tc.authorization)
					_, err := rides.ListRides(context.Background(), req)
					return err
				}},
				{"GetRide", func() error {
					req := connect.NewRequest(&v1.GetRideRequest{Id: ride.ID.String()})
					req.Header().Set("Authorization", tc.authorization)
					_, err := rides.GetRide(context.Background(), req)
					return err
				}},
				{"ListLocations", func() error {
					req := connect.NewRequest(&v1.ListLocationsRequest{})
					req.Header().Set("Authorization", tc.authorization)
					_, err := rides.ListLocations(context.Background(), req)
					return err
				}},
			}
			for _, call := range calls {
				if got := connect.CodeOf(call.send()); got != connect.CodeUnauthenticated {
					t.Errorf("%s() with %q: code = %v, want %v", call.name, tc.authorization, got, connect.CodeUnauthenticated)
				}
			}
		})
	}
}

func TestRegisterRejectsMalformedGuestRequests(t *testing.T) {
	t.Parallel()
	server, _ := registeredServer(t, &fakeDatabase{})
	rides := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)
	bad := "bad-id"

	t.Run("GetRide", func(t *testing.T) {
		_, err := rides.GetRide(context.Background(), connect.NewRequest(&v1.GetRideRequest{Id: bad}))
		if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
			t.Errorf("GetRide() code = %v, want %v", got, connect.CodeInvalidArgument)
		}
	})
	t.Run("ListRides", func(t *testing.T) {
		_, err := rides.ListRides(context.Background(), connect.NewRequest(&v1.ListRidesRequest{DepartureLocationId: &bad}))
		if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
			t.Errorf("ListRides() code = %v, want %v", got, connect.CodeInvalidArgument)
		}
	})
}

func TestRegisterServesBothServices(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	server, _ := registeredServer(t, &fakeDatabase{users: map[uuid.UUID]app.UserRecord{id: testUser(id, "caller@rice.edu")}})

	userReq := connect.NewRequest(&v1.GetUserRequest{})
	userReq.Header().Set("X-Test-User", id.String())
	user, err := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL).GetUser(context.Background(), userReq)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user.Msg.User.Id != id.String() {
		t.Errorf("GetUser() id = %q, want %q", user.Msg.User.Id, id)
	}

	rideReq := connect.NewRequest(&v1.ListLocationsRequest{})
	rideReq.Header().Set("X-Test-User", id.String())
	if _, err := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL).ListLocations(context.Background(), rideReq); err != nil {
		t.Errorf("ListLocations() error = %v", err)
	}
}

// TestRegisterAuthenticatesBeforeValidating checks interceptor order: an
// unauthenticated caller is refused without their request being inspected, and
// either rejection is logged exactly once.
func TestRegisterAuthenticatesBeforeValidating(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		authenticated bool
		want          connect.Code
	}{
		{name: "no identity", want: connect.CodeUnauthenticated},
		{name: "identity with an invalid request", authenticated: true, want: connect.CodeInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, logs := registeredServer(t, &fakeDatabase{})
			client := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)
			// Protovalidate rejects the raw 101-character name. The application
			// would trim it to "A" and save successfully, so only the interceptor
			// can produce this rejection.
			req := connect.NewRequest(&v1.UpdateUserRequest{FirstName: strings.Repeat(" ", 100) + "A", LastName: "User"})
			if tc.authenticated {
				req.Header().Set("X-Test-User", uuid.NewString())
			}

			_, err := client.UpdateUser(context.Background(), req)

			if got := connect.CodeOf(err); got != tc.want {
				t.Errorf("UpdateUser() code = %v, want %v", got, tc.want)
			}
			if lines := strings.Count(strings.TrimSpace(logs.String()), "\n") + 1; lines != 1 {
				t.Errorf("logged %d lines, want exactly 1:\n%s", lines, logs)
			}
			if want := `"code":"` + tc.want.String() + `"`; !strings.Contains(logs.String(), want) {
				t.Errorf("log does not record %s:\n%s", want, logs)
			}
		})
	}
}

func TestRegisterLimitsRequestBody(t *testing.T) {
	t.Parallel()
	server, _ := registeredServer(t, &fakeDatabase{})
	client := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)

	_, err := client.CreateRide(context.Background(), connect.NewRequest(&v1.CreateRideRequest{Notes: strings.Repeat("x", 70<<10)}))

	if got := connect.CodeOf(err); got != connect.CodeResourceExhausted {
		t.Errorf("CreateRide() with a 70KB body: code = %v, want %v", got, connect.CodeResourceExhausted)
	}
}

// TestRegisterSanitizesUnknownErrors checks the two halves of the contract: the
// storage detail never reaches the client, and it reaches the log exactly once.
func TestRegisterSanitizesUnknownErrors(t *testing.T) {
	t.Parallel()
	const secret = "postgres://service-role:secret@db.internal/carpool"
	server, logs := registeredServer(t, (&fakeDatabase{}).failing(opFindUser, errors.New(secret)))
	client := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)
	req := connect.NewRequest(&v1.GetUserRequest{})
	req.Header().Set("X-Test-User", uuid.NewString())

	_, err := client.GetUser(context.Background(), req)

	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("GetUser() code = %v, want %v", got, connect.CodeInternal)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("client error leaked the storage detail: %v", err)
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Errorf("client error = %v, want the generic message", err)
	}
	if got := strings.Count(logs.String(), secret); got != 1 {
		t.Errorf("logged the cause %d times, want exactly 1:\n%s", got, logs)
	}
	if !strings.Contains(logs.String(), carpoolv1connect.UserServiceGetUserProcedure) {
		t.Errorf("log does not name the procedure:\n%s", logs)
	}
}
