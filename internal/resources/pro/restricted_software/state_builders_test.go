// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

import (
	"context"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFlattenGeneral_RenameMapping verifies wire fields land on the UI-aligned
// attributes and that the configured-value-wins echo guard holds.
func TestFlattenGeneral_RenameMapping(t *testing.T) {
	g := &proclassic.RestrictedSoftwareGeneral{
		ID:                    new(42),
		Name:                  new("Block Chess"),
		ProcessName:           new("Chess.app"),
		MatchExactProcessName: new(true),
		SendNotification:      new(false),
		KillProcess:           new(true),
		DeleteExecutable:      new(false),
		DisplayMessage:        new("blocked"),
		Site:                  &proclassic.SiteObject{ID: new(-1), Name: new("None")},
	}
	// Fresh state (all null) → adopt server values.
	state := &RestrictedSoftwareGeneralModel{}
	flattenGeneral(g, state)

	if state.ID.ValueString() != "42" {
		t.Errorf("id: got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Block Chess" || state.ProcessName.ValueString() != "Chess.app" {
		t.Errorf("name/process_name not flattened")
	}
	if !state.RestrictExactProcessName.ValueBool() {
		t.Errorf("restrict_exact_process_name must reflect match_exact_process_name=true")
	}
	if state.SendEmailNotificationOnViolation.ValueBool() {
		t.Errorf("send_email_notification_on_violation must reflect send_notification=false")
	}
	if !state.KillProcess.ValueBool() {
		t.Errorf("kill_process not flattened")
	}
	if state.DeleteApplication.ValueBool() {
		t.Errorf("delete_application must reflect delete_executable=false")
	}
	// Sentinel site (id -1): derived name nulls, not the flaky server echo.
	if state.SiteID.ValueString() != "-1" || !state.SiteName.IsNull() {
		t.Errorf("site not flattened: id=%q name=%q (name should be null on the sentinel)", state.SiteID.ValueString(), state.SiteName.ValueString())
	}
}

// TestFlattenGeneral_PrefersConfigured confirms a caller-set value is preserved
// over a divergent server echo (the documented ProClassic tradeoff).
func TestFlattenGeneral_PrefersConfigured(t *testing.T) {
	g := &proclassic.RestrictedSoftwareGeneral{
		Name:           new("x"),
		ProcessName:    new("y"),
		DisplayMessage: new("server-message"),
	}
	state := &RestrictedSoftwareGeneralModel{
		DisplayMessage: types.StringValue("user-message"),
	}
	flattenGeneral(g, state)
	if state.DisplayMessage.ValueString() != "user-message" {
		t.Errorf("configured display_message must win, got %q", state.DisplayMessage.ValueString())
	}
}

// TestAssignRestrictedSoftwareResourceModel_IncludeUnmanagedHydratesFromScratch
// pins the config-generation contract: with includeUnmanaged set and an empty
// starting model, the wire-present scope is allocated and hydrated from the
// server.
func TestAssignRestrictedSoftwareResourceModel_IncludeUnmanagedHydratesFromScratch(t *testing.T) {
	state := &RestrictedSoftwareResourceModel{}
	src := &proclassic.RestrictedSoftware{
		ID:      new(9),
		General: &proclassic.RestrictedSoftwareGeneral{Name: new("blocked"), ProcessName: new("evil.app")},
		Scope: &proclassic.RestrictedSoftwareScope{
			AllComputers: new(true),
			ComputerGroups: &proclassic.RestrictedSoftwareScopeComputerGroups{
				ComputerGroup: &[]proclassic.IDName{{ID: new(11)}, {ID: new(22)}},
			},
			Exclusions: &proclassic.RestrictedSoftwareScopeExclusions{
				Users: &proclassic.RestrictedSoftwareScopeExclusionsUsers{
					User: &[]proclassic.IDName{{Name: new("alice")}},
				},
			},
		},
	}
	diags := assignRestrictedSoftwareResourceModel(context.Background(), state, src, true)
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
	if got := len(state.Scope.Exclusions.DirectoryServiceOrLocalUserNames.Elements()); got != 1 {
		t.Fatalf("expected 1 excluded user, got %d", got)
	}
}

func TestFlattenScope_TargetsAndNameKeyedExclusions(t *testing.T) {
	ctx := context.Background()
	s := &proclassic.RestrictedSoftwareScope{
		AllComputers: new(false),
		ComputerGroups: &proclassic.RestrictedSoftwareScopeComputerGroups{
			ComputerGroup: &[]proclassic.IDName{{ID: new(11), Name: new("All Managed")}},
		},
		Exclusions: &proclassic.RestrictedSoftwareScopeExclusions{
			Users: &proclassic.RestrictedSoftwareScopeExclusionsUsers{
				User: &[]proclassic.IDName{{Name: new("alice")}, {Name: new("bob")}},
			},
		},
	}
	state := &RestrictedSoftwareScopeModel{
		Targets:    &RestrictedSoftwareScopeTargetsModel{},
		Exclusions: &RestrictedSoftwareScopeExclusionsModel{},
	}
	flattenScope(ctx, s, state, false)

	if state.Targets.AllComputers.ValueBool() {
		t.Errorf("all_computers should be false")
	}
	if l := len(state.Targets.ComputerGroupIDs.Elements()); l != 1 {
		t.Errorf("expected 1 computer group ID, got %d", l)
	}
	// Empty target categories flatten to an empty set (the canonical "no
	// members" value for these Optional+Computed scope sets), not null.
	for _, tc := range []struct {
		label string
		set   types.Set
	}{
		{"BuildingIDs", state.Targets.BuildingIDs},
		{"DepartmentIDs", state.Targets.DepartmentIDs},
		{"ComputerIDs", state.Targets.ComputerIDs},
	} {
		if tc.set.IsNull() || len(tc.set.Elements()) != 0 {
			t.Errorf("empty target %s must be an empty set, got %v", tc.label, tc.set)
		}
	}
	if l := len(state.Exclusions.DirectoryServiceOrLocalUserNames.Elements()); l != 2 {
		t.Errorf("expected 2 excluded users, got %d", l)
	}
}
