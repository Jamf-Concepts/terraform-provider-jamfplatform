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

// policyAccountMaintenanceSecrets carries WriteOnly password values
// sourced from req.Config alongside per-account/per-section "should
// rotate?" decisions sourced by comparing plan vs state on
// `password_wo_version` companions. Create uses an all-true carrier
// (every plaintext is fresh). Update only flags rotated entries; the
// rest are omitted from the wire so Jamf retains its stored value
// under Classic's partial-merge semantics.
type policyAccountMaintenanceSecrets struct {
	// accountPasswords keyed by username — value is the plaintext to
	// emit on the wire, or nil to omit the <password/> element for
	// that account.
	accountPasswords map[string]*string
	// managedPassword is the management_account.managed_password
	// plaintext to emit, or nil to omit.
	managedPassword *string
	// ofPassword is the open_firmware_efi_password.of_password
	// plaintext to emit, or nil to omit.
	ofPassword *string
}

// noSecrets returns a carrier that emits no plaintext — every section
// keeps the existing stored value. Useful as the default for callers
// that do not touch account_maintenance secrets.
func noSecrets() *policyAccountMaintenanceSecrets {
	return &policyAccountMaintenanceSecrets{accountPasswords: map[string]*string{}}
}

// buildPolicyInput projects a plan PolicyResourceModel into a *proclassic.PolicyPost
// suitable for Create / Update. Each section follows the scope omission rules
// in STYLE_GUIDE.md §Scope helper: nil-pointer sub-blocks suppress wire emission
// entirely; empty child collections collapse all the way up to a nil parent.
func buildPolicyInput(ctx context.Context, plan PolicyResourceModel, secrets *policyAccountMaintenanceSecrets) (*proclassic.PolicyPost, diag.Diagnostics) {
	if secrets == nil {
		secrets = noSecrets()
	}
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

	if plan.Packages != nil {
		out.PackageConfiguration = buildPolicyPackageConfiguration(plan.Packages)
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

	// The four flattened account-maintenance blocks join back into a single
	// wire <account_maintenance> object. buildPolicyAccountMaintenance
	// self-guards: it returns nil when all four are absent, preserving the
	// "omit the wire object entirely" behaviour.
	out.AccountMaintenance = buildPolicyAccountMaintenance(plan, secrets)

	if plan.RestartOptions != nil {
		out.Reboot = buildPolicyReboot(plan.RestartOptions)
	}

	if plan.Maintenance != nil {
		out.Maintenance = buildPolicyMaintenance(plan.Maintenance)
	}

	if plan.FilesAndProcesses != nil {
		out.FilesProcesses = buildPolicyFilesProcesses(plan.FilesAndProcesses)
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
		Enabled:                    helpers.OptionalBoolPointer(m.Enabled),
		Trigger:                    helpers.OptionalStringPointer(m.Trigger),
		TriggerCheckin:             helpers.OptionalBoolPointer(m.TriggerCheckin),
		TriggerEnrollmentComplete:  helpers.OptionalBoolPointer(m.TriggerEnrollmentComplete),
		TriggerLogin:               helpers.OptionalBoolPointer(m.TriggerLogin),
		TriggerNetworkStateChanged: helpers.OptionalBoolPointer(m.TriggerNetworkStateChanged),
		TriggerStartup:             helpers.OptionalBoolPointer(m.TriggerStartup),
		TriggerOther:               helpers.OptionalStringPointer(m.TriggerOther),
		Frequency:                  helpers.OptionalStringPointer(m.Frequency),
		RetryEvent:                 helpers.OptionalStringPointer(m.RetryEvent),
		RetryAttempts:              optionalInt64ToInt(m.RetryAttempts),
		NotifyOnEachFailedRetry:    helpers.OptionalBoolPointer(m.NotifyOnEachFailedRetry),
		LocationUserOnly:           helpers.OptionalBoolPointer(m.LimitToJamfProAssignedUser),
		TargetDrive:                helpers.OptionalStringPointer(m.TargetDrive),
		Offline:                    helpers.OptionalBoolPointer(m.Offline),
		NetworkRequirements:        helpers.OptionalStringPointer(m.NetworkRequirements),
	}

	if catID := helpers.StringIDPtr(m.CategoryID); catID != nil {
		g.Category = &proclassic.CategoryObject{ID: catID}
	}
	if siteID := helpers.StringIDPtr(m.SiteID); siteID != nil {
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
			ForceAfpSmb:       helpers.OptionalBoolPointer(m.OverrideDefaultSettings.ForceAfpSmb),
			Sus:               helpers.OptionalStringPointer(m.OverrideDefaultSettings.Sus),
		}
	}

	return g, diags
}

