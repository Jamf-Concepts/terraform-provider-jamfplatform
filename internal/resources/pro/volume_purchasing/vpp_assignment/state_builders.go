// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_assignment

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignVPPAssignmentResourceModel refreshes a resource model from a GET. General
// scalars are always refreshed; each content Set is refreshed only when the
// caller already manages it (state Set non-null) so the server's always-returned
// collections don't fabricate management of a collection the user omitted. The
// optional scope block is refreshed only when state.Scope is non-nil (mirrors the
// content opt-out + the vpp_invitation precedent). Item names are discarded
// (only adam_id round-trips).
func assignVPPAssignmentResourceModel(ctx context.Context, state *VPPAssignmentResourceModel, api *proclassic.VppAssignment) {
	if api == nil || api.General == nil {
		return
	}
	g := api.General
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.VPPAdminAccountID = intStringOrNull(g.VppAdminAccountID)
	state.VPPAdminAccountName = helpers.StringPointerValueOrNull(g.VppAdminAccountName)

	// Content: refresh only managed (non-null) sets. The flatten returns a known
	// (possibly empty) Set so a managed-but-now-empty collection reconciles to []
	// rather than null (these are plain Optional attrs — a null after an empty
	// config is a "provider produced inconsistent result" error).
	if !state.IosAppAdamIDs.IsNull() {
		state.IosAppAdamIDs = flattenAdamSet(ctx, iosAppAdamIDs(api.IosApps))
	}
	if !state.MacAppAdamIDs.IsNull() {
		state.MacAppAdamIDs = flattenAdamSet(ctx, macAppAdamIDs(api.MacApps))
	}
	if !state.EbookAdamIDs.IsNull() {
		state.EbookAdamIDs = flattenAdamSet(ctx, ebookAdamIDs(api.Ebooks))
	}

	if state.Scope != nil && api.Scope != nil {
		flattenScope(ctx, api.Scope, state.Scope)
	}
}

// assignVPPAssignmentDataSourceModel populates a DS model from a GET. The DS
// always surfaces scope + the three content lists (read-only lookup).
func assignVPPAssignmentDataSourceModel(ctx context.Context, state *VPPAssignmentDataSourceModel, api *proclassic.VppAssignment) {
	if api == nil || api.General == nil {
		return
	}
	g := api.General
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.VPPAdminAccountID = intStringOrNull(g.VppAdminAccountID)
	state.VPPAdminAccountName = helpers.StringPointerValueOrNull(g.VppAdminAccountName)

	state.IosApps = flattenContentList(iosAppItems(api.IosApps))
	state.MacApps = flattenContentList(macAppItems(api.MacApps))
	state.Ebooks = flattenContentList(ebookItems(api.Ebooks))

	state.Scope = &scope.UserScopeModel{
		Targets:     &scope.UserScopeTargetsModel{},
		Limitations: &scope.UserScopeLimitationsModel{},
		Exclusions:  &scope.UserScopeExclusionsModel{},
	}
	if api.Scope != nil {
		flattenScope(ctx, api.Scope, state.Scope)
	}
}

func flattenScope(ctx context.Context, s *proclassic.VppAssignmentScope, state *scope.UserScopeModel) {
	if state.Targets != nil {
		state.Targets.AllJssUsers = helpers.BoolPointerValueOrNull(s.AllJssUsers)
		state.Targets.JssUserIDs = scope.FlattenIDNameSet(ctx, jssUsersSlice(s.JssUsers))
		state.Targets.JssUserGroupIDs = scope.FlattenIDNameSet(ctx, jssUserGroupsSlice(s.JssUserGroups))
	}

	if state.Limitations != nil {
		state.Limitations.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, limitationsUserGroupsSlice(s.Limitations))
	}
	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.JssUserIDs = scope.FlattenIDNameSet(ctx, exclJssUsersSlice(e.JssUsers))
		state.Exclusions.JssUserGroupIDs = scope.FlattenIDNameSet(ctx, exclJssUserGroupsSlice(e.JssUserGroups))
		state.Exclusions.DirectoryServiceUserGroupNames = flattenExclDSGroupSet(ctx, exclDSGroupsSlice(e.UserGroups))
	}
}

// ---- content accessors / flatteners --------------------------------------------

func iosAppAdamIDs(a *proclassic.VppAssignmentIosApps) []*int {
	if a == nil || a.IosApp == nil {
		return nil
	}
	out := make([]*int, 0, len(*a.IosApp))
	for _, it := range *a.IosApp {
		out = append(out, it.AdamID)
	}
	return out
}

