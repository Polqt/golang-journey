# Challenge 10 — Query Cost Optimizer (Join Order DP)

## Difficulty: Expert
## Category: Databases · Dynamic Programming · Graph Algorithms

---

## Problem Statement

Every relational database (PostgreSQL, MySQL, DuckDB) has a **query optimizer** that rewrites
your SQL before execution. The hardest part: **finding the optimal join order** for multi-table
joins.

A naive join order for 5 tables has `5! / 2 = 60` possibilities. For 10 tables: `10! / 2 =
1,814,400`. The correct solution uses **Selinger-style dynamic programming**: compute the optimal
plan for each subset of tables bottom-up, reusing sub-results.

---

## The Problem

Given:
- A list of **Relations** with known row counts (estimated by stats)
- A list of **Join predicates** (equality conditions between columns)
- **Cost model**: `cost(A ⋈ B) = cost(A) + cost(B) + |A| * selectivity(A,B) * |B|`

Find the **minimum-cost left-deep join tree** using the Selinger DP algorithm.

---

## Requirements

```go
// Relations
r := []Relation{
    {Name: "orders",    Rows: 1_000_000},
    {Name: "customers", Rows: 50_000},
    {Name: "products",  Rows: 10_000},
    {Name: "suppliers", Rows: 500},
    {Name: "regions",   Rows: 20},
}
// Predicates: defines selectivity between relation pairs
preds := []Predicate{
    {Left: "orders",    Right: "customers", Selectivity: 0.0001},
    {Left: "orders",    Right: "products",  Selectivity: 0.001},
    {Left: "products",  Right: "suppliers", Selectivity: 0.01},
    {Left: "suppliers", Right: "regions",   Selectivity: 0.05},
}
optimizer := NewOptimizer(r, preds)
plan := optimizer.BestPlan()
// plan.Cost, plan.Order — the optimal join sequence
```

---

## Algorithm (Selinger DP)

1. For each single relation S: `dp[{S}] = Plan{cost: 0, rows: S.Rows}`
2. For each subset size 2..N:
   - For each subset T of that size:
     - For each relation R in T:
       - Let S = T \ {R}
       - If `dp[S]` exists and join predicate connects S to R:
         - `newCost = dp[S].cost + dp[S].rows * selectivity(S,R) * R.rows`
         - If `newCost < dp[T].cost` → update `dp[T]`
3. Return `dp[allRelations]`

---

## Constraints

- Represent subsets as `uint32` bitmasks (supports up to 32 relations)
- Total complexity: O(3^N) — acceptable up to N=15
- Only consider plans where a join predicate exists between the subsets (avoid cartesian products unless no other option)

---

## Acceptance Criteria

- [ ] Correctly finds optimal order for the 5-relation example above
- [ ] Produces lower cost than any random join order (verified by exhaustive check for N≤6)
- [ ] Handles disconnected join graphs (Cartesian products as fallback)
- [ ] Benchmark: N=12 completes in < 100ms

---

## Stretch Goals

- Implement **bushy tree** plans (not just left-deep)
- Add **index scan cost**: if a join column has an index, use `index_cost = log2(rows)` instead
- Implement **histogram-based selectivity**: more accurate than fixed selectivity constants
