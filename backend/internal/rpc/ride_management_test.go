package rpc

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rice-apps/carpool4/backend/internal/app"
	"github.com/rice-apps/carpool4/backend/internal/auth"
	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

func TestCreateRideRejectsMalformedInput(t *testing.T) {
	departureID := uuid.NewString()
	arrivalID := uuid.NewString()
	departureTime := timestamppb.New(time.Now().Add(time.Hour))
	ctx := auth.WithUser(context.Background(), auth.User{ID: uuid.New(), Email: "owner@rice.edu"})
	server := NewServer(app.NewService(unavailableDatabase{}))

	for _, test := range []struct {
		name          string
		departureID   string
		arrivalID     string
		departureTime *timestamppb.Timestamp
	}{
		{"invalid departure ID", "invalid", arrivalID, departureTime},
		{"invalid arrival ID", departureID, "invalid", departureTime},
		{"missing timestamp", departureID, arrivalID, nil},
		{"timestamp out of range", departureID, arrivalID, &timestamppb.Timestamp{Seconds: 253402300800}},
		{"invalid nanoseconds", departureID, arrivalID, &timestamppb.Timestamp{Nanos: -1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := server.CreateRide(ctx, connect.NewRequest(&carpoolv1.CreateRideRequest{
				DepartureLocationId: test.departureID,
				ArrivalLocationId:   test.arrivalID,
				DepartureTime:       test.departureTime,
				Capacity:            3,
			}))
			if response != nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("CreateRide = %v, %v; want invalid argument", response, err)
			}
		})
	}
}
