// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func TestAssignPolicyResourceModel_MinimalPolicy(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{
			Name:    types.StringValue("tf-acc-min"),
			Enabled: types.BoolValue(true),
		},
	}
	src := &proclassic.Policy{
		ID: new(42),
		General: &proclassic.PolicyGeneral{
			ID:      new(42),
			Name:    new("tf-acc-min"),
			Enabled: new(true),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.ID.ValueString() != "42" {
		t.Fatalf("expected id=42, got %q", state.ID.ValueString())
	}
	if state.General.Name.ValueString() != "tf-acc-min" {
		t.Fatalf("expected general.name=tf-acc-min, got %q", state.General.Name.ValueString())
	}
	if !state.General.Enabled.ValueBool() {
		t.Fatalf("expected general.enabled=true")
	}
}

func TestAssignPolicyResourceModel_FlattensScopeIDs(t *testing.T) {
	t.Parallel()
	// computer_group_ids and building_ids are declared (managed) so their
	// wire members refresh into state; undeclared sibling categories stay
	// null under granular ownership.
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		Scope: &scope.ComputerScopeModel{Targets: &scope.ComputerScopeTargetsModel{
			ComputerGroupIDs: scope.EmptyStringSet(),
			BuildingIDs:      scope.EmptyStringSet(),
		}},
	}
	src := &proclassic.Policy{
		ID:      new(7),
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		Scope: &proclassic.PolicyScope{
			ComputerGroups: &proclassic.PolicyScopeComputerGroups{
				ComputerGroup: &[]proclassic.IDName{
					{ID: new(11)},
					{ID: new(22)},
				},
			},
			Buildings: &proclassic.PolicyScopeBuildings{
				Building: &[]proclassic.IDName{{ID: new(5)}},
			},
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.Scope == nil {
		t.Fatalf("expected scope populated")
	}
	if state.Scope.Targets == nil || state.Scope.Targets.ComputerGroupIDs.IsNull() {
		t.Fatalf("expected computer_group_ids populated")
	}
	if got := len(state.Scope.Targets.ComputerGroupIDs.Elements()); got != 2 {
		t.Fatalf("expected 2 computer_group_ids, got %d", got)
	}
	if got := len(state.Scope.Targets.BuildingIDs.Elements()); got != 1 {
		t.Fatalf("expected 1 building id, got %d", got)
	}
	if !state.Scope.Targets.ComputerIDs.IsNull() {
		t.Fatalf("undeclared computer_ids must stay null, got %v", state.Scope.Targets.ComputerIDs)
	}
}

func TestFlattenPolicyScope_ManagedRefreshUnmanagedStaysNull(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Managed categories carry a non-null current value (declared in config);
	// everything else is null = unmanaged and must stay null.
	state := &scope.ComputerScopeModel{
		Targets: &scope.ComputerScopeTargetsModel{
			ComputerIDs: scope.EmptyStringSet(), // managed, refreshes from wire
		},
		Limitations: &scope.ComputerScopeLimitationsModel{
			NetworkSegmentIDs: stringSet(t, "9"), // managed, drift-refreshes
		},
		Exclusions: &scope.ComputerScopeExclusionsModel{
			DirectoryServiceOrLocalUserNames: scope.EmptyStringSet(), // managed
		},
	}
	src := &proclassic.PolicyScope{
		AllComputers: new(false),
		Computers: &proclassic.PolicyScopeComputers{
			Computer: &[]proclassic.PolicyScopeComputersComputerItem{{ID: new(11)}, {ID: new(12)}},
		},
		ComputerGroups: &proclassic.PolicyScopeComputerGroups{
			ComputerGroup: &[]proclassic.IDName{{ID: new(5)}},
		},
		Limitations: &proclassic.PolicyScopeLimitations{
			NetworkSegments: &proclassic.PolicyScopeLimitationsNetworkSegments{
				NetworkSegment: &[]proclassic.IDName{{ID: new(2)}},
			},
		},
		Exclusions: &proclassic.PolicyScopeExclusions{
			Users: &proclassic.PolicyScopeExclusionsUsers{
				User: &[]proclassic.PolicyScopeExclusionsUsersUserItem{{Name: new("alice")}},
			},
		},
	}
	if diags := flattenPolicyScope(ctx, src, state, false); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var computerIDs []string
	state.Targets.ComputerIDs.ElementsAs(ctx, &computerIDs, false)
	if len(computerIDs) != 2 {
		t.Errorf("managed computer_ids should refresh from wire: got %v", computerIDs)
	}
	if !state.Targets.ComputerGroupIDs.IsNull() {
		t.Errorf("unmanaged computer_group_ids must stay null, got %v", state.Targets.ComputerGroupIDs)
	}
	if !state.Targets.AllComputers.IsNull() {
		t.Errorf("unmanaged all_computers must stay null, got %v", state.Targets.AllComputers)
	}
	var segIDs []string
	state.Limitations.NetworkSegmentIDs.ElementsAs(ctx, &segIDs, false)
	if len(segIDs) != 1 || segIDs[0] != "2" {
		t.Errorf("managed limitations network_segment_ids should drift-refresh: got %v", segIDs)
	}
	if !state.Limitations.IbeaconIDs.IsNull() {
		t.Errorf("unmanaged limitations ibeacon_ids must stay null, got %v", state.Limitations.IbeaconIDs)
	}
	var exclUsers []string
	state.Exclusions.DirectoryServiceOrLocalUserNames.ElementsAs(ctx, &exclUsers, false)
	if len(exclUsers) != 1 || exclUsers[0] != "alice" {
		t.Errorf("managed exclusion user names should refresh: got %v", exclUsers)
	}
	if !state.Exclusions.ComputerGroupIDs.IsNull() {
		t.Errorf("unmanaged exclusion computer_group_ids must stay null, got %v", state.Exclusions.ComputerGroupIDs)
	}
}

func TestFlattenPolicyScope_HydrateAllForMergeBase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// includeUnmanaged=true hydrates every wire-present category into a zero
	// model — the shape Update uses to build the read-merge-write base.
	state := &scope.ComputerScopeModel{}
	src := &proclassic.PolicyScope{
		AllComputers: new(false),
		ComputerGroups: &proclassic.PolicyScopeComputerGroups{
			ComputerGroup: &[]proclassic.IDName{{ID: new(5)}},
		},
		Exclusions: &proclassic.PolicyScopeExclusions{
			Users: &proclassic.PolicyScopeExclusionsUsers{
				User: &[]proclassic.PolicyScopeExclusionsUsersUserItem{{Name: new("alice")}},
			},
		},
	}
	if diags := flattenPolicyScope(ctx, src, state, true); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.Targets == nil || state.Targets.AllComputers.IsNull() || state.Targets.AllComputers.ValueBool() {
		t.Fatalf("expected all_computers hydrated false, got %+v", state.Targets)
	}
	var groupIDs []string
	state.Targets.ComputerGroupIDs.ElementsAs(ctx, &groupIDs, false)
	if len(groupIDs) != 1 || groupIDs[0] != "5" {
		t.Errorf("expected computer_group_ids hydrated, got %v", groupIDs)
	}
	if state.Exclusions == nil {
		t.Fatal("expected exclusions allocated")
	}
	var exclUsers []string
	state.Exclusions.DirectoryServiceOrLocalUserNames.ElementsAs(ctx, &exclUsers, false)
	if len(exclUsers) != 1 || exclUsers[0] != "alice" {
		t.Errorf("expected exclusion user names hydrated, got %v", exclUsers)
	}
	if state.Limitations != nil {
		t.Fatalf("expected limitations nil when wire-absent; got %+v", state.Limitations)
	}
}

func TestAssignPolicyResourceModel_RoundTripNotification(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		SelfService: &PolicySelfServiceModel{
			DisplayNotifications: types.BoolValue(true),
			NotificationLocation: types.StringValue("Self Service"),
		},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		SelfService: &proclassic.PolicySelfService{
			Notification:     &proclassic.NotificationValue{Enabled: new(true)},
			NotificationType: new("Self Service"),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !state.SelfService.DisplayNotifications.ValueBool() {
		t.Fatalf("expected display_notifications=true")
	}
	if state.SelfService.NotificationLocation.ValueString() != "Self Service" {
		t.Fatalf("expected notification_location=Self Service, got %q", state.SelfService.NotificationLocation.ValueString())
	}
}

func TestAssignPolicyResourceModel_PackageConfigurationDistributionPoint(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General:  &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		Packages: &PolicyPackagesModel{},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		PackageConfiguration: &proclassic.PolicyPackageConfiguration{
			DistributionPoint: new("Dummy DP"),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.Packages.DistributionPoint.ValueString() != "Dummy DP" {
		t.Fatalf("expected distribution_point=Dummy DP, got %q", state.Packages.DistributionPoint.ValueString())
	}
	if state.Packages.Packages != nil {
		t.Fatalf("expected packages nil when server returned none, got %+v", state.Packages.Packages)
	}
}

// TestAssignPolicyResourceModel_PackageConfigurationReportsDrift pins the
// wire-authoritative read on a managed attribute: an echoed value that differs
// from what state holds must land in state, so `terraform plan` reports the
// change. The classic /policies GET echoes
// <package_configuration><distribution_point> faithfully (Jamf Pro 11.31.1,
// wire-probed 2026-09-06). This asserts the opposite of what it did before
// issue #387 — the sticky read it used to pin is what made a Jamf Pro UI edit
// to a Terraform-managed attribute invisible forever.
func TestAssignPolicyResourceModel_PackageConfigurationReportsDrift(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		Packages: &PolicyPackagesModel{
			DistributionPoint: types.StringValue("Configured DP"),
		},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		PackageConfiguration: &proclassic.PolicyPackageConfiguration{
			DistributionPoint: new("Server DP"),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := state.Packages.DistributionPoint.ValueString(); got != "Server DP" {
		t.Fatalf("the wire value must win so drift is reported, got %q", got)
	}
}

// TestAssignPolicyResourceModel_StickyFieldsIgnoreDrift pins the other half of
// the #387 split: the handful of fields Jamf Pro does not echo, does not
// persist, or normalises keep the sticky read, so a configured value is not
// nulled or rewritten by a refresh. The evidence for each is in the doc comment
// of the flattener that holds it.
//
// The Policy passed in is the shape a real GET returns against the state
// declared here: empty no_execute_*, a server-forced any_ip_address, an
// override target_drive mirroring general.target_drive, a dropped
// force_afp_smb, an absent <notification> family and an empty
// leave_existing_default.
func TestAssignPolicyResourceModel_StickyFieldsIgnoreDrift(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{
			Name: types.StringValue("tf-acc"),
			DateTimeLimitations: &PolicyGeneralDateTimeLimitationsModel{
				NoExecuteStart: types.StringValue("1:00 AM"),
				NoExecuteEnd:   types.StringValue("2:00 AM"),
			},
			NetworkLimitations: &PolicyGeneralNetworkLimitationsModel{
				AnyIPAddress:             types.BoolValue(false),
				MinimumNetworkConnection: types.StringValue("No Minimum"),
			},
			OverrideDefaultSettings: &PolicyGeneralOverrideDefaultsModel{
				TargetDrive: types.StringValue("/Volumes/od"),
				ForceAfpSmb: types.BoolValue(true),
			},
		},
		SelfService: &PolicySelfServiceModel{
			DisplayNotifications: types.BoolValue(true),
			NotificationSubject:  types.StringValue("configured subject"),
		},
		Printers: &PolicyPrintersModel{LeaveExistingDefault: types.BoolValue(true)},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{
			Name: new("tf-acc"),
			DateTimeLimitations: &proclassic.PolicyGeneralDateTimeLimitations{
				NoExecuteStart: new(""),
				NoExecuteEnd:   new(""),
			},
			NetworkLimitations: &proclassic.PolicyGeneralNetworkLimitations{
				AnyIPAddress:             new(true),
				MinimumNetworkConnection: new("Ethernet"),
			},
			OverrideDefaultSettings: &proclassic.PolicyGeneralOverrideDefaultSettings{
				TargetDrive: new("/Volumes/mirrored"),
				ForceAfpSmb: new(false),
			},
		},
		SelfService: &proclassic.PolicySelfService{},
		Printers:    &proclassic.PolicyPrinters{},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	for _, tc := range []struct {
		name, want, got string
	}{
		{"general.date_time_limitations.no_execute_start", "1:00 AM", state.General.DateTimeLimitations.NoExecuteStart.ValueString()},
		{"general.date_time_limitations.no_execute_end", "2:00 AM", state.General.DateTimeLimitations.NoExecuteEnd.ValueString()},
		{"general.network_limitations.minimum_network_connection", "No Minimum", state.General.NetworkLimitations.MinimumNetworkConnection.ValueString()},
		{"general.override_default_settings.target_drive", "/Volumes/od", state.General.OverrideDefaultSettings.TargetDrive.ValueString()},
		{"self_service.notification_subject", "configured subject", state.SelfService.NotificationSubject.ValueString()},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: sticky read must keep %q, got %q", tc.name, tc.want, tc.got)
		}
	}
	if state.General.NetworkLimitations.AnyIPAddress.ValueBool() {
		t.Error("general.network_limitations.any_ip_address: sticky read must keep false")
	}
	if !state.General.OverrideDefaultSettings.ForceAfpSmb.ValueBool() {
		t.Error("general.override_default_settings.force_afp_smb: sticky read must keep true")
	}
	if !state.SelfService.DisplayNotifications.ValueBool() {
		t.Error("self_service.display_notifications: sticky read must keep true")
	}
	if !state.Printers.LeaveExistingDefault.ValueBool() {
		t.Error("printers.leave_existing_default: sticky read must keep true")
	}
}

// TestAssignPolicyResourceModel_IncludeUnmanagedHydratesFromScratch pins the
// config-generation contract: with includeUnmanaged set and an empty starting
// model, every wire-present section is allocated and hydrated from the server.
func TestAssignPolicyResourceModel_IncludeUnmanagedHydratesFromScratch(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{}
	src := &proclassic.Policy{
		ID:      new(9),
		General: &proclassic.PolicyGeneral{Name: new("tf-acc"), Enabled: new(true)},
		Scope: &proclassic.PolicyScope{
			AllComputers: new(true),
			ComputerGroups: &proclassic.PolicyScopeComputerGroups{
				ComputerGroup: &[]proclassic.IDName{{ID: new(11)}, {ID: new(22)}},
			},
			Exclusions: &proclassic.PolicyScopeExclusions{},
		},
		Scripts: &proclassic.PolicyScripts{
			Script: &[]proclassic.PolicyScriptsScriptItem{{ID: new(3), Priority: new("After")}},
		},
		PackageConfiguration: &proclassic.PolicyPackageConfiguration{
			Packages: &proclassic.PolicyPackageConfigurationPackages{
				Package: &[]proclassic.PolicyPackageConfigurationPackagesPackageItem{{ID: new(4)}},
			},
		},
		SelfService: &proclassic.PolicySelfService{UseForSelfService: new(true)},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src, true)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.Scope == nil || state.Scope.Targets == nil {
		t.Fatalf("expected scope.targets hydrated from scratch; got %+v", state.Scope)
	}
	if !state.Scope.Targets.AllComputers.ValueBool() {
		t.Fatal("expected all_computers hydrated true")
	}
	if got := len(state.Scope.Targets.ComputerGroupIDs.Elements()); got != 2 {
		t.Fatalf("expected 2 computer_group_ids, got %d", got)
	}
	if state.Scope.Exclusions == nil {
		t.Fatal("expected exclusions allocated when wire-present")
	}
	if state.Scope.Limitations != nil {
		t.Fatalf("expected limitations nil when wire-absent; got %+v", state.Scope.Limitations)
	}
	if state.Scripts == nil || len(state.Scripts.Scripts) != 1 {
		t.Fatalf("expected scripts hydrated; got %+v", state.Scripts)
	}
	if state.Packages == nil || len(state.Packages.Packages) != 1 {
		t.Fatalf("expected packages hydrated; got %+v", state.Packages)
	}
	if state.SelfService == nil || !state.SelfService.UseForSelfService.ValueBool() {
		t.Fatalf("expected self_service hydrated; got %+v", state.SelfService)
	}
}
