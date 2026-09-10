// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// declarationDirs are the subdirectories of `declarative/declarations` that hold declarations an
// author can write. `assets/credentials` is deliberately absent: those files describe the JSON
// document a credential asset's DataURL serves, not a declaration, and carry no declarationtype.
// `declarative/status` is absent for the same reason — status items are device-to-server reports,
// which is why the service refuses a declaration of kind STATUS — but its file names are still
// harvested, see statusItems.
var declarationDirs = []string{
	"configurations",
	"assets",
	"activations",
	"management",
}

// kindPrefixes maps a declaration type's reverse-domain prefix to the `kind` the service expects
// alongside it. The service accepts any pairing and silently keeps whatever it is given, so a
// mismatch is a plan-time-only finding — and one that needs no table, since the prefix determines
// the answer.
var kindPrefixes = []struct {
	prefix string
	kind   string
}{
	{"com.apple.configuration.", "CONFIGURATION"},
	{"com.apple.asset.", "ASSET"},
	{"com.apple.activation.", "ACTIVATION"},
	{"com.apple.management.", "MANAGEMENT"},
}

// kindForType returns the kind a declaration type belongs to, or "" when the prefix is unknown.
func kindForType(declarationType string) string {
	for _, candidate := range kindPrefixes {
		if strings.HasPrefix(declarationType, candidate.prefix) {
			return candidate.kind
		}
	}
	return ""
}

// declaration is the emitted shape of one declaration type.
type declaration struct {
	Title string             `json:"title,omitempty"`
	Kind  string             `json:"kind"`
	Keys  map[string]*schema `json:"keys"`
	Any   *schema            `json:"any,omitempty"`
	// Refs names the upstream branches declaring this type, so a diagnostic can distinguish
	// "Apple has never declared this" from "declared only on the pre-release seed branch".
	Refs []string `json:"refs,omitempty"`
}

// declarationTable is the emitted table, written to declarations.json.
type declarationTable struct {
	Source string `json:"source"`
	// Commits maps each upstream branch read to the commit it was read at.
	Commits map[string]string `json:"commits"`
	// Refs lists those branches in read order, the last being the most forward-looking.
	Refs         []string                `json:"refs"`
	Release      string                  `json:"release,omitempty"`
	Declarations map[string]*declaration `json:"declarations"`
	// StatusItems is the vocabulary accepted by
	// com.apple.configuration.management.status-subscriptions. Nothing in Apple's schemas types
	// those strings, so without this list they would be unvalidated free text.
	StatusItems []string `json:"statusItems"`
}

// generateDeclarations reads every declaration schema under each checkout and unions them, so a key
// Apple has published on a seed branch but not yet on release is recognised rather than reported as
// unknown. Jamf's generative-declarations component tracks the seed branch, so a table built from
// release alone reports keys the Jamf UI already offers as misspelled.
func generateDeclarations(roots []checkout, release string) (*declarationTable, error) {
	generated := &declarationTable{
		Source:       "https://github.com/apple/device-management",
		Commits:      make(map[string]string, len(roots)),
		Release:      release,
		Declarations: make(map[string]*declaration),
	}

	statusSeen := make(map[string]bool)

	for _, root := range roots {
		generated.Refs = append(generated.Refs, root.ref)
		generated.Commits[root.ref] = root.commit

		for _, dir := range declarationDirs {
			paths, err := filepath.Glob(filepath.Join(root.path, "declarative", "declarations", dir, "*.yaml"))
			if err != nil {
				return nil, err
			}
			sort.Strings(paths)
			for _, path := range paths {
				if err := mergeDeclaration(generated, path, root.ref); err != nil {
					return nil, err
				}
			}
		}

		for _, name := range statusItems(root.path) {
			statusSeen[name] = true
		}
	}

	if len(generated.Declarations) == 0 {
		return nil, fmt.Errorf("no declaration schemas found under %d checkout(s)", len(roots))
	}

	generated.StatusItems = sortedKeys(statusSeen)
	return generated, nil
}

