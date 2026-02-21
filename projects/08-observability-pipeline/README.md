# Project 08 — OpenTelemetry-Compatible Observability Pipeline

> **Difficulty**: Senior · **Domain**: Observability, Protocol Buffers, Stream Processing
> **Real-world analog**: OpenTelemetry Collector, Grafana Alloy, Datadog Agent, Vector

---

## Why This Project Exists

The OpenTelemetry Collector (otelcol) is one of the most widely deployed pieces of
infrastructure in modern cloud systems. It collects traces, metrics, and logs; transforms
and enriches them; and exports to multiple backends. This project builds a minimal but
real OTel-compatible pipeline that handles all three signals.

---

## Folder Structure

```
08-observability-pipeline/
├── go.mod
├── main.go                             # Pipeline server (OTLP receiver + exporters)
├── receiver/
│   ├── otlp_grpc.go                    # OTLP/gRPC receiver (traces + metrics + logs)
│   ├── otlp_http.go                    # OTLP/HTTP receiver (JSON or protobuf)
│   ├── prometheus.go                   # Prometheus scrape receiver
│   └── filelog.go                      # Log file tail receiver
├── processor/
│   ├── batch.go                        # Batch processor: buffer + flush on size/time
│   ├── filter.go                       # Attribute-based filter (include/exclude)
│   ├── attributes.go                   # Attribute mutation: add, remove, rename, hash
│   ├── sampling/
│   │   ├── tail_sampler.go             # Tail-based trace sampling
│   │   └── probabilistic.go           # Probabilistic sampling
│   └── transform.go                    # OTTL-lite expression language
├── exporter/
│   ├── otlp_grpc.go                    # Forward to downstream collector / Jaeger / Tempo
│   ├── prometheus.go                   # Prometheus metrics exposition
│   ├── loki.go                         # Push logs to Loki
│   ├── stdout.go                       # Debug: pretty-print to stdout
│   └── file.go                         # NDJSON export to rotating files
├── pipeline/
│   ├── pipeline.go                     # Wires receivers → processors → exporters
│   └── fanout.go                       # Fan-out to multiple exporters
├── proto/
│   └── otlp/                           # Vendored OTLP proto (minimal subset)
└── config/
    └── config.yaml                     # Pipeline topology config
```

---

## Implementation Guide

### Phase 1 — OTLP Data Model (Week 1)

The **OpenTelemetry data model** has three signals. Implement Go structs matching the
OTLP spec (don't use generated code from protobuf — write the structs manually):

```go
// Trace signal
type Trace struct {
    ResourceSpans []ResourceSpans
}
type ResourceSpans struct {
    Resource   Attributes
    ScopeSpans []ScopeSpans
}
type Span struct {
    TraceID    [16]byte
    SpanID     [8]byte
    ParentSpanID [8]byte
    Name       string
    Kind       SpanKind
    StartTime  time.Time
    EndTime    time.Time
    Attributes Attributes
    Status     SpanStatus
    Events     []SpanEvent
}

// Metrics signal
type Metric struct {
    Name       string
    Description string
    Unit       string
    Data       any  // Gauge | Sum | Histogram | ExponentialHistogram | Summary
}

// Logs signal  
type LogRecord struct {
    Timestamp         time.Time
    ObservedTimestamp time.Time
    SeverityNumber    int32
    SeverityText      string
    Body              any
    Attributes        Attributes
    TraceID           [16]byte
    SpanID            [8]byte
}
```

---

### Phase 2 — OTLP HTTP Receiver (Week 1-2)

Implement the OTLP/HTTP receiver. OTLP supports both **JSON** and **Protobuf** encoding
(selected by `Content-Type` header):
- `application/json` → standard JSON encoding
- `application/x-protobuf` → protobuf encoding (use `google.golang.org/protobuf`)

Endpoints:
- `POST /v1/traces` — receive spans
- `POST /v1/metrics` — receive metrics
- `POST /v1/logs` — receive log records

Each request returns `200 OK` with a `PartialSuccess` response body.

---

### Phase 3 — Batch Processor (Week 2)

The batch processor buffers signals and flushes based on:
1. **Size**: flush when batch reaches N items (default 8192)
2. **Time**: flush every T duration (default 200ms)
3. **Memory**: flush if estimated memory > threshold (default 50MB)

```go
type BatchProcessor struct {
    maxSize     int
    maxWait     time.Duration
    buf         []Signal
    mu          sync.Mutex
    flushTimer  *time.Timer
    downstream  Processor
}
```

**Critical**: implement a **timeout on export** so a slow downstream doesn't block the buffer.

---

### Phase 4 — Tail-Based Sampling (Week 2-3)

Head-based sampling (sample 10% of all traces) loses interesting errors. **Tail-based sampling**
waits until a trace is complete, then decides whether to keep it based on the full trace.

```go
type TailSampler struct {
    rules []SamplingRule  // match conditions
    cache *traceCache      // pending incomplete traces, TTL evicted
    policy Policy          // AlwaysKeepError | PercentageByService | etc.
}
```

**Steps**:
1. Buffer all spans for a trace (keyed by TraceID) for up to `decisionWait` (30s by default)
2. When a root span arrives (SpanID == ParentSpanID is zero), evaluate rules
3. If sampled: flush all buffered spans to exporter
4. If not sampled: drop buffered spans

**Rules example**:
```yaml
sampling:
  decisionWait: 30s
  rules:
    - name: "keep-errors"
      type: status_code
      statusCode: ERROR
      samplingRate: 1.0
    - name: "sample-rest"
      type: probabilistic
      samplingRate: 0.1
```

---

### Phase 5 — Pipeline Config + Fan-out (Week 3)

```yaml
receivers:
  otlp_http:
    endpoint: ":4318"
  filelog:
    path: "/var/log/app.log"
    format: logfmt

processors:
  batch:
    maxSize: 1000
    maxWait: 200ms
  attributes:
    actions:
      - key: "db.password"
        action: delete
      - key: "user.email"
        action: hash
        algorithm: sha256

exporters:
  prometheus:
    endpoint: ":9090"
  loki:
    endpoint: "http://loki:3100"
  stdout:
    enabled: true

pipelines:
  traces:
    receivers: [otlp_http]
    processors: [batch, attributes]
    exporters: [stdout]
  logs:
    receivers: [filelog]
    processors: [batch]
    exporters: [loki, stdout]
```

---

## Acceptance Criteria

- [ ] Receives 50,000 spans/sec sustained with < 10ms p99 receiver latency
- [ ] Tail sampler correctly keeps 100% of error traces
- [ ] Batch processor flushes within 2x `maxWait` of last item
- [ ] Attribute hashing is deterministic (same input → same hash)

---

## Stretch Goals

- Implement **OTTL** (OpenTelemetry Transformation Language) mini-interpreter for complex transforms
- Add **service graph** generation: derive service-to-service topology graph from trace spans
- Build a **TUI control plane**: live view of pipeline throughput, error rates, backlog
