// Package circuitbreaker demonstrates httpx circuit breaker features:
// - SimpleCircuitBreaker (built-in, allow/record pattern)
// - sony/gobreaker adapter (execute pattern) via WithExecutingCircuitBreaker
// - State transitions: Closed → Open → HalfOpen → Closed
package circuitbreaker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/n0l3r/httpx"
	gbadapter "github.com/n0l3r/httpx/breaker/gobreaker"
	gb "github.com/sony/gobreaker/v2"
)

// Run executes all circuit breaker examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  CIRCUIT BREAKER EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleSimpleCB()
	exampleSimpleCBRecovery()
	exampleGoBreakerAdapter()
	exampleCBWithLogging()
}

// [1] SimpleCircuitBreaker — opens after threshold failures.
func exampleSimpleCB() {
	fmt.Println("\n[1] SimpleCircuitBreaker — opens after 3 consecutive failures")

	cb := httpx.NewCircuitBreaker(httpx.CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 1,
		OpenTimeout:      100 * time.Millisecond,
	})

	// Simulate failures
	for i := range 3 {
		cb.RecordFailure("api.example.com")
		state := cb.Allow("api.example.com")
		fmt.Printf("  failure #%d → Allow: %v\n", i+1, formatErr(state))
	}

	// Circuit should now be open
	err := cb.Allow("api.example.com")
	fmt.Printf("  → Circuit is OPEN: %v\n", err)

	// After open timeout
	time.Sleep(110 * time.Millisecond)
	err = cb.Allow("api.example.com")
	fmt.Printf("  → After timeout (half-open): %v\n", formatErr(err))

	// Record success → closed
	cb.RecordSuccess("api.example.com")
	err = cb.Allow("api.example.com")
	fmt.Printf("  → After success (closed): %v\n", formatErr(err))
}

// [2] SimpleCircuitBreaker integrated into a client.
func exampleSimpleCBRecovery() {
	fmt.Println("\n[2] SimpleCircuitBreaker integrated with client")

	var (
		calls   atomic.Int32
		healthy atomic.Bool
	)
	healthy.Store(false) // start unhealthy

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if healthy.Load() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status":"ok"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cb := httpx.NewCircuitBreaker(httpx.CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		OpenTimeout:      80 * time.Millisecond,
	})

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCircuitBreaker(cb))

	// Two 5xx → circuit opens
	for range 2 {
		c.Get(context.Background(), "/health")
	}
	fmt.Printf("  → %d calls made (server unhealthy)\n", calls.Load())

	// Next call blocked by circuit breaker
	_, err := c.Get(context.Background(), "/health")
	fmt.Printf("  → Circuit open: %v\n", err)
	fmt.Printf("    Server calls still: %d (circuit blocked)\n", calls.Load())

	// Server recovers
	healthy.Store(true)
	time.Sleep(90 * time.Millisecond)

	// Half-open trial succeeds → circuit closes
	resp, err := c.Get(context.Background(), "/health")
	if err == nil {
		fmt.Printf("  ✓ Circuit closed after recovery, status=%d\n", resp.StatusCode())
	}
}

// [3] sony/gobreaker adapter.
func exampleGoBreakerAdapter() {
	fmt.Println("\n[3] sony/gobreaker adapter (execute pattern)")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 6 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"recovered":true}`)
	}))
	defer srv.Close()

	adapter := gbadapter.New(gbadapter.Config{
		Name:        "demo-api",
		Timeout:     80 * time.Millisecond,
		MaxRequests: 1,
		ReadyToTrip: func(c gb.Counts) bool {
			return c.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from, to gb.State) {
			fmt.Printf("    🔄 Circuit [%s]: %s → %s\n", name, from, to)
		},
	})

	// Use WithExecutingCircuitBreaker for execute-style circuit breakers
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithExecutingCircuitBreaker(adapter))

	// Generate failures to open the circuit
	for range 3 {
		c.Get(context.Background(), "/api")
	}

	// Circuit should be open now
	_, err := c.Get(context.Background(), "/api")
	fmt.Printf("  → Circuit open: %v\n", err != nil)

	// Wait for half-open, then succeed
	time.Sleep(90 * time.Millisecond)
	resp, err := c.Get(context.Background(), "/api")
	if err == nil && resp.IsSuccess() {
		fmt.Printf("  ✓ gobreaker adapter: recovered, status=%d\n", resp.StatusCode())
	} else if err != nil {
		fmt.Printf("  ✓ gobreaker still half-open/open: %v\n", err)
	}
}

// [4] Circuit breaker with logging.
func exampleCBWithLogging() {
	fmt.Println("\n[4] Circuit breaker + structured logging")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cb := httpx.NewCircuitBreaker(httpx.DefaultCircuitBreakerConfig)
	var logEvents []string

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithCircuitBreaker(cb),
		httpx.WithLogHook(httpx.LogHookFunc(func(e httpx.LogEvent) {
			logEvents = append(logEvents, fmt.Sprintf("%s %s → %d", e.Method, e.URL, e.StatusCode))
		})),
	)

	c.Get(context.Background(), "/ping")
	c.Get(context.Background(), "/health")

	fmt.Printf("  ✓ %d requests logged\n", len(logEvents))
	for _, ev := range logEvents {
		fmt.Printf("    %s\n", ev)
	}
}

func formatErr(err error) string {
	if err == nil {
		return "nil (allowed)"
	}
	return err.Error()
}
