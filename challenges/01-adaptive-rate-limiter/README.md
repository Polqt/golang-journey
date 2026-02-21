# Challenge 01 — Adaptive Rate Limiter

## Difficulty: Hard
## Category: Concurrency · Distributed Systems · Algorithms

---

## Problem Statement

Production API gateways (Envoy, Kong, AWS API Gateway) use **hybrid rate limiting** strategies
that combine the predictability of a **Token Bucket** with the precision of a **Sliding Window
Log** to handle bursty traffic while preventing starvation.

Your task is to build a **multi-tenant, adaptive rate limiter** that:

1. Accepts requests identified by a `tenantID` string
2. Enforces per-tenant rate limits with independent token buckets
3. **Adaptively tightens** limits when error rates (HTTP 5xx) for a tenant exceed a threshold
4. Exposes a `Stats()` method per tenant showing: allowed, rejected, current token count, adaptive factor
5. Is **goroutine-safe** — concurrent calls from many goroutines must produce correct results

---

## Constraints

- Do **not** use any third-party library
- Token refill must be time-based (wall clock), not tick-based
- Adaptive factor must down-scale tokens when `errorRate > 0.3` for a tenant (scale by `1 - errorRate`)
- Adaptive factor must recover gradually (1% recovery per second) when error rate drops back below 0.1
- All operations must complete in O(1) amortized time
- `Allow(tenantID string, wasError bool) bool` is your core API

---

## Example

```
limiter := NewAdaptiveRateLimiter(100, 10) // 100 tokens/sec, burst 10
limiter.Allow("tenant-A", false) // → true (token granted)
limiter.Allow("tenant-A", true)  // → true (but error counted)
limiter.Stats("tenant-A")        // → {Allowed:2, Rejected:0, Tokens:8.0, AdaptiveFactor:0.97}
```

---

## Hints

1. `sync.Mutex` per tenant map entry — avoid a global lock where possible
2. Model the token bucket as: `tokens = min(burst, tokens + rate * elapsed * adaptiveFactor)`
3. Track a rolling error count over the last N seconds using a **circular buffer**
4. The adaptive factor is a float64 in `[0.1, 1.0]` — never let it drop to zero
5. Test with `100` concurrent goroutines hammering the same tenant

---

## Acceptance Criteria

- [ ] Tokens correctly deplete and refill based on elapsed wall time
- [ ] Concurrent access produces no data races (`go test -race`)
- [ ] Error injection correctly reduces the adaptive factor over time
- [ ] Recovery correctly increases the adaptive factor back toward 1.0
- [ ] Benchmark shows < 1µs per `Allow()` call under contention

---

## Stretch Goals

- Implement a **sliding window counter** variant and benchmark both approaches
- Add per-tenant **priority queuing** so premium tenants get burst headroom
- Implement Redis-compatible Lua script equivalent (atomic CAS logic in Go)
