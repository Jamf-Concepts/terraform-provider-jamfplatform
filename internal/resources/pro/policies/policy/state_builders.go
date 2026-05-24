// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// assignPolicyResourceModel populates a resource model from the SDK Policy
// response. Server is authoritative for every Optional+Computed field — the
// helper reconciles against the current state so unmanaged fields stay null.
func assignPolicyResourceModel(ctx context.Context, state *PolicyResourceModel, p *proclassic.Policy) diag.Diagnostics {
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
	flattenPolicyGeneral(ctx, p.General, state.General)

	// Optional sections are only refreshed when the caller (plan or current
	// state) already manages them. Server returns every section in GET with
	// default values — populating an unmanaged section would violate the
	// framework's "produced inconsistent result after apply" check, because
	// plan said null and we'd return a populated object.
	if state.Scope != nil && p.Scope != nil {
		diags.Append(flattenPolicyScope(ctx, p.Scope, state.Scope)...)
	}
	if state.SelfService != nil && p.SelfService != nil {
		flattenPolicySelfService(p.SelfService, state.SelfService)
	}
	if state.PackageConfiguration != nil && p.PackageConfiguration != nil {
		flattenPolicyPackageConfiguration(p.PackageConfiguration, state.PackageConfiguration)
	}
	if state.Scripts != nil && p.Scripts != nil {
		flattenPolicyScripts(p.Scripts, state.Scripts)
	}
	if state.Printers != nil && p.Printers != nil {
		flattenPolicyPrinters(p.Printers, state.Printers)
	}
	if state.DockItems != nil && p.DockItems != nil {
		flattenPolicyDockItems(p.DockItems, state.DockItems)
	}
	if state.AccountMaintenance != nil && p.AccountMaintenance != nil {
		flattenPolicyAccountMaintenance(p.AccountMaintenance, state.AccountMaintenance)
	}
	if state.Reboot != nil && p.Reboot != nil {
		flattenPolicyReboot(p.Reboot, state.Reboot)
	}
	if state.Maintenance != nil && p.Maintenance != nil {
		flattenPolicyMaintenance(p.Maintenance, state.Maintenance)
	}
	if state.FilesProcesses != nil && p.FilesProcesses != nil {
		flattenPolicyFilesProcesses(p.FilesProcesses, state.FilesProcesses)
	}
	if state.UserInteraction != nil && p.UserInteraction != nil {
		flattenPolicyUserInteraction(p.UserInteraction, state.UserInteraction)
	}
	if state.DiskEncryption != nil && p.DiskEncryption != nil {
		flattenPolicyDiskEncryption(p.DiskEncryption, state.DiskEncryption)
	}

	return diags
}

