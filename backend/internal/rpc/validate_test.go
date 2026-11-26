package rpc

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/rice-apps/carpool4/backend/internal/gen/carpool/v1"
)

// TestValidationInterceptorEnforcesProfileContract checks declarative profile rules before handler execution.
func TestValidationInterceptorEnforcesProfileContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *carpoolv1.UpdateUserRequest
		valid   bool
	}{
		{name: "valid profile", request: &carpoolv1.UpdateUserRequest{FirstName: "Ada", LastName: "Lovelace"}, valid: true},
		{name: "empty first name", request: &carpoolv1.UpdateUserRequest{LastName: "Lovelace"}},
		{name: "whitespace last name", request: &carpoolv1.UpdateUserRequest{FirstName: "Ada", LastName: "   "}},
		{name: "oversized phone", request: &carpoolv1.UpdateUserRequest{FirstName: "Ada", LastName: "Lovelace", Phone: strings.Repeat("1", 33)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateRequest(t, tc.request)
			if tc.valid && err != nil {
				t.Fatalf("valid request rejected: %v", err)
			}
			if !tc.valid && connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("invalid request code = %v, want invalid_argument (error: %v)", connect.CodeOf(err), err)
			}
		})
	}
}

// validateRequest runs request through the validation interceptor and returns its rejection, if any.
func validateRequest(t *testing.T, request *carpoolv1.UpdateUserRequest) error {
	t.Helper()
	interceptor, err := newValidationInterceptor()
	if err != nil {
		t.Fatalf("newValidationInterceptor: %v", err)
	}
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})
	_, err = handler(context.Background(), connect.NewRequest(request))
	return err
}
