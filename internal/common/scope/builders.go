// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildIDSlice projects a Terraform Set<String> of numeric Jamf Pro IDs
// into the SDK pointer-slice expected by classic scope sub-blocks. mk is
// the per-call constructor that wraps each parsed int ID in the resource's
// SDK item type.
//
// Returns nil if the set is null, unknown, or empty so the SDK omits the
// parent XML element entirely (matching the wire convention of "absent
// block means no targets" — SCOPE_SPIKE §6.5).
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
	if len(elements) == 0 {
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
// of ID values. extract returns the int ID from the SDK item type. Returns
// types.SetNull(types.StringType) if items is nil or empty so state stays
// null on absent sub-blocks (matching omission semantics on the read side).
// Items whose extract returns nil are skipped — server should not return
// such items but defend.
func FlattenIDSlice[T any](
	ctx context.Context,
	items *[]T,
	extract func(T) *int,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if items == nil || len(*items) == 0 {
		return types.SetNull(types.StringType), diags
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
		return types.SetNull(types.StringType), diags
	}
	out, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return out, diags
}

// BuildNameSlice projects a Terraform Set<String> of names into the SDK
// pointer-slice expected by name-only scope sub-blocks
// (directory_service_or_local_user_names, directory_service_user_group_names,
// limit_to_user_group_names). Same nil/empty handling as BuildIDSlice —
// null, unknown, or empty input returns (nil, nil).
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
	if len(elements) == 0 {
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
// Items whose extract returns nil or empty string are skipped — server-side
// blanks have no meaningful TF representation.
func FlattenNameSlice[T any](
	ctx context.Context,
	items *[]T,
	extract func(T) *string,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if items == nil || len(*items) == 0 {
		return types.SetNull(types.StringType), diags
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
		return types.SetNull(types.StringType), diags
	}
	out, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return out, diags
}
