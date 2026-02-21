# Challenge 05 — Circuit Breaker State Machine

## Difficulty: Medium-Hard
## Category: Resilience Patterns · Distributed Systems

---

## Problem Statement

Netflix's Hystrix, resilience4j, and Go's `sony/gobreaker` provide circuit breakers that protect
services from cascading failures. The core insight: fail fast when a downstream is unhealthy
rather than queuing threads until timeouts cascade.

Implement a **production-grade circuit breaker** with these states:
- **CLOSED** — requests pass through; failures are counted
- **OPEN** — requests are immediately rejected with `ErrCircuitOpen`; auto-resets after `resetTimeout`
- **HALF-OPEN** — allows `probeCount` probe requests; if enough succeed → CLOSED, else → OPEN

---

## Requirements

- `Execute(fn func() error) error` — run fn if circuit is closed/half-open; reject if open
- Transitions CLOSED → OPEN when `failureRate > threshold` over a rolling window of N requests
- Transitions OPEN → HALF-OPEN after `resetTimeout` automatically
- Transitions HALF-OPEN → CLOSED when `probeSuccessRate >= 0.5`; → OPEN on any probe failure
- `State() CircuitState` and `Stats() BreakerStats` for observability

---

## Constraints

- Rolling window must be a fixed-size **circular buffer** (not unbounded slice)
- State transitions must be atomic — no transition can be observed mid-flight
- `Execute` must not hold any lock while calling `fn`

---

## Hints

1. Model state as an `atomic.Int32` for lockless state reads
2. Record outcome ring buffer: `[windowSize]bool` with head pointer
3. On state read in `Execute`, decide: reject (OPEN), allow+count (CLOSED), probe (HALF-OPEN)
4. Use a separate mutex only for state transitions and counter resets

---

## Acceptance Criteria

- [ ] Injecting 60% errors on a 10-request window triggers OPEN state
- [ ] After `resetTimeout`, the circuit transitions to HALF-OPEN
- [ ] Two successful probes close the circuit again
- [ ] Goroutine-safe with `go test -race`
