package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"sync"
)

// ============================================================
// CHALLENGE 08: Partitioned Counting Bloom Filter
// ============================================================
// Implement a counting Bloom filter with 4-bit counters,
// partitioned shards, merge capability, and cardinality estimation.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// ErrUnderflow is returned when removing a key that was never added.
var ErrUnderflow = fmt.Errorf("counter underflow: key not present")

// BloomConfig holds computed Bloom filter parameters.
type BloomConfig struct {
	Capacity   int
	TargetFPR  float64
	Partitions int
	M          int // total bit slots per partition
	K          int // number of hash functions
}

// TODO: Define partition struct:
//   - nibbles []byte (4-bit counter array, 2 per byte)
//   - m int (number of slots)
//   - mu sync.RWMutex

// TODO: Define BloomFilter struct:
//   - config BloomConfig
//   - shards []*partition

// computeParams returns optimal m and k given n and target FPR.
func computeParams(n int, fpr float64) (m int, k int) {
	ln2 := math.Log(2)
	mFloat := -float64(n) * math.Log(fpr) / (ln2 * ln2)
	m = int(math.Ceil(mFloat))
	k = int(math.Round(float64(m) / float64(n) * ln2))
	if k < 1 {
		k = 1
	}
	return
}

// NewBloomFilter creates a partitioned counting Bloom filter.
func NewBloomFilter(capacity int, targetFPR float64, partitions int) *BloomFilter {
	panic("implement me")
}

// Add adds key to the filter.
func (bf *BloomFilter) Add(key string) {
	panic("implement me")
}

// Remove decrements counters for key. Returns ErrUnderflow if a counter is already 0.
func (bf *BloomFilter) Remove(key string) error {
	panic("implement me")
}

// MightContain returns true if key might be in the set (false positives possible).
func (bf *BloomFilter) MightContain(key string) bool {
	panic("implement me")
}

// FalsePositiveRate estimates the current FPR based on fill ratio.
func (bf *BloomFilter) FalsePositiveRate() float64 {
	panic("implement me")
}

// EstimatedCount estimates the number of distinct items in the filter.
func (bf *BloomFilter) EstimatedCount() int64 {
	panic("implement me")
}

// Merge unions another filter of identical config into bf.
func (bf *BloomFilter) Merge(other *BloomFilter) error {
	panic("implement me")
}

// ============================================================
// Helpers
// ============================================================

// kHashes generates k hash positions for key within [0, m).
func kHashes(key string, m, k int) []int {
	h1 := fnv.New64a()
	h1.Write([]byte(key))
	a := h1.Sum64()

	h2 := fnv.New64()
	h2.Write([]byte(key))
	b := h2.Sum64()

	positions := make([]int, k)
	for i := 0; i < k; i++ {
		positions[i] = int((a + uint64(i)*b) % uint64(m))
	}
	return positions
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// BloomFilter — stub; replace with your implementation.
type BloomFilter struct {
	mu sync.RWMutex
}

func main() {
	fmt.Println("=== Counting Bloom Filter ===")

	bf := NewBloomFilter(10000, 0.01, 4)

	// --- Add items and verify MightContain ---
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		bf.Add(keys[i])
	}

	// Must not have false negatives
	fn := 0
	for _, k := range keys {
		if !bf.MightContain(k) {
			fn++
		}
	}
	fmt.Printf("False negatives: %d (must be 0)\n", fn)

	// Measure false positives on unseen keys
	fp := 0
	for i := 10000; i < 11000; i++ {
		if bf.MightContain(fmt.Sprintf("key-%d", i)) {
			fp++
		}
	}
	fpRate := float64(fp) / 1000.0
	fmt.Printf("False positive rate: %.3f (target < 0.02)\n", fpRate)

	// --- Remove and verify ---
	bf.Remove(keys[0])
	if bf.MightContain(keys[0]) {
		fmt.Println("INFO: key-0 might still match (hash collision possible)")
	} else {
		fmt.Println("PASS: key-0 correctly removed")
	}

	// --- Cardinality estimate ---
	est := bf.EstimatedCount()
	fmt.Printf("Estimated count: %d (actual: 999, expect within 20%%)\n", est)

	// --- Merge ---
	bf2 := NewBloomFilter(10000, 0.01, 4)
	for i := 5000; i < 6000; i++ {
		bf2.Add(fmt.Sprintf("extra-%d", i))
	}
	if err := bf.Merge(bf2); err != nil {
		fmt.Printf("Merge error: %v\n", err)
	} else {
		fmt.Println("Merge OK")
	}

	fmt.Printf("FPR current: %.4f\n", bf.FalsePositiveRate())
	fmt.Println("Done.")
	_ = math.Log
	_ = kHashes
}
