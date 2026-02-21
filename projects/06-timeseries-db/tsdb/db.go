// Package tsdb is the top-level time-series database.
// It ties together the in-memory Head, on-disk Blocks, and WAL.
package tsdb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Polqt/tsdb/tsdb/head"
)

// ─────────────────────────────────────────────────────────────
// Labels
// ─────────────────────────────────────────────────────────────

// Labels is a sorted list of name=value pairs that uniquely identify a series.
type Labels []Label

// Label is a single name=value pair.
type Label struct {
	Name  string
	Value string
}

// String returns a Prometheus-style label set.
func (l Labels) String() string {
	s := "{"
	for i, lbl := range l {
		if i > 0 {
			s += ","
		}
		s += lbl.Name + "=" + lbl.Value
	}
	return s + "}"
}

// Hash returns a stable hash of the label set.
func (l Labels) Hash() uint64 {
	// FNV-1a
	var h uint64 = 14695981039346656037
	for _, lbl := range l {
		for _, c := range lbl.Name + "=" + lbl.Value + "," {
			h ^= uint64(c)
			h *= 1099511628211
		}
	}
	return h
}

// ─────────────────────────────────────────────────────────────
// Sample
// ─────────────────────────────────────────────────────────────

// Sample is a single (timestamp, value) measurement.
type Sample struct {
	Ts  int64 // Unix milliseconds
	Val float64
}

// ─────────────────────────────────────────────────────────────
// Series — one in-memory time series
// ─────────────────────────────────────────────────────────────

// Series holds all chunks for one label set.
type Series struct {
	Labels Labels
	mu     sync.RWMutex
	chunks []*head.Chunk
	head   *head.Chunk // current writable chunk
}

// NewSeries creates an empty series.
func NewSeries(labels Labels) *Series {
	c := head.NewChunk()
	return &Series{Labels: labels, head: c, chunks: []*head.Chunk{c}}
}

// Append adds a sample to the series.
func (s *Series) Append(ts int64, v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// TODO: if head chunk is full (> maxChunkBytes), rotate: close head, open new chunk
	s.head.Append(ts, v)
}

// Samples returns all samples in the time range [minTs, maxTs].
func (s *Series) Samples(minTs, maxTs int64) []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Sample
	for _, chunk := range s.chunks {
		it := chunk.NewIterator()
		for it.Next() {
			if it.CurTs >= minTs && it.CurTs <= maxTs {
				result = append(result, Sample{Ts: it.CurTs, Val: it.CurVal})
			}
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────
// DB — top-level entrypoint
// ─────────────────────────────────────────────────────────────

// Options configures a DB instance.
type Options struct {
	Dir             string        // data directory
	RetentionPeriod time.Duration // data older than this is dropped
	BlockDuration   time.Duration // how long each block covers
	WALSegmentSize  int           // WAL segment size in bytes
}

// DefaultOptions returns sane defaults.
func DefaultOptions(dir string) Options {
	return Options{
		Dir:             dir,
		RetentionPeriod: 15 * 24 * time.Hour,
		BlockDuration:   2 * time.Hour,
		WALSegmentSize:  128 << 20, // 128 MB
	}
}

// DB is the top-level time-series database.
type DB struct {
	opts Options
	mu   sync.RWMutex

	// In-memory series index: label hash → *Series.
	series map[uint64]*Series

	// TODO: add WAL, block manager, compactor
}

// Open opens or creates a database at opts.Dir.
func Open(opts Options) (*DB, error) {
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", opts.Dir, err)
	}
	db := &DB{opts: opts, series: make(map[uint64]*Series)}
	// TODO: replay WAL from opts.Dir/wal/
	return db, nil
}

// Close flushes in-memory data and closes all resources.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	// TODO: flush remaining head chunks to disk block, close WAL
	return nil
}

// Write appends a batch of samples keyed by their label set.
func (db *DB) Write(labels Labels, samples []Sample) error {
	if len(samples) == 0 {
		return nil
	}
	// Ensure samples are sorted by timestamp.
	sort.Slice(samples, func(i, j int) bool { return samples[i].Ts < samples[j].Ts })

	db.mu.Lock()
	h := labels.Hash()
	s, ok := db.series[h]
	if !ok {
		s = NewSeries(labels)
		db.series[h] = s
	}
	db.mu.Unlock()

	for _, sm := range samples {
		s.Append(sm.Ts, sm.Val)
	}
	return nil
}

// Query returns all samples matching selector in the time range.
// Returns pointers to avoid copying the mutex embedded in Series.
func (db *DB) Query(selector Labels, minTs, maxTs int64) ([]*Series, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// TODO: build an inverted index (label name+value → series refs) for efficient matching.
	// Linear scan as MVP:
	var results []*Series
	for _, s := range db.series {
		if matchesSelector(s.Labels, selector) {
			results = append(results, s)
		}
	}
	return results, nil
}

// matchesSelector returns true if labels satisfy all selector pairs.
func matchesSelector(labels, selector Labels) bool {
	for _, sel := range selector {
		found := false
		for _, lbl := range labels {
			if lbl.Name == sel.Name && lbl.Value == sel.Value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// DataDir returns the path to the named sub-directory.
func (db *DB) DataDir(sub string) string {
	return filepath.Join(db.opts.Dir, sub)
}
