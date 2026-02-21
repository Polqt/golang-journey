package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// ============================================================
// CHALLENGE 01: Adaptive Rate Limiter
// ============================================================
// Implement a multi-tenant, adaptive rate limiter that combines
// Token Bucket semantics with error-rate-driven adaptive scaling.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// TenantStats holds observable metrics for a single tenant.
type TenantStats struct {
	Allowed        int64
	Rejected       int64
	Tokens         float64
	AdaptiveFactor float64
	ErrorRate      float64
}

// TODO: Define a tenantBucket struct that holds:
//   - tokens float64 (current token count)
//   - lastRefill time.Time
//   - adaptiveFactor float64 [0.1, 1.0]
//   - errorWindow []errorEntry (circular buffer of recent errors)
//   - allowed, rejected int64 counters
//   - mu sync.Mutex (per-tenant lock, not global!)

// TODO: Define AdaptiveRateLimiter struct with:
//   - rate float64 (tokens per second)
//   - burst float64 (max burst size)
//   - tenants map[string]*tenantBucket
//   - mu sync.RWMutex (guards the map, not the buckets)

// NewAdaptiveRateLimiter creates a limiter with the given rate and burst.
func NewAdaptiveRateLimiter(rate, burst float64) *AdaptiveRateLimiter {
	panic("implement me")
}

// Allow attempts to consume one token for tenantID.
// wasError reports whether the previous request from this tenant resulted in an error.
// Returns true if the request is allowed (token consumed), false if rate-limited.
func (r *AdaptiveRateLimiter) Allow(tenantID string, wasError bool) bool {
	panic("implement me")
}

// Stats returns a snapshot of metrics for the given tenant.
func (r *AdaptiveRateLimiter) Stats(tenantID string) TenantStats {
	panic("implement me")
}

// ============================================================
// Provided scaffolding — do not modify below this line
// ============================================================

// AdaptiveRateLimiter — stub for compilation; replace with your implementation.
type AdaptiveRateLimiter struct {
	rate  float64
	burst float64
	mu    sync.RWMutex
	// TODO: add tenants map
}

func main() {
	fmt.Println("=== Adaptive Rate Limiter ===")

	limiter := NewAdaptiveRateLimiter(100, 10)

	// --- Basic allow/reject test ---
	allowed := 0
	for i := 0; i < 15; i++ {
		if limiter.Allow("tenant-A", false) {
			allowed++
		}
	}
	fmt.Printf("Burst test            — allowed %d/15 (expect ~10)\n", allowed)

	// --- Error injection drives adaptive factor down ---
	time.Sleep(200 * time.Millisecond) // allow partial refill
	for i := 0; i < 50; i++ {
		limiter.Allow("tenant-A", i%3 == 0) // ~33% error rate
	}
	stats := limiter.Stats("tenant-A")
	fmt.Printf("After error injection — AdaptiveFactor: %.3f (expect < 0.80)\n", stats.AdaptiveFactor)

	// --- Recovery over time ---
	time.Sleep(3 * time.Second)
	for i := 0; i < 10; i++ {
		limiter.Allow("tenant-A", false)
	}
	statsAfter := limiter.Stats("tenant-A")
	fmt.Printf("After recovery        — AdaptiveFactor: %.3f (expect > %.3f)\n",
		statsAfter.AdaptiveFactor, stats.AdaptiveFactor)

	// --- Concurrency stress test ---
	limiter2 := NewAdaptiveRateLimiter(1000, 50)
	var wg sync.WaitGroup
	var concAllowed, concRejected int64
	var cmu sync.Mutex
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tenantID := fmt.Sprintf("tenant-%d", id%5)
			for j := 0; j < 200; j++ {
				ok := limiter2.Allow(tenantID, j%10 == 0)
				cmu.Lock()
				if ok {
					concAllowed++
				} else {
					concRejected++
				}
				cmu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("Concurrency stress    — allowed: %d, rejected: %d\n", concAllowed, concRejected)

	// --- Assert adaptive factor never goes to zero ---
	for _, tenant := range []string{"tenant-0", "tenant-1", "tenant-2", "tenant-3", "tenant-4"} {
		s := limiter2.Stats(tenant)
		if s.AdaptiveFactor < 0.1 {
			fmt.Printf("FAIL: %s adaptive factor %.3f below minimum 0.1\n", tenant, s.AdaptiveFactor)
		}
	}

	fmt.Println("Done — run with: go test -race ./...")

	// Prevent "imported and not used" if math is needed
	_ = math.MaxFloat64
}
