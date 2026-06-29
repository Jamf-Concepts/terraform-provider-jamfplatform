// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package permissions renders the Jamf API privileges a resource or data
// source requires into a Markdown table suitable for appending to a
// schema MarkdownDescription.
//
// The privilege data is sourced from the jamfplatform-go-sdk: every generated
// SDK sub-package (pro, devices, proclassic, ...) exposes a Privileges map
// keyed by method name. A construct declares the SDK methods it calls and this
// package turns the union of their required privileges into a deduplicated,
// operator-facing table. The legacy privilege names (e.g. "Read Buildings")
// are the labels shown in the Jamf Pro admin UI when granting privileges to a
// Jamf Platform API integration, so the table is genuinely user-facing rather
// than API plumbing.
package permissions

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
)

// Registry is the type of the per-package Privileges maps exported by the SDK
// (e.g. pro.Privileges). Aliased here so callers and Merge read cleanly.
type Registry = map[string]jamfplatform.MethodPrivileges

// Merge combines several SDK package registries into one lookup, for resources
// whose construct spans more than one SDK family (e.g. a Pro resource that
// also calls a ProClassic endpoint). Later registries win on key collision,
// which never happens in practice because method names are unique per family.
func Merge(registries ...Registry) Registry {
	out := make(Registry)
	for _, reg := range registries {
		maps.Copy(out, reg)
	}
	return out
}

// privilege is one deduplicated table row: a scoped identifier plus its
// human-readable Jamf Pro privilege name when one is published.
type privilege struct {
	scoped string
	legacy string
}

// collect returns the deduplicated privileges required across the named
// methods, plus the names not found in the registry. A missing slice lets the
// caller (and the drift-guard test) detect a method that was renamed or
// removed in the SDK. Scoped and Legacy are paired by index only when a
// method's two slices are the same length — the common single-resource CRUD
// case; when they differ (a handful of cross-resource Pro operations) the
// scoped identifiers are emitted without a legacy name rather than guessing a
// pairing the SDK does not encode.
func collect(reg Registry, methods []string) (privs []privilege, noPrivilege bool, missing []string) {
	seen := make(map[string]int) // scoped -> index into privs
	known := 0
	for _, name := range methods {
		mp, ok := reg[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		known++
		aligned := len(mp.Scoped) == len(mp.Legacy)
		for i, scoped := range mp.Scoped {
			legacy := ""
			if aligned {
				legacy = mp.Legacy[i]
			}
			if idx, dup := seen[scoped]; dup {
				// Prefer a row that carries a legacy name over one that lacks it.
				if privs[idx].legacy == "" && legacy != "" {
					privs[idx].legacy = legacy
				}
				continue
			}
			seen[scoped] = len(privs)
			privs = append(privs, privilege{scoped: scoped, legacy: legacy})
		}
	}
	// All resolved methods require no special privilege.
	noPrivilege = known > 0 && len(privs) == 0
	sort.Slice(privs, func(i, j int) bool { return privs[i].scoped < privs[j].scoped })
	return privs, noPrivilege, missing
}

// Section renders the full "Required Jamf privileges" Markdown block (heading,
// lead-in, and table) for the SDK methods a construct calls, ready to append
// to a schema MarkdownDescription. It returns "" when none of the named
// methods resolve to a privilege entry, so callers can append it
// unconditionally. Unknown method names are silently skipped here; the
// per-construct drift-guard test is responsible for failing the build when a
// declared method is absent from the registry.
func Section(reg Registry, methods ...string) string {
	privs, noPrivilege, _ := collect(reg, methods)

	if noPrivilege {
		return "\n\n**Required Jamf privileges**\n\n" +
			"None — any authenticated Jamf Platform API integration may call the underlying endpoints."
	}
	if len(privs) == 0 {
		return ""
	}

	hasLegacy := false
	for _, p := range privs {
		if p.legacy != "" {
			hasLegacy = true
			break
		}
	}

	var b strings.Builder
	b.WriteString("\n\n**Required Jamf privileges**\n\n")
	b.WriteString("The Jamf Platform API integration used by the provider must be granted the following privileges:\n\n")
	if hasLegacy {
		b.WriteString("| Jamf Pro privilege | Scoped name |\n|---|---|\n")
		for _, p := range privs {
			name := p.legacy
			if name == "" {
				name = "—"
			}
			fmt.Fprintf(&b, "| %s | `%s` |\n", name, p.scoped)
		}
	} else {
		b.WriteString("| Required privilege |\n|---|\n")
		for _, p := range privs {
			fmt.Fprintf(&b, "| `%s` |\n", p.scoped)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// Missing resolves the named methods against the registry and returns those
// absent from it. Drift-guard tests call this to assert a construct's declared
// SDK method list stays in sync with the SDK.
func Missing(reg Registry, methods ...string) []string {
	_, _, missing := collect(reg, methods)
	return missing
}
