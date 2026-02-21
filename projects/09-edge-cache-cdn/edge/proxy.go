// Package edge implements the caching reverse proxy.
package edge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Polqt/edgecache/edge/cache"
)

// ─────────────────────────────────────────────────────────────
// Request coalescing (collapse)
// ─────────────────────────────────────────────────────────────

// inflightCall represents an in-progress origin fetch.
type inflightCall struct {
	done chan struct{}
	resp *cache.CachedResponse
	err  error
}

// coalescer ensures only one in-flight origin request per cache key.
type coalescer struct {
	mu       sync.Mutex
	inflight map[string]*inflightCall
}

func newCoalescer() *coalescer {
	return &coalescer{inflight: make(map[string]*inflightCall)}
}

// do fetches fn() for key, coalescing concurrent callers.
// All waiters receive the same result.
func (c *coalescer) do(key string, fn func() (*cache.CachedResponse, error)) (*cache.CachedResponse, error) {
	c.mu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		<-call.done
		return call.resp, call.err
	}
	call := &inflightCall{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	call.resp, call.err = fn()
	close(call.done)

	c.mu.Lock()
	delete(c.inflight, key)
	c.mu.Unlock()

	return call.resp, call.err
}

// ─────────────────────────────────────────────────────────────
// Proxy
// ─────────────────────────────────────────────────────────────

// Config holds edge proxy configuration.
type Config struct {
	OriginURL        string        // upstream origin URL, e.g. http://api.example.com
	ListenAddr       string        // e.g. :8080
	AdminAddr        string        // e.g. :9000 (purge + stats API)
	CacheCapacity    int           // number of cached entries
	DefaultTTL       time.Duration // TTL when Cache-Control is absent
	StaleGracePeriod time.Duration // stale-while-revalidate grace window override
}

// DefaultConfig returns a development-friendly config.
func DefaultConfig() Config {
	return Config{
		OriginURL:        "http://localhost:8081",
		ListenAddr:       ":8080",
		AdminAddr:        ":9000",
		CacheCapacity:    10_000,
		DefaultTTL:       60 * time.Second,
		StaleGracePeriod: 10 * time.Second,
	}
}

// Proxy is the caching edge proxy.
type Proxy struct {
	cfg      Config
	store    *cache.Store
	origin   *url.URL
	coalesce *coalescer
	client   *http.Client

	// Stats.
	hits   int64
	misses int64
	stales int64
	mu     sync.Mutex
}

// NewProxy creates a Proxy with the given config.
func NewProxy(cfg Config) (*Proxy, error) {
	origin, err := url.Parse(cfg.OriginURL)
	if err != nil {
		return nil, fmt.Errorf("parse origin URL: %w", err)
	}
	return &Proxy{
		cfg:      cfg,
		store:    cache.New(cfg.CacheCapacity),
		origin:   origin,
		coalesce: newCoalescer(),
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// ServeHTTP handles an incoming HTTP request.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isCacheable(r) {
		// Pass-through non-GET/HEAD unconditionally.
		p.pass(w, r)
		return
	}

	key := cache.Key(r)
	now := time.Now()

	if resp := p.store.Get(key); resp != nil {
		if !resp.IsExpired(now) {
			p.serve(w, resp, "HIT")
			p.mu.Lock(); p.hits++; p.mu.Unlock()
			return
		}
		if resp.IsStale(now) {
			// Serve stale, trigger background revalidation.
			p.serve(w, resp, "STALE")
			p.mu.Lock(); p.stales++; p.mu.Unlock()
			go func() {
				cr, err := p.fetchOrigin(context.Background(), r)
				if err == nil {
					p.store.Set(key, cr)
				}
			}()
			return
		}
	}

	// Cache MISS — fetch from origin, coalescing concurrent requests.
	cr, err := p.coalesce.do(key, func() (*cache.CachedResponse, error) {
		return p.fetchOrigin(r.Context(), r)
	})
	if err != nil {
		http.Error(w, "origin error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if isCacheableStatus(cr.StatusCode) {
		p.store.Set(key, cr)
	}
	p.serve(w, cr, "MISS")
	p.mu.Lock(); p.misses++; p.mu.Unlock()
}

// serve writes a cached response to the client.
func (p *Proxy) serve(w http.ResponseWriter, resp *cache.CachedResponse, xCache string) {
	for key, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}
	w.Header().Set("X-Cache", xCache)
	w.Header().Set("Age", strconv.Itoa(int(resp.Age(time.Now()).Seconds())))
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)
}

// pass proxies a request to the origin without caching.
func (p *Proxy) pass(w http.ResponseWriter, r *http.Request) {
	cr, err := p.fetchOrigin(r.Context(), r)
	if err != nil {
		http.Error(w, "origin error: "+err.Error(), http.StatusBadGateway)
		return
	}
	p.serve(w, cr, "BYPASS")
}

// fetchOrigin fetches the request from the origin server.
func (p *Proxy) fetchOrigin(ctx context.Context, r *http.Request) (*cache.CachedResponse, error) {
	// Build origin request.
	target := *p.origin
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(ctx, r.Method, target.String(), r.Body)
	if err != nil {
		return nil, err
	}
	// Forward relevant headers.
	for _, h := range []string{"Accept", "Accept-Encoding", "Authorization", "Cookie"} {
		if v := r.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	req.Header.Set("X-Forwarded-For", strings.Split(r.RemoteAddr, ":")[0])

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("origin fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // cap at 10 MB
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	now := time.Now()
	maxAge, swr, noStore := cache.ParseCacheControl(resp.Header.Get("Cache-Control"))
	if noStore {
		maxAge = -1 // do not cache
	}
	if maxAge == 0 {
		maxAge = p.cfg.DefaultTTL
	}
	if swr == 0 {
		swr = p.cfg.StaleGracePeriod
	}

	return &cache.CachedResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       bytes.Clone(body),
		CachedAt:   now,
		Expires:    now.Add(maxAge),
		StaleUntil: now.Add(maxAge + swr),
		ETag:       resp.Header.Get("ETag"),
		LastMod:    resp.Header.Get("Last-Modified"),
	}, nil
}

// isCacheable returns true for GET/HEAD requests.
func isCacheable(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Method == http.MethodHead
}

// isCacheableStatus returns true for 200, 301, 404 (common cacheable statuses).
func isCacheableStatus(code int) bool {
	switch code {
	case 200, 203, 204, 206, 300, 301, 404, 410:
		return true
	}
	return false
}

// Stats returns hit/miss/stale counters.
func (p *Proxy) Stats() (hits, misses, stales int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hits, p.misses, p.stales
}

// StoreStats delegates to the underlying cache store.
func (p *Proxy) StoreStats() (size, capacity int) {
	return p.store.Stats()
}

// Purge removes all entries with the given URL prefix.
func (p *Proxy) Purge(prefix string) int {
	slog.Info("cache purge", "prefix", prefix)
	return p.store.Purge(prefix)
}
