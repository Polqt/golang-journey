# Challenge 08 — Partitioned Counting Bloom Filter

## Difficulty: Hard
## Category: Probabilistic Data Structures · Distributed Deduplication

---

## Problem Statement

Bloom filters power deduplication in Kafka (consumer offset skip), web crawlers (URL seen),
and ad tech (impression dedup). Standard Bloom filters can't remove items — the production
solution is a **Counting Bloom Filter** using 4-bit counters.

For distributed systems, you need a **partitioned** variant: multiple independent sub-filters,
each responsible for a key range based on the key's hash prefix. This enables:
- Shard-local operations without global locks
- Merging filters from distributed nodes (union semantics)
- Approximate cardinality estimation per shard

---

## Requirements

1. `Add(key string)` — increment counters for all k hash positions
2. `Remove(key string) error` — decrement counters (error if count would underflow to negative)
3. `MightContain(key string) bool` — false positives allowed; false negatives never
4. `FalsePositiveRate() float64` — estimate FPR based on current fill ratio
5. `Merge(other *BloomFilter) error` — union two filters of identical config
6. `EstimatedCount() int64` — approximate distinct keys using the formula: `-n/k * ln(1 - X/m)` where X = set bits

---

## Config

```go
NewBloomFilter(capacity int, targetFPR float64, partitions int)
// capacity: expected number of distinct items
// partitions: number of independent sub-filters
```

Automatically compute optimal `m` (bit array size) and `k` (hash count):
```
m = -n * ln(targetFPR) / (ln 2)^2
k = (m/n) * ln 2
```

---

## Constraints

- Use FNV-1a + MurmurHash3 (in stdlib via `encoding/binary` seeded variants) for k independent hashes
- 4-bit counters: store 2 counters per byte (nibble packing)
- Partitioned: key hashes to shard `hash(key) % partitions`
- All operations goroutine-safe

---

## Acceptance Criteria

- [ ] Observed FPR < 2x target FPR at designed capacity
- [ ] No false negatives ever
- [ ] Successful merge of two non-overlapping filters
- [ ] `EstimatedCount` within 20% of actual count

---

## Stretch Goals

- Implement **Scalable Bloom Filter** (SBF): auto-grow when FPR degrades
- Add **TTL Bloom Filter** using a rotating set of standard filters (sliding window)
