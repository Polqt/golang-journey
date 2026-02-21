package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ============================================================
// CHALLENGE 02: Fencing-Token Distributed Lock
// ============================================================
// Build a lease-based distributed lock manager with fencing
// tokens that prevent stale writers from overwriting fresh data.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// ErrLockHeld is returned when the lock is already held by another client.
var ErrLockHeld = errors.New("lock is held by another client")

// ErrStaleToken is returned when a write is attempted with an outdated token.
var ErrStaleToken = errors.New("stale fencing token rejected")

// ErrNotOwner is returned when a client tries to unlock a lock it doesn't hold.
var ErrNotOwner = errors.New("caller is not the lock owner")

// TODO: Define lockEntry struct:
//   - clientID string
//   - token int64
//   - expiresAt time.Time
//   - partitioned bool (simulates network partition)

// TODO: Define Resource struct:
//   - highWaterMark int64 (highest token ever accepted)
//   - data string
//   - mu sync.Mutex

// TODO: Define LockManager struct:
//   - mu sync.Mutex
//   - current *lockEntry (nil if unlocked)
//   - nextToken int64 (monotonically increasing counter)
//   - leaseDuration time.Duration
//   - resource *Resource

// NewLockManager creates a lock manager with the given lease duration.
func NewLockManager(leaseDuration time.Duration) *LockManager {
	panic("implement me")
}

// Lock attempts to acquire the lock for clientID.
// Returns a fencing token on success.
func (lm *LockManager) Lock(clientID string) (int64, error) {
	panic("implement me")
}

// Unlock releases the lock. Validates that clientID holds token.
func (lm *LockManager) Unlock(clientID string, token int64) error {
	panic("implement me")
}

// Renew extends the lease for clientID + token.
// If the client is simulated-partitioned, this call silently fails.
func (lm *LockManager) Renew(clientID string, token int64) error {
	panic("implement me")
}

// WriteResource attempts a resource write. Rejects stale tokens.
func (lm *LockManager) WriteResource(clientID string, token int64, data string) error {
	panic("implement me")
}

// SetPartitioned simulates a network partition for a client.
func (lm *LockManager) SetPartitioned(clientID string, partitioned bool) {
	panic("implement me")
}

// ReadResource returns the current resource data and the token that wrote it.
func (lm *LockManager) ReadResource() (data string, token int64) {
	panic("implement me")
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// LockManager — stub; replace with your implementation.
type LockManager struct {
	mu            sync.Mutex
	leaseDuration time.Duration
}

func main() {
	fmt.Println("=== Fencing Token Distributed Lock ===")

	lm := NewLockManager(300 * time.Millisecond)

	// --- Basic acquire / write / unlock ---
	tok1, err := lm.Lock("client-A")
	mustNil(err)
	fmt.Printf("client-A acquired lock, token=%d\n", tok1)

	err = lm.WriteResource("client-A", tok1, "version-1")
	mustNil(err)
	data, tok := lm.ReadResource()
	fmt.Printf("Resource: %q written by token=%d\n", data, tok)

	err = lm.Unlock("client-A", tok1)
	mustNil(err)

	// --- Fencing: stale token rejected after new client acquires ---
	tok2, _ := lm.Lock("client-B")
	fmt.Printf("client-B acquired lock, token=%d\n", tok2)

	err = lm.WriteResource("client-A", tok1, "STALE WRITE")
	if errors.Is(err, ErrStaleToken) {
		fmt.Println("PASS: stale token correctly rejected")
	} else {
		fmt.Printf("FAIL: expected ErrStaleToken, got: %v\n", err)
	}

	lm.Unlock("client-B", tok2)

	// --- Lease expiry ---
	tok3, _ := lm.Lock("client-C")
	fmt.Printf("client-C acquired lock, token=%d\n", tok3)
	time.Sleep(400 * time.Millisecond) // wait for lease to expire

	tok4, err := lm.Lock("client-D")
	if err == nil {
		fmt.Printf("PASS: lock re-acquired after expiry, token=%d\n", tok4)
	} else {
		fmt.Printf("FAIL: lock should have expired: %v\n", err)
	}

	// --- Token monotonicity ---
	lm.Unlock("client-D", tok4)
	tok5, _ := lm.Lock("client-E")
	if tok5 > tok4 {
		fmt.Printf("PASS: token strictly increasing (%d > %d)\n", tok5, tok4)
	} else {
		fmt.Printf("FAIL: token not monotonic (%d <= %d)\n", tok5, tok4)
	}

	// --- Partition simulation ---
	lm.Unlock("client-E", tok5)
	tok6, _ := lm.Lock("client-F")
	lm.SetPartitioned("client-F", true)
	time.Sleep(400 * time.Millisecond) // renewal silently fails; lease expires
	tok7, err := lm.Lock("client-G")
	if err == nil && tok7 > tok6 {
		fmt.Printf("PASS: partitioned client evicted, token=%d > %d\n", tok7, tok6)
	} else {
		fmt.Printf("Partition test result: err=%v tok7=%d tok6=%d\n", err, tok7, tok6)
	}

	fmt.Println("Done.")
}

func mustNil(err error) {
	if err != nil {
		panic(err)
	}
}
