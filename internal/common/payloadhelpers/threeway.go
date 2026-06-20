// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"bytes"
	"fmt"
	"maps"
	"slices"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

// ThreeWayDecision is the outcome of comparing a plan's payload to two
// reference values stashed in private state from the previous Apply.
type ThreeWayDecision int

const (
	// DecisionNoOp — user HCL semantically matches what we last sent and
	// the server still semantically matches what it returned at the last
	// Apply. The plan modifier should rewrite plan to state.
	DecisionNoOp ThreeWayDecision = iota
	// DecisionApply — user has changed their HCL since the last Apply.
	// The plan modifier should let the diff through unchanged so the
	// resource's Update runs.
	DecisionApply
	// DecisionDrift — user HCL is unchanged but the server-canonical
	// representation has diverged from what was captured at the last
	// Apply. Surface this as drift so the operator sees an admin UI edit.
	DecisionDrift
)

// ThreeWayCompare distinguishes three classes of "plan differs from
// state" so the plan modifier can decide between suppression, propagation,
// and explicit drift:
//
//   - planInput:     the user's HCL value on this plan.
//   - lastInput:     the HCL value the user last successfully applied —
//     stashed in resource private state at Create/Update.
//   - lastCanonical: the server response captured immediately after that
//     same Apply — also stashed in private state. This is
//     pre-divergence: any Jamf-side strips (e.g. PPPC
//     Location, top-level PayloadDescription clear) have
//     already taken effect, so a later mismatch against
//     serverNow is unambiguously admin drift.
//   - serverNow:     the just-refreshed server-canonical value held in
//     state.payloads at plan time.
//
// Decision matrix:
//
//   - planInput vs lastInput differ semantically → DecisionApply.
//     User changed HCL — propagate. This is the only path that compares
//     across the "user authoring" layer, so a key the user adds to HCL
//     surfaces even if Jamf would later strip it on write.
//   - lastCanonical vs serverNow differ semantically → DecisionDrift.
//     Both sides are server-canonical from the same wire layer, so any
//     asymmetric key, missing key, or value change is an admin-side edit
//     since the last Apply.
//   - otherwise → DecisionNoOp.
//
// Caller is expected to fall back to the legacy two-way
// PayloadsSemanticallyEqual when lastInput or lastCanonical is empty —
// fresh imports and pre-three-way-tracking resources have no private
// state to compare against.
func ThreeWayCompare(planInput, lastInput, lastCanonical, serverNow []byte) (ThreeWayDecision, error) {
	userChanged, err := payloadsStrictlyDiffer(planInput, lastInput)
	if err != nil {
		return 0, fmt.Errorf("comparing plan input to last-applied input: %w", err)
	}
	if userChanged {
		return DecisionApply, nil
	}
	adminDrift, err := payloadsStrictlyDiffer(lastCanonical, serverNow)
	if err != nil {
		return 0, fmt.Errorf("comparing last-applied canonical to current server: %w", err)
	}
	if adminDrift {
		return DecisionDrift, nil
	}
	return DecisionNoOp, nil
}

// payloadsStrictlyDiffer parses, masks (using MaskPayload's
// always-injected-key list), and structurally compares both sides. The
// compare is symmetric — asymmetric dict keys, length-mismatched arrays,
// and value mismatches all return true.
//
// Used for both arms of ThreeWayCompare because each arm compares two
// values sourced from the same authoring layer (both user HCL, or both
// server-canonical post-Apply). Cross-layer comparisons — the legacy
// "plan input vs current server" path — must continue to use
// PayloadsSemanticallyEqual's intersection semantics.
func payloadsStrictlyDiffer(a, b []byte) (bool, error) {
	ma, err := MaskPayload(a)
	if err != nil {
		return false, fmt.Errorf("masking left side: %w", err)
	}
	mb, err := MaskPayload(b)
	if err != nil {
		return false, fmt.Errorf("masking right side: %w", err)
	}
	return !structuralEqual(ma, mb), nil
}

// PayloadsStructurallyEqual returns true when the two payloads compare
// equal under strict semantics — i.e. the masked plist trees have
// identical keysets and values. Exposed for the Read-side drift detector
// in each profile resource: comparing the last-applied server canonical
// (from private state) against the current server response distinguishes
// admin-side UI edits (strict-differ) from steady-state refreshes
// (strict-equal).
func PayloadsStructurallyEqual(a, b []byte) (bool, error) {
	differ, err := payloadsStrictlyDiffer(a, b)
	if err != nil {
		return false, err
	}
	return !differ, nil
}

// structuralEqual is a recursive strict equality compare for the
// parsed-and-masked plist tree. Built on top of maps.EqualFunc and
// slices.EqualFunc — the comparator recurses via a closure so nested
// maps/arrays inherit the same semantics with no manual key walking.
//
// numericEqual handles the int64/uint64/int trio howett.net/plist emits
// depending on sign.
func structuralEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return maps.EqualFunc(av, bv, structuralEqual)
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return false
		}
		return slices.EqualFunc(av, bv, structuralEqual)
	case []uint8:
		bv, ok := b.([]uint8)
		if !ok {
			return false
		}
		return bytes.Equal(av, bv)
	case uint64:
		return plisthelpers.NumericEqual(int64(av), b)
	case int64:
		return plisthelpers.NumericEqual(av, b)
	case int:
		return plisthelpers.NumericEqual(int64(av), b)
	default:
		return a == b
	}
}
