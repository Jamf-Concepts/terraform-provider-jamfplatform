// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EmptyStringSet returns a known, empty Set<String>. It is the canonical
// "no members" value for a MANAGED classic scope target category: a declared
// `[]` config plans as `[]`, the read path flattens an empty wire result for a
// managed category to `[]`, and the read-merge-write update path uses it to
// normalise absent server sub-blocks. Unmanaged (omitted) categories are null,
// never `[]` — the null/empty distinction carries the ownership semantics.
// See scope/schema.go.
func EmptyStringSet() types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{})
}

// BuildIDSlice projects a Terraform Set<String> of numeric Jamf Pro IDs
// into the SDK pointer-slice expected by classic scope sub-blocks. mk is
// the per-call constructor that wraps each parsed int ID in the resource's
// SDK item type.
//
// Ownership-preserving omission semantics (wire-probed — see STYLE_GUIDE.md
// §Scope helper omission semantics):
//
//   - null / unknown → nil: the category is unmanaged, the SDK omits the
//     parent XML element entirely.
//   - empty (declared `[]`, or an empty merged value on the read-merge-write
//     update path) → a non-nil pointer to an EMPTY slice: the caller assigns
//     the parent wrapper, which marshals as an empty element (e.g.
//     `<computer_groups></computer_groups>`). The empty element is what
//     clears the category — a scope PUT replaces the whole subtree once any
//     category element is present, and a body whose scope has no category
//     elements at all is ignored by the server, so "clear the last member"
//     only works via the explicit empty wrapper.
//
// Element parse errors do not short-circuit — every failure is collected
// in one pass so the user sees them all at once. Returns (nil, diags) when
// any element fails to parse as an integer.
func BuildIDSlice[T any](
	ctx context.Context,
	set types.Set,
	mk func(int) T,
) (*[]T, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var elements []string
	diags.Append(set.ElementsAs(ctx, &elements, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]T, 0, len(elements))
	parseFailed := false
	for _, raw := range elements {
		id, err := strconv.Atoi(raw)
		if err != nil {
			diags.AddError(
				"Invalid ID in set",
				err.Error(),
			)
			parseFailed = true
			continue
		}
		out = append(out, mk(id))
	}
	if parseFailed {
		return nil, diags
	}
	return &out, diags
}

// FlattenIDSlice projects an SDK pointer-slice into a Terraform Set<String>
// of ID values. extract returns the int ID from the SDK item type. Returns an
// empty Set<String> (EmptyStringSet) if items is nil or empty: empty is the
// canonical "no members" value for a managed category, so an absent wire
// sub-block reads back as `[]`, not null. Ownership gating (unmanaged stays
// null) happens at the call site via RefreshManagedSet, not here. Items whose
// extract returns nil are skipped — server should not return such items but
// defend.
func FlattenIDSlice[T any](
	ctx context.Context,
	items *[]T,
	extract func(T) *int,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if items == nil || len(*items) == 0 {
		return EmptyStringSet(), diags
	}
	values := make([]string, 0, len(*items))
	for _, item := range *items {
		idPtr := extract(item)
		if idPtr == nil {
			continue
		}
		values = append(values, strconv.Itoa(*idPtr))
	}
	if len(values) == 0 {
		return EmptyStringSet(), diags
	}
	out, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return out, diags
}

// BuildNameSlice projects a Terraform Set<String> of names into the SDK
// pointer-slice expected by name-only scope sub-blocks
// (directory_service_or_local_user_names, directory_service_user_group_names,
// limit_to_user_group_names). Same omission semantics as BuildIDSlice —
// null/unknown → nil (unmanaged, element omitted); empty → non-nil empty
// slice (declared clear, empty element emitted).
func BuildNameSlice[T any](
	ctx context.Context,
	set types.Set,
	mk func(string) T,
) (*[]T, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var elements []string
	diags.Append(set.ElementsAs(ctx, &elements, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]T, 0, len(elements))
	for _, name := range elements {
		out = append(out, mk(name))
	}
	return &out, diags
}

// FlattenNameSlice projects an SDK pointer-slice into a Terraform Set<String>
// of name values. extract returns the name pointer from the SDK item type.
// Returns an empty Set<String> (EmptyStringSet) on nil/empty input — see
// FlattenIDSlice for the empty-is-canonical rationale. Items whose extract
// returns nil or empty string are skipped — server-side blanks have no
// meaningful TF representation.
func FlattenNameSlice[T any](
	ctx context.Context,
	items *[]T,
	extract func(T) *string,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if items == nil || len(*items) == 0 {
		return EmptyStringSet(), diags
	}
	values := make([]string, 0, len(*items))
	for _, item := range *items {
		namePtr := extract(item)
		if namePtr == nil || *namePtr == "" {
			continue
		}
		values = append(values, *namePtr)
	}
	if len(values) == 0 {
		return EmptyStringSet(), diags
	}
	out, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return out, diags
}
