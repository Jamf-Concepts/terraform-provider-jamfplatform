// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import "github.com/hashicorp/terraform-plugin-framework/types"

// RefreshManagedSet gates a scope category refresh on ownership: the wire
// value is adopted only when the category is managed — non-null in `current`,
// the model being written to state (the plan on create/update, the prior
// state on read; see STYLE_GUIDE.md §Scope helper for why the gate target is
// never a separate prior-state reference). An unmanaged (null) category stays
// null so members maintained in the admin UI never enter state or plans.
//
// includeUnmanaged bypasses the gate for the two hydrate-everything paths —
// import and `terraform query` config generation — where there is no plan to
// stay consistent with and full visibility is the point.
func RefreshManagedSet(current, wire types.Set, includeUnmanaged bool) types.Set {
	if includeUnmanaged || !current.IsNull() {
		return wire
	}
	return types.SetNull(types.StringType)
}

// RefreshManagedBool is the all-flag sibling of RefreshManagedSet. A nil wire
// pointer (the classic GET always echoes the flags, so this is defensive)
// keeps the current value rather than nulling a managed flag.
func RefreshManagedBool(current types.Bool, wire *bool, includeUnmanaged bool) types.Bool {
	if !includeUnmanaged && current.IsNull() {
		return types.BoolNull()
	}
	if wire == nil {
		return current
	}
	return types.BoolPointerValue(wire)
}
