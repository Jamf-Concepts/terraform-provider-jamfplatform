// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package macos_configuration_profile

import (
	"context"
	"slices"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenGeneral_LevelWireReadTranslated(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.OsXConfigurationProfileGeneral{
		ID:    new(42),
		Name:  new("ProfileX"),
		Level: new(levelWireReadCC),
	}, state)
	if state.ID.ValueString() != "42" {
		t.Fatalf("ID: got %v", state.ID)
	}
	if state.Level.ValueString() != levelUIComputer {
		t.Fatalf("Level: got %q want %q", state.Level.ValueString(), levelUIComputer)
	}
}

func TestFlattenGeneral_LevelUserSymmetric(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.OsXConfigurationProfileGeneral{Level: new(levelWireReadUC)}, state)
	if state.Level.ValueString() != levelUIUser {
		t.Fatalf("got %q want %q", state.Level.ValueString(), levelUIUser)
	}
}

func TestFlattenGeneral_CategoryAndSitePopulated(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.OsXConfigurationProfileGeneral{
		Category: &proclassic.CategoryObject{ID: new(64), Name: new("All Desktops")},
		Site:     &proclassic.SiteObject{ID: new(-1), Name: new("None")},
	}, state)
	if state.CategoryID.ValueString() != "64" {
		t.Fatalf("CategoryID: got %v", state.CategoryID)
	}
	if state.CategoryName.ValueString() != "All Desktops" {
		t.Fatalf("CategoryName: got %v", state.CategoryName)
	}
	if state.SiteID.ValueString() != "-1" {
		t.Fatalf("SiteID: got %v", state.SiteID)
	}
}

func TestFlattenGeneral_UUIDAndPayloadsExposed(t *testing.T) {
	t.Parallel()
	state := &GeneralModel{}
	flattenGeneral(&proclassic.OsXConfigurationProfileGeneral{
		UUID:     new("abc-uuid"),
		Payloads: new("<plist/>"),
	}, state)
	if state.UUID.ValueString() != "abc-uuid" {
		t.Fatalf("UUID: got %v", state.UUID)
	}
	if state.Payloads.ValueString() != "<plist/>" {
		t.Fatalf("Payloads: got %v", state.Payloads)
	}
}

