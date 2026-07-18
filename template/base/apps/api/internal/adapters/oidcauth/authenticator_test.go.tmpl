package oidcauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestOIDCEndpointTransportPolicy(t *testing.T) {
	for _, test := range []struct {
		name, endpoint string
		insecure       bool
		valid          bool
	}{{"secure", "https://id.example/token", false, true}, {"local HTTP", "http://localhost/token", true, true}, {"plaintext production", "http://id.example/token", false, false}, {"credentials", "https://user:pass@id.example/token", false, false}, {"fragment", "https://id.example/token#secret", false, false}, {"non HTTP", "ftp://id.example/token", true, false}} {
		t.Run(test.name, func(t *testing.T) {
			if valid := validateEndpoint(test.endpoint, test.insecure) == nil; valid != test.valid {
				t.Fatalf("valid=%v, want %v", valid, test.valid)
			}
		})
	}
}

func TestOIDCBackchannelPreservesPublicIssuer(t *testing.T) {
	const publicIssuer = "http://localhost:5556/dex"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/dex/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": publicIssuer, "jwks_uri": publicIssuer + "/keys",
			"authorization_endpoint": publicIssuer + "/auth", "token_endpoint": publicIssuer + "/token",
		})
	}))
	defer server.Close()
	verifier, err := New(context.Background(), publicIssuer, server.URL+"/dex", []string{"web"}, true)
	if err != nil || requests != 1 {
		t.Fatalf("backchannel discovery failed without preserving issuer: requests=%d err=%v", requests, err)
	}
	authorizationURL, tokenURL := verifier.Endpoints()
	if authorizationURL != publicIssuer+"/auth" {
		t.Fatalf("browser endpoint was rewritten: %s", authorizationURL)
	}
	if tokenURL != server.URL+"/dex/token" {
		t.Fatalf("server-side token endpoint was not rewritten to the backchannel: %s", tokenURL)
	}
}

func TestOIDCBackchannelRejectsTokenEndpointOnAnotherOrigin(t *testing.T) {
	const publicIssuer = "http://localhost:5556/dex"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": publicIssuer, "jwks_uri": publicIssuer + "/keys",
			"authorization_endpoint": publicIssuer + "/auth", "token_endpoint": "http://tokens.example.invalid/token",
		})
	}))
	defer server.Close()
	if _, err := New(context.Background(), publicIssuer, server.URL+"/dex", []string{"web"}, true); err == nil {
		t.Fatal("expected a cross-origin discovered token endpoint to fail closed")
	}
}

func TestVerifierValidatesEveryIdentityTokenBoundary(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                issuer,
				"jwks_uri":                              issuer + "/keys",
				"authorization_endpoint":                issuer + "/custom/authorize",
				"token_endpoint":                        issuer + "/custom/token",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test-key", Algorithm: "RS256", Use: "sig"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL

	authenticator, err := New(context.Background(), issuer, "", []string{"web-client", "mobile-client"}, true)
	if err != nil {
		t.Fatal(err)
	}
	authorizationURL, tokenURL := authenticator.Endpoints()
	if authorizationURL != issuer+"/custom/authorize" || tokenURL != issuer+"/custom/token" {
		t.Fatalf("OIDC endpoints were not sourced from discovery: %q %q", authorizationURL, tokenURL)
	}
	now := time.Now()
	tests := []struct {
		name      string
		claims    jwt.Claims
		key       *rsa.PrivateKey
		azp       string
		wantError bool
	}{
		{name: "web audience", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"web-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: key},
		{name: "mobile audience", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"mobile-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: key},
		{name: "multiple audiences with authorized party", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"web-client", "mobile-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, azp: "web-client", key: key},
		{name: "multiple audiences missing authorized party", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"web-client", "mobile-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: key, wantError: true},
		{name: "unauthorized authorized party", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"web-client", "other-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, azp: "other-client", key: key, wantError: true},
		{name: "wrong issuer", claims: jwt.Claims{Issuer: "https://evil.example", Subject: "user-1", Audience: jwt.Audience{"web-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: key, wantError: true},
		{name: "wrong audience", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"other-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: key, wantError: true},
		{name: "expired", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"web-client"}, Expiry: jwt.NewNumericDate(now.Add(-time.Minute))}, key: key, wantError: true},
		{name: "empty subject", claims: jwt.Claims{Issuer: issuer, Audience: jwt.Audience{"web-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: key, wantError: true},
		{name: "invalid signature", claims: jwt.Claims{Issuer: issuer, Subject: "user-1", Audience: jwt.Audience{"web-client"}, Expiry: jwt.NewNumericDate(now.Add(time.Minute))}, key: mustRSAKey(t), wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := signToken(t, test.key, test.claims, test.azp)
			claims, err := authenticator.Verify(context.Background(), token)
			if test.wantError && err == nil {
				t.Fatal("expected authentication failure")
			}
			if !test.wantError && (err != nil || claims.Subject != test.claims.Subject) {
				t.Fatalf("valid token rejected: claims=%+v err=%v", claims, err)
			}
		})
	}
	for _, raw := range []string{"", "Basic abc", "Bearer abc", "token with spaces"} {
		if _, err := authenticator.Verify(context.Background(), raw); err == nil {
			t.Errorf("accepted malformed identity token %q", raw)
		}
	}
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.Claims, azp string) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: key, KeyID: "test-key"}}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		t.Fatal(err)
	}
	payload := struct {
		jwt.Claims
		Email           string `json:"email"`
		Name            string `json:"name"`
		AuthorizedParty string `json:"azp,omitempty"`
	}{Claims: claims, Email: "person@example.com", Name: "Person", AuthorizedParty: azp}
	serialized, err := jwt.Signed(signer).Claims(payload).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(serialized)
}

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}
