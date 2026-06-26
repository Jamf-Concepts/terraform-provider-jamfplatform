// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_configuration_profile

import (
	"context"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/scope"
)

func TestFlattenGeneral_LevelWireReadTranslated(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.MobileDeviceConfigurationProfileGeneral{
		ID:    new(42),
		Name:  new("MobileX"),
		Level: new(levelWireReadDevice),
	}, state)
	if state.ID.ValueString() != "42" {
		t.Fatalf("ID: got %v", state.ID)
	}
	if state.Level.ValueString() != levelUIDevice {
		t.Fatalf("Level: got %q want %q", state.Level.ValueString(), levelUIDevice)
	}
}

func TestFlattenGeneral_LevelUserSymmetric(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.MobileDeviceConfigurationProfileGeneral{Level: new(levelWireReadUser)}, state)
	if state.Level.ValueString() != levelUIUser {
		t.Fatalf("got %q want %q", state.Level.ValueString(), levelUIUser)
	}
}

func TestFlattenGeneral_RedeployDaysBeforeCertificateExpires(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.MobileDeviceConfigurationProfileGeneral{
		RedeployDaysBeforeCertificateExpires: new(7),
	}, state)
	if state.RedeployDaysBeforeCertificateExpires.ValueInt64() != 7 {
		t.Fatalf("RedeployDays: got %v", state.RedeployDaysBeforeCertificateExpires)
	}
}

func TestFlattenGeneral_CategoryAndSitePopulated(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.MobileDeviceConfigurationProfileGeneral{
		Category: &proclassic.CategoryObject{ID: new(58), Name: new("Applications")},
		Site:     &proclassic.SiteObject{ID: new(-1), Name: new("None")},
	}, state)
	if state.CategoryID.ValueString() != "58" {
		t.Fatalf("CategoryID: got %v", state.CategoryID)
	}
	if state.CategoryName.ValueString() != "Applications" {
		t.Fatalf("CategoryName: got %v", state.CategoryName)
	}
	if state.SiteID.ValueString() != "-1" {
		t.Fatalf("SiteID: got %v", state.SiteID)
	}
}

func TestFlattenGeneral_DeploymentMethodMappedToDistributionMethod(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.MobileDeviceConfigurationProfileGeneral{
		DeploymentMethod: new(distributionMethodInstallAutomatically),
	}, state)
	if state.DistributionMethod.ValueString() != distributionMethodInstallAutomatically {
		t.Fatalf("DistributionMethod: got %v", state.DistributionMethod)
	}
}

func TestFlattenGeneral_UUIDAndPayloadsExposed(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.MobileDeviceConfigurationProfileGeneral{
		UUID:     new("test-uuid"),
		Payloads: new(proclassic.PayloadsXMLText("<plist/>")),
	}, state)
	if state.UUID.ValueString() != "test-uuid" {
		t.Fatalf("UUID: got %v", state.UUID)
	}
	if got := state.Payloads.ValueString(); !strings.Contains(got, "<plist") {
		t.Fatalf("Payloads: got %v", got)
	}
}

