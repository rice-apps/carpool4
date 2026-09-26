package rpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1/carpoolv1connect"
)

func TestPublicReadAuthenticationBoundary(t *testing.T) {
	strict := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("credential rejected"))
		}
	})
	mux := http.NewServeMux()
	if err := Register(mux, NewServer(app.NewService(unavailableDatabase{})), strict, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(mux)
	defer server.Close()
	rides := carpoolv1connect.NewRideServiceClient(server.Client(), server.URL)
	users := carpoolv1connect.NewUserServiceClient(server.Client(), server.URL)

	if _, err := rides.ListRides(context.Background(), connect.NewRequest(&carpoolv1.ListRidesRequest{})); connect.CodeOf(err) == connect.CodeUnauthenticated {
		t.Fatalf("guest ListRides = %v", err)
	}
	listReq := connect.NewRequest(&carpoolv1.ListRidesRequest{})
	listReq.Header().Set("Authorization", "Bearer invalid")
	if _, err := rides.ListRides(context.Background(), listReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("invalid ListRides credential = %v", err)
	}
	blankReq := connect.NewRequest(&carpoolv1.ListRidesRequest{})
	blankReq.Header().Set("Authorization", "")
	if _, err := rides.ListRides(context.Background(), blankReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("blank ListRides credential = %v", err)
	}
	if _, err := rides.GetRide(context.Background(), connect.NewRequest(&carpoolv1.GetRideRequest{Id: uuid.NewString()})); connect.CodeOf(err) == connect.CodeUnauthenticated {
		t.Fatalf("guest GetRide = %v", err)
	}
	getReq := connect.NewRequest(&carpoolv1.GetRideRequest{Id: uuid.NewString()})
	getReq.Header().Set("Authorization", "Bearer invalid")
	if _, err := rides.GetRide(context.Background(), getReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("invalid GetRide credential = %v", err)
	}
	if _, err := rides.ListLocations(context.Background(), connect.NewRequest(&carpoolv1.ListLocationsRequest{})); connect.CodeOf(err) == connect.CodeUnauthenticated {
		t.Fatalf("guest ListLocations = %v", err)
	}
	locationReq := connect.NewRequest(&carpoolv1.ListLocationsRequest{})
	locationReq.Header().Set("Authorization", "Bearer invalid")
	if _, err := rides.ListLocations(context.Background(), locationReq); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("invalid ListLocations credential = %v", err)
	}
	if _, err := rides.ListMyRides(context.Background(), connect.NewRequest(&carpoolv1.ListMyRidesRequest{})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("protected ListMyRides = %v", err)
	}
	if _, err := users.GetUser(context.Background(), connect.NewRequest(&carpoolv1.GetUserRequest{})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("protected GetUser = %v", err)
	}
}

type unavailableDatabase struct{}

func (unavailableDatabase) BeginTx(context.Context) (app.Transaction, error) {
	return nil, errors.New("database unavailable in authentication test")
}
