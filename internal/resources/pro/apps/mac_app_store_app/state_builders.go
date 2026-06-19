// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mac_app_store_app

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignMacAppResourceModel populates a resource model from the SDK
// MacApplication response. general is always refreshed (required block). The
// optional sections (scope / self_service / vpp) are only refreshed when the
// caller (plan or current state) already manages them: the classic server
// echoes every section on GET with default values, so populating an unmanaged
// section would violate the framework's "produced inconsistent result after
// apply" check (plan said null, we'd return a populated object). See
// feedback_server_derived_echo_attrs at block granularity.
func assignMacAppResourceModel(ctx context.Context, state *MacAppResourceModel, a *proclassic.MacApplication) diag.Diagnostics {
	var diags diag.Diagnostics
	if a == nil {
		return diags
	}

	if id := extractMacAppID(a); id != "" {
		state.ID = types.StringValue(id)
	}

	if state.General == nil {
		state.General = &MacAppGeneralModel{}
	}
	flattenMacAppGeneral(a.General, state.General)

	if state.Scope != nil && a.Scope != nil {
		flattenMacAppScope(ctx, a.Scope, state.Scope)
	}
	if state.SelfService != nil && a.SelfService != nil {
		flattenMacAppSelfService(a.SelfService, state.SelfService)
	}
	if state.Vpp != nil && a.Vpp != nil {
		flattenMacAppVpp(a.Vpp, state.Vpp)
	}

	return diags
}

func flattenMacAppGeneral(g *proclassic.MacApplicationGeneral, state *MacAppGeneralModel) {
	if g == nil {
		return
	}
	state.ID = helpers.StringValueFromIntPtr(g.ID)
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.Version = helpers.StringPointerValueOrNull(g.Version)
	state.BundleID = helpers.StringPointerValueOrNull(g.BundleID)
	state.URL = helpers.StringPointerValueOrNull(g.URL)
	state.IsFree = helpers.PreferCurrentBoolPointer(g.IsFree, state.IsFree)
	state.DeploymentType = helpers.PreferCurrentStringPointer(g.DeploymentType, state.DeploymentType)

	if g.Category != nil {
		state.CategoryID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Category.ID), state.CategoryID)
		state.CategoryName = helpers.StringPointerValueOrNull(g.Category.Name)
	} else {
		state.CategoryID = helpers.PreferCurrentStringPointer(nil, state.CategoryID)
		state.CategoryName = types.StringNull()
	}

	if g.Site != nil {
		state.SiteID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Site.ID), state.SiteID)
		state.SiteName = helpers.StringPointerValueOrNull(g.Site.Name)
	} else {
		state.SiteID = helpers.PreferCurrentStringPointer(nil, state.SiteID)
		state.SiteName = types.StringNull()
	}
}

