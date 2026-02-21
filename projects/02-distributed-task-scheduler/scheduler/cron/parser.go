// Package cron parses cron expressions and computes next execution times.
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schedule represents a parsed cron expression.
type Schedule struct {
	minute field // 0-59
	hour   field // 0-23
	dom    field // 1-31 (day of month)
	month  field // 1-12
	dow    field // 0-6 (day of week, 0=Sunday)
	expr   string
}

// field is a set of allowed values for a cron field.
type field struct {
	values [60]bool // generous size for all fields
}

// Parse parses a 5-field cron expression or shorthand.
// Supports: * , - / and shorthands @hourly @daily @weekly @monthly @yearly
func Parse(expr string) (*Schedule, error) {
	// Handle shorthands
	switch strings.TrimSpace(expr) {
	case "@hourly":
		return Parse("0 * * * *")
	case "@daily", "@midnight":
		return Parse("0 0 * * *")
	case "@weekly":
		return Parse("0 0 * * 0")
	case "@monthly":
		return Parse("0 0 1 * *")
	case "@yearly", "@annually":
		return Parse("0 0 1 1 *")
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields, got %d in %q", len(parts), expr)
	}

	s := &Schedule{expr: expr}
	var err error

	if s.minute, err = parseField(parts[0], 0, 59); err != nil {
		return nil, fmt.Errorf("cron minute: %w", err)
	}
	if s.hour, err = parseField(parts[1], 0, 23); err != nil {
		return nil, fmt.Errorf("cron hour: %w", err)
	}
	if s.dom, err = parseField(parts[2], 1, 31); err != nil {
		return nil, fmt.Errorf("cron dom: %w", err)
	}
	if s.month, err = parseField(parts[3], 1, 12); err != nil {
		return nil, fmt.Errorf("cron month: %w", err)
	}
	if s.dow, err = parseField(parts[4], 0, 6); err != nil {
		return nil, fmt.Errorf("cron dow: %w", err)
	}

	return s, nil
}

// Next returns the next activation time after t.
// Never returns t itself.
func (s *Schedule) Next(t time.Time) time.Time {
	// TODO: implement Next(t) by incrementing fields from minute upward.
	// Algorithm (simplified Quartz approach):
	//   1. Start from t + 1 minute (truncated to minute)
	//   2. Check month → if not active, advance to next active month, reset day/hour/minute
	//   3. Check dom AND dow → advance as needed
	//   4. Check hour → advance as needed
	//   5. Check minute → advance as needed
	//   6. If we've gone more than 4 years into the future, return zero (impossible schedule)
	_ = t
	panic("Next: not yet implemented")
}

func (s *Schedule) String() string { return s.expr }

// ─────────────────────────────────────────────────────────────
// Field Parser
// ─────────────────────────────────────────────────────────────

// parseField parses a single cron field into a set of active values.
// Supported: *, n, n-m, */n, n-m/n, a,b,c (and combinations)
func parseField(expr string, min, max int) (field, error) {
	var f field

	for _, part := range strings.Split(expr, ",") {
		if err := parseFieldPart(part, min, max, &f); err != nil {
			return f, err
		}
	}
	return f, nil
}

func parseFieldPart(part string, min, max int, f *field) error {
	// Step syntax: value/step or */step
	stepStr := ""
	if idx := strings.Index(part, "/"); idx >= 0 {
		stepStr = part[idx+1:]
		part = part[:idx]
	}

	step := 1
	if stepStr != "" {
		var err error
		step, err = strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return fmt.Errorf("invalid step %q", stepStr)
		}
	}

	var rangeMin, rangeMax int

	if part == "*" {
		rangeMin, rangeMax = min, max
	} else if idx := strings.Index(part, "-"); idx >= 0 {
		// Range: n-m
		lo, err1 := strconv.Atoi(part[:idx])
		hi, err2 := strconv.Atoi(part[idx+1:])
		if err1 != nil || err2 != nil {
			return fmt.Errorf("invalid range %q", part)
		}
		if lo < min || hi > max || lo > hi {
			return fmt.Errorf("range %d-%d out of bounds [%d,%d]", lo, hi, min, max)
		}
		rangeMin, rangeMax = lo, hi
	} else {
		// Single value
		n, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid value %q", part)
		}
		if n < min || n > max {
			return fmt.Errorf("value %d out of bounds [%d,%d]", n, min, max)
		}
		rangeMin, rangeMax = n, n
	}

	for v := rangeMin; v <= rangeMax; v += step {
		f.values[v] = true
	}
	return nil
}

func (f *field) active(v int) bool {
	if v >= 0 && v < len(f.values) {
		return f.values[v]
	}
	return false
}

// nextActive returns the smallest value >= v that is active, or -1 if none.
func (f *field) nextActive(v, max int) int {
	for i := v; i <= max; i++ {
		if f.active(i) {
			return i
		}
	}
	return -1
}
