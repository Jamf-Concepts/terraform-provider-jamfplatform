// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package aischemas

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseAcceptsAConstraintlessSchema pins the two bodies that are allowed to become an
// accept-everything Document: an empty body, and draft-07's bare `true`. Neither declares a
// constraint, so neither is grounds for failing a plan the service would accept.
func TestParseAcceptsAConstraintlessSchema(t *testing.T) {
	for _, raw := range []string{"", "true"} {
		document, err := Parse("com.example.tool", "2026-01-01", json.RawMessage(raw))
		if err != nil {
			t.Fatalf("Parse(%q): %v", raw, err)
		}
		if problems := document.Validate(map[string]any{"anything": 1}); len(problems) != 0 {
			t.Errorf("Parse(%q) must accept everything, got %d problems", raw, len(problems))
		}
	}
}

// TestParseRejectsAFalseSchema pins the shape whose accept-everything reading is its exact opposite:
// draft-07's bare `false` refuses every payload, so silently treating it as no constraints at all
// would validate nothing while reporting that it had.
func TestParseRejectsAFalseSchema(t *testing.T) {
	_, err := Parse("com.example.tool", "2026-01-01", json.RawMessage("false"))
	if err == nil {
		t.Fatal("a `false` schema must be an error, not an accept-everything document")
	}
	if !strings.Contains(err.Error(), "false") {
		t.Errorf("the error must name the shape it refused: %v", err)
	}
}

// TestParseRejectsANonObjectSchema pins that every remaining root travels the error path rather than
// the accept-everything one. A double-encoded schema — a JSON string holding the document — is the
// realistic case, and the one an accept-everything reading would hide for the provider's lifetime.
func TestParseRejectsANonObjectSchema(t *testing.T) {
	cases := map[string]string{
		"array":          `[{"type":"object"}]`,
		"number":         `7`,
		"null":           `null`,
		"double-encoded": `"{\"type\":\"object\"}"`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			document, err := Parse("com.example.tool", "2026-01-01", json.RawMessage(raw))
			if err == nil {
				t.Fatalf("Parse(%s) must fail, got a document (root nil: %t)", raw, document.root == nil)
			}
			if !strings.Contains(err.Error(), "com.example.tool") || !strings.Contains(err.Error(), "2026-01-01") {
				t.Errorf("the error must name the tool and schema version: %v", err)
			}
		})
	}
}
