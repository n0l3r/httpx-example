# httpx-demo

Implementasi komprehensif semua fitur [`httpx`](../httpx) — production-grade HTTP client library untuk Go.

## Menjalankan

```bash
# Clone & setup
cd httpx-demo

# Jalankan semua demo
go run main.go

# Jalankan kategori tertentu
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

## Struktur

```
httpx-demo/
├── main.go                          # Entry point, runner semua demo
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
    └── mock_test/      mock.go      # MockTransport untuk testing
```

---

## Demo per Kategori

### 🧱 Basic (`examples/basic`)

| # | Demo | Fitur |
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

| # | Demo | Fitur |
|---|---|---|
| 1 | Default policy | `httpx.DefaultRetryPolicy()` |
| 2 | Retry on 5xx | `RetryOnStatus5xx` |
| 3 | Retry on 429 | `RetryOnStatus429` |
| 4 | Custom condition | `RetryOnStatuses(503)` |
| 5 | Backoff strategies | `FullJitter`, `Exponential`, `Constant`, `Linear` |
| 6 | OnRetry callback | `policy.OnRetry` |
| 7 | Idempotent-only | `RetryOnlyIdempotent: true` |

### 💾 Cache (`examples/cache`)

| # | Demo | Fitur |
|---|---|---|
| 1 | MemoryCache | `httpx.NewMemoryCache(ttl)` |
| 2 | Hit vs miss | Different paths = different cache entries |
| 3 | NoopCache | `httpx.NoopCache{}` |
| 4 | TieredCache | `tiered.New(l1, l2)` — L1 back-fill dari L2 |
| 5 | TTL expiry | Auto-evict setelah TTL habis |
| 6 | Invalidation | `cache.Delete(key)` manual evict |
| 7 | POST not cached | Hanya GET yang di-cache |

### ⚡ Circuit Breaker (`examples/circuit_breaker`)

| # | Demo | Fitur |
|---|---|---|
| 1 | SimpleCircuitBreaker | `Closed → Open → HalfOpen → Closed` |
| 2 | Integrated with client | `WithCircuitBreaker(cb)` |
| 3 | gobreaker adapter | `WithExecutingCircuitBreaker(adapter)` |
| 4 | CB + logging | CB + `WithLogHook` |

### 🚦 Rate Limiter (`examples/rate_limiter`)

| # | Demo | Fitur |
|---|---|---|
| 1 | GlobalRateLimiter | `NewGlobalRateLimiter(rps, burst)` |
| 2 | PerHostRateLimiter | `NewPerHostRateLimiter(rps, burst, perHost)` |
| 3 | Throughput measurement | Verifikasi actual RPS ≈ target |
| 4 | Context cancel | RL + `context.WithTimeout` |

### 🔗 Middleware (`examples/middleware`)

| # | Demo | Fitur |
|---|---|---|
| 1 | Custom middleware | `RoundTripperFunc` timing wrapper |
| 2 | HeaderInjector | `httpx.HeaderInjector(headers)` |
| 3 | CorrelationIDInjector | `httpx.CorrelationIDInjector(header, fn)` |
| 4 | TimeoutMiddleware | `httpx.TimeoutMiddleware(d)` |
| 5 | SingleflightMiddleware | `httpx.SingleflightMiddleware()` |
| 6 | Chain order | A→B→C→←C←B←A |
| 7 | Before/After hooks | `WithBeforeRequest`, `WithAfterResponse` |

### 🔐 Auth (`examples/auth`)

| # | Demo | Fitur |
|---|---|---|
| 1 | OAuth 1.0a | `auth.OAuth1Transport` — HMAC-SHA256 signed |
| 2 | OAuth 2.0 static | `auth.StaticTokenSource` |
| 3 | OAuth 2.0 custom | Custom `TokenSource` (auto-refresh pattern) |
| 4 | HMAC signing | `auth.HMACTransport` — keyId + ts + sig |
| 5 | Idempotency Key | `auth.IdempotencyTransport` |
| 6 | Basic Auth | `.BasicAuth(user, pass)` di request builder |
| 7 | Bearer token | `.BearerToken(token)` di request builder |

### 📊 Tracing (`examples/tracing`)

| # | Demo | Fitur |
|---|---|---|
| 1 | Basic OTel span | `tracing.Transport` — span per request |
| 2 | Trace propagation | W3C `Traceparent` header injection |
| 3 | Error span | 5xx → span status `Error` |
| 4 | Manual span | Parent span wrapping multiple HTTP calls |

### 🔁 Singleflight (`examples/singleflight`)

| # | Demo | Fitur |
|---|---|---|
| 1 | SingleflightMiddleware | 10 goroutines → 1 server call |
| 2 | WithSingleflight | Client-level option |
| 3 | POST not deduped | POST selalu dikirim |
| 4 | Latency benefit | 20 concurrent calls selesai dalam ~1x server delay |

### 🧪 Mock (`examples/mock_test`)

| # | Demo | Fitur |
|---|---|---|
| 1 | Basic mock | `OnGet(path, handler)` |
| 2 | Simulate errors | Network error + 4xx + 5xx |
| 3 | Multi method | OnGet, OnPost, OnPut, OnDelete |
| 4 | Table-driven | Scenario testing pattern |
| 5 | CallCount | `mt.CallCount()`, `mt.Requests` |
| 6 | Default handler | Catch-all untuk unknown routes |

---

## Catatan Desain

- Demo ini menggunakan `httptest.NewServer` untuk semua test server — tidak butuh external API
- Setiap example adalah fungsi independen, bisa dipelajari dan dijalankan terpisah
- `go run main.go <category>` menjalankan hanya satu kategori
- Repo ini menggunakan `replace` directive di `go.mod` untuk point ke local `httpx` package

```
require github.com/NTR3667/httpx v0.0.0-...
replace github.com/NTR3667/httpx => ../httpx
```
