// Package metrics collects proxy statistics exposed on the admin endpoint.
package metrics

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────
// Counter
// ─────────────────────────────────────────────────────────────

// Counter is a monotonically increasing int64 counter.
type Counter struct{ v atomic.Int64 }

func (c *Counter) Inc()         { c.v.Add(1) }
func (c *Counter) Add(n int64)  { c.v.Add(n) }
func (c *Counter) Value() int64 { return c.v.Load() }

// ─────────────────────────────────────────────────────────────
// Histogram (for latency distribution)
// ─────────────────────────────────────────────────────────────

// Histogram collects duration samples and computes p50/p90/p99/max.
type Histogram struct {
	mu      sync.Mutex
	samples []float64 // milliseconds
}

func (h *Histogram) Record(d time.Duration) {
	h.mu.Lock()
	h.samples = append(h.samples, float64(d.Microseconds()))
	h.mu.Unlock()
}

// Percentile returns the p-th percentile (0–100) of collected samples.
func (h *Histogram) Percentile(p float64) float64 {
	h.mu.Lock()
	samples := make([]float64, len(h.samples))
	copy(samples, h.samples)
	h.mu.Unlock()
	if len(samples) == 0 {
		return 0
	}
	sort.Float64s(samples)
	idx := (p / 100) * float64(len(samples)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return samples[lo]
	}
	return samples[lo] + (samples[hi]-samples[lo])*(idx-float64(lo))
}

// ─────────────────────────────────────────────────────────────
// Collector
// ─────────────────────────────────────────────────────────────

// Collector holds all named counters and histograms for the proxy.
type Collector struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	histos   map[string]*Histogram
}

// NewCollector creates an empty collector.
func NewCollector() *Collector {
	return &Collector{
		counters: make(map[string]*Counter),
		histos:   make(map[string]*Histogram),
	}
}

// Record increments a named counter by n.
func (c *Collector) Record(name string, n int64) {
	c.mu.Lock()
	if _, ok := c.counters[name]; !ok {
		c.counters[name] = &Counter{}
	}
	c.counters[name].Add(n)
	c.mu.Unlock()
}

// RecordLatency records a duration sample in the named histogram.
func (c *Collector) RecordLatency(name string, d time.Duration) {
	c.mu.Lock()
	if _, ok := c.histos[name]; !ok {
		c.histos[name] = &Histogram{}
	}
	h := c.histos[name]
	c.mu.Unlock()
	h.Record(d)
}

// Handler exposes all metrics as plain text (Prometheus-compatible).
func (c *Collector) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		for name, ctr := range c.counters {
			fmt.Fprintf(w, "# COUNTER %s %d\n", name, ctr.Value())
		}
		for name, h := range c.histos {
			fmt.Fprintf(w, "# HISTOGRAM %s p50=%.0fµs p90=%.0fµs p99=%.0fµs\n",
				name, h.Percentile(50), h.Percentile(90), h.Percentile(99))
		}
	}
}
