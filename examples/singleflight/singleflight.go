// Package singleflight demonstrates httpx singleflight (request deduplication):
// - SingleflightMiddleware for concurrent GET deduplication
// - WithSingleflight client-level option
// - Only GET is deduplicated (POST is not)
package singleflight

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NTR3667/httpx"
)

// Run executes all singleflight examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  SINGLEFLIGHT EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleSingleflightMiddleware()
	exampleWithSingleflight()
	examplePostNotDeduplicated()
	exampleSingleflightLatency()
}

// [1] SingleflightMiddleware — concurrent GET deduplication.
func exampleSingleflightMiddleware() {
	fmt.Println("\n[1] SingleflightMiddleware — deduplicate concurrent GETs")

	var serverCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalls.Add(1)
		time.Sleep(50 * time.Millisecond) // simulate latency
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"calls":%d}`, serverCalls.Load())
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.SingleflightMiddleware()),
	)

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make([]string, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := c.Get(context.Background(), "/heavy-resource")
			if err == nil {
				results[idx] = resp.String()
			}
		}(i)
	}
	wg.Wait()

	fmt.Printf("  ✓ %d goroutines fired → server called %d time(s)\n",
		numGoroutines, serverCalls.Load())
	// All goroutines should have received the same response body
	allSame := true
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			allSame = false
			break
		}
	}
	fmt.Printf("    All received same response: %v\n", allSame)
}

// [2] WithSingleflight client option.
func exampleWithSingleflight() {
	fmt.Println("\n[2] WithSingleflight — client-level option")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"data":"result"}`)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithSingleflight(), // convenience option
	)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get(context.Background(), "/api/config")
		}()
	}
	wg.Wait()

	fmt.Printf("  ✓ 8 concurrent requests → server called %d time(s)\n", calls.Load())
}

// [3] POST is NOT deduplicated.
func examplePostNotDeduplicated() {
	fmt.Println("\n[3] POST requests are NOT deduplicated")

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.SingleflightMiddleware()),
	)

	const numPost = 5
	var wg sync.WaitGroup
	for range numPost {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Post(context.Background(), "/orders", httpx.WithJSONBody(map[string]int{"qty": 1}))
		}()
	}
	wg.Wait()

	fmt.Printf("  ✓ %d POST requests → server called %d time(s) (all executed)\n",
		numPost, calls.Load())
}

// [4] Singleflight reduces latency spikes.
func exampleSingleflightLatency() {
	fmt.Println("\n[4] Singleflight latency benefit")

	const serverDelay = 100 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(serverDelay)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"value":42}`)
	}))
	defer srv.Close()

	c, _ := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithMiddleware(httpx.SingleflightMiddleware()),
	)

	const numConcurrent = 20
	start := time.Now()
	var wg sync.WaitGroup
	for range numConcurrent {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get(context.Background(), "/expensive")
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("  ✓ %d concurrent calls completed in %v\n", numConcurrent, elapsed.Round(time.Millisecond))
	fmt.Printf("    Without singleflight: ~%v (serial) or network saturation (parallel)\n",
		time.Duration(numConcurrent)*serverDelay)
	fmt.Printf("    With singleflight: ~%v (single in-flight)\n", serverDelay)
}
