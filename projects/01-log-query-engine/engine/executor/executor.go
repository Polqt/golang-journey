// Package executor runs LogicalPlan queries against log file data.
package executor

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/Polqt/logql/engine/parser"
	"github.com/Polqt/logql/engine/planner"
)

// Result is a stream of result rows.
type Result struct {
	Columns []string
	Rows    []parser.Row
	Stats   ExecStats
}

// ExecStats contains query execution metrics.
type ExecStats struct {
	ScannedLines int64
	FilteredRows int64
	OutputRows   int64
	DurationMs   int64
}

// Engine executes queries against files.
type Engine struct {
	Workers int // parallelism; defaults to runtime.NumCPU()
}

// NewEngine creates a default engine.
func NewEngine() *Engine {
	return &Engine{Workers: runtime.NumCPU()}
}

// Execute runs a SELECT statement against the given files.
// Each file is accessible as table name "_" (or its basename).
func (e *Engine) Execute(ctx context.Context, stmt *planner.SelectStmt, files []string) (*Result, error) {
	// TODO: Phase 1 — scan all files in parallel using scanFile()
	// TODO: Phase 2 — filter each row using evalWhere()
	// TODO: Phase 3 — if GROUP BY present, group and aggregate
	// TODO: Phase 4 — if ORDER BY present, sort
	// TODO: Phase 5 — apply LIMIT
	// TODO: Phase 6 — project SELECT columns
	_ = stmt
	_ = files
	return nil, fmt.Errorf("engine.Execute: not yet implemented")
}

// scanFile reads a log file in chunks and sends rows to out.
// Uses multiple goroutines to parse chunks in parallel.
func scanFile(ctx context.Context, path string, out chan<- parser.Row, wg *sync.WaitGroup) {
	defer wg.Done()
	// TODO:
	// 1. Open file, create bufio.Scanner with large buffer (1MB)
	// 2. Detect format using parser.DetectFormat(firstLine)
	// 3. Scan lines, parse each with the detected parser
	// 4. Send each Row to out (check ctx.Done() for cancellation)
	_ = path
	_ = out
}

// evalWhere evaluates a WHERE expression against a Row.
// Returns (true, nil) if the row matches (or if where is nil).
func evalWhere(where planner.Expr, row parser.Row) (bool, error) {
	if where == nil {
		return true, nil
	}
	// TODO: recursively evaluate the expression tree
	// Handle: BinaryExpr (=,!=,<,>,<=,>=,AND,OR,LIKE,IN)
	//         UnaryExpr (NOT)
	//         ColumnRef → look up in row
	//         Literal → return value
	// Type coercion rules: compare numerically when both sides look numeric
	_ = row
	return false, fmt.Errorf("evalWhere: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// Aggregation
// ─────────────────────────────────────────────────────────────

// accumulator holds running aggregate state for a GROUP BY group.
type accumulator struct {
	count  int64
	sum    float64
	min    float64
	max    float64
	values []any // for AVG
}

// AggregateRows groups rows and computes aggregate functions.
// groupByExprs are the GROUP BY columns; aggExprs are COUNT/SUM/AVG/MIN/MAX.
func AggregateRows(rows []parser.Row, groupByExprs []planner.Expr, aggExprs []planner.Expr) ([]parser.Row, error) {
	// TODO: build groupKey string from evaluating groupByExprs on each row
	// TODO: accumulate into map[groupKey]*accumulator
	// TODO: emit one output row per group
	_ = rows
	return nil, fmt.Errorf("AggregateRows: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// Sorting
// ─────────────────────────────────────────────────────────────

// SortRows sorts rows by the given ORDER BY expressions.
// For small result sets (< 100K rows) use in-memory sort.
// For large results, implement external merge sort (stretch goal).
func SortRows(rows []parser.Row, orderBy []planner.OrderExpr) error {
	// TODO: use sort.SliceStable
	// For each OrderExpr, evaluate expr on two rows and compare
	_ = orderBy
	return fmt.Errorf("SortRows: not yet implemented")
}
