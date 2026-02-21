# Project 09 — Edge Cache CDN

> **Difficulty**: Senior · **Domain**: Networking, Caching, Distributed Systems, HTTP
> **Real-world analog**: Cloudflare, Fastly, Varnish Cache, nginx proxy_cache, Squid

---

## Why This Project Exists

CDNs are ubiquitous but mysterious. Understanding how edge caching, cache invalidation,
surrogate keys, origin shielding, and stale-while-revalidate work is essential knowledge
for any senior backend engineer. This project builds a real, multi-node edge cache that
handles real HTTP traffic.

---

## Folder Structure

```
09-edge-cache-cdn/
├── go.mod
├── main.go                              # CLI: edge start, control invalidate
├── edge/
│   ├── proxy.go                         # Reverse proxy + cache check
│   ├── cache/
│   │   ├── store.go                     # Cache store interface
│   │   ├── memory.go                    # In-memory LRU store (Challenge 04!)
│   │   └── disk.go                      # Memory-mapped disk store
│   ├── rules/
│   │   ├── ttl.go                       # TTL extraction from cache-control headers
│   │   ├── vary.go                      # Vary header handling
│   │   └── bypass.go                    # Cache bypass rules (cookies, auth)
│   ├── coalesce/
│   │   └── coalesce.go                  # Request coalescing (collapse / request collapse)
│   ├── revalidate/
│   │   └── stale.go                     # stale-while-revalidate + stale-if-error
│   └── purge/
│       ├── purge.go                     # URL + surrogate key purge
│       └── tags.go                      # Cache-Tag / Surrogate-Key index
├── origin/
│   └── shield.go                        # Origin shield (PoP-to-PoP fetching)
├── cluster/
│   ├── node.go                          # Node identity + peer list
│   ├── gossip.go                        # Lightweight gossip for purge propagation
│   └── replication.go                   # Cache warm-up from peers
├── api/
│   └── control.go                       # Purge / ban API (Varnish-style)
└── config/
    └── config.yaml
```

---

## Implementation Guide

### Phase 1 — Core HTTP Caching (Week 1)

Implement RFC 7234 HTTP caching semantics:

```
Incoming request
    ↓
[1] Is response in cache and fresh?  → serve from cache (HIT)
    ↓ no
[2] Should this request bypass cache? → proxy to origin, don't cache
    ↓ no
[3] Fetch from origin, parse cache-control headers
    ↓
[4] Cacheable? → store in cache
    ↓
[5] Return response
```

**Cache key**: `method + url + vary-header-values`

**TTL extraction** (in order of precedence):
1. `s-maxage` directive in `Cache-Control`
2. `max-age` directive
3. `Expires` header
4. Heuristic: 10% of `(Date - Last-Modified)` (RFC 7234 §4.2.2)

---

### Phase 2 — Request Coalescing (Week 2)

**The thundering herd problem**: when a cached object expires, hundreds of concurrent
requests all miss the cache and hammer the origin simultaneously.

**Solution — request coalescing** (also called "request collapsing"):
- First request to miss → send to origin, create a "pending" entry in cache
- All subsequent requests for the same key → wait for the first request to complete
- When origin responds → broadcast to all waiters

```go
type Coalescer struct {
    mu      sync.Mutex
    inflight map[string]*inflightRequest
}
type inflightRequest struct {
    done chan struct{}
    resp *CachedResponse
    err  error
}
func (c *Coalescer) Do(key string, fetch func() (*CachedResponse, error)) (*CachedResponse, error)
```

---

### Phase 3 — Stale-While-Revalidate (Week 2)

RFC 5861 introduces two powerful directives:
- `stale-while-revalidate=N`: serve stale content immediately, revalidate in background
- `stale-if-error=N`: serve stale content if origin returns 5xx (for up to N seconds)

```go
// On cache miss where object is "stale but allowed":
// 1. Immediately serve the stale response to the client
// 2. Asynchronously fetch fresh response from origin
// 3. Update cache with fresh response
```

This eliminates tail latency spikes on cache expiry — users always get fast responses.

---

### Phase 4 — Surrogate Key Invalidation (Week 3)

Traditional URL-based purging requires knowing every URL that needs invalidation (impractical
for content like "all blog posts by author X"). **Surrogate keys** (Cache-Tag in Cloudflare,
xkey in Fastly, Surrogate-Key in Varnish) solve this:

Origin sets on response: `Cache-Tag: author-42 post-100 category-tech`
Purge by tag: `PURGE-BY-TAG author-42` → instantly invalidates all cached responses with that tag

**Implementation**:
1. On store: parse `Cache-Tag` header, build inverted index: `tag → []cacheKeys`
2. On purge: look up tag in index, delete all associated cache entries
3. Index must be lock-efficient (shard by tag hash)

---

### Phase 5 — Multi-Node Gossip Invalidation (Week 3-4)

In a multi-node CDN, a purge must propagate to all edge nodes within seconds.
Use the gossip protocol from Challenge 09 as inspiration:

```go
// When node A receives a purge request:
1. Apply purge locally
2. Send PurgeMsg{key: ..., tag: ..., tombstone: timestamp} to K random peers
3. Each peer applies purge + re-gossips if tombstone is newer than seen before
4. Convergence in log2(N) rounds
```

**Idempotency**: use tombstone timestamps to ignore duplicate purge messages.

---

### Vary Header Handling (Advanced)

The `Vary` header makes caching per-client-variation:
```
Vary: Accept-Encoding, Accept-Language
```
This means cached responses are keyed by URL + `Accept-Encoding` value + `Accept-Language` value.

Store variants as a list under each URL key. Check all variants on cache lookup.  
**Watch out**: `Vary: *` means "never cache" — always bypass.

---

## Config Format

```yaml
edge:
  listen: ":8080"
  origin: "https://api.example.com"
  cache:
    maxMemoryMB: 512
    maxDiskMB: 10240
    defaultTTL: 60s
  coalescing: true
  staleWhileRevalidate: true
control:
  listen: ":8081"
  purgeSecret: "changeme"
cluster:
  peers:
    - "edge-2:9000"
    - "edge-3:9000"
```

---

## Acceptance Criteria

- [ ] Cache hit rate > 80% on repeated requests with cacheable responses
- [ ] Request coalescing reduces origin requests by > 90% on cache miss storms
- [ ] Surrogate key purge invalidates all tagged entries atomically
- [ ] Gossip purge propagates to 3 nodes within 1 second

---

## Stretch Goals

- Implement **ESI** (Edge Side Includes): assemble pages from fragment-level cache entries
- Add **HTTP/2 Push**: proactively push linked assets to browser
- Build a **cache analytics dashboard**: hit rate, byte hit rate, TTL distribution histogram