func flattenPolicyGeneral(ctx context.Context, g *proclassic.PolicyGeneral, state *PolicyGeneralModel) {
	if g == nil {
		return
	}
	state.ID = helpers.StringValueFromIntPtr(g.ID)
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.Enabled = preferCurrentBoolPointer(g.Enabled, state.Enabled)
	state.Trigger = preferCurrentStringPointer(g.Trigger, state.Trigger)
	state.TriggerCheckin = preferCurrentBoolPointer(g.TriggerCheckin, state.TriggerCheckin)
	state.TriggerEnrollmentComplete = preferCurrentBoolPointer(g.TriggerEnrollmentComplete, state.TriggerEnrollmentComplete)
	state.TriggerLogin = preferCurrentBoolPointer(g.TriggerLogin, state.TriggerLogin)
	state.TriggerLogout = preferCurrentBoolPointer(g.TriggerLogout, state.TriggerLogout)
	state.TriggerNetworkStateChanged = preferCurrentBoolPointer(g.TriggerNetworkStateChanged, state.TriggerNetworkStateChanged)
	state.TriggerStartup = preferCurrentBoolPointer(g.TriggerStartup, state.TriggerStartup)
	state.TriggerOther = preferCurrentStringPointer(g.TriggerOther, state.TriggerOther)
	state.Frequency = preferCurrentStringPointer(g.Frequency, state.Frequency)
	state.RetryEvent = preferCurrentStringPointer(g.RetryEvent, state.RetryEvent)
	if g.RetryAttempts != nil {
		state.RetryAttempts = preferCurrentInt(g.RetryAttempts, state.RetryAttempts)
	} else {
		state.RetryAttempts = types.Int64Null()
	}
	state.NotifyOnEachFailedRetry = preferCurrentBoolPointer(g.NotifyOnEachFailedRetry, state.NotifyOnEachFailedRetry)
	state.LocationUserOnly = preferCurrentBoolPointer(g.LocationUserOnly, state.LocationUserOnly)
	state.TargetDrive = preferCurrentStringPointer(g.TargetDrive, state.TargetDrive)
	state.Offline = preferCurrentBoolPointer(g.Offline, state.Offline)
	state.NetworkRequirements = preferCurrentStringPointer(g.NetworkRequirements, state.NetworkRequirements)

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

	// Sub-sections gated on caller management. Same rule as the top-level
	// sections in assignPolicyResourceModel.
	if state.DateTimeLimitations != nil && g.DateTimeLimitations != nil {
		flattenPolicyDateTimeLimitations(ctx, g.DateTimeLimitations, state.DateTimeLimitations)
	}
	if state.NetworkLimitations != nil && g.NetworkLimitations != nil {
		flattenPolicyNetworkLimitations(ctx, g.NetworkLimitations, state.NetworkLimitations)
	}
	if state.OverrideDefaultSettings != nil && g.OverrideDefaultSettings != nil {
		state.OverrideDefaultSettings.TargetDrive = preferCurrentStringPointer(g.OverrideDefaultSettings.TargetDrive, state.OverrideDefaultSettings.TargetDrive)
		state.OverrideDefaultSettings.DistributionPoint = preferCurrentStringPointer(g.OverrideDefaultSettings.DistributionPoint, state.OverrideDefaultSettings.DistributionPoint)
		state.OverrideDefaultSettings.ForceAfpSmb = preferCurrentBoolPointer(g.OverrideDefaultSettings.ForceAfpSmb, state.OverrideDefaultSettings.ForceAfpSmb)
		state.OverrideDefaultSettings.Sus = preferCurrentStringPointer(g.OverrideDefaultSettings.Sus, state.OverrideDefaultSettings.Sus)
	}
}

func flattenPolicyDateTimeLimitations(ctx context.Context, dtl *proclassic.PolicyGeneralDateTimeLimitations, state *PolicyGeneralDateTimeLimitationsModel) {
	state.ActivationDate = preferCurrentStringPointer(dtl.ActivationDate, state.ActivationDate)
	state.ExpirationDate = preferCurrentStringPointer(dtl.ExpirationDate, state.ExpirationDate)
	state.NoExecuteStart = preferCurrentStringPointer(dtl.NoExecuteStart, state.NoExecuteStart)
	state.NoExecuteEnd = preferCurrentStringPointer(dtl.NoExecuteEnd, state.NoExecuteEnd)

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
	state.MinimumNetworkConnection = preferCurrentStringPointer(nl.MinimumNetworkConnection, state.MinimumNetworkConnection)
	state.AnyIPAddress = preferCurrentBoolPointer(nl.AnyIPAddress, state.AnyIPAddress)
	if nl.NetworkSegments != nil {
		set, _ := scope.FlattenIDSlice(ctx, nl.NetworkSegments.NetworkSegment, func(i proclassic.IDName) *int { return i.ID })
		state.NetworkSegmentIDs = set
	} else {
		state.NetworkSegmentIDs = types.SetNull(types.StringType)
	}
}

