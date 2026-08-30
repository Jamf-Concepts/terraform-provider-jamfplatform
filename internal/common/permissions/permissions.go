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
	"unicode"

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

// legacyVerbActions maps the verb a Jamf Pro privilege name starts with to the
// action of the scoped identifier it must correspond to. Only the four CRUD
// verbs are listed: they are the ones a privilege name states unambiguously.
var legacyVerbActions = map[string]string{
	"Create": "create",
	"Read":   "read",
	"Update": "update",
	"Delete": "delete",
}

// pairLegacy returns the Jamf Pro privilege name to render against each of
// scoped, or nil when no pairing can be trusted. Index i of the result belongs
// to index i of scoped; an empty string means that row has no label.
//
// The SDK documents Scoped and Legacy as two sets with no bijection, and emits
// Scoped sorted against Legacy in spec order, so index pairing is a guess. It
// is also not the only thing available, and discarding the whole set when the
// guess fails throws away pairings the data determines exactly. So this
// degrades in stages, most trustworthy first:
//
//  1. index pairing, when verifiedPairing can confirm every row;
//  2. verb-keyed pairing, when every scoped identifier shares one capability —
//     then the label's leading CRUD verb names its action outright and the
//     assignment is a bijection no ordering can disturb;
//  3. a single scoped identifier, which every published name must belong to
//     because there is nothing else for them to belong to;
//  4. otherwise unlabelled, because a missing label is honest and a wrong one
//     is not.
//
// Stage 3 exists for Jamf's GA privilege collapse: several pre-GA privileges
// map onto one GA identifier (ListMacOSBrandingConfigurationsV1 needs both
// "Read Self Service Branding Configuration" and "Read Self Service" for
// `self-service:read`), and a length mismatch there is the collapse, not an
// unpairable set.
func pairLegacy(scoped, legacy []string) []string {
	if len(scoped) == 0 || len(legacy) == 0 {
		return nil
	}
	if len(scoped) == len(legacy) {
		if verifiedPairing(scoped, legacy) {
			return legacy
		}
		return verbKeyedPairing(scoped, legacy)
	}
	if len(scoped) == 1 {
		return []string{strings.Join(legacy, ", ")}
	}
	return nil
}

// verbKeyedPairing pairs each legacy name to the scoped identifier whose action
// matches the name's leading CRUD verb. It applies only when every scoped
// identifier shares one capability — otherwise the verb does not identify a row
// — and only when the result is a total bijection, so a set with a duplicate or
// uncheckable verb is refused rather than half-paired.
func verbKeyedPairing(scoped, legacy []string) []string {
	byAction := make(map[string]int, len(scoped))
	capability := capabilityOf(scoped[0])
	for i, s := range scoped {
		if capabilityOf(s) != capability {
			return nil
		}
		byAction[actionOf(s)] = i
	}
	out := make([]string, len(scoped))
	paired := 0
	for _, name := range legacy {
		verb, _, ok := strings.Cut(name, " ")
		if !ok {
			return nil
		}
		action, known := legacyVerbActions[verb]
		if !known {
			return nil
		}
		i, ok := byAction[action]
		if !ok || out[i] != "" {
			return nil
		}
		out[i] = name
		paired++
	}
	if paired != len(scoped) {
		return nil
	}
	return out
}

// verifiedPairing reports whether scoped[i] and legacy[i] can be trusted to
// describe the same privilege.
//
// A single privilege has no ordering to get wrong, so it is always trusted —
// which keeps every ordinary one-privilege method's admin-UI label intact. From
// two privileges up, each row must survive three independent checks:
//
//   - the legacy name's leading verb must name the action its scoped partner
//     carries. A verb outside the four CRUD ones (e.g. "Send Computer Remote
//     Command to Install Package") cannot be checked at all.
//   - the legacy name's remaining words must share a word with the scoped
//     identifier's capability. The verb alone is not enough: where every
//     privilege on a method has the SAME action the verb test passes whatever
//     the order, so it would wave through GetDeviceGroupsForDeviceV1's
//     [device-groups:read, devices:read] against [Read Computers, Read Mobile
//     Devices] — which labels `device-groups:read` "Read Computers".
//   - that shared word must be discriminating: no other capability on the
//     method may share a word with the same label. Overlap is symmetric, so
//     "device" is common to `device-groups` and `devices` and confirms nothing
//     about which row a label naming a device belongs to — the four
//     List{Smart,Static}MobileDeviceGroupMembership methods pass the first two
//     checks in either order. A sibling carrying the SAME capability is not a
//     rival, because the verb already separates those rows;
//     UploadInventoryPreloadCsvV2 pairs two inventory-preload-record
//     privileges beside two user ones and is correctly confirmed.
//
// Failing any check marks the whole method unverified rather than half-trusted,
// because a set with one bad row gives no reason to trust the others. Callers
// reach for pairLegacy rather than this function directly, so an unverifiable
// set still gets the reconstruction stages before it is given up on.
func verifiedPairing(scoped, legacy []string) bool {
	if len(scoped) != len(legacy) {
		return false
	}
	if len(scoped) < 2 {
		return true
	}
	for i, name := range legacy {
		verb, rest, ok := strings.Cut(name, " ")
		if !ok {
			return false
		}
		action, known := legacyVerbActions[verb]
		if !known {
			return false
		}
		if !hasField(scoped[i], action) {
			return false
		}
		if !sharesWord(capabilityOf(scoped[i]), rest) {
			return false
		}
		for j, other := range scoped {
			otherCapability := capabilityOf(other)
			if j == i || otherCapability == capabilityOf(scoped[i]) {
				continue
			}
			if sharesWord(otherCapability, rest) {
				return false
			}
		}
	}
	return true
}

