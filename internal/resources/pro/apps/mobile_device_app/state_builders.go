// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_app

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignMobileAppResourceModel populates a resource model from the SDK
// MobileDeviceApplication response. general is always refreshed (required block).
// The optional sections (scope / self_service / vpp / app_configuration) are only
// refreshed when the caller (plan or current state) already manages them: the
// classic server echoes every section on GET with default values, so populating
// an unmanaged section would violate the framework's "produced inconsistent
// result after apply" check. See feedback_server_derived_echo_attrs at block
// granularity.
func assignMobileAppResourceModel(ctx context.Context, state *MobileAppResourceModel, a *proclassic.MobileDeviceApplication) diag.Diagnostics {
	var diags diag.Diagnostics
	if a == nil {
		return diags
	}

	if id := extractMobileAppID(a); id != "" {
		state.ID = types.StringValue(id)
	}

	if state.General == nil {
		state.General = &MobileAppGeneralModel{}
	}
	flattenMobileAppGeneral(a.General, state.General)

	if state.Scope != nil && a.Scope != nil {
		flattenMobileAppScope(ctx, a.Scope, state.Scope)
	}
	if state.SelfService != nil && a.SelfService != nil {
		flattenMobileAppSelfService(a.SelfService, state.SelfService)
	}
	if state.Vpp != nil && a.Vpp != nil {
		flattenMobileAppVpp(a.Vpp, state.Vpp)
	}
	if state.AppConfiguration != nil && a.AppConfiguration != nil {
		state.AppConfiguration.Preferences = helpers.StringPointerValueOrNull(a.AppConfiguration.Preferences)
	}

	return diags
}

func flattenMobileAppGeneral(g *proclassic.MobileDeviceApplicationGeneral, state *MobileAppGeneralModel) {
	if g == nil {
		return
	}
	state.ID = helpers.StringValueFromIntPtr(g.ID)
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.Version = helpers.StringPointerValueOrNull(g.Version)
	state.BundleID = helpers.StringPointerValueOrNull(g.BundleID)
	state.OsType = helpers.StringPointerValueOrNull(g.OsType)

	// Server-managed read-only fields.
	state.DisplayName = helpers.StringPointerValueOrNull(g.DisplayName)
	state.Description = helpers.StringPointerValueOrNull(g.Description)
	state.InternalApp = helpers.BoolPointerValueOrNull(g.InternalApp)

	// Optional+Computed echoes.
	state.IsFree = preferCurrentBoolPointer(g.Free, state.IsFree)
	state.DeploymentType = preferCurrentStringPointer(g.DeploymentType, state.DeploymentType)
	state.ExternalURL = preferCurrentStringPointer(g.ExternalURL, state.ExternalURL)
	state.ItunesStoreURL = preferCurrentStringPointer(g.ItunesStoreURL, state.ItunesStoreURL)
	state.ItunesCountryRegion = preferCurrentStringPointer(g.ItunesCountryRegion, state.ItunesCountryRegion)
	state.ItunesSyncTime = preferCurrentInt64Pointer(g.ItunesSyncTime, state.ItunesSyncTime)
	state.MakeAvailableAfterInstall = preferCurrentBoolPointer(g.MakeAvailableAfterInstall, state.MakeAvailableAfterInstall)
	state.KeepDescriptionAndIconUpToDate = preferCurrentBoolPointer(g.KeepDescriptionAndIconUpToDate, state.KeepDescriptionAndIconUpToDate)
	state.KeepAppUpdatedOnDevices = preferCurrentBoolPointer(g.KeepAppUpdatedOnDevices, state.KeepAppUpdatedOnDevices)
	state.DeployAsManagedApp = preferCurrentBoolPointer(g.DeployAsManagedApp, state.DeployAsManagedApp)
	state.TakeOverManagement = preferCurrentBoolPointer(g.TakeOverManagement, state.TakeOverManagement)
	state.DeployAutomatically = preferCurrentBoolPointer(g.DeployAutomatically, state.DeployAutomatically)
	state.RemoveAppWhenMDMProfileIsRemoved = preferCurrentBoolPointer(g.RemoveAppWhenMDMProfileIsRemoved, state.RemoveAppWhenMDMProfileIsRemoved)
	state.PreventBackupOfAppData = preferCurrentBoolPointer(g.PreventBackupOfAppData, state.PreventBackupOfAppData)
	state.AllowUserToDelete = preferCurrentBoolPointer(g.AllowUserToDelete, state.AllowUserToDelete)
	state.RequireNetworkTethered = preferCurrentBoolPointer(g.RequireNetworkTethered, state.RequireNetworkTethered)
	state.HostExternally = preferCurrentBoolPointer(g.HostExternally, state.HostExternally)

	if g.Category != nil {
		state.CategoryID = preferCurrentStringPointer(stringFromIntPtr(g.Category.ID), state.CategoryID)
		state.CategoryName = helpers.StringPointerValueOrNull(g.Category.Name)
	} else {
		state.CategoryID = preferCurrentStringPointer(nil, state.CategoryID)
		state.CategoryName = types.StringNull()
	}

	if g.Site != nil {
		state.SiteID = preferCurrentStringPointer(stringFromIntPtr(g.Site.ID), state.SiteID)
		state.SiteName = helpers.StringPointerValueOrNull(g.Site.Name)
	} else {
		state.SiteID = preferCurrentStringPointer(nil, state.SiteID)
		state.SiteName = types.StringNull()
	}
}