func flattenPolicyScope(ctx context.Context, s *proclassic.PolicyScope, state *PolicyScopeModel) diag.Diagnostics {
	var diags diag.Diagnostics
	state.AllComputers = preferCurrentBoolPointer(s.AllComputers, state.AllComputers)
	state.AllJssUsers = preferCurrentBoolPointer(s.AllJssUsers, state.AllJssUsers)

	state.ComputerIDs = flattenComputerItemSet(ctx, s.Computers)
	state.ComputerGroupIDs = flattenIDNameSet(ctx, idNameSliceFromGroups(s.ComputerGroups))
	state.BuildingIDs = flattenIDNameSet(ctx, idNameSliceFromBuildings(s.Buildings))
	state.DepartmentIDs = flattenIDNameSet(ctx, idNameSliceFromDepartments(s.Departments))
	state.JssUserIDs = flattenIDNameSet(ctx, idNameSliceFromJssUsers(s.JssUsers))
	state.JssUserGroupIDs = flattenIDNameSet(ctx, idNameSliceFromJssUserGroups(s.JssUserGroups))

	if state.Limitations != nil && s.Limitations != nil {
		l := s.Limitations
		state.Limitations.NetworkSegmentIDs = flattenIDNameSet(ctx, idNameSliceFromLimitationsSegments(l.NetworkSegments))
		state.Limitations.IbeaconIDs = flattenIDNameSet(ctx, idNameSliceFromLimitationsIbeacons(l.Ibeacons))
		state.Limitations.DirectoryServiceOrLocalUserNames = flattenNameSet(ctx, idNameSliceFromLimitationsUsers(l.Users))
		state.Limitations.DirectoryServiceUserGroupNames = flattenNameSet(ctx, idNameSliceFromLimitationsUserGroups(l.UserGroups))
	}

	if state.Exclusions != nil && s.Exclusions != nil {
		e := s.Exclusions
		state.Exclusions.ComputerIDs = flattenExclComputerItemSet(ctx, e.Computers)
		state.Exclusions.ComputerGroupIDs = flattenIDNameSet(ctx, idNameSliceFromExclComputerGroups(e.ComputerGroups))
		state.Exclusions.BuildingIDs = flattenIDNameSet(ctx, idNameSliceFromExclBuildings(e.Buildings))
		state.Exclusions.DepartmentIDs = flattenIDNameSet(ctx, idNameSliceFromExclDepartments(e.Departments))
		state.Exclusions.JssUserIDs = flattenIDNameSet(ctx, idNameSliceFromExclJssUsers(e.JssUsers))
		state.Exclusions.JssUserGroupIDs = flattenIDNameSet(ctx, idNameSliceFromExclJssUserGroups(e.JssUserGroups))
		state.Exclusions.NetworkSegmentIDs = flattenExclNetworkSegmentSet(ctx, e.NetworkSegments)
		state.Exclusions.IbeaconIDs = flattenIDNameSet(ctx, idNameSliceFromExclIbeacons(e.Ibeacons))
		state.Exclusions.DirectoryServiceOrLocalUserNames = flattenExclUsersNameSet(ctx, e.Users)
		state.Exclusions.DirectoryServiceUserGroupNames = flattenNameSet(ctx, idNameSliceFromExclUserGroups(e.UserGroups))
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

func flattenIDNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenIDSlice(ctx, items, func(i proclassic.IDName) *int { return i.ID })
	return out
}

func flattenNameSet(ctx context.Context, items *[]proclassic.IDName) types.Set {
	out, _ := scope.FlattenNameSlice(ctx, items, func(i proclassic.IDName) *string { return i.Name })
	return out
}

// ---- self_service / packages / scripts / printers / dock_items / etc. ----------

func flattenPolicySelfService(ss *proclassic.PolicySelfService, state *PolicySelfServiceModel) {
	state.UseForSelfService = preferCurrentBoolPointer(ss.UseForSelfService, state.UseForSelfService)
	state.SelfServiceDisplayName = preferCurrentStringPointer(ss.SelfServiceDisplayName, state.SelfServiceDisplayName)
	state.InstallButtonText = preferCurrentStringPointer(ss.InstallButtonText, state.InstallButtonText)
	state.ReinstallButtonText = preferCurrentStringPointer(ss.ReinstallButtonText, state.ReinstallButtonText)
	state.SelfServiceDescription = preferCurrentStringPointer(ss.SelfServiceDescription, state.SelfServiceDescription)
	state.ForceUsersToViewDescription = preferCurrentBoolPointer(ss.ForceUsersToViewDescription, state.ForceUsersToViewDescription)
	state.FeatureOnMainPage = preferCurrentBoolPointer(ss.FeatureOnMainPage, state.FeatureOnMainPage)
	state.NotificationEnabled = flattenNotificationEnabled(ss.Notification, state.NotificationEnabled)
	state.NotificationType = preferCurrentStringPointer(ss.NotificationType, state.NotificationType)
	state.NotificationSubject = preferCurrentStringPointer(ss.NotificationSubject, state.NotificationSubject)
	state.NotificationMessage = preferCurrentStringPointer(ss.NotificationMessage, state.NotificationMessage)

	if state.SelfServiceIcon != nil && ss.SelfServiceIcon != nil {
		state.SelfServiceIcon.ID = preferCurrentStringPointer(stringFromIntPtr(ss.SelfServiceIcon.ID), state.SelfServiceIcon.ID)
		state.SelfServiceIcon.URI = preferCurrentStringPointer(ss.SelfServiceIcon.URI, state.SelfServiceIcon.URI)
		state.SelfServiceIcon.Filename = preferCurrentStringPointer(ss.SelfServiceIcon.Filename, state.SelfServiceIcon.Filename)
	}

	if state.Category != nil && ss.SelfServiceCategories != nil && ss.SelfServiceCategories.Category != nil && len(*ss.SelfServiceCategories.Category) > 0 {
		c := (*ss.SelfServiceCategories.Category)[0]
		state.Category.ID = preferCurrentStringPointer(stringFromIntPtr(c.ID), state.Category.ID)
		state.Category.Name = preferCurrentStringPointer(c.Name, state.Category.Name)
		state.Category.DisplayIn = preferCurrentBoolPointer(c.DisplayIn, state.Category.DisplayIn)
		state.Category.FeatureIn = preferCurrentBoolPointer(c.FeatureIn, state.Category.FeatureIn)
	}
}

func flattenPolicyPackageConfiguration(pc *proclassic.PolicyPackageConfiguration, state *PolicyPackageConfigurationModel) {
	state.DistributionPoint = preferCurrentStringPointer(pc.DistributionPoint, state.DistributionPoint)

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
	if pr.Size != nil {
		state.Size = types.Int64Value(int64(*pr.Size))
	} else {
		state.Size = types.Int64Null()
	}
	state.LeaveExistingDefault = preferCurrentBoolPointer(pr.LeaveExistingDefault, state.LeaveExistingDefault)

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
			Action:      helpers.StringPointerValueOrNull(p.Action),
			MakeDefault: helpers.BoolPointerValueOrNull(p.MakeDefault),
		})
	}
	state.Printers = out
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

