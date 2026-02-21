# Challenge 07 — Backpressure-Aware Streaming Pipeline

## Difficulty: Hard
## Category: Concurrency · Stream Processing · Systems Programming

---

## Problem Statement

Apache Kafka consumers, Flink, and Go's own `io.Pipe` all have to deal with **backpressure**:
what happens when the consumer is slower than the producer? The naive answer is "buffer
everything" — which leads to OOM and latency spikes.

The production answer is a **bounded pipeline with backpressure**: the producer blocks (or
applies rate limiting) when the downstream stage is full, and metrics reflect pipeline health.

Build a **multi-stage streaming pipeline** that:

1. Has **N configurable stages**, each running on a pool of worker goroutines
2. Uses **bounded channels** between stages (backpressure propagates upstream automatically)
3. Supports **fan-out** at any stage (broadcast to multiple downstream stages)
4. Allows each stage to define a `ProcessFn func(item Item) ([]Item, error)` — can expand or shrink
5. Tracks **stage latency**, **throughput**, **drop rate** (when using non-blocking sends)
6. Supports graceful shutdown via `context.Context`

---

## Requirements

```go
p := NewPipeline()
p.AddStage("parse",   4, parseFn,   WithBufferSize(100))
p.AddStage("enrich",  2, enrichFn,  WithBufferSize(50))
p.AddStage("sink",    1, sinkFn,    WithBufferSize(10))
p.Connect("parse", "enrich")
p.Connect("enrich", "sink")
p.Start(ctx)
p.Push(items...)   // blocks if first stage buffer is full
p.Stats()          // per-stage throughput/latency/errors
p.Drain()          // wait for all in-flight items to complete
```

---

## Constraints

- No third-party libraries
- Workers within a stage must run as goroutines
- A stage must not drop items unless `WithDropOnFull()` option is specified
- Shutdown must be graceful: flush all in-flight items before closing

---

## Acceptance Criteria

- [ ] No goroutine leaks after `Drain()` + context cancel
- [ ] Throughput scales linearly with worker count per stage up to CPU count
- [ ] Backpressure correctly slows the producer (not drops) without `WithDropOnFull`
- [ ] `Stats()` reports per-stage p50/p99 latency

---

## Stretch Goals

- Implement **circuit breaker integration**: if error rate > 20% on a stage, pause it
- Add a **reorder buffer**: out-of-order items are re-sequenced before the sink
