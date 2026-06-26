// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignPolicyResourceModel populates a resource model from the SDK Policy
// response. Server is authoritative for every Optional+Computed field — the
// helper reconciles against the current state so unmanaged fields stay null.
//
// includeUnmanaged inverts the section gates for the list resource's
// config-generation path (terraform query -generate-config-out): there is no
// plan to stay consistent with, so every wire-present optional section is
// allocated and hydrated from the server, yielding a complete exported config
// rather than a general-only one. CRUD callers pass false. Because the policy
// flatteners use the PreferCurrent* helpers (which adopt the wire value when
// the current state is null), allocating an empty section is sufficient for it
// to fully hydrate.
func assignPolicyResourceModel(ctx context.Context, state *PolicyResourceModel, p *proclassic.Policy, includeUnmanaged bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if p == nil {
		return diags
	}

	if p.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(p.ID)
	} else if p.General != nil && p.General.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(p.General.ID)
	}

	if state.General == nil {
		state.General = &PolicyGeneralModel{}
	}
	flattenPolicyGeneral(ctx, p.General, state.General, includeUnmanaged)

	// Optional sections are only refreshed when the caller (plan or current
	// state) already manages them. Server returns every section in GET with
	// default values — populating an unmanaged section would violate the
	// framework's "produced inconsistent result after apply" check, because
	// plan said null and we'd return a populated object. includeUnmanaged
	// allocates each wire-present section first so config generation hydrates
	// the whole policy.
	if includeUnmanaged && state.Scope == nil && p.Scope != nil {
		state.Scope = &scope.ComputerScopeModel{}
	}
	if state.Scope != nil && p.Scope != nil {
		diags.Append(flattenPolicyScope(ctx, p.Scope, state.Scope, includeUnmanaged)...)
	}
	if includeUnmanaged && state.SelfService == nil && p.SelfService != nil {
		state.SelfService = &PolicySelfServiceModel{}
	}
	if state.SelfService != nil && p.SelfService != nil {
		flattenPolicySelfService(p.SelfService, state.SelfService, includeUnmanaged)
	}
	if includeUnmanaged && state.Packages == nil && p.PackageConfiguration != nil {
		state.Packages = &PolicyPackagesModel{}
	}
	if state.Packages != nil && p.PackageConfiguration != nil {
		flattenPolicyPackageConfiguration(p.PackageConfiguration, state.Packages)
	}
	if includeUnmanaged && state.Scripts == nil && p.Scripts != nil {
		state.Scripts = &PolicyScriptsModel{}
	}
	if state.Scripts != nil && p.Scripts != nil {
		flattenPolicyScripts(p.Scripts, state.Scripts)
	}
	if includeUnmanaged && state.Printers == nil && p.Printers != nil {
		state.Printers = &PolicyPrintersModel{}
	}
	if state.Printers != nil && p.Printers != nil {
		flattenPolicyPrinters(p.Printers, state.Printers)
	}
	if includeUnmanaged && state.DockItems == nil && p.DockItems != nil {
		state.DockItems = &PolicyDockItemsModel{}
	}
	if state.DockItems != nil && p.DockItems != nil {
		flattenPolicyDockItems(p.DockItems, state.DockItems)
	}
	// The four flattened account-maintenance blocks all read from the single
	// wire <account_maintenance> object. Each is refreshed only when the
	// caller already manages it (per-section gate), mirroring the optional
	// section discipline above.
	if p.AccountMaintenance != nil {
		flattenPolicyAccountMaintenance(p.AccountMaintenance, state, includeUnmanaged)
	}
	if includeUnmanaged && state.RestartOptions == nil && p.Reboot != nil {
		state.RestartOptions = &PolicyRestartOptionsModel{}
	}
	if state.RestartOptions != nil && p.Reboot != nil {
		flattenPolicyReboot(p.Reboot, state.RestartOptions)
	}
	if includeUnmanaged && state.Maintenance == nil && p.Maintenance != nil {
		state.Maintenance = &PolicyMaintenanceModel{}
	}
	if state.Maintenance != nil && p.Maintenance != nil {
		flattenPolicyMaintenance(p.Maintenance, state.Maintenance)
	}
	if includeUnmanaged && state.FilesAndProcesses == nil && p.FilesProcesses != nil {
		state.FilesAndProcesses = &PolicyFilesAndProcessesModel{}
	}
	if state.FilesAndProcesses != nil && p.FilesProcesses != nil {
		flattenPolicyFilesProcesses(p.FilesProcesses, state.FilesAndProcesses)
	}
	if includeUnmanaged && state.UserInteraction == nil && p.UserInteraction != nil {
		state.UserInteraction = &PolicyUserInteractionModel{}
	}
	if state.UserInteraction != nil && p.UserInteraction != nil {
		flattenPolicyUserInteraction(p.UserInteraction, state.UserInteraction)
	}
	if includeUnmanaged && state.DiskEncryption == nil && p.DiskEncryption != nil {
		state.DiskEncryption = &PolicyDiskEncryptionModel{}
	}
	if state.DiskEncryption != nil && p.DiskEncryption != nil {
		flattenPolicyDiskEncryption(p.DiskEncryption, state.DiskEncryption)
	}

	return diags
}

