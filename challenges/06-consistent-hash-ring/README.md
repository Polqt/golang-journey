# Challenge 06 — Consistent Hash Ring with Virtual Nodes

## Difficulty: Hard
## Category: Distributed Systems · Data Structures

---

## Problem Statement

DynamoDB, Cassandra, and Riak use **consistent hashing** to distribute keys across a cluster
without a centralized directory. When nodes join or leave, only `K/N` keys are remapped (where K
= keys, N = nodes) rather than remapping everything.

Virtual nodes (vnodes) improve load balance: each physical node owns multiple evenly spread
positions on the ring. Real systems (Cassandra, DynamoDB) use 150–256 vnodes per physical node.

---

## Requirements

1. `Add(nodeID string, weight int)` — add a node; `weight` controls its vnode count
2. `Remove(nodeID string)` — remove a node and all its vnodes
3. `Lookup(key string) string` — return the responsible nodeID for key (clockwise successor)
4. `Replicas(key string, n int) []string` — return n distinct next nodes (for replication)
5. `Distribution() map[string]int` — count how many unique keys each node owns from a test set

---

## Constraints

- Use xxHash or `fnv` for the ring hash function (stdlib only)
- Ring must be a sorted slice of `uint32` positions (binary search for lookup)
- `weight` vnodes per node: default 150
- After `Remove`, all keys that pointed to the removed node now correctly point to its clockwise successor
- All operations goroutine-safe

---

## Hints

1. For each vnode: `hash(nodeID + "#" + strconv.Itoa(i))` → ring position
2. `sort.SearchUint32s` for O(log n) lookup
3. For `Replicas`, walk clockwise, skip vnodes pointing to already-seen physical nodes

---

## Acceptance Criteria

- [ ] Load standard deviation across 5 equal-weight nodes < 15% with 10,000 keys
- [ ] After removing a node, its keys correctly route to the next node
- [ ] Adding a node steals keys approximately proportional to its weight

---

## Stretch Goals

- Implement **jump consistent hash** and benchmark against ring-based approach
- Add **bounded load** (Google's 2017 paper): cap any node at `1.25x average load`
