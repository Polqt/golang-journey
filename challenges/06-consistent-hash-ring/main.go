package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"sync"
)

// ============================================================
// CHALLENGE 06: Consistent Hash Ring with Virtual Nodes
// ============================================================
// Implement a consistent hash ring with weighted virtual nodes
// for balanced, fault-tolerant key distribution.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

const DefaultVNodeCount = 150

// TODO: Define vnode struct:
//   - pos uint32 (ring position)
//   - nodeID string

// TODO: Define HashRing struct:
//   - mu sync.RWMutex
//   - vnodes []vnode (sorted by pos)
//   - weights map[string]int
//   - nodeIDs map[string]struct{} (set of physical nodes)

// NewHashRing creates a new consistent hash ring.
func NewHashRing() *HashRing {
	panic("implement me")
}

// Add registers nodeID with weight virtual nodes on the ring.
func (r *HashRing) Add(nodeID string, weight int) {
	panic("implement me")
}

// Remove removes nodeID and all its virtual nodes from the ring.
func (r *HashRing) Remove(nodeID string) {
	panic("implement me")
}

// Lookup returns the responsible node for key (clockwise successor).
func (r *HashRing) Lookup(key string) string {
	panic("implement me")
}

// Replicas returns n distinct physical nodes responsible for key.
// First element is the primary (Lookup result), followed by successors.
func (r *HashRing) Replicas(key string, n int) []string {
	panic("implement me")
}

// Distribution returns a map of nodeID → number of keys from testKeys each owns.
func (r *HashRing) Distribution(testKeys []string) map[string]int {
	panic("implement me")
}

// ============================================================
// Helpers
// ============================================================

// hashKey returns a uint32 hash of s.
func hashKey(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// HashRing — stub; replace with your implementation.
type HashRing struct {
	mu sync.RWMutex
}

func main() {
	fmt.Println("=== Consistent Hash Ring ===")

	ring := NewHashRing()
	nodes := []string{"node-A", "node-B", "node-C", "node-D", "node-E"}
	for _, n := range nodes {
		ring.Add(n, DefaultVNodeCount)
	}

	// --- Basic lookup ---
	for _, key := range []string{"user:1001", "order:9982", "session:abc"} {
		n := ring.Lookup(key)
		fmt.Printf("Lookup(%q) → %s\n", key, n)
	}

	// --- Load distribution standard deviation ---
	var testKeys []string
	for i := 0; i < 10000; i++ {
		testKeys = append(testKeys, "key:"+strconv.Itoa(i))
	}
	dist := ring.Distribution(testKeys)
	var counts []float64
	var total float64
	for _, n := range nodes {
		c := float64(dist[n])
		counts = append(counts, c)
		total += c
	}
	mean := total / float64(len(nodes))
	var variance float64
	for _, c := range counts {
		variance += (c - mean) * (c - mean)
	}
	stddev := math.Sqrt(variance / float64(len(nodes)))
	pct := stddev / mean * 100
	fmt.Printf("Distribution stddev = %.1f%% of mean (expect < 15%%)\n", pct)
	for _, n := range nodes {
		fmt.Printf("  %s: %d keys\n", n, dist[n])
	}

	// --- Remove a node and verify key migration ---
	keyPre := ring.Lookup("user:1001")
	ring.Remove(keyPre)
	keyPost := ring.Lookup("user:1001")
	if keyPost != keyPre {
		fmt.Printf("PASS: after removing %s, user:1001 → %s\n", keyPre, keyPost)
	} else {
		fmt.Printf("FAIL: key still routes to removed node %s\n", keyPre)
	}

	// --- Replication ---
	ring2 := NewHashRing()
	for _, n := range nodes {
		ring2.Add(n, DefaultVNodeCount)
	}
	replicas := ring2.Replicas("important-data", 3)
	unique := make(map[string]bool)
	for _, r := range replicas {
		unique[r] = true
	}
	if len(unique) == 3 {
		fmt.Printf("PASS: 3 distinct replicas = %v\n", replicas)
	} else {
		fmt.Printf("FAIL: replicas not distinct: %v\n", replicas)
	}

	fmt.Println("Done.")
	_ = sort.SearchInts
}