func flattenPolicyGeneral(ctx context.Context, g *proclassic.PolicyGeneral, state *PolicyGeneralModel, includeUnmanaged bool) {
	if g == nil {
		return
	}

	if includeUnmanaged {
		if state.DateTimeLimitations == nil && g.DateTimeLimitations != nil {
			state.DateTimeLimitations = &PolicyGeneralDateTimeLimitationsModel{}
		}
		if state.NetworkLimitations == nil && g.NetworkLimitations != nil {
			state.NetworkLimitations = &PolicyGeneralNetworkLimitationsModel{}
		}
		if state.OverrideDefaultSettings == nil && g.OverrideDefaultSettings != nil {
			state.OverrideDefaultSettings = &PolicyGeneralOverrideDefaultsModel{}
		}
	}
	state.ID = helpers.StringValueFromIntPtr(g.ID)
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.Enabled = helpers.PreferCurrentBoolPointer(g.Enabled, state.Enabled)
	state.Trigger = helpers.PreferCurrentStringPointer(g.Trigger, state.Trigger)
	state.TriggerCheckin = helpers.PreferCurrentBoolPointer(g.TriggerCheckin, state.TriggerCheckin)
	state.TriggerEnrollmentComplete = helpers.PreferCurrentBoolPointer(g.TriggerEnrollmentComplete, state.TriggerEnrollmentComplete)
	state.TriggerLogin = helpers.PreferCurrentBoolPointer(g.TriggerLogin, state.TriggerLogin)
	state.TriggerNetworkStateChanged = helpers.PreferCurrentBoolPointer(g.TriggerNetworkStateChanged, state.TriggerNetworkStateChanged)
	state.TriggerStartup = helpers.PreferCurrentBoolPointer(g.TriggerStartup, state.TriggerStartup)
	state.TriggerOther = helpers.PreferCurrentStringPointer(g.TriggerOther, state.TriggerOther)
	state.Frequency = helpers.PreferCurrentStringPointer(g.Frequency, state.Frequency)
	state.RetryEvent = helpers.PreferCurrentStringPointer(g.RetryEvent, state.RetryEvent)
	if g.RetryAttempts != nil {
		state.RetryAttempts = preferCurrentInt(g.RetryAttempts, state.RetryAttempts)
	} else {
		state.RetryAttempts = types.Int64Null()
	}
	state.NotifyOnEachFailedRetry = helpers.PreferCurrentBoolPointer(g.NotifyOnEachFailedRetry, state.NotifyOnEachFailedRetry)
	state.LimitToJamfProAssignedUser = helpers.PreferCurrentBoolPointer(g.LocationUserOnly, state.LimitToJamfProAssignedUser)
	state.TargetDrive = helpers.PreferCurrentStringPointer(g.TargetDrive, state.TargetDrive)
	state.Offline = helpers.PreferCurrentBoolPointer(g.Offline, state.Offline)
	state.NetworkRequirements = helpers.PreferCurrentStringPointer(g.NetworkRequirements, state.NetworkRequirements)

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

	// Sub-sections gated on caller management. Same rule as the top-level
	// sections in assignPolicyResourceModel.
	if state.DateTimeLimitations != nil && g.DateTimeLimitations != nil {
		flattenPolicyDateTimeLimitations(ctx, g.DateTimeLimitations, state.DateTimeLimitations)
	}
	if state.NetworkLimitations != nil && g.NetworkLimitations != nil {
		flattenPolicyNetworkLimitations(ctx, g.NetworkLimitations, state.NetworkLimitations)
	}
	if state.OverrideDefaultSettings != nil && g.OverrideDefaultSettings != nil {
		state.OverrideDefaultSettings.TargetDrive = helpers.PreferCurrentStringPointer(g.OverrideDefaultSettings.TargetDrive, state.OverrideDefaultSettings.TargetDrive)
		state.OverrideDefaultSettings.DistributionPoint = helpers.PreferCurrentStringPointer(g.OverrideDefaultSettings.DistributionPoint, state.OverrideDefaultSettings.DistributionPoint)
		state.OverrideDefaultSettings.ForceAfpSmb = helpers.PreferCurrentBoolPointer(g.OverrideDefaultSettings.ForceAfpSmb, state.OverrideDefaultSettings.ForceAfpSmb)
		state.OverrideDefaultSettings.Sus = helpers.PreferCurrentStringPointer(g.OverrideDefaultSettings.Sus, state.OverrideDefaultSettings.Sus)
	}
}