// mergeDeclaration folds one upstream schema file into the accumulating table.
func mergeDeclaration(generated *declarationTable, path, ref string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	spec, err := parseDeclarationSpec(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}

	declarationType := strings.TrimSpace(spec.declarationType)
	if declarationType == "" {
		// No declarationtype means the file documents a sub-payload rather than a declaration.
		return nil
	}

	entry := generated.Declarations[declarationType]
	if entry == nil {
		entry = &declaration{
			Title: spec.title,
			Kind:  kindForType(declarationType),
			Keys:  make(map[string]*schema),
		}
		generated.Declarations[declarationType] = entry
	}
	entry.Refs = appendRef(entry.Refs, ref)

	for name, keySchema := range dictionaryKeys(spec.payloadKeys, 1) {
		entry.Keys[name] = unionSchema(entry.Keys[name], keySchema, ref)
	}
	for _, key := range spec.payloadKeys {
		if key != nil && key.Key == wildcardKey {
			wildcard := reduce(key, 1)
			wildcard.Required = false
			entry.Any = unionSchema(entry.Any, wildcard, ref)
		}
	}
	return nil
}

// unionSchema merges an incoming key schema into whatever an earlier branch produced, recording
// every branch that declares it. When two branches disagree on a key's type the later branch wins,
// because the branches are read oldest first and the newer definition is the one Jamf tracks; the
// disagreement is preserved in Refs rather than silently collapsed.
func unionSchema(existing, incoming *schema, ref string) *schema {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		// Stamp the whole subtree, not just this node: a nested key first seen on the release
		// branch would otherwise carry no ref, and the next branch read would make it look
		// seed-only — reporting a long-standing key as pre-release.
		stampRefs(incoming, ref)
		return incoming
	}

	merged := *incoming
	merged.Refs = appendRef(existing.Refs, ref)

	// A key required on one branch and optional on another is treated as required only if the
	// newest branch says so, which `merged` already carries from incoming.

	if existing.Keys != nil || incoming.Keys != nil {
		keys := make(map[string]*schema, len(existing.Keys)+len(incoming.Keys))
		for name, sub := range existing.Keys {
			keys[name] = sub
		}
		for name, sub := range incoming.Keys {
			keys[name] = unionSchema(existing.Keys[name], sub, ref)
		}
		merged.Keys = keys
	}
	merged.Any = unionSchema(existing.Any, incoming.Any, ref)
	merged.Item = unionSchema(existing.Item, incoming.Item, ref)

	return &merged
}

// stampRefs records a branch against a schema and everything beneath it.
func stampRefs(target *schema, ref string) {
	if target == nil {
		return
	}
	target.Refs = appendRef(target.Refs, ref)
	for _, sub := range target.Keys {
		stampRefs(sub, ref)
	}
	stampRefs(target.Any, ref)
	stampRefs(target.Item, ref)
}

// appendRef adds a branch name once, preserving read order.
func appendRef(refs []string, ref string) []string {
	for _, existing := range refs {
		if existing == ref {
			return refs
		}
	}
	return append(refs, ref)
}

// statusItems harvests the status-item vocabulary from the file names under `declarative/status`.
// The names are not carried inside the files — `device.identifier.serial-number.yaml` describes the
// item `device.identifier.serial-number` — so the directory listing is the only source.
func statusItems(root string) []string {
	paths, err := filepath.Glob(filepath.Join(root, "declarative", "status", "*.yaml"))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".yaml")
		// statusreason describes the shape of a failure reason rather than a subscribable item.
		if name == "" || name == "statusreason" {
			continue
		}
		names = append(names, name)
	}
	return names
}

// sortedKeys returns the set's members in sorted order.
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// declSpec is one upstream declaration schema file. It differs from a profile schema only in where
// the identifier lives: `payload.declarationtype` rather than `payload.payloadtype`.
type declSpec struct {
	title           string
	declarationType string
	payloadKeys     []*specKey
}

// parseDeclarationSpec reads one declaration schema file. It walks YAML at node level for the same
// reason parseSpec does: upstream defines self-referential structures through anchors, which struct
// unmarshalling rejects.
func parseDeclarationSpec(raw []byte) (*declSpec, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	root := document(&doc)
	if root == nil {
		return nil, fmt.Errorf("empty document")
	}
	return &declSpec{
		title:           scalar(field(root, "title")),
		declarationType: scalar(field(document(field(root, "payload")), "declarationtype")),
		payloadKeys:     parseKeys(field(root, "payloadkeys"), map[*yaml.Node]bool{}),
	}, nil
}
