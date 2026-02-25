// Package ratelimiter demonstrates httpx rate limiting features:
// - GlobalRateLimiter (in-process token bucket)
// - PerHostRateLimiter (per-host token bucket)
// - Rate limiter + context cancellation
package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/n0l3r/httpx"
	"golang.org/x/time/rate"
)

// Run executes all rate limiter examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  RATE LIMITER EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleGlobalRateLimiter()
	examplePerHostRateLimiter()
	exampleRateLimiterThroughput()
	exampleRateLimiterContextCancel()
}

// [1] GlobalRateLimiter — all requests share one limit.
func exampleGlobalRateLimiter() {
	fmt.Println("\n[1] GlobalRateLimiter — 5 req/sec, burst 2")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rl := httpx.NewGlobalRateLimiter(rate.Limit(5), 2)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRateLimiter(rl))

	start := time.Now()
	for i := range 5 {
		resp, err := c.Get(context.Background(), fmt.Sprintf("/item/%d", i))
		if err != nil {
			fmt.Printf("  ✗ request %d: %v\n", i, err)
			continue
		}
		fmt.Printf("  ✓ request %d at %dms → %d\n", i, time.Since(start).Milliseconds(), resp.StatusCode())
	}
}

// [2] PerHostRateLimiter — different limits per host.
func examplePerHostRateLimiter() {
	fmt.Println("\n[2] PerHostRateLimiter — slow host gets lower limit")

	fastSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fastSrv.Close()

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer slowSrv.Close()

	fastHost := fastSrv.Listener.Addr().String()
	slowHost := slowSrv.Listener.Addr().String()

	rl := httpx.NewPerHostRateLimiter(
		rate.Limit(10), 5, // default: 10 req/s, burst 5
		map[string]*rate.Limiter{
			slowHost: rate.NewLimiter(rate.Limit(1), 1), // slow host: 1 req/s
		},
	)

	c, _ := httpx.New(httpx.WithRateLimiter(rl))

	// Fast host: 3 rapid requests
	start := time.Now()
	for i := range 3 {
		c.Get(context.Background(), "http://"+fastHost+"/")
		fmt.Printf("  fast host req %d at %dms\n", i+1, time.Since(start).Milliseconds())
	}

	// Slow host: requests are throttled to 1/s
	start = time.Now()
	for i := range 2 {
		c.Get(context.Background(), "http://"+slowHost+"/")
		fmt.Printf("  slow host req %d at %dms\n", i+1, time.Since(start).Milliseconds())
	}
}

// [3] Rate limiter throughput measurement.
func exampleRateLimiterThroughput() {
	fmt.Println("\n[3] Throughput measurement — 10 req/sec")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const targetRPS = 10
	const numRequests = 15

	rl := httpx.NewGlobalRateLimiter(rate.Limit(targetRPS), 1)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRateLimiter(rl))

	start := time.Now()
	var wg sync.WaitGroup
	for i := range numRequests {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Get(context.Background(), fmt.Sprintf("/item/%d", n))
		}(i)
	}
	wg.Wait()

	elapsed := time.Since(start)
	actualRPS := float64(numRequests) / elapsed.Seconds()
	fmt.Printf("  ✓ %d requests in %v\n", numRequests, elapsed.Round(time.Millisecond))
	fmt.Printf("    Actual RPS: %.1f (target: %d)\n", actualRPS, targetRPS)
}

// [4] Rate limiter with context cancellation.
func exampleRateLimiterContextCancel() {
	fmt.Println("\n[4] Rate limiter + context cancellation")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Very slow limiter: 0.5 req/s (one request every 2 seconds)
	rl := httpx.NewGlobalRateLimiter(rate.Limit(0.5), 0)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithRateLimiter(rl))

	// First request: uses the burst token
	c.Get(context.Background(), "/first")

	// Second request: would wait 2 seconds — cancel after 100ms
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var blocked atomic.Bool
	blocked.Store(true)

	go func() {
		time.Sleep(50 * time.Millisecond)
		if blocked.Load() {
			fmt.Println("  → request is blocked by rate limiter...")
		}
	}()

	_, err := c.Get(ctx, "/second")
	blocked.Store(false)

	if err != nil {
		fmt.Printf("  ✓ Rate limiter blocked, context cancelled: %v\n", err)
	}
}