func macAppAdamIDs(a *proclassic.VppAssignmentMacApps) []*int {
	if a == nil || a.MacApp == nil {
		return nil
	}
	out := make([]*int, 0, len(*a.MacApp))
	for _, it := range *a.MacApp {
		out = append(out, it.AdamID)
	}
	return out
}

func ebookAdamIDs(a *proclassic.VppAssignmentEbooks) []*int {
	if a == nil || a.Ebook == nil {
		return nil
	}
	out := make([]*int, 0, len(*a.Ebook))
	for _, it := range *a.Ebook {
		out = append(out, it.AdamID)
	}
	return out
}

// flattenAdamSet builds a known Set[Int64] from the server's adam_id pointers.
// Returns a known empty Set (NOT null) for an empty/absent collection so a
// managed-but-now-cleared content set reconciles to [] rather than null.
func flattenAdamSet(ctx context.Context, ids []*int) types.Set {
	values := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == nil {
			continue
		}
		values = append(values, int64(*id))
	}
	out, _ := types.SetValueFrom(ctx, types.Int64Type, values)
	return out
}

// contentName / contentAdamID adapt one element of each content type into the DS
// {adam_id, name} object. Defined per type because the SDK item structs share no
// interface.

func iosAppItems(a *proclassic.VppAssignmentIosApps) []contentTriple {
	if a == nil || a.IosApp == nil {
		return nil
	}
	out := make([]contentTriple, 0, len(*a.IosApp))
	for _, it := range *a.IosApp {
		out = append(out, contentTriple{adamID: it.AdamID, name: it.Name})
	}
	return out
}

func macAppItems(a *proclassic.VppAssignmentMacApps) []contentTriple {
	if a == nil || a.MacApp == nil {
		return nil
	}
	out := make([]contentTriple, 0, len(*a.MacApp))
	for _, it := range *a.MacApp {
		out = append(out, contentTriple{adamID: it.AdamID, name: it.Name})
	}
	return out
}

func ebookItems(a *proclassic.VppAssignmentEbooks) []contentTriple {
	if a == nil || a.Ebook == nil {
		return nil
	}
	out := make([]contentTriple, 0, len(*a.Ebook))
	for _, it := range *a.Ebook {
		out = append(out, contentTriple{adamID: it.AdamID, name: it.Name})
	}
	return out
}

// contentTriple is a type-erased content item for the DS list flatten.
type contentTriple struct {
	adamID *int
	name   *string
}

// flattenContentList builds a Computed list of {adam_id, name} objects. Always
// returns a known (possibly empty) list so the Computed attribute never stays
// unknown after read.
func flattenContentList(items []contentTriple) types.List {
	elems := make([]attr.Value, 0, len(items))
	for _, it := range items {
		adam := types.Int64Null()
		if it.adamID != nil {
			adam = types.Int64Value(int64(*it.adamID))
		}
		elems = append(elems, types.ObjectValueMust(contentAttrTypes, map[string]attr.Value{
			"adam_id": adam,
			"name":    helpers.StringPointerValueOrNull(it.name),
		}))
	}
	return types.ListValueMust(contentObjectType, elems)
}

// ---- scope sub-slice accessors -------------------------------------------------

func jssUsersSlice(u *proclassic.VppAssignmentScopeJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func jssUserGroupsSlice(g *proclassic.VppAssignmentScopeJssUserGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.UserGroup
}

func limitationsUserGroupsSlice(l *proclassic.VppAssignmentScopeLimitations) *[]proclassic.IDName {
	if l == nil || l.UserGroups == nil {
		return nil
	}
	return l.UserGroups.UserGroup
}

func exclJssUsersSlice(u *proclassic.VppAssignmentScopeExclusionsJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func exclJssUserGroupsSlice(g *proclassic.VppAssignmentScopeExclusionsJssUserGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.UserGroup
}

func exclDSGroupsSlice(g *proclassic.VppAssignmentScopeExclusionsUserGroups) *[]proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem {
	if g == nil {
		return nil
	}
	return g.UserGroup
}

// ---- scope set flatteners ------------------------------------------------------

func flattenExclDSGroupSet(ctx context.Context, items *[]proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem) types.Set {
	out, _ := scope.FlattenNameSlice(ctx, items, func(i proclassic.VppAssignmentScopeExclusionsUserGroupsUserGroupItem) *string { return i.Name })
	return out
}
