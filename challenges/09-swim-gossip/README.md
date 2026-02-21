# Challenge 09 — SWIM Gossip Failure Detector

## Difficulty: Expert
## Category: Distributed Systems · Protocols · Networks

---

## Problem Statement

**SWIM** (Scalable Weakly-consistent Infection-style Process Group Membership protocol) is the
foundation of failure detection in Consul, Serf, CockroachDB, and Cassandra. It solves the
classic problem: in a distributed cluster, how do nodes detect failures without centralized
health checks that don't scale?

SWIM's key insight: random probing + indirect probing + gossip infection avoids the N² bandwidth
of all-to-all heartbeating while achieving logarithmic convergence time.

---

## Requirements

Simulate a cluster of N in-process nodes running SWIM-lite:

1. Each node maintains a **membership list**: `{nodeID, state: Alive|Suspect|Dead, incarnation}`
2. Every `protocolPeriod`, each node:
   - Picks a random member, sends a **direct ping**
   - If no ACK within `pingTimeout`, selects K random nodes to send **indirect ping requests**
   - If no indirect ACK, marks the node as **Suspect**
3. A Suspect node that isn't refuted within `suspectTimeout` is marked **Dead**
4. All state changes are **gossiped** piggyback on outgoing messages (infection-style)
5. A node can **refute** its own Suspect status by incrementing its `incarnation` number

---

## Simulation API

```go
cluster := NewCluster(10, ClusterConfig{...})
cluster.Start()
cluster.Kill(nodeID)    // hard kill a node (no more messages)
cluster.Partition(a, b) // block messages between node a and b
cluster.WaitConverged(timeout time.Duration) bool
cluster.MembershipState() map[string]NodeState
```

---

## Constraints

- All communication is in-process (use channels simulating network)
- Messages must have simulated latency (configurable `latencyMin/Max`)
- No external libraries
- Convergence: within `2 * log2(N) * protocolPeriods`, all alive nodes agree on dead nodes

---

## Acceptance Criteria

- [ ] After `Kill(nodeID)`, all other nodes mark it Dead within convergence time
- [ ] A partitioned node is eventually suspected/dead from the other side
- [ ] A falsely suspected node refutes itself using incarnation increment
- [ ] Works correctly for N=20 nodes

---

## Stretch Goals

- Implement **anti-entropy**: periodic full state sync to fix edge cases
- Add **user events**: arbitrary payload gossip (like Serf's custom events)
