// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package permissions renders the Jamf API permissions a resource or data
// source requires into a Markdown table suitable for appending to a schema
// MarkdownDescription.
//
// The data is sourced from the jamfplatform-go-sdk: every generated SDK
// sub-package (pro, devices, proclassic, ...) exposes a Privileges map keyed by
// method name, whose Scoped field carries the GA capability permissions the
// endpoint requires in {capability}:{action} form. A construct declares the SDK
// methods it calls and this package turns the union of their requirements into
// one deduplicated, operator-facing table.
//
// The table is written for the person creating the API integration, which
// happens in Jamf Account, so a row is a row of Jamf Account's permission
// picker: the section it sits under, the name printed beside its checkboxes,
// and which of those checkboxes to tick. catalogue.go holds that mapping.
//
// The SDK also publishes each method's pre-GA Jamf Pro privilege names
// (MethodPrivileges.Legacy) and this package used to render them beside the
// capability. It no longer does. Scoped and Legacy are independent sets — the
// GA consolidation mapped several pre-GA privileges onto one capability, and
// where the lengths do match the spec's own orders disagree — so no per-row
// label can be derived from them, and Jamf Account no longer offers the pre-GA
// names to grant.
package permissions

import (
	"fmt"
	"maps"
	"slices"
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

// requirement is one deduplicated table row: a capability plus every action on
// it the construct needs. One row per capability rather than per
// capability-action pair, because that is how Jamf Account presents it — a
// permission with a checkbox per action — and it collapses the four rows of an
// ordinary CRUD resource into one.
type requirement struct {
	capability string
	actions    map[string]bool
}

// splitScoped divides a GA capability permission into its capability and action
// halves. The retired three-part beta slug ("create:pro:buildings") is not
// handled: the SDK's Scoped field documents the two-part GA form as the only
// one it emits, and every entry in every shipped registry is that form, so a
// value that does not split in two is a shape this package has never seen and
// is surfaced verbatim rather than guessed at.
func splitScoped(scoped string) (capability, action string, ok bool) {
	capability, action, ok = strings.Cut(scoped, ":")
	if !ok || capability == "" || action == "" {
		return "", "", false
	}
	return capability, action, true
}

// collect returns the deduplicated capability requirements across the named
// methods, plus the names not found in the registry. A missing slice lets the
// caller (and the drift-guard test) detect a method that was renamed or removed
// in the SDK.
//
// Rows are ordered by Jamf Account's own section order and then by permission
// name, so the table reads in the order an operator ticks it. A capability the
// catalogue does not know sorts to the end: it is a capability Jamf has added
// since catalogue.go was last transcribed, and the drift-guard test is what
// turns that into a build failure — rendering must still produce something
// truthful in the meantime.
func collect(reg Registry, methods []string) (reqs []requirement, noPrivilege bool, missing []string) {
	seen := make(map[string]int) // capability -> index into reqs
	known := 0
	for _, name := range methods {
		mp, ok := reg[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		known++
		for _, scoped := range mp.Scoped {
			capability, action, ok := splitScoped(scoped)
			if !ok {
				capability, action = scoped, ""
			}
			idx, dup := seen[capability]
			if !dup {
				idx = len(reqs)
				seen[capability] = idx
				reqs = append(reqs, requirement{capability: capability, actions: map[string]bool{}})
			}
			if action != "" {
				reqs[idx].actions[action] = true
			}
		}
	}
	// All resolved methods require no special permission.
	noPrivilege = known > 0 && len(reqs) == 0
	sort.Slice(reqs, func(i, j int) bool {
		ci, ni := rowKey(reqs[i].capability)
		cj, nj := rowKey(reqs[j].capability)
		if ci != cj {
			return ci < cj
		}
		if ni != nj {
			return ni < nj
		}
		return reqs[i].capability < reqs[j].capability
	})
	return reqs, noPrivilege, missing
}

// rowKey returns the sort key of a capability's row: its section's index in
// Jamf Account's ordering, and the permission name within that section.
func rowKey(capability string) (int, string) {
	e, ok := catalogue[capability]
	if !ok {
		return len(categoryOrder), capability
	}
	return slices.Index(categoryOrder, e.category), e.name
}

// actionList renders an action set as the words Jamf Account prints beside the
// checkboxes, in the platform's own create/read/update/delete/deploy/execute
// order. An action outside those six is passed through verbatim so a new one
// shows up as itself rather than vanishing.
func actionList(actions map[string]bool) string {
	var out []string
	for _, a := range actionOrder {
		if actions[a] {
			out = append(out, actionLabels[a])
		}
	}
	var extra []string
	for a := range actions {
		if _, known := actionLabels[a]; !known {
			extra = append(extra, a)
		}
	}
	sort.Strings(extra)
	out = append(out, extra...)
	return strings.Join(out, ", ")
}

// Section renders the full "Required Jamf permissions" Markdown block (heading,
// lead-in, and table) for the SDK methods a construct calls, ready to append to
// a schema MarkdownDescription. It returns "" when none of the named methods
// resolve to a registry entry, so callers can append it unconditionally.
// Unknown method names are silently skipped here; the per-construct drift-guard
// test is responsible for failing the build when a declared method is absent
// from the registry.
func Section(reg Registry, methods ...string) string {
	reqs, noPrivilege, _ := collect(reg, methods)

	if noPrivilege {
		return "\n\n**Required Jamf permissions**\n\n" +
			"None — any authenticated Jamf Platform API integration may call the underlying endpoints."
	}
	if len(reqs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n**Required Jamf permissions**\n\n")
	b.WriteString("Grant the API integration the following permissions in Jamf Account. " +
		"`Category` and `Permission` name the section and row of the permission picker; " +
		"`Actions` are the boxes to tick within that row.\n\n")
	b.WriteString("| Category | Permission | Actions | API capability |\n|---|---|---|---|\n")
	unknown := false
	for _, r := range reqs {
		e, ok := catalogue[r.capability]
		if !ok {
			unknown = true
			e = entry{category: "—", name: "—"}
		}
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` |\n", e.category, e.name, actionList(r.actions), r.capability)
	}
	// A dash in both name columns means the capability postdates this
	// provider's copy of Jamf's permissions map, not that the permission is
	// ungrantable — say which, because the two look identical in a table.
	if unknown {
		b.WriteString("\n`—` marks a capability this provider release has no Jamf Account name recorded for. " +
			"Grant it by searching the permission picker for the API capability shown.\n")
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

// Renders reports whether a Section-rendered block grants scoped, a GA
// capability permission in {capability}:{action} form.
//
// It exists for the per-construct drift-guard tests, which assert that a
// construct's table really did render the permissions its SDK methods require.
// Those tests used to look for the scoped string with strings.Contains, which
// stopped working when the table moved to Jamf Account's own vocabulary: the
// capability and the action now sit in different cells, and the action is a
// word rather than a slug. Parsing the table back is the honest check — a
// substring match on the capability alone would pass a row that ticks the wrong
// boxes.
func Renders(section, scoped string) bool {
	capability, action, ok := splitScoped(scoped)
	if !ok {
		return false
	}
	label, ok := actionLabels[action]
	if !ok {
		return false
	}
	for line := range strings.SplitSeq(section, "\n") {
		cells := tableCells(line)
		if len(cells) != 4 || cells[3] != "`"+capability+"`" {
			continue
		}
		if slices.Contains(strings.Split(cells[2], ", "), label) {
			return true
		}
	}
	return false
}

// tableCells splits one Markdown table row into its trimmed cells, or returns
// nil for a line that is not one.
func tableCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	var out []string
	for cell := range strings.SplitSeq(strings.Trim(line, "|"), "|") {
		out = append(out, strings.TrimSpace(cell))
	}
	return out
}