func flattenMobileAppScope(ctx context.Context, s *proclassic.MobileDeviceApplicationScope, state *scope.MobileScopeModelNoIbeacons) {
	state.AllMobileDevices = preferCurrentBoolPointer(s.AllMobileDevices, state.AllMobileDevices)
	state.AllJssUsers = preferCurrentBoolPointer(s.AllJssUsers, state.AllJssUsers)

	state.MobileDeviceIDs = flattenMobileDeviceItemSet(ctx, s.MobileDevices)
	state.MobileDeviceGroupIDs = flattenIDNameSet(ctx, mobileDeviceGroupSlice(s.MobileDeviceGroups))
	state.BuildingIDs = flattenIDNameSet(ctx, buildingSlice(s.Buildings))
	state.DepartmentIDs = flattenIDNameSet(ctx, departmentSlice(s.Departments))
	state.UserIDs = flattenIDNameSet(ctx, jssUserSlice(s.JssUsers))
	state.UserGroupIDs = flattenIDNameSet(ctx, jssUserGroupSlice(s.JssUserGroups))

	if state.Limitations != nil && s.Limitations != nil {
		l := s.Limitations
		state.Limitations.NetworkSegmentIDs = flattenIDNameSet(ctx, limitationsSegmentSlice(l.NetworkSegments))
		state.Limitations.DirectoryServiceOrLocalUserNames = flattenNameSet(ctx, limitationsUserSlice(l.Users))
		state.Limitations.DirectoryServiceUserGroupNames = flattenNameSet(ctx, limitationsUserGroupSlice(l.UserGroups))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.MobileDeviceIDs = flattenExclMobileDeviceItemSet(ctx, e.MobileDevices)
		state.Exclusions.MobileDeviceGroupIDs = flattenIDNameSet(ctx, exclMobileDeviceGroupSlice(e.MobileDeviceGroups))
		state.Exclusions.BuildingIDs = flattenIDNameSet(ctx, exclBuildingSlice(e.Buildings))
		state.Exclusions.DepartmentIDs = flattenIDNameSet(ctx, exclDepartmentSlice(e.Departments))
		state.Exclusions.UserIDs = flattenIDNameSet(ctx, exclJssUserSlice(e.JssUsers))
		state.Exclusions.UserGroupIDs = flattenIDNameSet(ctx, exclJssUserGroupSlice(e.JssUserGroups))
		state.Exclusions.NetworkSegmentIDs = flattenExclNetworkSegmentSet(ctx, e.NetworkSegments)
		state.Exclusions.DirectoryServiceOrLocalUserNames = flattenExclUsersNameSet(ctx, e.Users)
		state.Exclusions.DirectoryServiceUserGroupNames = flattenNameSet(ctx, exclUserGroupSlice(e.UserGroups))
	}
}

func flattenMobileAppSelfService(ss *proclassic.MobileDeviceApplicationSelfService, state *MobileAppSelfServiceModel) {
	state.InstallButtonText = preferCurrentStringPointer(ss.SelfServiceInstallButtonText, state.InstallButtonText)
	state.AfterInstallButtonText = preferCurrentStringPointer(ss.SelfServiceAfterInstallButtonText, state.AfterInstallButtonText)
	state.SelfServiceDescription = preferCurrentStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.FeatureOnMainPage = preferCurrentBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)

	var apiEnabled *bool
	if ss.Notification != nil {
		apiEnabled = ss.Notification.Enabled
	}
	state.NotificationEnabled = preferCurrentBoolPointer(apiEnabled, state.NotificationEnabled)
	state.NotificationSubject = preferCurrentStringPointer(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = preferCurrentStringPointer(ss.NotificationMessage, state.NotificationMessage)

	if state.SelfServiceIcon != nil && ss.SelfServiceIcon != nil {
		state.SelfServiceIcon.ID = preferCurrentStringPointer(stringFromIntPtr(ss.SelfServiceIcon.ID), state.SelfServiceIcon.ID)
		state.SelfServiceIcon.URI = preferCurrentStringPointer(ss.SelfServiceIcon.URI, state.SelfServiceIcon.URI)
	}

	if state.SelfServiceCategories != nil && ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil {
		flattenMobileAppSelfServiceCategories(*ss.SelfServiceCategories.Category, state)
	}
}

