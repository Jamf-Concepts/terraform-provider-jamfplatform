// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ebook

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// This file implements the read-merge-write overlay that gives the ebook scope
// its per-category granular ownership, following internal/common/scope/merge.go
// (the ebook union+classes shape is resource-local, so the merge is too). A
// classic scope PUT replaces the ENTIRE <scope> subtree the moment any category
// element is present in the body — and ebook's computer+mobile union is ONE
// subtree: a body carrying only computer categories also wipes the mobile
// categories (wire-probed 2026-07-08 on /ebooks). Update therefore GETs the
// live object, flattens its scope hydrate-all, overlays the declared plan
// categories here, and builds the wire scope from the merged model. Every
// merged field is non-null so the builder emits every category explicitly
// (empty wrappers for empty categories).
//
// All-flag precedence: a merged all-flag that is true empties exactly the
// target categories its AllFlagConflictsWith validator names — never
// limitations/exclusions, which coexist with a true flag and are preserved.

// pickSet returns the declared plan set when the category is managed
// (non-null), else the live value, else an empty set — merged output is always
// non-null so the builder emits every category explicitly. Local reimplement
// of the unexported shared-scope helper.
func pickSet(declared, server types.Set) types.Set {
	if !declared.IsNull() {
		return declared
	}
	if !server.IsNull() {
		return server
	}
	return scope.EmptyStringSet()
}

// pickBool is the all-flag sibling of pickSet: declared wins, else the live
// value, else false.
func pickBool(declared, server types.Bool) types.Bool {
	if !declared.IsNull() {
		return declared
	}
	if !server.IsNull() {
		return server
	}
	return types.BoolValue(false)
}

// derefOrZero returns the pointed-to sub-block model, or its zero value when
// the block is absent. Zero-value types.Set / types.Bool fields report null,
// which is exactly the "unmanaged" signal pickSet/pickBool key on.
func derefOrZero[T any](m *T) T {
	if m != nil {
		return *m
	}
	var zero T
	return zero
}

// mergeEbookScope overlays a declared ebook dual-target-union scope onto the
// live scope. plan nil (scope unmanaged) returns nil — the caller must not send
// a scope element at all. server nil is treated as an empty live scope.
func mergeEbookScope(plan, server *EbookScopeModel) *EbookScopeModel {
	if plan == nil {
		return nil
	}
	var srv EbookScopeModel
	if server != nil {
		srv = *server
	}
	pt, st := plan.TargetsOrZero(), srv.TargetsOrZero()
	targets := &EbookScopeTargetsModel{
		AllComputers:         pickBool(pt.AllComputers, st.AllComputers),
		AllMobileDevices:     pickBool(pt.AllMobileDevices, st.AllMobileDevices),
		AllJssUsers:          pickBool(pt.AllJssUsers, st.AllJssUsers),
		ComputerIDs:          pickSet(pt.ComputerIDs, st.ComputerIDs),
		ComputerGroupIDs:     pickSet(pt.ComputerGroupIDs, st.ComputerGroupIDs),
		MobileDeviceIDs:      pickSet(pt.MobileDeviceIDs, st.MobileDeviceIDs),
		MobileDeviceGroupIDs: pickSet(pt.MobileDeviceGroupIDs, st.MobileDeviceGroupIDs),
		BuildingIDs:          pickSet(pt.BuildingIDs, st.BuildingIDs),
		DepartmentIDs:        pickSet(pt.DepartmentIDs, st.DepartmentIDs),
		UserIDs:              pickSet(pt.UserIDs, st.UserIDs),
		UserGroupIDs:         pickSet(pt.UserGroupIDs, st.UserGroupIDs),
		ClassIDs:             pickSet(pt.ClassIDs, st.ClassIDs),
	}
	// All-flag precedence — empty exactly the categories each flag's
	// AllFlagConflictsWith validator names (see ebookScopeAttributes).
	if targets.AllComputers.ValueBool() {
		targets.ComputerIDs = scope.EmptyStringSet()
		targets.ComputerGroupIDs = scope.EmptyStringSet()
	}
	if targets.AllMobileDevices.ValueBool() {
		targets.MobileDeviceIDs = scope.EmptyStringSet()
		targets.MobileDeviceGroupIDs = scope.EmptyStringSet()
	}
	if targets.AllJssUsers.ValueBool() {
		targets.UserIDs = scope.EmptyStringSet()
		targets.UserGroupIDs = scope.EmptyStringSet()
	}

	pl, sl := derefOrZero(plan.Limitations), derefOrZero(srv.Limitations)
	pe, se := derefOrZero(plan.Exclusions), derefOrZero(srv.Exclusions)
	return &EbookScopeModel{
		Targets: targets,
		Limitations: &EbookScopeLimitationsModel{
			NetworkSegmentIDs:                pickSet(pl.NetworkSegmentIDs, sl.NetworkSegmentIDs),
			DirectoryServiceOrLocalUserNames: pickSet(pl.DirectoryServiceOrLocalUserNames, sl.DirectoryServiceOrLocalUserNames),
			DirectoryServiceUserGroupNames:   pickSet(pl.DirectoryServiceUserGroupNames, sl.DirectoryServiceUserGroupNames),
		},
		Exclusions: &EbookScopeExclusionsModel{
			ComputerIDs:                      pickSet(pe.ComputerIDs, se.ComputerIDs),
			ComputerGroupIDs:                 pickSet(pe.ComputerGroupIDs, se.ComputerGroupIDs),
			MobileDeviceIDs:                  pickSet(pe.MobileDeviceIDs, se.MobileDeviceIDs),
			MobileDeviceGroupIDs:             pickSet(pe.MobileDeviceGroupIDs, se.MobileDeviceGroupIDs),
			BuildingIDs:                      pickSet(pe.BuildingIDs, se.BuildingIDs),
			DepartmentIDs:                    pickSet(pe.DepartmentIDs, se.DepartmentIDs),
			UserIDs:                          pickSet(pe.UserIDs, se.UserIDs),
			UserGroupIDs:                     pickSet(pe.UserGroupIDs, se.UserGroupIDs),
			NetworkSegmentIDs:                pickSet(pe.NetworkSegmentIDs, se.NetworkSegmentIDs),
			DirectoryServiceOrLocalUserNames: pickSet(pe.DirectoryServiceOrLocalUserNames, se.DirectoryServiceOrLocalUserNames),
			DirectoryServiceUserGroupNames:   pickSet(pe.DirectoryServiceUserGroupNames, se.DirectoryServiceUserGroupNames),
		},
	}
}
