// Package pipeline wires receivers → processors → exporters into a running pipeline.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// ─────────────────────────────────────────────────────────────
// Signal types
// ─────────────────────────────────────────────────────────────

// SignalKind identifies what kind of telemetry a component handles.
type SignalKind string

const (
	Traces  SignalKind = "traces"
	Metrics SignalKind = "metrics"
	Logs    SignalKind = "logs"
)

// ─────────────────────────────────────────────────────────────
// Span (Trace signal unit)
// ─────────────────────────────────────────────────────────────

// SpanKind mirrors the OTLP span kind enum.
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// Attribute is a key-value pair.
type Attribute struct {
	Key   string
	Value any
}

// Span represents a single OTLP trace span.
type Span struct {
	TraceID    [16]byte
	SpanID     [8]byte
	ParentID   [8]byte
	Name       string
	Kind       SpanKind
	StartNs    int64  // nanoseconds since Unix epoch
	EndNs      int64
	Attributes []Attribute
	StatusCode int
	StatusMsg  string
	Events     []SpanEvent
}

// SpanEvent is a timestamped annotation on a span.
type SpanEvent struct {
	TimeNs     int64
	Name       string
	Attributes []Attribute
}

// TraceID returns a hex string for display.
func (s *Span) TraceIDHex() string {
	return fmt.Sprintf("%x", s.TraceID[:])
}

// DurationNs returns the span duration in nanoseconds.
func (s *Span) DurationNs() int64 { return s.EndNs - s.StartNs }

// ─────────────────────────────────────────────────────────────
// DataPoint (Metrics signal unit)
// ─────────────────────────────────────────────────────────────

// DataPoint is one metric observation.
type DataPoint struct {
	Name       string
	Ts         int64
	Value      float64
	Labels     map[string]string
	MetricKind string // gauge, counter, histogram, summary
}

// ─────────────────────────────────────────────────────────────
// LogRecord (Logs signal unit)
// ─────────────────────────────────────────────────────────────

// LogRecord is one structured log line.
type LogRecord struct {
	Ts         int64
	Severity   int
	Body       string
	Attributes []Attribute
	TraceID    [16]byte
	SpanID     [8]byte
}

// ─────────────────────────────────────────────────────────────
// Batch — groups many spans/metrics/logs
// ─────────────────────────────────────────────────────────────

// Batch is a collection of telemetry items of one signal kind.
type Batch struct {
	Kind     SignalKind
	Spans    []*Span
	Points   []*DataPoint
	LogRecs  []*LogRecord
}

// Len returns the number of items in the batch.
func (b *Batch) Len() int {
	return len(b.Spans) + len(b.Points) + len(b.LogRecs)
}

// ─────────────────────────────────────────────────────────────
// Component interfaces
// ─────────────────────────────────────────────────────────────

// Receiver accepts telemetry from the outside world and emits Batches.
type Receiver interface {
	Start(ctx context.Context, out chan<- *Batch) error
	Stop() error
	Name() string
}

// Processor transforms or filters batches.
type Processor interface {
	Process(ctx context.Context, b *Batch) (*Batch, error)
	Name() string
}

// Exporter sends batches to a backend.
type Exporter interface {
	Export(ctx context.Context, b *Batch) error
	Name() string
}

// ─────────────────────────────────────────────────────────────
// Pipeline
// ─────────────────────────────────────────────────────────────

// Pipeline connects receivers → processors → exporters.
type Pipeline struct {
	receivers  []Receiver
	processors []Processor
	exporters  []Exporter

	ch   chan *Batch
	wg   sync.WaitGroup
}

// New creates a pipeline with the given buffer size.
func New(bufSize int) *Pipeline {
	return &Pipeline{ch: make(chan *Batch, bufSize)}
}

// AddReceiver appends a receiver.
func (p *Pipeline) AddReceiver(r Receiver) { p.receivers = append(p.receivers, r) }

// AddProcessor appends a processor (executed in order).
func (p *Pipeline) AddProcessor(pr Processor) { p.processors = append(p.processors, pr) }

// AddExporter appends an exporter (fan-out: all exporters receive each batch).
func (p *Pipeline) AddExporter(e Exporter) { p.exporters = append(p.exporters, e) }

// Start launches all receivers and the processing loop.
func (p *Pipeline) Start(ctx context.Context) error {
	for _, r := range p.receivers {
		slog.Info("starting receiver", "name", r.Name())
		if err := r.Start(ctx, p.ch); err != nil {
			return fmt.Errorf("receiver %s: %w", r.Name(), err)
		}
	}

	p.wg.Add(1)
	go p.run(ctx)
	return nil
}

// run is the main processing goroutine.
func (p *Pipeline) run(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-p.ch:
			if !ok {
				return
			}
			p.processBatch(ctx, batch)
		}
	}
}

func (p *Pipeline) processBatch(ctx context.Context, b *Batch) {
	var err error
	for _, proc := range p.processors {
		if b, err = proc.Process(ctx, b); err != nil {
			slog.Warn("processor error", "processor", proc.Name(), "err", err)
			return
		}
		if b == nil || b.Len() == 0 {
			return // dropped by processor
		}
	}
	for _, exp := range p.exporters {
		if err := exp.Export(ctx, b); err != nil {
			slog.Warn("exporter error", "exporter", exp.Name(), "err", err)
		}
	}
}

// Stop gracefully shuts down all receivers and waits for pending batches.
func (p *Pipeline) Stop() {
	for _, r := range p.receivers {
		if err := r.Stop(); err != nil {
			slog.Warn("receiver stop error", "name", r.Name(), "err", err)
		}
	}
	close(p.ch)
	p.wg.Wait()
}