func flattenMacAppScope(ctx context.Context, s *proclassic.MacApplicationScope, state *scope.ComputerScopeModelNoIbeacons) {
	// Targets are gated on caller management, mirroring the limitations /
	// exclusions sub-blocks below: populating a targets block the user did not
	// declare would violate the framework's "produced inconsistent result after
	// apply" check (plan said null, we would return a populated object).
	if state.Targets != nil {
		state.Targets.AllComputers = helpers.PreferCurrentBoolPointer(s.AllComputers, state.Targets.AllComputers)
		state.Targets.AllJssUsers = helpers.PreferCurrentBoolPointer(s.AllJssUsers, state.Targets.AllJssUsers)

		state.Targets.ComputerIDs = flattenComputerItemSet(ctx, s.Computers)
		state.Targets.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, computerGroupSlice(s.ComputerGroups))
		state.Targets.BuildingIDs = scope.FlattenIDNameSet(ctx, buildingSlice(s.Buildings))
		state.Targets.DepartmentIDs = scope.FlattenIDNameSet(ctx, departmentSlice(s.Departments))
		state.Targets.UserIDs = scope.FlattenIDNameSet(ctx, jssUserSlice(s.JssUsers))
		state.Targets.UserGroupIDs = scope.FlattenIDNameSet(ctx, jssUserGroupSlice(s.JssUserGroups))
	}

	if state.Limitations != nil && s.Limitations != nil {
		l := s.Limitations
		state.Limitations.NetworkSegmentIDs = scope.FlattenIDNameSet(ctx, limitationsSegmentSlice(l.NetworkSegments))
		state.Limitations.DirectoryServiceOrLocalUserNames = scope.FlattenNameSet(ctx, limitationsUserSlice(l.Users))
		state.Limitations.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, limitationsUserGroupSlice(l.UserGroups))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.ComputerIDs = flattenExclComputerItemSet(ctx, e.Computers)
		state.Exclusions.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, exclComputerGroupSlice(e.ComputerGroups))
		state.Exclusions.BuildingIDs = scope.FlattenIDNameSet(ctx, exclBuildingSlice(e.Buildings))
		state.Exclusions.DepartmentIDs = scope.FlattenIDNameSet(ctx, exclDepartmentSlice(e.Departments))
		state.Exclusions.UserIDs = scope.FlattenIDNameSet(ctx, exclJssUserSlice(e.JssUsers))
		state.Exclusions.UserGroupIDs = scope.FlattenIDNameSet(ctx, exclJssUserGroupSlice(e.JssUserGroups))
		state.Exclusions.NetworkSegmentIDs = flattenExclNetworkSegmentSet(ctx, e.NetworkSegments)
		state.Exclusions.DirectoryServiceOrLocalUserNames = flattenExclUsersNameSet(ctx, e.Users)
		state.Exclusions.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, exclUserGroupSlice(e.UserGroups))
	}
}

func flattenMacAppSelfService(ss *proclassic.MacApplicationSelfService, state *MacAppSelfServiceModel) {
	state.InstallButtonText = helpers.PreferCurrentStringPointer(ss.InstallButtonText, state.InstallButtonText)
	state.SelfServiceDescription = helpers.PreferCurrentStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.ForceUsersToViewDescription = helpers.PreferCurrentBoolPointer(ss.ForceUsersToViewDescription, state.ForceUsersToViewDescription)
	state.FeatureOnMainPage = helpers.PreferCurrentBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)

	var apiEnabled *bool
	var apiMethod *string
	if ss.Notification != nil {
		apiEnabled = ss.Notification.Enabled
		apiMethod = ss.Notification.Method
	}
	state.NotificationEnabled = helpers.PreferCurrentBoolPointer(apiEnabled, state.NotificationEnabled)
	state.NotificationMethod = helpers.PreferCurrentStringPointer(apiMethod, state.NotificationMethod)
	state.NotificationSubject = helpers.PreferCurrentStringPointer(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = helpers.PreferCurrentStringPointer(ss.NotificationMessage, state.NotificationMessage)

	if state.SelfServiceIcon != nil && ss.SelfServiceIcon != nil {
		state.SelfServiceIcon.ID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(ss.SelfServiceIcon.ID), state.SelfServiceIcon.ID)
		state.SelfServiceIcon.URI = helpers.PreferCurrentStringPointer(ss.SelfServiceIcon.URI, state.SelfServiceIcon.URI)
	}

	if state.SelfServiceCategories != nil && ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil {
		flattenMacAppSelfServiceCategories(*ss.SelfServiceCategories.Category, state)
	}
}

