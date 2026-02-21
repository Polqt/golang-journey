package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ============================================================
// CHALLENGE 07: Backpressure-Aware Streaming Pipeline
// ============================================================
// Build a multi-stage concurrent pipeline with bounded buffers,
// fan-out, and per-stage observability.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// Item is the unit of data flowing through the pipeline.
type Item struct {
	ID      int64
	Payload string
	Meta    map[string]string
}

// ProcessFn is the user function for each stage.
// It receives one item and returns zero or more output items.
type ProcessFn func(ctx context.Context, item Item) ([]Item, error)

// StageOption is a functional option for stage configuration.
type StageOption func(*stageConfig)

// stageConfig holds per-stage options (internal).
type stageConfig struct {
	bufferSize int
	dropOnFull bool
}

// WithBufferSize sets the inter-stage channel buffer size.
func WithBufferSize(n int) StageOption {
	return func(c *stageConfig) { c.bufferSize = n }
}

// WithDropOnFull makes the stage drop items when its input buffer is full.
func WithDropOnFull() StageOption {
	return func(c *stageConfig) { c.dropOnFull = true }
}

// StageStats holds observable metrics for one pipeline stage.
type StageStats struct {
	Name         string
	Processed    int64
	Errors       int64
	Dropped      int64
	P50LatencyMs float64
	P99LatencyMs float64
}

// PipelineStats holds all stage stats.
type PipelineStats struct {
	Stages []StageStats
}

// TODO: Define stage struct
// TODO: Define Pipeline struct

// NewPipeline creates an empty pipeline.
func NewPipeline() *Pipeline {
	panic("implement me")
}

// AddStage registers a named stage with `workers` parallel workers.
func (p *Pipeline) AddStage(name string, workers int, fn ProcessFn, opts ...StageOption) {
	panic("implement me")
}

// Connect links the output of stage `from` to the input of stage `to`.
func (p *Pipeline) Connect(from, to string) {
	panic("implement me")
}

// Start launches all stage workers. Blocks until all workers are running.
func (p *Pipeline) Start(ctx context.Context) {
	panic("implement me")
}

// Push sends items into the first stage of the pipeline.
// Blocks if the first stage buffer is full (unless WithDropOnFull).
func (p *Pipeline) Push(items ...Item) {
	panic("implement me")
}

// Drain waits for all in-flight items to be processed and all workers to exit.
func (p *Pipeline) Drain() {
	panic("implement me")
}

// Stats returns per-stage metrics.
func (p *Pipeline) Stats() PipelineStats {
	panic("implement me")
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// Pipeline — stub; replace with your implementation.
type Pipeline struct {
	mu sync.Mutex
}

func main() {
	fmt.Println("=== Backpressure-Aware Streaming Pipeline ===")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := NewPipeline()

	// Stage 1: parse — adds metadata
	parseFn := func(ctx context.Context, item Item) ([]Item, error) {
		item.Meta = map[string]string{"parsed": "true"}
		time.Sleep(time.Duration(rand.Intn(2)) * time.Millisecond)
		return []Item{item}, nil
	}

	// Stage 2: enrich — slow consumer to test backpressure
	enrichFn := func(ctx context.Context, item Item) ([]Item, error) {
		time.Sleep(5 * time.Millisecond) // slower than parser
		item.Meta["enriched"] = "true"
		return []Item{item}, nil
	}

	// Stage 3: sink — records results
	var sinkMu sync.Mutex
	var sunk []Item
	sinkFn := func(ctx context.Context, item Item) ([]Item, error) {
		sinkMu.Lock()
		sunk = append(sunk, item)
		sinkMu.Unlock()
		return nil, nil
	}

	p.AddStage("parse", 4, parseFn, WithBufferSize(50))
	p.AddStage("enrich", 2, enrichFn, WithBufferSize(20))
	p.AddStage("sink", 1, sinkFn, WithBufferSize(10))
	p.Connect("parse", "enrich")
	p.Connect("enrich", "sink")
	p.Start(ctx)

	// Push 200 items
	start := time.Now()
	for i := 0; i < 200; i++ {
		p.Push(Item{ID: int64(i), Payload: fmt.Sprintf("payload-%d", i)})
	}
	p.Drain()
	elapsed := time.Since(start)

	sinkMu.Lock()
	sunkCount := len(sunk)
	sinkMu.Unlock()

	fmt.Printf("Processed %d/200 items in %v\n", sunkCount, elapsed.Round(time.Millisecond))

	stats := p.Stats()
	for _, s := range stats.Stages {
		fmt.Printf("Stage %-8s — processed=%d errors=%d dropped=%d p99=%.1fms\n",
			s.Name, s.Processed, s.Errors, s.Dropped, s.P99LatencyMs)
	}

	fmt.Println("Done.")
}
