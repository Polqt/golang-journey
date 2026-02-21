package main

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"sync"
)

// ============================================================
// CHALLENGE 03: Write-Ahead Log with Crash Recovery
// ============================================================
// Implement a durable WAL that guarantees write-before-apply
// semantics and can reconstruct in-memory state after a crash.
//
// READ THE README.md BEFORE STARTING.
// ============================================================

// LSN is a Log Sequence Number — uniquely identifies a WAL entry.
type LSN uint64

const (
	RecordTypeData       byte = 0x01
	RecordTypeCheckpoint byte = 0x02
	MaxSegmentSize            = 4 * 1024 * 1024 // 4MB
)

// WALEntry represents a decoded log record.
type WALEntry struct {
	LSN   LSN
	Type  byte
	Key   string
	Value string
}

// TODO: Define WAL struct:
//   - dir string (directory for segment files)
//   - mu sync.Mutex
//   - currentFile *os.File
//   - currentSize int64
//   - nextLSN LSN
//   - store map[string]string (in-memory KV)
//   - appliedUpTo LSN (highest applied LSN)

// NewWAL creates a new WAL in the given directory (creates if not exists).
func NewWAL(dir string) (*WAL, error) {
	panic("implement me")
}

// Append writes a new DATA record to the WAL and returns its LSN.
// Must fsync before returning.
func (w *WAL) Append(key, value string) (LSN, error) {
	panic("implement me")
}

// Apply marks the entry at lsn as applied and updates the in-memory store.
func (w *WAL) Apply(lsn LSN) error {
	panic("implement me")
}

// Checkpoint writes a CHECKPOINT record, fsyncs, then truncates all
// WAL entries prior to the checkpoint in older segments.
func (w *WAL) Checkpoint() error {
	panic("implement me")
}

// Recover replays all DATA entries after the last CHECKPOINT into the
// in-memory store. Must handle torn writes (CRC mismatch) gracefully.
func (w *WAL) Recover() error {
	panic("implement me")
}

// Get reads a key from the in-memory store.
func (w *WAL) Get(key string) (string, bool) {
	panic("implement me")
}

// Close flushes and closes the WAL file.
func (w *WAL) Close() error {
	panic("implement me")
}

// ============================================================
// Helpers (feel free to use or discard)
// ============================================================

// encodeRecord encodes a WAL record with a CRC32 footer.
// Format: [4-byte length][1-byte type][payload][4-byte crc32]
func encodeRecord(recordType byte, payload []byte) []byte {
	length := uint32(1 + len(payload) + 4)
	buf := make([]byte, 4+1+len(payload)+4)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = recordType
	copy(buf[5:], payload)
	checksum := crc32.ChecksumIEEE(buf[4 : 5+len(payload)])
	binary.BigEndian.PutUint32(buf[5+len(payload):], checksum)
	return buf
}

// ============================================================
// Scaffolding — do not modify
// ============================================================

// WAL — stub; replace with your implementation.
type WAL struct {
	mu    sync.Mutex
	dir   string
	store map[string]string
}

func main() {
	fmt.Println("=== Write-Ahead Log with Crash Recovery ===")

	dir := os.TempDir() + "/wal-test"
	os.RemoveAll(dir)

	// --- Normal write + apply cycle ---
	wal, err := NewWAL(dir)
	mustNil(err)

	lsns := make([]LSN, 5)
	keys := []string{"a", "b", "c", "d", "e"}
	vals := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	for i, k := range keys {
		lsns[i], err = wal.Append(k, vals[i])
		mustNil(err)
	}
	for _, lsn := range lsns {
		mustNil(wal.Apply(lsn))
	}
	v, ok := wal.Get("c")
	fmt.Printf("Get(c) = %q, found=%v (expect gamma, true)\n", v, ok)

	// --- Checkpoint ---
	mustNil(wal.Checkpoint())

	// --- Append more after checkpoint ---
	lsn6, _ := wal.Append("f", "zeta")
	wal.Apply(lsn6)
	mustNil(wal.Close())

	// --- Simulate crash: corrupt last 3 bytes of last segment ---
	files, _ := filepath("wal-test")
	if len(files) > 0 {
		f, _ := os.OpenFile(files[len(files)-1], os.O_RDWR, 0644)
		stat, _ := f.Stat()
		f.Truncate(stat.Size() - 3)
		f.Close()
		fmt.Println("Simulated torn write (truncated last 3 bytes)")
	}

	// --- Recovery ---
	wal2, err := NewWAL(dir)
	mustNil(err)
	mustNil(wal2.Recover())

	for _, k := range keys {
		v, ok := wal2.Get(k)
		fmt.Printf("Recovered Get(%s) = %q, found=%v\n", k, v, ok)
	}
	mustNil(wal2.Close())
	os.RemoveAll(dir)

	fmt.Println("Done.")
	_ = encodeRecord // silence unused warning if not yet used
}

// filepath is a helper that returns segment files in the WAL dir.
func filepath(dir string) ([]string, error) {
	entries, err := os.ReadDir(os.TempDir() + "/" + dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		paths = append(paths, os.TempDir()+"/"+dir+"/"+e.Name())
	}
	return paths, nil
}

func mustNil(err error) {
	if err != nil {
		panic(err)
	}
}
