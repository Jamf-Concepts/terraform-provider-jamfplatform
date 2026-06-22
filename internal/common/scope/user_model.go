// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package scope

import "github.com/hashicorp/terraform-plugin-framework/types"

// UserScopeModel is the Terraform model for a Jamf classic user-based <scope>
// block. Field names + tfsdk tags match UserScopeAttributes. This is the
// USER-BASED scope subset (a third scope shape distinct from
// computer/mobile): targets are Jamf Pro users / Jamf Pro user groups
// (id-keyed) plus the all_jss_users flag; limitations and exclusions carry
// directory-service (LDAP) user groups by NAME.
//
// Write semantics (wire-probed): scope is ALWAYS-EMIT — the server merges on
// PUT (omitting a collection retains it), and within a present collection the
// write is a full replace. To make scope declarative, the resource's input
// builder emits the full <scope> skeleton (empty elements to clear) whenever
// this block is declared. A nil *UserScopeModel omits <scope> entirely and
// leaves the server's scope untouched.
//
// Consumed by vpp_invitation and vpp_assignment. The build/flatten glue stays
// per-resource because VppInvitation* and VppAssignment* are distinct generated
// SDK structs with no shared interface. See STYLE_GUIDE.md §Scope helper.
type UserScopeModel struct {
	Targets     *UserScopeTargetsModel     `tfsdk:"targets"`
	Limitations *UserScopeLimitationsModel `tfsdk:"limitations"`
	Exclusions  *UserScopeExclusionsModel  `tfsdk:"exclusions"`
}

// UserScopeTargetsModel models <scope> targets — the all_jss_users flag plus
// the Jamf Pro user / user-group ID sets, mirroring the admin UI's Targets tab.
type UserScopeTargetsModel struct {
	AllJssUsers     types.Bool `tfsdk:"all_jss_users"`
	JssUserIDs      types.Set  `tfsdk:"jss_user_ids"`
	JssUserGroupIDs types.Set  `tfsdk:"jss_user_group_ids"`
}

// TargetsOrZero returns the targets sub-model, or a zero value with null
// fields when the block was omitted, so input-builders can read target fields
// without a nil-guard.
func (m UserScopeModel) TargetsOrZero() UserScopeTargetsModel {
	if m.Targets != nil {
		return *m.Targets
	}
	return UserScopeTargetsModel{
		AllJssUsers:     types.BoolNull(),
		JssUserIDs:      types.SetNull(types.StringType),
		JssUserGroupIDs: types.SetNull(types.StringType),
	}
}

// UserScopeLimitationsModel models <scope><limitations>. The UI exposes only
// "Directory Service User Groups". These are NAME-keyed (wire-probed:
// PUT-by-id → 409, PUT-by-name → 201); the SDK's IDName item type is a superset
// and only its name is populated.
type UserScopeLimitationsModel struct {
	DirectoryServiceUserGroupNames types.Set `tfsdk:"directory_service_user_group_names"`
}

// UserScopeExclusionsModel models <scope><exclusions>. The UI exposes Users /
// User Groups (id-keyed Jamf objects) and Directory Service User Groups
// (name-keyed, wire-confirmed name-only).
type UserScopeExclusionsModel struct {
	JssUserIDs                     types.Set `tfsdk:"jss_user_ids"`
	JssUserGroupIDs                types.Set `tfsdk:"jss_user_group_ids"`
	DirectoryServiceUserGroupNames types.Set `tfsdk:"directory_service_user_group_names"`
}