func flattenPolicyAccountMaintenance(am *proclassic.PolicyAccountMaintenance, state *PolicyAccountMaintenanceModel) {
	if am.Accounts != nil && am.Accounts.Account != nil {
		// Build a username → planned-plaintext-password lookup so the Set's
		// unordered round-trip preserves the per-account Sensitive plaintext.
		// The server never echoes the plaintext, so an index-based pairing is
		// wrong: Set hashing reorders items and the i-th flattened entry will
		// not correspond to the i-th state entry. Username is the natural key
		// for Create/Reset/Delete/DisableFileVault — Probe 9 confirmed the
		// classic API accepts duplicate usernames across actions, so this
		// preserves plaintext correctly even in the multi-action policy
		// pattern (policy 6791 baseline).
		// passwordByUsername + shaByUsername lookup is LOAD-BEARING for any
		// policy where the Set may reorder between plan and apply. Do NOT
		// "simplify" to index-based pairing — Sets are inherently unordered,
		// and the classic API has been observed reordering account entries on
		// round-trip, so the i-th flattened entry is not guaranteed to match
		// the i-th state entry. The username-keyed pair lets us:
		//   - Preserve the Sensitive plaintext `password` across the Sets's
		//     unordered round-trip (server never echoes plaintext).
		//   - Carry the prior `password_sha256` forward via
		//     preferServerOrCurrentString when the server omits the SHA echo
		//     on Update (same `""` / `nil` regression that hits
		//     open_firmware_efi_password.of_password_sha256 — applied
		//     prophylactically here so an Update-step acc test won't regress).
		passwordByUsername := make(map[string]types.String, len(state.Accounts))
		shaByUsername := make(map[string]types.String, len(state.Accounts))
		for _, prev := range state.Accounts {
			if !prev.Username.IsNull() && !prev.Username.IsUnknown() {
				passwordByUsername[prev.Username.ValueString()] = prev.Password
				shaByUsername[prev.Username.ValueString()] = prev.PasswordSha256
			}
		}

		items := *am.Accounts.Account
		out := make([]PolicyAccountItemModel, 0, len(items))
		for _, a := range items {
			currentPassword := types.StringNull()
			currentSha := types.StringNull()
			if a.Username != nil {
				if p, ok := passwordByUsername[*a.Username]; ok {
					currentPassword = p
				}
				if s, ok := shaByUsername[*a.Username]; ok {
					currentSha = s
				}
			}
			out = append(out, PolicyAccountItemModel{
				Action:                         helpers.StringPointerValueOrNull(a.Action),
				Username:                       helpers.StringPointerValueOrNull(a.Username),
				Realname:                       helpers.StringPointerValueOrNull(a.Realname),
				Password:                       currentPassword, // server doesn't echo plaintext — preserve plan/state value
				PasswordSha256:                 preferServerOrCurrentString(a.PasswordSha256, currentSha),
				PermanentlyDeleteHomeDirectory: invertBoolPointerValueOrNull(a.ArchiveHomeDirectory),
				ArchiveHomeDirectoryTo:         helpers.StringPointerValueOrNull(a.ArchiveHomeDirectoryTo),
				Home:                           helpers.StringPointerValueOrNull(a.Home),
				Hint:                           helpers.StringPointerValueOrNull(a.Hint),
				Picture:                        helpers.StringPointerValueOrNull(a.Picture),
				Admin:                          helpers.BoolPointerValueOrNull(a.Admin),
				FilevaultEnabled:               helpers.BoolPointerValueOrNull(a.FilevaultEnabled),
				SecureTokenAllowed:             helpers.BoolPointerValueOrNull(a.SecureTokenAllowed),
			})
		}
		state.Accounts = out
	} else {
		state.Accounts = nil
	}

	if am.DirectoryBindings != nil && am.DirectoryBindings.Binding != nil {
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
		state.ManagementAccount.Action = preferCurrentStringPointer(am.ManagementAccount.Action, state.ManagementAccount.Action)
		// managed_password not echoed — preserve user-supplied
		state.ManagementAccount.ManagedPasswordLength = preferCurrentInt(am.ManagementAccount.ManagedPasswordLength, state.ManagementAccount.ManagedPasswordLength)
	}

	if state.OpenFirmwareEfiPassword != nil && am.OpenFirmwareEfiPassword != nil {
		state.OpenFirmwareEfiPassword.OfMode = preferCurrentStringPointer(am.OpenFirmwareEfiPassword.OfMode, state.OpenFirmwareEfiPassword.OfMode)
		// of_password not echoed — preserve user-supplied. of_password_sha256
		// is the Jamf-returned sentinel `********************` once a password
		// is set, but the server intermittently omits the echo on Update
		// round-trips (observed: of_mode `command` → `full` clears the SHA
		// from the response even though the password is still stored).
		// Preserve the prior state SHA when the server returns nil so the
		// Computed attribute does not flip back to null — Update-time
		// inconsistency would otherwise trip the framework's
		// "produced inconsistent result after apply" check.
		state.OpenFirmwareEfiPassword.OfPasswordSha256 = preferServerOrCurrentString(am.OpenFirmwareEfiPassword.OfPasswordSha256, state.OpenFirmwareEfiPassword.OfPasswordSha256)
	}
}

