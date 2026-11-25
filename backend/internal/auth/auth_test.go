package auth_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/rice-apps/carpool4/backend/internal/auth"
)

const (
	validSubject = "4f091c48-d3ad-4b09-bce2-a7549a760a49"
	validEmail   = "alex@rice.edu"
)

// testJWKSServer serves a signing key and records JWKS request paths under a
// mutex so tests can inspect startup and refresh behavior.
//
// Fields:
//   - URL: gives the fixture's Supabase base URL.
//   - key: signs tokens accepted by the fixture's JWKS.
//   - server: serves the JWKS response during tests.
//   - mu: protects the recorded request paths.
//   - requestURI: records paths fetched from the fixture.
type testJWKSServer struct {
	URL        string
	key        *ecdsa.PrivateKey
	server     *httptest.Server
	mu         sync.Mutex
	requestURI []string
}

// newTestJWKSServer serves a single ES256 JWK, matching Supabase's asymmetric
// signing-key format, and records the path used to fetch it.
func newTestJWKSServer(t *testing.T) *testJWKSServer {
	t.Helper()

	// Create the signing key and public JWKS served by this fixture.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	jwk, err := jwkset.NewJWKFromKey(key.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{KID: "test-key", ALG: jwkset.AlgES256},
	})
	if err != nil {
		t.Fatalf("build JWK: %v", err)
	}

	body, err := json.Marshal(jwkset.JWKSMarshal{Keys: []jwkset.JWKMarshal{jwk.Marshal()}})
	if err != nil {
		t.Fatalf("marshal JWKS: %v", err)
	}

	// Record each fetch so tests can inspect startup and refresh behavior.
	fixture := &testJWKSServer{key: key}
	fixture.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture.mu.Lock()
		fixture.requestURI = append(fixture.requestURI, r.URL.RequestURI())
		fixture.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	fixture.URL = fixture.server.URL
	t.Cleanup(fixture.server.Close)

	return fixture
}

// requestedURIs returns a copy of the paths fetched from the JWKS fixture.
func (s *testJWKSServer) requestedURIs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requestURI...)
}

// validClaims builds an unexpired authenticated claim set whose issuer is
// derived from the fixture's Supabase base URL.
func validClaims(serverURL string) jwt.MapClaims {
	return jwt.MapClaims{
		"sub":   validSubject,
		"email": validEmail,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iss":   serverURL + "/auth/v1",
		"aud":   "authenticated",
	}
}

// signToken signs claims with key and the fixture's default key ID.
func signToken(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims) string {
	return signTokenWithKID(t, key, claims, "test-key")
}

// signTokenWithKID signs claims with an ES256 private key and selected key ID.
// It fails the calling test if signing fails.
func signTokenWithKID(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims, kid string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// newInterceptor constructs an interceptor for a test Supabase URL and ties
// its key-refresh context to test cleanup.
func newInterceptor(t *testing.T, supabaseURL string) connect.UnaryInterceptorFunc {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	interceptor, err := auth.NewInterceptor(ctx, supabaseURL)
	if err != nil {
		t.Fatalf("NewInterceptor: %v", err)
	}
	return interceptor
}

// callWithAuthorization invokes a Connect interceptor with the supplied
// Authorization header. It returns the user seen by the handler, whether one
// was present, and the interceptor error.
func callWithAuthorization(
	t *testing.T,
	interceptor connect.UnaryInterceptorFunc,
	authorization string,
) (auth.User, bool, error) {
	return callWithAuthorizationContext(t, context.Background(), interceptor, authorization)
}

// callWithAuthorizationContext runs the authentication fixture under ctx so
// tests can check cancellation, returning the observed user and call error.
func callWithAuthorizationContext(
	t *testing.T,
	ctx context.Context,
	interceptor connect.UnaryInterceptorFunc,
	authorization string,
) (auth.User, bool, error) {
	t.Helper()

	// Capture the identity that reaches the downstream handler.
	var got auth.User
	var ok bool
	handler := interceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		got, ok = auth.GetSignedInUser(ctx)
		return connect.NewResponse(&struct{}{}), nil
	})

	req := connect.NewRequest(&struct{}{})
	if authorization != "" {
		req.Header().Set("Authorization", authorization)
	}

	_, err := handler(ctx, req)
	return got, ok, err
}

