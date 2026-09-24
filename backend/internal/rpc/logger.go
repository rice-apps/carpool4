package rpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// newLoggingInterceptor returns a unary interceptor that logs each request's
// procedure, duration, and outcome through logger, including private error causes.
func newLoggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			duration := time.Since(start)
			if err != nil {
				// Log the original cause while the response keeps its client-safe message.
				logger.Error("request failed",
					slog.String("procedure", req.Spec().Procedure),
					slog.String("code", connect.CodeOf(err).String()),
					slog.Duration("duration", duration),
					slog.String("error", loggedCause(err).Error()),
				)
			} else {
				logger.Info("request handled",
					slog.String("procedure", req.Spec().Procedure),
					slog.Duration("duration", duration),
				)
			}
			return resp, err
		}
	}
}

// loggedCause returns the original cause of a sanitized RPC error, or err unchanged.
func loggedCause(err error) error {
	var safeErr *safeRPCError
	if errors.As(err, &safeErr) {
		return safeErr.cause
	}
	return err
}
