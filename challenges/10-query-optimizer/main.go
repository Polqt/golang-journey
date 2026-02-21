package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// ============================================================
// CHALLENGE 10: Query Cost Optimizer (Selinger DP)
// ============================================================
// Implement a Selinger-style dynamic programming join order
// optimizer using bitmask subset enumeration.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// Relation represents a database table with a known row count.
type Relation struct {
	Name string
	Rows int64
}

// Predicate represents a join condition between two relations.
type Predicate struct {
	Left        string
	Right       string
	Selectivity float64 // fraction of rows after join [0,1]
}

// JoinPlan represents the optimal plan for a subset of relations.
type JoinPlan struct {
	Subset uint32   // bitmask of relations
	Cost   float64  // accumulated cost
	Rows   int64    // estimated output rows
	Order  []string // join order (left to right)
}

// Optimizer holds the query optimization state.
type Optimizer struct {
	relations []Relation
	preds     map[uint32]float64 // bitmask(a|b) → selectivity
	index     map[string]int     // relation name → index
}

// NewOptimizer creates an optimizer from relations and predicates.
func NewOptimizer(relations []Relation, predicates []Predicate) *Optimizer {
	panic("implement me")
}

// BestPlan runs the Selinger DP algorithm and returns the optimal join plan.
func (o *Optimizer) BestPlan() JoinPlan {
	panic("implement me")
}

// AllPlanCosts returns the cost of every possible join order (for N<=6 validation).
func (o *Optimizer) AllPlanCosts() []JoinPlan {
	panic("implement me")
}

// ============================================================
// Helpers
// ============================================================

// popcount returns the number of set bits in x.
func popcount(x uint32) int {
	count := 0
	for x != 0 {
		count += int(x & 1)
		x >>= 1
	}
	return count
}

// subsets returns all subsets of mask with exactly k bits set.
func subsets(mask uint32, k int) []uint32 {
	var result []uint32
	// Gosper's hack: enumerate all k-bit subsets of mask
	// (standard bit-manipulation trick for subset enumeration)
	sub := uint32((1 << k) - 1) // smallest k-bit number
	for sub <= mask {
		if sub&mask == sub {
			result = append(result, sub)
		}
		// next combination — Gosper's hack
		c := sub & (^sub + 1)
		r := sub + c
		sub = (((r ^ sub) >> 2) / c) | r
		if sub > mask && popcount(sub) != k {
			break
		}
	}
	return result
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

func main() {
	fmt.Println("=== Query Cost Optimizer (Selinger DP) ===")

	relations := []Relation{
		{Name: "orders", Rows: 1_000_000},
		{Name: "customers", Rows: 50_000},
		{Name: "products", Rows: 10_000},
		{Name: "suppliers", Rows: 500},
		{Name: "regions", Rows: 20},
	}
	predicates := []Predicate{
		{Left: "orders", Right: "customers", Selectivity: 0.0001},
		{Left: "orders", Right: "products", Selectivity: 0.001},
		{Left: "products", Right: "suppliers", Selectivity: 0.01},
		{Left: "suppliers", Right: "regions", Selectivity: 0.05},
	}

	opt := NewOptimizer(relations, predicates)
	best := opt.BestPlan()
	fmt.Printf("Best plan cost: %.2f\n", best.Cost)
	fmt.Printf("Best join order: %v\n", best.Order)

	// --- Verify it's better than all alternatives (brute-force check for N=5) ---
	all := opt.AllPlanCosts()
	minCost := math.MaxFloat64
	for _, p := range all {
		if p.Cost < minCost {
			minCost = p.Cost
		}
	}
	if math.Abs(best.Cost-minCost) < 1.0 {
		fmt.Printf("PASS: DP cost (%.2f) matches brute-force minimum (%.2f)\n", best.Cost, minCost)
	} else {
		fmt.Printf("FAIL: DP cost %.2f != brute-force minimum %.2f\n", best.Cost, minCost)
	}

	// --- Scalability benchmark for N=12 ---
	bigRelations := make([]Relation, 12)
	for i := range bigRelations {
		bigRelations[i] = Relation{
			Name: fmt.Sprintf("t%d", i),
			Rows: int64(rand.Intn(1_000_000) + 1000),
		}
	}
	bigPreds := make([]Predicate, 0)
	for i := 0; i < 11; i++ {
		bigPreds = append(bigPreds, Predicate{
			Left:        fmt.Sprintf("t%d", i),
			Right:       fmt.Sprintf("t%d", i+1),
			Selectivity: 0.001 + rand.Float64()*0.01,
		})
	}
	opt12 := NewOptimizer(bigRelations, bigPreds)
	start := time.Now()
	plan12 := opt12.BestPlan()
	elapsed := time.Since(start)
	fmt.Printf("N=12 best cost: %.2f in %v (expect < 100ms)\n", plan12.Cost, elapsed.Round(time.Millisecond))

	fmt.Println("Done.")
	_ = subsets
	_ = popcount
	_ = math.MaxFloat64
}