func flattenPolicyReboot(r *proclassic.PolicyReboot, state *PolicyRebootModel) {
	state.Message = preferCurrentStringPointer(r.Message, state.Message)
	state.StartupDisk = preferCurrentStringPointer(r.StartupDisk, state.StartupDisk)
	state.SpecifyStartup = preferCurrentStringPointer(r.SpecifyStartup, state.SpecifyStartup)
	state.NoUserLoggedIn = preferCurrentStringPointer(r.NoUserLoggedIn, state.NoUserLoggedIn)
	state.UserLoggedIn = preferCurrentStringPointer(r.UserLoggedIn, state.UserLoggedIn)
	state.MinutesUntilReboot = preferCurrentInt(r.MinutesUntilReboot, state.MinutesUntilReboot)
	state.StartRebootTimerImmediately = preferCurrentBoolPointer(r.StartRebootTimerImmediately, state.StartRebootTimerImmediately)
	state.FileVault2Reboot = preferCurrentBoolPointer(r.FileVault2Reboot, state.FileVault2Reboot)
}

func flattenPolicyMaintenance(m *proclassic.PolicyMaintenance, state *PolicyMaintenanceModel) {
	state.Recon = preferCurrentBoolPointer(m.Recon, state.Recon)
	state.ResetName = preferCurrentBoolPointer(m.ResetName, state.ResetName)
	state.InstallAllCachedPackages = preferCurrentBoolPointer(m.InstallAllCachedPackages, state.InstallAllCachedPackages)
	state.Heal = preferCurrentBoolPointer(m.Heal, state.Heal)
	state.Prebindings = preferCurrentBoolPointer(m.Prebindings, state.Prebindings)
	state.Permissions = preferCurrentBoolPointer(m.Permissions, state.Permissions)
	state.Byhost = preferCurrentBoolPointer(m.Byhost, state.Byhost)
	state.SystemCache = preferCurrentBoolPointer(m.SystemCache, state.SystemCache)
	state.UserCache = preferCurrentBoolPointer(m.UserCache, state.UserCache)
	state.Verify = preferCurrentBoolPointer(m.Verify, state.Verify)
}

