# Project 01 — LogQL: A SQL-Like Query Engine for Structured Logs

> **Difficulty**: Senior · **Domain**: Storage Engines, Query Processing, CLI
> **Real-world analog**: ClickHouse, Grafana Loki, DataDog Log Analytics

---

## Why This Project Exists

Every company ingests terabytes of structured logs (NDJSON, CSV, key=value pairs). Searching
them with `grep` doesn't scale, and shipping them to a SaaS tool costs thousands per month.
This project builds a **local, offline SQL-like query engine** for log files that:

- Parses multiple log formats (NDJSON, logfmt, CSV) into a unified column store
- Executes `SELECT`, `WHERE`, `GROUP BY`, `ORDER BY`, `LIMIT` queries
- Runs queries across **multiple files** and **goroutine-parallelized** chunks
- Provides a **live tail mode** and a **TUI REPL**

---

## Folder Structure

```
01-log-query-engine/
├── go.mod
├── main.go                    # CLI entrypoint
├── cmd/
│   ├── query.go               # 'logql query' subcommand
│   ├── tail.go                # 'logql tail' subcommand
│   └── repl.go                # interactive REPL mode
├── engine/
│   ├── parser/
│   │   ├── ndjson.go          # NDJSON → Row
│   │   ├── logfmt.go          # key=value → Row
│   │   └── csv.go             # CSV → Row
│   ├── planner/
│   │   ├── sql.go             # SQL → LogicalPlan AST
│   │   └── optimizer.go       # predicate pushdown, limit pushdown
│   ├── executor/
│   │   ├── scan.go            # parallel file scanner
│   │   ├── filter.go          # WHERE evaluation
│   │   ├── aggregate.go       # GROUP BY + COUNT/SUM/AVG/MIN/MAX
│   │   ├── sort.go            # ORDER BY with external merge sort
│   │   └── project.go         # SELECT column projection
│   └── catalog/
│       ├── schema.go          # inferred Schema from sample rows
│       └── index.go           # optional bloom filter per field
├── renderer/
│   ├── table.go               # tabular output (tablewriter-style)
│   └── json.go                # JSON output mode
└── testdata/
    ├── nginx.ndjson           # sample nginx access logs
    └── app.logfmt             # sample app logs
```

---

## Implementation Guide

### Phase 1 — Data Ingestion (Week 1)

**Goal**: Read log files of different formats into a unified `Row` type.

```go
// engine/parser/row.go
type Row map[string]any  // field → value (string, int64, float64, bool, time.Time)

type Parser interface {
    Parse(line []byte) (Row, error)
}
```

**Steps**:
1. Implement `NDJSONParser` using `encoding/json` — unmarshal each line as `map[string]any`
2. Implement `LogfmtParser` — split on spaces, parse `key=value` or `key="quoted value"`
3. Implement `CSVParser` using `encoding/csv` — first line is header
4. Write a `DetectFormat(sample []byte) Parser` function that auto-detects based on the first line
5. Benchmark: target 500MB/s parsing throughput using `bufio.Scanner`

**Key insight**: Store all values as `any` for schema flexibility, but coerce to typed values
lazily during query execution.

---

### Phase 2 — SQL Parser (Week 1-2)

**Goal**: Parse a subset of SQL into an AST.

Supported syntax:
```sql
SELECT level, COUNT(*), AVG(duration_ms)
FROM "app.logfmt"
WHERE level = 'error' AND duration_ms > 500
GROUP BY level
ORDER BY COUNT(*) DESC
LIMIT 20
```

**Steps**:
1. Build a **hand-written recursive descent parser** (do NOT use `antlr` or `participle`)
2. AST node types: `SelectStmt`, `WhereClause`, `BinaryExpr`, `FuncCallExpr`, `Literal`
3. Support operators: `=`, `!=`, `>`, `<`, `>=`, `<=`, `AND`, `OR`, `NOT`, `LIKE`, `IN`
4. Aggregate functions: `COUNT`, `SUM`, `AVG`, `MIN`, `MAX`

**Pro tip**: Start with a tokenizer (`IDENT`, `STRING`, `NUMBER`, `KEYWORD`, `OP`) before building
the parser. This separates concerns cleanly.

---

### Phase 3 — Query Executor (Week 2-3)

**Goal**: Execute the AST against actual data.

```
Files → [Scanner] → [Filter] → [Aggregate/Project] → [Sort] → [Limit] → Output
```

**Steps**:
1. `scan.go`: Open file, detect format, yield `Row` over a channel (streaming — never load all rows)
2. `filter.go`: Evaluate `WhereClause` against each Row — return bool
3. `aggregate.go`: For `GROUP BY`, maintain a `map[groupKey]*accumulator` per group
4. `sort.go`: Collect results into memory; if > 10MB, implement external merge sort using tmp files
5. `project.go`: Select only requested columns from each Row

**Parallelism trick**: Split the input file at line boundaries into N chunks (one per CPU),
process in parallel goroutines, merge results. Use `sync.WaitGroup` + `errgroup`.

---

### Phase 4 — Optimizer (Week 3)

**Goal**: Make queries 10x faster without changing results.

Optimizations to implement:
1. **Predicate pushdown**: evaluate `WHERE` before aggregation (reduce rows early)
2. **Limit pushdown**: for `ORDER BY x LIMIT 10`, use a min-heap of size 10 instead of full sort
3. **Column pruning**: only parse fields referenced in SELECT/WHERE from each row
4. **Bloom filter skip**: if a field has a bloom filter index and WHERE tests exact equality,
   skip chunks that definitely don't contain the value

---

### Phase 5 — CLI + TUI REPL (Week 4)

**Goal**: Deliver a polished CLI tool.

```bash
# Query mode
logql query -f nginx.ndjson "SELECT status, COUNT(*) FROM _ GROUP BY status ORDER BY 2 DESC"

# Tail mode (live follow)
logql tail -f /var/log/app.logfmt -q "WHERE level = 'error'"

# Interactive REPL
logql repl nginx.ndjson
> SELECT * FROM _ WHERE status = 500 LIMIT 5
> \timing    # toggle query timing
> \format json
> \quit
```

Use standard `flag` or `os.Args` for the CLI — no `cobra` allowed (build it yourself to learn).
For the REPL: use `bufio.Scanner` on `os.Stdin` with a simple `readline`-style prompt.

---

## Acceptance Criteria

- [ ] Query 100MB NDJSON file in < 3 seconds on a modern laptop
- [ ] GROUP BY with 5 aggregate functions works correctly
- [ ] Interactive REPL responds to queries within 1s for files < 1GB
- [ ] All 3 log formats parse correctly
- [ ] `--explain` flag shows execution plan and estimated row counts

---

## Stretch Goals

- Add `JOIN` between two log files on a common field
- Implement a **columnar index** (sorted on-disk per field) for accelerating range queries
- Add `WINDOW` functions: `ROW_NUMBER()`, `LAG()`, `LEAD()`
- Export results to Parquet using only stdlib binary writing
