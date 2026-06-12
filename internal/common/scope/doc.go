// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package scope provides shared schema attributes, builders, and validators
// for the <scope> block of Jamf Classic-API resources (policies, ebooks,
// mac applications, mobile device applications, OS X configuration profiles,
// mobile device configuration profiles, patch policies, restricted software).
//
// Each scope block mirrors the Jamf Pro admin UI's three Scope tabs:
// the all-flags and every per-category target set live inside a `targets`
// sub-block, with `limitations` and `exclusions` as its siblings. `targets`
// is Optional-only; the all-flags inside are Optional+Computed and use
// UseNonNullStateForUnknown so the null→present transition on `targets` does
// not trip a post-apply consistency error.
//
// Sub-block items are modelled as flat Set<String> of IDs (or names for the
// directory-service categories), not nested objects. Server-augmented <name>
// and <udid> wire fields are discarded on read; only IDs round-trip through
// Terraform state. Full rule set in STYLE_GUIDE.md §Scope helper.
//
// Cross-field invariants (all_computers ⇒ no per-computer targets, etc.)
// are enforced by AllFlagConflictsWith, a value-discriminated bool
// validator. Per-resource exceptions (e.g. RestrictedSoftware rejects
// limitations entirely) live in the resource package, not here.
package scope
