// Package retry demonstrates httpx retry and backoff features:
// - DefaultRetryPolicy
// - RetryOnNetworkError, RetryOnStatus5xx, RetryOnStatus429
// - Custom retry conditions (RetryOnStatuses, RetryOnErrors)
// - Exponential backoff, FullJitter, Constant, Linear
// - OnRetry callback
// - RetryOnlyIdempotent flag
package retry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/NTR3667/httpx"
)

// Run executes all retry examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  RETRY & BACKOFF EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleDefaultRetry()
	exampleRetryOn5xx()
	exampleRetryOn429()
	exampleCustomCondition()
	exampleExponentialBackoff()
	exampleOnRetryCallback()
	exampleRetryOnlyIdempotent()
}

// [1] Default retry policy — retries on network errors and 5xx.
func exampleDefaultRetry() {
	fmt.Println("\n[1] Default retry policy (network error + 5xx + 429)")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithRetryPolicy(httpx.DefaultRetryPolicy()),
	)

	resp, err := c.Get(context.Background(), "/")
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ Succeeded after %d attempts, status=%d\n", calls.Load(), resp.StatusCode())
}

// [2] Retry on 5xx with custom attempt count.
func exampleRetryOn5xx() {
	fmt.Println("\n[2] Retry on 5xx — up to 4 attempts")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 4 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, `{"ok":true}`)
	}))
	defer srv.Close()

	policy := &httpx.RetryPolicy{
		MaxAttempts: 4,
		Backoff:     httpx.ConstantBackoff(0), // no wait for demo
		Conditions:  []httpx.RetryConditionFunc{httpx.RetryOnStatus5xx},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRetryPolicy(policy))
	resp, _ := c.Get(context.Background(), "/")
	fmt.Printf("  ✓ attempts=%d  status=%d\n", calls.Load(), resp.StatusCode())
}

// [3] Retry on 429 Too Many Requests.
func exampleRetryOn429() {
	fmt.Println("\n[3] Retry on 429 Too Many Requests")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprintln(w, `{"data":"ok"}`)
	}))
	defer srv.Close()

	policy := &httpx.RetryPolicy{
		MaxAttempts: 3,
		Backoff:     httpx.ConstantBackoff(0),
		Conditions:  []httpx.RetryConditionFunc{httpx.RetryOnStatus429},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRetryPolicy(policy))
	resp, _ := c.Get(context.Background(), "/")
	fmt.Printf("  ✓ attempts=%d  status=%d\n", calls.Load(), resp.StatusCode())
}

// [4] Custom retry condition.
func exampleCustomCondition() {
	fmt.Println("\n[4] Custom retry condition — retry on 503 only")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := &httpx.RetryPolicy{
		MaxAttempts: 3,
		Backoff:     httpx.ConstantBackoff(0),
		Conditions:  []httpx.RetryConditionFunc{httpx.RetryOnStatuses(503)},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRetryPolicy(policy))
	resp, _ := c.Get(context.Background(), "/")
	fmt.Printf("  ✓ attempts=%d  status=%d\n", calls.Load(), resp.StatusCode())
}

// [5] Exponential backoff strategies comparison.
func exampleExponentialBackoff() {
	fmt.Println("\n[5] Backoff strategies (delay per attempt)")

	strategies := []struct {
		name string
		bo   httpx.BackoffStrategy
	}{
		{"FullJitter     ", httpx.FullJitterBackoff(100*time.Millisecond, 5*time.Second)},
		{"Exponential    ", httpx.ExponentialBackoff(100*time.Millisecond, 5*time.Second, 0.1)},
		{"Constant (500ms)", httpx.ConstantBackoff(500 * time.Millisecond)},
		{"Linear (100ms) ", httpx.LinearBackoff(100*time.Millisecond, 100*time.Millisecond)},
	}

	for _, s := range strategies {
		delays := make([]string, 4)
		for i := range delays {
			delays[i] = s.bo(i).Round(time.Millisecond).String()
		}
		fmt.Printf("  %-20s attempt 0-3: %v\n", s.name, delays)
	}
}

// [6] OnRetry callback — log each retry attempt.
func exampleOnRetryCallback() {
	fmt.Println("\n[6] OnRetry callback")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := &httpx.RetryPolicy{
		MaxAttempts: 3,
		Backoff:     httpx.ConstantBackoff(0),
		Conditions:  []httpx.RetryConditionFunc{httpx.RetryOnStatus5xx},
		OnRetry: func(attempt int, req *http.Request, resp *http.Response, err error) {
			statusCode := 0
			if resp != nil {
				statusCode = resp.StatusCode
			}
			fmt.Printf("    → retry #%d  status=%d\n", attempt, statusCode)
		},
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRetryPolicy(policy))
	resp, _ := c.Get(context.Background(), "/")
	fmt.Printf("  ✓ final status=%d  total_calls=%d\n", resp.StatusCode(), calls.Load())
}

// [7] RetryOnlyIdempotent — skip retry for POST.
func exampleRetryOnlyIdempotent() {
	fmt.Println("\n[7] RetryOnlyIdempotent — POST is not retried")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	policy := &httpx.RetryPolicy{
		MaxAttempts:         3,
		Backoff:             httpx.ConstantBackoff(0),
		Conditions:          []httpx.RetryConditionFunc{httpx.RetryOnStatus5xx},
		RetryOnlyIdempotent: true, // POST will NOT be retried
	}

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRetryPolicy(policy))
	c.Post(context.Background(), "/resource", httpx.WithJSONBody(map[string]string{"x": "y"}))
	fmt.Printf("  ✓ POST called %d time(s) (expected 1, no retry)\n", calls.Load())

	// GET should still be retried
	calls.Store(0)
	c.Get(context.Background(), "/resource")
	fmt.Printf("  ✓ GET called %d time(s) (expected 3, with retry)\n", calls.Load())
}