// sharesWord reports whether a capability slug and the descriptive half of a
// Jamf Pro privilege name have a word in common — "device-groups" against
// "Smart Mobile Device Groups" does, "device-groups" against "Computers" does
// not. Both sides are lowercased, split on anything non-alphanumeric and
// de-pluralised by a trailing "s", which is enough to match Jamf's own two
// spellings of the same noun ("Categories"/`categories`, "Check-In"/`check-in`,
// `users`/"Create User") without a vocabulary to maintain.
func sharesWord(capability, description string) bool {
	want := make(map[string]bool)
	for _, w := range splitWords(capability) {
		want[w] = true
	}
	for _, w := range splitWords(description) {
		if want[w] {
			return true
		}
	}
	return false
}

// splitWords lowercases s, splits it on every non-alphanumeric run and strips a
// trailing "s" from each word.
func splitWords(s string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		out = append(out, strings.TrimSuffix(w, "s"))
	}
	return out
}

// capabilityOf returns the capability half of a scoped identifier, and actionOf
// the action half. Both tolerate the two spellings Jamf has shipped: the GA
// {capability}:{action} form puts the capability first, the older
// {action}:{scope}:{resource} form puts it last. Keying off the field count
// rather than a vocabulary keeps them agreeing with hasField, which matches an
// action in any position for the same reason.
func capabilityOf(scoped string) string {
	if fields := strings.Split(scoped, ":"); len(fields) == 3 {
		return fields[2]
	}
	capability, _, _ := strings.Cut(scoped, ":")
	return capability
}

func actionOf(scoped string) string {
	fields := strings.Split(scoped, ":")
	if len(fields) == 3 {
		return fields[0]
	}
	return fields[len(fields)-1]
}

// hasField reports whether action is one of scoped's colon-delimited fields.
// Matching any field rather than a fixed position keeps the check working across
// both privilege spellings Jamf has shipped — the GA {capability}:{action} form
// puts the action last, the older {action}:pro:{resource} form put it first —
// without the check needing to know which one it is looking at.
func hasField(scoped, action string) bool {
	for field := range strings.SplitSeq(scoped, ":") {
		if field == action {
			return true
		}
	}
	return false
}

// collect returns the deduplicated privileges required across the named
// methods, plus the names not found in the registry. A missing slice lets the
// caller (and the drift-guard test) detect a method that was renamed or
// removed in the SDK. pairLegacy decides which Jamf Pro privilege name each
// scoped identifier may be labelled with, so a pairing the SDK does not encode
// is reconstructed where the data determines it and dropped where it does not.
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
		paired := pairLegacy(mp.Scoped, mp.Legacy)
		for i, scoped := range mp.Scoped {
			legacy := ""
			if paired != nil {
				legacy = paired[i]
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

	hasLegacy, hasBlank, hasCollapsed := false, false, false
	for _, p := range privs {
		if p.legacy == "" {
			hasBlank = true
			continue
		}
		hasLegacy = true
		if strings.Contains(p.legacy, ", ") {
			hasCollapsed = true
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
		// The left column is the pre-GA privilege name for *this operation*, not
		// an alias of the scoped identifier: Jamf's GA collapse maps several
		// pre-GA privileges onto one identifier, so the relationship is
		// many-to-one in both directions across the API. Say so where it shows,
		// rather than leaving a reader to read a blank cell as a rendering bug.
		if hasCollapsed {
			b.WriteString("\nWhere a row lists more than one Jamf Pro privilege, the single scoped privilege replaced all of them: grant every name listed on a Jamf Pro version that predates the scoped privileges.\n")
		}
		if hasBlank {
			b.WriteString("\n`—` means Jamf publishes no Jamf Pro privilege name that can be matched to that scoped privilege with confidence, so none is guessed here — grant it by its scoped name.\n")
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
