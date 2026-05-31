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
	if state.SiteID.ValueString() != "-1" || state.SiteName.ValueString() != "None" {
		t.Errorf("site not flattened: id=%q name=%q", state.SiteID.ValueString(), state.SiteName.ValueString())
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
		Exclusions: &RestrictedSoftwareScopeExclusionsModel{},
	}
	flattenScope(ctx, s, state)

	if state.AllComputers.ValueBool() {
		t.Errorf("all_computers should be false")
	}
	if l := len(state.ComputerGroupIDs.Elements()); l != 1 {
		t.Errorf("expected 1 computer group ID, got %d", l)
	}
	// Empty target categories must flatten to null (server returns empty elements).
	if !state.BuildingIDs.IsNull() || !state.DepartmentIDs.IsNull() || !state.ComputerIDs.IsNull() {
		t.Errorf("empty targets must be null")
	}
	if l := len(state.Exclusions.DirectoryServiceOrLocalUserNames.Elements()); l != 2 {
		t.Errorf("expected 2 excluded users, got %d", l)
	}
}
