# Challenge 03 — Write-Ahead Log (WAL) with Crash Recovery

## Difficulty: Expert
## Category: Storage Engines · Durability · File I/O

---

## Problem Statement

Every production database (PostgreSQL, RocksDB, etcd) uses a **Write-Ahead Log (WAL)** to
guarantee durability without flushing every write to the main data store. The rule is simple:
*log before you apply*.

Your task is to implement a minimal WAL engine that:

1. Persists log entries to disk **before** they are applied to an in-memory key-value store
2. Assigns each entry a **Log Sequence Number (LSN)** — strictly increasing
3. Supports **checkpointing**: write a checkpoint marker and truncate old segments
4. On startup, **replays** all entries after the last checkpoint to reconstruct state
5. Handles a simulated crash (process kill mid-write) without data corruption

---

## Requirements

| Method | Behaviour |
|---|---|
| `Append(key, value string) (LSN, error)` | Write entry to WAL, increment LSN |
| `Apply(lsn LSN) error` | Apply the entry at LSN to the in-memory map |
| `Checkpoint() error` | Fsync, write checkpoint marker, truncate preceding entries |
| `Recover() error` | Called at startup — replay unapplied entries after last checkpoint |
| `Get(key string) (string, bool)` | Read from in-memory map |

---

## File Format

```
[4 bytes: record length][1 byte: record type][N bytes: payload][4 bytes: CRC32]
Record types: 0x01=DATA, 0x02=CHECKPOINT
Payload for DATA: LSN(8) | key_len(2) | key | value
```

---

## Constraints

- Use only `os`, `encoding/binary`, `hash/crc32`, `sync` from stdlib
- Each segment file is max 4MB — rotate to a new file when full
- Recovery must detect and skip **torn writes** (partial records detected by CRC mismatch)
- WAL must be **fsync'd** before `Append` returns

---

## Hints

1. Open the WAL file with `os.O_APPEND | os.O_SYNC | os.O_CREATE`
2. Write a length-prefixed record; on read, validate CRC before applying
3. The checkpoint marker stores the LSN — on recovery, seek to the most recent checkpoint
4. Model your `Recover()` as: find last checkpoint, then replay all DATA records after it

---

## Acceptance Criteria

- [ ] After crash-simulation (truncate last 3 bytes of WAL file) + restart, state is consistent
- [ ] Checkpointing reduces WAL file size while preserving all applied state
- [ ] Torn writes are skipped without panicking
- [ ] 10,000 appends complete in < 500ms on an SSD

---

## Stretch Goals

- Implement **group commit**: batch multiple appends into a single fsync
- Add **MVCC snapshot**: allow reads at a specific LSN
