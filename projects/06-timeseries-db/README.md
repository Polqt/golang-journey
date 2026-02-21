# Project 06 — Time-Series Database Engine

> **Difficulty**: Expert · **Domain**: Storage Engines, Data Structures, Performance
> **Real-world analog**: InfluxDB, Prometheus TSDB, VictoriaMetrics, TimescaleDB

---

## Why This Project Exists

Time-series data is the fastest-growing data category. Every metrics system, IoT platform, and
financial data feed generates it. Standard relational databases fail at it because of their row
layout. This project builds a real time-series storage engine from scratch — the same ideas
used inside Prometheus and InfluxDB 3.0.

---

## Folder Structure

```
06-timeseries-db/
├── go.mod
├── main.go                           # CLI: tsdb query, tsdb write, tsdb compact
├── tsdb/
│   ├── series.go                     # Series ID (labels → uint64 fingerprint)
│   ├── head/
│   │   ├── head.go                   # In-memory WAL + chunk buffer (recent data)
│   │   ├── chunk.go                  # XOR-encoded chunk (Gorilla compression)
│   │   └── memchunk.go               # Active in-memory chunk for appending
│   ├── block/
│   │   ├── block.go                  # Immutable on-disk block
│   │   ├── index.go                  # Inverted index: label → series IDs
│   │   ├── chunks.go                 # Chunk file reader/writer
│   │   └── tombstone.go              # Tombstone file for deletions
│   ├── compact/
│   │   └── compactor.go              # Merge blocks + apply tombstones
│   ├── query/
│   │   ├── querier.go                # Multi-block query fan-out
│   │   ├── series_set.go             # Merge iterator for results
│   │   └── eval.go                   # PromQL-lite: rate(), sum(), avg()
│   └── wal/
│       └── wal.go                    # Write-Ahead Log for head persistence
└── api/
    └── http.go                       # Prometheus-compatible /api/v1 endpoints
```

---

## Core Data Model

```go
// A data point
type Sample struct {
    Timestamp int64   // Unix milliseconds
    Value     float64
}

// A time series is identified by a set of labels
type Labels map[string]string  // {"__name__": "cpu_usage", "host": "web-01", "cpu": "0"}

// A series contains all samples for one label set
type Series struct {
    Labels  Labels
    Samples []Sample
}
```

---

## Implementation Guide

### Phase 1 — Gorilla XOR Compression (Week 1)

Prometheus and InfluxDB use **Gorilla compression** (Facebook's 2015 paper) for time-series
chunks. Key insight: consecutive timestamps and values differ by small amounts — XOR encoding
exploits this for massive compression (12 bytes/sample → ~1.37 bytes/sample average).

**Timestamp compression**:
1. Store first timestamp as-is (varint)
2. Store second delta: `d1 = t2 - t1`
3. For each subsequent: `dod = (tn - tn-1) - (tn-1 - tn-2)`. If dod=0 → 1 bit '0'. If small → encode in fewer bits.

**Value compression**:
1. XOR each value with previous
2. If XOR=0 → 1 bit '0'
3. Otherwise encode leading zeros + meaningful bits

```go
type XORChunk struct {
    buf     []byte
    bw      *bitWriter
    samples int
    minT, maxT int64
    lastT    int64
    lastDelta int64
    lastV    uint64
    leading  uint8
    trailing uint8
}
func (c *XORChunk) Append(t int64, v float64)
func (c *XORChunk) Iterator() Iterator
```

---

### Phase 2 — Head Block (WAL + In-Memory Chunks) (Week 2)

The **Head** is the mutable, in-memory component that accepts new writes:

```
Write → WAL (durability) → In-Memory Chunks (fast query)
```

**Steps**:
1. On `Append(labels, t, v)`: compute label fingerprint, get/create series
2. Write to WAL immediately (reuse Challenge 03 WAL implementation!)
3. Append to the active XOR chunk for that series
4. When chunk fills (120 samples or > 2 hours): seal it, start new active chunk
5. Every 2 hours: flush sealed chunks to disk as a new Block, then truncate WAL

**Series fingerprinting**: `fnv64(sortedLabelPairs)` → uint64 series ID

---

### Phase 3 — Block Format (Week 2-3)

An immutable Block covers a 2-hour time range and contains:

```
block/
  chunks/
    000001    # chunk data file (series chunk data, binary)
  index        # inverted label index (posting lists)
  meta.json    # {minTime, maxTime, numSamples, numSeries}
  tombstones   # deleted time ranges
```

**Chunk file format**:
```
[magic: 4][version: 1][chunks...][CRC32: 4]
Each chunk: [series_id: 8][min_t: 8][max_t: 8][type: 1][len: 4][data: N]
```

**Inverted index**: for each label value pair (`host=web-01`), store a sorted list of series
IDs that have it. Use roaring bitmaps or sorted uint64 slices.

---

### Phase 4 — Query Engine (Week 3)

```go
// Range query (like Prometheus range_query)
querier := db.Querier(ctx, startMs, endMs)
seriesSet := querier.Select(labels.Matcher{Name:"__name__", Value:"cpu_usage"})
for seriesSet.Next() {
    s := seriesSet.At()
    fmt.Println(s.Labels(), s.Iterator())
}
```

**Fan-out over blocks**:
1. Find all blocks that overlap `[start, end]`
2. For each block, find matching series IDs via inverted index
3. Open chunk data, create iterator
4. Merge-sort iterators by timestamp (use a min-heap)

**PromQL-lite functions**:
- `rate(metric[5m])`: per-second rate of increase of a counter
- `sum(metric) by (host)`: aggregate by label
- `avg_over_time(metric[1h])`: moving average

---

### Phase 5 — Compaction (Week 4)

Compaction merges multiple 2-hour blocks into larger 24-hour blocks:

1. Find blocks in the same 24-hour window
2. Merge all chunk data (already XOR-compressed — read and re-emit)
3. Rebuild inverted index from merged series
4. Apply tombstones (remove deleted samples)
5. Write new block, delete old blocks

---

### Phase 6 — HTTP API (Week 4)

Prometheus-compatible API:
```
GET /api/v1/query?query=cpu_usage{host="web-01"}&time=1708300000
GET /api/v1/query_range?query=rate(cpu_usage[5m])&start=...&end=...&step=60
POST /api/v1/write  (Prometheus remote write format)
GET  /metrics        (Prometheus exposition format)
```

---

## Acceptance Criteria

- [ ] Gorilla compression achieves < 2 bytes/sample on realistic metrics data
- [ ] `10M samples/second` write throughput on a modern CPU
- [ ] Range query over 24 hours with 1000 series completes in < 100ms
- [ ] Compaction reduces storage by > 20% (tombstone application)

---

## Stretch Goals

- Implement **downsampling**: for data > 30 days old, store only 5-minute averages
- Add **remote read/write** Prometheus compatibility
- Build a **TUI dashboard** showing ingestion rate, query latency, storage size
