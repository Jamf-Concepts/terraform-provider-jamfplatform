// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package class

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// stringSliceFromSet extracts a plain []string from a set, returning an empty
// (non-nil) slice when the set is null or unknown. Used by the always-emit input
// builders so an empty collection serialises as an empty wrapper element and the
// server clears the corresponding members.
func stringSliceFromSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return []string{}, nil
	}
	return helpers.SetToStringSlice(ctx, set)
}

// intSliceFromSet extracts a []int from a set of stringified IDs, returning an
// empty (non-nil) slice when the set is null or unknown. Non-integer elements
// produce a diagnostic.
func intSliceFromSet(ctx context.Context, set types.Set, attrLabel string) ([]int, diag.Diagnostics) {
	strs, diags := stringSliceFromSet(ctx, set)
	if diags.HasError() {
		return nil, diags
	}
	ids := make([]int, 0, len(strs))
	for _, s := range strs {
		n, err := strconv.Atoi(s)
		if err != nil {
			diags.AddError(
				"Invalid ID in "+attrLabel,
				"Expected an integer Jamf Pro ID but got "+strconv.Quote(s)+".",
			)
			return nil, diags
		}
		ids = append(ids, n)
	}
	return ids, diags
}

// idStringsFromIntSlice converts a server-returned *[]int into a []string for
// state. Returns nil for a nil or empty slice.
func idStringsFromIntSlice(ids *[]int) []string {
	if ids == nil || len(*ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(*ids))
	for _, n := range *ids {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

// derefStringSlice returns the underlying []string for a non-nil *[]string, or
// nil otherwise.
func derefStringSlice(s *[]string) []string {
	if s == nil {
		return nil
	}
	return *s
}

// reconcileStringSet maps server-returned values back into a Terraform set while
// keeping an always-emit authoritative collection consistent:
//
//   - When the server returns no members, the prior shape is preserved — a null
//     attribute stays null and an explicit empty set stays empty — so the
//     framework's "produced inconsistent result after apply" check does not trip.
//   - When caseInsensitive is set (username sets), any server value that matches
//     a configured value case-insensitively keeps the configured casing. Jamf Pro
//     canonicalises usernames (e.g. Kyle@X → kyle@x); without this the canonical
//     echo would surface as perpetual drift.
func reconcileStringSet(ctx context.Context, apiValues []string, current types.Set, caseInsensitive bool) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(apiValues) == 0 {
		if current.IsNull() || current.IsUnknown() {
			return types.SetNull(types.StringType), diags
		}
		// Prior value was an explicit (possibly empty) set: represent the now-empty
		// server state as an empty set, not null, to match the plan.
		set, d := types.SetValueFrom(ctx, types.StringType, []string{})
		diags.Append(d...)
		return set, diags
	}

	values := apiValues
	if caseInsensitive && !current.IsNull() && !current.IsUnknown() {
		cur, d := helpers.SetToStringSlice(ctx, current)
		diags.Append(d...)
		if diags.HasError() {
			return types.SetNull(types.StringType), diags
		}
		byLower := make(map[string]string, len(cur))
		for _, c := range cur {
			byLower[strings.ToLower(c)] = c
		}
		values = make([]string, len(apiValues))
		for i, a := range apiValues {
			if orig, ok := byLower[strings.ToLower(a)]; ok {
				values[i] = orig
			} else {
				values[i] = a
			}
		}
	}

	set, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return set, diags
}

// computedIDSet maps a server-returned *[]int into a Computed set of stringified
// IDs, returning a null set when empty. Used for the read-only student_ids /
// teacher_ids echoes (no reconcile: server is authoritative and IDs are exact).
func computedIDSet(ctx context.Context, ids *[]int) (types.Set, diag.Diagnostics) {
	strs := idStringsFromIntSlice(ids)
	if len(strs) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, strs)
}
