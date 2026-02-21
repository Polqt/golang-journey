# Project 04 — Service Mesh Sidecar Proxy

> **Difficulty**: Expert · **Domain**: Networking, TLS, Observability, L7 Proxy
> **Real-world analog**: Envoy Proxy, Linkerd2 proxy, Istio, Consul Connect

---

## Why This Project Exists

Kubernetes service meshes handle mTLS, retries, circuit breaking, and observability for
microservices. The heart of every mesh is a **sidecar proxy** — a L7 proxy that intercepts all
inbound/outbound traffic. This project builds a minimal but real sidecar proxy that handles
mTLS, load balancing, circuit breaking, and Prometheus metrics.

---

## Folder Structure

```
04-service-mesh-proxy/
├── go.mod
├── main.go                          # CLI: proxy start, keygen, config-validate
├── proxy/
│   ├── listener.go                  # TCP/HTTP listener setup
│   ├── router.go                    # Request routing + upstream selection
│   ├── upstream.go                  # Upstream connection pool
│   ├── circuit_breaker.go           # Per-upstream circuit breaker
│   ├── retry.go                     # Retry policy with backoff + budget
│   ├── balancer/
│   │   ├── roundrobin.go            # Weighted round-robin
│   │   ├── leastconn.go             # Least connections
│   │   └── random.go
│   └── middleware/
│       ├── ratelimit.go             # Per-client token bucket
│       ├── timeout.go               # Request timeout wrapper
│       └── tracing.go               # W3C Trace Context propagation
├── mtls/
│   ├── ca.go                        # Self-signed CA + cert issuance
│   ├── certstore.go                 # In-memory cert cache + rotation
│   └── verify.go                    # SPIFFE SVID verification
├── metrics/
│   ├── collector.go                 # Prometheus-compatible metrics
│   └── server.go                    # /metrics HTTP endpoint
└── config/
    └── config.go                    # YAML config loader
```

---

## Implementation Guide

### Phase 1 — TCP Reverse Proxy (Week 1)

Start with a simple TCP reverse proxy before adding HTTP awareness.

```go
type Proxy struct {
    Listen string   // e.g. ":8080"
    Upstreams []string // e.g. ["localhost:9001", "localhost:9002"]
}
```

**Steps**:
1. `net.Listen("tcp", addr)` → accept loop
2. For each conn: select upstream using round-robin, dial upstream, `io.Copy` bidirectionally
3. Track bytes transferred and connection count
4. Add graceful shutdown: drain in-flight connections on SIGTERM

**Key technique**: bidirectional copy needs two goroutines. Use `io.Copy` in each direction
and close both sides when either returns.

---

### Phase 2 — HTTP/1.1 + HTTP/2 Proxy (Week 1-2)

Upgrade to HTTP-aware proxying using `net/http/httputil.ReverseProxy`.
But implement the core yourself rather than using the stdlib helper:

1. Parse incoming HTTP request
2. Rewrite `Host` header, add `X-Forwarded-For`
3. Forward to upstream, stream response back
4. Handle `Connection: Upgrade` (WebSocket passthrough)

For **HTTP/2**: use `golang.org/x/net/http2` (allowed as a stdlib extension).

---

### Phase 3 — mTLS with Self-Signed PKI (Week 2)

Build a minimal CA that issues short-lived SPIFFE SVIDs:

```go
// Generate root CA
ca, _ := mtls.NewCA("mesh.local")

// Issue a service cert for "payments.svc.mesh.local"
cert, _ := ca.IssueSVID("spiffe://mesh.local/payments", 24*time.Hour)
```

**SPIFFE URI SANs** allow services to authenticate each other by workload identity, not IP.

**Steps**:
1. Use `crypto/x509`, `crypto/ecdsa`, `crypto/rand` — no external PKI library
2. Generate ECDSA P-256 keys (smaller and faster than RSA)
3. Issue certs with `URIs: [spiffe://mesh.local/serviceA]` in SAN
4. Create `tls.Config` with `ClientAuth: tls.RequireAndVerifyClientCert`
5. Implement cert rotation: re-issue certs 1 hour before expiry

---

### Phase 4 — Load Balancing + Circuit Breaking (Week 3)

Implement three balancing strategies and wire in the Challenge 05 circuit breaker:

```go
type Balancer interface {
    Next() (Upstream, error)
    Report(upstream Upstream, err error, latencyMs int64)
}
```

- `RoundRobin`: iterate through upstreams in order, skip OPEN circuit breakers
- `LeastConn`: track active connections per upstream, pick minimum
- Configure via YAML: `balancer: least_conn`

---

### Phase 5 — Observability (Week 3-4)

Expose Prometheus metrics at `/metrics`:

```
mesh_requests_total{service="payments", status="200"} 15234
mesh_request_duration_seconds{service="payments", quantile="0.99"} 0.045
mesh_upstream_active_connections{upstream="payments:9001"} 12
mesh_circuit_breaker_state{upstream="payments:9001"} 0  # 0=closed, 1=open
```

Implement a **W3C Trace Context** (`traceparent` header) propagation:
- Extract `traceparent` from incoming requests
- Generate new span ID, keep trace ID
- Inject into upstream request

---

## Config Format

```yaml
proxy:
  listen: ":8080"
  upstreams:
    - addr: "localhost:9001"
      weight: 2
    - addr: "localhost:9002"
      weight: 1
  balancer: least_conn
  retry:
    attempts: 3
    perTryTimeout: 500ms
  circuitBreaker:
    windowSize: 20
    failureThreshold: 0.5
    resetTimeout: 10s
mtls:
  enabled: true
  ca: "/etc/mesh/ca.crt"
  cert: "/etc/mesh/svc.crt"
  key: "/etc/mesh/svc.key"
metrics:
  listen: ":9090"
```

---

## Acceptance Criteria

- [ ] mTLS handshake completes in < 5ms (ECDSA P-256)
- [ ] Load balancing distributes requests within 5% of target ratio
- [ ] Circuit breaker trips after 50% failure rate over 20-request window
- [ ] `10,000 req/sec` sustained with p99 < 2ms added latency (proxy overhead)
- [ ] Prometheus metrics update in real time

---

## Stretch Goals

- Implement **gRPC proxying** (HTTP/2 streaming)
- Add **request hedging**: send duplicate request to second upstream if first takes > p95 latency
- Build a **control plane** API: dynamically add/remove upstreams without restart
