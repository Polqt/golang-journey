# Challenge 02 — Fencing-Token Distributed Lock

## Difficulty: Hard
## Category: Distributed Systems · Concurrency · Safety

---

## Problem Statement

Distributed locks break under process pause (GC, swap, slow disk). LinkedIn, Cloudflare, and
Google solve this with **fencing tokens** — a monotonically increasing integer issued with every
lock acquisition. Any resource that accepts writes must **reject** requests carrying a token older
than the last seen one, even if the lock appears held.

You will implement a **fencing-token lock manager** with lease expiry, exactly as described in
Martin Kleppmann's "Designing Data-Intensive Applications" Chapter 8.

---

## Requirements

1. `Lock(clientID string) (token int64, err error)` — acquire lock, return fencing token
2. `Unlock(clientID string, token int64) error` — release lock, validate ownershiptoken
3. `WriteResource(clientID string, token int64, data string) error` — simulated resource write that **rejects stale tokens**
4. Lease-based auto-expiry: if a lock holder doesn't renew within `leaseDuration`, it is auto-released
5. `Renew(clientID string, token int64) error` — extend the lease
6. All operations are goroutine-safe

---

## Constraints

- Tokens must be **strictly monotonically increasing** across all acquisitions (even after release)
- Auto-expiry must be implemented without a dedicated goroutine (lazy expiry on next `Lock()` call)
- `WriteResource` tracks `highWaterMark`: the highest token it has ever seen — reject `token < highWaterMark`
- Simulate network partition by adding `SetPartitioned(clientID string, partitioned bool)` that causes `Renew()` to fail silently

---

## Example

```
lm := NewLockManager(500 * time.Millisecond)
tok1, _ := lm.Lock("client-A")  // tok1 = 1
lm.WriteResource("client-A", tok1, "data-v1") // OK
time.Sleep(600ms)                // lease expires

tok2, _ := lm.Lock("client-B")  // tok2 = 2 (new, higher)
lm.WriteResource("client-A", tok1, "stale!")  // ERROR: stale token
lm.WriteResource("client-B", tok2, "fresh")   // OK
```

---

## Hints

1. Keep a single `sync.Mutex` on the `LockManager`; this is a coordinator, not a hot path
2. `highWaterMark` is on the **resource**, not the lock manager
3. Use `time.Now().After(lock.expiresAt)` for lazy expiry — no goroutines needed
4. Think about what happens when two clients race to acquire after expiry

---

## Acceptance Criteria

- [ ] Token is strictly incremented even across expiry/reacquire cycles
- [ ] Stale client cannot write after its lease expires and another client acquires
- [ ] Partitioned client's renewal fails, causing eventual expiry
- [ ] No goroutine leaks (verify with `goleak`)

---

## Stretch Goals

- Implement **Redlock** (multi-node quorum) variant with N in-process "Redis" mocks
- Add persistence: snapshot lock state to a file and recover on restart