func flattenPolicyFilesProcesses(fp *proclassic.PolicyFilesProcesses, state *PolicyFilesProcessesModel) {
	state.SearchByPath = preferCurrentStringPointer(fp.SearchByPath, state.SearchByPath)
	state.DeleteFile = preferCurrentBoolPointer(fp.DeleteFile, state.DeleteFile)
	state.LocateFile = preferCurrentStringPointer(fp.LocateFile, state.LocateFile)
	state.UpdateLocateDatabase = preferCurrentBoolPointer(fp.UpdateLocateDatabase, state.UpdateLocateDatabase)
	state.SpotlightSearch = preferCurrentStringPointer(fp.SpotlightSearch, state.SpotlightSearch)
	state.SearchForProcess = preferCurrentStringPointer(fp.SearchForProcess, state.SearchForProcess)
	state.KillProcess = preferCurrentBoolPointer(fp.KillProcess, state.KillProcess)
	state.RunCommand = preferCurrentStringPointer(fp.RunCommand, state.RunCommand)
}

func flattenPolicyUserInteraction(u *proclassic.PolicyUserInteraction, state *PolicyUserInteractionModel) {
	state.MessageStart = preferCurrentStringPointer(u.MessageStart, state.MessageStart)
	state.AllowUsersToDefer = preferCurrentBoolPointer(u.AllowUsersToDefer, state.AllowUsersToDefer)
	state.AllowDeferralUntilUtc = preferCurrentStringPointer(u.AllowDeferralUntilUtc, state.AllowDeferralUntilUtc)
	state.AllowDeferralMinutes = preferCurrentInt(u.AllowDeferralMinutes, state.AllowDeferralMinutes)
	state.MessageFinish = preferCurrentStringPointer(u.MessageFinish, state.MessageFinish)
}

func flattenPolicyDiskEncryption(d *proclassic.PolicyDiskEncryption, state *PolicyDiskEncryptionModel) {
	state.Action = preferCurrentStringPointer(d.Action, state.Action)
	state.DiskEncryptionConfigurationID = preferCurrentInt(d.DiskEncryptionConfigurationID, state.DiskEncryptionConfigurationID)
	state.AuthRestart = preferCurrentBoolPointer(d.AuthRestart, state.AuthRestart)
	state.RemediateKeyType = preferCurrentStringPointer(d.RemediateKeyType, state.RemediateKeyType)
	state.RemediateDiskEncryptionConfigurationID = preferCurrentInt(d.RemediateDiskEncryptionConfigurationID, state.RemediateDiskEncryptionConfigurationID)
}

// ---- small helpers -------------------------------------------------------------

func stringFromIntPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}