func flattenPolicyDateTimeLimitations(ctx context.Context, dtl *proclassic.PolicyGeneralDateTimeLimitations, state *PolicyGeneralDateTimeLimitationsModel) {
	state.ActivationDate = helpers.PreferCurrentStringPointer(dtl.ActivationDate, state.ActivationDate)
	state.ExpirationDate = helpers.PreferCurrentStringPointer(dtl.ExpirationDate, state.ExpirationDate)
	state.NoExecuteStart = helpers.PreferCurrentStringPointer(dtl.NoExecuteStart, state.NoExecuteStart)
	state.NoExecuteEnd = helpers.PreferCurrentStringPointer(dtl.NoExecuteEnd, state.NoExecuteEnd)

	if dtl.NoExecuteOn != nil && dtl.NoExecuteOn.Day != nil && len(*dtl.NoExecuteOn.Day) > 0 {
		set, diagsLocal := types.SetValueFrom(ctx, types.StringType, *dtl.NoExecuteOn.Day)
		if !diagsLocal.HasError() {
			state.NoExecuteOn = set
		}
	} else {
		state.NoExecuteOn = types.SetNull(types.StringType)
	}
}

func flattenPolicyNetworkLimitations(ctx context.Context, nl *proclassic.PolicyGeneralNetworkLimitations, state *PolicyGeneralNetworkLimitationsModel) {
	state.MinimumNetworkConnection = helpers.PreferCurrentStringPointer(nl.MinimumNetworkConnection, state.MinimumNetworkConnection)
	state.AnyIPAddress = helpers.PreferCurrentBoolPointer(nl.AnyIPAddress, state.AnyIPAddress)
	if nl.NetworkSegments != nil {
		set, _ := scope.FlattenIDSlice(ctx, nl.NetworkSegments.NetworkSegment, func(i proclassic.IDName) *int { return i.ID })
		state.NetworkSegmentIDs = set
	} else {
		state.NetworkSegmentIDs = types.SetNull(types.StringType)
	}
}

func flattenPolicyScope(ctx context.Context, s *proclassic.PolicyScope, state *scope.ComputerScopeModel, includeUnmanaged bool) diag.Diagnostics {
	var diags diag.Diagnostics

	if includeUnmanaged {
		if state.Targets == nil {
			state.Targets = &scope.ComputerScopeTargetsModel{}
		}
		if state.Limitations == nil && s.Limitations != nil {
			state.Limitations = &scope.ComputerScopeLimitationsModel{}
		}
		if state.Exclusions == nil && s.Exclusions != nil {
			state.Exclusions = &scope.ComputerScopeExclusionsModel{}
		}
	}

	// Targets are gated on caller management, mirroring the limitations /
	// exclusions sub-blocks below: populating a targets block the user did not
	// declare would violate the framework's "produced inconsistent result after
	// apply" check (plan said null, we would return a populated object).
	if state.Targets != nil {
		state.Targets.AllComputers = helpers.PreferCurrentBoolPointer(s.AllComputers, state.Targets.AllComputers)
		state.Targets.AllJssUsers = helpers.PreferCurrentBoolPointer(s.AllJssUsers, state.Targets.AllJssUsers)

		state.Targets.ComputerIDs = flattenComputerItemSet(ctx, s.Computers)
		state.Targets.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromGroups(s.ComputerGroups))
		state.Targets.BuildingIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromBuildings(s.Buildings))
		state.Targets.DepartmentIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromDepartments(s.Departments))
		state.Targets.UserIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromJssUsers(s.JssUsers))
		state.Targets.UserGroupIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromJssUserGroups(s.JssUserGroups))
	}

	if state.Limitations != nil && s.Limitations != nil {
		l := s.Limitations
		state.Limitations.NetworkSegmentIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromLimitationsSegments(l.NetworkSegments))
		state.Limitations.IbeaconIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromLimitationsIbeacons(l.Ibeacons))
		state.Limitations.DirectoryServiceOrLocalUserNames = scope.FlattenNameSet(ctx, idNameSliceFromLimitationsUsers(l.Users))
		state.Limitations.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, idNameSliceFromLimitationsUserGroups(l.UserGroups))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.ComputerIDs = flattenExclComputerItemSet(ctx, e.Computers)
		state.Exclusions.ComputerGroupIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromExclComputerGroups(e.ComputerGroups))
		state.Exclusions.BuildingIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromExclBuildings(e.Buildings))
		state.Exclusions.DepartmentIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromExclDepartments(e.Departments))
		state.Exclusions.UserIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromExclJssUsers(e.JssUsers))
		state.Exclusions.UserGroupIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromExclJssUserGroups(e.JssUserGroups))
		state.Exclusions.NetworkSegmentIDs = flattenExclNetworkSegmentSet(ctx, e.NetworkSegments)
		state.Exclusions.IbeaconIDs = scope.FlattenIDNameSet(ctx, idNameSliceFromExclIbeacons(e.Ibeacons))
		state.Exclusions.DirectoryServiceOrLocalUserNames = flattenExclUsersNameSet(ctx, e.Users)
		state.Exclusions.DirectoryServiceUserGroupNames = scope.FlattenNameSet(ctx, idNameSliceFromExclUserGroups(e.UserGroups))
	}

	return diags
}

// ---- scope sub-slice accessors -------------------------------------------------

