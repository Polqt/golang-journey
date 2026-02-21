# Challenge 04 — Concurrent LRU with TTL and Frequency Bias

## Difficulty: Medium-Hard
## Category: Data Structures · Caching · Concurrency

---

## Problem Statement

Redis, Memcached, and Varnish use variants of LRU that address its core weakness: a full cache
scan (large sequential read) evicts *hot* keys. The fix used in production is **LFU-biased LRU**
(used by Redis 4.0+): evict the entry that is *both* least recently used *and* least frequently
accessed, with a time-to-live to prevent stale data.

Implement a **thread-safe, TTL-aware, frequency-biased LRU cache** with O(1) all operations.

---

## Requirements

| Method | Behaviour |
|---|---|
| `Set(key, value string, ttl time.Duration)` | Insert / update entry with TTL |
| `Get(key string) (string, bool)` | Return value; update recency and frequency count |
| `Delete(key string)` | Explicitly remove an entry |
| `Len() int` | Number of non-expired entries |
| `Stats() CacheStats` | Hits, misses, evictions counters |

Eviction policy: when at capacity, pick the candidate from the **bottom 20% of LRU order** with
the **lowest access frequency**. Expired entries are lazily removed on access.

---

## Constraints

- All operations must be O(1) — no O(n) scans
- Goroutine-safe with **no global lock** for reads (hint: `sync.RWMutex`)
- Maximum capacity is set at construction time
- Access frequency is stored as a uint32 counter per entry (saturates at 255 to save memory — like Redis)

---

## Hints

1. Use a doubly-linked list + map for the LRU layer (standard approach)
2. Segment the LRU tail into a "victim pool" of size `capacity * 0.2`
3. Within the victim pool, pick the entry with minimum frequency counter
4. TTL expiry: store `expiresAt time.Time`; check on `Get` — if expired, treat as miss and delete lazily
5. For the frequency counter, increment on `Get` and cap at 255

---

## Acceptance Criteria

- [ ] `go test -race` passes
- [ ] Hit rate on Zipf workload (80/20 access pattern) > 85% at 1000 capacity
- [ ] 1M operations/sec throughput on 8 cores
- [ ] Expired entries are never returned

---

## Stretch Goals

- Implement **segmented LRU (SLRU)** — probationary and protected segments
- Add `Range(fn func(k, v string) bool)` for cache-warming snapshots