func buildPolicyDateTimeLimitations(ctx context.Context, m *PolicyGeneralDateTimeLimitationsModel) (*proclassic.PolicyGeneralDateTimeLimitations, diag.Diagnostics) {
	var diags diag.Diagnostics
	dtl := &proclassic.PolicyGeneralDateTimeLimitations{
		ActivationDate: helpers.OptionalStringPointer(m.ActivationDate),
		ExpirationDate: helpers.OptionalStringPointer(m.ExpirationDate),
		NoExecuteStart: helpers.OptionalStringPointer(m.NoExecuteStart),
		NoExecuteEnd:   helpers.OptionalStringPointer(m.NoExecuteEnd),
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
		AnyIPAddress:             helpers.OptionalBoolPointer(m.AnyIPAddress),
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

func buildPolicyScope(ctx context.Context, m *scope.ComputerScopeModel) (*proclassic.PolicyPostScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	t := m.TargetsOrZero()
	s := &proclassic.PolicyPostScope{
		AllComputers: helpers.OptionalBoolPointer(t.AllComputers),
		AllJssUsers:  helpers.OptionalBoolPointer(t.AllJssUsers),
	}

	computers, d := scope.BuildIDSlice(ctx, t.ComputerIDs, func(id int) proclassic.PolicyScopeComputersComputerItem {
		return proclassic.PolicyScopeComputersComputerItem{ID: &id}
	})
	diags.Append(d...)
	if computers != nil {
		s.Computers = &proclassic.PolicyScopeComputers{Computer: computers}
	}

	computerGroups, d := scope.BuildIDSlice(ctx, t.ComputerGroupIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if computerGroups != nil {
		s.ComputerGroups = &proclassic.PolicyScopeComputerGroups{ComputerGroup: computerGroups}
	}

	buildings, d := scope.BuildIDSlice(ctx, t.BuildingIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if buildings != nil {
		s.Buildings = &proclassic.PolicyScopeBuildings{Building: buildings}
	}

	departments, d := scope.BuildIDSlice(ctx, t.DepartmentIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if departments != nil {
		s.Departments = &proclassic.PolicyScopeDepartments{Department: departments}
	}

	jssUsers, d := scope.BuildIDSlice(ctx, t.UserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		s.JssUsers = &proclassic.PolicyScopeJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, t.UserGroupIDs, func(id int) proclassic.IDName {
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

	// Omission semantics (STYLE_GUIDE.md §Scope helper): collapse to nil when
	// every child pointer is nil so the wire payload omits <scope> entirely
	// rather than emitting an empty <scope></scope> element.
	if s.AllComputers == nil && s.AllJssUsers == nil && s.Computers == nil &&
		s.ComputerGroups == nil && s.Buildings == nil && s.Departments == nil &&
		s.JssUsers == nil && s.JssUserGroups == nil &&
		s.Limitations == nil && s.Exclusions == nil {
		return nil, diags
	}
	return s, diags
}

func buildPolicyScopeLimitations(ctx context.Context, m *scope.ComputerScopeLimitationsModel) (*proclassic.PolicyScopeLimitations, diag.Diagnostics) {
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

	// Always emit the block when the caller's model declares `limitations`. A
	// scope PUT replaces the whole subtree, so an explicit empty element is the
	// clear gesture for a declared-empty category; undeclared (null) categories
	// are preserved upstream by the read-merge-write update, which hands this
	// builder a fully non-null merged model.
	return l, diags
}

func buildPolicyScopeExclusions(ctx context.Context, m *scope.ComputerScopeExclusionsModel) (*proclassic.PolicyScopeExclusions, diag.Diagnostics) {
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

	jssUsers, d := scope.BuildIDSlice(ctx, m.UserIDs, func(id int) proclassic.IDName {
		return proclassic.IDName{ID: &id}
	})
	diags.Append(d...)
	if jssUsers != nil {
		e.JssUsers = &proclassic.PolicyScopeExclusionsJssUsers{User: jssUsers}
	}

	jssUserGroups, d := scope.BuildIDSlice(ctx, m.UserGroupIDs, func(id int) proclassic.IDName {
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

	// Always emit the block when declared — see buildPolicyScopeLimitations.
	return e, diags
}

func buildPolicySelfService(m *PolicySelfServiceModel) *proclassic.PolicyPostSelfService {
	ss := &proclassic.PolicyPostSelfService{
		UseForSelfService:           helpers.OptionalBoolPointer(m.UseForSelfService),
		SelfServiceDisplayName:      helpers.OptionalStringPointer(m.SelfServiceDisplayName),
		InstallButtonText:           helpers.OptionalStringPointer(m.InstallButtonText),
		ReinstallButtonText:         helpers.OptionalStringPointer(m.ReinstallButtonText),
		SelfServiceDescription:      helpers.OptionalStringPointer(m.SelfServiceDescription),
		ForceUsersToViewDescription: helpers.OptionalBoolPointer(m.EnsureUsersViewDescription),
		FeatureOnMainPage:           helpers.OptionalBoolPointer(m.IncludeInFeaturedCategory),
		Notification:                buildNotificationEnabled(m.DisplayNotifications),
		NotificationType:            helpers.OptionalStringPointer(m.NotificationLocation),
		NotificationSubject:         helpers.OptionalStringPointer(m.NotificationSubject),
		NotificationMessage:         helpers.OptionalStringPointer(m.NotificationMessage),
	}

	if m.SelfServiceIcon != nil {
		icon := &proclassic.PolicySelfServiceSelfServiceIcon{
			ID:       helpers.StringIDPtr(m.SelfServiceIcon.ID),
			URI:      helpers.OptionalStringPointer(m.SelfServiceIcon.URI),
			Filename: helpers.OptionalStringPointer(m.SelfServiceIcon.Filename),
		}
		if icon.ID != nil || icon.URI != nil || icon.Filename != nil {
			ss.SelfServiceIcon = icon
		}
	}

	if len(m.Categories) > 0 {
		cats := make([]proclassic.PolicySelfServiceSelfServiceCategoriesCategoryItem, 0, len(m.Categories))
		for _, c := range m.Categories {
			cats = append(cats, proclassic.PolicySelfServiceSelfServiceCategoriesCategoryItem{
				ID:        helpers.StringIDPtr(c.ID),
				Name:      helpers.OptionalStringPointer(c.Name),
				DisplayIn: helpers.OptionalBoolPointer(c.DisplayIn),
				FeatureIn: helpers.OptionalBoolPointer(c.FeatureIn),
			})
		}
		ss.SelfServiceCategories = &proclassic.PolicySelfServiceSelfServiceCategories{Category: &cats}
	}

	return ss
}

func buildPolicyPackageConfiguration(m *PolicyPackagesModel) *proclassic.PolicyPostPackageConfiguration {
	dp := helpers.OptionalStringPointer(m.DistributionPoint)
	if len(m.Packages) == 0 && dp == nil {
		return nil
	}
	out := &proclassic.PolicyPostPackageConfiguration{
		DistributionPoint: dp,
	}
	if len(m.Packages) > 0 {
		items := make([]proclassic.PolicyPackageConfigurationPackagesPackageItem, 0, len(m.Packages))
		for _, p := range m.Packages {
			items = append(items, proclassic.PolicyPackageConfigurationPackagesPackageItem{
				ID:            helpers.StringIDPtr(p.ID),
				Name:          helpers.OptionalStringPointer(p.Name),
				Action:        helpers.OptionalStringPointer(p.Action),
				Fut:           helpers.OptionalBoolPointer(p.Fut),
				Feu:           helpers.OptionalBoolPointer(p.Feu),
				UpdateAutorun: helpers.OptionalBoolPointer(p.UpdateAutorun),
			})
		}
		out.Packages = &proclassic.PolicyPackageConfigurationPackages{Package: &items}
	}
	return out
}

func buildPolicyScripts(m *PolicyScriptsModel) *proclassic.PolicyPostScripts {
	if len(m.Scripts) == 0 {
		return nil
	}
	items := make([]proclassic.PolicyScriptsScriptItem, 0, len(m.Scripts))
	for _, s := range m.Scripts {
		items = append(items, proclassic.PolicyScriptsScriptItem{
			ID:          helpers.StringIDPtr(s.ID),
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
		LeaveExistingDefault: helpers.OptionalBoolPointer(m.LeaveExistingDefault),
	}
	if len(m.Printers) > 0 {
		items := make([]proclassic.PolicyPrintersPrinterItem, 0, len(m.Printers))
		for _, pr := range m.Printers {
			items = append(items, proclassic.PolicyPrintersPrinterItem{
				ID:          helpers.StringIDPtr(pr.ID),
				Name:        helpers.OptionalStringPointer(pr.Name),
				Action:      printerActionToWire(pr.Action),
				MakeDefault: helpers.OptionalBoolPointer(pr.MakeDefault),
			})
		}
		p.Printer = &items
	}
	if p.LeaveExistingDefault == nil && p.Printer == nil {
		return nil
	}
	return p
}

// printerActionToWire translates the UI-canonical Terraform attribute value
// (Map / Unmap) into the classic wire value (install / uninstall). The schema
// validator restricts user input to Map / Unmap; this function is the
// outbound half of that translation. Null/unknown values pass through as
// nil so the wire omits the <action> element.
func printerActionToWire(action types.String) *string {
	if !helpers.IsConfiguredValue(action) {
		return nil
	}
	switch action.ValueString() {
	case "Map":
		v := proclassic.PolicyPrintersPrinterItemActionInstall
		return &v
	case "Unmap":
		v := proclassic.PolicyPrintersPrinterItemActionUninstall
		return &v
	default:
		// Validator should have rejected anything else; fall through to
		// passthrough so an unexpected value surfaces in the server response.
		v := action.ValueString()
		return &v
	}
}

func buildPolicyDockItems(m *PolicyDockItemsModel) *proclassic.PolicyPostDockItems {
	if len(m.DockItems) == 0 {
		return nil
	}
	items := make([]proclassic.PolicyPostDockItemsDockItemItem, 0, len(m.DockItems))
	for _, di := range m.DockItems {
		items = append(items, proclassic.PolicyPostDockItemsDockItemItem{
			ID:     helpers.StringIDPtr(di.ID),
			Name:   helpers.OptionalStringPointer(di.Name),
			Action: helpers.OptionalStringPointer(di.Action),
		})
	}
	return &proclassic.PolicyPostDockItems{DockItem: &items}
}

// buildPolicyAccountMaintenance joins the four flattened account-maintenance
// blocks (local_accounts, management_account, directory_bindings,
// efi_password) back into the single classic wire <account_maintenance>
// object. It self-guards: when all four blocks are absent it returns nil so
// the wire omits the object entirely.
func buildPolicyAccountMaintenance(plan PolicyResourceModel, secrets *policyAccountMaintenanceSecrets) *proclassic.PolicyPostAccountMaintenance {
	am := &proclassic.PolicyPostAccountMaintenance{}

	if len(plan.LocalAccounts) > 0 {
		items := make([]proclassic.PolicyAccountMaintenanceAccountsAccountItem, 0, len(plan.LocalAccounts))
		for _, a := range plan.LocalAccounts {
			var password *string
			if !a.Username.IsNull() && !a.Username.IsUnknown() {
				if p, ok := secrets.accountPasswords[a.Username.ValueString()]; ok {
					password = p
				}
			}
			items = append(items, proclassic.PolicyAccountMaintenanceAccountsAccountItem{
				Action:                 helpers.OptionalStringPointer(a.Action),
				Username:               helpers.OptionalStringPointer(a.Username),
				Realname:               helpers.OptionalStringPointer(a.Realname),
				Password:               password,
				ArchiveHomeDirectory:   invertOptionalBoolPointer(a.PermanentlyDeleteHomeDirectory),
				ArchiveHomeDirectoryTo: helpers.OptionalStringPointer(a.ArchiveHomeDirectoryTo),
				Home:                   helpers.OptionalStringPointer(a.Home),
				Hint:                   helpers.OptionalStringPointer(a.Hint),
				Picture:                helpers.OptionalStringPointer(a.Picture),
				Admin:                  helpers.OptionalBoolPointer(a.Admin),
				FilevaultEnabled:       helpers.OptionalBoolPointer(a.FilevaultEnabled),
				SecureTokenAllowed:     helpers.OptionalBoolPointer(a.SecureTokenAllowed),
			})
		}
		am.Accounts = &proclassic.PolicyAccountMaintenanceAccounts{Account: &items}
	}

	if len(plan.DirectoryBindings) > 0 {
		items := make([]proclassic.IDName, 0, len(plan.DirectoryBindings))
		for _, b := range plan.DirectoryBindings {
			items = append(items, proclassic.IDName{
				ID:   helpers.StringIDPtr(b.ID),
				Name: helpers.OptionalStringPointer(b.Name),
			})
		}
		am.DirectoryBindings = &proclassic.PolicyAccountMaintenanceDirectoryBindings{Binding: &items}
	}

	if plan.ManagementAccount != nil {
		am.ManagementAccount = &proclassic.PolicyAccountMaintenanceManagementAccount{
			Action:                helpers.OptionalStringPointer(plan.ManagementAccount.Action),
			ManagedPassword:       secrets.managedPassword,
			ManagedPasswordLength: optionalInt64ToInt(plan.ManagementAccount.ManagedPasswordLength),
		}
	}

	if plan.EfiPassword != nil {
		am.OpenFirmwareEfiPassword = &proclassic.PolicyAccountMaintenanceOpenFirmwareEfiPassword{
			OfMode:     helpers.OptionalStringPointer(plan.EfiPassword.OfMode),
			OfPassword: secrets.ofPassword,
		}
	}

	if am.Accounts == nil && am.DirectoryBindings == nil && am.ManagementAccount == nil && am.OpenFirmwareEfiPassword == nil {
		return nil
	}
	return am
}

func buildPolicyReboot(m *PolicyRestartOptionsModel) *proclassic.PolicyPostReboot {
	return &proclassic.PolicyPostReboot{
		Message:                     helpers.OptionalStringPointer(m.Message),
		StartupDisk:                 helpers.OptionalStringPointer(m.StartupDisk),
		SpecifyStartup:              helpers.OptionalStringPointer(m.SpecifyStartup),
		NoUserLoggedIn:              helpers.OptionalStringPointer(m.NoUserLoggedIn),
		UserLoggedIn:                helpers.OptionalStringPointer(m.UserLoggedIn),
		MinutesUntilReboot:          optionalInt64ToInt(m.DelayMinutes),
		StartRebootTimerImmediately: helpers.OptionalBoolPointer(m.StartRebootTimerImmediately),
		FileVault2Reboot:            helpers.OptionalBoolPointer(m.FileVault2Reboot),
	}
}

func buildPolicyMaintenance(m *PolicyMaintenanceModel) *proclassic.PolicyPostMaintenance {
	return &proclassic.PolicyPostMaintenance{
		Recon:                    helpers.OptionalBoolPointer(m.UpdateInventory),
		ResetName:                helpers.OptionalBoolPointer(m.ResetComputerNames),
		InstallAllCachedPackages: helpers.OptionalBoolPointer(m.InstallCachedPackages),
		Permissions:              helpers.OptionalBoolPointer(m.FixDiskPermissions),
		Byhost:                   helpers.OptionalBoolPointer(m.FixByhostFiles),
		SystemCache:              helpers.OptionalBoolPointer(m.FlushSystemCaches),
		UserCache:                helpers.OptionalBoolPointer(m.FlushUserCaches),
		Verify:                   helpers.OptionalBoolPointer(m.VerifyStartupDisk),
	}
}

func buildPolicyFilesProcesses(m *PolicyFilesAndProcessesModel) *proclassic.PolicyPostFilesProcesses {
	return &proclassic.PolicyPostFilesProcesses{
		SearchByPath:         helpers.OptionalStringPointer(m.SearchByPath),
		DeleteFile:           helpers.OptionalBoolPointer(m.DeleteFileIfFound),
		LocateFile:           helpers.OptionalStringPointer(m.SearchByFilename),
		UpdateLocateDatabase: helpers.OptionalBoolPointer(m.UpdateLocateDatabase),
		SpotlightSearch:      helpers.OptionalStringPointer(m.SearchBySpotlight),
		SearchForProcess:     helpers.OptionalStringPointer(m.SearchForProcess),
		KillProcess:          helpers.OptionalBoolPointer(m.KillProcessIfFound),
		RunCommand:           helpers.OptionalStringPointer(m.ExecuteCommand),
	}
}

// minutesPerDay is the wire granularity for <allow_deferral_minutes>. The
// classic API stores deferral duration in minutes and enforces a
// multiple-of-1440 (one day) constraint; the provider only sends multiples
// derived from `deferral_days * minutesPerDay`.
const minutesPerDay = 1440

// buildPolicyUserInteraction maps the synthetic `deferral_type` enum onto the
// classic API's three underlying wire fields:
//
//   - `none`     → <allow_users_to_defer>false</> <allow_deferral_until_utc/> <allow_deferral_minutes>0</>
//   - `date`     → <allow_users_to_defer>true</>  <allow_deferral_until_utc>UTC</> <allow_deferral_minutes>0</>
//   - `duration` → <allow_users_to_defer>true</>  <allow_deferral_until_utc/> <allow_deferral_minutes>days*1440</>
//
// When `deferral_type` is null/unknown the deferral trio is left out of the
// payload so the server retains its prior values (matches every other
// Optional+Computed attribute in this resource). All three wire fields are
// always emitted together when a type is set — including explicit zeroes for
// the off-axis fields — so in-place transitions between Date and Duration
// clear the previously-stored value (wire-confirmed Update probe 2026-05-27).
func buildPolicyUserInteraction(m *PolicyUserInteractionModel) *proclassic.PolicyPostUserInteraction {
	out := &proclassic.PolicyPostUserInteraction{
		MessageStart:  helpers.OptionalStringPointer(m.StartMessage),
		MessageFinish: helpers.OptionalStringPointer(m.CompleteMessage),
	}
	if !helpers.IsConfiguredValue(m.DeferralType) {
		return out
	}
	emptyStr := ""
	zero := 0
	yes := true
	no := false
	switch m.DeferralType.ValueString() {
	case "none":
		out.AllowUsersToDefer = &no
		out.AllowDeferralUntilUtc = &emptyStr
		out.AllowDeferralMinutes = &zero
	case "date":
		until := emptyStr
		if helpers.IsConfiguredValue(m.DeferralUntilUtc) {
			until = m.DeferralUntilUtc.ValueString()
		}
		out.AllowUsersToDefer = &yes
		out.AllowDeferralUntilUtc = &until
		out.AllowDeferralMinutes = &zero
	case "duration":
		mins := 0
		if helpers.IsConfiguredValue(m.DeferralDays) {
			mins = int(m.DeferralDays.ValueInt64()) * minutesPerDay
		}
		out.AllowUsersToDefer = &yes
		out.AllowDeferralUntilUtc = &emptyStr
		out.AllowDeferralMinutes = &mins
	}
	return out
}

func buildPolicyDiskEncryption(m *PolicyDiskEncryptionModel) *proclassic.PolicyPostDiskEncryption {
	return &proclassic.PolicyPostDiskEncryption{
		Action:                                 helpers.OptionalStringPointer(m.Action),
		DiskEncryptionConfigurationID:          optionalInt64ToInt(m.DiskEncryptionConfigurationID),
		AuthRestart:                            helpers.OptionalBoolPointer(m.AuthRestart),
		RemediateKeyType:                       helpers.OptionalStringPointer(m.RemediateKeyType),
		RemediateDiskEncryptionConfigurationID: optionalInt64ToInt(m.RemediateDiskEncryptionConfigurationID),
	}
}
