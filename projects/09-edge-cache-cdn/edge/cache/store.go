// Package cache implements the in-memory HTTP response cache.
package cache

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────
// CachedResponse
// ─────────────────────────────────────────────────────────────

// CachedResponse stores a complete HTTP response for replay.
type CachedResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte // full response body in memory
	CachedAt   time.Time
	Expires    time.Time // derived from Cache-Control max-age or Expires header
	ETag       string
	LastMod    string
	// Age indicates how long this entry has been cached.
	// Stale-while-revalidate: if StaleUntil.After(now), serve stale while fetching fresh.
	StaleUntil time.Time
}

// IsExpired returns true if the response has passed its Expires time.
func (r *CachedResponse) IsExpired(now time.Time) bool {
	return now.After(r.Expires)
}

// IsStale returns true if past Expires but still within StaleUntil.
func (r *CachedResponse) IsStale(now time.Time) bool {
	return r.IsExpired(now) && now.Before(r.StaleUntil)
}

// Age returns how long this entry has been in cache.
func (r *CachedResponse) Age(now time.Time) time.Duration {
	return now.Sub(r.CachedAt)
}

// ─────────────────────────────────────────────────────────────
// CacheKey
// ─────────────────────────────────────────────────────────────

// Key computes the cache key for an HTTP request.
// Default: host + path + sorted query string.
// Vary headers are handled separately.
func Key(r *http.Request) string {
	// TODO: include Vary headers in key computation for correct content negotiation.
	return r.Host + r.URL.RequestURI()
}

// ─────────────────────────────────────────────────────────────
// Store (LRU eviction + TTL)
// ─────────────────────────────────────────────────────────────

type entry struct {
	key  string
	resp *CachedResponse
}

// Store is an LRU in-memory cache of HTTP responses.
type Store struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*list.Element
	lru      *list.List
}

// New creates a Store with the given capacity in number of entries.
func New(capacity int) *Store {
	return &Store{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves a cached response. Returns nil if not found or hard-expired.
// Stale entries are returned with IsStale()==true so callers can revalidate.
func (s *Store) Get(key string) *CachedResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	el, ok := s.items[key]
	if !ok {
		return nil
	}
	e := el.Value.(*entry)
	now := time.Now()
	// Hard expiry: stale-while-revalidate window also expired.
	if !e.resp.IsStale(now) && e.resp.IsExpired(now) {
		s.lru.Remove(el)
		delete(s.items, key)
		return nil
	}
	s.lru.MoveToFront(el)
	return e.resp
}

// Set stores a response for key, evicting the LRU entry if at capacity.
func (s *Store) Set(key string, resp *CachedResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.items[key]; ok {
		s.lru.MoveToFront(el)
		el.Value.(*entry).resp = resp
		return
	}
	if s.lru.Len() >= s.capacity {
		oldest := s.lru.Back()
		if oldest != nil {
			s.lru.Remove(oldest)
			delete(s.items, oldest.Value.(*entry).key)
		}
	}
	el := s.lru.PushFront(&entry{key: key, resp: resp})
	s.items[key] = el
}

// Delete removes a key from the cache (used by purge).
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.items[key]; ok {
		s.lru.Remove(el)
		delete(s.items, key)
	}
}

// Purge removes all entries whose keys match the given prefix.
func (s *Store) Purge(prefix string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for key, el := range s.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			s.lru.Remove(el)
			delete(s.items, key)
			count++
		}
	}
	return count
}

// Stats returns hit/capacity information.
func (s *Store) Stats() (size, capacity int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lru.Len(), s.capacity
}

// ─────────────────────────────────────────────────────────────
// Cache-Control parsing helpers
// ─────────────────────────────────────────────────────────────

// ParseCacheControl parses Cache-Control header directives.
// Returns (maxAge, staleWhileRevalidate, noStore bool).
func ParseCacheControl(header string) (maxAge, swr time.Duration, noStore bool) {
	// TODO: implement proper Cache-Control parser per RFC 7234:
	//   tokenize on ", ", parse key=value pairs
	//   handle: max-age=N, s-maxage=N, stale-while-revalidate=N, no-store, no-cache, private
	_ = header
	return 0, 0, false
}