// flattenMacAppSelfServiceCategories refreshes the managed category set,
// matching server items to existing state items by ID so caller-authored
// values (display_in / feature_in) stick across refreshes. Server items not in
// state are appended; the set is keyed by category ID.
func flattenMacAppSelfServiceCategories(api []proclassic.MacApplicationSelfServiceSelfServiceCategoriesCategoryItem, state *MacAppSelfServiceModel) {
	byID := make(map[string]MacAppSelfServiceCategoryModel, len(state.SelfServiceCategories))
	for _, c := range state.SelfServiceCategories {
		byID[c.ID.ValueString()] = c
	}

	out := make([]MacAppSelfServiceCategoryModel, 0, len(api))
	for _, c := range api {
		idStr := ""
		if s := helpers.StringFromIntPtr(c.ID); s != nil {
			idStr = *s
		}
		current := byID[idStr]
		out = append(out, MacAppSelfServiceCategoryModel{
			ID:        types.StringValue(idStr),
			Name:      helpers.PreferCurrentStringPointer(c.Name, current.Name),
			DisplayIn: helpers.PreferCurrentBoolPointer(c.DisplayIn, current.DisplayIn),
			FeatureIn: helpers.PreferCurrentBoolPointer(c.FeatureIn, current.FeatureIn),
		})
	}
	state.SelfServiceCategories = out
}

func flattenMacAppVpp(v *proclassic.MacApplicationVpp, state *MacAppVppModel) {
	state.AssignVppDeviceBasedLicenses = helpers.PreferCurrentBoolPointer(v.AssignVppDeviceBasedLicenses, state.AssignVppDeviceBasedLicenses)
	state.VppAdminAccountID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(v.VppAdminAccountID), state.VppAdminAccountID)
	state.TotalVppLicenses = helpers.Int64FromIntPtr(v.TotalVppLicenses)
	state.RemainingVppLicenses = helpers.Int64FromIntPtr(v.RemainingVppLicenses)
	state.UsedVppLicenses = helpers.Int64FromIntPtr(v.UsedVppLicenses)
}

// ---- scope sub-slice accessors -------------------------------------------------

func computerGroupSlice(g *proclassic.MacApplicationScopeComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func buildingSlice(b *proclassic.MacApplicationScopeBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func departmentSlice(d *proclassic.MacApplicationScopeDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func jssUserSlice(u *proclassic.MacApplicationScopeJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func jssUserGroupSlice(u *proclassic.MacApplicationScopeJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func limitationsSegmentSlice(s *proclassic.MacApplicationScopeLimitationsNetworkSegments) *[]proclassic.IDName {
	if s == nil {
		return nil
	}
	return s.NetworkSegment
}

func limitationsUserSlice(u *proclassic.MacApplicationScopeLimitationsUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func limitationsUserGroupSlice(u *proclassic.MacApplicationScopeLimitationsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func exclComputerGroupSlice(g *proclassic.MacApplicationScopeExclusionsComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func exclBuildingSlice(b *proclassic.MacApplicationScopeExclusionsBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func exclDepartmentSlice(d *proclassic.MacApplicationScopeExclusionsDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func exclJssUserSlice(u *proclassic.MacApplicationScopeExclusionsJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func exclJssUserGroupSlice(u *proclassic.MacApplicationScopeExclusionsJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func exclUserGroupSlice(u *proclassic.MacApplicationScopeExclusionsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

// ---- set flatteners ------------------------------------------------------------

func flattenComputerItemSet(ctx context.Context, c *proclassic.MacApplicationScopeComputers) types.Set {
	if c == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, c.Computer, func(i proclassic.MacApplicationScopeComputersComputerItem) *int { return i.ID })
	return out
}

func flattenExclComputerItemSet(ctx context.Context, c *proclassic.MacApplicationScopeExclusionsComputers) types.Set {
	if c == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, c.Computer, func(i proclassic.MacApplicationScopeExclusionsComputersComputerItem) *int { return i.ID })
	return out
}

func flattenExclNetworkSegmentSet(ctx context.Context, n *proclassic.MacApplicationScopeExclusionsNetworkSegments) types.Set {
	if n == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, n.NetworkSegment, func(i proclassic.MacApplicationScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int { return i.ID })
	return out
}

func flattenExclUsersNameSet(ctx context.Context, u *proclassic.MacApplicationScopeExclusionsUsers) types.Set {
	if u == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenNameSlice(ctx, u.User, func(i proclassic.MacApplicationScopeExclusionsUsersUserItem) *string { return i.Name })
	return out
}
