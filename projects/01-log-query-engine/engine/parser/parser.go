// Package parser provides log format parsers.
// Supported formats: NDJSON, logfmt, CSV.
package parser

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Row is a parsed log line: a map from field name to typed value.
// Supported value types: string, int64, float64, bool, time.Time.
type Row map[string]any

// Parser is the interface all format parsers implement.
type Parser interface {
	Parse(line []byte) (Row, error)
	// Name returns the format name for error messages.
	Name() string
}

// ─────────────────────────────────────────────────────────────
// NDJSON Parser
// ─────────────────────────────────────────────────────────────

// NDJSONParser parses newline-delimited JSON logs.
type NDJSONParser struct{}

func (p *NDJSONParser) Name() string { return "ndjson" }

func (p *NDJSONParser) Parse(line []byte) (Row, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("ndjson parse: %w", err)
	}
	row := make(Row, len(raw))
	for k, v := range raw {
		row[k] = coerce(v)
	}
	return row, nil
}

// ─────────────────────────────────────────────────────────────
// Logfmt Parser
// ─────────────────────────────────────────────────────────────

// LogfmtParser parses logfmt-style lines: key=value key="quoted value"
type LogfmtParser struct{}

func (p *LogfmtParser) Name() string { return "logfmt" }

func (p *LogfmtParser) Parse(line []byte) (Row, error) {
	// TODO: implement logfmt tokenizer
	// Rules:
	//   - Split tokens on whitespace
	//   - Each token is either "key=value" or "key=\"quoted value\""
	//   - Bare key (no =) → key: true
	//   - Values are coerced to int64/float64/bool if possible
	_ = line
	return nil, fmt.Errorf("logfmt parser: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// CSV Parser
// ─────────────────────────────────────────────────────────────

// CSVParser parses CSV files where the first line is a header.
type CSVParser struct {
	headers []string
}

// NewCSVParser reads the header line from r and returns a ready parser.
func NewCSVParser(r io.Reader) (*CSVParser, error) {
	cr := csv.NewReader(bufio.NewReader(r))
	headers, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("csv header: %w", err)
	}
	return &CSVParser{headers: headers}, nil
}

func (p *CSVParser) Name() string { return "csv" }

func (p *CSVParser) Parse(line []byte) (Row, error) {
	// TODO: parse CSV line using csv.NewReader(bytes.NewReader(line))
	// Map headers[i] → coerce(record[i])
	_ = line
	return nil, fmt.Errorf("csv parser: not yet implemented")
}

// ─────────────────────────────────────────────────────────────
// Format Detection
// ─────────────────────────────────────────────────────────────

// DetectFormat examines the first non-empty line and returns the best parser.
func DetectFormat(sample []byte) Parser {
	line := bytes.TrimSpace(sample)
	if len(line) == 0 {
		return &NDJSONParser{}
	}
	if line[0] == '{' {
		return &NDJSONParser{}
	}
	if bytes.Contains(line, []byte(",")) && !bytes.Contains(line, []byte("=")) {
		// Heuristic: commas without = signs suggests CSV
		// (caller must create CSVParser with header separately)
		return &NDJSONParser{} // fallback; caller should handle CSV specially
	}
	if bytes.Contains(line, []byte("=")) {
		return &LogfmtParser{}
	}
	return &NDJSONParser{}
}

// ─────────────────────────────────────────────────────────────
// Type Coercion
// ─────────────────────────────────────────────────────────────

// coerce converts JSON-decoded interface{} values to their best Go type.
func coerce(v any) any {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return int64(t)
		}
		return t
	case string:
		// Try int
		if i, err := strconv.ParseInt(t, 10, 64); err == nil {
			return i
		}
		// Try float
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f
		}
		// Try bool
		if b, err := strconv.ParseBool(t); err == nil {
			return b
		}
		// Try time (common log timestamp formats)
		for _, layout := range []string{
			time.RFC3339Nano, time.RFC3339,
			"2006-01-02T15:04:05", "2006-01-02 15:04:05",
			"02/Jan/2006:15:04:05 -0700",
		} {
			if ts, err := time.Parse(layout, t); err == nil {
				return ts
			}
		}
		return t
	default:
		return v
	}
}

// CoerceString coerces a raw string token (used by logfmt/csv).
func CoerceString(s string) any {
	return coerce(strings.TrimSpace(s))
}
