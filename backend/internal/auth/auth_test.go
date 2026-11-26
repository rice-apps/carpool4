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

// signToken signs claims as Supabase does, under the given key ID.
func signToken(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims, kid string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// newInterceptor constructs an interceptor for a test Supabase URL and ties its
// key-refresh context to test cleanup.
func newInterceptor(t *testing.T, supabaseURL string) connect.UnaryInterceptorFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	interceptor, err := auth.NewInterceptor(ctx, supabaseURL)
	if err != nil {
		t.Fatalf("NewInterceptor() error = %v", err)
	}
	return interceptor
}

// call runs the interceptor under ctx with the given Authorization header and
// reports the identity the downstream handler saw.
func call(t *testing.T, ctx context.Context, interceptor connect.UnaryInterceptorFunc, authorization string) (auth.User, bool, error) {
	t.Helper()
	var got auth.User
	var signedIn bool
	handler := interceptor(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		got, signedIn = auth.GetSignedInUser(ctx)
		return connect.NewResponse(&struct{}{}), nil
	})

	req := connect.NewRequest(&struct{}{})
	if authorization != "" {
		req.Header().Set("Authorization", authorization)
	}
	_, err := handler(ctx, req)
	return got, signedIn, err
}

// requireRejected fails t unless the call was refused without an identity
// reaching the handler.
func requireRejected(t *testing.T, signedIn bool, err error) {
	t.Helper()
	if got := connect.CodeOf(err); got != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want %v (error: %v)", got, connect.CodeUnauthenticated, err)
	}
	if signedIn {
		t.Error("a rejected token still placed an identity in the handler context")
	}
}

func TestInterceptorAcceptsAValidSupabaseToken(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL), "test-key")

	user, signedIn, err := call(t, t.Context(), interceptor, "Bearer "+token)
	if err != nil {
		t.Fatalf("call with a valid token error = %v", err)
	}

	if !signedIn {
		t.Fatal("no identity reached the handler")
	}
	if user.ID != uuid.MustParse(validSubject) {
		t.Errorf("ID = %v, want %v", user.ID, validSubject)
	}
	if user.Email != validEmail {
		t.Errorf("Email = %q, want %q", user.Email, validEmail)
	}
}

// TestInterceptorRejectsBadRegisteredClaims checks the claims a Supabase token
// must carry. Each case starts from a valid set and breaks exactly one thing.
func TestInterceptorRejectsBadRegisteredClaims(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(jwt.MapClaims)
	}{
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
		{"no expiry", func(c jwt.MapClaims) { delete(c, "exp") }},
		{"no issuer", func(c jwt.MapClaims) { delete(c, "iss") }},
		{"another project's issuer", func(c jwt.MapClaims) { c["iss"] = "https://another.supabase.co/auth/v1" }},
		{"no audience", func(c jwt.MapClaims) { delete(c, "aud") }},
		{"anon audience", func(c jwt.MapClaims) { c["aud"] = "anon" }},
		{"subject is not a UUID", func(c jwt.MapClaims) { c["sub"] = "user-123" }},
		{"subject is the nil UUID", func(c jwt.MapClaims) { c["sub"] = uuid.Nil.String() }},
		{"no email", func(c jwt.MapClaims) { delete(c, "email") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := newTestJWKSServer(t)
			interceptor := newInterceptor(t, server.URL)
			claims := validClaims(server.URL)
			tc.mutate(claims)

			_, signedIn, err := call(t, t.Context(), interceptor, "Bearer "+signToken(t, server.key, claims, "test-key"))

			requireRejected(t, signedIn, err)
		})
	}
}

// TestInterceptorRejectsForgedSignatures covers the signature attacks a public
// endpoint invites: an unsigned token, a token signed with HMAC using the
// public verification key as the shared secret, and one signed with a key the
// JWKS does not publish.
func TestInterceptorRejectsForgedSignatures(t *testing.T) {
	t.Parallel()

	t.Run("unsigned token", func(t *testing.T) {
		t.Parallel()
		server := newTestJWKSServer(t)
		interceptor := newInterceptor(t, server.URL)
		token := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims(server.URL))
		token.Header["kid"] = "test-key"
		forged, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("sign unsigned token: %v", err)
		}

		_, signedIn, err := call(t, t.Context(), interceptor, "Bearer "+forged)

		requireRejected(t, signedIn, err)
	})

	t.Run("HMAC signed with the public key", func(t *testing.T) {
		t.Parallel()
		server := newTestJWKSServer(t)
		interceptor := newInterceptor(t, server.URL)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims(server.URL))
		token.Header["kid"] = "test-key"
		public := elliptic.Marshal(server.key.Curve, server.key.X, server.key.Y)
		forged, err := token.SignedString(public)
		if err != nil {
			t.Fatalf("sign HMAC token: %v", err)
		}

		_, signedIn, err := call(t, t.Context(), interceptor, "Bearer "+forged)

		requireRejected(t, signedIn, err)
	})

	t.Run("signed with an unpublished key", func(t *testing.T) {
		t.Parallel()
		server := newTestJWKSServer(t)
		interceptor := newInterceptor(t, server.URL)
		forgedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generate forged key: %v", err)
		}

		_, signedIn, err := call(t, t.Context(), interceptor,
			"Bearer "+signToken(t, forgedKey, validClaims(server.URL), "test-key"))

		requireRejected(t, signedIn, err)
	})
}

