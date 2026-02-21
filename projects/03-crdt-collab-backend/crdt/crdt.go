// Package crdt provides conflict-free replicated data types.
package crdt

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────
// Vector Clock
// ─────────────────────────────────────────────────────────────

// VClock is a vector clock for causality tracking.
// Maps nodeID → logical clock counter.
type VClock map[string]uint64

// Increment returns a new VClock with nodeID's counter incremented.
func (v VClock) Increment(nodeID string) VClock {
	next := v.Clone()
	next[nodeID]++
	return next
}

// HappensBefore returns true if v causally precedes other.
func (v VClock) HappensBefore(other VClock) bool {
	// TODO: for all nodes: v[n] <= other[n], and at least one is strictly less
	panic("VClock.HappensBefore: not yet implemented")
}

// Concurrent returns true if neither v nor other causally precedes the other.
func (v VClock) Concurrent(other VClock) bool {
	return !v.HappensBefore(other) && !other.HappensBefore(v)
}

// Merge returns the component-wise maximum of v and other.
func (v VClock) Merge(other VClock) VClock {
	// TODO: for each nodeID in v and other, take max(v[n], other[n])
	panic("VClock.Merge: not yet implemented")
}

// Clone returns a deep copy.
func (v VClock) Clone() VClock {
	c := make(VClock, len(v))
	for k, val := range v {
		c[k] = val
	}
	return c
}

// ─────────────────────────────────────────────────────────────
// LWW Register
// ─────────────────────────────────────────────────────────────

// LWWRegister is a Last-Write-Wins register.
// On a timestamp tie, the higher NodeID wins (lexicographic).
type LWWRegister[T any] struct {
	mu        sync.RWMutex
	value     T
	timestamp time.Time
	nodeID    string
}

// Set updates the register if ts > current timestamp (or tie-break on nodeID).
func (r *LWWRegister[T]) Set(val T, ts time.Time, nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: update r.value, r.timestamp, r.nodeID if ts > r.timestamp
	// tie-break: if ts == r.timestamp, keep higher nodeID (string compare)
	panic("LWWRegister.Set: not yet implemented")
}

// Get returns the current value and its timestamp.
func (r *LWWRegister[T]) Get() (T, time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value, r.timestamp
}

// Merge pulls in a remote register's state.
func (r *LWWRegister[T]) Merge(other *LWWRegister[T]) {
	r.Set(other.value, other.timestamp, other.nodeID)
}

// ─────────────────────────────────────────────────────────────
// PN Counter
// ─────────────────────────────────────────────────────────────

// PNCounter is a Positive-Negative counter CRDT.
// Supports both increment and decrement without conflicts.
type PNCounter struct {
	mu       sync.RWMutex
	positive map[string]int64 // nodeID → positive increments
	negative map[string]int64 // nodeID → negative decrements
}

// NewPNCounter creates a zeroed PN counter.
func NewPNCounter() *PNCounter {
	return &PNCounter{
		positive: make(map[string]int64),
		negative: make(map[string]int64),
	}
}

// Increment adds delta to this node's positive counter.
func (c *PNCounter) Increment(nodeID string, delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.positive[nodeID] += delta
}

// Decrement adds delta to this node's negative counter.
func (c *PNCounter) Decrement(nodeID string, delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.negative[nodeID] += delta
}

// Value returns the current counter value (sum of positives - sum of negatives).
func (c *PNCounter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// TODO: sum all positive values, subtract sum of all negative values
	panic("PNCounter.Value: not yet implemented")
}

// Merge merges another counter into this one (take max per component).
func (c *PNCounter) Merge(other *PNCounter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	// TODO: for each nodeID in other.positive: c.positive[n] = max(c.positive[n], other.positive[n])
	// Same for negative
	panic("PNCounter.Merge: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// OR-Set
// ─────────────────────────────────────────────────────────────

// ORSet is an Observed-Remove Set CRDT.
// Add wins over concurrent Remove because removes only target specific add-tags.
type ORSet struct {
	mu       sync.RWMutex
	elements map[string]map[string]struct{} // value → set of add-tags (UUIDs)
}

// NewORSet creates an empty OR-Set.
func NewORSet() *ORSet {
	return &ORSet{elements: make(map[string]map[string]struct{})}
}

// Add adds value to the set with a unique tag derived from nodeID + timestamp.
// Returns the tag (so callers can gossip it to peers).
func (s *ORSet) Add(value, nodeID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// TODO: generate unique tag = nodeID + ":" + time.Now().Nanoseconds()
	// Add tag to s.elements[value]
	panic("ORSet.Add: not yet implemented")
}

// Remove removes all current tags for value. Concurrent adds are unaffected.
func (s *ORSet) Remove(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.elements, value)
}

// Contains returns true if value has at least one active add-tag.
func (s *ORSet) Contains(value string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tags, ok := s.elements[value]
	return ok && len(tags) > 0
}

// Values returns a sorted list of all values in the set.
func (s *ORSet) Values() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []string
	for v, tags := range s.elements {
		if len(tags) > 0 {
			result = append(result, v)
		}
	}
	return result
}

// Merge merges another OR-Set's elements in (union of add-tags).
func (s *ORSet) Merge(other *ORSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	// TODO: for each (value, tags) in other.elements:
	//   union tags into s.elements[value]
	panic("ORSet.Merge: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// RGA (Replicated Growable Array) — for Text
// ─────────────────────────────────────────────────────────────

// RGANodeID uniquely identifies an RGA node globally.
type RGANodeID struct {
	Seq    uint64 // per-node sequence number
	NodeID string // originating node
}

// RGANode is one character in the RGA linked array.
type RGANode struct {
	ID          RGANodeID
	InsertAfter RGANodeID // nil-equivalent: RGANodeID{}
	Char        rune
	Deleted     bool // tombstone
}

// RGA is a Replicated Growable Array for collaborative text editing.
type RGA struct {
	mu    sync.RWMutex
	nodes []RGANode         // sorted by position (invariant)
	index map[RGANodeID]int // ID → index in nodes slice
	seqNo uint64            // local sequence counter
}

// NewRGA creates an empty RGA.
func NewRGA() *RGA {
	return &RGA{index: make(map[RGANodeID]int)}
}

// Insert inserts a character after the node with afterID.
// Use zero-value RGANodeID{} to insert at beginning.
func (r *RGA) Insert(afterID RGANodeID, char rune, nodeID string) RGANode {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: increment r.seqNo, create RGANode{ID:{Seq, nodeID}, InsertAfter:afterID, Char:char}
	// Find position of afterID in r.nodes, insert new node after it
	// Handle concurrent inserts at same position: sort by (Seq desc, NodeID asc) for total order
	panic("RGA.Insert: not yet implemented")
}

// Delete marks the node with id as deleted (tombstone).
func (r *RGA) Delete(id RGANodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: find id in r.index, set r.nodes[idx].Deleted = true
	panic("RGA.Delete: not yet implemented")
}

// Text returns the current document text (ignores tombstones).
func (r *RGA) Text() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// TODO: walk r.nodes in order, append non-deleted chars
	panic("RGA.Text: not yet implemented")
}

// Apply applies a remote operation (insert or delete).
func (r *RGA) Apply(op RGANode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: if op.Deleted → tombstone; else → insert at correct position
	// Insert position: find afterID, then find correct spot considering concurrent inserts
	_ = op
	return fmt.Errorf("RGA.Apply: not yet implemented")
}
