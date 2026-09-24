package rpc

import (
	"context"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

// newValidationInterceptor returns an interceptor for the protobuf validation rules.
// Invalid requests receive a Connect InvalidArgument error; setup may also fail.
func newValidationInterceptor() (connect.UnaryInterceptorFunc, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Check declared request rules before the operation reaches its handler.
			if msg, ok := req.Any().(proto.Message); ok {
				if err := validator.Validate(msg); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, err)
				}
			}
			return next(ctx, req)
		}
	}, nil
}
