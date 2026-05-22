// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

// buildPolicyInput projects a plan PolicyResourceModel into a *proclassic.PolicyPost
// suitable for Create / Update. Each section follows SCOPE_SPIKE §6.5 omission
// semantics: nil-pointer sub-blocks suppress wire emission entirely; empty
// child collections collapse all the way up to a nil parent.
func buildPolicyInput(ctx context.Context, plan PolicyResourceModel) (*proclassic.PolicyPost, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &proclassic.PolicyPost{}

	if plan.General != nil {
		general, d := buildPolicyGeneral(ctx, plan.General)
		diags.Append(d...)
		out.General = general
	}

	if plan.Scope != nil {
		s, d := buildPolicyScope(ctx, plan.Scope)
		diags.Append(d...)
		out.Scope = s
	}

	if plan.SelfService != nil {
		out.SelfService = buildPolicySelfService(plan.SelfService)
	}

	if plan.PackageConfiguration != nil {
		out.PackageConfiguration = buildPolicyPackageConfiguration(plan.PackageConfiguration)
	}

	if plan.Scripts != nil {
		out.Scripts = buildPolicyScripts(plan.Scripts)
	}

	if plan.Printers != nil {
		out.Printers = buildPolicyPrinters(plan.Printers)
	}

	if plan.DockItems != nil {
		out.DockItems = buildPolicyDockItems(plan.DockItems)
	}

	if plan.AccountMaintenance != nil {
		out.AccountMaintenance = buildPolicyAccountMaintenance(plan.AccountMaintenance)
	}

	if plan.Reboot != nil {
		out.Reboot = buildPolicyReboot(plan.Reboot)
	}

	if plan.Maintenance != nil {
		out.Maintenance = buildPolicyMaintenance(plan.Maintenance)
	}

	if plan.FilesProcesses != nil {
		out.FilesProcesses = buildPolicyFilesProcesses(plan.FilesProcesses)
	}

	if plan.UserInteraction != nil {
		out.UserInteraction = buildPolicyUserInteraction(plan.UserInteraction)
	}

	if plan.DiskEncryption != nil {
		out.DiskEncryption = buildPolicyDiskEncryption(plan.DiskEncryption)
	}

	return out, diags
}

func buildPolicyGeneral(ctx context.Context, m *PolicyGeneralModel) (*proclassic.PolicyPostGeneral, diag.Diagnostics) {
	var diags diag.Diagnostics
	g := &proclassic.PolicyPostGeneral{
		Name:                       helpers.OptionalStringPointer(m.Name),
		Enabled:                    optionalBoolPointer(m.Enabled),
		Trigger:                    helpers.OptionalStringPointer(m.Trigger),
		TriggerCheckin:             optionalBoolPointer(m.TriggerCheckin),
		TriggerEnrollmentComplete:  optionalBoolPointer(m.TriggerEnrollmentComplete),
		TriggerLogin:               optionalBoolPointer(m.TriggerLogin),
		TriggerLogout:              optionalBoolPointer(m.TriggerLogout),
		TriggerNetworkStateChanged: optionalBoolPointer(m.TriggerNetworkStateChanged),
		TriggerStartup:             optionalBoolPointer(m.TriggerStartup),
		TriggerOther:               helpers.OptionalStringPointer(m.TriggerOther),
		Frequency:                  helpers.OptionalStringPointer(m.Frequency),
		RetryEvent:                 helpers.OptionalStringPointer(m.RetryEvent),
		RetryAttempts:              optionalInt64ToInt(m.RetryAttempts),
		NotifyOnEachFailedRetry:    optionalBoolPointer(m.NotifyOnEachFailedRetry),
		LocationUserOnly:           optionalBoolPointer(m.LocationUserOnly),
		TargetDrive:                helpers.OptionalStringPointer(m.TargetDrive),
		Offline:                    optionalBoolPointer(m.Offline),
		NetworkRequirements:        helpers.OptionalStringPointer(m.NetworkRequirements),
	}

	if catID := stringIDPtr(m.CategoryID); catID != nil {
		g.Category = &proclassic.CategoryObject{ID: catID}
	}
	if siteID := stringIDPtr(m.SiteID); siteID != nil {
		g.Site = &proclassic.SiteObject{ID: siteID}
	}

	if m.DateTimeLimitations != nil {
		dtl, d := buildPolicyDateTimeLimitations(ctx, m.DateTimeLimitations)
		diags.Append(d...)
		g.DateTimeLimitations = dtl
	}

	if m.NetworkLimitations != nil {
		nl, d := buildPolicyNetworkLimitations(ctx, m.NetworkLimitations)
		diags.Append(d...)
		g.NetworkLimitations = nl
	}

	if m.OverrideDefaultSettings != nil {
		g.OverrideDefaultSettings = &proclassic.PolicyGeneralOverrideDefaultSettings{
			TargetDrive:       helpers.OptionalStringPointer(m.OverrideDefaultSettings.TargetDrive),
			DistributionPoint: helpers.OptionalStringPointer(m.OverrideDefaultSettings.DistributionPoint),
			ForceAfpSmb:       optionalBoolPointer(m.OverrideDefaultSettings.ForceAfpSmb),
			Sus:               helpers.OptionalStringPointer(m.OverrideDefaultSettings.Sus),
		}
	}

	return g, diags
}

