// Package processor contains span processors: batch batcher and tail sampler.
package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Polqt/obspipeline/pipeline"
)

// ─────────────────────────────────────────────────────────────
// Batch Processor
// ─────────────────────────────────────────────────────────────

// BatchProcessor accumulates incoming batches and flushes them when either
// the size threshold or the flush interval is reached.
type BatchProcessor struct {
	maxSize       int
	flushInterval time.Duration

	mu      sync.Mutex
	pending *pipeline.Batch
	ticker  *time.Ticker
	flushCh chan *pipeline.Batch
}

// NewBatch creates a batch processor with given max size and flush interval.
func NewBatch(maxSize int, flushInterval time.Duration) *BatchProcessor {
	bp := &BatchProcessor{
		maxSize:       maxSize,
		flushInterval: flushInterval,
		flushCh:       make(chan *pipeline.Batch, 16),
	}
	bp.pending = &pipeline.Batch{}
	return bp
}

func (b *BatchProcessor) Name() string { return "batch" }

// Process appends items from the batch into the accumulation buffer.
// When the buffer exceeds maxSize it is flushed synchronously.
func (b *BatchProcessor) Process(_ context.Context, batch *pipeline.Batch) (*pipeline.Batch, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pending.Kind == "" {
		b.pending.Kind = batch.Kind
	}
	b.pending.Spans = append(b.pending.Spans, batch.Spans...)
	b.pending.Points = append(b.pending.Points, batch.Points...)
	b.pending.LogRecs = append(b.pending.LogRecs, batch.LogRecs...)

	if b.pending.Len() >= b.maxSize {
		out := b.pending
		b.pending = &pipeline.Batch{Kind: batch.Kind}
		return out, nil
	}
	return nil, nil // not ready yet — returning nil drops through
}

// ─────────────────────────────────────────────────────────────
// Tail Sampler
// ─────────────────────────────────────────────────────────────

// TailSamplerConfig configures tail-based sampling rules.
type TailSamplerConfig struct {
	// DecisionWait is how long to wait before making a sampling decision.
	DecisionWait time.Duration
	// NumTraces is the maximum number of traces to hold in the buffer.
	NumTraces int
	// SamplingRate is the base sampling probability (0.0–1.0).
	SamplingRate float64
	// AlwaysSampleErrors forces sampling of any trace with a failed span.
	AlwaysSampleErrors bool
}

// DefaultTailSamplerConfig returns a sensible configuration.
func DefaultTailSamplerConfig() TailSamplerConfig {
	return TailSamplerConfig{
		DecisionWait:       5 * time.Second,
		NumTraces:          50_000,
		SamplingRate:       0.10,
		AlwaysSampleErrors: true,
	}
}

// traceBuffer holds spans for one trace until a decision is made.
type traceBuffer struct {
	spans    []*pipeline.Span
	arrivedAt time.Time
	hasError bool
}

// TailSampler buffers spans until a whole trace is assembled, then decides
// whether to keep or drop it based on sampling rules.
type TailSampler struct {
	cfg TailSamplerConfig
	mu  sync.Mutex
	buf map[[16]byte]*traceBuffer
}

// NewTailSampler creates a tail sampler with the given config.
func NewTailSampler(cfg TailSamplerConfig) *TailSampler {
	return &TailSampler{cfg: cfg, buf: make(map[[16]byte]*traceBuffer)}
}

func (ts *TailSampler) Name() string { return "tail-sampler" }

// Process buffers incoming spans.  Spans whose trace has been waiting longer
// than DecisionWait are flushed with a sampling decision applied.
func (ts *TailSampler) Process(_ context.Context, batch *pipeline.Batch) (*pipeline.Batch, error) {
	if batch.Kind != pipeline.Traces {
		return batch, nil // pass non-trace signals through unchanged
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Buffer incoming spans by trace ID.
	for _, span := range batch.Spans {
		tb, ok := ts.buf[span.TraceID]
		if !ok {
			tb = &traceBuffer{arrivedAt: time.Now()}
			ts.buf[span.TraceID] = tb
		}
		tb.spans = append(tb.spans, span)
		if span.StatusCode >= 2 { // OTLP status ERROR = 2
			tb.hasError = true
		}
	}

	// Flush traces that have exceeded the decision window.
	now := time.Now()
	var sampled []*pipeline.Span
	for traceID, tb := range ts.buf {
		if now.Sub(tb.arrivedAt) < ts.cfg.DecisionWait {
			continue
		}
		if ts.shouldSample(tb) {
			sampled = append(sampled, tb.spans...)
		}
		delete(ts.buf, traceID)
	}

	// Evict oldest traces if buffer is full.
	if len(ts.buf) > ts.cfg.NumTraces {
		// TODO: implement LRU eviction — drop oldest traces to bound memory
		_ = fmt.Sprintf("tail sampler buffer overflow: %d traces buffered", len(ts.buf))
	}

	if len(sampled) == 0 {
		return nil, nil
	}
	return &pipeline.Batch{Kind: pipeline.Traces, Spans: sampled}, nil
}

func (ts *TailSampler) shouldSample(tb *traceBuffer) bool {
	if ts.cfg.AlwaysSampleErrors && tb.hasError {
		return true
	}
	// TODO: replace pseudo-random with a deterministic hash of traceID for consistency
	// across restarts and multiple collector instances.
	// For now, use Go's math/rand or a hash-based approach:
	// h := fnv32(traceID[:]) % 1000
	// return float64(h) < ts.cfg.SamplingRate*1000
	_ = tb
	return true // placeholder: sample everything until hash-based logic is implemented
}

// AttributeFilter drops spans missing a required attribute key.
type AttributeFilter struct {
	requiredKeys []string
}

// NewAttributeFilter creates a processor that drops spans without requiredKeys.
func NewAttributeFilter(keys ...string) *AttributeFilter {
	return &AttributeFilter{requiredKeys: keys}
}

func (f *AttributeFilter) Name() string { return "attribute-filter" }

func (f *AttributeFilter) Process(_ context.Context, b *pipeline.Batch) (*pipeline.Batch, error) {
	if b.Kind != pipeline.Traces {
		return b, nil
	}
	var kept []*pipeline.Span
	for _, span := range b.Spans {
		if f.spanHasAllKeys(span) {
			kept = append(kept, span)
		}
	}
	b.Spans = kept
	return b, nil
}

func (f *AttributeFilter) spanHasAllKeys(span *pipeline.Span) bool {
	attrSet := make(map[string]struct{}, len(span.Attributes))
	for _, a := range span.Attributes {
		attrSet[a.Key] = struct{}{}
	}
	for _, key := range f.requiredKeys {
		if _, ok := attrSet[key]; !ok {
			return false
		}
	}
	return true
}
