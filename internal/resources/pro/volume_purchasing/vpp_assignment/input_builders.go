// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// buildVPPAssignmentInput projects the plan into an SDK *proclassic.VppAssignmentPost
// for Create / Update.
//
// General scalars (name, vpp_admin_account_id) are always-emitted. The three
// content collections are OPT-OUT, keyed on IsNull at the call site (NOT on
// whether the built slice is nil):
//   - null set        → leave the wire block nil → omitted → server retains it.
//   - empty set ([])  → emit a non-nil wrapper with a nil inner slice → marshals
//     as <ios_apps></ios_apps> → server clears it.
//   - populated set   → emit the full list → server full-replaces it.
//
// Content item name is server-resolved (wire-probed) — only adam_id is sent.
//
// Scope, when declared, is emitted as a FULL skeleton (every wrapper present,
// empty when its set is null/empty) so the server full-replaces it. A nil Scope
// omits <scope> entirely and leaves the server's scope untouched.
func buildVPPAssignmentInput(ctx context.Context, plan VPPAssignmentResourceModel) (*proclassic.VppAssignmentPost, diag.Diagnostics) {
	var diags diag.Diagnostics

	general := &proclassic.VppAssignmentPostGeneral{
		Name:              helpers.OptionalStringPointer(plan.Name),
		VppAdminAccountID: helpers.StringIDPtr(plan.VPPAdminAccountID),
	}
	out := &proclassic.VppAssignmentPost{General: general}

	// ios_apps (opt-out).
	if !plan.IosAppAdamIDs.IsNull() {
		items, d := buildAdamItems(ctx, plan.IosAppAdamIDs, func(id int) proclassic.VppAssignmentPostIosAppsIosAppItem {
			return proclassic.VppAssignmentPostIosAppsIosAppItem{AdamID: &id}
		})
		diags.Append(d...)
		out.IosApps = &proclassic.VppAssignmentPostIosApps{IosApp: items}
	}

	// mac_apps (opt-out).
	if !plan.MacAppAdamIDs.IsNull() {
		items, d := buildAdamItems(ctx, plan.MacAppAdamIDs, func(id int) proclassic.VppAssignmentPostMacAppsMacAppItem {
			return proclassic.VppAssignmentPostMacAppsMacAppItem{AdamID: &id}
		})
		diags.Append(d...)
		out.MacApps = &proclassic.VppAssignmentPostMacApps{MacApp: items}
	}

	// ebooks (opt-out).
	if !plan.EbookAdamIDs.IsNull() {
		items, d := buildAdamItems(ctx, plan.EbookAdamIDs, func(id int) proclassic.VppAssignmentPostEbooksEbookItem {
			return proclassic.VppAssignmentPostEbooksEbookItem{AdamID: &id}
		})
		diags.Append(d...)
		out.Ebooks = &proclassic.VppAssignmentPostEbooks{Ebook: items}
	}

	if plan.Scope != nil {
		s, d := buildScope(ctx, plan.Scope)
		diags.Append(d...)
		out.Scope = s
	}

	return out, diags
}

// buildScope emits the full <scope> skeleton (always-emit). Every collection
// wrapper is present so the server full-replaces it; a nil inner slice marshals
// as an empty element, which clears that collection. The scope wrapper is the
// Post type, but its sub-blocks reuse the shared non-Post VppAssignmentScope*
// types.
func buildScope(ctx context.Context, m *scope.UserScopeModel) (*proclassic.VppAssignmentPostScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	t := m.TargetsOrZero()
	s := &proclassic.VppAssignmentPostScope{
		AllJssUsers: helpers.OptionalBoolPointer(t.AllJssUsers),
	}

	jssUsers, d := scope.BuildIDSlice(ctx, t.JssUserIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	s.JssUsers = &proclassic.VppAssignmentScopeJssUsers{User: jssUsers}

	jssUserGroups, d := scope.BuildIDSlice(ctx, t.JssUserGroupIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
	diags.Append(d...)
	s.JssUserGroups = &proclassic.VppAssignmentScopeJssUserGroups{UserGroup: jssUserGroups}

	// limitations.user_groups: directory-service groups, NAME-keyed (populate Name).
	var limNames *[]proclassic.IDName
	if m.Limitations != nil {
		limNames, d = scope.BuildNameSlice(ctx, m.Limitations.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
			n := name
			return proclassic.IDName{Name: &n}
		})
		diags.Append(d...)
	}
	s.Limitations = &proclassic.VppAssignmentScopeLimitations{
		UserGroups: &proclassic.VppAssignmentScopeLimitationsUserGroups{UserGroup: limNames},
	}

	// exclusions: id-keyed jss_users / jss_user_groups + name-keyed user_groups.
	excl := &proclassic.VppAssignmentScopeExclusions{}
	var exclJssUsers, exclJssUserGroups *[]proclassic.IDName
	var exclDSGroups *[]proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem
	if m.Exclusions != nil {
		exclJssUsers, d = scope.BuildIDSlice(ctx, m.Exclusions.JssUserIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
		diags.Append(d...)
		exclJssUserGroups, d = scope.BuildIDSlice(ctx, m.Exclusions.JssUserGroupIDs, func(id int) proclassic.IDName { return proclassic.IDName{ID: &id} })
		diags.Append(d...)
		exclDSGroups, d = scope.BuildNameSlice(ctx, m.Exclusions.DirectoryServiceUserGroupNames, func(name string) proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem {
			n := name
			return proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem{Name: &n}
		})
		diags.Append(d...)
	}
	excl.JssUsers = &proclassic.VppAssignmentScopeExclusionsJssUsers{User: exclJssUsers}
	excl.JssUserGroups = &proclassic.VppAssignmentScopeExclusionsJssUserGroups{UserGroup: exclJssUserGroups}
	excl.UserGroups = &proclassic.VppAssignmentScopeExclusionsUserGroups{UserGroup: exclDSGroups}
	s.Exclusions = excl

	return s, diags
}

// buildAdamItems projects a Terraform Set[Int64] of Apple catalog adam IDs into
// the SDK item pointer-slice. Returns nil (so the wrapper marshals as an empty,
// clearing element) for an empty/unknown set. The caller decides whether to emit
// the wrapper at all (the opt-out null check lives at the call site).
func buildAdamItems[T any](ctx context.Context, set types.Set, mk func(int) T) (*[]T, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var elements []int64
	diags.Append(set.ElementsAs(ctx, &elements, false)...)
	if diags.HasError() || len(elements) == 0 {
		return nil, diags
	}
	out := make([]T, 0, len(elements))
	for _, raw := range elements {
		out = append(out, mk(int(raw)))
	}
	return &out, diags
}
