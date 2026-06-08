// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package vpp_invitation

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignVPPInvitationResourceModel refreshes a resource model from a GET. General
// scalars are always refreshed; the optional scope block is refreshed only when
// the caller already manages it (state.Scope non-nil) so the server's
// always-returned <scope> doesn't fabricate a block the user never declared.
// invitation_usages (read-only) is always refreshed.
func assignVPPInvitationResourceModel(ctx context.Context, state *VPPInvitationResourceModel, api *proclassic.VppInvitation) {
	if api == nil || api.General == nil {
		return
	}
	g := api.General
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	if g.VppAccount != nil {
		state.VPPAccountID = helpers.StringValueFromIntPtr(g.VppAccount.ID)
	}
	state.DistributionMethod = helpers.StringPointerValueOrNull(g.DistributionMethod)
	state.AutoRegisterManagedUsers = helpers.BoolPointerValueOrNull(g.AutoRegisterManagedUsers)
	state.SenderName = helpers.StringPointerValueOrNull(g.SenderName)
	state.SenderEmailAddress = helpers.StringPointerValueOrNull(g.SenderEmailAddress)
	state.Subject = helpers.StringPointerValueOrNull(g.Subject)
	state.Message = helpers.StringPointerValueOrNull(g.Message)
	state.RequireLogin = helpers.BoolPointerValueOrNull(g.RequireLogin)

	if state.Scope != nil && api.Scope != nil {
		flattenScope(ctx, api.Scope, state.Scope)
	}
	state.InvitationUsages = flattenUsages(api.InvitationUsages)
}

// assignVPPInvitationDataSourceModel populates a DS model from a GET. The DS
// always surfaces scope + usages (read-only lookup).
func assignVPPInvitationDataSourceModel(ctx context.Context, state *VPPInvitationDataSourceModel, api *proclassic.VppInvitation) {
	if api == nil || api.General == nil {
		return
	}
	g := api.General
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	if g.VppAccount != nil {
		state.VPPAccountID = helpers.StringValueFromIntPtr(g.VppAccount.ID)
	}
	state.DistributionMethod = helpers.StringPointerValueOrNull(g.DistributionMethod)
	state.AutoRegisterManagedUsers = helpers.BoolPointerValueOrNull(g.AutoRegisterManagedUsers)
	state.SenderName = helpers.StringPointerValueOrNull(g.SenderName)
	state.SenderEmailAddress = helpers.StringPointerValueOrNull(g.SenderEmailAddress)
	state.Subject = helpers.StringPointerValueOrNull(g.Subject)
	state.Message = helpers.StringPointerValueOrNull(g.Message)
	state.RequireLogin = helpers.BoolPointerValueOrNull(g.RequireLogin)

	state.Scope = &scope.UserScopeModel{
		Limitations: &scope.UserScopeLimitationsModel{},
		Exclusions:  &scope.UserScopeExclusionsModel{},
	}
	if api.Scope != nil {
		flattenScope(ctx, api.Scope, state.Scope)
	}
	state.InvitationUsages = flattenUsages(api.InvitationUsages)
}

func flattenScope(ctx context.Context, s *proclassic.VppInvitationScope, state *scope.UserScopeModel) {
	state.AllJssUsers = helpers.BoolPointerValueOrNull(s.AllJssUsers)
	state.JssUserIDs = flattenIDNameSet(ctx, jssUsersSlice(s.JssUsers))
	state.JssUserGroupIDs = flattenIDNameSet(ctx, jssUserGroupsSlice(s.JssUserGroups))

	if state.Limitations != nil {
		state.Limitations.DirectoryServiceUserGroupNames = flattenNameSet(ctx, limitationsUserGroupsSlice(s.Limitations))
	}
	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.JssUserIDs = flattenIDNameSet(ctx, exclJssUsersSlice(e.JssUsers))
		state.Exclusions.JssUserGroupIDs = flattenIDNameSet(ctx, exclJssUserGroupsSlice(e.JssUserGroups))
		state.Exclusions.DirectoryServiceUserGroupNames = flattenExclDSGroupSet(ctx, exclDSGroupsSlice(e.UserGroups))
	}
}

// ---- scope sub-slice accessors -------------------------------------------------

func jssUsersSlice(u *proclassic.VppInvitationScopeJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func jssUserGroupsSlice(g *proclassic.VppInvitationScopeJssUserGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.UserGroup
}

func limitationsUserGroupsSlice(l *proclassic.VppInvitationScopeLimitations) *[]proclassic.IDName {
	if l == nil || l.UserGroups == nil {
		return nil
	}
	return l.UserGroups.UserGroup
}

func exclJssUsersSlice(u *proclassic.VppInvitationScopeExclusionsJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func exclJssUserGroupsSlice(g *proclassic.VppInvitationScopeExclusionsJssUserGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.UserGroup
}

func exclDSGroupsSlice(g *proclassic.VppInvitationScopeExclusionsUserGroups) *[]proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem {
	if g == nil {
		return nil
	}
	return g.UserGroup
}

// ---- set flatteners ------------------------------------------------------------

func flattenIDNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenIDSlice(ctx, items, func(i proclassic.IDName) *int { return i.ID })
	return out
}

func flattenNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenNameSlice(ctx, items, func(i proclassic.IDName) *string { return i.Name })
	return out
}

func flattenExclDSGroupSet(ctx context.Context, items *[]proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem) types.Set {
	out, _ := scope.FlattenNameSlice(ctx, items, func(i proclassic.VppInvitationScopeExclusionsUserGroupsUserGroupItem) *string { return i.Name })
	return out
}