// callWithToken calls the fixture with token formatted as a Bearer credential.
func callWithToken(
	t *testing.T,
	interceptor connect.UnaryInterceptorFunc,
	token string,
) (auth.User, bool, error) {
	t.Helper()
	return callWithAuthorization(t, interceptor, "Bearer "+token)
}

// requireCode reports a test error unless err carries the expected Connect code.
func requireCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	if got := connect.CodeOf(err); got != want {
		t.Errorf("code = %v, want %v (error: %v)", got, want, err)
	}
}

// TestInterceptor_ValidSupabaseToken checks that verified claims reach the
// downstream handler as a typed caller identity.
func TestInterceptor_ValidSupabaseToken(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL))

	user, ok, err := callWithToken(t, interceptor, token)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !ok {
		t.Fatal("expected auth.User in context, found none")
	}
	if got, want := user.ID, uuid.MustParse(validSubject); got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := user.Email, validEmail; got != want {
		t.Errorf("Email = %q, want %q", got, want)
	}
}

// TestInterceptor_BearerSchemeIsCaseInsensitive checks accepted scheme casing.
func TestInterceptor_BearerSchemeIsCaseInsensitive(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL))

	for _, scheme := range []string{"Bearer", "bearer", "BEARER"} {
		t.Run(scheme, func(t *testing.T) {
			user, ok, err := callWithAuthorization(t, interceptor, scheme+" "+token)
			if err != nil {
				t.Fatalf("handler: %v", err)
			}
			if !ok || user.ID != uuid.MustParse(validSubject) {
				t.Fatalf("user = %+v, present = %v", user, ok)
			}
		})
	}
}

// TestInterceptor_RequiresBearerAuthorization checks malformed or absent
// credentials are rejected before a handler sees a user.
func TestInterceptor_RequiresBearerAuthorization(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL))

	for _, tc := range []struct {
		name          string
		authorization string
	}{
		{"absent", ""},
		{"raw token", token},
		{"scheme only", "Bearer"},
		{"scheme and gap", "Bearer "},
		{"basic scheme", "Basic " + token},
		{"double space", "Bearer  " + token},
		{"trailing space", "Bearer " + token + " "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, err := callWithAuthorization(t, interceptor, tc.authorization)
			requireCode(t, err, connect.CodeUnauthenticated)
			if ok {
				t.Error("handler received an authenticated user")
			}
		})
	}
}

// TestInterceptor_RejectsExpiredToken checks expired signed claims fail auth.
func TestInterceptor_RejectsExpiredToken(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	claims := validClaims(server.URL)
	claims["exp"] = time.Now().Add(-time.Hour).Unix()

	_, _, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
	requireCode(t, err, connect.CodeUnauthenticated)
}

// TestInterceptor_RequiresExpiration checks tokens without exp fail auth.
func TestInterceptor_RequiresExpiration(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	claims := validClaims(server.URL)
	delete(claims, "exp")

	_, _, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
	requireCode(t, err, connect.CodeUnauthenticated)
}

// TestInterceptor_RequiresExpectedIssuer checks the configured Supabase issuer.
func TestInterceptor_RequiresExpectedIssuer(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)

	for name, issuer := range map[string]any{
		"missing": nil,
		"wrong":   "https://another-project.supabase.co/auth/v1",
	} {
		t.Run(name, func(t *testing.T) {
			claims := validClaims(server.URL)
			if issuer == nil {
				delete(claims, "iss")
			} else {
				claims["iss"] = issuer
			}
			_, _, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
			requireCode(t, err, connect.CodeUnauthenticated)
		})
	}
}

