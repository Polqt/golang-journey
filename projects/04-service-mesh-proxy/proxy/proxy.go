// Package proxy implements the core TCP/HTTP proxy with routing and circuit breaking.
package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Polqt/meshproxy/config"
	"github.com/Polqt/meshproxy/metrics"
)

// ─────────────────────────────────────────────────────────────
// Circuit Breaker
// ─────────────────────────────────────────────────────────────

// cbState represents the three states of the circuit breaker.
type cbState int32

const (
	cbClosed   cbState = iota // requests flow normally
	cbOpen                    // all requests fail fast
	cbHalfOpen                // limited probe requests allowed
)

// CircuitBreaker guards one upstream endpoint.
type CircuitBreaker struct {
	cfg      config.CBConfig
	state    atomic.Int32
	failures atomic.Int32
	lastTrip atomic.Int64 // UnixNano of last CLOSED→OPEN transition
	halfReqs atomic.Int32 // requests allowed in HALF-OPEN
}

// NewCircuitBreaker creates a CB with the given configuration.
func NewCircuitBreaker(cfg config.CBConfig) *CircuitBreaker {
	return &CircuitBreaker{cfg: cfg}
}

// Allow returns true if the request should be forwarded.
func (cb *CircuitBreaker) Allow() bool {
	switch cbState(cb.state.Load()) {
	case cbClosed:
		return true
	case cbOpen:
		// TODO: check if reset timeout has elapsed; if so transition to HALF-OPEN
		panic("CircuitBreaker.Allow open state: not yet implemented")
	case cbHalfOpen:
		// Allow at most cfg.HalfOpenMaxReqs concurrent probes
		return cb.halfReqs.Add(1) <= int32(cb.cfg.HalfOpenMaxReqs)
	}
	return false
}

// RecordSuccess records a successful response, resetting failures if HALF-OPEN.
func (cb *CircuitBreaker) RecordSuccess() {
	// TODO: if HALF-OPEN → transition back to CLOSED, reset counters
	cb.failures.Store(0)
}

// RecordFailure records a failed response, potentially tripping to OPEN.
func (cb *CircuitBreaker) RecordFailure() {
	f := cb.failures.Add(1)
	if int(f) >= cb.cfg.MaxFailures {
		// TODO: transition CLOSED → OPEN, set lastTrip = time.Now().UnixNano()
		panic("CircuitBreaker.RecordFailure trip: not yet implemented")
	}
}

// ─────────────────────────────────────────────────────────────
// Load Balancer (weighted round-robin)
// ─────────────────────────────────────────────────────────────

// upstream is a single backend with its circuit breaker.
type upstream struct {
	cfg     config.UpstreamCfg
	cb      *CircuitBreaker
	proxy   *httputil.ReverseProxy
	connSem chan struct{} // semaphore limiting concurrent connections
}

// LoadBalancer distributes requests across healthy upstreams.
type LoadBalancer struct {
	mu        sync.RWMutex
	upstreams []*upstream
	current   atomic.Int64 // round-robin index
	cbCfg     config.CBConfig
}

// NewLoadBalancer builds an LB from config.
func NewLoadBalancer(cfgs []config.UpstreamCfg, cbCfg config.CBConfig) *LoadBalancer {
	lb := &LoadBalancer{cbCfg: cbCfg}
	for _, c := range cfgs {
		u, _ := url.Parse("http://" + c.Addr)
		sem := make(chan struct{}, max(c.MaxConn, 100))
		// prefill semaphore
		for i := 0; i < cap(sem); i++ {
			sem <- struct{}{}
		}
		lb.upstreams = append(lb.upstreams, &upstream{
			cfg:     c,
			cb:      NewCircuitBreaker(cbCfg),
			proxy:   httputil.NewSingleHostReverseProxy(u),
			connSem: sem,
		})
	}
	return lb
}

// Next picks the next healthy upstream (weighted round-robin + CB check).
func (lb *LoadBalancer) Next() (*upstream, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	n := len(lb.upstreams)
	if n == 0 {
		return nil, fmt.Errorf("no upstreams configured")
	}
	for attempt := 0; attempt < n; attempt++ {
		idx := int(lb.current.Add(1)) % n
		u := lb.upstreams[idx]
		if u.cb.Allow() {
			return u, nil
		}
	}
	return nil, fmt.Errorf("all upstreams are circuit-broken")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────────────────────
// Server
// ─────────────────────────────────────────────────────────────

// Server is the sidecar proxy HTTP server.
type Server struct {
	cfg     *config.Config
	lb      *LoadBalancer
	metrics *metrics.Collector
	http    *http.Server
}

// NewServer creates a proxy server.
func NewServer(cfg *config.Config, col *metrics.Collector) *Server {
	lb := NewLoadBalancer(cfg.Upstreams, cfg.CircuitBreaker)
	s := &Server{cfg: cfg, lb: lb, metrics: col}
	s.http = &http.Server{
		Handler:      s,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}
	return s
}

// Serve accepts connections from the given listener.
func (s *Server) Serve(l net.Listener) error {
	return s.http.Serve(l)
}

// ServeHTTP proxies one HTTP request to a healthy upstream.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	u, err := s.lb.Next()
	if err != nil {
		http.Error(w, "service unavailable: "+err.Error(), http.StatusServiceUnavailable)
		s.metrics.Record("proxy.no_upstream", 1)
		return
	}

	// Acquire connection semaphore (non-blocking attempt).
	select {
	case <-u.connSem:
		defer func() { u.connSem <- struct{}{} }()
	default:
		http.Error(w, "upstream at capacity", http.StatusTooManyRequests)
		return
	}

	// TODO: mTLS wrapping of outbound connection
	// TODO: add retry loop up to cfg.RetryPolicy.MaxAttempts

	rw := &statusCapture{ResponseWriter: w}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.Timeout)
	defer cancel()

	u.proxy.ServeHTTP(rw, r.WithContext(ctx))

	if rw.status >= 500 {
		u.cb.RecordFailure()
		s.metrics.Record("proxy.upstream_error", 1)
	} else {
		u.cb.RecordSuccess()
	}

	latency := time.Since(start)
	slog.Info("proxied", "upstream", u.cfg.Name, "status", rw.status, "latency", latency)
	s.metrics.RecordLatency("proxy.latency", latency)
}

// statusCapture wraps ResponseWriter to capture the status code.
type statusCapture struct {
	http.ResponseWriter
	status int
}

func (s *statusCapture) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// ─────────────────────────────────────────────────────────────
// TCP Tunnel (for non-HTTP traffic)
// ─────────────────────────────────────────────────────────────

// TunnelTCP bidirectionally copies bytes between src and dst.
func TunnelTCP(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	done := make(chan struct{}, 2)
	go func() { io.Copy(dst, src); done <- struct{}{} }()
	go func() { io.Copy(src, dst); done <- struct{}{} }()
	<-done
}
