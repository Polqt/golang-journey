package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ============================================================
// CHALLENGE 09: SWIM Gossip Failure Detector
// ============================================================
// Implement a SWIM-lite protocol simulation with direct probing,
// indirect probing, suspicion, and gossip-based dissemination.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// NodeState represents a member's liveness state.
type NodeState int

const (
	StateAlive   NodeState = iota
	StateSuspect           // probed but no ACK; awaiting refutation
	StateDead              // timed out as Suspect; presumed dead
)

func (s NodeState) String() string {
	switch s {
	case StateAlive:
		return "Alive"
	case StateSuspect:
		return "Suspect"
	case StateDead:
		return "Dead"
	default:
		return "Unknown"
	}
}

// MemberInfo is the view one node has of another.
type MemberInfo struct {
	NodeID      string
	State       NodeState
	Incarnation int64 // incremented by node itself to refute suspicion
}

// ClusterConfig holds SWIM protocol tuning parameters.
type ClusterConfig struct {
	ProtocolPeriod time.Duration // how often each node probes
	PingTimeout    time.Duration // ACK timeout for direct probe
	SuspectTimeout time.Duration // how long before Suspect → Dead
	IndirectK      int           // how many indirect probers to use
	LatencyMin     time.Duration // simulated message latency range
	LatencyMax     time.Duration
}

// DefaultClusterConfig returns sensible defaults.
func DefaultClusterConfig() ClusterConfig {
	return ClusterConfig{
		ProtocolPeriod: 200 * time.Millisecond,
		PingTimeout:    50 * time.Millisecond,
		SuspectTimeout: 600 * time.Millisecond,
		IndirectK:      3,
		LatencyMin:     2 * time.Millisecond,
		LatencyMax:     15 * time.Millisecond,
	}
}

// Message is a SWIM protocol packet.
type Message struct {
	From   string
	Type   MsgType
	Target string       // for Ping/PingReq
	Gossip []MemberInfo // piggybacked state
}

// MsgType identifies the SWIM message kind.
type MsgType int

const (
	MsgPing    MsgType = iota // direct probe
	MsgAck                    // response to Ping or PingReq
	MsgPingReq                // indirect probe request
)

// TODO: Define node struct:
//   - id string
//   - config ClusterConfig
//   - membership map[string]*MemberInfo
//   - incarnation int64 (self)
//   - inbox chan Message
//   - peers map[string]chan Message (simulated network)
//   - partitioned map[string]bool (blocked peers)
//   - mu sync.RWMutex
//   - ctx context.Context

// TODO: Define Cluster struct:
//   - nodes map[string]*node
//   - config ClusterConfig
//   - mu sync.Mutex

// NewCluster creates N nodes with the given config and wires their inboxes.
func NewCluster(n int, cfg ClusterConfig) *Cluster {
	panic("implement me")
}

// Start launches all node protocol goroutines.
func (c *Cluster) Start() {
	panic("implement me")
}

// Kill hard-kills nodeID (stops it from sending or receiving).
func (c *Cluster) Kill(nodeID string) {
	panic("implement me")
}

// Partition blocks messages between nodeA and nodeB (one-way or two-way).
func (c *Cluster) Partition(nodeA, nodeB string) {
	panic("implement me")
}

// WaitConverged polls until all alive nodes agree on the membership state,
// or returns false after timeout.
func (c *Cluster) WaitConverged(timeout time.Duration) bool {
	panic("implement me")
}

// MembershipState returns the membership view of a random alive node.
func (c *Cluster) MembershipState() map[string]NodeState {
	panic("implement me")
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// Cluster — stub; replace with your implementation.
type Cluster struct {
	mu     sync.Mutex
	config ClusterConfig
}

func main() {
	fmt.Println("=== SWIM Gossip Failure Detector ===")

	cfg := DefaultClusterConfig()
	cluster := NewCluster(10, cfg)
	cluster.Start()

	time.Sleep(500 * time.Millisecond) // let initial gossip converge

	// --- Verify all nodes see each other as Alive ---
	state := cluster.MembershipState()
	aliveCount := 0
	for _, s := range state {
		if s == StateAlive {
			aliveCount++
		}
	}
	fmt.Printf("Initial alive nodes: %d/10 (expect 10)\n", aliveCount)

	// --- Kill a node and wait for detection ---
	cluster.Kill("node-5")
	converged := cluster.WaitConverged(5 * time.Second)
	fmt.Printf("Converged after kill: %v (expect true)\n", converged)

	state = cluster.MembershipState()
	node5state := state["node-5"]
	fmt.Printf("node-5 state: %s (expect Dead)\n", node5state)

	aliveCount = 0
	for id, s := range state {
		if s == StateAlive {
			aliveCount++
			_ = id
		}
	}
	fmt.Printf("Alive after kill: %d (expect 9)\n", aliveCount)

	// --- Partition test ---
	cluster2 := NewCluster(6, cfg)
	cluster2.Start()
	time.Sleep(300 * time.Millisecond)

	// Partition node-0,1,2 from node-3,4,5
	for _, a := range []string{"node-0", "node-1", "node-2"} {
		for _, b := range []string{"node-3", "node-4", "node-5"} {
			cluster2.Partition(a, b)
		}
	}
	time.Sleep(2 * time.Second)
	fmt.Println("Partition test: nodes in isolated partitions should suspect each other")

	// --- Refutation test ---
	cluster3 := NewCluster(5, ClusterConfig{
		ProtocolPeriod: 100 * time.Millisecond,
		PingTimeout:    20 * time.Millisecond,
		SuspectTimeout: 300 * time.Millisecond,
		IndirectK:      2,
		LatencyMin:     1 * time.Millisecond,
		LatencyMax:     5 * time.Millisecond,
	})
	cluster3.Start()
	// Inject artificial suspicion of node-2 (it should refute itself)
	time.Sleep(1 * time.Second)
	s3 := cluster3.MembershipState()
	fmt.Printf("After refutation period, node-2 state: %s (expect Alive)\n", s3["node-2"])

	fmt.Println("Done.")
	// Prevent unused import error
	_, _ = rand.Intn, context.Background
}
