// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	diags := assignPolicyResourceModel(context.Background(), state, src)
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
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		Scope:   &PolicyScopeModel{},
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
	diags := assignPolicyResourceModel(context.Background(), state, src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.Scope == nil {
		t.Fatalf("expected scope populated")
	}
	if state.Scope.ComputerGroupIDs.IsNull() {
		t.Fatalf("expected computer_group_ids populated")
	}
	if got := len(state.Scope.ComputerGroupIDs.Elements()); got != 2 {
		t.Fatalf("expected 2 computer_group_ids, got %d", got)
	}
	if got := len(state.Scope.BuildingIDs.Elements()); got != 1 {
		t.Fatalf("expected 1 building id, got %d", got)
	}
}

func TestAssignPolicyResourceModel_RoundTripNotification(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		SelfService: &PolicySelfServiceModel{
			NotificationEnabled: types.BoolValue(true),
			NotificationType:    types.StringValue("Self Service"),
		},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		SelfService: &proclassic.PolicySelfService{
			Notification:     &proclassic.NotificationValue{Enabled: new(true)},
			NotificationType: new("Self Service"),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !state.SelfService.NotificationEnabled.ValueBool() {
		t.Fatalf("expected notification_enabled=true")
	}
	if state.SelfService.NotificationType.ValueString() != "Self Service" {
		t.Fatalf("expected notification_type=Self Service, got %q", state.SelfService.NotificationType.ValueString())
	}
}

func TestAssignPolicyResourceModel_PackageConfigurationDistributionPoint(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General:              &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		PackageConfiguration: &PolicyPackageConfigurationModel{},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		PackageConfiguration: &proclassic.PolicyPackageConfiguration{
			DistributionPoint: new("Dummy DP"),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.PackageConfiguration.DistributionPoint.ValueString() != "Dummy DP" {
		t.Fatalf("expected distribution_point=Dummy DP, got %q", state.PackageConfiguration.DistributionPoint.ValueString())
	}
	if state.PackageConfiguration.Packages != nil {
		t.Fatalf("expected packages nil when server returned none, got %+v", state.PackageConfiguration.Packages)
	}
}

func TestAssignPolicyResourceModel_PackageConfigurationConfiguredWins(t *testing.T) {
	t.Parallel()
	state := &PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc")},
		PackageConfiguration: &PolicyPackageConfigurationModel{
			DistributionPoint: types.StringValue("Configured DP"),
		},
	}
	src := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{Name: new("tf-acc")},
		PackageConfiguration: &proclassic.PolicyPackageConfiguration{
			DistributionPoint: new("Server DP"),
		},
	}
	diags := assignPolicyResourceModel(context.Background(), state, src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := state.PackageConfiguration.DistributionPoint.ValueString(); got != "Configured DP" {
		t.Fatalf("preferCurrentStringPointer should keep configured value, got %q", got)
	}
}
