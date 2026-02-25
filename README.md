# httpx-example

Comprehensive examples demonstrating all features of [`httpx`](../httpx) — production-grade HTTP client library for Go.

## Running

```bash
# Clone & setup
cd httpx-example

# Run all examples
go run main.go

# Run a specific category
go run main.go basic
go run main.go retry
go run main.go cache
go run main.go circuit-breaker
go run main.go rate-limiter
go run main.go middleware
go run main.go auth
go run main.go tracing
go run main.go singleflight
go run main.go mock
```

---

## Structure

```
httpx-example/
├── main.go                          # Entry point, category runner
└── examples/
    ├── basic/          basic.go     # Core client features
    ├── retry/          retry.go     # Retry + backoff strategies
    ├── cache/          cache.go     # MemoryCache, NoopCache, TieredCache
    ├── circuit_breaker/ circuit_breaker.go  # SimpleCircuitBreaker + gobreaker
    ├── rate_limiter/   rate_limiter.go      # GlobalRateLimiter, PerHostRateLimiter
    ├── middleware/     middleware.go         # Custom & built-in middlewares
    ├── auth/           auth.go      # OAuth1, OAuth2, HMAC, Idempotency, Basic Auth
    ├── tracing/        tracing.go   # OpenTelemetry spans + propagation
    ├── singleflight/   singleflight.go      # Request deduplication
    └── mock_test/      mock.go      # MockTransport for testing
```

---

## Examples by Category

### 🧱 Basic (`examples/basic`)

| # | Example | Feature |
|---|---|---|
| 1 | New client with options | `WithBaseURL`, `WithTimeout`, `WithDefaultHeader`, `WithConnectionPool` |
| 2 | GetJSON | `c.GetJSON(ctx, path, &out)` |
| 3 | PostJSON | `c.PostJSON(ctx, path, body, &out)` |
| 4 | PutJSON | `c.PutJSON(ctx, path, body, &out)` |
| 5 | Delete | `c.Delete(ctx, path)` |
| 6 | Fluent builder | `.Header().Query().Accept().BearerToken().Build()` |
| 7 | Response helpers | `.IsSuccess()`, `.IsClientError()`, `.EnsureSuccess()` |
| 8 | Context deadline | `context.WithTimeout` → `httpx.IsTimeout(err)` |
| 9 | Default headers | `WithDefaultHeaders(map)` |

### 🔄 Retry (`examples/retry`)

| # | Example | Feature |
|---|---|---|
| 1 | Default policy | `httpx.DefaultRetryPolicy()` |
| 2 | Retry on 5xx | `RetryOnStatus5xx` |
| 3 | Retry on 429 | `RetryOnStatus429` |
| 4 | Custom condition | `RetryOnStatuses(503)` |
| 5 | Backoff strategies | `FullJitter`, `Exponential`, `Constant`, `Linear` |
| 6 | OnRetry callback | `policy.OnRetry` |
| 7 | Idempotent-only | `RetryOnlyIdempotent: true` |

### 💾 Cache (`examples/cache`)

| # | Example | Feature |
|---|---|---|
| 1 | MemoryCache | `httpx.NewMemoryCache(ttl)` |
| 2 | Hit vs miss | Different paths produce separate cache entries |
| 3 | NoopCache | `httpx.NoopCache{}` |
| 4 | TieredCache | `tiered.New(l1, l2)` — L1 back-fill from L2 |
| 5 | TTL expiry | Entry auto-evicted after TTL expires |
| 6 | Invalidation | `cache.Delete(key)` manual eviction |
| 7 | POST not cached | Only GET requests are eligible for caching |

### ⚡ Circuit Breaker (`examples/circuit_breaker`)

| # | Example | Feature |
|---|---|---|
| 1 | SimpleCircuitBreaker | `Closed → Open → HalfOpen → Closed` |
| 2 | Integrated with client | `WithCircuitBreaker(cb)` |
| 3 | gobreaker adapter | `WithExecutingCircuitBreaker(adapter)` |
| 4 | CB + logging | Circuit breaker combined with `WithLogHook` |

### 🚦 Rate Limiter (`examples/rate_limiter`)

| # | Example | Feature |
|---|---|---|
| 1 | GlobalRateLimiter | `NewGlobalRateLimiter(rps, burst)` |
| 2 | PerHostRateLimiter | `NewPerHostRateLimiter(rps, burst, perHost)` |
| 3 | Throughput measurement | Verify actual RPS stays within configured limit |
| 4 | Context cancel | Rate limiter respects `context.WithTimeout` |

### 🔗 Middleware (`examples/middleware`)

| # | Example | Feature |
|---|---|---|
| 1 | Custom middleware | `RoundTripperFunc` timing wrapper |
| 2 | HeaderInjector | `httpx.HeaderInjector(headers)` |
| 3 | CorrelationIDInjector | `httpx.CorrelationIDInjector(header, fn)` |
| 4 | TimeoutMiddleware | `httpx.TimeoutMiddleware(d)` |
| 5 | SingleflightMiddleware | `httpx.SingleflightMiddleware()` |
| 6 | Chain order | A→B→C→server→C→B→A execution order |
| 7 | Before/After hooks | `WithBeforeRequest`, `WithAfterResponse` |

### 🔐 Auth (`examples/auth`)

| # | Example | Feature |
|---|---|---|
| 1 | OAuth 1.0a | `auth.OAuth1Transport` — HMAC-SHA256 signed |
| 2 | OAuth 2.0 static | `auth.StaticTokenSource` |
| 3 | OAuth 2.0 custom | Custom `TokenSource` (auto-refresh pattern) |
| 4 | HMAC signing | `auth.HMACTransport` — keyId + timestamp + signature |
| 5 | Idempotency Key | `auth.IdempotencyTransport` |
| 6 | Basic Auth | `.BasicAuth(user, pass)` on request builder |
| 7 | Bearer token | `.BearerToken(token)` on request builder |

### 📊 Tracing (`examples/tracing`)

| # | Example | Feature |
|---|---|---|
| 1 | Basic OTel span | `tracing.Transport` — one span per request |
| 2 | Trace propagation | W3C `Traceparent` header injection |
| 3 | Error span | 5xx → span status set to `Error` |
| 4 | Manual span | Parent span wrapping multiple HTTP calls |

### 🔁 Singleflight (`examples/singleflight`)

| # | Example | Feature |
|---|---|---|
| 1 | SingleflightMiddleware | 10 goroutines → exactly 1 server call |
| 2 | WithSingleflight | Client-level option |
| 3 | POST not deduplicated | POST requests always reach the server |
| 4 | Latency benefit | 20 concurrent calls complete in ~1x server delay |

### 🧪 Mock (`examples/mock_test`)

| # | Example | Feature |
|---|---|---|
| 1 | Basic mock | `OnGet(path, handler)` |
| 2 | Simulate errors | Network error + 4xx + 5xx |
| 3 | Multi method | OnGet, OnPost, OnPut, OnDelete |
| 4 | Table-driven | Parameterized scenario testing pattern |
| 5 | CallCount | `mt.CallCount()`, `mt.Requests` |
| 6 | Default handler | Catch-all for unregistered routes |

---

## Design Notes

- All examples use `httptest.NewServer` — no external API or service required
- Each example is an independent function and can be studied or run in isolation
- `go run main.go <category>` runs only the specified category
- This repo uses a `replace` directive in `go.mod` to reference the local `httpx` package

```
require github.com/n0l3r/httpx v0.0.0-...
replace github.com/n0l3r/httpx => ../httpx
```