func buildPolicyDateTimeLimitations(ctx context.Context, m *PolicyGeneralDateTimeLimitationsModel) (*proclassic.PolicyGeneralDateTimeLimitations, diag.Diagnostics) {
	var diags diag.Diagnostics
	dtl := &proclassic.PolicyGeneralDateTimeLimitations{
		ActivationDate:      helpers.OptionalStringPointer(m.ActivationDate),
		ActivationDateEpoch: optionalInt64ToInt(m.ActivationDateEpoch),
		ActivationDateUtc:   helpers.OptionalStringPointer(m.ActivationDateUtc),
		ExpirationDate:      helpers.OptionalStringPointer(m.ExpirationDate),
		ExpirationDateUtc:   helpers.OptionalStringPointer(m.ExpirationDateUtc),
		NoExecuteStart:      helpers.OptionalStringPointer(m.NoExecuteStart),
		NoExecuteEnd:        helpers.OptionalStringPointer(m.NoExecuteEnd),
	}

	if helpers.IsConfiguredValue(m.ExpirationDateEpoch) {
		bi := &proclassic.BigInt{}
		if ok := bi.SetString(m.ExpirationDateEpoch.ValueString()); ok {
			dtl.ExpirationDateEpoch = bi
		} else {
			diags.AddError(
				"Invalid expiration_date_epoch",
				"expiration_date_epoch must be a base-10 integer string; got "+strconv.Quote(m.ExpirationDateEpoch.ValueString()),
			)
		}
	}

	if helpers.IsConfiguredValue(m.NoExecuteOn) {
		var days []string
		diags.Append(m.NoExecuteOn.ElementsAs(ctx, &days, false)...)
		if !diags.HasError() && len(days) > 0 {
			dtl.NoExecuteOn = &proclassic.PolicyGeneralDateTimeLimitationsNoExecuteOn{
				Day: &days,
			}
		}
	}

	return dtl, diags
}

func buildPolicyNetworkLimitations(ctx context.Context, m *PolicyGeneralNetworkLimitationsModel) (*proclassic.PolicyGeneralNetworkLimitations, diag.Diagnostics) {
	var diags diag.Diagnostics
	nl := &proclassic.PolicyGeneralNetworkLimitations{
		MinimumNetworkConnection: helpers.OptionalStringPointer(m.MinimumNetworkConnection),
		AnyIPAddress:             optionalBoolPointer(m.AnyIPAddress),
	}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		nl.NetworkSegments = &proclassic.PolicyGeneralNetworkLimitationsNetworkSegments{
			NetworkSegment: segs,
		}
	}

	return nl, diags
}