func TestInterceptorRequiresABearerCredential(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL), "test-key")

	for _, tc := range []struct {
		name, authorization string
	}{
		{"absent", ""},
		{"bare token with no scheme", token},
		{"scheme with no token", "Bearer"},
		{"scheme and a space only", "Bearer "},
		{"wrong scheme", "Basic " + token},
		{"double space", "Bearer  " + token},
		{"trailing space", "Bearer " + token + " "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, signedIn, err := call(t, t.Context(), interceptor, tc.authorization)

			requireRejected(t, signedIn, err)
		})
	}
}

// TestInterceptorAcceptsAnyBearerCase checks the scheme match is
// case-insensitive, as RFC 7235 requires.
func TestInterceptorAcceptsAnyBearerCase(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL), "test-key")

	for _, scheme := range []string{"Bearer", "bearer", "BEARER"} {
		t.Run(scheme, func(t *testing.T) {
			t.Parallel()
			_, signedIn, err := call(t, t.Context(), interceptor, scheme+" "+token)
			if err != nil {
				t.Fatalf("call with %q scheme error = %v", scheme, err)
			}
			if !signedIn {
				t.Errorf("scheme %q did not authenticate", scheme)
			}
		})
	}
}

// TestInterceptorLeavesDomainPolicyToSupabase checks this verifier accepts any
// email: restricting admission to Rice is a signup-time policy, not a property
// of a token that has already been issued.
func TestInterceptorLeavesDomainPolicyToSupabase(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	claims := validClaims(server.URL)
	claims["email"] = "alex@gmail.com"

	user, signedIn, err := call(t, t.Context(), interceptor, "Bearer "+signToken(t, server.key, claims, "test-key"))
	if err != nil {
		t.Fatalf("call error = %v", err)
	}

	if !signedIn {
		t.Fatal("no identity reached the handler")
	}
	if user.Email != "alex@gmail.com" {
		t.Errorf("Email = %q, want it passed through unchanged", user.Email)
	}
}

func TestNewInterceptorDerivesTheJWKSURL(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)

	_ = newInterceptor(t, server.URL+"/")

	requests := server.requestedURIs()
	if len(requests) != 1 {
		t.Fatalf("JWKS requests = %v, want exactly one at startup", requests)
	}
	if want := "/auth/v1/.well-known/jwks.json"; requests[0] != want {
		t.Errorf("JWKS request URI = %q, want %q", requests[0], want)
	}
}

// TestNewInterceptorReportsInitialJWKSFailure checks the server refuses to boot
// without signing keys, rather than starting and rejecting every request.
func TestNewInterceptorReportsInitialJWKSFailure(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)
	server.server.Close()

	if _, err := auth.NewInterceptor(context.Background(), server.URL); err == nil {
		t.Error("NewInterceptor() with an unreachable JWKS = nil, want an error")
	}
}

// TestInterceptorDoesNotRefreshKeysForACancelledRequest checks an unknown key ID
// cannot drive repeated JWKS fetches. An unknown kid is reachable before
// authentication, so a per-request refresh would make this service an amplifier
// against the Supabase JWKS endpoint.
func TestInterceptorDoesNotRefreshKeysForACancelledRequest(t *testing.T) {
	t.Parallel()
	server := newTestJWKSServer(t)
	interceptor := newInterceptor(t, server.URL)
	token := signToken(t, server.key, validClaims(server.URL), "unknown-key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, signedIn, err := call(t, ctx, interceptor, "Bearer "+token)

	requireRejected(t, signedIn, err)
	if got := server.requestedURIs(); len(got) != 1 {
		t.Errorf("JWKS requests = %v, want only the startup fetch", got)
	}
}
