// Package middleware demonstrates httpx middleware features:
// - Custom RoundTripper middleware
// - HeaderInjector
// - CorrelationIDInjector
// - TimeoutMiddleware
// - SingleflightMiddleware
// - Before/After hooks
// - Middleware chaining order
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/n0l3r/httpx"
)

// Run executes all middleware examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  MIDDLEWARE EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleCustomMiddleware()
	exampleHeaderInjector()
	exampleCorrelationID()
	exampleTimeoutMiddleware()
	exampleSingleflightMiddleware()
	exampleMiddlewareChainOrder()
	exampleBeforeAfterHooks()
}

// [1] Custom middleware — log timing per request.
func exampleCustomMiddleware() {
	fmt.Println("\n[1] Custom timing middleware")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	timingMW := func(next http.RoundTripper) http.RoundTripper {
		return httpx.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := next.RoundTrip(req)
			fmt.Printf("  [timing] %s %s → %dms\n",
				req.Method, req.URL.Path, time.Since(start).Milliseconds())
			return resp, err
		})
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithMiddleware(timingMW))
	c.Get(context.Background(), "/api/v1/users")
	c.Post(context.Background(), "/api/v1/orders", httpx.WithJSONBody(map[string]int{"qty": 1}))
}

// [2] HeaderInjector — inject static headers.
func exampleHeaderInjector() {
	fmt.Println("\n[2] HeaderInjector — inject static headers")

	var gotHeaders map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = map[string]string{
			"X-Service": r.Header.Get("X-Service"),
			"X-Version": r.Header.Get("X-Version"),
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.HeaderInjector(map[string]string{
			"X-Service": "payment-svc",
			"X-Version": "2.1.0",
		})),
	)

	c.Get(context.Background(), "/ping")
	fmt.Printf("  ✓ X-Service: %q\n", gotHeaders["X-Service"])
	fmt.Printf("    X-Version: %q\n", gotHeaders["X-Version"])
}

// [3] CorrelationIDInjector — auto-inject request IDs.
func exampleCorrelationID() {
	fmt.Println("\n[3] CorrelationIDInjector — auto-inject request IDs")

	var gotIDs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIDs = append(gotIDs, r.Header.Get("X-Request-ID"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.CorrelationIDInjector("X-Request-ID", func() string {
			b := make([]byte, 4)
			rand.Read(b)
			return "req-" + hex.EncodeToString(b)
		})),
	)

	for range 3 {
		c.Get(context.Background(), "/")
	}

	fmt.Printf("  ✓ Request IDs (all unique): %v\n", gotIDs)
	allUnique := len(unique(gotIDs)) == len(gotIDs)
	fmt.Printf("    All unique: %v\n", allUnique)
}

// [4] TimeoutMiddleware — per-request timeout.
func exampleTimeoutMiddleware() {
	fmt.Println("\n[4] TimeoutMiddleware — 50ms per request")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(200 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.TimeoutMiddleware(50*time.Millisecond)),
	)

	resp, err := c.Get(context.Background(), "/fast")
	fmt.Printf("  fast endpoint: status=%d err=%v\n", statusOrZero(resp), err)

	_, err = c.Get(context.Background(), "/slow")
	fmt.Printf("  slow endpoint: err=%v\n", err)
}

// [5] SingleflightMiddleware — deduplicate concurrent GETs.
func exampleSingleflightMiddleware() {
	fmt.Println("\n[5] SingleflightMiddleware — deduplicates concurrent GET requests")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond) // simulate latency
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"call":%d}`, calls.Load())
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.SingleflightMiddleware()),
	)

	// Fire 5 concurrent requests to the same URL
	results := make(chan int, 5)
	for range 5 {
		go func() {
			resp, err := c.Get(context.Background(), "/data")
			if err == nil {
				results <- resp.StatusCode()
			}
		}()
	}

	for range 5 {
		<-results
	}
	fmt.Printf("  ✓ 5 concurrent GETs → server called %d time(s) (deduplicated)\n", calls.Load())
}

// [6] Middleware chain order.
func exampleMiddlewareChainOrder() {
	fmt.Println("\n[6] Middleware chain order — outermost first")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var order []string

	makeOrderMW := func(name string) httpx.Middleware {
		return func(next http.RoundTripper) http.RoundTripper {
			return httpx.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				order = append(order, name+"→")
				resp, err := next.RoundTrip(req)
				order = append(order, "←"+name)
				return resp, err
			})
		}
	}

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(
			makeOrderMW("A"),
			makeOrderMW("B"),
			makeOrderMW("C"),
		),
	)

	c.Get(context.Background(), "/")
	fmt.Printf("  ✓ Execution order: %v\n", order)
	fmt.Println("    (outermost registered = first executed)")
}

// [7] Before/After hooks.
func exampleBeforeAfterHooks() {
	fmt.Println("\n[7] Before/After request hooks")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithBeforeRequest(func(ctx context.Context, req *http.Request) {
			fmt.Printf("  [before] %s %s\n", req.Method, req.URL.Path)
		}),
		httpx.WithAfterResponse(func(ctx context.Context, req *http.Request, resp *http.Response, err error) {
			if resp != nil {
				fmt.Printf("  [after]  %s %s → %d\n", req.Method, req.URL.Path, resp.StatusCode)
			}
		}),
	)

	c.Get(context.Background(), "/orders")
	c.Post(context.Background(), "/orders", httpx.WithJSONBody(map[string]string{"item": "book"}))
}

// ---

func unique(ss []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func statusOrZero(r *httpx.Response) int {
	if r == nil {
		return 0
	}
	return r.StatusCode()
}