func buildPolicyScope(ctx context.Context, m *PolicyScopeModel) (*proclassic.PolicyPostScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	s := &proclassic.PolicyPostScope{
		AllComputers: optionalBoolPointer(m.AllComputers),
	}

	computers, d := scope.BuildIDSlice(ctx, m.ComputerIDs, func(id int) proclassic.PolicyScopeComputersComputerItem {
		return proclassic.PolicyScopeComputersComputerItem{ID: &id}
	})
	diags.Append(d...)
	if computers != nil {
		s.Computers = &proclassic.PolicyScopeComputers{Computer: computers}
	}

	computerGroups, d := scope.BuildIDSlice(ctx, m.ComputerGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if computerGroups != nil {
		s.ComputerGroups = &proclassic.PolicyScopeComputerGroups{ComputerGroup: computerGroups}
	}

	buildings, d := scope.BuildIDSlice(ctx, m.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		s.Buildings = &proclassic.PolicyScopeBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, m.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		s.Departments = &proclassic.PolicyScopeDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, m.JssUserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		s.JssUsers = &proclassic.PolicyScopeJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, m.JssUserGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUserGroups != nil {
		s.JssUserGroups = &proclassic.PolicyScopeJssUserGroups{UserGroup: jssUserGroups}
	}

	if m.Limitations != nil {
		l, ld := buildPolicyScopeLimitations(ctx, m.Limitations)
		diags.Append(ld...)
		s.Limitations = l
	}

	if m.Exclusions != nil {
		e, ed := buildPolicyScopeExclusions(ctx, m.Exclusions)
		diags.Append(ed...)
		s.Exclusions = e
	}

	// Omission semantics (SCOPE_SPIKE §6.5): collapse to nil when every child
	// pointer is nil so the wire payload omits <scope> entirely rather than
	// emitting an empty <scope></scope> element.
	if s.AllComputers == nil && s.Computers == nil && s.ComputerGroups == nil &&
		s.Buildings == nil && s.Departments == nil && s.JssUsers == nil &&
		s.JssUserGroups == nil && s.Limitations == nil && s.Exclusions == nil {
		return nil, diags
	}
	return s, diags
}

func buildPolicyScopeLimitations(ctx context.Context, m *PolicyScopeLimitationsModel) (*proclassic.PolicyScopeLimitations, diag.Diagnostics) {
	var diags diag.Diagnostics
	l := &proclassic.PolicyScopeLimitations{}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		l.NetworkSegments = &proclassic.PolicyScopeLimitationsNetworkSegments{NetworkSegment: segs}
	}

	ibeacons, d := scope.BuildIDSlice(ctx, m.IbeaconIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if ibeacons != nil {
		l.Ibeacons = &proclassic.PolicyScopeLimitationsIbeacons{Ibeacon: ibeacons}
	}

	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		l.Users = &proclassic.PolicyScopeLimitationsUsers{User: users}
	}

	userGroups, d := scope.BuildNameSlice(ctx, m.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if userGroups != nil {
		l.UserGroups = &proclassic.PolicyScopeLimitationsUserGroups{UserGroup: userGroups}
	}

	// Collapse to nil if no child set was assigned.
	if l.NetworkSegments == nil && l.Ibeacons == nil && l.Users == nil && l.UserGroups == nil {
		return nil, diags
	}
	return l, diags
}

func buildPolicyScopeExclusions(ctx context.Context, m *PolicyScopeExclusionsModel) (*proclassic.PolicyScopeExclusions, diag.Diagnostics) {
	var diags diag.Diagnostics
	e := &proclassic.PolicyScopeExclusions{}

	computers, d := scope.BuildIDSlice(ctx, m.ComputerIDs, func(id int) proclassic.PolicyScopeExclusionsComputersComputerItem {
		return proclassic.PolicyScopeExclusionsComputersComputerItem{ID: &id}
	})
	diags.Append(d...)
	if computers != nil {
		e.Computers = &proclassic.PolicyScopeExclusionsComputers{Computer: computers}
	}

	computerGroups, d := scope.BuildIDSlice(ctx, m.ComputerGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if computerGroups != nil {
		e.ComputerGroups = &proclassic.PolicyScopeExclusionsComputerGroups{ComputerGroup: computerGroups}
	}

	buildings, d := scope.BuildIDSlice(ctx, m.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		e.Buildings = &proclassic.PolicyScopeExclusionsBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, m.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		e.Departments = &proclassic.PolicyScopeExclusionsDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, m.JssUserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		e.JssUsers = &proclassic.PolicyScopeExclusionsJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, m.JssUserGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUserGroups != nil {
		e.JssUserGroups = &proclassic.PolicyScopeExclusionsJssUserGroups{UserGroup: jssUserGroups}
	}

	segs, d := scope.BuildIDSlice(ctx, m.NetworkSegmentIDs, func(id int) proclassic.PolicyScopeExclusionsNetworkSegmentsNetworkSegmentItem {
		return proclassic.PolicyScopeExclusionsNetworkSegmentsNetworkSegmentItem{ID: &id}
	})
	diags.Append(d...)
	if segs != nil {
		e.NetworkSegments = &proclassic.PolicyScopeExclusionsNetworkSegments{NetworkSegment: segs}
	}

	ibeacons, d := scope.BuildIDSlice(ctx, m.IbeaconIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if ibeacons != nil {
		e.Ibeacons = &proclassic.PolicyScopeExclusionsIbeacons{Ibeacon: ibeacons}
	}

	users, d := scope.BuildNameSlice(ctx, m.DirectoryServiceOrLocalUserNames, func(name string) proclassic.PolicyScopeExclusionsUsersUserItem {
		n := name
		return proclassic.PolicyScopeExclusionsUsersUserItem{Name: &n}
	})
	diags.Append(d...)
	if users != nil {
		e.Users = &proclassic.PolicyScopeExclusionsUsers{User: users}
	}

	userGroups, d := scope.BuildNameSlice(ctx, m.DirectoryServiceUserGroupNames, func(name string) proclassic.IDName {
		n := name
		return proclassic.IDName{Name: &n}
	})
	diags.Append(d...)
	if userGroups != nil {
		e.UserGroups = &proclassic.PolicyScopeExclusionsUserGroups{UserGroup: userGroups}
	}

	if e.Computers == nil && e.ComputerGroups == nil && e.Buildings == nil && e.Departments == nil &&
		e.JssUsers == nil && e.JssUserGroups == nil && e.NetworkSegments == nil && e.Ibeacons == nil &&
		e.Users == nil && e.UserGroups == nil {
		return nil, diags
	}
	return e, diags
}

func buildPolicySelfService(m *PolicySelfServiceModel) *proclassic.PolicyPostSelfService {
	ss := &proclassic.PolicyPostSelfService{
		UseForSelfService:           optionalBoolPointer(m.UseForSelfService),
		SelfServiceDisplayName:      helpers.OptionalStringPointer(m.SelfServiceDisplayName),
		InstallButtonText:           helpers.OptionalStringPointer(m.InstallButtonText),
		ReinstallButtonText:         helpers.OptionalStringPointer(m.ReinstallButtonText),
		SelfServiceDescription:      helpers.OptionalStringPointer(m.SelfServiceDescription),
		ForceUsersToViewDescription: optionalBoolPointer(m.ForceUsersToViewDescription),
		FeatureOnMainPage:           optionalBoolPointer(m.FeatureOnMainPage),
		Notification:                buildNotificationEnabled(m.NotificationEnabled),
		NotificationType:            helpers.OptionalStringPointer(m.NotificationType),
		NotificationSubject:         helpers.OptionalStringPointer(m.NotificationSubject),
		NotificationMessage:         helpers.OptionalStringPointer(m.NotificationMessage),
	}

	if m.SelfServiceIcon != nil {
		icon := &proclassic.PolicySelfServiceSelfServiceIcon{
			ID:       stringIDPtr(m.SelfServiceIcon.ID),
			URI:      helpers.OptionalStringPointer(m.SelfServiceIcon.URI),
			Filename: helpers.OptionalStringPointer(m.SelfServiceIcon.Filename),
		}
		if icon.ID != nil || icon.URI != nil || icon.Filename != nil {
			ss.SelfServiceIcon = icon
		}
	}

	if m.Category != nil {
		cat := &proclassic.PolicySelfServiceSelfServiceCategoriesCategory{
			ID:        stringIDPtr(m.Category.ID),
			Name:      helpers.OptionalStringPointer(m.Category.Name),
			DisplayIn: optionalBoolPointer(m.Category.DisplayIn),
			FeatureIn: optionalBoolPointer(m.Category.FeatureIn),
		}
		if cat.ID != nil || cat.Name != nil || cat.DisplayIn != nil || cat.FeatureIn != nil {
			ss.SelfServiceCategories = &proclassic.PolicySelfServiceSelfServiceCategories{Category: cat}
		}
	}

	return ss
}

func buildPolicyPackageConfiguration(m *PolicyPackageConfigurationModel) *proclassic.PolicyPostPackageConfiguration {
	if len(m.Packages) == 0 {
		return nil
	}
	items := make([]proclassic.PolicyPackageConfigurationPackagesPackageItem, 0, len(m.Packages))
	for _, p := range m.Packages {
		items = append(items, proclassic.PolicyPackageConfigurationPackagesPackageItem{
			ID:            stringIDPtr(p.ID),
			Name:          helpers.OptionalStringPointer(p.Name),
			Action:        helpers.OptionalStringPointer(p.Action),
			Fut:           optionalBoolPointer(p.Fut),
			Feu:           optionalBoolPointer(p.Feu),
			UpdateAutorun: optionalBoolPointer(p.UpdateAutorun),
		})
	}
	return &proclassic.PolicyPostPackageConfiguration{
		Packages: &proclassic.PolicyPackageConfigurationPackages{Package: &items},
	}
}

func buildPolicyScripts(m *PolicyScriptsModel) *proclassic.PolicyPostScripts {
	if len(m.Scripts) == 0 {
		return nil
	}
	items := make([]proclassic.PolicyScriptsScriptItem, 0, len(m.Scripts))
	for _, s := range m.Scripts {
		items = append(items, proclassic.PolicyScriptsScriptItem{
			ID:          stringIDPtr(s.ID),
			Name:        helpers.OptionalStringPointer(s.Name),
			Priority:    helpers.OptionalStringPointer(s.Priority),
			Parameter4:  helpers.OptionalStringPointer(s.Parameter4),
			Parameter5:  helpers.OptionalStringPointer(s.Parameter5),
			Parameter6:  helpers.OptionalStringPointer(s.Parameter6),
			Parameter7:  helpers.OptionalStringPointer(s.Parameter7),
			Parameter8:  helpers.OptionalStringPointer(s.Parameter8),
			Parameter9:  helpers.OptionalStringPointer(s.Parameter9),
			Parameter10: helpers.OptionalStringPointer(s.Parameter10),
			Parameter11: helpers.OptionalStringPointer(s.Parameter11),
		})
	}
	return &proclassic.PolicyPostScripts{Script: &items}
}

func buildPolicyPrinters(m *PolicyPrintersModel) *proclassic.PolicyPostPrinters {
	p := &proclassic.PolicyPostPrinters{
		LeaveExistingDefault: optionalBoolPointer(m.LeaveExistingDefault),
	}
	if len(m.Printers) > 0 {
		items := make([]proclassic.PolicyPrintersPrinterItem, 0, len(m.Printers))
		for _, pr := range m.Printers {
			items = append(items, proclassic.PolicyPrintersPrinterItem{
				ID:          stringIDPtr(pr.ID),
				Name:        helpers.OptionalStringPointer(pr.Name),
				Action:      helpers.OptionalStringPointer(pr.Action),
				MakeDefault: optionalBoolPointer(pr.MakeDefault),
			})
		}
		p.Printer = &items
	}
	if p.LeaveExistingDefault == nil && p.Printer == nil {
		return nil
	}
	return p
}

func buildPolicyDockItems(m *PolicyDockItemsModel) *proclassic.PolicyPostDockItems {
	if len(m.DockItems) == 0 {
		return nil
	}
	items := make([]proclassic.PolicyPostDockItemsDockItemItem, 0, len(m.DockItems))
	for _, di := range m.DockItems {
		items = append(items, proclassic.PolicyPostDockItemsDockItemItem{
			ID:     stringIDPtr(di.ID),
			Name:   helpers.OptionalStringPointer(di.Name),
			Action: helpers.OptionalStringPointer(di.Action),
		})
	}
	return &proclassic.PolicyPostDockItems{DockItem: &items}
}

func buildPolicyAccountMaintenance(m *PolicyAccountMaintenanceModel) *proclassic.PolicyPostAccountMaintenance {
	am := &proclassic.PolicyPostAccountMaintenance{}

	if len(m.Accounts) > 0 {
		items := make([]proclassic.PolicyAccountMaintenanceAccountsAccountItem, 0, len(m.Accounts))
		for _, a := range m.Accounts {
			items = append(items, proclassic.PolicyAccountMaintenanceAccountsAccountItem{
				Action:                 helpers.OptionalStringPointer(a.Action),
				Username:               helpers.OptionalStringPointer(a.Username),
				Realname:               helpers.OptionalStringPointer(a.Realname),
				Password:               helpers.OptionalStringPointer(a.Password),
				ArchiveHomeDirectory:   optionalBoolPointer(a.ArchiveHomeDirectory),
				ArchiveHomeDirectoryTo: helpers.OptionalStringPointer(a.ArchiveHomeDirectoryTo),
				Home:                   helpers.OptionalStringPointer(a.Home),
				Hint:                   helpers.OptionalStringPointer(a.Hint),
				Picture:                helpers.OptionalStringPointer(a.Picture),
				Admin:                  optionalBoolPointer(a.Admin),
				FilevaultEnabled:       optionalBoolPointer(a.FilevaultEnabled),
				SecureTokenAllowed:     optionalBoolPointer(a.SecureTokenAllowed),
			})
		}
		am.Accounts = &proclassic.PolicyAccountMaintenanceAccounts{Account: &items}
	}

	if len(m.DirectoryBindings) > 0 {
		items := make([]proclassic.IDName, 0, len(m.DirectoryBindings))
		for _, b := range m.DirectoryBindings {
			items = append(items, proclassic.IDName{
				ID:   stringIDPtr(b.ID),
				Name: helpers.OptionalStringPointer(b.Name),
			})
		}
		am.DirectoryBindings = &proclassic.PolicyAccountMaintenanceDirectoryBindings{Binding: &items}
	}

	if m.ManagementAccount != nil {
		am.ManagementAccount = &proclassic.PolicyAccountMaintenanceManagementAccount{
			Action:                helpers.OptionalStringPointer(m.ManagementAccount.Action),
			ManagedPassword:       helpers.OptionalStringPointer(m.ManagementAccount.ManagedPassword),
			ManagedPasswordLength: optionalInt64ToInt(m.ManagementAccount.ManagedPasswordLength),
		}
	}

	if m.OpenFirmwareEfiPassword != nil {
		am.OpenFirmwareEfiPassword = &proclassic.PolicyAccountMaintenanceOpenFirmwareEfiPassword{
			OfMode:     helpers.OptionalStringPointer(m.OpenFirmwareEfiPassword.OfMode),
			OfPassword: helpers.OptionalStringPointer(m.OpenFirmwareEfiPassword.OfPassword),
		}
	}

	if am.Accounts == nil && am.DirectoryBindings == nil && am.ManagementAccount == nil && am.OpenFirmwareEfiPassword == nil {
		return nil
	}
	return am
}

func buildPolicyReboot(m *PolicyRebootModel) *proclassic.PolicyPostReboot {
	return &proclassic.PolicyPostReboot{
		Message:                     helpers.OptionalStringPointer(m.Message),
		StartupDisk:                 helpers.OptionalStringPointer(m.StartupDisk),
		SpecifyStartup:              helpers.OptionalStringPointer(m.SpecifyStartup),
		NoUserLoggedIn:              helpers.OptionalStringPointer(m.NoUserLoggedIn),
		UserLoggedIn:                helpers.OptionalStringPointer(m.UserLoggedIn),
		MinutesUntilReboot:          optionalInt64ToInt(m.MinutesUntilReboot),
		StartRebootTimerImmediately: optionalBoolPointer(m.StartRebootTimerImmediately),
		FileVault2Reboot:            optionalBoolPointer(m.FileVault2Reboot),
	}
}

func buildPolicyMaintenance(m *PolicyMaintenanceModel) *proclassic.PolicyPostMaintenance {
	return &proclassic.PolicyPostMaintenance{
		Recon:                    optionalBoolPointer(m.Recon),
		ResetName:                optionalBoolPointer(m.ResetName),
		InstallAllCachedPackages: optionalBoolPointer(m.InstallAllCachedPackages),
		Heal:                     optionalBoolPointer(m.Heal),
		Prebindings:              optionalBoolPointer(m.Prebindings),
		Permissions:              optionalBoolPointer(m.Permissions),
		Byhost:                   optionalBoolPointer(m.Byhost),
		SystemCache:              optionalBoolPointer(m.SystemCache),
		UserCache:                optionalBoolPointer(m.UserCache),
		Verify:                   optionalBoolPointer(m.Verify),
	}
}

func buildPolicyFilesProcesses(m *PolicyFilesProcessesModel) *proclassic.PolicyPostFilesProcesses {
	return &proclassic.PolicyPostFilesProcesses{
		SearchByPath:         helpers.OptionalStringPointer(m.SearchByPath),
		DeleteFile:           optionalBoolPointer(m.DeleteFile),
		LocateFile:           helpers.OptionalStringPointer(m.LocateFile),
		UpdateLocateDatabase: optionalBoolPointer(m.UpdateLocateDatabase),
		SpotlightSearch:      helpers.OptionalStringPointer(m.SpotlightSearch),
		SearchForProcess:     helpers.OptionalStringPointer(m.SearchForProcess),
		KillProcess:          optionalBoolPointer(m.KillProcess),
		RunCommand:           helpers.OptionalStringPointer(m.RunCommand),
	}
}

func buildPolicyUserInteraction(m *PolicyUserInteractionModel) *proclassic.PolicyPostUserInteraction {
	return &proclassic.PolicyPostUserInteraction{
		MessageStart:          helpers.OptionalStringPointer(m.MessageStart),
		AllowUsersToDefer:     optionalBoolPointer(m.AllowUsersToDefer),
		AllowDeferralUntilUtc: helpers.OptionalStringPointer(m.AllowDeferralUntilUtc),
		AllowDeferralMinutes:  optionalInt64ToInt(m.AllowDeferralMinutes),
		MessageFinish:         helpers.OptionalStringPointer(m.MessageFinish),
	}
}

func buildPolicyDiskEncryption(m *PolicyDiskEncryptionModel) *proclassic.PolicyPostDiskEncryption {
	return &proclassic.PolicyPostDiskEncryption{
		Action:                                 helpers.OptionalStringPointer(m.Action),
		DiskEncryptionConfigurationID:          optionalInt64ToInt(m.DiskEncryptionConfigurationID),
		AuthRestart:                            optionalBoolPointer(m.AuthRestart),
		RemediateKeyType:                       helpers.OptionalStringPointer(m.RemediateKeyType),
		RemediateDiskEncryptionConfigurationID: optionalInt64ToInt(m.RemediateDiskEncryptionConfigurationID),
	}
}
