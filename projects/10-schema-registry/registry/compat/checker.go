// Package compat implements JSON Schema compatibility checking.
package compat

import (
	"encoding/json"
	"fmt"

	"github.com/Polqt/schemaregistry/registry/schema"
)

// ─────────────────────────────────────────────────────────────
// JSON Schema diff (structural)
// ─────────────────────────────────────────────────────────────

// jsonSchema is a minimal structural representation.
type jsonSchema struct {
	Type                 string                 `json:"type"`
	Properties           map[string]jsonSchema  `json:"properties"`
	Required             []string               `json:"required"`
	AdditionalProperties *bool                  `json:"additionalProperties"`
	Items                *jsonSchema            `json:"items"`
	Enum                 []interface{}          `json:"enum"`
	AllOf                []jsonSchema           `json:"allOf"`
	AnyOf                []jsonSchema           `json:"anyOf"`
	OneOf                []jsonSchema           `json:"oneOf"`
}

func parseJSON(content string) (jsonSchema, error) {
	var s jsonSchema
	if err := json.Unmarshal([]byte(content), &s); err != nil {
		return jsonSchema{}, err
	}
	return s, nil
}

// ─────────────────────────────────────────────────────────────
// Compatibility rules
// ─────────────────────────────────────────────────────────────

// Checker implements schema.CompatChecker for JSON Schema.
type Checker struct{}

// Check validates that proposed is compatible with existing under level.
func (c *Checker) Check(format schema.Format, existing, proposed string, level schema.CompatLevel) error {
	switch format {
	case schema.FormatJSONSchema, schema.FormatAvro:
		return c.checkJSON(existing, proposed, level)
	case schema.FormatProtobuf:
		// TODO: implement Protobuf field addition/removal checks.
		return nil
	default:
		return fmt.Errorf("compat check not implemented for format %q", format)
	}
}

func (c *Checker) checkJSON(existing, proposed string, level schema.CompatLevel) error {
	eSchema, err := parseJSON(existing)
	if err != nil {
		return fmt.Errorf("parse existing schema: %w", err)
	}
	pSchema, err := parseJSON(proposed)
	if err != nil {
		return fmt.Errorf("parse proposed schema: %w", err)
	}

	switch level {
	case schema.CompatBackward, schema.CompatBackwardTransitive:
		// New schema can read data written by old schema.
		// Rules:
		//   - No required fields added in proposed that don't exist in existing.
		//   - No fields removed from proposed that were in existing.
		return c.checkBackward(eSchema, pSchema)

	case schema.CompatForward, schema.CompatForwardTransitive:
		// Old schema can read data written by new schema.
		// Rules:
		//   - No fields added to proposed without defaults (additionalProperties).
		//   - No fields removed from existing that were not optional.
		return c.checkForward(eSchema, pSchema)

	case schema.CompatFull, schema.CompatFullTransitive:
		if err := c.checkBackward(eSchema, pSchema); err != nil {
			return err
		}
		return c.checkForward(eSchema, pSchema)

	case schema.CompatNone:
		return nil
	}
	return nil
}

// checkBackward: proposed must be able to deserialise data written by existing.
func (c *Checker) checkBackward(existing, proposed jsonSchema) error {
	// Check that all required fields in proposed also exist in existing.
	for _, reqField := range proposed.Required {
		if _, ok := existing.Properties[reqField]; !ok {
			return fmt.Errorf(
				"BACKWARD incompatibility: proposed adds required field %q which is absent in existing", reqField)
		}
	}

	// Check that no field type is changed to an incompatible type.
	for name, pProp := range proposed.Properties {
		eProp, ok := existing.Properties[name]
		if !ok {
			continue // new optional field — allowed in backward compat
		}
		if pProp.Type != "" && eProp.Type != "" && pProp.Type != eProp.Type {
			return fmt.Errorf(
				"BACKWARD incompatibility: field %q type changed from %q to %q", name, eProp.Type, pProp.Type)
		}
	}
	return nil
}

// checkForward: existing must be able to deserialise data written by proposed.
func (c *Checker) checkForward(existing, proposed jsonSchema) error {
	// TODO: check that any new required field in proposed has a default/fallback.
	for _, reqField := range proposed.Required {
		if _, ok := existing.Properties[reqField]; !ok {
			// New required field that existing schema doesn't know about.
			return fmt.Errorf(
				"FORWARD incompatibility: proposed adds required field %q unknown to existing", reqField)
		}
	}
	return nil
}

// Diff returns a human-readable summary of changes between two JSON schemas.
func Diff(existing, proposed string) ([]string, error) {
	eSchema, err := parseJSON(existing)
	if err != nil {
		return nil, err
	}
	pSchema, err := parseJSON(proposed)
	if err != nil {
		return nil, err
	}

	var changes []string

	// Added fields.
	for name := range pSchema.Properties {
		if _, ok := eSchema.Properties[name]; !ok {
			changes = append(changes, fmt.Sprintf("+ field added: %q", name))
		}
	}
	// Removed fields.
	for name := range eSchema.Properties {
		if _, ok := pSchema.Properties[name]; !ok {
			changes = append(changes, fmt.Sprintf("- field removed: %q", name))
		}
	}
	// Changed types.
	for name, pProp := range pSchema.Properties {
		if eProp, ok := eSchema.Properties[name]; ok {
			if pProp.Type != eProp.Type && pProp.Type != "" && eProp.Type != "" {
				changes = append(changes, fmt.Sprintf("~ field %q type changed: %q → %q", name, eProp.Type, pProp.Type))
			}
		}
	}
	// Required field changes.
	eReq := toSet(eSchema.Required)
	pReq := toSet(pSchema.Required)
	for f := range pReq {
		if !eReq[f] {
			changes = append(changes, fmt.Sprintf("! field %q became required", f))
		}
	}
	for f := range eReq {
		if !pReq[f] {
			changes = append(changes, fmt.Sprintf("! field %q became optional", f))
		}
	}

	return changes, nil
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