func TestFlattenScope_NilSubBlocksProduceNullSets(t *testing.T) {
	t.Parallel()
	// Reconcile semantics: state.AllComputers stays null when the user did
	// not author it, even if the server reports a value (Optional+Computed
	// contract). Pre-populate to BoolValue(false) so reconcile substitutes
	// the server value.
	state := &ScopeModel{AllComputers: types.BoolValue(false)}
	diags := flattenScope(context.Background(), &proclassic.OsXConfigurationProfileScope{
		AllComputers: new(true),
	}, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !state.AllComputers.ValueBool() {
		t.Fatal("all_computers not propagated")
	}
	for label, s := range map[string]types.Set{
		"ComputerIDs":      state.ComputerIDs,
		"ComputerGroupIDs": state.ComputerGroupIDs,
		"BuildingIDs":      state.BuildingIDs,
		"DepartmentIDs":    state.DepartmentIDs,
		"UserIDs":          state.UserIDs,
		"UserGroupIDs":     state.UserGroupIDs,
	} {
		if !s.IsNull() {
			t.Fatalf("%s expected null when SDK sub-block absent, got %v", label, s)
		}
	}
}

// TestFlattenScope_ReconcileLeavesNullWhenStateUnconfigured pins the
// Optional+Computed contract: a user who never authored `all_computers`
// stays at null in state even when the server reports a value.
func TestFlattenScope_ReconcileLeavesNullWhenStateUnconfigured(t *testing.T) {
	t.Parallel()
	state := &ScopeModel{}
	flattenScope(context.Background(), &proclassic.OsXConfigurationProfileScope{
		AllComputers: new(true),
	}, state)
	if !state.AllComputers.IsNull() {
		t.Fatalf("expected AllComputers to stay null for unconfigured state, got %v", state.AllComputers)
	}
}

func TestFlattenScope_ComputerIDsPopulated(t *testing.T) {
	t.Parallel()
	state := &ScopeModel{}
	src := &proclassic.OsXConfigurationProfileScope{
		Computers: &proclassic.OsXConfigurationProfileScopeComputers{
			Computer: &[]proclassic.OsXConfigurationProfileScopeComputersComputerItem{
				{ID: new(28), Name: new("hostA")},
				{ID: new(31), Name: new("hostB")},
			},
		},
	}
	diags := flattenScope(context.Background(), src, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.ComputerIDs.IsNull() {
		t.Fatal("expected ComputerIDs populated")
	}
	var ids []string
	dd := state.ComputerIDs.ElementsAs(context.Background(), &ids, false)
	if dd.HasError() {
		t.Fatalf("ElementsAs: %v", dd)
	}
	if !contains(ids, "28") || !contains(ids, "31") {
		t.Fatalf("expected 28+31 in computer_ids; got %v", ids)
	}
}

func TestFlattenScopeLimitations_NetworkSegmentWithUID(t *testing.T) {
	t.Parallel()
	state := &ScopeLimitationsModel{}
	src := &proclassic.OsXConfigurationProfileScopeLimitations{
		NetworkSegments: &proclassic.OsXConfigurationProfileScopeLimitationsNetworkSegments{
			NetworkSegment: &[]proclassic.OsXConfigurationProfileScopeLimitationsNetworkSegmentsNetworkSegmentItem{
				{ID: new(5), Name: new("Lab"), Uid: new("43_5")},
			},
		},
	}
	diags := flattenScopeLimitations(context.Background(), src, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var ids []string
	if dd := state.NetworkSegmentIDs.ElementsAs(context.Background(), &ids, false); dd.HasError() {
		t.Fatalf("ElementsAs: %v", dd)
	}
	if len(ids) != 1 || ids[0] != "5" {
		t.Fatalf("expected [\"5\"], got %v", ids)
	}
}

func TestFlattenScopeLimitations_DSUserNames(t *testing.T) {
	t.Parallel()
	state := &ScopeLimitationsModel{}
	diags := flattenScopeLimitations(context.Background(), &proclassic.OsXConfigurationProfileScopeLimitations{
		Users: &proclassic.OsXConfigurationProfileScopeLimitationsUsers{
			User: &[]proclassic.IDName{{Name: new("alice")}, {Name: new("bob")}},
		},
	}, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var names []string
	if dd := state.DirectoryServiceOrLocalUserNames.ElementsAs(context.Background(), &names, false); dd.HasError() {
		t.Fatalf("ElementsAs: %v", dd)
	}
	if len(names) != 2 || !contains(names, "alice") || !contains(names, "bob") {
		t.Fatalf("expected alice+bob, got %v", names)
	}
}

func TestFlattenScopeExclusions_ComputersAndUserGroupsByName(t *testing.T) {
	t.Parallel()
	state := &ScopeExclusionsModel{}
	src := &proclassic.OsXConfigurationProfileScopeExclusions{
		Computers: &proclassic.OsXConfigurationProfileScopeExclusionsComputers{
			Computer: &[]proclassic.OsXConfigurationProfileScopeExclusionsComputersComputerItem{
				{ID: new(28)},
			},
		},
		UserGroups: &proclassic.OsXConfigurationProfileScopeExclusionsUserGroups{
			UserGroup: &[]proclassic.IDName{{Name: new("DS-Group")}},
		},
	}
	diags := flattenScopeExclusions(context.Background(), src, state)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var ids, names []string
	state.ComputerIDs.ElementsAs(context.Background(), &ids, false)
	state.DirectoryServiceUserGroupNames.ElementsAs(context.Background(), &names, false)
	if len(ids) != 1 || ids[0] != "28" {
		t.Fatalf("expected computer 28, got %v", ids)
	}
	if len(names) != 1 || names[0] != "DS-Group" {
		t.Fatalf("expected DS-Group, got %v", names)
	}
}

func TestFlattenSelfService_NotificationRecombined(t *testing.T) {
	t.Parallel()
	// Pre-configure each field so ReconcileOptional* helpers substitute the
	// server response. The wire is reliable post-SDK fix (Method-first marshal
	// order) — flatten uses the standard reconcile pattern for every field.
	state := &SelfServiceModel{
		DisplayNotifications: types.BoolValue(false),
		NotificationLocation: types.StringValue(""),
		NotificationSubject:  types.StringValue(""),
	}
	flattenSelfService(&proclassic.OsXConfigurationProfileSelfService{
		Notification: &proclassic.NotificationValue{
			Enabled: new(true),
			Method:  new(notificationLocationSelfServiceAndCenter),
		},
		NotificationSubject: new("Subj"),
		NotificationMessage: new("Body"),
	}, state)
	if !state.DisplayNotifications.ValueBool() {
		t.Fatal("DisplayNotifications: expected true (initial assignment when state was null)")
	}
	if state.NotificationLocation.ValueString() != notificationLocationSelfServiceAndCenter {
		t.Fatalf("NotificationLocation: got %q", state.NotificationLocation.ValueString())
	}
	if state.NotificationSubject.ValueString() != "Subj" {
		t.Fatalf("NotificationSubject: got %v", state.NotificationSubject)
	}
}

// TestFlattenSelfService_DisplayNotificationsReconciledFromWire — the SDK
// now emits Method-before-Enabled so the Classic API preserves the bool.
// State reconciles from the server value when the user has it configured.
func TestFlattenSelfService_DisplayNotificationsReconciledFromWire(t *testing.T) {
	t.Parallel()
	state := &SelfServiceModel{
		DisplayNotifications: types.BoolValue(false), // pre-configured to allow reconcile substitution
	}
	flattenSelfService(&proclassic.OsXConfigurationProfileSelfService{
		Notification: &proclassic.NotificationValue{
			Enabled: new(true),
		},
	}, state)
	if !state.DisplayNotifications.ValueBool() {
		t.Fatalf("expected DisplayNotifications to reconcile to true from server, got %v", state.DisplayNotifications)
	}
}

func TestFlattenSelfService_SecurityRemovalDisallowed(t *testing.T) {
	t.Parallel()
	state := &SelfServiceModel{RemovalDisallowed: types.StringValue("")}
	flattenSelfService(&proclassic.OsXConfigurationProfileSelfService{
		Security: &proclassic.OsXConfigurationProfileSelfServiceSecurity{
			RemovalDisallowed: new(removalDisallowedNever),
		},
	}, state)
	if state.RemovalDisallowed.ValueString() != removalDisallowedNever {
		t.Fatalf("RemovalDisallowed: got %q", state.RemovalDisallowed.ValueString())
	}
}

func TestFlattenSelfService_CategoriesAsList(t *testing.T) {
	t.Parallel()
	state := &SelfServiceModel{}
	flattenSelfService(&proclassic.OsXConfigurationProfileSelfService{
		SelfServiceCategories: &proclassic.OsXConfigurationProfileSelfServiceSelfServiceCategories{
			Category: &[]proclassic.OsXConfigurationProfileSelfServiceSelfServiceCategoriesCategoryItem{
				{ID: new(64), Name: new("All Desktops"), DisplayIn: new(true), FeatureIn: new(false)},
				{ID: new(46), Name: new("All Laptops"), DisplayIn: new(true), FeatureIn: new(true)},
			},
		},
	}, state)
	if len(state.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(state.Categories))
	}
	if state.Categories[0].ID.ValueString() != "64" || state.Categories[1].ID.ValueString() != "46" {
		t.Fatalf("category IDs: got %v / %v", state.Categories[0].ID, state.Categories[1].ID)
	}
	if !state.Categories[1].FeatureIn.ValueBool() {
		t.Fatal("category[1].FeatureIn expected true")
	}
}

func TestAssignResourceModel_TopLevelIDFromGeneral(t *testing.T) {
	t.Parallel()
	state := &ResourceModel{}
	diags := assignResourceModel(context.Background(), state, &proclassic.OsXConfigurationProfile{
		General: &proclassic.OsXConfigurationProfileGeneral{ID: new(99), Name: new("X")},
	})
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
	diags := assignResourceModel(context.Background(), state, &proclassic.OsXConfigurationProfile{
		ID:      new(1),
		General: &proclassic.OsXConfigurationProfileGeneral{Name: new("X")},
		Scope: &proclassic.OsXConfigurationProfileScope{
			AllComputers: new(true),
		},
		SelfService: &proclassic.OsXConfigurationProfileSelfService{
			InstallButtonText: new("Install"),
		},
	})
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if state.Scope != nil {
		t.Fatalf("expected Scope unchanged (state did not declare it); got %+v", state.Scope)
	}
	if state.SelfService != nil {
		t.Fatalf("expected SelfService unchanged; got %+v", state.SelfService)
	}
}

func TestAssignResourceModel_PopulatedSubBlocksRefreshed(t *testing.T) {
	t.Parallel()
	// Pre-populate with configured values so the reconcile helpers
	// substitute server-returned values.
	state := &ResourceModel{
		Scope:       &ScopeModel{AllComputers: types.BoolValue(false)},
		SelfService: &SelfServiceModel{InstallButtonText: types.StringValue("")},
	}
	diags := assignResourceModel(context.Background(), state, &proclassic.OsXConfigurationProfile{
		ID:      new(2),
		General: &proclassic.OsXConfigurationProfileGeneral{Name: new("X")},
		Scope:   &proclassic.OsXConfigurationProfileScope{AllComputers: new(true)},
		SelfService: &proclassic.OsXConfigurationProfileSelfService{
			InstallButtonText: new("Install"),
		},
	})
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !state.Scope.AllComputers.ValueBool() {
		t.Fatal("Scope.AllComputers not refreshed")
	}
	if state.SelfService.InstallButtonText.ValueString() != "Install" {
		t.Fatalf("SelfService.InstallButtonText: got %q", state.SelfService.InstallButtonText.ValueString())
	}
}

func contains(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}
