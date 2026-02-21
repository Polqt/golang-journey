package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================
// CHALLENGE 05: Circuit Breaker State Machine
// ============================================================
// Implement a three-state circuit breaker with rolling window
// failure detection and automatic recovery probing.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// ErrCircuitOpen is returned when the circuit is OPEN.
var ErrCircuitOpen = errors.New("circuit open")

// CircuitState represents the breaker state.
type CircuitState int32

const (
	StateClosed   CircuitState = 0
	StateOpen     CircuitState = 1
	StateHalfOpen CircuitState = 2
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

// BreakerStats holds observable metrics.
type BreakerStats struct {
	State            CircuitState
	TotalRequests    int64
	Failures         int64
	Successes        int64
	ConsecutiveFails int64
	FailureRate      float64
}

// BreakerConfig holds circuit breaker configuration.
type BreakerConfig struct {
	WindowSize       int           // rolling window size
	FailureThreshold float64       // e.g. 0.5 = 50% triggers OPEN
	ResetTimeout     time.Duration // OPEN → HALF-OPEN after this
	ProbeCount       int           // max probes in HALF-OPEN
}

// TODO: Define CircuitBreaker struct:
//   - config BreakerConfig
//   - state atomic.Int32
//   - mu sync.Mutex
//   - window []bool (circular buffer: true=failure)
//   - head int (circular buffer head)
//   - windowCount int (filled slots)
//   - openedAt time.Time
//   - probesSent, probesSuccess int
//   - stats BreakerStats

// NewCircuitBreaker creates a circuit breaker with the given config.
func NewCircuitBreaker(cfg BreakerConfig) *CircuitBreaker {
	panic("implement me")
}

// Execute runs fn through the circuit breaker.
// Returns ErrCircuitOpen immediately when the circuit is OPEN.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	panic("implement me")
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	panic("implement me")
}

// Stats returns a snapshot of circuit breaker metrics.
func (cb *CircuitBreaker) Stats() BreakerStats {
	panic("implement me")
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// CircuitBreaker — stub; replace with your implementation.
type CircuitBreaker struct {
	config BreakerConfig
	state  atomic.Int32
	mu     sync.Mutex
}

func main() {
	fmt.Println("=== Circuit Breaker State Machine ===")

	cfg := BreakerConfig{
		WindowSize:       10,
		FailureThreshold: 0.6,
		ResetTimeout:     300 * time.Millisecond,
		ProbeCount:       3,
	}
	cb := NewCircuitBreaker(cfg)

	// --- Inject failures to trip the breaker ---
	failFn := func() error { return errors.New("downstream error") }
	successFn := func() error { return nil }

	for i := 0; i < 7; i++ {
		cb.Execute(failFn)
	}
	fmt.Printf("After 7 failures: state=%s (expect OPEN)\n", cb.State())

	// --- Requests while OPEN are rejected immediately ---
	err := cb.Execute(successFn)
	if errors.Is(err, ErrCircuitOpen) {
		fmt.Println("PASS: request rejected while OPEN")
	} else {
		fmt.Printf("FAIL: expected ErrCircuitOpen, got: %v\n", err)
	}

	// --- Wait for reset timeout → HALF-OPEN ---
	time.Sleep(400 * time.Millisecond)
	fmt.Printf("After reset timeout: state=%s (expect HALF-OPEN)\n", cb.State())

	// --- Successful probes → CLOSED ---
	for i := 0; i < 3; i++ {
		cb.Execute(successFn)
	}
	fmt.Printf("After 3 successful probes: state=%s (expect CLOSED)\n", cb.State())

	// --- Trip again, then partial failure in HALF-OPEN → re-OPEN ---
	for i := 0; i < 7; i++ {
		cb.Execute(failFn)
	}
	time.Sleep(400 * time.Millisecond) // → HALF-OPEN
	cb.Execute(failFn)                 // probe fails → back to OPEN
	fmt.Printf("After failed probe: state=%s (expect OPEN)\n", cb.State())

	// --- Concurrency test ---
	cb2 := NewCircuitBreaker(cfg)
	var wg sync.WaitGroup
	var blocked, passed int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				fn := successFn
				if id%2 == 0 {
					fn = failFn
				}
				err := cb2.Execute(fn)
				if errors.Is(err, ErrCircuitOpen) {
					atomic.AddInt64(&blocked, 1)
				} else {
					atomic.AddInt64(&passed, 1)
				}
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("Concurrency: passed=%d blocked=%d\n", passed, blocked)
	fmt.Println("Done. Run: go test -race ./...")
}
