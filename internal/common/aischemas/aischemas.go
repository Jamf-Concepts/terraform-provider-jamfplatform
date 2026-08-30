// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package aischemas validates an AI Governance policy's settings against the vendor JSON Schema the
// Jamf Platform serves for that product and schema version, so a payload the service would refuse —
// or worse, accept and never apply — is caught during `terraform plan`.
//
// The schemas are fetched live rather than snapshotted, because they move: Claude Code shipped two
// schema versions in three months, and the service exposes a schemaDrift flag precisely to nudge
// admins off stale ones. A snapshot would go wrong in the direction that produces false errors.
//
// The check is deliberately partial, and the partiality is measured rather than assumed. Across the
// three products the service serves today, every `$ref` is local (no network fetch during
// validation), every `pattern` is RE2-compatible, and the composition load is `allOf` used as a
// single-element `$ref` wrapper. The keywords below are implemented; the rest are skipped, and their
// absence is a documented choice, not an oversight:
//
//   - if/then/else and not: conditional application. Five sites across all three schemas, none of
//     them a typo-catcher.
//   - format: an annotation in draft-07, not an assertion.
//   - patternProperties, dependencies, contains, multipleOf, minProperties, maxProperties: absent
//     from every schema the service serves.
//
// Full draft-07 compliance is unreachable for any Go validator here regardless: draft-07 specifies
// ECMA-262 regular expressions and Go's regexp is RE2, which has no lookaround or backreferences. A
// pattern that will not compile is skipped rather than guessed at.
//
// The service remains the authority. This package exists to move a failure earlier, and to catch the
// one failure the service never reports: an undeclared key under a schema that accepts extra keys is
// stored verbatim and silently never applied. See Problem.Advisory for how far to trust each finding.
package aischemas

import (
	"encoding/json"
	"fmt"
)

// maxRefDepth bounds `$ref` expansion. The Codex schema is recursive, so a cycle that never
// descends into a value would otherwise not terminate; the per-value visited set catches the common
// case and this catches the rest.
const maxRefDepth = 64

// Document is a parsed vendor JSON Schema for one product at one schema version.
type Document struct {
	toolID        string
	schemaVersion string
	root          map[string]any
}

// Parse decodes a vendor schema document. The toolID and schemaVersion are carried only so
// diagnostics can name what the settings were checked against.
//
// A schema that is not a JSON object — draft-07 permits a bare `true`/`false` schema — parses to a
// Document that accepts everything, because a validator that rejected on an unusable schema would
// block a configuration the service would take.
func Parse(toolID, schemaVersion string, raw json.RawMessage) (*Document, error) {
	doc := &Document{toolID: toolID, schemaVersion: schemaVersion}
	if len(raw) == 0 {
		return doc, nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode %s schema %s: %w", toolID, schemaVersion, err)
	}
	if root, ok := decoded.(map[string]any); ok {
		doc.root = root
	}
	return doc, nil
}

// ToolID returns the product the schema describes.
func (d *Document) ToolID() string { return d.toolID }

// SchemaVersion returns the schema version the schema describes.
func (d *Document) SchemaVersion() string { return d.schemaVersion }

// Validate checks a settings object against the schema, returning every problem found. The order is
// deterministic — depth-first through the settings, keys visited in sorted order — so a diagnostic
// set does not reshuffle between plans. A nil Document accepts everything, so a caller that could not fetch a schema degrades to no
// findings rather than to false ones.
func (d *Document) Validate(settings map[string]any) []Problem {
	if d == nil || d.root == nil {
		return nil
	}
	w := &walker{root: d.root}
	w.validate(d.root, settings, "")
	return w.problems
}
