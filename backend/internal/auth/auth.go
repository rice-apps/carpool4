package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	authenticatedAudience = "authenticated"
	jwksFetchTimeout      = 10 * time.Second
)

// User is a verified caller. ID is the Supabase subject UUID, and Email comes
// from the verified token.
//
// Fields:
//   - ID: identifies the caller by Supabase subject.
//   - Email: carries the caller's verified token email.
type User struct {
	ID    uuid.UUID
	Email string
}

// contextKey keeps the verified identity separate from other context values.
type contextKey struct{}

// WithUser returns a child context carrying the already verified caller u.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// UserFromContext reads the verified caller from ctx. The bool is false when
// no authentication interceptor stored a User.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}

// claims decodes the email and standard JWT fields needed for admission to
// the Go API; signup policy belongs to Supabase Auth.
//
// Fields:
//   - Email: supplies the caller email from the signed token.
//   - RegisteredClaims: carries the standard subject, issuer, audience, and expiration claims.
type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// NewInterceptor creates a Connect authentication interceptor.
//
// Inputs:
//   - ctx (context.Context): controls signing-key fetches and refreshes.
//   - supabaseURL (string): the Supabase project's base URL.
//
// It returns the interceptor or an error if the initial key fetch fails. The
// interceptor rejects invalid JWTs with Unauthenticated and passes a verified
// User in context.
func NewInterceptor(ctx context.Context, supabaseURL string) (connect.UnaryInterceptorFunc, error) {
	// Find the signing keys published by this Supabase project.
	issuer := strings.TrimRight(supabaseURL, "/") + "/auth/v1"
	jwksURL := issuer + "/.well-known/jwks.json"
	returnErrorOnInitialFetchFailure := false
	kf, err := keyfunc.NewDefaultOverrideCtx(ctx, []string{jwksURL}, keyfunc.Override{
		HTTPTimeout:               jwksFetchTimeout,
		NoErrorReturnFirstHTTPReq: &returnErrorOnInitialFetchFailure,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize JWKS: %w", err)
	}

	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Read the bearer credential from the incoming request.
			tokenStr, ok := bearerToken(req.Header().Get("Authorization"))
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing bearer token"))
			}

			// Check the signature, expiration, issuer, and audience together.
			var c claims
			token, err := jwt.ParseWithClaims(
				tokenStr,
				&c,
				kf.KeyfuncCtx(ctx),
				jwt.WithExpirationRequired(),
				jwt.WithIssuer(issuer),
				jwt.WithAudience(authenticatedAudience),
			)
			if err != nil || !token.Valid {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}

			// A signed token still needs a usable user ID and email.
			id, err := uuid.Parse(c.Subject)
			if err != nil || id == uuid.Nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}

			if c.Email == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}

			// Only the verified claims establish caller identity for later RPC work.
			ctx = WithUser(ctx, User{ID: id, Email: c.Email})
			return next(ctx, req)
		}
	}

	return interceptor, nil
}

// bearerToken returns the token in authorization and whether it is a valid
// Bearer credential. The scheme is case insensitive; empty tokens and tokens
// containing whitespace are rejected.
func bearerToken(authorization string) (string, bool) {
	// Split only at the separator between the scheme and token.
	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", false
	}
	return token, true
}