// flattenMobileAppSelfServiceCategories refreshes the managed category set,
// matching server items to existing state items by ID so caller-authored values
// (display_in) stick across refreshes. Server items not in state are appended;
// the set is keyed by category ID.
func flattenMobileAppSelfServiceCategories(api []proclassic.MobileDeviceApplicationSelfServiceSelfServiceCategoriesCategoryItem, state *MobileAppSelfServiceModel) {
	byID := make(map[string]MobileAppSelfServiceCategoryModel, len(state.SelfServiceCategories))
	for _, c := range state.SelfServiceCategories {
		byID[c.ID.ValueString()] = c
	}

	out := make([]MobileAppSelfServiceCategoryModel, 0, len(api))
	for _, c := range api {
		idStr := ""
		if s := stringFromIntPtr(c.ID); s != nil {
			idStr = *s
		}
		current := byID[idStr]
		out = append(out, MobileAppSelfServiceCategoryModel{
			ID:        types.StringValue(idStr),
			Name:      preferCurrentStringPointer(c.Name, current.Name),
			DisplayIn: preferCurrentBoolPointer(c.DisplayIn, current.DisplayIn),
		})
	}
	state.SelfServiceCategories = out
}

func flattenMobileAppVpp(v *proclassic.MobileDeviceApplicationVpp, state *MobileAppVppModel) {
	state.AssignVppDeviceBasedLicenses = preferCurrentBoolPointer(v.AssignVppDeviceBasedLicenses, state.AssignVppDeviceBasedLicenses)
	state.VppAdminAccountID = preferCurrentStringPointer(stringFromIntPtr(v.VppAdminAccountID), state.VppAdminAccountID)
}

// ---- scope sub-slice accessors -------------------------------------------------

func mobileDeviceGroupSlice(g *proclassic.MobileDeviceApplicationScopeMobileDeviceGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.MobileDeviceGroup
}

func buildingSlice(b *proclassic.MobileDeviceApplicationScopeBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func departmentSlice(d *proclassic.MobileDeviceApplicationScopeDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func jssUserSlice(u *proclassic.MobileDeviceApplicationScopeJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func jssUserGroupSlice(u *proclassic.MobileDeviceApplicationScopeJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func limitationsSegmentSlice(s *proclassic.MobileDeviceApplicationScopeLimitationsNetworkSegments) *[]proclassic.IDName {
	if s == nil {
		return nil
	}
	return s.NetworkSegment
}

func limitationsUserSlice(u *proclassic.MobileDeviceApplicationScopeLimitationsUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func limitationsUserGroupSlice(u *proclassic.MobileDeviceApplicationScopeLimitationsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func exclMobileDeviceGroupSlice(g *proclassic.MobileDeviceApplicationScopeExclusionsMobileDeviceGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.MobileDeviceGroup
}

func exclBuildingSlice(b *proclassic.MobileDeviceApplicationScopeExclusionsBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func exclDepartmentSlice(d *proclassic.MobileDeviceApplicationScopeExclusionsDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func exclJssUserSlice(u *proclassic.MobileDeviceApplicationScopeExclusionsJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func exclJssUserGroupSlice(u *proclassic.MobileDeviceApplicationScopeExclusionsJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func exclUserGroupSlice(u *proclassic.MobileDeviceApplicationScopeExclusionsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

// ---- set flatteners ------------------------------------------------------------

func flattenMobileDeviceItemSet(ctx context.Context, m *proclassic.MobileDeviceApplicationScopeMobileDevices) types.Set {
	if m == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, m.MobileDevice, func(i proclassic.MobileDeviceApplicationScopeMobileDevicesMobileDeviceItem) *int { return i.ID })
	return out
}

func flattenExclMobileDeviceItemSet(ctx context.Context, m *proclassic.MobileDeviceApplicationScopeExclusionsMobileDevices) types.Set {
	if m == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, m.MobileDevice, func(i proclassic.MobileDeviceApplicationScopeExclusionsMobileDevicesMobileDeviceItem) *int {
		return i.ID
	})
	return out
}

func flattenExclNetworkSegmentSet(ctx context.Context, n *proclassic.MobileDeviceApplicationScopeExclusionsNetworkSegments) types.Set {
	if n == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, n.NetworkSegment, func(i proclassic.MobileDeviceApplicationScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int {
		return i.ID
	})
	return out
}

func flattenExclUsersNameSet(ctx context.Context, u *proclassic.MobileDeviceApplicationScopeExclusionsUsers) types.Set {
	if u == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenNameSlice(ctx, u.User, func(i proclassic.MobileDeviceApplicationScopeExclusionsUsersUserItem) *string { return i.Name })
	return out
}

func flattenIDNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenIDSlice(ctx, items, func(i proclassic.IDName) *int { return i.ID })
	return out
}

func flattenNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenNameSlice(ctx, items, func(i proclassic.IDName) *string { return i.Name })
	return out
}
