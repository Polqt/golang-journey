# Project 10 — Schema Registry with Compatibility Engine

> **Difficulty**: Senior · **Domain**: Protocol Design, API Design, Data Engineering
> **Real-world analog**: Confluent Schema Registry, AWS Glue Schema Registry, Buf.build

---

## Why This Project Exists

Schema evolution is one of the hardest problems in data engineering. Adding a field to a
Protobuf message sounds harmless — until an old consumer reads a message written by a new
producer and crashes. **Schema registries** prevent this by enforcing compatibility rules
before a new schema version is deployed. Confluent's Schema Registry serves millions of
Kafka schemas per day.

---

## Folder Structure

```
10-schema-registry/
├── go.mod
├── main.go                              # CLI: serve, register, validate, diff
├── registry/
│   ├── store/
│   │   ├── store.go                     # Schema store interface
│   │   └── sqlite.go                    # SQLite-backed persistence
│   ├── schema/
│   │   ├── schema.go                    # Schema type definition + version
│   │   ├── json_schema.go               # JSON Schema validator (RFC draft-07)
│   │   └── protobuf.go                  # Proto descriptor parsing + validation
│   ├── compat/
│   │   ├── checker.go                   # Compatibility check orchestrator
│   │   ├── json_compat.go               # JSON Schema compatibility rules
│   │   └── proto_compat.go              # Protobuf field evolution rules
│   └── subject.go                       # Subject = (topic, schema_type) namespace
├── api/
│   ├── v1/
│   │   ├── subjects.go                  # Subject CRUD endpoints
│   │   ├── schemas.go                   # Schema register + lookup endpoints
│   │   └── compat.go                    # Compatibility check endpoint
│   └── middleware.go                    # Auth, rate limit, CORS
├── diff/
│   ├── diff.go                          # Schema diff computation
│   └── render.go                        # Human-readable diff output
├── cli/
│   ├── register.go                      # CLI: register a schema file
│   ├── validate.go                      # CLI: validate a message against schema
│   └── diff.go                          # CLI: diff two schema versions
└── config/
    └── config.yaml
```

---

## Implementation Guide

### Phase 1 — Schema Storage Model (Week 1)

```go
type SchemaType string
const (
    SchemaTypeJSON     SchemaType = "JSON"
    SchemaTypeProtobuf SchemaType = "PROTOBUF"
    SchemaTypeAvro     SchemaType = "AVRO"
)

type Schema struct {
    ID         int64      // global, auto-incrementing
    Subject    string     // e.g. "orders-value", "payments-key"
    Version    int        // per-subject version, starts at 1
    SchemaType SchemaType
    Definition string     // raw schema text (JSON or .proto)
    CreatedAt  time.Time
}

// Subjects namespace schemas: one subject per Kafka topic+key/value
type Subject struct {
    Name            string
    CompatibilityLevel CompatibilityLevel
    Schemas         []Schema
}
```

**Database schema** (SQLite):
```sql
CREATE TABLE schemas (
    id INTEGER PRIMARY KEY,
    subject TEXT NOT NULL,
    version INTEGER NOT NULL,
    schema_type TEXT NOT NULL,
    definition TEXT NOT NULL,
    fingerprint TEXT NOT NULL,  -- SHA256 of canonical definition
    created_at DATETIME NOT NULL,
    UNIQUE(subject, version)
);
CREATE TABLE subjects (
    name TEXT PRIMARY KEY,
    compatibility TEXT NOT NULL DEFAULT 'BACKWARD'
);
```

---

### Phase 2 — JSON Schema Validator (Week 1-2)

Implement a **JSON Schema (draft-07) validator** from scratch (no `jsonschema` library).

Core subset to support:
- `type`: string, number, integer, boolean, array, object, null
- `properties` + `required`
- `additionalProperties`
- `items` (array element schema)
- `$ref` (local reference resolution: `#/definitions/...`)
- `oneOf`, `anyOf`, `allOf`
- `minimum`, `maximum`, `minLength`, `maxLength`, `pattern`
- `enum`

