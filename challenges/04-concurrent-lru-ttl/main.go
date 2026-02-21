package main

import (
	"container/list"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ============================================================
// CHALLENGE 04: Concurrent LRU with TTL and Frequency Bias
// ============================================================
// Build an O(1) thread-safe LRU cache with lazy TTL expiry and
// frequency-biased eviction (Redis-style).
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// CacheStats holds observable cache metrics.
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
}

// TODO: Define entry struct:
//   - key, value string
//   - freq uint8 (saturates at 255)
//   - expiresAt time.Time
//   - elem *list.Element (pointer back to LRU list element)

// TODO: Define Cache struct:
//   - mu sync.RWMutex
//   - cap int
//   - items map[string]*entry
//   - lru *list.List (front = most recent)
//   - stats CacheStats

// NewCache creates a new cache with the given capacity.
func NewCache(capacity int) *Cache {
	panic("implement me")
}

// Set inserts or updates key with the given value and TTL.
func (c *Cache) Set(key, value string, ttl time.Duration) {
	panic("implement me")
}

// Get retrieves a value. Returns ("", false) on miss or expiry.
func (c *Cache) Get(key string) (string, bool) {
	panic("implement me")
}

// Delete explicitly removes a key.
func (c *Cache) Delete(key string) {
	panic("implement me")
}

// Len returns the count of non-expired entries.
func (c *Cache) Len() int {
	panic("implement me")
}

// Stats returns a snapshot of cache metrics.
func (c *Cache) Stats() CacheStats {
	panic("implement me")
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// Cache — stub; replace with your implementation.
type Cache struct {
	mu sync.RWMutex
	l  *list.List
}

func main() {
	fmt.Println("=== Concurrent LRU with TTL + Frequency Bias ===")

	c := NewCache(5)

	// --- Basic set / get ---
	c.Set("a", "1", time.Minute)
	c.Set("b", "2", time.Minute)
	c.Set("c", "3", time.Minute)
	v, ok := c.Get("a")
	fmt.Printf("Get(a)=%q ok=%v (expect 1 true)\n", v, ok)

	// --- Eviction: add beyond capacity ---
	c.Set("d", "4", time.Minute)
	c.Set("e", "5", time.Minute)
	// Warm up "c" frequency so it survives eviction
	for i := 0; i < 10; i++ {
		c.Get("c")
	}
	c.Set("f", "6", time.Minute) // should evict something from the LRU tail
	fmt.Printf("Len after eviction = %d (expect 5)\n", c.Len())

	// --- TTL expiry ---
	c.Set("z", "zzz", 100*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	_, found := c.Get("z")
	fmt.Printf("TTL expired Get(z) found=%v (expect false)\n", found)

	// --- Stats ---
	stats := c.Stats()
	fmt.Printf("Stats: hits=%d misses=%d evictions=%d\n", stats.Hits, stats.Misses, stats.Evictions)

	// --- Zipf workload hit rate ---
	c2 := NewCache(1000)
	zipf := rand.NewZipf(rand.New(rand.NewSource(42)), 1.1, 1, 9999)
	for i := 0; i < 100_000; i++ {
		k := fmt.Sprintf("k%d", zipf.Uint64())
		if _, hit := c2.Get(k); !hit {
			c2.Set(k, k, time.Minute)
		}
	}
	s2 := c2.Stats()
	hitRate := float64(s2.Hits) / float64(s2.Hits+s2.Misses) * 100
	fmt.Printf("Zipf hit rate = %.1f%% (expect > 85%%)\n", hitRate)

	// --- Concurrency ---
	c3 := NewCache(100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				key := fmt.Sprintf("k%d", (id*1000+j)%150)
				if _, hit := c3.Get(key); !hit {
					c3.Set(key, key, time.Minute)
				}
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("Concurrent test done. Len=%d\n", c3.Len())
	fmt.Println("Run with: go test -race ./...")
}