// TestInterceptor_RequiresAuthenticatedAudience checks that anon and missing
// audiences cannot enter the authenticated API.
func TestInterceptor_RequiresAuthenticatedAudience(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)

	for name, audience := range map[string]any{
		"missing": nil,
		"wrong":   "anon",
	} {
		t.Run(name, func(t *testing.T) {
			claims := validClaims(server.URL)
			if audience == nil {
				delete(claims, "aud")
			} else {
				claims["aud"] = audience
			}
			_, _, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
			requireCode(t, err, connect.CodeUnauthenticated)
		})
	}
}

// TestInterceptor_RequiresNonNilUUIDSubject checks subject parsing and the
// nil UUID rejection before identity is placed in context.
func TestInterceptor_RequiresNonNilUUIDSubject(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)

	for _, subject := range []string{"user-123", "00000000-0000-0000-0000-000000000000"} {
		t.Run(subject, func(t *testing.T) {
			claims := validClaims(server.URL)
			claims["sub"] = subject
			_, ok, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
			requireCode(t, err, connect.CodeUnauthenticated)
			if ok {
				t.Error("handler received an authenticated user")
			}
		})
	}
}

// TestInterceptor_RejectsWrongSigningKey checks signatures against the JWKS.
func TestInterceptor_RejectsWrongSigningKey(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	forgedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate forged key: %v", err)
	}

	_, _, err = callWithToken(t, interceptor, signToken(t, forgedKey, validClaims(server.URL)))
	requireCode(t, err, connect.CodeUnauthenticated)
}

// TestInterceptor_DoesNotApplyEmailDomainPolicy checks that account admission
// stays with Supabase rather than the Go JWT verifier.
func TestInterceptor_DoesNotApplyEmailDomainPolicy(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	claims := validClaims(server.URL)
	claims["email"] = "alex@gmail.com"

	user, ok, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !ok {
		t.Fatal("expected auth.User in context, found none")
	}
	if got, want := user.Email, "alex@gmail.com"; got != want {
		t.Errorf("Email = %q, want %q", got, want)
	}
}

// TestInterceptor_RequiresEmailClaim checks missing caller email is rejected.
func TestInterceptor_RequiresEmailClaim(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	claims := validClaims(server.URL)
	delete(claims, "email")

	_, ok, err := callWithToken(t, interceptor, signToken(t, server.key, claims))
	requireCode(t, err, connect.CodeUnauthenticated)
	if ok {
		t.Error("handler received an authenticated user")
	}
}

// TestNewInterceptor_DerivesJWKSURLFromSupabaseURL checks the initial fetch
// path and trailing-slash normalization.
func TestNewInterceptor_DerivesJWKSURLFromSupabaseURL(t *testing.T) {
	server := newTestJWKSServer(t)
	_ = newInterceptor(t, server.URL+"/")

	requests := server.requestedURIs()
	if len(requests) != 1 {
		t.Fatalf("JWKS requests = %v, want exactly one startup request", requests)
	}
	if got, want := requests[0], "/auth/v1/.well-known/jwks.json"; got != want {
		t.Errorf("JWKS request URI = %q, want %q", got, want)
	}
}

// TestNewInterceptor_ReportsInitialJWKSFailure checks startup fails when
// signing keys cannot be fetched.
func TestNewInterceptor_ReportsInitialJWKSFailure(t *testing.T) {
	server := newTestJWKSServer(t)
	server.server.Close()

	if _, err := auth.NewInterceptor(context.Background(), server.URL); err == nil {
		t.Error("expected an error constructing the interceptor with an unreachable JWKS URL")
	}
}

// TestInterceptor_JWKSRefreshHonorsRequestContext checks a canceled request with an
// unknown key ID does not trigger another JWKS fetch. An unknown kid is reachable
// before authentication, so a per-request refresh would turn this service into an
// amplifier against the Supabase JWKS endpoint.
func TestInterceptor_JWKSRefreshHonorsRequestContext(t *testing.T) {
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signTokenWithKID(t, server.key, validClaims(server.URL), "unknown-key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := callWithAuthorizationContext(t, ctx, interceptor, "Bearer "+token)
	requireCode(t, err, connect.CodeUnauthenticated)
	if got := server.requestedURIs(); len(got) != 1 {
		t.Errorf("JWKS requests = %v, want no refresh after request cancellation", got)
	}
}
