// Package head provides the in-memory write head for the time-series database.
// It stores the most recent data points using Gorilla XOR compression.
package head

import (
	"fmt"
	"math"
	"math/bits"
	"sync"
)

// ─────────────────────────────────────────────────────────────
// Gorilla XOR Chunk (Facebook 2015 paper — Section 4.1)
// ─────────────────────────────────────────────────────────────
// Timestamps: delta-of-delta encoded, zigzag, variable-length.
// Values:     XOR with previous; if XOR==0, bit 0; else control bits
//             for (leading zeros, meaningful bits length, payload).

const (
	firstDeltaBits = 14   // first timestamp delta stored in 14 bits
	maxChunkBytes  = 1024 // tunable: flush block when full
)

// Chunk is a single compressed block of (timestamp, value) pairs.
type Chunk struct {
	mu sync.RWMutex

	buf    []byte
	bitPos int // current write position in bits

	numSamples int

	// Previous values for delta and XOR encoding.
	tLast      int64
	tDeltaLast int64
	vLast      uint64
	vLeadLast  int
	vMeanLast  int
}

// NewChunk allocates a fresh chunk.
func NewChunk() *Chunk {
	return &Chunk{buf: make([]byte, 0, maxChunkBytes)}
}

// Append adds a (timestamp ms, value) pair to the chunk.
// Must be called with monotonically increasing timestamps.
func (c *Chunk) Append(ts int64, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.numSamples == 0 {
		c.appendFirst(ts, v)
	} else if c.numSamples == 1 {
		c.appendSecond(ts, v)
	} else {
		c.appendDelta(ts, v)
	}
	c.numSamples++
}

func (c *Chunk) appendFirst(ts int64, v float64) {
	// Store full timestamp and value (64 bits each).
	c.writeBits(uint64(ts), 64)
	c.writeBits(math.Float64bits(v), 64)
	c.tLast = ts
	c.vLast = math.Float64bits(v)
}

func (c *Chunk) appendSecond(ts int64, v float64) {
	delta := ts - c.tLast
	c.writeBits(uint64(delta), firstDeltaBits)
	c.appendValue(v)
	c.tDeltaLast = delta
	c.tLast = ts
}

func (c *Chunk) appendDelta(ts int64, v float64) {
	// Timestamp: delta of delta + zigzag encoding.
	delta := ts - c.tLast
	dod := delta - c.tDeltaLast // delta-of-delta
	c.encodeTimestampDoD(dod)
	c.tDeltaLast = delta
	c.tLast = ts

	// Value: XOR encoding.
	c.appendValue(v)
}

// encodeTimestampDoD encodes delta-of-delta using variable-length blocks per the paper.
func (c *Chunk) encodeTimestampDoD(dod int64) {
	// TODO: implement Gorilla timestamp encoding (Table 1 in paper):
	//   dod == 0               → bit "0"
	//   fits in [-64,63]       → "10" + 7-bit zigzag
	//   fits in [-256,255]     → "110" + 9-bit zigzag
	//   fits in [-2048,2047]   → "1110" + 12-bit zigzag
	//   fits in [-65536,65535] → "11110" + 16-bit zigzag
	//   else                   → "11111" + 64-bit zigzag
	panic("Chunk.encodeTimestampDoD: not yet implemented")
}

// appendValue XOR-compresses a float64 against the previous value.
func (c *Chunk) appendValue(v float64) {
	vBits := math.Float64bits(v)
	xor := c.vLast ^ vBits
	c.vLast = vBits

	if xor == 0 {
		c.writeBit(0)
		return
	}
	c.writeBit(1)

	// TODO: implement Gorilla value encoding (Section 4.1.2):
	//   leading := bits.LeadingZeros64(xor)
	//   trailing := bits.TrailingZeros64(xor)
	//   meaningful := 64 - leading - trailing
	//   if leading >= c.vLeadLast && trailing >= 64 - c.vLeadLast - c.vMeanLast:
	//       bit "0": reuse previous leading/meaningful window
	//   else:
	//       bit "1": write new 5-bit leading, 6-bit meaningful, then payload
	_ = bits.LeadingZeros64(xor)
	panic("Chunk.appendValue: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// Bit-level I/O helpers
// ─────────────────────────────────────────────────────────────

func (c *Chunk) writeBit(b uint8) {
	if c.bitPos%8 == 0 {
		c.buf = append(c.buf, 0)
	}
	byteIdx := c.bitPos / 8
	bitOff := 7 - c.bitPos%8
	if b != 0 {
		c.buf[byteIdx] |= 1 << uint(bitOff)
	}
	c.bitPos++
}

func (c *Chunk) writeBits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		c.writeBit(uint8((v >> uint(i)) & 1))
	}
}

// ─────────────────────────────────────────────────────────────
// Iteration
// ─────────────────────────────────────────────────────────────

// Iterator reads samples from a frozen chunk.
type Iterator struct {
	c      *Chunk
	bitPos int
	idx    int

	tLast      int64
	tDeltaLast int64
	vLast      uint64
	vLeadLast  int
	vMeanLast  int

	CurTs  int64
	CurVal float64
	Err    error
}

// NewIterator returns a fresh iterator at the start of the chunk.
func (c *Chunk) NewIterator() *Iterator {
	return &Iterator{c: c}
}

// Next advances to the next sample; returns false when exhausted.
func (it *Iterator) Next() bool {
	if it.idx >= it.c.numSamples {
		return false
	}
	// TODO: implement delta-of-delta and XOR decoding (mirror of Append)
	it.Err = fmt.Errorf("Iterator.Next: not yet implemented")
	return false
}