func idNameSliceFromGroups(g *proclassic.PolicyScopeComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func idNameSliceFromBuildings(b *proclassic.PolicyScopeBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func idNameSliceFromDepartments(d *proclassic.PolicyScopeDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func idNameSliceFromJssUsers(u *proclassic.PolicyScopeJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func idNameSliceFromJssUserGroups(u *proclassic.PolicyScopeJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func idNameSliceFromLimitationsSegments(s *proclassic.PolicyScopeLimitationsNetworkSegments) *[]proclassic.IDName {
	if s == nil {
		return nil
	}
	return s.NetworkSegment
}

func idNameSliceFromLimitationsIbeacons(i *proclassic.PolicyScopeLimitationsIbeacons) *[]proclassic.IDName {
	if i == nil {
		return nil
	}
	return i.Ibeacon
}

func idNameSliceFromLimitationsUsers(u *proclassic.PolicyScopeLimitationsUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func idNameSliceFromLimitationsUserGroups(u *proclassic.PolicyScopeLimitationsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func idNameSliceFromExclComputerGroups(g *proclassic.PolicyScopeExclusionsComputerGroups) *[]proclassic.IDName {
	if g == nil {
		return nil
	}
	return g.ComputerGroup
}

func idNameSliceFromExclBuildings(b *proclassic.PolicyScopeExclusionsBuildings) *[]proclassic.IDName {
	if b == nil {
		return nil
	}
	return b.Building
}

func idNameSliceFromExclDepartments(d *proclassic.PolicyScopeExclusionsDepartments) *[]proclassic.IDName {
	if d == nil {
		return nil
	}
	return d.Department
}

func idNameSliceFromExclJssUsers(u *proclassic.PolicyScopeExclusionsJssUsers) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.User
}

func idNameSliceFromExclJssUserGroups(u *proclassic.PolicyScopeExclusionsJssUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

func idNameSliceFromExclIbeacons(i *proclassic.PolicyScopeExclusionsIbeacons) *[]proclassic.IDName {
	if i == nil {
		return nil
	}
	return i.Ibeacon
}

func idNameSliceFromExclUserGroups(u *proclassic.PolicyScopeExclusionsUserGroups) *[]proclassic.IDName {
	if u == nil {
		return nil
	}
	return u.UserGroup
}

// ---- set flatteners ------------------------------------------------------------

func flattenComputerItemSet(ctx context.Context, c *proclassic.PolicyScopeComputers) types.Set {
	if c == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, c.Computer, func(i proclassic.PolicyScopeComputersComputerItem) *int { return i.ID })
	return out
}

func flattenExclComputerItemSet(ctx context.Context, c *proclassic.PolicyScopeExclusionsComputers) types.Set {
	if c == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, c.Computer, func(i proclassic.PolicyScopeExclusionsComputersComputerItem) *int { return i.ID })
	return out
}

func flattenExclNetworkSegmentSet(ctx context.Context, n *proclassic.PolicyScopeExclusionsNetworkSegments) types.Set {
	if n == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenIDSlice(ctx, n.NetworkSegment, func(i proclassic.PolicyScopeExclusionsNetworkSegmentsNetworkSegmentItem) *int { return i.ID })
	return out
}

func flattenExclUsersNameSet(ctx context.Context, u *proclassic.PolicyScopeExclusionsUsers) types.Set {
	if u == nil {
		return types.SetNull(types.StringType)
	}
	out, _ := scope.FlattenNameSlice(ctx, u.User, func(i proclassic.PolicyScopeExclusionsUsersUserItem) *string { return i.Name })
	return out
}

// ---- self_service / packages / scripts / printers / dock_items / etc. ----------

func flattenPolicySelfService(ss *proclassic.PolicySelfService, state *PolicySelfServiceModel, includeUnmanaged bool) {
	if includeUnmanaged {
		if state.SelfServiceIcon == nil && ss.SelfServiceIcon != nil {
			state.SelfServiceIcon = &PolicySelfServiceIconModel{}
		}
		if state.Categories == nil {
			state.Categories = []PolicySelfServiceCategoryItemModel{}
		}
	}
	state.UseForSelfService = helpers.PreferCurrentBoolPointer(ss.UseForSelfService, state.UseForSelfService)
	state.SelfServiceDisplayName = helpers.PreferCurrentStringPointer(ss.SelfServiceDisplayName, state.SelfServiceDisplayName)
	state.InstallButtonText = helpers.PreferCurrentStringPointer(ss.InstallButtonText, state.InstallButtonText)
	state.ReinstallButtonText = helpers.PreferCurrentStringPointer(ss.ReinstallButtonText, state.ReinstallButtonText)
	state.SelfServiceDescription = helpers.PreferCurrentStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.EnsureUsersViewDescription = helpers.PreferCurrentBoolPointer(ss.ForceUsersToViewDescription, state.EnsureUsersViewDescription)
	state.IncludeInFeaturedCategory = helpers.PreferCurrentBoolPointer(ss.FeatureOnMainPage, state.IncludeInFeaturedCategory)
	state.DisplayNotifications = flattenNotificationEnabled(ss.Notification, state.DisplayNotifications)
	state.NotificationLocation = helpers.PreferCurrentStringPointer(ss.NotificationType, state.NotificationLocation)
	state.NotificationSubject = helpers.PreferCurrentStringPointer(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = helpers.PreferCurrentStringPointer(ss.NotificationMessage, state.NotificationMessage)

	if state.SelfServiceIcon != nil && ss.SelfServiceIcon != nil {
		state.SelfServiceIcon.ID = helpers.PreferCurrentStringPointer(helpers.StringFromIntPtr(ss.SelfServiceIcon.ID), state.SelfServiceIcon.ID)
		state.SelfServiceIcon.URI = helpers.PreferCurrentStringPointer(ss.SelfServiceIcon.URI, state.SelfServiceIcon.URI)
		state.SelfServiceIcon.Filename = helpers.PreferCurrentStringPointer(ss.SelfServiceIcon.Filename, state.SelfServiceIcon.Filename)
	}

	// Self Service categories are an unordered Set; rebuild the slice
	// directly from the wire. The classic /policies endpoint always echoes
	// id/name/display_in/feature_in for each surviving category, so the
	// Optional+Computed flags never come back null (no preferCurrent
	// needed). Refresh only when the caller manages the block.
	if state.Categories != nil {
		if ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil {
			items := *ss.SelfServiceCategories.Category
			out := make([]PolicySelfServiceCategoryItemModel, 0, len(items))
			for _, c := range items {
				out = append(out, PolicySelfServiceCategoryItemModel{
					ID:        helpers.StringValueFromIntPtr(c.ID),
					Name:      helpers.StringPointerValueOrNull(c.Name),
					DisplayIn: helpers.BoolPointerValueOrNull(c.DisplayIn),
					FeatureIn: helpers.BoolPointerValueOrNull(c.FeatureIn),
				})
			}
			state.Categories = out
		} else {
			state.Categories = []PolicySelfServiceCategoryItemModel{}
		}
	}
}

func flattenPolicyPackageConfiguration(pc *proclassic.PolicyPackageConfiguration, state *PolicyPackagesModel) {
	state.DistributionPoint = helpers.PreferCurrentStringPointer(pc.DistributionPoint, state.DistributionPoint)

	if pc.Packages == nil || pc.Packages.Package == nil {
		state.Packages = nil
		return
	}
	items := *pc.Packages.Package
	out := make([]PolicyPackageItemModel, 0, len(items))
	for _, p := range items {
		out = append(out, PolicyPackageItemModel{
			ID:            helpers.StringValueFromIntPtr(p.ID),
			Name:          helpers.StringPointerValueOrNull(p.Name),
			Action:        helpers.StringPointerValueOrNull(p.Action),
			Fut:           helpers.BoolPointerValueOrNull(p.Fut),
			Feu:           helpers.BoolPointerValueOrNull(p.Feu),
			UpdateAutorun: helpers.BoolPointerValueOrNull(p.UpdateAutorun),
		})
	}
	state.Packages = out
}

func flattenPolicyScripts(sc *proclassic.PolicyScripts, state *PolicyScriptsModel) {
	if sc.Script == nil {
		state.Scripts = nil
		return
	}
	items := *sc.Script
	out := make([]PolicyScriptItemModel, 0, len(items))
	for _, s := range items {
		out = append(out, PolicyScriptItemModel{
			ID:          helpers.StringValueFromIntPtr(s.ID),
			Name:        helpers.StringPointerValueOrNull(s.Name),
			Priority:    helpers.StringPointerValueOrNull(s.Priority),
			Parameter4:  helpers.StringPointerValueOrNull(s.Parameter4),
			Parameter5:  helpers.StringPointerValueOrNull(s.Parameter5),
			Parameter6:  helpers.StringPointerValueOrNull(s.Parameter6),
			Parameter7:  helpers.StringPointerValueOrNull(s.Parameter7),
			Parameter8:  helpers.StringPointerValueOrNull(s.Parameter8),
			Parameter9:  helpers.StringPointerValueOrNull(s.Parameter9),
			Parameter10: helpers.StringPointerValueOrNull(s.Parameter10),
			Parameter11: helpers.StringPointerValueOrNull(s.Parameter11),
		})
	}
	state.Scripts = out
}

func flattenPolicyPrinters(pr *proclassic.PolicyPrinters, state *PolicyPrintersModel) {
	state.LeaveExistingDefault = helpers.PreferCurrentBoolPointer(pr.LeaveExistingDefault, state.LeaveExistingDefault)

	if pr.Printer == nil {
		state.Printers = nil
		return
	}
	items := *pr.Printer
	out := make([]PolicyPrinterItemModel, 0, len(items))
	for _, p := range items {
		out = append(out, PolicyPrinterItemModel{
			ID:          helpers.StringValueFromIntPtr(p.ID),
			Name:        helpers.StringPointerValueOrNull(p.Name),
			Action:      printerActionFromWire(p.Action),
			MakeDefault: helpers.BoolPointerValueOrNull(p.MakeDefault),
		})
	}
	state.Printers = out
}

// printerActionFromWire is the inbound half of the UI/wire translation for
// `printers[].action`. Wire-visible `install` / `uninstall` become the
// UI-canonical `Map` / `Unmap` in Terraform state. Unrecognized values pass
// through unchanged so any future wire-side enum additions surface as drift
// rather than silently mapping to the wrong UI label.
func printerActionFromWire(action *string) types.String {
	if action == nil {
		return types.StringNull()
	}
	switch *action {
	case "install":
		return types.StringValue("Map")
	case "uninstall":
		return types.StringValue("Unmap")
	default:
		return types.StringValue(*action)
	}
}

func flattenPolicyDockItems(d *proclassic.PolicyDockItems, state *PolicyDockItemsModel) {
	if d.DockItem == nil {
		state.DockItems = nil
		return
	}
	items := *d.DockItem
	out := make([]PolicyDockItemModel, 0, len(items))
	for _, di := range items {
		out = append(out, PolicyDockItemModel{
			ID:     helpers.StringValueFromIntPtr(di.ID),
			Name:   helpers.StringPointerValueOrNull(di.Name),
			Action: helpers.StringPointerValueOrNull(di.Action),
		})
	}
	state.DockItems = out
}

// flattenPolicyAccountMaintenance splits the single wire <account_maintenance>
// object across the four flattened top-level state blocks. Each block is
// refreshed only when the caller already manages it (state.LocalAccounts /
// DirectoryBindings non-nil, ManagementAccount / EfiPassword non-nil) —
// populating an unmanaged section would violate the framework's "produced
// inconsistent result after apply" check.
func flattenPolicyAccountMaintenance(am *proclassic.PolicyAccountMaintenance, state *PolicyResourceModel, includeUnmanaged bool) {
	if includeUnmanaged {
		if state.LocalAccounts == nil && am.Accounts != nil && am.Accounts.Account != nil {
			state.LocalAccounts = []PolicyAccountItemModel{}
		}
		if state.DirectoryBindings == nil && am.DirectoryBindings != nil && am.DirectoryBindings.Binding != nil {
			state.DirectoryBindings = []PolicyDirectoryBindingItemModel{}
		}
		if state.ManagementAccount == nil && am.ManagementAccount != nil {
			state.ManagementAccount = &PolicyManagementAccountModel{}
		}
		if state.EfiPassword == nil && am.OpenFirmwareEfiPassword != nil {
			state.EfiPassword = &PolicyOpenFirmwareEfiPasswordModel{}
		}
	}
	if state.LocalAccounts != nil && am.Accounts != nil && am.Accounts.Account != nil {
		// Reorder wire accounts to match the plan's declared username
		// order. The Jamf classic /policies endpoint does not preserve
		// account order on round-trip — entries can come back in
		// arbitrary order. Without this reordering, the framework's
		// post-apply consistency check trips on Sensitive attribute drift
		// (the new state's accounts[i].password is null at a different
		// index than the plan's accounts[i].password).
		//
		// `state` here is actually the plan model (CRUD passes &plan to
		// assignPolicyResourceModel), so state.LocalAccounts holds the plan
		// order at this point. We build a wire-keyed-by-username map and
		// emit in plan order. Any wire-only entries (server echoed a
		// username not in plan) append at the end so they remain visible.
		//
		// Username is the natural key — Probe 9 confirmed the classic API
		// accepts duplicate usernames across Create/Reset/Delete/
		// DisableFileVault actions, but the multi-action acceptance
		// coverage uses distinct usernames per entry so the
		// username-keyed lookup is unambiguous here.
		wireItems := *am.Accounts.Account
		wireByUsername := make(map[string]proclassic.PolicyAccountMaintenanceAccountsAccountItem, len(wireItems))
		wireUsernameOrder := make([]string, 0, len(wireItems))
		for _, a := range wireItems {
			if a.Username == nil {
				continue
			}
			wireByUsername[*a.Username] = a
			wireUsernameOrder = append(wireUsernameOrder, *a.Username)
		}
		woByUsername := make(map[string]types.Int64, len(state.LocalAccounts))
		for _, prev := range state.LocalAccounts {
			if !prev.Username.IsNull() && !prev.Username.IsUnknown() {
				woByUsername[prev.Username.ValueString()] = prev.PasswordWoVersion
			}
		}
		emit := func(a proclassic.PolicyAccountMaintenanceAccountsAccountItem) PolicyAccountItemModel {
			currentWo := types.Int64Null()
			if a.Username != nil {
				if w, ok := woByUsername[*a.Username]; ok {
					currentWo = w
				}
			}
			return PolicyAccountItemModel{
				Action:                         helpers.StringPointerValueOrNull(a.Action),
				Username:                       helpers.StringPointerValueOrNull(a.Username),
				Realname:                       helpers.StringPointerValueOrNull(a.Realname),
				Password:                       types.StringNull(), // WriteOnly — framework strips from state
				PasswordWoVersion:              currentWo,          // round-trip from prior state
				PermanentlyDeleteHomeDirectory: invertBoolPointerValueOrNull(a.ArchiveHomeDirectory),
				ArchiveHomeDirectoryTo:         helpers.StringPointerValueOrNull(a.ArchiveHomeDirectoryTo),
				Home:                           helpers.StringPointerValueOrNull(a.Home),
				Hint:                           helpers.StringPointerValueOrNull(a.Hint),
				Picture:                        helpers.StringPointerValueOrNull(a.Picture),
				Admin:                          helpers.BoolPointerValueOrNull(a.Admin),
				FilevaultEnabled:               helpers.BoolPointerValueOrNull(a.FilevaultEnabled),
				SecureTokenAllowed:             helpers.BoolPointerValueOrNull(a.SecureTokenAllowed),
			}
		}
		consumed := make(map[string]bool, len(wireItems))
		out := make([]PolicyAccountItemModel, 0, len(wireItems))
		for _, prev := range state.LocalAccounts {
			if prev.Username.IsNull() || prev.Username.IsUnknown() {
				continue
			}
			u := prev.Username.ValueString()
			if a, ok := wireByUsername[u]; ok && !consumed[u] {
				out = append(out, emit(a))
				consumed[u] = true
			}
		}
		// Append wire-only entries (not in plan) at the end in wire order.
		for _, u := range wireUsernameOrder {
			if consumed[u] {
				continue
			}
			out = append(out, emit(wireByUsername[u]))
			consumed[u] = true
		}
		state.LocalAccounts = out
	} else {
		state.LocalAccounts = nil
	}

	if state.DirectoryBindings != nil && am.DirectoryBindings != nil && am.DirectoryBindings.Binding != nil {
		items := *am.DirectoryBindings.Binding
		out := make([]PolicyDirectoryBindingItemModel, 0, len(items))
		for _, b := range items {
			out = append(out, PolicyDirectoryBindingItemModel{
				ID:   helpers.StringValueFromIntPtr(b.ID),
				Name: helpers.StringPointerValueOrNull(b.Name),
			})
		}
		state.DirectoryBindings = out
	} else {
		state.DirectoryBindings = nil
	}

	// management_account + open_firmware_efi_password are Optional sibling
	// blocks. The classic API returns default-shaped objects for both even
	// when the caller did not set them, so unconditionally populating state
	// would violate the framework's "produced inconsistent result after
	// apply" check: plan said null, we would return a populated object.
	// Mirror the assignPolicyResourceModel section-level gate — only refresh
	// when the caller already manages the sub-block.
	if state.ManagementAccount != nil && am.ManagementAccount != nil {
		state.ManagementAccount.Action = helpers.PreferCurrentStringPointer(am.ManagementAccount.Action, state.ManagementAccount.Action)
		// managed_password is WriteOnly — framework strips from state.
		// managed_password_wo_version round-trips as a regular Optional
		// Int64; assignPolicyResourceModel does not overwrite the prior
		// state value here (the API never echoes it).
		state.ManagementAccount.ManagedPasswordLength = preferCurrentInt(am.ManagementAccount.ManagedPasswordLength, state.ManagementAccount.ManagedPasswordLength)
	}

	if state.EfiPassword != nil && am.OpenFirmwareEfiPassword != nil {
		state.EfiPassword.OfMode = helpers.PreferCurrentStringPointer(am.OpenFirmwareEfiPassword.OfMode, state.EfiPassword.OfMode)
		// of_password is WriteOnly — framework strips from state.
		// of_password_wo_version round-trips as a regular Optional Int64;
		// the API never echoes it, so the prior state value passes through
		// unchanged.
	}
}

func flattenPolicyReboot(r *proclassic.PolicyReboot, state *PolicyRestartOptionsModel) {
	state.Message = helpers.PreferCurrentStringPointer(r.Message, state.Message)
	state.StartupDisk = helpers.PreferCurrentStringPointer(r.StartupDisk, state.StartupDisk)
	state.SpecifyStartup = helpers.PreferCurrentStringPointer(r.SpecifyStartup, state.SpecifyStartup)
	state.NoUserLoggedIn = helpers.PreferCurrentStringPointer(r.NoUserLoggedIn, state.NoUserLoggedIn)
	state.UserLoggedIn = helpers.PreferCurrentStringPointer(r.UserLoggedIn, state.UserLoggedIn)
	state.DelayMinutes = preferCurrentInt(r.MinutesUntilReboot, state.DelayMinutes)
	state.StartRebootTimerImmediately = helpers.PreferCurrentBoolPointer(r.StartRebootTimerImmediately, state.StartRebootTimerImmediately)
	state.FileVault2Reboot = helpers.PreferCurrentBoolPointer(r.FileVault2Reboot, state.FileVault2Reboot)
}

func flattenPolicyMaintenance(m *proclassic.PolicyMaintenance, state *PolicyMaintenanceModel) {
	state.UpdateInventory = helpers.PreferCurrentBoolPointer(m.Recon, state.UpdateInventory)
	state.ResetComputerNames = helpers.PreferCurrentBoolPointer(m.ResetName, state.ResetComputerNames)
	state.InstallCachedPackages = helpers.PreferCurrentBoolPointer(m.InstallAllCachedPackages, state.InstallCachedPackages)
	state.FixDiskPermissions = helpers.PreferCurrentBoolPointer(m.Permissions, state.FixDiskPermissions)
	state.FixByhostFiles = helpers.PreferCurrentBoolPointer(m.Byhost, state.FixByhostFiles)
	state.FlushSystemCaches = helpers.PreferCurrentBoolPointer(m.SystemCache, state.FlushSystemCaches)
	state.FlushUserCaches = helpers.PreferCurrentBoolPointer(m.UserCache, state.FlushUserCaches)
	state.VerifyStartupDisk = helpers.PreferCurrentBoolPointer(m.Verify, state.VerifyStartupDisk)
}

func flattenPolicyFilesProcesses(fp *proclassic.PolicyFilesProcesses, state *PolicyFilesAndProcessesModel) {
	state.SearchByPath = helpers.PreferCurrentStringPointer(fp.SearchByPath, state.SearchByPath)
	state.DeleteFileIfFound = helpers.PreferCurrentBoolPointer(fp.DeleteFile, state.DeleteFileIfFound)
	state.SearchByFilename = helpers.PreferCurrentStringPointer(fp.LocateFile, state.SearchByFilename)
	state.UpdateLocateDatabase = helpers.PreferCurrentBoolPointer(fp.UpdateLocateDatabase, state.UpdateLocateDatabase)
	state.SearchBySpotlight = helpers.PreferCurrentStringPointer(fp.SpotlightSearch, state.SearchBySpotlight)
	state.SearchForProcess = helpers.PreferCurrentStringPointer(fp.SearchForProcess, state.SearchForProcess)
	state.KillProcessIfFound = helpers.PreferCurrentBoolPointer(fp.KillProcess, state.KillProcessIfFound)
	state.ExecuteCommand = helpers.PreferCurrentStringPointer(fp.RunCommand, state.ExecuteCommand)
}

// flattenPolicyUserInteraction collapses the wire's three-field deferral
// representation back into the synthetic `deferral_type` enum the schema
// exposes. The trio is treated as a unit — the wire is always authoritative
// here (server enforces the cross-field invariants on every PUT, so the
// stored values are always self-consistent).
func flattenPolicyUserInteraction(u *proclassic.PolicyUserInteraction, state *PolicyUserInteractionModel) {
	state.StartMessage = helpers.PreferCurrentStringPointer(u.MessageStart, state.StartMessage)
	state.CompleteMessage = helpers.PreferCurrentStringPointer(u.MessageFinish, state.CompleteMessage)

	defer_ := false
	if u.AllowUsersToDefer != nil {
		defer_ = *u.AllowUsersToDefer
	}
	until := ""
	if u.AllowDeferralUntilUtc != nil {
		until = *u.AllowDeferralUntilUtc
	}
	mins := 0
	if u.AllowDeferralMinutes != nil {
		mins = *u.AllowDeferralMinutes
	}

	switch {
	case defer_ && until != "":
		state.DeferralType = types.StringValue("date")
		state.DeferralUntilUtc = types.StringValue(until)
		state.DeferralDays = types.Int64Null()
	case defer_ && mins > 0:
		state.DeferralType = types.StringValue("duration")
		state.DeferralUntilUtc = types.StringNull()
		state.DeferralDays = types.Int64Value(int64(mins / minutesPerDay))
	default:
		// !defer OR (defer && until=="" && mins==0). The latter is a wire-
		// inconsistent shape the server should never persist; normalise to
		// "none" so state still satisfies the schema's enum constraint.
		state.DeferralType = types.StringValue("none")
		state.DeferralUntilUtc = types.StringNull()
		state.DeferralDays = types.Int64Null()
	}
}

func flattenPolicyDiskEncryption(d *proclassic.PolicyDiskEncryption, state *PolicyDiskEncryptionModel) {
	state.Action = helpers.PreferCurrentStringPointer(d.Action, state.Action)
	state.DiskEncryptionConfigurationID = preferCurrentInt(d.DiskEncryptionConfigurationID, state.DiskEncryptionConfigurationID)
	state.AuthRestart = helpers.PreferCurrentBoolPointer(d.AuthRestart, state.AuthRestart)
	state.RemediateKeyType = helpers.PreferCurrentStringPointer(d.RemediateKeyType, state.RemediateKeyType)
	state.RemediateDiskEncryptionConfigurationID = preferCurrentInt(d.RemediateDiskEncryptionConfigurationID, state.RemediateDiskEncryptionConfigurationID)
}

// ---- small helpers -------------------------------------------------------------