func TestFlattenScope_NilSubBlocksProduceEmptySets(t *testing.T) {
	t.Parallel()
	state := &scope.MobileScopeModel{Targets: &scope.MobileScopeTargetsModel{AllMobileDevices: types.BoolValue(false)}}
	diags := flattenScope(context.Background(), &proclassic.MobileDeviceConfigurationProfileScope{
		AllMobileDevices: new(true),
	}, state, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !state.Targets.AllMobileDevices.ValueBool() {
		t.Fatal("all_mobile_devices not propagated")
	}
	for label, s := range map[string]types.Set{
		"MobileDeviceIDs":      state.Targets.MobileDeviceIDs,
		"MobileDeviceGroupIDs": state.Targets.MobileDeviceGroupIDs,
		"BuildingIDs":          state.Targets.BuildingIDs,
		"DepartmentIDs":        state.Targets.DepartmentIDs,
		"UserIDs":              state.Targets.UserIDs,
		"UserGroupIDs":         state.Targets.UserGroupIDs,
	} {
		if s.IsNull() || len(s.Elements()) != 0 {
			t.Fatalf("%s expected empty set when SDK sub-block absent, got %v", label, s)
		}
	}
}

func TestFlattenScope_ReconcileLeavesNullWhenStateUnconfigured(t *testing.T) {
	t.Parallel()
	state := &scope.MobileScopeModel{}
	flattenScope(context.Background(), &proclassic.MobileDeviceConfigurationProfileScope{
		AllMobileDevices: new(true),
	}, state, false)
	if state.Targets != nil {
		t.Fatalf("expected Targets to stay nil for unconfigured state, got %v", state.Targets)
	}
}

func TestFlattenScope_MobileDeviceIDsPopulated(t *testing.T) {
	t.Parallel()
	state := &scope.MobileScopeModel{Targets: &scope.MobileScopeTargetsModel{}}
	diags := flattenScope(context.Background(), &proclassic.MobileDeviceConfigurationProfileScope{
		MobileDevices: &proclassic.MobileDeviceConfigurationProfileScopeMobileDevices{
			MobileDevice: &[]proclassic.MobileDeviceConfigurationProfileScopeMobileDevicesMobileDeviceItem{
				{ID: new(12)},
				{ID: new(31)},
			},
		},
	}, state, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.Targets.MobileDeviceIDs.IsNull() {
		t.Fatal("expected MobileDeviceIDs populated")
	}
	var ids []string
	if dd := state.Targets.MobileDeviceIDs.ElementsAs(context.Background(), &ids, false); dd.HasError() {
		t.Fatalf("ElementsAs: %v", dd)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d: %v", len(ids), ids)
	}
}

func TestFlattenScopeLimitations_DSUserNamesAndNetworkSegments(t *testing.T) {
	t.Parallel()
	state := &scope.MobileScopeLimitationsModel{}
	diags := flattenScopeLimitations(context.Background(), &proclassic.MobileDeviceConfigurationProfileScopeLimitations{
		Users: &proclassic.MobileDeviceConfigurationProfileScopeLimitationsUsers{
			User: &[]proclassic.IDName{{Name: new("alice")}, {Name: new("bob")}},
		},
		NetworkSegments: &proclassic.MobileDeviceConfigurationProfileScopeLimitationsNetworkSegments{
			NetworkSegment: &[]proclassic.IDName{{ID: new(5)}},
		},
	}, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var names []string
	if dd := state.DirectoryServiceOrLocalUserNames.ElementsAs(context.Background(), &names, false); dd.HasError() {
		t.Fatalf("ElementsAs users: %v", dd)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 user names, got %v", names)
	}
	var segIDs []string
	if dd := state.NetworkSegmentIDs.ElementsAs(context.Background(), &segIDs, false); dd.HasError() {
		t.Fatalf("ElementsAs segs: %v", dd)
	}
	if len(segIDs) != 1 || segIDs[0] != "5" {
		t.Fatalf("expected [\"5\"], got %v", segIDs)
	}
}

func TestFlattenScopeExclusions_MobileDevicesAndUserGroups(t *testing.T) {
	t.Parallel()
	state := &scope.MobileScopeExclusionsModel{}
	diags := flattenScopeExclusions(context.Background(), &proclassic.MobileDeviceConfigurationProfileScopeExclusions{
		MobileDevices: &proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDevices{
			MobileDevice: &[]proclassic.MobileDeviceConfigurationProfileScopeExclusionsMobileDevicesMobileDeviceItem{
				{ID: new(12)},
			},
		},
		UserGroups: &proclassic.MobileDeviceConfigurationProfileScopeExclusionsUserGroups{
			UserGroup: &[]proclassic.IDName{{Name: new("DS-Group")}},
		},
	}, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var ids []string
	state.MobileDeviceIDs.ElementsAs(context.Background(), &ids, false)
	if len(ids) != 1 || ids[0] != "12" {
		t.Fatalf("expected mobile device 12, got %v", ids)
	}
	var names []string
	state.DirectoryServiceUserGroupNames.ElementsAs(context.Background(), &names, false)
	if len(names) != 1 || names[0] != "DS-Group" {
		t.Fatalf("expected DS-Group, got %v", names)
	}
}

func TestFlattenSelfService_SecurityRemovalDisallowed(t *testing.T) {
	t.Parallel()
	state := &SelfServiceModel{RemovalDisallowed: types.StringValue("")}
	flattenSelfService(&proclassic.MobileDeviceConfigurationProfileSelfService{
		Security: &proclassic.MobileDeviceConfigurationProfileSelfServiceSecurity{
			RemovalDisallowed: new(removalDisallowedNever),
		},
	}, state)
	if state.RemovalDisallowed.ValueString() != removalDisallowedNever {
		t.Fatalf("RemovalDisallowed: got %q", state.RemovalDisallowed.ValueString())
	}
}

func TestFlattenSelfService_AuthorizationPasswordReconciled(t *testing.T) {
	t.Parallel()
	state := &SelfServiceModel{AuthorizationPassword: types.StringValue("")}
	flattenSelfService(&proclassic.MobileDeviceConfigurationProfileSelfService{
		Security: &proclassic.MobileDeviceConfigurationProfileSelfServiceSecurity{
			RemovalDisallowed: new(removalDisallowedWithAuthorization),
			Password:          new("s3cr3t"),
		},
	}, state)
	if state.AuthorizationPassword.ValueString() != "s3cr3t" {
		t.Fatalf("AuthorizationPassword: got %q", state.AuthorizationPassword.ValueString())
	}
}

func TestFlattenSelfService_CategoriesAsList(t *testing.T) {
	t.Parallel()
	state := &SelfServiceModel{}
	flattenSelfService(&proclassic.MobileDeviceConfigurationProfileSelfService{
		SelfServiceCategories: &proclassic.MobileDeviceConfigurationProfileSelfServiceSelfServiceCategories{
			Category: &[]proclassic.Category{
				{ID: new(58), Name: new("Applications")},
				{ID: new(44), Name: new("Auto-Update")},
			},
		},
	}, state)
	if len(state.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(state.Categories))
	}
	if state.Categories[0].ID.ValueString() != "58" || state.Categories[1].ID.ValueString() != "44" {
		t.Fatalf("category IDs: got %v / %v", state.Categories[0].ID, state.Categories[1].ID)
	}
}

func TestAssignResourceModel_TopLevelIDFromGeneral(t *testing.T) {
	t.Parallel()
	state := &ResourceModel{}
	diags := assignResourceModel(context.Background(), state, &proclassic.MobileDeviceConfigurationProfile{
		General: &proclassic.MobileDeviceConfigurationProfileGeneral{ID: new(99), Name: new("X")},
	}, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.ID.ValueString() != "99" {
		t.Fatalf("ID: got %v", state.ID)
	}
}

func TestAssignResourceModel_OptionalSubBlocksSkippedWhenStateNil(t *testing.T) {
	t.Parallel()
	state := &ResourceModel{}
	diags := assignResourceModel(context.Background(), state, &proclassic.MobileDeviceConfigurationProfile{
		ID:      new(1),
		General: &proclassic.MobileDeviceConfigurationProfileGeneral{Name: new("X")},
		Scope: &proclassic.MobileDeviceConfigurationProfileScope{
			AllMobileDevices: new(true),
		},
		SelfService: &proclassic.MobileDeviceConfigurationProfileSelfService{
			SelfServiceDescription: new("desc"),
		},
	}, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.Scope != nil {
		t.Fatalf("expected Scope unchanged; got %+v", state.Scope)
	}
	if state.SelfService != nil {
		t.Fatalf("expected SelfService unchanged; got %+v", state.SelfService)
	}
}

func TestAssignResourceModel_PopulatedSubBlocksRefreshed(t *testing.T) {
	t.Parallel()
	state := &ResourceModel{
		Scope:       &scope.MobileScopeModel{Targets: &scope.MobileScopeTargetsModel{AllMobileDevices: types.BoolValue(false)}},
		SelfService: &SelfServiceModel{RemovalDisallowed: types.StringValue("")},
	}
	diags := assignResourceModel(context.Background(), state, &proclassic.MobileDeviceConfigurationProfile{
		ID:      new(2),
		General: &proclassic.MobileDeviceConfigurationProfileGeneral{Name: new("X")},
		Scope:   &proclassic.MobileDeviceConfigurationProfileScope{AllMobileDevices: new(true)},
		SelfService: &proclassic.MobileDeviceConfigurationProfileSelfService{
			Security: &proclassic.MobileDeviceConfigurationProfileSelfServiceSecurity{
				RemovalDisallowed: new(removalDisallowedNever),
			},
		},
	}, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !state.Scope.Targets.AllMobileDevices.ValueBool() {
		t.Fatal("Scope.Targets.AllMobileDevices not refreshed")
	}
	if state.SelfService.RemovalDisallowed.ValueString() != removalDisallowedNever {
		t.Fatalf("SelfService.RemovalDisallowed: got %q", state.SelfService.RemovalDisallowed.ValueString())
	}
}

// TestAssignResourceModel_IncludeUnmanagedHydratesFromScratch pins the
// config-generation contract: with includeUnmanaged set and an empty starting
// model, every wire-present optional section is allocated and hydrated.
func TestAssignResourceModel_IncludeUnmanagedHydratesFromScratch(t *testing.T) {
	t.Parallel()
	state := &ResourceModel{}
	diags := assignResourceModel(context.Background(), state, &proclassic.MobileDeviceConfigurationProfile{
		ID:      new(7),
		General: &proclassic.MobileDeviceConfigurationProfileGeneral{Name: new("X")},
		Scope: &proclassic.MobileDeviceConfigurationProfileScope{
			AllMobileDevices: new(true),
			Exclusions:       &proclassic.MobileDeviceConfigurationProfileScopeExclusions{},
		},
		SelfService: &proclassic.MobileDeviceConfigurationProfileSelfService{
			SelfServiceDescription: new("desc"),
		},
	}, true)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.Scope == nil || state.Scope.Targets == nil {
		t.Fatalf("expected Scope.Targets hydrated from scratch; got %+v", state.Scope)
	}
	if !state.Scope.Targets.AllMobileDevices.ValueBool() {
		t.Fatal("expected all_mobile_devices hydrated true")
	}
	if state.Scope.Exclusions == nil {
		t.Fatal("expected Exclusions allocated when wire-present")
	}
	if state.Scope.Limitations != nil {
		t.Fatalf("expected Limitations to stay nil when wire-absent; got %+v", state.Scope.Limitations)
	}
	if state.SelfService == nil {
		t.Fatalf("expected SelfService hydrated; got nil")
	}
}
