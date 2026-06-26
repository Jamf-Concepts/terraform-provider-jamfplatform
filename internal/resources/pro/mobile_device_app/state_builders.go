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
//
// includeUnmanaged inverts those section gates for the list resource's
// config-generation path (terraform query -generate-config-out): there is no
// plan to stay consistent with, so every wire-present optional section is
// allocated and hydrated, yielding a complete exported config rather than a
// general-only one. CRUD callers pass false. The mobile-app flatteners use the
// PreferCurrent* helpers (which adopt the wire value when the current state is
// null), so allocating an empty section is sufficient for it to fully hydrate.
func assignMobileAppResourceModel(ctx context.Context, state *MobileAppResourceModel, a *proclassic.MobileDeviceApplication, includeUnmanaged bool) diag.Diagnostics {
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

	if includeUnmanaged && state.Scope == nil && a.Scope != nil {
		state.Scope = &scope.MobileScopeModelNoIbeacons{}
	}
	if state.Scope != nil && a.Scope != nil {
		flattenMobileAppScope(ctx, a.Scope, state.Scope, includeUnmanaged)
	}
	if includeUnmanaged && state.SelfService == nil && a.SelfService != nil {
		state.SelfService = &MobileAppSelfServiceModel{}
	}
	if state.SelfService != nil && a.SelfService != nil {
		flattenMobileAppSelfService(a.SelfService, state.SelfService)
	}
	if includeUnmanaged && state.Vpp == nil && a.Vpp != nil {
		state.Vpp = &MobileAppVppModel{}
	}
	if state.Vpp != nil && a.Vpp != nil {
		flattenMobileAppVpp(a.Vpp, state.Vpp)
	}
	if includeUnmanaged && state.AppConfiguration == nil && a.AppConfiguration != nil {
		state.AppConfiguration = &MobileAppAppConfigurationModel{}
	}
	if state.AppConfiguration != nil && a.AppConfiguration != nil {
		// Preserve the configured value when the server differs only by newline
		// style / a stripped trailing newline (server strips it on round-trip);
		// otherwise reflect the server value as drift.
		state.AppConfiguration.Preferences = preservePreferences(a.AppConfiguration.Preferences, state.AppConfiguration.Preferences)
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
	// os_type echo is asymmetric (wire-probed): a POST never persists/echoes it,
	// and a non-internal app never carries it, but once set via a PUT to an
	// in-house app it is stored and echoed on every GET. Reflect the echo when
	// present (authoritative; surfaces external drift) and fall back to the
	// configured/prior value when absent so a Required attribute is never nulled
	// (which would trip "inconsistent result after apply"). Create issues a
	// follow-up PUT precisely so the server persists it and this echo is present.
	state.OsType = serverWhenPresentString(g.OsType, state.OsType)

	// Server-managed read-only field. (display_name and internal_app are not
	// modeled — see MobileAppGeneralModel.)
	state.Description = helpers.StringPointerValueOrNull(g.Description)

	// Optional+Computed echoes.
	state.IsFree = helpers.PreferCurrentBoolPointer(g.Free, state.IsFree)
	state.DeploymentType = helpers.PreferCurrentStringPointer(g.DeploymentType, state.DeploymentType)
	state.ExternalURL = helpers.PreferCurrentStringPointer(g.ExternalURL, state.ExternalURL)
	state.ItunesStoreURL = helpers.PreferCurrentStringPointer(g.ItunesStoreURL, state.ItunesStoreURL)
	state.ItunesCountryRegion = helpers.PreferCurrentStringPointer(g.ItunesCountryRegion, state.ItunesCountryRegion)
	state.ItunesSyncTime = preferCurrentInt64Pointer(g.ItunesSyncTime, state.ItunesSyncTime)
	state.MakeAvailableAfterInstall = helpers.PreferCurrentBoolPointer(g.MakeAvailableAfterInstall, state.MakeAvailableAfterInstall)
	state.KeepDescriptionAndIconUpToDate = helpers.PreferCurrentBoolPointer(g.KeepDescriptionAndIconUpToDate, state.KeepDescriptionAndIconUpToDate)
	state.KeepAppUpdatedOnDevices = helpers.PreferCurrentBoolPointer(g.KeepAppUpdatedOnDevices, state.KeepAppUpdatedOnDevices)
	state.DeployAsManagedApp = helpers.PreferCurrentBoolPointer(g.DeployAsManagedApp, state.DeployAsManagedApp)
	state.TakeOverManagement = helpers.PreferCurrentBoolPointer(g.TakeOverManagement, state.TakeOverManagement)
	state.DeployAutomatically = helpers.PreferCurrentBoolPointer(g.DeployAutomatically, state.DeployAutomatically)
	state.RemoveAppWhenMDMProfileIsRemoved = helpers.PreferCurrentBoolPointer(g.RemoveAppWhenMDMProfileIsRemoved, state.RemoveAppWhenMDMProfileIsRemoved)
	state.PreventBackupOfAppData = helpers.PreferCurrentBoolPointer(g.PreventBackupOfAppData, state.PreventBackupOfAppData)
	state.AllowUserToDelete = helpers.PreferCurrentBoolPointer(g.AllowUserToDelete, state.AllowUserToDelete)
	state.RequireNetworkTethered = helpers.PreferCurrentBoolPointer(g.RequireNetworkTethered, state.RequireNetworkTethered)
	state.HostExternally = helpers.PreferCurrentBoolPointer(g.HostExternally, state.HostExternally)

	if g.Category != nil {
		state.CategoryID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Category.ID), state.CategoryID)
		state.CategoryName = helpers.DerivedRefName(g.Category.ID, g.Category.Name)
	} else {
		state.CategoryID = helpers.PreferCurrentStringPointer(nil, state.CategoryID)
		state.CategoryName = types.StringNull()
	}

	if g.Site != nil {
		state.SiteID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(g.Site.ID), state.SiteID)
		state.SiteName = helpers.DerivedRefName(g.Site.ID, g.Site.Name)
	} else {
		state.SiteID = helpers.PreferCurrentStringPointer(nil, state.SiteID)
		state.SiteName = types.StringNull()
	}
}

