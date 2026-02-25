// Package auth demonstrates httpx authentication helpers:
// - OAuth 1.0a signing
// - OAuth 2.0 Bearer token (static + custom token source)
// - HMAC request signing
// - Idempotency Key injection
// - Basic Auth
// - Bearer token via request builder
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/n0l3r/httpx"
	httpxauth "github.com/n0l3r/httpx/auth"
)

// Run executes all auth examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  AUTH HELPERS EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleOAuth1()
	exampleOAuth2Static()
	exampleOAuth2CustomSource()
	exampleHMACSigning()
	exampleIdempotencyKey()
	exampleBasicAuth()
	exampleBearerTokenBuilder()
}

// [1] OAuth 1.0a signing.
func exampleOAuth1() {
	fmt.Println("\n[1] OAuth 1.0a — HMAC-SHA256 signed requests")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &httpxauth.OAuth1Transport{
		Config: httpxauth.OAuth1Config{
			ConsumerKey:    "my-consumer-key",
			ConsumerSecret: "my-consumer-secret",
			Token:          "my-access-token",
			TokenSecret:    "my-token-secret",
		},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))
	resp, err := c.Get(context.Background(), "/api/resource")
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ status=%d\n", resp.StatusCode())
	fmt.Printf("    Authorization: %s...\n", truncate(gotAuth, 80))
	fmt.Printf("    Contains oauth_signature: %v\n", strings.Contains(gotAuth, "oauth_signature"))
}

// [2] OAuth 2.0 — static token source.
func exampleOAuth2Static() {
	fmt.Println("\n[2] OAuth 2.0 — static Bearer token")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &httpxauth.OAuth2Transport{
		Source: &httpxauth.StaticTokenSource{AccessToken: "eyJhbGciOiJSUzI1NiJ9.my-token"},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))
	c.Get(context.Background(), "/api/me")
	fmt.Printf("  ✓ Authorization: %s\n", gotAuth)
}

// [3] OAuth 2.0 — custom token source (e.g. auto-refresh).
func exampleOAuth2CustomSource() {
	fmt.Println("\n[3] OAuth 2.0 — custom token source (simulated refresh)")

	callCount := 0
	var gotTokens []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTokens = append(gotTokens, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Custom token source that rotates tokens.
	tokens := []string{"token-v1", "token-v2", "token-v3"}
	source := &rotatingTokenSource{tokens: tokens, idx: &callCount}

	transport := &httpxauth.OAuth2Transport{Source: source}
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))

	for range 3 {
		c.Get(context.Background(), "/api/data")
	}

	fmt.Printf("  ✓ Each request used a different token:\n")
	for _, t := range gotTokens {
		fmt.Printf("    %s\n", t)
	}
}

// [4] HMAC request signing.
func exampleHMACSigning() {
	fmt.Println("\n[4] HMAC-SHA256 request signing")

	secret := []byte("super-secret-key")
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Signature")
		// Verify signature format: keyId=...,ts=...,sig=...
		if strings.Contains(gotSig, "keyId=") && strings.Contains(gotSig, "sig=") {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	transport := &httpxauth.HMACTransport{
		Config: httpxauth.HMACConfig{
			KeyID:  "key-2024",
			Secret: secret,
			Header: "X-Signature",
		},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))
	resp, _ := c.Get(context.Background(), "/api/orders")
	fmt.Printf("  ✓ status=%d\n", resp.StatusCode())
	fmt.Printf("    X-Signature: %s\n", truncate(gotSig, 70))

	// Verify the signature locally
	parts := parseSignature(gotSig)
	msg := "GET\n" + srv.URL + "/api/orders\n" + parts["ts"]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	expected := hex.EncodeToString(mac.Sum(nil))
	fmt.Printf("    Signature valid: %v\n", parts["sig"] == expected)
}

// [5] Idempotency Key injection.
func exampleIdempotencyKey() {
	fmt.Println("\n[5] Idempotency Key — auto-injected for non-GET requests")

	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	transport := &httpxauth.IdempotencyTransport{Header: "Idempotency-Key"}
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithTransport(transport))

	for range 3 {
		c.Post(context.Background(), "/payments", httpx.WithJSONBody(map[string]int{"amount": 100}))
	}

	fmt.Printf("  ✓ 3 POST requests, each with unique idempotency key:\n")
	allUnique := true
	seen := map[string]bool{}
	for i, k := range keys {
		if seen[k] {
			allUnique = false
		}
		seen[k] = true
		fmt.Printf("    req %d: %s\n", i+1, k)
	}
	fmt.Printf("    All unique: %v\n", allUnique)

	// GET should NOT get idempotency key
	keys = nil
	c.Get(context.Background(), "/payments")
	fmt.Printf("  ✓ GET has no idempotency key: %v\n", len(keys) > 0 && keys[0] == "")
}

// [6] Basic Auth via request builder.
func exampleBasicAuth() {
	fmt.Println("\n[6] HTTP Basic Auth")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		gotAuth = fmt.Sprintf("user=%s pass=%s", user, pass)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL))
	req, _ := c.NewRequest(context.Background(), "GET", "/admin").
		BasicAuth("admin", "secret").
		Build()

	resp, _ := c.Do(req)
	fmt.Printf("  ✓ status=%d auth=%q\n", resp.StatusCode(), gotAuth)
}

// [7] Bearer token via request builder.
func exampleBearerTokenBuilder() {
	fmt.Println("\n[7] Bearer token via request builder")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL))
	req, _ := c.NewRequest(context.Background(), "GET", "/profile").
		BearerToken("eyJhbGciOiJIUzI1NiJ9.payload.signature").
		Build()

	c.Do(req)
	fmt.Printf("  ✓ Authorization: %s\n", gotAuth)
}

// ---

type rotatingTokenSource struct {
	tokens []string
	idx    *int
}

func (r *rotatingTokenSource) Token(_ context.Context) (string, error) {
	t := r.tokens[*r.idx%len(r.tokens)]
	*r.idx++
	return t, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func parseSignature(sig string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(sig, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			out[kv[0]] = kv[1]
		}
	}
	return out
}