```go
type Validator struct {
    root map[string]any  // parsed JSON Schema
}
func NewValidator(schema string) (*Validator, error)
func (v *Validator) Validate(data any) []ValidationError
type ValidationError struct {
    Path    string  // JSON pointer: "/items/0/name"
    Message string
}
```

---

### Phase 3 — Compatibility Checker (Week 2-3)

This is the **core value** of the registry. Implement four compatibility levels:

| Level | Rule |
|---|---|
| `BACKWARD` | New schema can **read** data written with old schema (consumers upgrade first) |
| `FORWARD` | Old schema can **read** data written with new schema (producers upgrade first) |
| `FULL` | Both BACKWARD + FORWARD simultaneously |
| `NONE` | No compatibility checks |

**JSON Schema compatibility rules**:

| Change | BACKWARD | FORWARD |
|---|---|---|
| Add optional field | ✅ allowed | ❌ breaks |
| Add required field | ❌ breaks | ✅ allowed |
| Remove field | ❌ breaks | ✅ allowed |
| Change field type | ❌ breaks (both) | ❌ breaks (both) |
| Broaden type (int→number) | ✅ | ❌ |
| Narrow type (number→int) | ❌ | ✅ |

**Protobuf compatibility rules**:
- Field number must not be reused (ever)
- Field type changes only allowed within compatible wire types
- Repeated→singular or singular→repeated is breaking
- Adding a field is always backward compatible (proto3 defaults)
- Renaming a field is safe (proto uses numbers, not names)

---

### Phase 4 — REST API (Week 3)

Confluent-compatible API:

```
# Register a new schema version
POST /subjects/{subject}/versions
Body: {"schemaType":"JSON","schema":"{...}"}
Response: {"id":42}

# Get the latest schema for a subject
GET /subjects/{subject}/versions/latest
Response: {"subject":"orders-value","version":3,"id":42,"schema":"{...}"}

# Check compatibility before registering
POST /compatibility/subjects/{subject}/versions/latest
Body: {"schema":"{...}"}
Response: {"is_compatible":true}

# List all subjects
GET /subjects

# List versions for a subject
GET /subjects/{subject}/versions

# Get schema by global ID (for producers/consumers to look up by ID)  
GET /schemas/ids/{id}
```

---

### Phase 5 — Schema Diff + CLI (Week 4)

```bash
# Register a new schema
$ schema-registry register --subject orders-value --file schema.json
Registered schema version 3 (ID: 42)

# Check compatibility before registering
$ schema-registry check --subject orders-value --file schema.json
✅ BACKWARD compatible with version 2

# Show what changed
$ schema-registry diff --subject orders-value --from 1 --to 3
+ properties.shipping_address (object, optional)
+ properties.discount_code (string, optional)
~ properties.price type: integer → number (BACKWARD compatible)
- properties.legacy_field (REMOVED — BREAKING if BACKWARD mode!)

# Validate a message against the latest schema
$ echo '{"id":1,"price":9.99}' | schema-registry validate --subject orders-value
❌ ValidationError: /required: missing field "customer_id"
```

---

### Phase 6 — Schema Fingerprinting + Deduplication (Week 4)

A **fingerprint** is the canonical SHA-256 of a schema after normalization (sorted keys,
removed whitespace). This allows:
1. Detecting duplicate registrations (return existing ID if fingerprint matches)
2. Efficient equality checks without full text comparison

JSON Schema normalization:
1. Parse to `map[string]any`
2. Recursively sort all object keys alphabetically
3. Re-encode with no extra whitespace

---

## Acceptance Criteria

- [ ] JSON Schema validation correctly rejects 20+ standard test vectors
- [ ] Compatibility checker correctly identifies all breaking changes in the table above
- [ ] Protobuf field number reuse is detected and rejected
- [ ] 10,000 schema lookups/sec (schemas cached after first load)
- [ ] Fingerprint deduplication prevents duplicate schemas

---

## Stretch Goals

- Implement **Avro schema** support (avro.apache.org/docs/current/specification.html)
- Add **schema migration scripts**: auto-generate migration code when a breaking change is allowed with `NONE` mode
- Build a **TUI schema browser**: browse subjects, versions, diff interactively
- Implement **GraphQL schema compatibility** (field deprecation, type widening rules)
