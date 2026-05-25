// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	set, diags := types.SetValue(types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("failed to build test set: %v", diags)
	}
	return set
}

func TestBuildPolicyInput_MinimalPolicy(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{
			Name:    types.StringValue("tf-acc-minimal"),
			Enabled: types.BoolValue(true),
		},
	}
	got, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got == nil || got.General == nil {
		t.Fatalf("expected General to be populated")
	}
	if got.General.Name == nil || *got.General.Name != "tf-acc-minimal" {
		t.Fatalf("expected name=tf-acc-minimal, got %+v", got.General.Name)
	}
	if got.General.Enabled == nil || !*got.General.Enabled {
		t.Fatalf("expected enabled=true")
	}
	if got.Scope != nil {
		t.Fatalf("expected Scope to be nil when not in plan, got %+v", got.Scope)
	}
}

func TestBuildPolicyInput_ScopeWithComputerGroupAndBuilding(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc-scoped")},
		Scope: &PolicyScopeModel{
			ComputerGroupIDs: stringSet(t, "11", "22"),
			BuildingIDs:      stringSet(t, "7"),
		},
	}
	got, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.Scope == nil {
		t.Fatalf("expected Scope to be populated")
	}
	if got.Scope.ComputerGroups == nil || got.Scope.ComputerGroups.ComputerGroup == nil {
		t.Fatalf("expected ComputerGroups.ComputerGroup populated")
	}
	if n := len(*got.Scope.ComputerGroups.ComputerGroup); n != 2 {
		t.Fatalf("expected 2 computer groups, got %d", n)
	}
	if got.Scope.Buildings == nil || got.Scope.Buildings.Building == nil {
		t.Fatalf("expected Buildings.Building populated")
	}
	if n := len(*got.Scope.Buildings.Building); n != 1 {
		t.Fatalf("expected 1 building, got %d", n)
	}
}

func TestBuildPolicyInput_ScopeOmissionSemantics(t *testing.T) {
	t.Parallel()
	// An empty scope block in HCL must collapse all the way up to a nil
	// PolicyPostScope so the wire request omits the <scope> element entirely
	// rather than emitting <scope></scope>.
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc-empty-scope")},
		Scope:   &PolicyScopeModel{},
	}
	got, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.Scope != nil {
		t.Fatalf("expected Scope to collapse to nil for empty scope block, got %+v", got.Scope)
	}
}

func TestBuildPolicyInput_AllComputersFlag(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc-universal")},
		Scope: &PolicyScopeModel{
			AllComputers: types.BoolValue(true),
		},
	}
	got, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.Scope == nil || got.Scope.AllComputers == nil || !*got.Scope.AllComputers {
		t.Fatalf("expected AllComputers=true")
	}
}

func TestBuildPolicyInput_SelfServiceNotificationSplit(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc-ss")},
		SelfService: &PolicySelfServiceModel{
			UseForSelfService:    types.BoolValue(true),
			DisplayNotifications: types.BoolValue(true),
			NotificationLocation: types.StringValue("Self Service"),
			NotificationSubject:  types.StringValue("hello"),
		},
	}
	got, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.SelfService == nil || got.SelfService.Notification == nil {
		t.Fatalf("expected SelfService.Notification populated")
	}
	if got.SelfService.Notification.Enabled == nil || !*got.SelfService.Notification.Enabled {
		t.Fatalf("expected notification.Enabled=true")
	}
	// Method must NOT travel via NotificationValue.Method (that would emit a
	// second <notification> element which the server interprets as
	// overwriting the bool). It travels via the sibling NotificationType
	// (wire) field, which the provider models as NotificationLocation.
	if got.SelfService.Notification.Method != nil {
		t.Fatalf("expected notification.Method=nil; method travels via NotificationType, got %+v", got.SelfService.Notification.Method)
	}
	if got.SelfService.NotificationType == nil || *got.SelfService.NotificationType != "Self Service" {
		t.Fatalf("expected NotificationType=Self Service, got %+v", got.SelfService.NotificationType)
	}
	if got.SelfService.NotificationSubject == nil || *got.SelfService.NotificationSubject != "hello" {
		t.Fatalf("expected notification_subject=hello")
	}
}

func TestBuildPolicyInput_PackagesAndScripts(t *testing.T) {
	t.Parallel()
	plan := PolicyResourceModel{
		General: &PolicyGeneralModel{Name: types.StringValue("tf-acc-pkg")},
		PackageConfiguration: &PolicyPackageConfigurationModel{
			DistributionPoint: types.StringValue("Dummy DP"),
			Packages: []PolicyPackageItemModel{
				{ID: types.StringValue("100"), Action: types.StringValue("Install")},
			},
		},
		Scripts: &PolicyScriptsModel{
			Scripts: []PolicyScriptItemModel{
				{ID: types.StringValue("9"), Priority: types.StringValue("Before")},
			},
		},
	}
	got, diags := buildPolicyInput(context.Background(), plan, noSecrets())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.PackageConfiguration == nil || got.PackageConfiguration.Packages == nil {
		t.Fatalf("expected packages populated")
	}
	if got.PackageConfiguration.DistributionPoint == nil || *got.PackageConfiguration.DistributionPoint != "Dummy DP" {
		t.Fatalf("expected distribution_point=Dummy DP, got %v", got.PackageConfiguration.DistributionPoint)
	}
	if got.Scripts == nil || got.Scripts.Script == nil {
		t.Fatalf("expected scripts populated")
	}
}

func TestBuildPolicyPackageConfiguration_DistributionPointOnly(t *testing.T) {
	t.Parallel()
	m := &PolicyPackageConfigurationModel{
		DistributionPoint: types.StringValue("Dummy DP"),
	}
	got := buildPolicyPackageConfiguration(m)
	if got == nil {
		t.Fatalf("expected non-nil package_configuration when distribution_point is set")
	}
	if got.Packages != nil {
		t.Fatalf("expected packages nil when no package items provided, got %+v", got.Packages)
	}
	if got.DistributionPoint == nil || *got.DistributionPoint != "Dummy DP" {
		t.Fatalf("expected distribution_point=Dummy DP, got %v", got.DistributionPoint)
	}
}

func TestBuildPolicyPackageConfiguration_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	m := &PolicyPackageConfigurationModel{}
	if got := buildPolicyPackageConfiguration(m); got != nil {
		t.Fatalf("expected nil for empty package_configuration, got %+v", got)
	}
}
