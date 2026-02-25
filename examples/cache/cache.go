// Package cache demonstrates httpx caching features:
// - MemoryCache (in-process TTL)
// - NoopCache (disable caching)
// - TieredCache (L1 memory + L2 any backend)
// - Custom cache key / invalidation
package cache

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/n0l3r/httpx"
	"github.com/n0l3r/httpx/cache/tiered"
)

// Run executes all cache examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  CACHE EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleMemoryCache()
	exampleCacheHitMiss()
	exampleNoopCache()
	exampleTieredCache()
	exampleCacheTTLExpiry()
	exampleCacheInvalidation()
	exampleCacheOnlyGet()
}

func countingServer() (*httptest.Server, *atomic.Int32) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"path":%q,"call":%d}`, r.URL.Path, calls.Load())
	}))
	return srv, &calls
}

// [1] Basic MemoryCache usage.
func exampleMemoryCache() {
	fmt.Println("\n[1] MemoryCache — GET requests are cached")

	srv, calls := countingServer()
	defer srv.Close()

	cache := httpx.NewMemoryCache(1 * time.Minute)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(cache))

	_, _ = c.Get(context.Background(), "/products")      // server call
	resp, _ := c.Get(context.Background(), "/products")  // cache hit
	_, _ = c.Get(context.Background(), "/products")      // cache hit

	fmt.Printf("  ✓ 3 requests, server called %d time(s)\n", calls.Load())
	fmt.Printf("    Cached body: %s\n", resp.String())
}

// [2] Cache hit vs miss.
func exampleCacheHitMiss() {
	fmt.Println("\n[2] Cache hit vs miss — different paths")

	srv, calls := countingServer()
	defer srv.Close()

	cache := httpx.NewMemoryCache(1 * time.Minute)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(cache))

	paths := []string{"/a", "/b", "/a", "/c", "/b", "/a"}
	for _, p := range paths {
		c.Get(context.Background(), p)
	}
	fmt.Printf("  ✓ %d requests for 3 distinct paths, server called %d time(s)\n",
		len(paths), calls.Load())
}

// [3] NoopCache — disables caching entirely.
func exampleNoopCache() {
	fmt.Println("\n[3] NoopCache — caching disabled")

	srv, calls := countingServer()
	defer srv.Close()

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(httpx.NoopCache{}))

	for range 3 {
		c.Get(context.Background(), "/data")
	}
	fmt.Printf("  ✓ 3 requests, server called %d time(s) (no cache)\n", calls.Load())
}

// [4] TieredCache — L1 memory (short TTL) + L2 memory (long TTL).
func exampleTieredCache() {
	fmt.Println("\n[4] TieredCache — L1 (30s) + L2 (5min)")

	srv, calls := countingServer()
	defer srv.Close()

	l1 := httpx.NewMemoryCache(30 * time.Second)
	l2 := httpx.NewMemoryCache(5 * time.Minute)
	tieredCache := tiered.New(l1, l2)

	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(tieredCache))

	// First request → server
	c.Get(context.Background(), "/products")
	fmt.Printf("  → after 1st request: server calls=%d\n", calls.Load())

	// Second request → L1 hit
	c.Get(context.Background(), "/products")
	fmt.Printf("  → after 2nd request: server calls=%d (L1 hit)\n", calls.Load())

	// Simulate L1 expiry by hitting L2 directly
	l1.Delete(srv.URL + "/products")
	c.Get(context.Background(), "/products")
	fmt.Printf("  → after L1 eviction: server calls=%d (L2 hit, L1 back-filled)\n", calls.Load())

	fmt.Printf("  ✓ TieredCache working correctly\n")
}

// [5] Cache TTL expiry.
func exampleCacheTTLExpiry() {
	fmt.Println("\n[5] Cache TTL expiry")

	srv, calls := countingServer()
	defer srv.Close()

	cache := httpx.NewMemoryCache(50 * time.Millisecond) // very short TTL
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(cache))

	c.Get(context.Background(), "/item")
	fmt.Printf("  → call 1: server calls=%d\n", calls.Load())

	c.Get(context.Background(), "/item")
	fmt.Printf("  → call 2 (cache hit): server calls=%d\n", calls.Load())

	time.Sleep(60 * time.Millisecond) // wait for TTL expiry

	c.Get(context.Background(), "/item")
	fmt.Printf("  → call 3 (after TTL): server calls=%d (cache expired)\n", calls.Load())
}

// [6] Manual cache invalidation.
func exampleCacheInvalidation() {
	fmt.Println("\n[6] Cache invalidation")

	srv, calls := countingServer()
	defer srv.Close()

	cache := httpx.NewMemoryCache(5 * time.Minute)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(cache))

	key := srv.URL + "/resource"
	c.Get(context.Background(), "/resource")
	c.Get(context.Background(), "/resource") // hit
	fmt.Printf("  → before invalidation: server calls=%d\n", calls.Load())

	// Invalidate manually
	cache.Delete(key)

	c.Get(context.Background(), "/resource") // miss → server call
	fmt.Printf("  → after invalidation: server calls=%d\n", calls.Load())
}

// [7] POST requests are NOT cached.
func exampleCacheOnlyGet() {
	fmt.Println("\n[7] Only GET requests are cached (POST is not)")

	srv, calls := countingServer()
	defer srv.Close()

	cache := httpx.NewMemoryCache(5 * time.Minute)
	c, _ := httpx.New(httpx.WithBaseURL(srv.URL), httpx.WithCache(cache))

	for range 3 {
		c.Post(context.Background(), "/resource", httpx.WithJSONBody(map[string]int{"x": 1}))
	}
	fmt.Printf("  ✓ 3 POST requests → server called %d time(s) (POST never cached)\n", calls.Load())
}
