// Package schema defines the core schema types and the in-memory registry.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────
// Schema types
// ─────────────────────────────────────────────────────────────

// Format identifies the schema language.
type Format string

const (
	FormatJSONSchema Format = "json-schema"
	FormatAvro       Format = "avro"
	FormatProtobuf   Format = "protobuf"
)

// Schema is one immutable schema version.
type Schema struct {
	Subject  string    // logical name, e.g. "payments.v1"
	Version  int       // monotonically increasing per subject
	ID       int       // globally unique ID across all subjects
	Format   Format
	Content  string    // raw schema text
	Hash     string    // SHA-256 of Content (for deduplication)
	Created  time.Time
	Metadata map[string]string // arbitrary key-value tags
}

// ─────────────────────────────────────────────────────────────
// Compatibility levels
// ─────────────────────────────────────────────────────────────

// CompatLevel defines the compatibility rule for a subject.
type CompatLevel string

const (
	CompatNone           CompatLevel = "NONE"
	CompatBackward       CompatLevel = "BACKWARD"
	CompatForward        CompatLevel = "FORWARD"
	CompatFull           CompatLevel = "FULL"
	CompatBackwardTransitive CompatLevel = "BACKWARD_TRANSITIVE"
	CompatForwardTransitive  CompatLevel = "FORWARD_TRANSITIVE"
	CompatFullTransitive     CompatLevel = "FULL_TRANSITIVE"
)

// ─────────────────────────────────────────────────────────────
// Errors
// ─────────────────────────────────────────────────────────────

var (
	ErrSubjectNotFound    = errors.New("subject not found")
	ErrSchemaNotFound     = errors.New("schema not found")
	ErrIncompatibleSchema = errors.New("schema is not compatible")
	ErrInvalidSchema      = errors.New("invalid schema")
)

// ─────────────────────────────────────────────────────────────
// Registry
// ─────────────────────────────────────────────────────────────

// subjectEntry holds all versions of one subject.
type subjectEntry struct {
	versions []*Schema // index 0 = version 1
	compat   CompatLevel
}

// Registry is the in-memory schema registry.
type Registry struct {
	mu       sync.RWMutex
	subjects map[string]*subjectEntry // subject → entry
	byID     map[int]*Schema          // global id → schema
	nextID   int
	checker  CompatChecker
}

// CompatChecker validates a proposed schema against an existing one.
type CompatChecker interface {
	Check(format Format, existing, proposed string, level CompatLevel) error
}

// New creates a Registry with the given compatibility checker.
func New(checker CompatChecker) *Registry {
	return &Registry{
		subjects: make(map[string]*subjectEntry),
		byID:     make(map[int]*Schema),
		nextID:   1,
		checker:  checker,
	}
}

// Register registers a new schema version for the subject.
// Returns the schema ID, or error if compatibility check fails.
func (r *Registry) Register(subject string, format Format, content string) (int, error) {
	if err := validateSyntax(format, content); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidSchema, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.subjects[subject]
	if !ok {
		entry = &subjectEntry{compat: CompatBackward}
		r.subjects[subject] = entry
	}

	// Deduplication: if content matches latest, return existing ID.
	if len(entry.versions) > 0 {
		latest := entry.versions[len(entry.versions)-1]
		if latest.Content == content {
			return latest.ID, nil
		}
		// Compatibility check.
		if r.checker != nil && entry.compat != CompatNone {
			if err := r.checker.Check(format, latest.Content, content, entry.compat); err != nil {
				return 0, fmt.Errorf("%w: %v", ErrIncompatibleSchema, err)
			}
		}
	}

	id := r.nextID
	r.nextID++
	s := &Schema{
		Subject: subject,
		Version: len(entry.versions) + 1,
		ID:      id,
		Format:  format,
		Content: content,
		Hash:    contentHash(content),
		Created: time.Now(),
	}
	entry.versions = append(entry.versions, s)
	r.byID[id] = s
	return id, nil
}

// GetByID returns the schema with the given global ID.
func (r *Registry) GetByID(id int) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byID[id]
	if !ok {
		return nil, ErrSchemaNotFound
	}
	return s, nil
}

// GetLatest returns the latest version of a subject.
func (r *Registry) GetLatest(subject string) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.subjects[subject]
	if !ok {
		return nil, ErrSubjectNotFound
	}
	return entry.versions[len(entry.versions)-1], nil
}

// GetVersion returns a specific version for a subject.
func (r *Registry) GetVersion(subject string, version int) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.subjects[subject]
	if !ok {
		return nil, ErrSubjectNotFound
	}
	if version < 1 || version > len(entry.versions) {
		return nil, ErrSchemaNotFound
	}
	return entry.versions[version-1], nil
}

// Subjects returns all subject names.
func (r *Registry) Subjects() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	subs := make([]string, 0, len(r.subjects))
	for name := range r.subjects {
		subs = append(subs, name)
	}
	return subs
}

// Versions returns all version numbers for a subject.
func (r *Registry) Versions(subject string) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.subjects[subject]
	if !ok {
		return nil, ErrSubjectNotFound
	}
	vers := make([]int, len(entry.versions))
	for i := range entry.versions {
		vers[i] = i + 1
	}
	return vers, nil
}

// SetCompatLevel updates the compatibility level for a subject.
func (r *Registry) SetCompatLevel(subject string, level CompatLevel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.subjects[subject]
	if !ok {
		return ErrSubjectNotFound
	}
	entry.compat = level
	return nil
}

// CompatLevel returns the current compatibility level for a subject.
func (r *Registry) CompatLevel(subject string) (CompatLevel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.subjects[subject]
	if !ok {
		return "", ErrSubjectNotFound
	}
	return entry.compat, nil
}

// Delete removes a subject and all its versions.
func (r *Registry) Delete(subject string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.subjects[subject]
	if !ok {
		return ErrSubjectNotFound
	}
	for _, s := range entry.versions {
		delete(r.byID, s.ID)
	}
	delete(r.subjects, subject)
	return nil
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

// validateSyntax does a lightweight syntax check of the schema content.
func validateSyntax(format Format, content string) error {
	switch format {
	case FormatJSONSchema:
		var v interface{}
		return json.Unmarshal([]byte(content), &v)
	case FormatAvro:
		// TODO: parse Avro JSON schema — at minimum validate it's valid JSON
		var v interface{}
		return json.Unmarshal([]byte(content), &v)
	case FormatProtobuf:
		// TODO: parse .proto file syntax (requires a proto parser)
		if len(content) == 0 {
			return fmt.Errorf("empty protobuf schema")
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

// contentHash returns a hex SHA-256 of content.
func contentHash(content string) string {
	// TODO: hash with crypto/sha256
	// For now return a length-based placeholder.
	return fmt.Sprintf("sha256:%d", len(content))
}