// flattenMobileAppScope refreshes the scope sub-blocks the caller already
// manages. When includeUnmanaged is set (config generation) every wire-present
// sub-block is first allocated so the from-scratch read hydrates the full scope
// rather than leaving unmanaged targets/limitations/exclusions null.
func flattenMobileAppScope(ctx context.Context, s *proclassic.MobileDeviceApplicationScope, state *scope.MobileScopeModelNoIbeacons, includeUnmanaged bool) {
	if includeUnmanaged {
		if state.Targets == nil {
			state.Targets = &scope.MobileScopeTargetsModel{}
		}
		if state.Limitations == nil && s.Limitations != nil {
			state.Limitations = &scope.MobileScopeLimitationsModelNoIbeacons{}
		}
		if state.Exclusions == nil && s.Exclusions != nil {
			state.Exclusions = &scope.MobileScopeExclusionsModelNoIbeacons{}
		}
	}

	if state.Targets != nil {
		state.Targets.AllMobileDevices = helpers.PreferCurrentBoolPointer(s.AllMobileDevices, state.Targets.AllMobileDevices)
		state.Targets.AllJssUsers = helpers.PreferCurrentBoolPointer(s.AllJssUsers, state.Targets.AllJssUsers)

		state.Targets.MobileDeviceIDs = flattenMobileDeviceItemSet(ctx, s.MobileDevices)
		state.Targets.MobileDeviceGroupIDs = scope.FlattenIDNameSet(ctx, mobileDeviceGroupSlice(s.MobileDeviceGroups))
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
		state.Exclusions.MobileDeviceIDs = flattenExclMobileDeviceItemSet(ctx, e.MobileDevices)
		state.Exclusions.MobileDeviceGroupIDs = scope.FlattenIDNameSet(ctx, exclMobileDeviceGroupSlice(e.MobileDeviceGroups))
		state.Exclusions.BuildingIDs = scope.FlattenIDNameSet(ctx, exclBuildingSlice(e.Buildings))
		state.Exclusions.DepartmentIDs = scope.FlattenIDNameSet(ctx, exclDepartmentSlice(e.Departments))
		state.Exclusions.UserIDs = scope.FlattenIDNameSet(ctx, exclJssUserSlice(e.JssUsers))
		state.Exclusions.UserGroupIDs = scope.FlattenIDNameSet(ctx, exclJssUserGroupSlice(e.JssUserGroups))
		state.Exclusions.NetworkSegmentIDs = flattenExclNetworkSegmentSet(ctx, e.NetworkSegments)
		state.Exclusions.DirectoryServiceOrLocalUserNames = flattenExclUsersNameSet(ctx, e.Users)
		state.Exclusions.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, exclUserGroupSlice(e.UserGroups))
	}
}

func flattenMobileAppSelfService(ss *proclassic.MobileDeviceApplicationSelfService, state *MobileAppSelfServiceModel) {
	state.InstallButtonText = helpers.PreferCurrentStringPointer(ss.SelfServiceInstallButtonText, state.InstallButtonText)
	state.AfterInstallButtonText = helpers.PreferCurrentStringPointer(ss.SelfServiceAfterInstallButtonText, state.AfterInstallButtonText)
	state.SelfServiceDescription = helpers.PreferCurrentStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.FeatureOnMainPage = helpers.PreferCurrentBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)

	var apiEnabled *bool
	if ss.Notification != nil {
		apiEnabled = ss.Notification.Enabled
	}
	state.NotificationEnabled = helpers.PreferCurrentBoolPointer(apiEnabled, state.NotificationEnabled)
	state.NotificationSubject = helpers.PreferCurrentStringPointer(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = helpers.PreferCurrentStringPointer(ss.NotificationMessage, state.NotificationMessage)

	if state.SelfServiceIcon != nil && ss.SelfServiceIcon != nil {
		state.SelfServiceIcon.ID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(ss.SelfServiceIcon.ID), state.SelfServiceIcon.ID)
		state.SelfServiceIcon.URI = helpers.PreferCurrentStringPointer(ss.SelfServiceIcon.URI, state.SelfServiceIcon.URI)
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
		if s := helpers.StringFromIntPtr(c.ID); s != nil {
			idStr = *s
		}
		current := byID[idStr]
		out = append(out, MobileAppSelfServiceCategoryModel{
			ID:        types.StringValue(idStr),
			Name:      helpers.PreferCurrentStringPointer(c.Name, current.Name),
			DisplayIn: helpers.PreferCurrentBoolPointer(c.DisplayIn, current.DisplayIn),
		})
	}
	state.SelfServiceCategories = out
}

func flattenMobileAppVpp(v *proclassic.MobileDeviceApplicationVpp, state *MobileAppVppModel) {
	state.AssignVppDeviceBasedLicenses = helpers.PreferCurrentBoolPointer(v.AssignVppDeviceBasedLicenses, state.AssignVppDeviceBasedLicenses)
	state.VppAdminAccountID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(v.VppAdminAccountID), state.VppAdminAccountID)
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
