# Project 03 — CRDT Collaboration Backend

> **Difficulty**: Expert · **Domain**: Distributed Systems, Real-time Sync, WebSockets
> **Real-world analog**: Figma's multiplayer, Notion's real-time editing, Google Docs, Automerge

---

## Why This Project Exists

Operational Transforms (OT) are notoriously hard to implement correctly. The industry is moving
toward **Conflict-free Replicated Data Types (CRDTs)** — data structures that always merge
consistently without coordination. This project builds a real-time collaboration backend using
three foundational CRDTs.

---

## Folder Structure

```
03-crdt-collab-backend/
├── go.mod
├── main.go
├── crdt/
│   ├── lww_register.go       # Last-Write-Wins Register
│   ├── pn_counter.go         # PN-Counter (increment/decrement)
│   ├── or_set.go             # Observed-Remove Set (add/remove without conflicts)
│   ├── rga.go                # Replicated Growable Array (for text editing)
│   └── vclock.go             # Vector Clock for causality tracking
├── session/
│   ├── session.go            # Per-document session: clients + state
│   └── hub.go                # WebSocket hub: route ops to sessions
├── transport/
│   ├── ws.go                 # WebSocket upgrade + message routing
│   └── protocol.go           # Op message protocol (JSON over WS)
├── storage/
│   └── snapshot.go           # Periodic CRDT state snapshot to disk
└── client/
    └── demo/
        └── index.html        # Minimal browser client for testing (no JS framework)
```

---

## Core Concepts

### CRDT Types You'll Implement

| CRDT | Use Case | Key Property |
|---|---|---|
| LWW-Register | Cursor position, user presence | Last timestamp wins |
| PN-Counter | Like counts, word count | Inc/Dec convergence |
| OR-Set | Tags, participant list | Add/Remove without conflicts |
| RGA | Collaborative text | Per-character tombstones |

---

## Implementation Guide

### Phase 1 — Vector Clocks (Week 1)

```go
type VClock map[string]uint64  // nodeID → counter

func (v VClock) Increment(nodeID string) VClock
func (v VClock) HappensBefore(other VClock) bool
func (v VClock) Concurrent(other VClock) bool
func (v VClock) Merge(other VClock) VClock
```

Vector clocks are the causality backbone for CRDTs that need ordering.
- `HappensBefore`: true if for all nodes, v[n] <= other[n] and at least one is strictly less
- `Merge`: take max per component

---

### Phase 2 — LWW Register + PN Counter (Week 1)

**LWW Register**: stores a single value with a timestamp. On merge, keep the higher timestamp.

```go
type LWWRegister[T any] struct {
    Value     T
    Timestamp time.Time
    NodeID    string    // for tie-breaking on equal timestamps
}
func (r *LWWRegister[T]) Set(val T, ts time.Time, nodeID string)
func (r *LWWRegister[T]) Merge(other LWWRegister[T])
```

**PN Counter**: separate P (positive) and N (negative) GCounters per node.

```go
type PNCounter struct {
    positive map[string]int64  // nodeID → increments
    negative map[string]int64  // nodeID → decrements
}
func (c *PNCounter) Increment(nodeID string, delta int64)
func (c *PNCounter) Decrement(nodeID string, delta int64)
func (c *PNCounter) Value() int64
func (c *PNCounter) Merge(other *PNCounter)
```

---

### Phase 3 — OR-Set (Week 2)

The tricky one: both add and remove the same element without conflict.

**Solution**: Every add attaches a unique tag (UUID). Remove only removes specific tags. If a
concurrent add and remove happen, the add wins (because the remove only targets old tags).

```go
type ORSet struct {
    elements map[string]map[string]bool  // value → set of add-tags
    removed  map[string]bool             // removed tags
}
func (s *ORSet) Add(value, nodeID string) string  // returns tag
func (s *ORSet) Remove(value string)               // removes all current tags
func (s *ORSet) Contains(value string) bool
func (s *ORSet) Values() []string
func (s *ORSet) Merge(other *ORSet)
```

---

### Phase 4 — RGA (Replicated Growable Array) for Text (Week 2-3)

RGA is the academic basis of Logoot and LSEQ. Each character has a unique identifier and a
"insert-after" pointer. Deletions are tombstones (the node stays but is invisible).

```go
type RGANode struct {
    ID        RGANodeID  // (sequenceNum, nodeID) — globally unique
    InsertAfter RGANodeID
    Char      rune
    Deleted   bool
}
type RGA struct {
    nodes []RGANode  // sorted by ID
    index map[RGANodeID]int
}
func (r *RGA) Insert(afterID RGANodeID, char rune, nodeID string) RGANode
func (r *RGA) Delete(id RGANodeID)
func (r *RGA) Text() string
func (r *RGA) Merge(op RGANode)  // apply a remote op
```

**Key insight**: when two clients insert after the same node concurrently, break ties by
`nodeID` comparison to determine a total order.

---

### Phase 5 — WebSocket Hub + Real-time Sync (Week 3)

```
Client A ─WS─► Hub ──► Session("doc-1") ──► [apply op + broadcast to peers]
Client B ─WS─► Hub ──► Session("doc-1") ──► same session
```

Protocol (JSON over WebSocket):
```json
{"type":"op", "docID":"doc-1", "op":{"kind":"rga_insert","afterID":"...","char":"h","nodeID":"A"}}
{"type":"state","docID":"doc-1","text":"hello","presence":{"A":{"cursor":2}}}
```

**Steps**:
1. On connect: client sends `JOIN doc-1`, server sends full state snapshot
2. On op: server applies to CRDT, broadcasts to all other clients in session
3. Every 30s: server snapshots CRDT state to disk

---

## Acceptance Criteria

- [ ] Two clients typing concurrently converge to identical text
- [ ] OR-Set add/remove conflicts resolve correctly
- [ ] PN-Counter never gives wrong value under concurrent updates
- [ ] WebSocket clients receive ops in < 50ms LAN latency

---

## Stretch Goals

- Implement **Yjs-compatible** delta sync (only send diffs, not full state)
- Add **persistence with replay**: store all ops, reconstruct any document from scratch
- Build a **conflict visualizer TUI** showing before/after merge state
