// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package restricted_software

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func idSet(ids ...string) types.Set {
	vals := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, types.StringValue(id))
	}
	return types.SetValueMust(types.StringType, vals)
}

// TestBuildGeneral_RenameMapping verifies the UI-aligned attribute names map to
// the correct classic wire fields.
func TestBuildGeneral_RenameMapping(t *testing.T) {
	m := &RestrictedSoftwareGeneralModel{
		Name:                             types.StringValue("Block Chess"),
		ProcessName:                      types.StringValue("Chess.app"),
		RestrictExactProcessName:         types.BoolValue(true),
		SendEmailNotificationOnViolation: types.BoolValue(true),
		KillProcess:                      types.BoolValue(true),
		DeleteApplication:                types.BoolValue(false),
		DisplayMessage:                   types.StringValue("blocked"),
		SiteID:                           types.StringValue("3"),
	}
	g := buildGeneral(m)
	if g.Name == nil || *g.Name != "Block Chess" {
		t.Errorf("name not mapped: %+v", g.Name)
	}
	if g.ProcessName == nil || *g.ProcessName != "Chess.app" {
		t.Errorf("process_name not mapped")
	}
	if g.MatchExactProcessName == nil || !*g.MatchExactProcessName {
		t.Errorf("restrict_exact_process_name must map to match_exact_process_name=true")
	}
	if g.SendNotification == nil || !*g.SendNotification {
		t.Errorf("send_email_notification_on_violation must map to send_notification=true")
	}
	if g.KillProcess == nil || !*g.KillProcess {
		t.Errorf("kill_process not mapped")
	}
	if g.DeleteExecutable == nil || *g.DeleteExecutable {
		t.Errorf("delete_application must map to delete_executable=false")
	}
	if g.DisplayMessage == nil || *g.DisplayMessage != "blocked" {
		t.Errorf("display_message not mapped")
	}
	if g.Site == nil || g.Site.ID == nil || *g.Site.ID != 3 {
		t.Errorf("site id not mapped: %+v", g.Site)
	}
}

// TestBuildGeneral_OmitsUnsetBools confirms null bools collapse to nil so the
// classic omitempty tags drop the elements (preserving server defaults).
func TestBuildGeneral_OmitsUnsetBools(t *testing.T) {
	m := &RestrictedSoftwareGeneralModel{
		Name:                             types.StringValue("x"),
		ProcessName:                      types.StringValue("y"),
		RestrictExactProcessName:         types.BoolNull(),
		SendEmailNotificationOnViolation: types.BoolNull(),
		KillProcess:                      types.BoolNull(),
		DeleteApplication:                types.BoolNull(),
		DisplayMessage:                   types.StringNull(),
		SiteID:                           types.StringNull(),
	}
	g := buildGeneral(m)
	if g.MatchExactProcessName != nil || g.SendNotification != nil || g.KillProcess != nil || g.DeleteExecutable != nil {
		t.Errorf("unset bools must be nil, got %+v", g)
	}
	if g.DisplayMessage != nil {
		t.Errorf("unset display_message must be nil")
	}
	if g.Site != nil {
		t.Errorf("unset site must be nil")
	}
}

func TestBuildScope_TargetsAndCollapse(t *testing.T) {
	ctx := context.Background()

	// Empty model collapses to nil so <scope> is omitted entirely.
	empty := &RestrictedSoftwareScopeModel{
		Targets: &RestrictedSoftwareScopeTargetsModel{
			AllComputers:     types.BoolNull(),
			ComputerIDs:      types.SetNull(types.StringType),
			ComputerGroupIDs: types.SetNull(types.StringType),
			BuildingIDs:      types.SetNull(types.StringType),
			DepartmentIDs:    types.SetNull(types.StringType),
		},
	}
	s, d := buildScope(ctx, empty)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if s != nil {
		t.Errorf("empty scope must collapse to nil, got %+v", s)
	}

	// Populated targets project into the wire struct.
	full := &RestrictedSoftwareScopeModel{
		Targets: &RestrictedSoftwareScopeTargetsModel{
			AllComputers:     types.BoolValue(false),
			ComputerIDs:      idSet("1", "2"),
			ComputerGroupIDs: idSet("11"),
			BuildingIDs:      idSet("5"),
			DepartmentIDs:    idSet("7"),
		},
	}
	s, d = buildScope(ctx, full)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if s == nil || s.Computers == nil || s.Computers.Computer == nil || len(*s.Computers.Computer) != 2 {
		t.Fatalf("computers not mapped: %+v", s)
	}
	if s.ComputerGroups == nil || s.Buildings == nil || s.Departments == nil {
		t.Errorf("group/building/department targets not mapped")
	}
}

func TestBuildScopeExclusions_NameKeyedUsers(t *testing.T) {
	ctx := context.Background()
	m := &RestrictedSoftwareScopeExclusionsModel{
		ComputerIDs:      types.SetNull(types.StringType),
		ComputerGroupIDs: idSet("8"),
		BuildingIDs:      types.SetNull(types.StringType),
		DepartmentIDs:    types.SetNull(types.StringType),
		DirectoryServiceOrLocalUserNames: types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("alice"), types.StringValue("bob"),
		}),
	}
	e, d := buildScopeExclusions(ctx, m)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if e == nil || e.ComputerGroups == nil {
		t.Fatalf("computer group exclusion not mapped")
	}
	if e.Users == nil || e.Users.User == nil || len(*e.Users.User) != 2 {
		t.Fatalf("user exclusions not mapped: %+v", e.Users)
	}
	// Users must be NAME-keyed (free-text), never ID-keyed.
	for _, u := range *e.Users.User {
		if u.Name == nil || u.ID != nil {
			t.Errorf("exclusion user must be name-keyed, got %+v", u)
		}
	}
}
